# 边端工位交互规格说明

> 对应原型：`docs/design/edge-station-prototype.html`（单文件，双击浏览器打开）
> 状态：设计稿 v1，待现场确认后进入实现
> 目标读者：前端实现者、边端后端维护者、现场工艺负责人

---

## 0. 调研结论（写规格之前必须先对齐的事实）

### 0.1 边端只注册一个工位插件

`internal/edge/app/app.go` 里 `plugin.Get(cfg.Edge.Type)` 只取**一个**插件，`RegisterRoutes(v1, db)` 也只挂这一个插件的路由。

**结论：** 落筒机上的 `/api/v1/pallets`、`/api/v1/warehouses` 这些接口**根本不存在**，不是"没权限"而是 404。任何工位页面只能调用：

1. 本工位插件的路由；
2. `router.go` 里的通用路由（`/edge/*`、`/system/modules`）。

原型里四个工位共用一套导航只是演示方便；真实边端页面**不提供工位切换**，由 `edge.yaml` 的 `edge.type` 在启动时决定，`web/src/router/index.ts` 也只注册本工位一个路由。

### 0.2 前端拿不到 WebSocket

- `edge/app/router.go` **没有任何 `/ws` 路由**。`embed.go` 里只是把 `/ws` 前缀当作 API 路径返回 404。
- `web/src/composables/useWebSocket.ts` 连的是 `/api/v1/ws?token=...`，这是中心端（admin）才有的。`station/Index.vue` 用了它，在边端会一直 `connected === false`，那个"离线"标签是假的。

**结论：** 边端页面的实时性只能靠**轮询**。建议：

| 数据 | 轮询周期 | 接口 |
| --- | --- | --- |
| PLC 设备在线状态 | 5 s | `GET /api/v1/edge/plc/status` |
| 同步状态 / 待同步条数 | 5 s | `GET /api/v1/edge/sync/status` |
| 工位业务数据（落筒、丝饼、栈板、库位） | 2 s（触摸屏本机，开销可接受） | 各插件 GET |
| 打印机在线状态 | 30 s | `GET /api/v1/edge/printers` |

### 0.3 打印机模板没有列表接口

`app.go` 启动时从 `cfg.Edge.ConfigDir/labels/templates` 加载 `.zpl`，但 `/edge/print` 只接受 `template_name` 字符串，**没有接口能列出有哪些模板**，也没有接口列出可用打印机的编码清单（`/edge/printers` 返回的是配置明细，勉强能当清单用）。

**结论：** 页面上"选模板"目前只能硬编码模板名。要么补 `GET /api/v1/edge/templates`，要么把模板清单放进 `/edge/info`。见第 5 节"待实现"。

### 0.4 现有页面与真实接口的偏差

| 页面 | 调用了 | 实际情况 |
| --- | --- | --- |
| `station/Index.vue` | `/alarms?status=active`、`PUT /alarms/:id/acknowledge`、`POST /labels/print` | **边端不存在这些接口**。而且 `Index.vue` 没有被 router 引用（edgeRoutes 和 adminRoutes 都指向 Winding/Sorting/Packaging/WarehouseOp），当前是死代码。 |
| `Winding.vue` | `/lots`、`/doffings`、`/trolleys`、`/stats/daily-output` | 存在，路径对得上。但用 `lot_id` 筛选、`doffings` 分页是 `items` 包裹，写法要核对。 |
| `Sorting.vue` | `/bobbins`、`/stats/grade-distribution` | `/stats/grade-distribution` **不存在**，真实是 `/quality/distribution`。当前页面会静默失败（`load()` 里 Promise.all 一挂全挂）。 |
| `Packaging.vue` | `/packing-orders`、`/pallets` | 存在。但顶部的"在线"标签是写死的 `<n-tag type="success">在线</n-tag>`，不是真实状态。 |
| `WarehouseOp.vue` | `/warehouses`、`/warehouses/:id/slots`、`/warehouses/:id/stats` | 存在。`手动入库/出库` 按钮的 `requireAuth(() => {})` 是空实现，按钮点了没反应。 |

---

## 1. 通用设计约束

这些不是建议，是硬约束，实现时逐条对照：

