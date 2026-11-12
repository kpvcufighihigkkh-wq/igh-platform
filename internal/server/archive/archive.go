// Package archive 负责把过期业务数据导出成文件后再删除。
//
// 设计原则（按重要性排序）：
//  1. **绝不丢数据**：先导出、校验落盘、再删除。导出失败或行数对不上一律中止。
//  2. **删除可追溯**：每个归档文件旁边留一个清单（manifest），记录删了哪些 ID。
//  3. **不破坏引用**：先处理依赖关系——丝饼随落筒一起归档，但被栈板引用的丝饼保留。
//  4. **可演练**：DryRun 只统计不删除，上线前先用它看一遍会删什么。
package archive

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"gorm.io/gorm"
)

// Policy 是单张表的归档策略。
type Policy struct {
	// Table 是实际表名（与 gorm 的 TableName 一致）。
	Table string
	// TimeColumn 是判断"有多旧"的时间列。
	TimeColumn string
	// RetainDays 是保留天数。超过这个天数的数据进入归档。
	RetainDays int
	// Where 是额外过滤条件（如只归档处理完成的事件）。
	Where string
	// WhereArgs 是 Where 的参数。
	WhereArgs []any
	// DeleteWith 是"必须一起删"的子表：这些表通过 Column 指向本表主键。
	// 不一起删会留下指向不存在父行的孤儿记录。
	DeleteWith []SubTable
}

// SubTable 是从表及其外键列。
type SubTable struct {
	Table  string
	Column string
}

// SubTable 的删除还要考虑"被其他表引用的情况"，用 ProtectedBy 表达：
// 若从表的某行被第三方表引用，则不删这一行，也不删它对应的父行。
type Protection struct {
	SubTable     string
	SubColumn    string
	ByTable      string
	ByColumn     string
}

