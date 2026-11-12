# 旧主库自动回退为备库（P1-5）设计

> 面向：现场运维、后续维护者
> 代码：[`cmd/hactl/rejoin.go`](../../cmd/hactl/rejoin.go)、[`internal/shared/ha/quorum.go`](../../internal/shared/ha/quorum.go)
> 验证：`sh deployments/ha-sim/verify-rejoin.sh`

---

## 1. 要解决的到底是什么问题

`auto_rejoin_as_standby` 原本只是一个配置字段，代码里没有任何实现。看似只是"少个功能"，实际上它背后藏着**两个独立的事故**，而且第一个比第二个更容易漏掉。

### 事故一：旧主库重启后是无围栏的可写主库

正常切换流程是：A 宕机 → B 提升 → 集群恢复可用。

问题出在 A 恢复的时候。A 的数据目录里没有 `standby.signal`，`pg_control` 记录的状态是"主库"。所以 postgres 一启动，A 就认为自己仍然是主库，**可以接受写入**。

它和有租约的 B 构成两个可写主库 —— 这就是脑裂。

现有的 `Decide` 逻辑会在一秒内发现并执行 `ActionFence`，但：

- 这一秒内业务请求可能已经写进去了；
- 如果 hactl 进程根本没起来（比如服务配置错了、进程崩了），A 会**永久**保持可写状态。

### 事故二（实测才发现）：旧主库的写请求会挂死，而不是报错

这一条我原本没有预料到，是靠 `verify-rejoin.sh` 跑出来的。

A 重启后，`postgresql.auto.conf` 里还留着它当主库时的配置：

```conf
synchronous_commit = 'remote_apply'
synchronous_standby_names = '*'
```

它的语义是"等我唯一的同步备库确认后再提交"。但现在唯一的备库已经提升成新主库了，**没有人会来确认**。

于是每一个写操作都卡在 `wait_event = SyncRep` 上，永不返回。

更糟的是：**`statement_timeout` 管不住这个等待**。实测一个 `INSERT` 挂了近 10 分钟，`statement_timeout=3s` 设了也没用。业务侧看到的现象是"大面积请求超时、日志里没有任何错误"——这比直接报错难查得多。

### 顺带发现的第三个坑：杀掉客户端不会回滚服务端的等待

`psql` 被 `timeout` 掐断之后，服务端那个后端仍然挂在 `SyncRep` 上。一旦同步条件满足，**它照样会提交**。

所以"客户端超时/断连"完全不能当作围栏手段。围栏必须由服务端主动 `pg_terminate_backend`。

---

## 2. 三层防线

### 第一层：预置只读（部署期）

两台的 `postgresql.auto.conf` 里预置：

```conf
default_transaction_read_only = on
```

这是**跨重启保留**的。效果是：任何一台机器上的 postgres 无论因为什么原因启动，**在任何进程介入之前就是只读的**。

只有一条路径能清除它：hactl 在 `Decide(Held, Primary) = ActionServe` 时执行 `serve()`。

这一层把"hactl 一秒轮询窗口"压缩到 0，也覆盖了"hactl 根本没跑"的情况。

实现见 [`rejoin.go` 的 `serve()`](../../cmd/hactl/rejoin.go)。

### 第二层：硬围栏（运行期）

`Decide` 判定需要围栏时，`hardFence` 按**固定顺序**做三件事：

```
1. ALTER SYSTEM SET default_transaction_read_only = on  + reload
      → 让新连接一开始就是只读

2. pg_terminate_backend(所有 client backend)
      → 断开已有连接。连接池里的老连接持有可写的会话默认值，不断就继续写。
        同时让卡在 SyncRep 上的事务被 abort 回滚。

3. ALTER SYSTEM SET synchronous_standby_names = ''  + reload
      → 让后续写入快速失败，而不是挂起
```

**这个顺序本身是安全性的一部分。**

如果反过来先清 `synchronous_standby_names`，那些卡在 `SyncRep` 上的 `INSERT` 会**立刻拿到确认并提交**，分叉数据就这样落盘了。必须先断开连接让它们回滚，再解除同步。

原代码的 `fence()` 第一行是 `SET default_transaction_read_only = on` —— 那只作用于 hactl 自己那条连接，对业务连接池毫无影响，是**空操作**。这条代码看起来像做过围栏，实际上什么都没做。

### 第三层：自动回退（事后）

被围栏的旧主库不会自己恢复成健康的备库。`ActionRejoin` 负责把它拉回来：

```
停库 → pg_rewind 回滚到分叉点 → 写 standby.signal + primary_conninfo → 起库 → 追 WAL
```

---