| 约束 | 数值 / 做法 | 原因 |
| --- | --- | --- |
| 按钮最小高度 | 64 px | 戴沾油手套，指腹触控面积大 |
| 主操作按钮 | 112 px，占行宽 | 拇指自然落点，减少误触相邻按钮 |
| 正文字号 | ≥ 18 px | 车间距离 1~1.5 m 观看，且部分工人有老花 |
| 数字字号 | 38~52 px，等宽数字 | 远处扫一眼就能读出产量 |
| 禁止 hover 才出现的操作 | 所有操作常驻可见 | 触摸屏没有 hover 状态 |
| 禁止整页滚动 | 页面 `overflow: hidden`，仅"最近记录"列表内部滚动，高度固定 | 现场的屏是固定在支架上的，滚动会滑走 |
| 禁止二次确认弹窗（可撤销操作） | 落筒确认、等级提交直接执行 + toast 反馈 | 每锭弹一次确认，一个班要弹上千次 |
| 必须二次确认（不可撤销） | 手动出库、换垛清空、手动干预 | 会造成账实不符 |
| 触摸反馈 | `:active { transform: translateY(2px); filter: brightness(1.35) }` | 手套按下去没有触觉反馈，必须靠视觉 |
| 颜色语义 | 绿 `#2ecc71` 正常 / 黄 `#f0b429` 待处理 / 红 `#ef4444` 异常 / 灰 空闲 | 与现有 `#0d1117` 深色主题一致 |

---

## 2. 全局状态定义

### 2.1 网络状态（顶部常驻，不可点）

| 状态 | 触发条件 | 界面表现 | 业务影响 |
| --- | --- | --- | --- |
| `online` 在线 | `GET /edge/sync/status` 返回 `online: true` | 绿色徽标"在线" | 全部操作可用 |
| `offline` 离线 | `online: false` 且持续 > 3 s | 徽标变红 + 顶部红条 + 工作区降饱和（`grayscale(.75)`） | 本地写入照常；标记 `need-net` 的操作置灰并说明原因 |
| `syncing` 同步中 | `online: true` 且 `queue_depth > 0` | 徽标变黄 + 顶部黄条"正在上传 N 条" | 允许读，写入排队；禁止二次出库类操作 |

**轮询：** 每 5 s 请求 `GET /api/v1/edge/sync/status`，取 `data.online` 和 `data.queue_depth`。
红条文案必须带**待同步条数**（`queue_depth`），否则工人不知道断网期间的数据会不会丢。

### 2.2 PLC 状态

`GET /api/v1/edge/plc/status` 返回 `{ "<device_code>": { connected, last_read_ms, host, port, protocol, model, errors } }`。

页面不显示 `host/port`（工人不关心），只显示：

- 每个设备一行：`● 卷绕机 W1  在线` / `● 卷绕机 W3  重连中（已失败 4 次）`
- `connected: false` 且 `errors > 3` → 红点；`connected: false` 且 `errors ≤ 3` → 黄点"重连中"
- `last_read_ms > 10000` → 即使 `connected: true` 也显示黄点"数据延迟 12s"

**这是工人判断"页面数字是不是真的"的唯一依据**，必须放在右上角固定位置。

### 2.3 打印机状态

`GET /api/v1/edge/printers` 返回 `[{ code, host, port, online, last_check }]`。

- 页面只显示 `code` 和在线状态，不显示 IP
- `online: false` → 打印按钮置灰，提示"打印机离线，请检查线缆"
- 打印失败（`POST /edge/print` 返回非 0）→ toast 红色 + 打印队列计数 +1，不阻塞工人继续操作

---

## 3. 工位一：落筒 winding

### 3.1 页面目标

让挡车工在**不离开工位、不低头看键盘**的情况下，确认一次落筒、绑定料车、补打标签。

### 3.2 进入条件

- `edge.type === "winding"`
- 无登录（边端路由守卫 `if (edgeType) { next() }`，工位页面不需要 token）

### 3.3 关键数据来源：落筒是 PLC 自动写入的

读 `internal/edge/plugin/winding/points.go` 可确认：`PointInterpreter.HandlePoints` 在检测到落筒信号后，会在**本地 SQLite 事务里直接 `Create(&Doffing{})` 和批量 `Create(&Bobbins)`**（`persistLocal`）。

**这意味着：落筒记录不是靠工人点按钮产生的。** 工人点的"落筒完成确认"只能是：

- 方案 A：一个纯确认动作（记录"已复核"），不改数据；
- 方案 B：调用 `POST /doffings` 手动补录（PLC 漏采时用）；
- 方案 C：不做按钮，只做展示。

原型按**方案 A** 画的（按钮存在、点了给反馈），因为工人看到一个"确认"按钮会有安全感，但**这一点必须由用户拍板**，见第 6 节 Q2。

### 3.4 主流程

```
1. 页面加载 → 轮询 /plc/status、/lots?status=active、/doffings?page=1&page_size=10、
   /stats/daily-output?days=1、/stats/winder
   ↓ 验证：顶部批号显示当前活跃批次；3 个统计块有数字
2. 锭位盘显示 8 个锭位（bobbins_per_doff = 8），长度条随 PLC 轮询增长
   ↓ 验证：数字每 2 s 变化一次；涨满的锭位变绿 + "满筒 · 等待落筒"
3. 工人按「落筒完成确认」
   ↓ 验证：出现 toast；锭位清零；今日落筒 +1；离线时红条里的待同步数 +1
4. 工人按「料车装载」→ 选/扫料车号 → 提交
   ↓ 调用：POST /trolleys/load  { number, rfid?, lot_id, doffing_ids, line_id? }
   ↓ 验证：料车列表新增一条状态 loaded
5. 需要时按「标签补打」→ 选模板 → 打印
   ↓ 调用：POST /edge/print  { printer_code, template_name: "bobbin.zpl", data: {...} }
```

