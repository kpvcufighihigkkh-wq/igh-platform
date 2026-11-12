package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"gorm.io/gorm"
)

// ERP 同步模式。
const (
	// ModeDisabled 表示不向 ERP 推送任何数据（默认值）
	ModeDisabled = "disabled"
	// ModePushOnly 表示只把数据写到 ERP 的数据库（旧库 fangsi，MySQL 5.7）
	ModePushOnly = "push_only"
	// ModeBidirectional 表示通过 HTTP API 双向同步
	ModeBidirectional = "bidirectional"
)

// ERPConfig 是 ERP 集成配置，对应 configs/erp/integration.yaml。
type ERPConfig struct {
	Mode     string `mapstructure:"mode" yaml:"mode"`
	PushOnly struct {
		Target     string   `mapstructure:"target" yaml:"target"`
		// User / Password 是 ERP 库的凭据。
		// 特意与 Target 分开：URL 里带密码会被原样打进日志和配置文件的历史里。
		// 支持环境变量展开（见 LoadERPConfig）。
		User       string   `mapstructure:"user" yaml:"user"`
		Password   string   `mapstructure:"password" yaml:"password"`
		Tables     []string `mapstructure:"tables" yaml:"tables"`
		Retry      int      `mapstructure:"retry" yaml:"retry"`
		TimeoutSec int      `mapstructure:"timeout_sec" yaml:"timeout_sec"`
	} `mapstructure:"push_only" yaml:"push_only"`
	Bidirectional struct {
		APIEndpoint string `mapstructure:"api_endpoint" yaml:"api_endpoint"`
		Push        struct {
			Tables  []string `mapstructure:"tables" yaml:"tables"`
			Trigger string   `mapstructure:"trigger" yaml:"trigger"`
		} `mapstructure:"push" yaml:"push"`
		Pull struct {
			Resources   []string `mapstructure:"resources" yaml:"resources"`
			IntervalSec int      `mapstructure:"interval_sec" yaml:"interval_sec"`
		} `mapstructure:"pull" yaml:"pull"`
	} `mapstructure:"bidirectional" yaml:"bidirectional"`
}

// ErrERPNotImplemented 表示当前模式下不具备实际推送能力。
//
// 单独定义是为了让调用方能区分"真的推失败了（网络/对端拒绝）"和
// "这个模式根本没实现"。两者都不该被当成成功。
var ErrERPNotImplemented = errors.New("erp: 该同步模式尚未实现，数据未推送")

type ERPService struct {
	db     *gorm.DB
	config ERPConfig
	// mappings 是新系统表 → ERP 表的字段映射，由 main 从 mapping.yaml 载入。
	// 为空时 push_only 会明确失败，而不是把对不上的字段硬塞过去。
	mappings map[string]TableMapping
	// client 可替换，测试时注入 httptest 客户端
	client *http.Client

	// erpConn 是 ERP 库的连接池，惰性创建后复用。
	// 每次推送都 sql.Open 的话，一次批量 50 条就是 50 次 TCP+认证握手。
	dbMu    sync.Mutex
	erpConn *sql.DB
}