## 3. 回退的准入条件

回退比围栏危险得多（要停库、动数据目录），所以它**不走 `Decide` 的决策表**，而是走独立的 `ShouldRejoin`，逐个显式检查。

| 条件 | 不满足时会发生什么 |
|---|---|
| `auto_rejoin_as_standby = true` | 不启用就直接不做 |
| 本机租约 = `LeaseLost` | 持有租约说明自己是合法主库，不需要回退；租约未知时连自己是主是备都不知道，动数据库是灾难 |
| 本机数据库 = `DBPrimary` | 不是主库就没有"回退"一说 |
| **对端租约 = `LeaseHeld`** | **安全阀**：必须确知对端是被 etcd 授权的当前主库 |
| 对端数据库 = `DBPrimary` | 对端是备库或连不上，就没有可同步的目标 |
| 未超过连续失败上限 | 失败通常是数据目录损坏、权限问题、WAL 已被回收——不会自己好 |
| 已过冷却期 | 防止每秒重试把数据库反复停启 |

### 为什么"对端持租约"这条最关键

设想两台机器都认为自己该回退，就会**互相 rewind**：A 的数据被 B 覆盖，B 的数据又被 A 覆盖。这比脑裂更难恢复——原始数据都不在了。

而"本机 lost + 对端 held"和"本机 held + 对端 lost"在 etcd 的单一 leader key 下**不可能同时成立**。这一条保证了回退是单向的。

测试 [`TestRejoin_NeverMutualRewind`](../../internal/shared/ha/quorum_test.go) 专门验证这一点。

---

## 4. 执行细节与踩过的坑

### 4.1 pg_rewind 需要超级用户

只给 `REPLICATION` 权限的角色会得到：

```
pg_rewind: error: could not fetch remote file "global/pg_control":
ERROR: permission denied for function pg_read_binary_file
```

因为 rewind 要读源库的 `global/pg_control`，需要的权限比流复制高。代码里用的是 `Database.User`（普通账号），**不是** `ReplicationUser`。

### 4.2 wal_log_hints 必须写在 initdb 里

```
pg_rewind: error: target server needs to use either data checksums or "wal_log_hints = on"
```

只在 postgres 启动参数里加 `-c wal_log_hints=on` **不够**——pg_rewind 检查的是 `pg_control` 里记录的 initdb 期选项。必须：

- `initdb` 时带 `wal_log_hints`，或
- `initdb --data-checksums`

生产环境两台服务器初始化时必须确认这一点，否则 rewind 一律失败，回退功能实际不可用。

### 4.3 rewind 前必须 `--dry-run`

`runRewind` 先跑 `--dry-run`，通过后再真跑。原因：真跑时才发现权限/版本问题，数据目录可能已经被动过一部分，回滚更麻烦。

### 4.4 rewind 失败绝不启动 postgres

中途失败时数据目录可能处于半写状态。以主库身份启动等于把脏数据暴露给业务写入口。宁可停机等人工，也不能造成分叉。

代码里 rewind 失败直接返回错误，不执行 `pg_ctl start`。

### 4.5 rewind 后要清掉残留的只读标志

`postgresql.auto.conf` 里 `default_transaction_read_only = on` 是围栏时写进去的。rewind 之后如果不清，备库无法重放 WAL。`writeStandbyConfig` 之后的启动脚本里需要 `sed -i '/default_transaction_read_only/d'`。

---

## 5. WAL 保留：rewind 的前提

rewind 需要从分叉点开始重放 WAL。如果主库已经回收了那段 WAL，rewind 直接失败，只能全量 `pg_basebackup` 重建（几十分钟到几小时）。

当前模拟环境的配置：

```sql
max_slot_wal_keep_size = -1    -- 无上限
pg_replication_slots         -- 空
```

这组合意味着：提升为主库后 WAL 会被回收，旧主库想 rewind 时源 WAL 已经没了。

**建议的部署配置**（尚未实现，需要现场确认磁盘容量）：

```sql
-- 每台机器为对端预留一个物理槽
SELECT pg_create_physical_replication_slot('igh_server_b_rejoin', true);
-- 给槽设上限，防止对端长期不回来撑爆磁盘
ALTER SYSTEM SET max_slot_wal_keep_size = '4GB';
```

代价是需要额外的磁盘空间。**需要现场确认**：两台机器的 PG 数据盘剩余空间。

复制的配置方式还有一个选择：让回退后的备库用 `primary_slot_name` 绑定到预留槽（避免 WAL 被回收）。这需要 `primary_conninfo` 里带 `primary_slot_name=...`。

---

## 6. 验证