### 3.5 异常分支

| 场景 | 页面表现 | 处理 |
| --- | --- | --- |
| **断网** | 顶部红条，工作区降饱和 | 落筒确认/料车装载写本地并计数；「手动干预」置灰（需要中心审核） |
| **PLC 断连** | 右上角 W3 变红"重连中"；锭位盘数字**冻结不动** | 不弹窗。冻结的数字比错误提示更容易被误认为"机器停了"，所以要在锭位盘标题旁显示"数据延迟 12s" |
| **PLC 恢复** | 状态点变绿，数字恢复跳动 | toast 提示"W3 已恢复"，避免工人没注意到延误 |
| **打印失败** | 红色 toast "打印失败：连接被拒绝"；打印队列 +1 | 不重试、不阻塞；工人可再按一次补打 |
| **打印机离线** | 打印按钮置灰 + "打印机离线" | 落筒主流程完全不受影响 |
| **重复落筒**（PLC 连发两次） | 后端 `points.go` 已做去重（按 doff_no + 时间窗）；页面不额外处理 | 页面只显示，不去重 |
| **落筒数 ≠ 8**（缺锭） | 列表里该条标黄"缺 1 锭" | 需要中心侧核对；页面不做修正入口 |

### 3.6 状态定义

| 元素 | 状态 | 视觉 |
| --- | --- | --- |
| 锭位 | 卷绕中 | 灰边框 + 蓝色进度条 |
| 锭位 | 满筒待落 | 绿边框 + 绿条 + "满筒 · 等待落筒" |
| 落筒记录 | 完整 | 绿色 tag "完整" |
| 落筒记录 | 缺锭 | 黄色 tag "异常" |
| 落筒按钮 | 无满筒 | 可点，但点了提示"当前没有满筒锭位"（不做 disabled，否则工人会以为页面卡了） |

---

## 4. 工位二：外检 / 分选 sorting

### 4.1 页面目标

一锭一扫、一下定级。**扫码是唯一输入源，工人不能手打编码。**

### 4.2 进入条件

- `edge.type === "sorting"`
- 该工位有 Macsa 打标机，`app.go` 里 `SetRuntime` 注入了 `printer.Client` 和 `plc.Manager`，打标握手由 `sorting/printing.go` 的 `PrinterController` 负责。

### 4.3 主流程

```
1. 扫码枪读入条码 → 输入框自动获得焦点，收到回车即提交查询
   ↓ 调用：GET /bobbins/:id（或按条码查，见 Q3）
   ↓ 验证：页面显示丝饼号、批号、落筒号、称重值
2. 工人按等级大按钮（AAA / AA / AA1 / AA2 / A1 / A）
   ↓ 调用：POST /quality/sorting  { bobbin_id, grade, defect_code }
   ↓ 验证：toast "已提交 AA"；最近分拣列表新增一条；等级分布计数 +1
3. （可选）先点缺陷代码（可多选），再点等级
   ↓ defect_code 传逗号拼接串或第一个代码（见 Q4）
4. PLC 侧：Start[i]=1 → 控制器打印 → 回写 Done[i]=1
   ↓ 页面显示 Start/Step/Done 三个握手值，让维修工能判断卡在哪一步
```

### 4.4 称重

`POST /quality/weighing  { bobbin_id, weight }` 由 `setWeighing` 处理，会调用 `evaluateWeightGrade` 自动算重量等级，再 `computeFinalGrade` 合成最终等级。

**称重值从哪来**（秤走串口 / PLC 模拟量 / 手工输入）代码里没有体现，属待确认项（Q5）。原型里按"PLC 自动带出"画的。

### 4.5 异常分支

| 场景 | 页面表现 | 处理 |
| --- | --- | --- |
| **扫码无效**（条码不存在） | 红色 toast "条码不存在：xxx"；输入框清空并保持焦点 | `GET /bobbins/:id` 返回 404 时；**不弹模态框**，否则工人要先点掉弹窗才能继续扫 |
| **重复扫码** | 黄色 toast "这一锭已经扫过了：xxx"；不提交 | 前端记住最近一次的 bobbin id 比对；后端 `setSorting` 没有唯一性保护，重复提交会覆盖等级 |
| **扫码枪没读到** | 输入框保持空 + 占位提示 | 回车且内容为空 → 提示"没有读到条码，请重新扫描" |
| **断网** | 顶部红条 | 分拣提交写本地，计数 |
| **打标机离线** | 握手区显示"打标机 L04-1 离线" | 允许提交等级（数据不能丢），但提示"等级已记录，标签未打印" |
| **PLC 打标超时** | 握手区高亮 `Start=1 / Done=0` 超过 5 s → 变红 | 显示"打标超时，请检查打标机"；不自动重试（会重复打标） |
| **同步中** | 顶部黄条 | 禁止提交（`syncing` 期间按钮置灰），避免本地队列和上行队列打架 |

