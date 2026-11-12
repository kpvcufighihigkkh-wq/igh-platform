# 监控与告警运维手册

本文说明 IGH Platform 的 Prometheus 监控怎么起、告警怎么分级、收到告警后第一件事做什么，
以及**目前哪些指标缺失、需要补**。

> 前置阅读：`deployments/prometheus.yml`（采集配置）、
> `deployments/prometheus/alerts.yml`（告警规则）。

---

## 1. 现状：监控能看什么，不能看什么（先看这段）

这一节比后面的操作步骤更重要。**不清楚这套监控的边界，值班时会误判。**

指标定义全部来自 `internal/server/metrics/metrics.go`。该文件定义了 9 个指标，
但截至当前版本，全仓库只有 2 个真正被写入（`internal/server/middleware/metrics.go`）：

| 指标 | 类型 | 标签 | 状态 |
|---|---|---|---|
| `igh_http_requests_total` | Counter | method, path, status | ✅ 真实可用 |
| `igh_http_request_duration_seconds` | Histogram | method, path | ✅ 真实可用 |
| `igh_edge_events_total` | CounterVec | edge_id, event_type | ❌ 从未写入，**不导出任何序列** |
| `igh_edge_nodes_online` | Gauge | — | ❌ 从未写入，恒为 0 |
| `igh_active_alarms` | Gauge | — | ❌ 从未写入，恒为 0 |
| `igh_doffings_total` | Counter | — | ❌ 从未写入，恒为 0 |
| `igh_bobbins_total` | Counter | — | ❌ 从未写入，恒为 0 |
| `igh_plc_connected` | GaugeVec | edge_id, device_code | ❌ 从未写入，**不导出任何序列** |
| `igh_warehouse_occupancy` | GaugeVec | warehouse_code, status | ❌ 从未写入，**不导出任何序列** |

**两种失效方式必须分清**（这是本手册最容易踩的坑）：

- **非 Vec 的 Gauge/Counter**（`igh_edge_nodes_online` 等）在注册时即导出初值 0。
  所以 `igh_edge_nodes_online == 0` 这种规则会**永久告警**，不是"没数据"，而是天天误报。
- **Vec 类型**在没有任何 `WithLabelValues` 调用时，连 `# HELP` / `# TYPE` 都不导出。
  所以 `absent()` 或 `== 0` 匹配不到任何序列，告警**永远不会触发**——看起来有规则，实际是瞎的。

结论：**任何新写的告警规则，必须先在真实 `/metrics` 输出里确认该指标存在**：

```bash
curl -s http://127.0.0.1:8099/metrics | grep igh_
```

另外，`client_golang` 默认注册表自带以下指标，可直接用于告警（已实测导出）：

- `up{job,instance}` —— 抓取是否成功
- `go_*` —— 35 条，含 `go_goroutines`、`go_gc_duration_seconds` 等
- `process_*` —— 6 条，含 `process_resident_memory_bytes`、`process_open_fds`、
  `process_max_fds`、`process_start_time_seconds` 等
- `promhttp_*` —— `/metrics` 处理器自身的请求统计

**还有一处概念性区别**：`/health` 返回的 `database.healthy`、`database.generation`、
`ha.role`、`ha.peer_alive`、`uptime_sec` 是 **JSON 字段，不是 Prometheus 指标**。
Prometheus 抓不到它们，所以**无法直接告警**。目前数据库相关告警全部缺失，
原因就在这里（见第 5 节）。

### 1.1 写规则时最容易静默失效的三种写法

本仓库的 `deployments/prometheus/alerts.test.yml` 用 `promtool test rules`
实际验证过下面三个坑，**每一条都曾让规则"看起来配好了，实际永远不响"**。
改规则时请照这个清单自查，并跑一遍测试文件。

