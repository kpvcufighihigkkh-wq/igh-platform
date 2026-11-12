#!/usr/bin/env bash
# 双机热备 + 中心自动跟随主库 —— 一键验证脚本。
#
# 在一台开发机上模拟：3 节点 etcd、PG 主备（同步复制）、两个 hactl、
# 一个中心服务（多地址配置）。跑完会打印 PASS/FAIL 清单。
#
# 用法（在 Git Bash 里）：
#   bash deployments/ha-sim/verify.sh
#   bash deployments/ha-sim/verify.sh --keep      # 结束后保留环境，便于手工观察
#
# 它验证什么：
#   1. 3 个 etcd 形成 Raft 多数派
#   2. PG 同步复制生效（remote_apply + sync_state=sync）
#   3. 两个 hactl 只有一个持有租约，另一个等待而不是抢
#   4. 强杀主库后，备库在 10 秒内被提升
#   5. 提升前的数据一行不丢
#   6. 中心服务在 10 秒内自动切到新主库，且写入恢复
#   7. 旧主库重启后被围栏（只读），不能产生双写
#
# 它验证不了：真实断电、网线拔出、时钟漂移、两机网络分区。
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

COMPOSE_FILE="$ROOT/deployments/ha-sim/docker-compose.yml"
CENTER_CFG="$ROOT/configs-ha-sim"
CENTER_PORT=8103
DB_PASSWORD_VAL="igh_dev_2026"

CENTER_LOG="$(mktemp -t igh-center.XXXXXX.log)"
A_LOG="$(mktemp -t igh-hactl-a.XXXXXX.log)"
B_LOG="$(mktemp -t igh-hactl-b.XXXXXX.log)"

KEEP=${1:-}
CENTER_PID=""
A_PID=""
B_PID=""
FAILED=0

say()  { printf '\n\033[36m== %s ==\033[0m\n' "$*"; }
ok()   { printf '  \033[32mPASS\033[0m %s\n' "$*"; }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAILED=$((FAILED+1)); }
info() { printf '       %s\n' "$*"; }

cleanup() {
  [ -n "$CENTER_PID" ] && kill "$CENTER_PID" 2>/dev/null
  [ -n "$A_PID" ] && kill "$A_PID" 2>/dev/null
  [ -n "$B_PID" ] && kill "$B_PID" 2>/dev/null
  if [ "$KEEP" != "--keep" ]; then
    docker compose -f "$COMPOSE_FILE" down -v >/dev/null 2>&1
    echo
    echo "环境已清理（加 --keep 可保留）。日志："
    echo "  中心:   $CENTER_LOG"
    echo "  hactl-a: $A_LOG"
    echo "  hactl-b: $B_LOG"
  else
    echo
    echo "--keep：环境保留在运行状态。手工清理："
    echo "  docker compose -f $COMPOSE_FILE down -v"
  fi
}
trap cleanup EXIT INT TERM

# 等一个条件成立，超时失败。$1=超时秒数 $2=描述，命令用 $3...
wait_until() {
  local timeout=$1 desc=$2; shift 2
  local i=0
  while [ $i -lt $timeout ]; do
    if "$@" >/dev/null 2>&1; then return 0; fi
    sleep 1; i=$((i+1))
  done
  return 1
}