### 4.6 状态定义

| 元素 | 状态 |
| --- | --- |
| 丝饼 | 待分拣（`sorting_grade` 为空） |
| 丝饼 | 已分拣（有 `sorting_grade`） |
| 丝饼 | 已定级（有 `final_grade`） |
| 缺陷按钮 | 未选 / 已选（黄色边框） |
| 等级按钮 | 未选 / 选中（蓝色边框，提交后自动清除） |

---

## 5. 工位三：包装 packaging

### 5.1 页面目标

扫锭上料 → 看栈板装载位置 → 出垛打标。**工人只需要扫，不需要填任何字段。**

### 5.2 进入条件

- `edge.type === "packaging"`
- `app.go` 里 `SetRuntime` 注入后，`PalletizerController` 负责"打包机堆完一垛"的 PLC 握手。

### 5.3 主流程

```
1. 页面加载 → GET /packing-orders?page=1&page_size=20 → 找到 status=in_progress 的订单
   ↓ 验证：顶部显示订单号、批号、等级、已完成/目标
2. 工人逐锭扫码
   ↓ 累积到本地数组
   ↓ 调用（扫满一垛或按"完成上垛"时）：POST /bobbins/intake
     { bobbin_ids: [...], palletizer_id, replace }
   ↓ 验证：返回每条丝饼的校验结果；失败项逐条列出，不遇错即停
3. 打包机 PLC 置 PalletComplete=1 → 控制器读批号/等级/锭数/层数/重量
   → 打 pallet.zpl → 回写 PalletAck=1
   ↓ 页面显示 PalletComplete / PalletAck 握手值
4. （打包机没读到时）工人按「完成上垛」手动补一垛
   ↓ 调用：POST /pallets/complete
     { pallet_code, lot_id, grade_code, bobbin_count, layers, total_weight, packing_order_id }
   ↓ 验证：返回栈板记录；订单 packed_quantity 按锭数累加；达到目标自动 completed
5. 需要补打 → POST /edge/print  { printer_code: "label-pkg-01", template_name: "pallet.zpl", data: {...} }
```

### 5.4 栈板装载位置

原型画的是 **6 列 × 4 层 = 24 格，每格 2 锭 = 48 锭/垛**。

**这个数字是假设的**，来自堆垛常见布局，代码里 `Palletizer.Layers` 是从 PLC 读的，设备型号不同布局也不同。必须现场核对（Q6）。

- 当前装载格：蓝色高亮
- 已装满格：绿色
- 未开始格：灰色
- 库位/格位变化时闪一下（`cellflash` 动画 0.8 s）

### 5.5 异常分支

| 场景 | 页面表现 | 处理 |
| --- | --- | --- |
| **扫码无效** | 红色 toast；不影响已扫入的锭 | `intakeBobbins` 逐条校验返回，页面把失败码列出来 |
| **重复扫码** | 黄色 toast "已在本垛中"；不入数组 | 前端数组去重 |
| **换批次/换垛** | 按「换垛清空」二次确认 | `intake` 带 `replace: true` 才会清掉本工位旧标记；不带会把上一盘残留算进下一垛 |
| **本垛已满** | 拒绝继续扫，提示"本垛已满 48 锭，请先完成上垛" | 防止多扫 |
| **打包机断连** | 握手区显示 `PalletComplete` 冻结 + 红点 | 允许手动「完成上垛」兜底，否则产线堵死 |
| **出垛重复**（PLC 连发） | 后端 `completePacking` 幂等：同 `pallet_code` 直接返回已有记录 | 页面不额外处理；但 `pallet_code` 必须由 PLC 或规则生成，不能让工人手输 |
| **订单已满** | 进度条变绿 + "订单已完成" | 拒绝再上料，提示换订单 |
| **打印失败** | 红色 toast；栈板记录已落库（先落库再打印） | 标签可补打，数据不能丢 |

### 5.6 状态定义

| 元素 | 状态 |
| --- | --- |
| 包装订单 | `pending` / `in_progress` / `completed` |
| 栈板 | `created`（`completePacking` 写死）→ 后续由中心推进 |
| 上料 | 本垛已扫 N 锭（前端状态，不上报，直到提交） |
| 格位 | 空 / 已装 / 当前装载位 |

