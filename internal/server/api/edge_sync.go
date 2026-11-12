package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/igh-platform/igh-platform/internal/server/model"
	"github.com/igh-platform/igh-platform/internal/shared/types"
	"gorm.io/gorm"
)

func registerEdgeSyncRoutes(group *gin.RouterGroup, db *gorm.DB) {
	sync := group.Group("/sync")

	sync.GET("/lots", edgeSyncList(db, &model.Lot{}, "updated_at"))
	sync.GET("/lot-weights", edgeSyncListAll(db, &model.LotWeight{}))
	sync.GET("/lot-grade-ranges", edgeSyncListAll(db, &model.LotGradeRange{}))
	sync.GET("/lines", edgeSyncList(db, &model.ProductionLine{}, "created_at"))
	sync.GET("/grades", edgeSyncListAll(db, &model.Grade{}))
	sync.GET("/defects", edgeSyncListAll(db, &model.Defect{}))
	sync.GET("/dict-items", edgeSyncListAll(db, &model.DictItem{}))
	sync.GET("/shifts", edgeSyncList(db, &model.Shift{}, "start_at"))
	sync.GET("/settings", edgeSyncList(db, &model.SystemSetting{}, "updated_at"))
	sync.GET("/weighing-rules", edgeSyncListAll(db, &model.WeighingRule{}))
	sync.GET("/product-types", edgeSyncListAll(db, &model.ProductType{}))

	// 生产与仓储的主数据。分拣/包装/仓储插件的 Inbound 依赖这些，
	// 缺一条对应插件就整个拉取失败，日志里表现为 404。
	sync.GET("/doffings", edgeSyncList(db, &model.Doffing{}, "created_at"))
	sync.GET("/bobbins", edgeSyncList(db, &model.Bobbin{}, "updated_at"))
	sync.GET("/trolleys", edgeSyncList(db, &model.Trolley{}, "created_at"))
	sync.GET("/pallets", edgeSyncList(db, &model.Pallet{}, "created_at"))
	sync.GET("/picking-orders", edgeSyncList(db, &model.PickingOrder{}, "created_at"))
	sync.GET("/warehouses", edgeSyncList(db, &model.Warehouse{}, "created_at"))
	sync.GET("/warehouse-slots", edgeSyncList(db, &model.WarehouseSlot{}, "updated_at"))
}

// tableOf 从模型推导表名，供 db.Table 使用。
//
// 复用 gorm 自己的命名策略（含 TableName() 覆盖），避免手写表名漂移。
// db 为 nil 时返回空串（注册路由时可以没有连接）。
func tableOf(db *gorm.DB, model any) string {
	if db == nil {
		return ""
	}
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil || stmt.Schema == nil {
		return ""
	}
	return stmt.Schema.Table
}

// tableResolver 惰性解析表名。
//
// 不用在注册时解析：路由注册允许 db 为 nil（测试里枚举路由就是这样），
// 且正式启动时 db 一定可用，第一次请求解析一次即可。
func tableResolver(db *gorm.DB, model any) func() string {
	var once sync.Once
	var table string
	return func() string {
		once.Do(func() { table = tableOf(db, model) })
		return table
	}
}

func edgeSyncList(db *gorm.DB, model any, timeColumn string) gin.HandlerFunc {
	table := tableResolver(db, model)
	return func(c *gin.Context) {
		// 用 Table 而不是 Model：Model 会让 gorm 把模型字段类型当成 Scan 目标
		// （scan.go 的 prepareValues 在 Statement.Schema != nil 时按字段类型建目标），
		// 于是 json.RawMessage 字段拿 *json.RawMessage 去接驱动的 string 而报错。
		// 这些接口只是把行原样转成 JSON 下发，不需要模型类型参与。
		q := db.Table(table())
		if since := c.Query("since"); since != "" {
			t, err := time.Parse(time.RFC3339, since)
			if err == nil {
				q = q.Where(timeColumn+" > ?", t)
			}
		}

		var results []map[string]any
		if err := q.Find(&results).Error; err != nil {
			c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: results})
	}
}

func edgeSyncListAll(db *gorm.DB, model any) gin.HandlerFunc {
	table := tableResolver(db, model)
	return func(c *gin.Context) {
		var results []map[string]any
		if err := db.Table(table()).Find(&results).Error; err != nil {
			c.JSON(http.StatusInternalServerError, types.Response{Code: 500, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, types.Response{Code: 0, Data: results})
	}
}