1. **空向量参与比较，结果还是空向量。**
   序列彻底消失（例如边端不再上报）时，`sum(rate(...))` 返回的是空向量而不是 0，
   于是 `... == 0` 什么都不匹配。修法：`( sum(...) or vector(0) ) == 0`。
   `IghAllCenterInstancesDown` 就有这个问题：两台全挂时 `count(up == 1)` 是空向量，
   不加 `or vector(0)` 这条最关键的告警永远不会触发。

2. **指标自身被算进了判据。**
   Prometheus 每 15 秒抓一次 `/metrics`，这个请求同样计入
   `igh_http_requests_total{path="/metrics"}`。所以"没有任何流量"这类规则
   必须排除 `/metrics` 与 `/health`，否则 rate 永远大于 0。

3. **`unless` 的语义与直觉相反。**
   `A unless B` 是"取 A 中不在 B 里的部分"。写"心跳停了但整体没停"时，
   如果 B 用的是全站流量，那么心跳停、其它流量正常时 B 为假，`unless` 反而会把 A 吃掉，
   规则永远不触发。正确写法是用 `and` 加一个正向条件：
   `(心跳速率或0) == 0 and on() (其它业务流量) > 0`。

规则改动后必须跑：

```bash
docker run --rm -v "$PWD/deployments/prometheus:/etc/prometheus:ro" \
  --entrypoint promtool prom/prometheus:v2.53.0 \
  test rules /etc/prometheus/alerts.test.yml
```

---

## 2. 启动 Prometheus + Alertmanager

### 2.1 目录约定

```
deployments/
├── prometheus.yml              # 主配置
└── prometheus/
    ├── alerts.yml              # 告警规则
    ├── alerts.test.yml         # 规则单元测试（promtool test rules）
    └── targets/
        └── center.json         # file_sd 目标清单（双机热备的两台都列在这里）
```

容器内路径固定为：

```
/etc/prometheus/prometheus.yml
/etc/prometheus/alerts.yml
/etc/prometheus/targets/center.json
```

`prometheus.yml` 里用的就是这些绝对路径，**换目录时三处要一起改**，否则会报
`rule_files does not point to an existing file`。

### 2.2 部署到现场服务器

```bash
# 1) 放置配置
sudo mkdir -p /etc/prometheus/targets
sudo cp deployments/prometheus.yml            /etc/prometheus/prometheus.yml
sudo cp deployments/prometheus/alerts.yml     /etc/prometheus/alerts.yml
sudo cp deployments/prometheus/targets/center.json /etc/prometheus/targets/center.json
# 测试文件不是运行必需的，但和规则放在一起便于现场改完规则立刻自测
sudo cp deployments/prometheus/alerts.test.yml /etc/prometheus/alerts.test.yml

# 2) 上线前先校验（必做，不要跳过）
docker run --rm \
  -v /etc/prometheus:/etc/prometheus \
  --entrypoint promtool prom/prometheus:v2.53.0 \
  check config /etc/prometheus/prometheus.yml

docker run --rm \
  -v /etc/prometheus:/rules:ro \
  --entrypoint promtool prom/prometheus:v2.53.0 \
  check rules /rules/alerts.yml

# 2b) 规则语义测试（比 check 更重要）
# check 只验证表达式能解析；test 验证它在给定数据下"会不会真的响"。
# 目录整体挂在 /etc/prometheus，因为测试文件里 rule_files 写的是
# /etc/prometheus/alerts.yml（与 CI 保持一致）。
docker run --rm \
  -v /etc/prometheus:/etc/prometheus:ro \
  --entrypoint promtool prom/prometheus:v2.53.0 \
  test rules /etc/prometheus/alerts.test.yml

# 3) 启动
docker run -d --name igh-prometheus \
  --restart=always \
  -p 9090:9090 \
  -v /etc/prometheus:/etc/prometheus:ro \
  -v igh-prometheus-data:/prometheus \
  prom/prometheus:v2.53.0 \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.retention.time=30d
```