func NewERPService(db *gorm.DB, cfg ERPConfig) *ERPService {
	s := &ERPService{db: db, config: cfg}
	timeout := time.Duration(cfg.PushOnly.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	s.client = &http.Client{Timeout: timeout}
	return s
}

// SetHTTPClient 替换 HTTP 客户端（测试用）。
func (s *ERPService) SetHTTPClient(c *http.Client) { s.client = c }

// Mode 返回当前生效的同步模式。
//
// 空值按 disabled 处理：配置忘了写 mode 时应当"什么都不做"，
// 而不是"假装在同步"。
func (s *ERPService) Mode() string {
	if strings.TrimSpace(s.config.Mode) == "" {
		return ModeDisabled
	}
	return s.config.Mode
}

func (s *ERPService) IsEnabled() bool {
	m := s.Mode()
	return m != "" && m != ModeDisabled
}

// RetryLimit 返回失败重试上限。
//
// 配置没写时给 3：旧实现直接读配置字段，未加载时是 0，
// 而判据是 retry_count < Retry（0 < 0 恒假），重试功能因此完全失效。
func (s *ERPService) RetryLimit() int {
	if s.config.PushOnly.Retry > 0 {
		return s.config.PushOnly.Retry
	}
	return 3
}

func (s *ERPService) EnqueueSync(target string, recordID int64, payload any) error {
	if !s.IsEnabled() {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erp: 序列化 %s/%d 失败: %w", target, recordID, err)
	}
	record := &model.ERPSyncQueue{
		Target:    target,
		RecordID:  recordID,
		Payload:   string(data),
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	return s.db.Create(record).Error
}

func (s *ERPService) ListQueue(status string, page, pageSize int) ([]model.ERPSyncQueue, int64, error) {
	var items []model.ERPSyncQueue
	var total int64
	q := s.db.Model(&model.ERPSyncQueue{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (s *ERPService) RetryFailed() (int, error) {
	result := s.db.Model(&model.ERPSyncQueue{}).
		Where("status = ? AND retry_count < ?", "failed", s.RetryLimit()).
		Updates(map[string]any{"status": "pending"})
	return int(result.RowsAffected), result.Error
}

func (s *ERPService) ProcessQueue() {
	if !s.IsEnabled() {
		return
	}
	var items []model.ERPSyncQueue
	if err := s.db.Where("status = ?", "pending").Order("created_at").Limit(50).
		Find(&items).Error; err != nil {
		logger.L.Errorw("erp: 读取待推送队列失败", "error", err)
		return
	}

	for _, item := range items {
		err := s.pushToERP(item)
		if err != nil {
			// 失败就把原因写进队列、保持 failed，让重试与人工排查都能看到。
			// 绝不能标成 sent —— 那等于告诉系统"已经推给 ERP 了"。
			if uerr := s.db.Model(&item).Updates(map[string]any{
				"status":        "failed",
				"retry_count":   gorm.Expr("retry_count + 1"),
				"error_message": err.Error(),
			}).Error; uerr != nil {
				// 连"标记失败"都写不进去，这条会一直停在 pending 被反复重推，
				// 必须单独告警，否则 ERP 侧会收到重复数据。
				logger.L.Errorw("erp: 标记失败状态也写入失败，该条会重复推送",
					"queue_id", item.ID, "target", item.Target, "error", uerr)
			}
			logger.L.Warnw("erp sync failed",
				"target", item.Target, "id", item.RecordID, "mode", s.Mode(), "error", err)
			continue
		}
		now := time.Now()
		// 推送成功但状态写失败，同样会重复推送（虽然 upsert 幂等，但会白跑）。
		if uerr := s.db.Model(&item).Updates(map[string]any{
			"status": "sent", "sent_at": now,
		}).Error; uerr != nil {
			logger.L.Errorw("erp: 推送成功但标记 sent 失败，该条会重复推送",
				"queue_id", item.ID, "target", item.Target, "error", uerr)
		}
	}
}

// pushToERP 执行一次真实推送。
//
// 返回 nil 只代表"对端确认收到了"。任何无法确认的情况都必须返回错误：
//   - 模式未实现（表没有配字段映射，见 erp/mapping.yaml）
//   - 配置缺失（目标地址为空）
//   - 对端返回非 2xx
func (s *ERPService) pushToERP(item model.ERPSyncQueue) error {
	switch s.Mode() {
	case ModePushOnly:
		return s.pushToMySQL(item)
	case ModeBidirectional:
		return s.pushToAPI(item)
	default:
		return fmt.Errorf("erp: 未知同步模式 %q", s.config.Mode)
	}
}

// pushToMySQL 通过数据库直连把记录写进 ERP 库（fangsi，MySQL 5.7）。
//
// 为什么不直接把 payload 塞过去：ERP 侧的字段名与语义由对方决定，
// fangsi 的 doffings 有 order_code / paper_tube / specification，
// 新系统根本没有同名字段；trolleys 是"一车两落筒"的扁平结构
// （doffing_1_id / winder_name_2…），与新系统的 doffing_ids JSON 数组完全不同。
// 所以必须走 mapping.yaml 声明的显式映射。
//
// 幂等：用 INSERT ... ON DUPLICATE KEY UPDATE，同一记录重复推送只更新不重复插入。
// 这样边端/队列重试不会在 ERP 侧造出重复行。
func (s *ERPService) pushToMySQL(item model.ERPSyncQueue) error {
	target := strings.TrimSpace(s.config.PushOnly.Target)
	if target == "" {
		return fmt.Errorf("erp: 未配置 push_only.target")
	}

	m, ok := s.mappings[item.Target]
	if !ok {
		return fmt.Errorf("%w: 表 %q 没有配置字段映射（见 erp/mapping.yaml）",
			ErrERPNotImplemented, item.Target)
	}
	if m.Key == "" {
		return fmt.Errorf("erp: 表 %q 的映射缺少 key 列，无法保证幂等", item.Target)
	}

	view, err := s.buildView(item)
	if err != nil {
		return err
	}
	cols, vals, err := applyMapping(m, view)
	if err != nil {
		return err
	}
	if len(cols) == 0 {
		return fmt.Errorf("erp: 表 %q 的映射没有产生任何列", item.Target)
	}

	query := buildUpsertSQL(m.Target, m.Key, cols)
	db, err := s.erpDB(target)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(s.erpTimeoutSec())*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, query, vals...); err != nil {
		return fmt.Errorf("erp: 写入 %s 失败: %w", m.Target, err)
	}
	logger.L.Infow("erp row pushed",
		"table", m.Target, "key", m.Key, "record_id", item.RecordID, "cols", len(cols))
	return nil
}

// erpDB 返回复用的 ERP 数据库连接池。
//
// 不能每次推送都 sql.Open：ProcessQueue 一次最多处理 50 条，
// 那样就是 50 次 TCP 握手 + 身份认证，对远程库既慢又浪费。
// 连接池由 database/sql 自己管理，这里只负责惰性创建一次。
func (s *ERPService) erpDB(target string) (*sql.DB, error) {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	if s.erpConn != nil {
		return s.erpConn, nil
	}
	db, err := sql.Open("mysql", mysqlDSN(target, s.config))
	if err != nil {
		return nil, fmt.Errorf("erp: 打开 ERP 数据库失败: %w", err)
	}
	// 队列是串行处理的，两个连接足够；设上限避免异常时把对端连接数吃满
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)

	s.erpConn = db
	return db, nil
}

// Close 释放 ERP 连接池（服务退出时调用）。
func (s *ERPService) Close() error {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()
	if s.erpConn == nil {
		return nil
	}
	err := s.erpConn.Close()
	s.erpConn = nil
	return err
}

// buildUpsertSQL 生成 INSERT ... ON DUPLICATE KEY UPDATE。
//
// 所有列都在 UPDATE 子句里刷新（除主键），这样重试能纠正之前写错的值，
// 而不是"第一次写进去就永远不改"。
func buildUpsertSQL(table, key string, cols []string) string {
	placeholders := make([]string, len(cols))
	updates := make([]string, 0, len(cols))
	for i, c := range cols {
		placeholders[i] = "?"
		if c == key {
			continue
		}
		updates = append(updates, fmt.Sprintf("`%s`=VALUES(`%s`)", c, c))
	}

	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = "`" + c + "`"
	}

	sql := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		table, strings.Join(quoted, ","), strings.Join(placeholders, ","))
	if len(updates) > 0 {
		sql += " ON DUPLICATE KEY UPDATE " + strings.Join(updates, ",")
	}
	return sql
}

// mysqlDSN 由 target（形如 mysql://10.30.30.144:3306/fangsi）与凭据拼出 DSN。
//
// 凭据单独放在 push_only.user / push_only.password，而不是塞进 URL：
// URL 里带密码会被原样打进日志与配置文件的版本历史。
func mysqlDSN(target string, cfg ERPConfig) string {
	host, port, dbName := parseMySQLTarget(target)
	user := cfg.PushOnly.User
	pass := cfg.PushOnly.Password
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local&timeout=5s",
		user, pass, host, port, dbName)
}