---

## 6. 工位四：仓库 warehouse

### 6.1 页面目标

入库确认、出库、查库位。**库位坐标必须来自 PLC 或扫码，不能手填 row/col/layer。**

### 6.2 进入条件

- `edge.type === "warehouse"`
- `warehouse/controller.go` 处理 PLC 库位信号：`SlotChange=1` → 读 Action/Row/Col/Layer → 落库 → 回写 `SlotAck=1`。

### 6.3 主流程

```
1. 页面加载 → GET /warehouses → 取第一个仓库
   ↓ GET /warehouses/:id/slots、GET /warehouses/:id/stats
2. 库位网格按 row_no / col_no / layer_no 渲染
   ↓ 状态：empty 灰 / occupied 绿 / reserved 黄
3. 入库：
   a. 扫托盘码 → 得到 pallet
   b. 按「入库确认」→ 自动分配库位
      → POST /warehouses/:id/allocate（返回第一个 empty 槽并置 reserved）
      → POST /warehouses/:id/inbound
        { row, col, layer, occupant_type, occupant_id, lot_id }
   ↓ 验证：网格该格变绿；stats 的 occupied +1、empty -1
4. 出库：
   a. 扫托盘码 → 找到占用它的库位
   b. 按「手动出库」→ 二次确认
      → POST /warehouses/:id/outbound  { row, col, layer }
   ↓ 验证：网格该格变灰；stats 反向变化
```

### 6.4 异常分支

| 场景 | 页面表现 | 处理 |
| --- | --- | --- |
| **断网** | 顶部红条 | 入库/出库**继续允许**（本地 SQLite），但要标记待同步 |
| **PLC 断连** | 网格冻结，标题旁显示"数据延迟" | 允许手动操作，但操作前提示"库位状态可能不是最新的" |
| **库位已被占用** | 红色 toast "库位不可用" | 后端返回 409 `slot not available`；页面不重试 |
| **出库空库位** | 红色 toast "库位不是占用状态" | 后端返回 409 `slot is not occupied` |
| **没有空闲位** | 红色 toast "没有空闲库位" | `allocate` 返回 404 `no empty slot available` |
| **重复入库** | `SlotChange` 已处理的坐标不重复产生事件 | 由 `controller.go` 保证；页面不处理 |
| **扫码托盘不存在** | 红色 toast + 输入框清空 | 找不到时不允许继续 |

### 6.5 状态定义

| 元素 | 状态 |
| --- | --- |
| 库位 | `empty` 灰 / `occupied` 绿 / `reserved` 黄 |
| 库位 | 3 天无变化 → 斜纹（需加 `updated_at` 计算，属增强项） |
| 出入库记录 | 入库 / 出库 |

---

## 7. 页面元素 → 数据来源 → 后端接口

所有路径前缀 `/api/v1`（`web/src/api/http.ts` 的 `baseURL`）。标"**待实现**"的是调研中确认**不存在**的端点。

### 7.1 落筒 winding

| 页面元素 | 数据来源 | 接口 | 方法 | 状态 |
| --- | --- | --- | --- | --- |
| 当前批号 / 规格 | 本地 `lots` 表 | `/lots?status=active` | GET | 存在 |
| 本次落筒进度（N/8 锭） | 本地 `doffings.bobbin_count` 或 PLC 快照 | `/doffings?page=1&page_size=1` | GET | 存在 |
| 今日产量（落数 / 锭数） | 本地 `doffings` 按日聚合 | `/stats/daily-output?days=1` | GET | 存在（返回 `[{date, doff_count, bobbin_count}]`） |
| 累计统计 | 本地表 count | `/stats/winder` | GET | 存在（返回 `total_doffings/total_bobbins/today_doffs/today_bobbins`） |
| 锭位实时长度 | PLC 点表 | **无直读接口** | — | **待实现**，见 7.6 |
| 锭位状态（满筒/卷绕） | PLC | 同上 | — | 待实现 |
| 最近落筒列表 | 本地 `doffings` | `/doffings?page=1&page_size=10` | GET | 存在 |
| 料车装载 | 提交 | `/trolleys/load` | POST | 存在，body `{number, rfid?, lot_id, doffing_ids, line_id?}` |
| 料车状态列表 | 本地 `trolleys` | `/trolleys?page=1&page_size=10` | GET | 存在（返回裸数组，非 `items`） |
| 落筒完成确认 | 见 Q2 | `/doffings` | POST | 存在，body `{line_id, winder_name, doff_no, lot_id, shift_id?, bobbin_count}` |
| 丝饼列表 | 本地 `bobbins` | `/bobbins?page=1&page_size=50` | GET | 存在 |
| 单个丝饼 | 本地 | `/bobbins/:id` | GET | 存在 |
| 标签补打 | 模板 + 打印机 | `/edge/print` | POST | 存在，body `{printer_code, template_name, data}` |
| 打印机在线状态 | 打印客户端 | `/edge/printers` | GET | 存在，返回 `[{code, host, port, online, last_check}]` |
| 手动干预 | — | **无** | — | **待实现**（`Index.vue` 里的按钮是空函数） |
| 网络状态 | 同步引擎 | `/edge/sync/status` | GET | 存在，返回 `{online, queue_depth}` |
| PLC 设备状态 | PLC 管理器 | `/edge/plc/status` | GET | 存在，返回 `{code: {connected, last_read_ms, host, port, protocol, model, errors}}` |
| 工位标识 / 线路 | 配置 | `/edge/info` | GET | 存在，返回 `{edge_id, type, line, mode, online, ha}` |
| 工位识别（按 IP） | 配置 | `/station/identify` | GET | 存在（边端用不上，`embed.go` 已改成注入方式） |

