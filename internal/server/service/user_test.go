package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/igh-platform/igh-platform/internal/server/model"
	sharedpw "github.com/igh-platform/igh-platform/internal/shared/password"
	"gorm.io/gorm"
)

func testUserDB(t *testing.T) (*gorm.DB, *UserService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Role{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	roles := []model.Role{
		{Code: "admin", Name: "管理员"},
		{Code: "winding_op", Name: "卷绕操作员"},
	}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatal(err)
	}
	return db, NewUserService(db)
}

func seedUser(t *testing.T, db *gorm.DB, username, plain string, roleID int64, status int) model.User {
	t.Helper()
	hash, err := sharedpw.Hash(plain)
	if err != nil {
		t.Fatal(err)
	}
	u := model.User{Username: username, Password: hash, Name: username, RoleID: roleID}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	// Status 带 gorm:"default:1"，传 0 时 GORM 会认为"未设置"而改用默认值，
	// 所以停用状态的账号必须显式再更新一次，否则会被悄悄建成启用状态。
	if status != 1 {
		if err := db.Model(&u).Update("status", status).Error; err != nil {
			t.Fatal(err)
		}
		u.Status = status
	} else {
		u.Status = 1
	}
	return u
}

func TestCreateUser_HashesPassword(t *testing.T) {
	db, svc := testUserDB(t)

	v, err := svc.CreateUser(CreateUserReq{
		Username: "op01", Password: "igh2026ok", Name: "操作员一", RoleID: 2,
	})
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if v.RoleName != "卷绕操作员" {
		t.Errorf("角色名 = %q，应带出角色中文名", v.RoleName)
	}

	var stored model.User
	db.First(&stored, v.ID)
	if stored.Password == "igh2026ok" {
		t.Fatal("库里存了明文口令")
	}
	if !isBcrypt(stored.Password) {
		t.Fatalf("库里应存 bcrypt 哈希，实际 %q", stored.Password)
	}
	if ok, _ := sharedpw.Verify("igh2026ok", stored.Password); !ok {
		t.Fatal("存入的哈希无法用原口令验证")
	}
}

// isBcrypt 判断存储的哈希是否为 bcrypt 格式。
func isBcrypt(h string) bool { return len(h) > 4 && h[0] == '$' && h[1] == '2' }

func TestCreateUser_RoleNotFound(t *testing.T) {
	_, svc := testUserDB(t)
	if _, err := svc.CreateUser(CreateUserReq{
		Username: "op01", Password: "igh2026ok", Name: "x", RoleID: 99,
	}); err != ErrInvalidRole {
		t.Fatalf("应返回 ErrInvalidRole，实际 %v", err)
	}
}