`--storage.tsdb.retention.time=30d`：现场排查往往要回溯两三周前的现象，
默认 15 天偏短；磁盘预算按每天约 50–100MB（数据量小时）估算。

### 2.3 本机开发环境

仓库根目录的 `docker-compose.yml` 已经有 prometheus（9090）与 grafana（3001）服务，
但它挂载的还是旧路径，**需要同步改**：

```yaml
  prometheus:
    image: prom/prometheus:v2.53.0      # 固定版本，不要用 latest
    ports:
      - "9090:9090"
    volumes:
      # 规则目录整体挂进来：prometheus.yml 里写的是 /etc/prometheus/alerts.yml
      # 与 /etc/prometheus/targets/center.json，直接对齐容器内路径，
      # 避免出现"本地能跑、线上路径不对"的差异。
      - ./deployments/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./deployments/prometheus:/etc/prometheus:ro
```

> 注意：compose 里 `host.docker.internal` 只在 Docker Desktop 上可用；
> Linux 宿主机需要额外加 `extra_hosts: ["host.docker.internal:host-gateway"]`。
>
> 本机开发时把 `prometheus.yml` 的 `igh-server` job 换成文件末尾注释里的
> `host.docker.internal:8099` 写法（开发配置 `configs-dev3` 是 8101，
> HA 模拟 `configs-ha-sim` 是 8103）。

### 2.4 Alertmanager

最小可用配置（`/etc/alertmanager/alertmanager.yml`），先用邮件打底：

```yaml
route:
  # 按 severity 分流：critical 立即发，warning 合并后发
  receiver: 'default'
  group_by: ['alertname', 'instance', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    - matchers: ['severity="critical"']
      receiver: 'oncall'
      group_wait: 10s
      repeat_interval: 1h
    - matchers: ['severity="warning"']
      receiver: 'default'
      repeat_interval: 12h

receivers:
  - name: 'default'
    email_configs:
      - to: 'ops@example.com'
        from: 'alert@example.com'
        smarthost: 'smtp.example.com:587'
  - name: 'oncall'
    email_configs:
      - to: 'oncall@example.com'
        from: 'alert@example.com'
        smarthost: 'smtp.example.com:587'
```

`repeat_interval` 的取舍：critical 1 小时重复提醒（没处理就再响一次），
warning 12 小时（避免同一件事一天响十几次，让人开始无视告警）。
告警疲劳比没有告警更危险。

把 `prometheus.yml` 的 `alerting.alertmanagers.targets` 改成现场实际地址后重启 Prometheus。

---

## 3. 告警分级

| 级别 | 判定标准 | 通知方式 | 响应时限 |
|---|---|---|---|
| **critical** | 业务已受影响或即将中断，不处理就出事 | 立即呼叫人（电话/短信/群 @所有人） | 15 分钟内响应 |
| **warning** | 冗余已降低或趋势劣化，还有缓冲时间 | 邮件/群消息，不呼叫 | 当班内处理 |

现有规则共 14 条（`promtool check rules` 可复核）：

**critical（5 条）**
- `IghServerDown` —— 单台中心实例抓不到
- `IghAllCenterInstancesDown` —— 两台都抓不到，业务整体中断
- `IghHTTP5xxRateHigh` —— 单个接口 5xx 比例超标
- `IghHTTP5xxOverallHigh` —— 整体 5xx 比例超标
- `IghEdgeHeartbeatStopped` —— 边端心跳全停

**warning（9 条）**
- `IghNoIncomingTraffic` —— 进程活着但收不到业务流量
- `IghHTTPLatencyP99High`、`IghHTTPPathCardinalityHigh`
- `IghEdgeUploadStopped`、`IghEdgeIngestErrors`
- `IghHighMemoryUsage`、`IghGoroutineLeak`、`IghHighFDUsage`、`IghServerRestarting`

> 为什么 `IghServerDown` 单台也是 critical：热备下业务暂时不受影响，
> 但冗余已经归零。这个状态下任何一个小故障都会变成全线停机，
> 必须在当天处理完，不能留到第二天。