// Result 是单张表的归档结果。
type Result struct {
	Table      string    `json:"table"`
	Cutoff     time.Time `json:"cutoff"`
	Exported   int       `json:"exported"`
	Deleted    int       `json:"deleted"`
	Protected  int       `json:"protected"`
	File       string    `json:"file,omitempty"`
	DryRun     bool      `json:"dry_run"`
	Err        string    `json:"error,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

type Service struct {
	db  *gorm.DB
	dir string
	// batchSize 是每批删除的行数：一次删几十万行会长时间持锁，
	// 现场夜间跑归档时可能把在线写入阻塞住。
	batchSize int
	// DryRun 为真时只统计、不写文件、不删除
	DryRun bool
	// policies 允许调用方覆盖，测试里用
	policies []Policy
	protect  []Protection
}

// DefaultPolicies 是现场确定的保留期。
func DefaultPolicies(doffingDays, eventDays, erpDays int) []Policy {
	return []Policy{
		{
			Table: "doffings", TimeColumn: "created_at", RetainDays: doffingDays,
			// 丝饼是落筒的从表，必须一起删，否则留下指向不存在落筒的孤儿。
			DeleteWith: []SubTable{{Table: "bobbins", Column: "doffing_id"}},
		},
		{
			Table: "edge_events", TimeColumn: "received_at", RetainDays: eventDays,
			// 只归档处理完成的。failed 记录是排查依据，received 是还没处理完的，
			// 删掉会让"为什么这条事件没生效"永久失去线索。
			Where: "status = ?", WhereArgs: []any{"processed"},
		},
		{
			Table: "erp_sync_queues", TimeColumn: "created_at", RetainDays: erpDays,
			// 只归档已推送的。pending / failed 还在队列里等着重试，删了就丢业务。
			Where: "status = ?", WhereArgs: []any{"sent"},
		},
	}
}

func New(db *gorm.DB, dir string, policies []Policy) *Service {
	return &Service{
		db:        db,
		dir:       dir,
		batchSize: 1000,
		policies:  policies,
		protect: []Protection{
			// 被栈板引用的丝饼不能删：pallet_bobbins 是"哪一垛里有哪几个锭"的
			// 唯一依据，删了追溯就断链（客诉查锭、ERP 报工都依赖它）。
			{SubTable: "bobbins", SubColumn: "id", ByTable: "pallet_bobbins", ByColumn: "bobbin_id"},
		},
	}
}

// SetBatchSize 调整每批删除行数（测试用）。
func (s *Service) SetBatchSize(n int) {
	if n > 0 {
		s.batchSize = n
	}
}

// Run 依次执行所有策略。
func (s *Service) Run(ctx context.Context) []Result {
	results := make([]Result, 0, len(s.policies))
	for _, p := range s.policies {
		results = append(results, s.archiveTable(ctx, p))
	}
	return results
}

// 返回命名返回值而不是局部变量：defer 里改的必须是最终返回的那个 Result，
// 用局部变量时 return r 早已把副本交出去了，FinishedAt 会一直是零值。
func (s *Service) archiveTable(ctx context.Context, p Policy) (r Result) {
	r = Result{Table: p.Table, DryRun: s.DryRun, StartedAt: time.Now()}
	defer func() { r.FinishedAt = time.Now() }()

	if p.RetainDays <= 0 {
		r.Err = "保留天数必须大于 0"
		return r
	}
	// 按"自然日"算截止时间，避免同一批数据在一天内反复跨过边界。
	r.Cutoff = truncateToDay(time.Now().AddDate(0, 0, -p.RetainDays))

	// 1. 确定要归档的主键集合（与被保护的行在 SQL 层就排除掉）
	ids, protected, err := s.collectIDs(ctx, p, r.Cutoff)
	if err != nil {
		r.Err = "收集待归档 ID 失败: " + err.Error()
		return r
	}
	r.Protected = protected
	if len(ids) == 0 {
		logger.L.Infow("archive: 无过期数据", "table", p.Table, "cutoff", r.Cutoff.Format("2006-01-02"))
		return r
	}

	if s.DryRun {
		// DryRun 只报会删多少，不写文件也不删
		r.Exported = len(ids)
		r.Deleted = 0
		logger.L.Infow("archive: 预演", "table", p.Table, "would_archive", len(ids),
			"protected", protected, "cutoff", r.Cutoff.Format("2006-01-02"))
		return r
	}

	// 2. 导出到文件（先写临时文件，校验通过再改名，避免半截文件被当成完整归档）
	file, exported, err := s.export(ctx, p, ids, r.Cutoff)
	if err != nil {
		r.Err = "导出失败（未删除任何数据）: " + err.Error()
		return r
	}
	r.File = file
	r.Exported = exported

	// 3. 校验：导出行数必须和待删 ID 数一致。
	//    少一行就说明导出过程中出了问题，此时删除等于丢数据。
	if exported != len(ids) {
		r.Err = fmt.Sprintf("导出行数 %d 与待删 ID 数 %d 不一致，已中止删除", exported, len(ids))
		return r
	}

	// 4. 批量删除（含从表）
	deleted, err := s.deleteIDs(ctx, p, ids)
	if err != nil {
		r.Err = "删除失败（文件已生成，可人工处理）: " + err.Error()
		r.Deleted = deleted
		return r
	}
	r.Deleted = deleted

	logger.L.Infow("archive: 完成",
		"table", p.Table, "exported", exported, "deleted", deleted,
		"protected", protected, "file", file)
	return r
}

// protectedPredicate 生成"该主表行被第三方表引用"的 EXISTS 子查询片段。
//
// 注意不能把 pallet_bobbins.bobbin_id 直接和 doffings.id 比——
// 那是丝饼 ID 和落筒 ID，两回事。必须经由从表关联：
//
//	EXISTS (
//	  SELECT 1 FROM bobbins s
//	  JOIN pallet_bobbins r ON r.bobbin_id = s.id
//	  WHERE s.doffing_id = doffings.id
//	)
func (s *Service) protectedPredicate(p Policy, pr Protection, negate bool) (string, bool) {
	var sub SubTable
	found := false
	for _, st := range p.DeleteWith {
		if st.Table == pr.SubTable {
			sub = st
			found = true
			break
		}
	}
	if !found {
		return "", false
	}
	op := "EXISTS"
	if negate {
		op = "NOT EXISTS"
	}
	return fmt.Sprintf("%s (SELECT 1 FROM %s s JOIN %s r ON r.%s = s.%s WHERE s.%s = %s.id)",
		op, pr.SubTable, pr.ByTable, pr.ByColumn, pr.SubColumn, sub.Column, p.Table), true
}

// collectIDs 找出"足够旧、且不被保护"的主键。
func (s *Service) collectIDs(ctx context.Context, p Policy, cutoff time.Time) ([]int64, int, error) {
	base := func() *gorm.DB {
		q := s.db.WithContext(ctx).Table(p.Table).
			Where(fmt.Sprintf("%s < ?", p.TimeColumn), cutoff)
		if p.Where != "" {
			q = q.Where(p.Where, p.WhereArgs...)
		}
		return q
	}

	// 排除被第三方表引用的行。用 NOT EXISTS 而不是 NOT IN：
	// 子查询返回大量行时 NOT IN 会退化成逐行比较，慢到不可用。
	q := base()
	protectedPreds := make([]string, 0, len(s.protect))
	for _, pr := range s.protect {
		if pred, ok := s.protectedPredicate(p, pr, true); ok {
			q = q.Where(pred)
			// 记下同一条规则的非否定形式，下面数"被保护行数"用它。
			// 只对真正适用于本表的规则计数：不然每张表都会把全部待归档行
			// 当成"被保护"报出来（protected == exported），完全是噪音。
			if pos, ok := s.protectedPredicate(p, pr, false); ok {
				protectedPreds = append(protectedPreds, pos)
			}
		}
	}

	var ids []int64
	if err := q.Pluck("id", &ids).Error; err != nil {
		return nil, 0, err
	}

	// 统计被保护的行数（仅用于报告，不影响归档结果）
	protected := 0
	if len(protectedPreds) > 0 {
		pq := base()
		for _, pred := range protectedPreds {
			pq = pq.Where(pred)
		}
		var n int64
		if err := pq.Count(&n).Error; err == nil {
			protected = int(n)
		}
	}
	return ids, protected, nil
}

// hasSubTable 判断保护规则是否适用于这张表（被引用的必须是本表的从表）。
func hasSubTable(p Policy, table string) bool {
	for _, st := range p.DeleteWith {
		if st.Table == table {
			return true
		}
	}
	return false
}

// export 把主表及从表的数据写成 gzip 压缩的 JSON Lines 文件。
//
// 格式选 JSONL 而不是 CSV：字段里有多层 JSON（如 position_detail），
// CSV 需要转义且容易在导入方解析错。一侧一个 gzip 文件也能直接用
// gunzip | jq 处理，现场排查方便。
func (s *Service) export(ctx context.Context, p Policy, ids []int64, cutoff time.Time) (string, int, error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", 0, fmt.Errorf("创建归档目录失败: %w", err)
	}

	name := fmt.Sprintf("%s_%s.jsonl.gz", p.Table, cutoff.Format("20060102"))
	final := filepath.Join(s.dir, name)
	tmp := final + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return "", 0, fmt.Errorf("创建临时文件失败: %w", err)
	}
	// 出错时删除半截文件，避免下次误以为已有归档
	success := false
	defer func() {
		f.Close()
		if !success {
			os.Remove(tmp)
		}
	}()

	gz := gzip.NewWriter(f)
	enc := json.NewEncoder(gz)

	exported := 0
	for start := 0; start < len(ids); start += s.batchSize {
		end := start + s.batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]

		// 主表
		rows, err := s.fetchRows(ctx, p.Table, batch)
		if err != nil {
			return "", 0, err
		}
		for _, row := range rows {
			if err := encodeRecord(enc, p.Table, row); err != nil {
				return "", 0, err
			}
			exported++
		}

		// 从表（同样按本批主键取，保证主从一致）
		for _, st := range p.DeleteWith {
			subRows, err := s.fetchSubRows(ctx, st, batch)
			if err != nil {
				return "", 0, err
			}
			for _, row := range subRows {
				if err := encodeRecord(enc, st.Table, row); err != nil {
					return "", 0, err
				}
				// 从表行不单独计数：导出行数校验只针对主表，
				// 因为删除也是按主表 ID 驱动，从表行数不影响安全性。
			}
		}
	}

	if err := gz.Close(); err != nil {
		return "", 0, fmt.Errorf("关闭压缩流失败: %w", err)
	}
	// 落盘后再改名：进程在这之间崩溃时，留下的是 .tmp，
	// 不会出现"文件存在但内容不全"的情况。
	if err := f.Sync(); err != nil {
		return "", 0, fmt.Errorf("刷盘失败: %w", err)
	}
	f.Close()
	if err := os.Rename(tmp, final); err != nil {
		return "", 0, fmt.Errorf("重命名归档文件失败: %w", err)
	}
	success = true

	if err := s.writeManifest(final, p, ids, cutoff, exported); err != nil {
		// 清单写失败不阻止删除：归档文件本身已经完整，
		// 只是少了一份"删了哪些 ID"的辅助记录。
		logger.L.Warnw("archive: 写清单失败", "file", final, "error", err)
	}
	return final, exported, nil
}

// manifest 记录这次归档删除了哪些主键，便于事后核对与恢复。
type manifest struct {
	Table     string    `json:"table"`
	Cutoff    string    `json:"cutoff"`
	Exported  int       `json:"exported"`
	IDs       []int64   `json:"ids"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) writeManifest(file string, p Policy, ids []int64, cutoff time.Time, exported int) error {
	m := manifest{
		Table: p.Table, Cutoff: cutoff.Format("2006-01-02"),
		Exported: exported, IDs: ids, CreatedAt: time.Now(),
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file+".manifest.json", raw, 0o644)
}