### 7.2 外检 sorting

| 页面元素 | 数据来源 | 接口 | 方法 | 状态 |
| --- | --- | --- | --- | --- |
| 扫码丝饼 | 条码 → 丝饼 | `/bobbins/:id` | GET | 存在（**按 ID 查，不是按条码**，见 Q3） |
| 待检队列 | 本地 `bobbins` where `sorting_grade` 为空 | `/bobbins?page=1&page_size=30` | GET | 存在 |
| 称重值 | PLC 或秤 | — | — | **待实现**，见 Q5 |
| 重量等级 | 规则计算 | `/weighing-rules?lot_id=` | GET | 存在（返回规则数组） |
| 提交等级 | 提交 | `/quality/sorting` | POST | 存在，body `{bobbin_id, grade, defect_code?}` |
| 提交称重 | 提交 | `/quality/weighing` | POST | 存在，body `{bobbin_id, weight}` |
| 等级分布 | 本地统计 | `/quality/distribution?lot_id=` | GET | 存在，返回 `[{final_grade, count}]` |
| 等级选项 | 字典 | `/grades` | GET | 存在 |
| 缺陷选项 | 字典 | `/defects` | GET | 存在 |
| 批号列表 | 本地 `lots` | `/lots?status=active` | GET | 存在 |
| 料车列表 | 本地 `trolleys` | `/trolleys` | GET | 存在 |
| 打标握手 Start/Step/Done | PLC 点表 | 页面不直读 | — | **待实现**（目前只能从 `/edge/plc/read` 按点表读，见 7.6） |
| 打标机在线 | 打印客户端 | `/edge/printers` | GET | 存在 |

### 7.3 包装 packaging

| 页面元素 | 数据来源 | 接口 | 方法 | 状态 |
| --- | --- | --- | --- | --- |
| 当前包装订单 | 本地 `packing_orders` | `/packing-orders?page=1&page_size=20` | GET | 存在（页面自己按 `status=in_progress` 过滤） |
| 新建订单 | 提交 | `/packing-orders` | POST | 存在 |
| 开始订单 | 提交 | `/packing-orders/:id/start` | PUT | 存在 |
| 已打包 / 目标 | `packed_quantity` / `target_quantity` | 同上 | — | 存在 |
| 扫码上料 | 逐锭提交 | `/bobbins/intake` | POST | 存在，body `{bobbin_ids: [...], palletizer_id?, replace?}` |
| 当前在包装位的丝饼 | 本地 `bobbins` | `/bobbins/packing` | GET | 存在 |
| 完成上垛 | 提交 | `/pallets/complete` | POST | 存在，body `{pallet_code, lot_id, grade_code, bobbin_count, layers, total_weight?, packing_order_id?}` |
| 栈板标签 | 模板 | `/edge/print` | POST | 存在（`template_name: "pallet.zpl"`，模板在 `configs/labels/templates/`） |
| 最近栈板 | 本地 `pallets` | `/pallets?page=1&page_size=20` | GET | 存在 |
| 栈板装载位置 | 打包机 PLC | — | — | **待实现**，见 7.6 |
| 打包机握手 PalletComplete/PalletAck | PLC | — | — | 待实现 |
| 拣货单 | 本地 | `/picking-orders` | GET/POST | 存在 |
| 拣货开始/完成 | 提交 | `/picking-orders/:id/start` `/complete` | PUT | 存在 |
| 批号列表 | 本地 | `/lots` | GET | 存在 |

### 7.4 仓库 warehouse

