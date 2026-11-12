package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/igh-platform/igh-platform/internal/server/model"
	sharedpw "github.com/igh-platform/igh-platform/internal/shared/password"
	"gorm.io/gorm"
)

// 用户管理的业务错误。handler 按这些错误映射到 HTTP 状态码，
// 而不是把 gorm 的原始错误直接抛给前端——那样会把表名、SQL 泄露出去。
var (
	ErrUserNotFound     = errors.New("用户不存在")
	ErrUsernameTaken    = errors.New("用户名已存在")
	ErrInvalidRole      = errors.New("角色不存在")
	ErrLastAdmin        = errors.New("不能停用或删除最后一个管理员账号")
	ErrCannotDisableSelf = errors.New("不能停用或删除自己正在使用的账号")
	ErrWrongOldPassword = errors.New("原密码不正确")
)

// UserView 是返回给前端的用户信息。
//
// 单独定义而不直接返回 model.User：
//   - 密码字段已经 json:"-"，但结构体演进时容易漏
//   - 需要带出角色名，前端列表要显示"管理员"而不是 role_id=1
type UserView struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone,omitempty"`
	RoleID    int64     `json:"role_id"`
	RoleCode  string    `json:"role_code"`
	RoleName  string    `json:"role_name"`
	StationID *int64    `json:"station_id,omitempty"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// LegacyHashCount 返回仍在使用旧 SHA-256 哈希的账号数量。
//
// 登录时会把它们逐个升级；这个数字用于给运维一个"还剩多少没升级"的进度。
func (s *UserService) LegacyHashCount() (int64, error) {
	var passwords []string
	if err := s.db.Model(&model.User{}).Pluck("password", &passwords).Error; err != nil {
		return 0, err
	}
	var n int64
	for _, p := range passwords {
		if sharedpw.IsLegacy(p) {
			n++
		}
	}
	return n, nil
}

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// ListUsers 返回用户列表与角色信息。
func (s *UserService) ListUsers(keyword string, status *int) ([]UserView, error) {
	q := s.db.Model(&model.User{})
	if kw := strings.TrimSpace(keyword); kw != "" {
		// 用 LOWER(...) LIKE 而不是 ILIKE：ILIKE 是 PostgreSQL 专有语法，
		// 测试用的 SQLite 不支持（会直接报语法错误）。
		// PostgreSQL 里 LOWER() 会绕过普通索引，但用户表只有几十行，可接受。
		like := "%" + strings.ToLower(kw) + "%"
		q = q.Where("LOWER(username) LIKE ? OR LOWER(name) LIKE ?", like, like)
	}
	if status != nil {
		q = q.Where("status = ?", *status)
	}

	var users []model.User
	if err := q.Order("id").Find(&users).Error; err != nil {
		return nil, err
	}

	// 一次查出全部角色再映射，避免 N 个用户 N 次查询
	var roles []model.Role
	if err := s.db.Find(&roles).Error; err != nil {
		return nil, err
	}
	roleByID := make(map[int64]model.Role, len(roles))
	for _, r := range roles {
		roleByID[r.ID] = r
	}

	views := make([]UserView, 0, len(users))
	for _, u := range users {
		v := UserView{
			ID: u.ID, Username: u.Username, Name: u.Name, Phone: u.Phone,
			RoleID: u.RoleID, StationID: u.StationID, Status: u.Status,
			CreatedAt: u.CreatedAt,
		}
		if r, ok := roleByID[u.RoleID]; ok {
			v.RoleCode, v.RoleName = r.Code, r.Name
		}
		views = append(views, v)
	}
	return views, nil
}

// ListRoles 返回全部角色，供前端下拉选择。
func (s *UserService) ListRoles() ([]model.Role, error) {
	var roles []model.Role
	err := s.db.Order("id").Find(&roles).Error
	return roles, err
}

// CreateUserReq 是新建用户入参。口令单独传，不和 User 结构混用，
// 避免"把 model.User 整个绑定到请求体"时 password 被意外接收。
type CreateUserReq struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	RoleID    int64  `json:"role_id"`
	StationID *int64 `json:"station_id"`
}

func (s *UserService) CreateUser(req CreateUserReq) (*UserView, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	if len(username) > 64 {
		return nil, fmt.Errorf("用户名不能超过 64 个字符")
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("姓名不能为空")
	}

	if err := s.checkRoleExists(req.RoleID); err != nil {
		return nil, err
	}

	hash, err := sharedpw.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:  username,
		Password:  hash,
		Name:      strings.TrimSpace(req.Name),
		Phone:     strings.TrimSpace(req.Phone),
		RoleID:    req.RoleID,
		StationID: req.StationID,
		Status:    1,
	}

	// 依赖数据库唯一索引而不是"先查再插"：
	// 两个管理员同时提交同名账号时，先查后插会双双通过，最后由唯一索引报错。
	// 直接插入并识别唯一键冲突，判断依据更可靠。
	if err := s.db.Create(user).Error; err != nil {
		if isDuplicateKey(err) {
			return nil, ErrUsernameTaken
		}
		return nil, err
	}

	v := s.viewOf(user)
	return &v, nil
}

// UpdateUserReq 是修改用户入参。口令不在这里改，走 ChangePassword / ResetPassword。
type UpdateUserReq struct {
	Name      *string `json:"name"`
	Phone     *string `json:"phone"`
	RoleID    *int64  `json:"role_id"`
	StationID *int64  `json:"station_id"`
	Status    *int    `json:"status"`
}

// UpdateUser 修改用户。operatorID 用于阻止"把自己停用"。
func (s *UserService) UpdateUser(id int64, req UpdateUserReq, operatorID int64) (*UserView, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	updates := map[string]any{}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("姓名不能为空")
		}
		updates["name"] = name
	}
	if req.Phone != nil {
		updates["phone"] = strings.TrimSpace(*req.Phone)
	}
	if req.RoleID != nil {
		if err := s.checkRoleExists(*req.RoleID); err != nil {
			return nil, err
		}
		// 改角色可能让最后一个管理员消失
		if user.RoleID != *req.RoleID {
			if err := s.ensureNotLastAdmin(user, "修改角色"); err != nil {
				return nil, err
			}
		}
		updates["role_id"] = *req.RoleID
	}
	if req.StationID != nil {
		updates["station_id"] = *req.StationID
	}
	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 {
			return nil, fmt.Errorf("状态只能是 0（停用）或 1（启用）")
		}
		if *req.Status == 0 {
			if id == operatorID {
				return nil, ErrCannotDisableSelf
			}
			if err := s.ensureNotLastAdmin(user, "停用"); err != nil {
				return nil, err
			}
		}
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	v := s.viewOf(&user)
	return &v, nil
}

// DeleteUser 停用账号（软删）。
//
// 不做物理删除：doffings / edge_events 等业务表里记录了 operator 与 created_by，
// 物理删掉用户会让历史记录失去归属，排查时无法回溯是谁操作的。
func (s *UserService) DeleteUser(id, operatorID int64) error {
	if id == operatorID {
		return ErrCannotDisableSelf
	}
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := s.ensureNotLastAdmin(user, "删除"); err != nil {
		return err
	}
	return s.db.Model(&user).Updates(map[string]any{
		"status": 0, "updated_at": time.Now(),
	}).Error
}

// ResetPassword 由管理员重置口令，不需要原密码。
func (s *UserService) ResetPassword(id int64, newPassword string) error {
	hash, err := sharedpw.Hash(newPassword)
	if err != nil {
		return err
	}
	res := s.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]any{
		"password": hash, "updated_at": time.Now(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// ChangePassword 由用户自己修改口令，必须校验原密码。
func (s *UserService) ChangePassword(id int64, oldPassword, newPassword string) error {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	ok, _ := sharedpw.Verify(oldPassword, user.Password)
	if !ok {
		return ErrWrongOldPassword
	}
	if oldPassword == newPassword {
		return fmt.Errorf("新口令不能与原口令相同")
	}

	hash, err := sharedpw.Hash(newPassword)
	if err != nil {
		return err
	}
	return s.db.Model(&user).Updates(map[string]any{
		"password": hash, "updated_at": time.Now(),
	}).Error
}

// UpgradePasswordHash 在登录成功后把旧 SHA-256 哈希升级为 bcrypt。
//
// 只在验证通过后调用，写失败不影响登录（下次登录会再试一次）。
func (s *UserService) UpgradePasswordHash(id int64, plainPassword string) error {
	hash, err := sharedpw.Hash(plainPassword)
	if err != nil {
		// 旧口令可能不满足新的强度要求（比如 6 位）。这种情况跳过升级，
		// 不能因为"口令太短"就把人挡在门外。
		return nil
	}
	return s.db.Model(&model.User{}).Where("id = ?", id).
		Update("password", hash).Error
}

func (s *UserService) checkRoleExists(roleID int64) error {
	if roleID <= 0 {
		return ErrInvalidRole
	}
	var n int64
	if err := s.db.Model(&model.Role{}).Where("id = ?", roleID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return ErrInvalidRole
	}
	return nil
}

// ensureNotLastAdmin 阻止让系统失去最后一个可用的管理员。
//
// 判断依据是"启用状态的管理员角色账号数"。只剩一个时拒绝停用/删除/改角色，
// 否则现场会把自己锁在系统外面，只能进数据库改回来。
func (s *UserService) ensureNotLastAdmin(user model.User, action string) error {
	var role model.Role
	if err := s.db.First(&role, user.RoleID).Error; err != nil {
		return err
	}
	if role.Code != "admin" {
		return nil
	}

	var activeAdmins int64
	err := s.db.Model(&model.User{}).
		Where("role_id = ? AND status = 1 AND id <> ?", role.ID, user.ID).
		Count(&activeAdmins).Error
	if err != nil {
		return err
	}
	if activeAdmins == 0 {
		return ErrLastAdmin
	}
	return nil
}

func (s *UserService) viewOf(u *model.User) UserView {
	v := UserView{
		ID: u.ID, Username: u.Username, Name: u.Name, Phone: u.Phone,
		RoleID: u.RoleID, StationID: u.StationID, Status: u.Status,
		CreatedAt: u.CreatedAt,
	}
	var role model.Role
	if err := s.db.First(&role, u.RoleID).Error; err == nil {
		v.RoleCode, v.RoleName = role.Code, role.Name
	}
	return v
}

// isDuplicateKey 识别唯一键冲突。
//
// 同时兼容 PostgreSQL（SQLSTATE 23505）与 SQLite（测试库）的报错文本，
// 避免为了一个判断引入数据库驱动特定的错误类型。
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "sqlstate 23505") ||
		strings.Contains(msg, "already exists")
}