---

## 4. 每条告警的现场排查动作（第一件事）

按"先确认现象、再定位层次"的顺序做，**不要一上来就重启服务**——
重启会把证据清掉，同一个问题下次还会来。

### `IghServerDown`
1. `ssh` 到告警里的 instance，`systemctl status igh-server`（Linux）
   或 `sc query igh-server`（Windows）。
2. 进程不在 → 看 `logs/` 最后 200 行有没有 panic / fatal。
3. 进程在但端口不通 → `ss -lntp | grep 8099` 确认监听，
   再 `curl -s http://127.0.0.1:8099/metrics | head` 本机自测。
4. 本机能通、Prometheus 不通 → 查防火墙与 Prometheus 到该机的网络。

### `IghAllCenterInstancesDown`
1. 先确认是不是 Prometheus 自己的问题：打开 `http://<prom>:/targets`，
   看所有目标是不是同时变红。同时红通常是抓取侧故障（Prometheus 挂了/配置错）。
2. 再看 VIP：`ip addr show | grep <VIP>` 在哪台机器上。
   VIP 所在的机器没在跑 igh-server，是典型的"流量漂到空节点"。
3. 两台机器交叉验证：互相 `curl` 对方的 8099。

### `IghNoIncomingTraffic`
**这条不表示服务坏了，表示服务收不到流量**，排查方向与"挂了"完全不同。
1. `curl -s http://127.0.0.1:8099/api/v1/station/identify` 本机打一发，
   观察 `igh_http_requests_total` 是否增长（`curl /metrics | grep igh_http_requests_total`）。
   增长 → 服务正常，问题在上游（VIP/代理/前端）。
2. 检查 VIP 是否漂到了别的节点：`arping` / `ip neigh` 查 VIP 对应的 MAC。
3. 检查前端页面是否整体打不开（可能是 nginx 或前端资源问题）。

### `IghHTTP5xxRateHigh` / `IghHTTP5xxOverallHigh`
1. 看 `path` 标签定位是哪个接口，先查这个接口近 10 分钟的日志 error 行。
2. **立刻查数据库**：`curl -s http://127.0.0.1:8099/health | python -m json.tool`，
   看 `database.healthy` 与 `database.primary`。
   `healthy: false` 说明数据库不可写，是 5xx 最常见的原因。
3. 若 `healthy: true` 但 generation 刚变过，说明刚发生过主备切换，
   5xx 可能是切换瞬间的连带影响，观察 5 分钟是否自愈。

### `IghHTTPLatencyP99High`
1. 查数据库慢查询（`pg_stat_activity` 里 `state='active' and now()-query_start > interval '5 second'`）。
2. 确认归档任务是否正在跑（每天 3 点，`archive.enabled: true` 时）。
   归档删除会短暂持锁，与在线写入抢锁会拉高延迟。
3. 看 `process_open_fds` 是否接近上限（连接池打满）。

### `IghHTTPPathCardinalityHigh`
1. 判断是不是在被人扫描：`curl -s /metrics | grep igh_http_requests_total | grep -v '/api/'`
   —— 出现大量奇怪路径即是。
2. 若是扫描，考虑在防火墙层拦掉；若路径看着像正常业务但很零散，
   说明前端有拼错的 URL，需要前端排查。
3. 指标本身要修：`middleware/metrics.go` 里 `c.FullPath()` 为空时回退到
   `c.Request.URL.Path`，这会让每个未匹配路径都成为一个新标签值。
   建议改为统一记为 `__unmatched__`（见第 5 节）。

### `IghEdgeHeartbeatStopped`
1. 确认边端 PC 是否在线（ping）。
2. 边端进程：任务管理器 / `systemctl status igh-edge`。
3. 若边端在跑但仍无心跳，查边端的服务端地址配置与网络连通性：
   `curl -X POST <center>/api/v1/edge/heartbeat`（会返回 401，返回 401 说明网络通）。