// parseMySQLTarget 解析 mysql://host[:port]/dbname。
//
// 缺端口时用 3306；缺库名时返回空串（由调用方报错），
// 不猜默认库名——连错库比连不上更危险。
func parseMySQLTarget(target string) (host, port, dbName string) {
	rest := target
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	// 去掉可能的查询串
	if i := strings.Index(rest, "?"); i >= 0 {
		rest = rest[:i]
	}
	if i := strings.Index(rest, "/"); i >= 0 {
		dbName = rest[i+1:]
		rest = rest[:i]
	}
	host = rest
	port = "3306"
	if i := strings.LastIndex(rest, ":"); i >= 0 {
		host = rest[:i]
		port = rest[i+1:]
	}
	return host, port, dbName
}

func (s *ERPService) erpTimeoutSec() int {
	if s.config.PushOnly.TimeoutSec > 0 {
		return s.config.PushOnly.TimeoutSec
	}
	return 10
}

// pushToAPI 通过 HTTP 推到 ERP 接口。
//
// 请求形如：POST {api_endpoint}/{target}，body 为记录 JSON。
//
// 注意：这个路径约定是按 configs/erp/integration.yaml 里
// "api_endpoint + push.tables" 的形式推断的，**必须与 ERP 厂商确认**后
// 才能用于生产。对端返回非 2xx 一律当失败。
func (s *ERPService) pushToAPI(item model.ERPSyncQueue) error {
	base := strings.TrimRight(strings.TrimSpace(s.config.Bidirectional.APIEndpoint), "/")
	if base == "" {
		return fmt.Errorf("erp: 未配置 bidirectional.api_endpoint")
	}
	if item.Target == "" {
		return fmt.Errorf("erp: 队列项 %d 缺少目标表名", item.ID)
	}
	if !json.Valid([]byte(item.Payload)) {
		return fmt.Errorf("erp: 队列项 %d 的 payload 不是合法 JSON", item.ID)
	}

	url := base + "/" + item.Target
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(item.Payload)))
	if err != nil {
		return fmt.Errorf("erp: 构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("erp: 推送 %s 失败: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("erp: 推送 %s 被拒: HTTP %d", url, resp.StatusCode)
	}
	return nil
}