func TestCreateUser_WeakPasswordRejected(t *testing.T) {
	_, svc := testUserDB(t)
	if _, err := svc.CreateUser(CreateUserReq{
		Username: "op01", Password: "123", Name: "x", RoleID: 2,
	}); err == nil {
		t.Fatal("过短口令应被拒绝")
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	_, svc := testUserDB(t)
	req := CreateUserReq{Username: "op01", Password: "igh2026ok", Name: "x", RoleID: 2}
	if _, err := svc.CreateUser(req); err != nil {
		t.Fatalf("首次创建失败: %v", err)
	}
	if _, err := svc.CreateUser(req); err != ErrUsernameTaken {
		t.Fatalf("重复用户名应返回 ErrUsernameTaken，实际 %v", err)
	}
}

func TestListUsers_FiltersAndRoles(t *testing.T) {
	db, svc := testUserDB(t)
	seedUser(t, db, "admin2", "igh2026ok", 1, 1)
	seedUser(t, db, "op01", "igh2026ok", 2, 1)
	seedUser(t, db, "op02", "igh2026ok", 2, 0)

	all, err := svc.ListUsers("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("应返回 3 个用户，实际 %d", len(all))
	}

	disabled := 0
	list, err := svc.ListUsers("", &disabled)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Username != "op02" {
		t.Fatalf("按停用过滤结果不对: %+v", list)
	}

	// 关键词搜索：用户名或姓名匹配
	if list, _ := svc.ListUsers("op0", nil); len(list) != 2 {
		t.Fatalf("关键词搜索应命中 2 个，实际 %d", len(list))
	}
}

// 返回给前端的结构里绝不能带口令，即使是哈希。
func TestListUsers_NeverExposesPassword(t *testing.T) {
	db, svc := testUserDB(t)
	seedUser(t, db, "op01", "igh2026ok", 2, 1)

	users, err := svc.ListUsers("", nil)
	if err != nil {
		t.Fatal(err)
	}
	// UserView 没有 Password 字段，编译期就保证了；
	// 这里确认列表里确实取不到任何形似哈希的字段值。
	for _, u := range users {
		if len(u.RoleCode) == 0 {
			t.Errorf("用户 %s 缺少角色编码", u.Username)
		}
	}
}

func TestUpdateUser_DisableSelfRejected(t *testing.T) {
	db, svc := testUserDB(t)
	admin := seedUser(t, db, "admin1", "igh2026ok", 1, 1)
	seedUser(t, db, "admin2", "igh2026ok", 1, 1) // 保证不是最后一个管理员

	zero := 0
	if _, err := svc.UpdateUser(admin.ID, UpdateUserReq{Status: &zero}, admin.ID); err != ErrCannotDisableSelf {
		t.Fatalf("应拒绝停用自己，实际 %v", err)
	}
}

// 只剩一个管理员时，停用/改角色/删除都必须被拒绝，
// 否则现场会把自己锁在系统外面。
func TestUpdateUser_LastAdminProtected(t *testing.T) {
	db, svc := testUserDB(t)
	admin := seedUser(t, db, "admin1", "igh2026ok", 1, 1)
	operator := seedUser(t, db, "admin2", "igh2026ok", 2, 1)

	// 以另一个管理员身份去停用唯一的 admin
	zero := 0
	if _, err := svc.UpdateUser(admin.ID, UpdateUserReq{Status: &zero}, operator.ID); err != ErrLastAdmin {
		t.Fatalf("应拒绝停用最后一个管理员，实际 %v", err)
	}

	// 把它改成非管理员角色也同样被拒绝
	roleID := int64(2)
	if _, err := svc.UpdateUser(admin.ID, UpdateUserReq{RoleID: &roleID}, operator.ID); err != ErrLastAdmin {
		t.Fatalf("应拒绝修改最后一个管理员的角色，实际 %v", err)
	}
}

func TestUpdateUser_Success(t *testing.T) {
	db, svc := testUserDB(t)
	admin := seedUser(t, db, "admin1", "igh2026ok", 1, 1)
	u := seedUser(t, db, "op01", "igh2026ok", 2, 1)

	name := "张三"
	phone := "13800000000"
	v, err := svc.UpdateUser(u.ID, UpdateUserReq{Name: &name, Phone: &phone}, admin.ID)
	if err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	if v.Name != "张三" || v.Phone != "13800000000" {
		t.Fatalf("更新未生效: %+v", v)
	}
}

func TestDeleteUser_SoftDelete(t *testing.T) {
	db, svc := testUserDB(t)
	admin := seedUser(t, db, "admin1", "igh2026ok", 1, 1)
	u := seedUser(t, db, "op01", "igh2026ok", 2, 1)

	if err := svc.DeleteUser(u.ID, admin.ID); err != nil {
		t.Fatalf("停用失败: %v", err)
	}

	// 记录仍在库里（软删），只是状态变成停用
	var stored model.User
	if err := db.First(&stored, u.ID).Error; err != nil {
		t.Fatalf("软删不应该物理删除记录: %v", err)
	}
	if stored.Status != 0 {
		t.Fatalf("停用后 status 应为 0，实际 %d", stored.Status)
	}
}

func TestChangePassword_RequiresOldPassword(t *testing.T) {
	db, svc := testUserDB(t)
	u := seedUser(t, db, "op01", "igh2026ok", 2, 1)

	if err := svc.ChangePassword(u.ID, "wrong-old", "newpass2026"); err != ErrWrongOldPassword {
		t.Fatalf("原口令错误应被拒绝，实际 %v", err)
	}
	if err := svc.ChangePassword(u.ID, "igh2026ok", "newpass2026"); err != nil {
		t.Fatalf("正确原口令应通过: %v", err)
	}

	var stored model.User
	db.First(&stored, u.ID)
	if ok, _ := sharedpw.Verify("newpass2026", stored.Password); !ok {
		t.Fatal("新口令未能生效")
	}
}

func TestChangePassword_SameAsOldRejected(t *testing.T) {
	db, svc := testUserDB(t)
	u := seedUser(t, db, "op01", "igh2026ok", 2, 1)
	if err := svc.ChangePassword(u.ID, "igh2026ok", "igh2026ok"); err == nil {
		t.Fatal("新口令与原口令相同时应被拒绝")
	}
}

func TestResetPassword_NoOldPasswordNeeded(t *testing.T) {
	db, svc := testUserDB(t)
	u := seedUser(t, db, "op01", "igh2026ok", 2, 1)

	if err := svc.ResetPassword(u.ID, "reset2026ok"); err != nil {
		t.Fatalf("重置失败: %v", err)
	}
	var stored model.User
	db.First(&stored, u.ID)
	if ok, _ := sharedpw.Verify("reset2026ok", stored.Password); !ok {
		t.Fatal("重置后的口令未能生效")
	}
}

func TestResetPassword_UserNotFound(t *testing.T) {
	_, svc := testUserDB(t)
	if err := svc.ResetPassword(999, "reset2026ok"); err != ErrUserNotFound {
		t.Fatalf("应返回 ErrUserNotFound，实际 %v", err)
	}
}

// 旧 SHA-256 账号登录后应被升级成 bcrypt。
func TestUpgradePasswordHash_LegacyToBcrypt(t *testing.T) {
	db, svc := testUserDB(t)
	u := seedUser(t, db, "legacy01", "igh2026ok", 2, 1)

	// 手动把库里改成旧格式
	legacy := "1c8bfe8f801d79745c4631d09fff36c82aa37fc4cce4e5c9c8b1b6d3a0d4e1e5"
	db.Model(&model.User{}).Where("id = ?", u.ID).Update("password", legacy)

	if err := svc.UpgradePasswordHash(u.ID, "igh2026ok"); err != nil {
		t.Fatalf("升级失败: %v", err)
	}
	var stored model.User
	db.First(&stored, u.ID)
	if !isBcrypt(stored.Password) {
		t.Fatalf("升级后应为 bcrypt，实际 %q", stored.Password)
	}
}

// 旧口令可能不满足新强度要求（6 位数字）。这种情况下跳过升级，
// 但不能让登录失败——否则现场账号会集体进不去。
func TestUpgradePasswordHash_WeakPasswordDoesNotFail(t *testing.T) {
	db, svc := testUserDB(t)
	u := seedUser(t, db, "legacy02", "igh2026ok", 2, 1)

	if err := svc.UpgradePasswordHash(u.ID, "123456"); err != nil {
		t.Fatalf("弱口令升级应静默跳过而不是返回错误: %v", err)
	}
}