| 页面元素 | 数据来源 | 接口 | 方法 | 状态 |
| --- | --- | --- | --- | --- |
| 仓库列表 | 本地 `warehouses` | `/warehouses` | GET | 存在 |
| 仓库详情 | 本地 | `/warehouses/:id` | GET | 存在 |
| 库位网格 | 本地 `warehouse_slots` | `/warehouses/:id/slots` | GET | 存在（裸数组） |
| 占用 / 空闲 / 预留 | 本地统计 | `/warehouses/:id/stats` | GET | 存在 |
| 分配库位 | 提交 | `/warehouses/:id/allocate` | POST | 存在，返回被置为 `reserved` 的 slot |
| 入库确认 | 提交 | `/warehouses/:id/inbound` | POST | 存在，body `{row, col, layer, occupant_type, occupant_id, lot_id?}` |
| 手动出库 | 提交 | `/warehouses/:id/outbound` | POST | 存在，body `{row, col, layer}` |
| 批号列表 | 本地 | `/lots` | GET | 存在 |
| 网络 / PLC / 打印机状态 | 通用 | 见 7.1 | — | 存在 |

### 7.5 通用（所有工位共用）

| 页面元素 | 接口 | 方法 | 说明 |
| --- | --- | --- | --- |
| 网络状态徽标 + 顶部条 | `/edge/sync/status` | GET | `{online, queue_depth}` |
| 设备状态区 | `/edge/plc/status` | GET | 每设备 `{connected, last_read_ms, errors, ...}` |
| 打印机状态 | `/edge/printers` | GET | 数组 |
| 主动检查打印机 | `/edge/printers/check` | POST | 点"测试打印"前先调一次 |
| 打印 | `/edge/print` | POST | `{printer_code, template_name, data}` 或 `{printer_code, raw_zpl}` |
| 工位标识 | `/edge/info` | GET | `type` / `line.code` / `line.name` |
| 现场配置待应用 | `/edge/config` | GET | 返回 `{version, applied, pending, problems}`，沿用 `Index.vue` 的交互（人工确认才生效） |
| 应用 / 放弃配置 | `/edge/config/apply` `/discard` | POST | 存在 |
| 模块启用状态 | `/system/modules` | GET | `{winding, quality, warehouse, packaging, ...}` |
| 健康检查（页面自检用） | `/health`（无 `/api/v1` 前缀） | GET | `{status, edge_id, type, online}` |
| 诊断信息 | `/edge/diagnostics` | GET | 维修工用，工位页不展示 |
| 升级状态 | `/edge/update/status` | GET | 工位页顶栏小图标即可 |

### 7.6 明确"待实现"的接口

这三个是所有工位都绕不过去的缺口，**不是设计取舍，是后端确实没有**：

| 缺口 | 影响 | 建议实现 |
| --- | --- | --- |
| **PLC 实时值直读**<br>`/edge/plc/status` 只给连接状态，不给点值 | 落筒锭位长度、包装机装载位置、外检握手值都没法"跳" | `GET /api/v1/edge/plc/:device/values` 返回最近一次轮询的解码快照（`plc.Manager` 里已有 `deviceSnapshot()`，加一个 `LastValues` 字段即可） |
| **标签模板清单** | 页面无法"选模板"，只能硬编码 | `GET /api/v1/edge/templates` 返回 `labels/templates/*.zpl` 的文件名列表（`printer.Client` 已有 `LoadTemplates`，记下文件名就行） |
| **重复扫码的唯一性保护** | 后端 `setSorting` 直接 `Updates`，重复提交会覆盖等级，且不报错 | `POST /quality/sorting` 加"已分拣则返回 409"；或前端在本地记 `sortedSet` 挡住（原型按前端挡） |

另外两个**非缺口但要注意**的：
- `/trolleys`、`/warehouses`、`/warehouses/:id/slots` 返回**裸数组**，不是 `{items: [...]}`。`Winding.vue` 里写的是 `trolleyRes.data?.items || []`，这条永远拿到空数组。
- `Sorting.vue` 调的 `/stats/grade-distribution` 不存在，正确路径是 `/quality/distribution`。

---

## 8. 待用户确认的问题清单

按重要性排序。前 5 个会直接改变页面结构，不定下来没法进入实现。

### Q1. 落筒页是否需要在"落筒完成前"就显示锭位？

现状：原型按"每个锭位的实时长度在涨"画的。
但 `points.go` 只在**落筒事件发生时**写 `doffings` / `bobbins`，**没有"卷绕进行中"的中间态**。要画这个动态，必须新增 PLC 实时值接口（7.6 第一条）。

- 如果**要**：需要后端补 `plc values` 接口，页面按 2 s 轮询 8 个锭位长度。
- 如果**不要**：落筒页只显示"已完成的落筒记录"，锭位盘改成"本次落筒 8 个锭位的确认/缺锭状态"，静态展示。

**浮浮酱倾向要**：工人不看锭位盘就不知道哪台机快满了。

### Q2. "落筒完成确认"按钮到底做什么？

PLC 已经自动写库了（`persistLocal`），那这个按钮是：