func encodeRecord(enc *json.Encoder, table string, row map[string]any) error {
	// 包一层 {"table":..., "row":...}：一个文件里既有主表也有从表，
	// 导入时需要知道每行属于哪张表。
	wrapped := map[string]any{"table": table, "row": row}
	if err := enc.Encode(wrapped); err != nil {
		return fmt.Errorf("写入归档记录失败（table=%s）: %w", table, err)
	}
	return nil
}

func (s *Service) fetchRows(ctx context.Context, table string, ids []int64) ([]map[string]any, error) {
	// 用 Table() 而不是 Model()：后者会按模型字段类型构造扫描目标，
	// json.RawMessage / text 列在某些驱动下会报 unsupported Scan。
	var rows []map[string]any
	if err := s.db.WithContext(ctx).Table(table).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %w", table, err)
	}
	return rows, nil
}

func (s *Service) fetchSubRows(ctx context.Context, st SubTable, parentIDs []int64) ([]map[string]any, error) {
	var rows []map[string]any
	if err := s.db.WithContext(ctx).Table(st.Table).
		Where(fmt.Sprintf("%s IN ?", st.Column), parentIDs).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %w", st.Table, err)
	}
	return rows, nil
}

// deleteIDs 分批删除主表及其从表。
//
// 从表先删、主表后删：反过来会在中间状态留下指向不存在父行的孤儿，
// 如果有外键约束会直接失败。
func (s *Service) deleteIDs(ctx context.Context, p Policy, ids []int64) (int, error) {
	deleted := 0
	for start := 0; start < len(ids); start += s.batchSize {
		end := start + s.batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]

		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, st := range p.DeleteWith {
				// 从表也要排除被保护的（例如已上栈板的丝饼）。
				//
				// 这里判断的是从表自己的行是否被第三方引用，
				// 所以直接用 r.<引用的列> = <从表>.id，不需要再关联。
				q := tx.Table(st.Table).Where(fmt.Sprintf("%s IN ?", st.Column), batch)
				for _, pr := range s.protect {
					if pr.SubTable != st.Table {
						continue
					}
					q = q.Where(fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s r WHERE r.%s = %s.id)",
						pr.ByTable, pr.ByColumn, st.Table))
				}
				if err := q.Delete(nil).Error; err != nil {
					return fmt.Errorf("删除 %s 失败: %w", st.Table, err)
				}
			}

			// 主表：同样排除"仍有从表行被第三方引用"的记录
			q := tx.Table(p.Table).Where("id IN ?", batch)
			for _, pr := range s.protect {
				if pred, ok := s.protectedPredicate(p, pr, true); ok {
					q = q.Where(pred)
				}
			}
			res := q.Delete(nil)
			if res.Error != nil {
				return fmt.Errorf("删除 %s 失败: %w", p.Table, res.Error)
			}
			deleted += int(res.RowsAffected)
			return nil
		})
		if err != nil {
			return deleted, err
		}
	}
	return deleted, nil
}

func truncateToDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