4. **注意**：这条是全局限判断（不含 edge_id 标签），
   意味着它只在**所有**边端都停时才触发。单台边端掉线目前无法告警（见第 5 节）。

### `IghEdgeUploadStopped`
心跳在、上传停 → 边端"活着但不干活"，多半是本地队列积压或插件卡死。
1. 查该边端的 `queue_depth`：中心 `/api/v1/edge` 列表接口，
   或直接查库 `SELECT code, queue_depth, last_heartbeat FROM edge_nodes;`。
2. 边端日志里找 `sync` / `queue` 相关错误。
3. 检查 PLC 连接：边端往往因为采集不到数据而停止产生事件。

### `IghEdgeIngestErrors`
1. 看 status 标签：401/403 → 边端 API-Key 与 `project.yaml` 的 `server.api_key` 不一致。
2. 5xx → 查中心日志与数据库可写性。
3. 400 → 边端 SDK 版本与中心接口不匹配（常见于边端升级未完成）。

### `IghHighMemoryUsage`
1. `curl http://127.0.0.1:8099/debug/pprof/heap > heap.out`（如已开 pprof）。
2. 对比一小时前是否持续增长；稳定在阈值附近说明只是机器本身较忙，
   线性上涨才是泄漏。
3. 应急：记录当前 `/metrics` 与日志，再重启服务，之后用 heap 快照定位。

### `IghGoroutineLeak`
1. `curl /debug/pprof/goroutine?debug=1`，看数量最多的函数栈。
2. 常见来源：未关闭的数据库行、未退出的 goroutine 循环、WebSocket 连接未释放。
3. 与 `IghHighMemoryUsage` 同时出现时，优先按 goroutine 泄漏处理。

### `IghHighFDUsage`
1. `ls /proc/<pid>/fd | wc -l` 确认数量，`ls -l /proc/<pid>/fd | awk '{print $NF}' | sort | uniq -c | sort -rn` 看是哪个类型占的。
2. 大量 socket → 连接泄漏（HTTP 客户端未 Close、WebSocket 未断开）。
3. 应急可临时调大 `LimitNOFILE`（`igh-server.service` 里已是 65536），
   但这是治标，必须找到泄漏点。

### `IghServerRestarting`
1. `journalctl -u igh-server --since "2 hours ago" | grep -iE "panic|fatal|OOM"`.
2. `dmesg | grep -i "killed process"` —— 是否被 OOM killer 杀。
3. 若是 Windows 服务：事件查看器 → Windows 日志 → 应用程序。

---

## 5. 缺失但重要的指标（按优先级）

这一节是后续开发的输入。**每条都写明了该在哪加**，可以直接当任务派发。
加完后请同步删除 `alerts.yml` 里对应的 `TODO(需补指标)` 注释块。

### P0 —— 没有它就无法对最严重的故障告警

**1. 数据库可写性 / 主备状态**
- 指标：`igh_db_healthy`(Gauge)、`igh_db_generation`(Gauge)、`igh_db_is_primary`(Gauge)
- 位置：`internal/server/database/failover.go` 的 `watch()` 循环里 Set，
  数据源就是现成的 `p.healthy` 与 `p.generation`（`Status()` 已经读它们）。
- 现状：这些值只在 `/health` 的 JSON 里，Prometheus 完全看不到。
- 后果：数据库被围栏（只读）时进程健康、端口正常、`up` 为 1，
  **现有监控发现不了**，只有等到大量 5xx 才暴露，已经影响产线了。

**2. 数据库连接池**
- 指标：直接注册 `sqlstats.NewDBStatsCollector`，得到
  `sql_db_open_connections`、`sql_db_in_use_connections`、`sql_db_wait_*` 等。
- 位置：`internal/server/database/database.go` 中 `gorm.Open` 之后，
  对 `pool.DB()` 注册 collector。