```bash
sh deployments/ha-sim/verify-rejoin.sh
```

脚本用独立容器 + 临时卷，跑完自动清理，不影响 `ha-sim` 主环境。9 个阶段全流程复现：

```
──── 准备：A 主库 + B 备库（同步复制）
  [PASS] 同步复制建立（sync_state=sync）
──── 第 1 步：写入旧数据
  [PASS] 旧数据已复制到 B（2 行）
──── 第 2 步：A 宕机
──── 第 3 步：B 提升为新主库
  [PASS] B 已成为主库（可写）
──── 第 4 步：B 在分叉后写入新数据
  [PASS] B 主库写入成功（3 行）
──── 第 5 步：A 重启 —— 危险的时刻
  [PASS] A 以**主库**身份启动 —— 这正是脑裂的起点
  [PASS] A 仍带着 synchronous_standby_names='*'（备库已不在）
  [PASS] A 的写操作卡在同步复制等待上并超时（SyncRep 挂死，非报错）
  [PASS] A 接受了分叉数据（diverged-on-A 已落盘）
──── 第 6 步：硬围栏 A
  [PASS] A 已置为只读
  [PASS] 围栏后 A 写入被拒绝
──── 第 7 步：pg_rewind 把 A 回滚到分叉点
  [PASS] pg_rewind 执行成功
──── 第 8 步：写成备库并启动
  [PASS] A 已回退为备库（pg_is_in_recovery=true）
──── 第 9 步：验证数据一致性
    A: 1:old-1,2:old-2,3:new-on-B
    B: 1:old-1,2:old-2,3:new-on-B
  [PASS] A 与 B 数据完全一致
  [PASS] A 的分叉数据已被 pg_rewind 丢弃
  [PASS] B 的新数据已同步到 A
════ 全部通过：旧主库可安全回退为备库 ════
```

### 单元测试

| 测试 | 覆盖 |
|---|---|
| `TestShouldRejoin`（11 个子用例） | 每个准入条件被放过去的后果 |
| `TestRejoin_NeverMutualRewind` | 两台机器不可能同时满足回退条件 |
| `TestWriteStandbyConfig` | standby.signal + primary_conninfo |
| `TestWriteStandbyConfig_EscapesQuotes` | 口令含单引号时 conf 不被破坏 |
| `TestPgRewindPath` | 路径推导（含 Windows .exe） |
| `TestPeerNode` | 对端选择排除见证节点 |
| `TestPeerDBAddr` | 地址推导与环境变量覆盖 |
| `TestHardFence_BlocksWrites` | **需要 `IGH_TEST_PG_DSN`**：围栏后新连接真的写不进去 |
| `TestReadDBState` | 主/备状态识别 |

集成测试跑法：

```bash
IGH_TEST_PG_DSN="host=127.0.0.1 port=15432 user=igh password=xxx dbname=igh_platform sslmode=disable" \
  go test ./cmd/hactl/ -count=1 -run TestHardFence -v
```

没设环境变量时 skip —— CI 上没有数据库，这是正确行为。

---

## 7. 已知限制

1. **只能处理"旧主库回来"这一种收敛**。两台都掉线、见证节点也挂、etcd 失去多数派时，任何自动决策都会增加风险，此时一律围栏等人工。
2. **rewind 失败后不自动重建**。全量 `pg_basebackup` 会占用大量 IO 和数十分钟，不适合自动触发。达到失败上限后人工决定。
3. **Windows 上 `pg_ctl stop` 与容器 PID 1 的区别**。生产环境 hactl 与 PG 同机，`pg_ctl` 正常使用；模拟环境里 postgres 是容器 PID 1，停库会让容器退出，所以验证脚本用一次性容器挂载数据卷来执行 rewind。
4. **不自动切回原主**。A 回退成备库后，即使它原来是"主库角色"，也不会自动抢回主库位置。切回来意味着又一次停服，收益不抵成本。需要时人工执行。

---

## 8. 上线检查清单

- [ ] 两台服务器 `initdb` 时开了 `wal_log_hints` 或 `--data-checksums`
- [ ] `cluster.yaml` 的 `database.data_dir` / `pg_ctl` / `pg_rewind` 路径正确
- [ ] 数据库账号能读 `global/pg_control`（超级用户或等效权限）
- [ ] 确认 PG 数据盘剩余空间够放 `max_slot_wal_keep_size`
- [ ] 复制槽已创建（或明确接受"rewind 失败需全量重建"）
- [ ] `verify-rejoin.sh` 在测试环境跑通
- [ ] `auto_rejoin_as_standby` 先设 `false` 观察一周，确认围栏日志无异常后再打开