sql_primary() { docker exec igh-ha-pg-primary psql -U igh -d igh_platform -tAc "$1" 2>/dev/null | tr -d '[:space:]'; }
sql_standby() { docker exec igh-ha-pg-standby psql -U igh -d igh_platform -tAc "$1" 2>/dev/null | tr -d '[:space:]'; }
etcd_get()    { docker exec igh-ha-etcd-1 etcdctl --endpoints=http://etcd-1:2379 get "$1" 2>/dev/null; }

# ---------------------------------------------------------------------------
say "0. 预检：清理上次遗留的测试进程"
# ---------------------------------------------------------------------------
if tasklist //FI "IMAGENAME eq hactl.exe" 2>/dev/null | grep -qi hactl.exe; then
  info "发现遗留的 hactl.exe（上一轮测试未退干净），3 秒后清理，Ctrl+C 可中断"
  sleep 3
  taskkill //F //IM hactl.exe >/dev/null 2>&1
fi
if netstat -ano | grep -q ":$CENTER_PORT .*LISTENING"; then
  pid=$(netstat -ano | grep ":$CENTER_PORT .*LISTENING" | awk '{print $5}' | head -1)
  info "端口 $CENTER_PORT 被占用（PID $pid），停止它"
  taskkill //F //PID "$pid" >/dev/null 2>&1
fi

if ! command -v docker >/dev/null; then
  bad "找不到 docker，无法继续"
  exit 1
fi
ok "docker 可用"

# ---------------------------------------------------------------------------
say "1. 启动拓扑：3 etcd + PG 主备"
# ---------------------------------------------------------------------------
docker compose -f "$COMPOSE_FILE" down -v >/dev/null 2>&1
docker compose -f "$COMPOSE_FILE" up -d >/dev/null 2>&1

if wait_until 60 "pg-primary healthy" bash -c \
  'docker exec igh-ha-pg-primary pg_isready -U igh -d igh_platform'; then
  ok "PG 主库就绪"
else
  bad "PG 主库 60 秒内未就绪"; exit 1
fi

if wait_until 90 "standby streaming" bash -c \
  '[ "$(docker exec igh-ha-pg-primary psql -U igh -d igh_platform -tAc "SELECT count(*) FROM pg_stat_replication WHERE state=(E'"'"'streaming'"'"');" 2>/dev/null | tr -d "[:space:]")" = "1" ]'; then
  ok "备库已建立流复制"
else
  bad "备库 90 秒内未建立流复制"; exit 1
fi

# ---------------------------------------------------------------------------
say "2. 开启同步复制并校验"
# ---------------------------------------------------------------------------
docker exec igh-ha-pg-primary psql -U igh -d postgres \
  -c "ALTER SYSTEM SET synchronous_commit='remote_apply';" \
  -c "ALTER SYSTEM SET synchronous_standby_names='*';" \
  -c "SELECT pg_reload_conf();" >/dev/null 2>&1
sleep 2

sync_commit=$(sql_primary "SHOW synchronous_commit;")
sync_state=$(sql_primary "SELECT sync_state FROM pg_stat_replication WHERE state='streaming';")
[ "$sync_commit" = "remote_apply" ] && ok "synchronous_commit = remote_apply" || bad "synchronous_commit = $sync_commit（期望 remote_apply）"
[ "$sync_state" = "sync" ] && ok "备库 sync_state = sync" || bad "备库 sync_state = $sync_state（期望 sync）"

etcd_health=$(docker exec igh-ha-etcd-1 etcdctl \
  --endpoints=http://etcd-1:2379,http://etcd-2:2379,http://etcd-3:2379 \
  endpoint health 2>/dev/null | grep -c "is healthy")
[ "$etcd_health" = "3" ] && ok "etcd 三节点全部健康" || bad "etcd 健康节点数 = $etcd_health（期望 3）"

# ---------------------------------------------------------------------------
say "3. 启动两个 hactl，验证选主唯一性"
# ---------------------------------------------------------------------------
IGH_DB_PASSWORD="$DB_PASSWORD_VAL" IGH_DB_PORT=15432 \
  ./bin/hactl.exe -config deployments/ha-sim/cluster-sim.yaml \
  -node server-a -run -interval 1s >"$A_LOG" 2>&1 &
A_PID=$!

IGH_DB_PASSWORD="$DB_PASSWORD_VAL" IGH_DB_PORT=15433 \
  ./bin/hactl.exe -config deployments/ha-sim/cluster-sim.yaml \
  -node server-b -run -interval 1s >"$B_LOG" 2>&1 &
B_PID=$!

sleep 8

# 注意不能用 `grep -c ... || echo 0`：grep -c 无匹配时返回退出码 1，
# 会把 "0" 和 echo 的 "0" 一起输出成 "0\n0"，导致后面 -gt 比较报错。
a_held=$(grep -c "lease=held" "$A_LOG" 2>/dev/null)
b_held=$(grep -c "lease=held" "$B_LOG" 2>/dev/null)
a_held=${a_held:-0}
b_held=${b_held:-0}
if [ "$a_held" -gt 0 ] && [ "$b_held" -eq 0 ]; then
  ok "server-a 持有租约，server-b 等待（无脑裂）"
elif [ "$b_held" -gt 0 ] && [ "$a_held" -eq 0 ]; then
  ok "server-b 持有租约，server-a 等待（无脑裂）"
else
  bad "两个节点同时 held（a=$a_held b=$b_held），存在脑裂"
fi

if grep -q "standby-caught-up" "$B_LOG" 2>/dev/null || grep -q "standby-caught-up" "$A_LOG" 2>/dev/null; then
  ok "备库被正确识别为 standby-caught-up"
else
  bad "没有任何节点识别到 standby-caught-up"
fi

# ---------------------------------------------------------------------------
say "4. 启动中心服务（多地址配置）"
# ---------------------------------------------------------------------------
DB_PASSWORD="$DB_PASSWORD_VAL" ./bin/igh-server.exe \
  --config "$CENTER_CFG" --log-level info >"$CENTER_LOG" 2>&1 &
CENTER_PID=$!

if wait_until 30 "center health" bash -c \
  "curl -sf -m 2 http://127.0.0.1:$CENTER_PORT/health"; then
  ok "中心服务已启动"
else
  bad "中心服务 30 秒内未就绪"; exit 1
fi

before_primary=$(curl -s -m 3 "http://127.0.0.1:$CENTER_PORT/health" | sed -n 's/.*"primary":"\([^"]*\)".*/\1/p')
info "中心当前连接：$before_primary"

# 写入一批已知数据
docker exec igh-ha-pg-primary psql -U igh -d igh_platform \
  -c "DROP TABLE IF EXISTS ha_probe;" \
  -c "CREATE TABLE ha_probe(seq int primary key, payload text);" \
  -c "INSERT INTO ha_probe(seq,payload) SELECT g,'row-'||g FROM generate_series(1,100) g;" >/dev/null 2>&1
written=$(sql_primary "SELECT count(*) FROM ha_probe;")
[ "$written" = "100" ] && ok "主库写入 100 行（同步提交）" || bad "写入行数 = $written（期望 100）"

# ---------------------------------------------------------------------------
say "5. 强杀主库，测量切换"
# ---------------------------------------------------------------------------
kill_ms=$(date +%s%3N)
docker kill igh-ha-pg-primary >/dev/null 2>&1
info "主库已强杀（模拟断电）"

promoted=""
i=0
while [ $i -lt 20 ]; do
  state=$(sql_standby "SELECT pg_is_in_recovery();")
  if [ "$state" = "f" ]; then promoted=$(( $(date +%s%3N) - kill_ms )); break; fi
  sleep 1; i=$((i+1))
done
if [ -n "$promoted" ]; then
  info "备库提升耗时 ${promoted}ms"
  [ "$promoted" -le 10000 ] && ok "提升在 10 秒内完成" || bad "提升耗时 ${promoted}ms，超过 10 秒"
else
  bad "20 秒内备库未被提升"; exit 1
fi

# ---------------------------------------------------------------------------
say "6. 零数据丢失校验"
# ---------------------------------------------------------------------------
rows=$(sql_standby "SELECT count(*) FROM ha_probe;")
gaps=$(sql_standby "SELECT count(*) FROM generate_series(1,100) g WHERE NOT EXISTS (SELECT 1 FROM ha_probe WHERE seq=g);")
[ "$rows" = "100" ] && ok "新主库有 100 行，一行没丢" || bad "新主库行数 = $rows（期望 100，丢了 $((100-rows)) 行）"
[ "$gaps" = "0" ] && ok "无空洞（1..100 连续）" || bad "有 $gaps 个空洞"

# ---------------------------------------------------------------------------
say "7. 中心自动跟随新主库"
# ---------------------------------------------------------------------------
follow_ms=""
i=0
while [ $i -lt 30 ]; do
  now_primary=$(curl -s -m 2 "http://127.0.0.1:$CENTER_PORT/health" 2>/dev/null | sed -n 's/.*"primary":"\([^"]*\)".*/\1/p')
  if [ -n "$now_primary" ] && [ "$now_primary" != "$before_primary" ]; then
    follow_ms=$(( $(date +%s%3N) - kill_ms )); break
  fi
  sleep 1; i=$((i+1))
done

if [ -n "$follow_ms" ]; then
  info "中心跟随耗时 ${follow_ms}ms（含等待备库提升）"
  [ "$follow_ms" -le 15000 ] && ok "中心已自动切到新主库：$now_primary" || bad "中心跟随耗时 ${follow_ms}ms，偏慢"
else
  bad "30 秒内中心没有跟随新主库"
fi

gen=$(curl -s -m 2 "http://127.0.0.1:$CENTER_PORT/health" | sed -n 's/.*"generation":\([0-9]*\).*/\1/p')
[ "${gen:-0}" -ge 1 ] && ok "generation 已递增到 $gen" || bad "generation 未递增（当前 ${gen:-0}）"

write_code=$(curl -s -o /dev/null -w "%{http_code}" -m 8 -X POST \
  "http://127.0.0.1:$CENTER_PORT/api/v1/edge/register" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: sk-igh-dev-key-change-in-production" \
  -d '{"edge_id":"ha-verify-probe","type":"winding","version":"verify"}')
[ "$write_code" = "200" ] && ok "切换后写接口返回 200" || bad "切换后写接口返回 $write_code"

landed=$(sql_standby "SELECT count(*) FROM edge_nodes WHERE code='ha-verify-probe';")
[ "$landed" = "1" ] && ok "写入落在新主库" || bad "写入未落在新主库（count=$landed）"

# ---------------------------------------------------------------------------
say "8. 旧主库重启后必须被围栏（防脑裂）"
# ---------------------------------------------------------------------------
docker start igh-ha-pg-primary >/dev/null 2>&1
wait_until 30 "old primary up" bash -c \
  'docker exec igh-ha-pg-primary pg_isready -U igh -d igh_platform' >/dev/null 2>&1
sleep 5

ro=$(sql_primary "SHOW default_transaction_read_only;")
[ "$ro" = "on" ] && ok "旧主库已被围栏为只读" || bad "旧主库仍可写（default_transaction_read_only=$ro），存在双写风险"

old_has_probe=$(sql_primary "SELECT count(*) FROM edge_nodes WHERE code='ha-verify-probe';")
[ "$old_has_probe" = "0" ] && ok "旧主库上没有新写入的数据（确认写入只去了新主库）" \
  || info "旧主库上也有该记录（count=$old_has_probe），检查是否发生过双写"

# ---------------------------------------------------------------------------
say "结果"
# ---------------------------------------------------------------------------
if [ "$FAILED" -eq 0 ]; then
  printf '  \033[32m全部通过\033[0m\n'
else
  printf '  \033[31m%d 项失败\033[0m\n' "$FAILED"
  exit 1
fi