- 后果：连接池打满时表现为请求全部排队，延迟飙升但错误率可能不升，
  是最难排查的一类故障。

**3. 边端在线数量（全局）**
- 指标：`igh_edge_nodes_online`（指标已定义，只差写入）
- 位置：`internal/server/service/edge.go` 的 `Heartbeat()` 里，
  或用一个每分钟跑的定时任务 `COUNT(*) WHERE status='online'`。
- 后果：`igh_edge_nodes_online` 现在恒为 0，你既无法知道有几台边端在线，
  也无法发现"边端从 12 台掉到 2 台"这种渐进式故障——
  现有的 `IghEdgeHeartbeatStopped` 只在**全部**停掉时才响。

### P1 —— 直接对应业务损失

**4. 边端本地队列深度（按 edge_id）**
- 指标：`igh_edge_queue_depth{edge_id}`(Gauge)
- 位置：`Heartbeat()` 里 `req.QueueDepth` 已经在写了数据库，顺手 Set 即可。
- 用途：积压告警。队列持续增长 = 边端在囤数据但传不上来，
  现场表现为看板数据延迟，往往比"完全断掉"更隐蔽。

**5. ERP 推送队列**
- 指标：`igh_erp_queue_pending`(Gauge)、`igh_erp_queue_failed`(Gauge)、
  `igh_erp_push_total{target,result}`(Counter)
- 位置：`internal/server/service/erp.go` 的 `ProcessQueue()` 与 `RetryFailed()`。
- 注意：代码里已经有一个极危险的场景——"标记失败状态也写入失败"会导致
  **重复推送给 ERP**（`erp.go:180` 有专门日志），目前只有日志、没有计数器，
  无法告警。这是会造成业务事故的，建议优先补。

**6. 数据归档结果**
- 指标：`igh_archive_run_total{table,result}`(Counter)、
  `igh_archive_last_success_timestamp`(Gauge)、`igh_archive_deleted_rows_total{table}`(Counter)
- 位置：`cmd/server/main.go` 的 `runArchiveDaily()`（`Run()` 已返回带 `Err` 的 `Result` 列表）。
- 后果：归档失败只会让磁盘慢慢涨，是"沉默的定时炸弹"。
  过期数据不删还会拖慢查询。

**7. 活跃告警数**
- 指标：`igh_active_alarms`（已定义，只差写入）
- 位置：`internal/server/service/alarm.go`，在告警产生/清除时 Inc/Dec，
  或定时 `COUNT` 未关闭告警。
- 用途：MES 自身的告警（设备故障等）数量是业务健康度的直接体现。

### P2 —— 提升可观测性，不补也能运转

**8. 边端 PLC 连接状态**
- `igh_plc_connected{edge_id,device_code}` 已定义未写入，
  在边端心跳携带的 `PLCStatus` 落地时 Set 即可。补上后能快速定位
  "哪台设备的 PLC 断了"，而不是等操作工来报。

**9. 磁盘空间**
- 用 `node_exporter` 的 `node_filesystem_avail_bytes`，无需自己写指标。
  日志与归档文件都在本地盘，磁盘写满是现场最常见的"服务突然全挂"原因。

**10. HTTP path 标签归一化**
- 不是新指标，而是修 `middleware/metrics.go`：
  `c.FullPath() == ""` 时统一记为 `__unmatched__`，避免标签基数无上限增长。
- 现有 `IghHTTPPathCardinalityHigh` 只能事后发现，不能根治。

---

## 6. 日常检查清单

- [ ] `http://<prom>:9090/targets` 所有目标为 UP
- [ ] `http://<prom>:9090/rules` 所有规则状态正常，无 "No data"（可能是指标缺失）
- [ ] Alertmanager 的 `Silences` 没有长期未清理的静默（静默过期后告警会突然全响）
- [ ] 每周确认 `promtool check config` 仍通过（配置被手工改过是常见事故源）