- A. 纯确认，记一个 `confirmed_at`，不改业务数据；
- B. 手动补录，调 `POST /doffings`（PLC 漏采时用）；
- C. 不要按钮，只在自动落筒时给一条 toast。

原型按 A 画的。如果选 B，要防重复；如果选 C，页面最下方那一排按钮会空出一块。

### Q3. 外检扫码是按"丝饼 ID"还是"条码"查？

后端只有 `GET /bobbins/:id`（主键 id）。如果扫码枪扫出来的是 `W3-0245` 这种业务编码，需要：

- 新增 `GET /bobbins/by-code/:code`，或
- 让扫码枪输出纯数字 id（现场可行性存疑）。

**这个不定，外检页面第一步就走不通。**

### Q4. 缺陷代码和等级的关系？

现在 `POST /quality/sorting` 的 `defect_code` 是**单值字符串**（`sortingReq.DefectCode string`），但原型画的是**多选**。

三个选项：

- A. 单选：点一个缺陷，再点等级；
- B. 多选：前端把多个代码拼成 `MAO,YOU` 传过去，后端 `computeFinalGrade` 要能解析；
- C. 缺陷代码只做记录，不参与定级。

**另外：** 选了缺陷代码，等级是否自动降级？还是让工人自己决定？`computeFinalGrade` 目前的逻辑只看 `sorting_grade` + `weight`，不看 `defect_code`。

### Q5. 称重值从哪里来？

`POST /quality/weighing` 需要 `weight`，但代码里没有"秤"的数据源。三种可能：

- A. 电子秤走串口，边端加一个串口读取模块；
- B. 秤接入 PLC 模拟量，从点表读；
- C. 工人手输（**与"禁止手打"的约束冲突**）。

原型按 B 画的（称重块显示 5.412 kg）。如果实际是 A，需要在 `app.go` 里加一个串口设备。

### Q6. 包装栈板布局到底是几层几列？

原型画的是 6 列 × 4 层 = 24 格，每格 2 锭 = 48 锭/垛。这个数字纯属假设。

需要现场确认：

- 每垛多少个锭位？（48？）
- 每层几个位置？
- 打包机能不能读到"当前装到第几格"？（`Palletizer` 点表里只有 `Layers`，没有列位置）

**如果 PLC 给不出列位置**，格子网格就画不出来，只能退化成一个"第 2/4 层，已装 24/48 锭"的进度条。

### Q7. 外检返修的完整流程是什么？

现在只有"分拣等级"和"称重"，没看到"返修"概念。需要确认：

- 判 A 级的丝饼，是直接报废、退回卷绕、还是单独堆一垛？
- "返修"要不要在工位页面上有入口？
- 返修后再检，等级怎么覆盖？（现在 `setSorting` 是直接覆盖）

### Q8. 仓库的"手动入库/出库"由谁操作？

两种可能：

- A. PLC 自动记录出入库（`controller.go` 已经在做），工人只在异常时手动纠正；
- B. 工人每次都手动点，PLC 只做辅助校验。

如果是 A，"手动入库/出库"应该是**低调的次要按钮 + 二次确认**（原型这么画的）。
如果是 B，两个按钮要升为主操作（112 px），并且要有"扫托盘 → 自动定位库位"的完整流程。

### Q9. 料车号怎么输入？

现在原型是"按料车装载 → 弹窗选/扫"。但 `POST /trolleys/load` 要 `number`（整数）和可选的 `rfid`。

- 料车上有 RFID 吗？还是条码？还是只有编号？
- 如果只有编号，工人怎么输入？**这一项会打破"禁止手打"约束**。

### Q10. 打印机模板名要不要让工人选？

原型画的是"模板固定为 `bobbin.zpl`，只显示不选"。

- 现场会不会有多种标签（比如换规格要换模板）？
- 如果要选，需要后端补模板清单接口（7.6），并且要加一个"模板选择"的下拉——但下拉在触摸屏上很难点，可能需要改成"大按钮组"。

---

## 9. 本次没有做的事

明确划界，避免后续误以为漏了：

1. **没改任何 `web/src/` 下的代码**，包括 `station/*.vue`、`router/index.ts`。
2. **没提交 git**。
3. 原型里的假数据、假 PLC 跳动、假握手时序都是 `setInterval` 造的，不是真实采集；所有"已提交""已同步"的 toast 都只改 DOM。
4. 原型故意保留了"工位切换"和"模拟断网"两个开发用控件，真实实现必须去掉（工位由 `edge.type` 决定，断网是网络事实不是开关）。
5. `Index.vue` 里那套"现场配置待应用"的卡片没有画进原型。它属于运维页面而不是工位操作页面，建议单独出一个 `Ops.vue`，或者收进 `/edge/config` 的独立路由。这块现有的交互逻辑（暂存 → 人工确认 → 应用）是对的，不用重做。
