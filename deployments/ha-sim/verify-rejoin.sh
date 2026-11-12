#!/bin/sh
# 验证"旧主库自动回退为备库"的完整链路（P1-5）。
#
# 复现的正是最危险的场景：
#
#   1. A 是主库、B 是备库，同步复制
#   2. A 宕机，B 提升为新主库
#   3. A 重启 —— 数据目录还停在"自己是主库"的状态
#   4. 如果不管，A 就是一个**可写的第二主库**（脑裂）
#   5. 正确做法：硬围栏 → pg_rewind 回滚到分叉点 → 写成备库 → 追平
#
# 用独立容器和临时卷，跑完自动清理，不影响 ha-sim 主环境。
#
# 用法：
#   sh deployments/ha-sim/verify-rejoin.sh
#
# 全部命令以 postgres 用户执行：postgres 官方镜像允许 root 启动容器，
# 但 pg_ctl / pg_rewind / pg_basebackup 都拒绝以 root 运行。
set -e

# Git Bash（MSYS2）会把 /var/lib/... 这类参数改写成 Windows 路径，
# 导致容器里收到 "C:/Program Files/Git/var/lib/postgresql/data"。
# 关掉这个转换，参数才能原样传给 docker。
export MSYS_NO_PATHCONV=1

NET=igh-rejoin-net
A=igh-rejoin-a
B=igh-rejoin-b
PA=55432
PB=55433
PW=rejoinpass_2026

FAILED=0
pass() { echo "  [PASS] $1"; }
fail() { echo "  [FAIL] $1"; FAILED=1; }
step() { echo; echo "──── $1"; }

cleanup() {
    docker rm -f "$A" "$B" >/dev/null 2>&1 || true
    docker volume rm igh-rejoin-va igh-rejoin-vb >/dev/null 2>&1 || true
    docker network rm "$NET" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

# -A 是 unaligned，方便在 shell 里比较
psqlA() { docker exec -i "$A" psql -U igh -d igh_platform -t -A "$@"; }
psqlB() { docker exec -i "$B" psql -U igh -d igh_platform -t -A "$@"; }
# postgres 用户执行数据目录相关命令
asPG_A() { docker exec -u postgres "$A" sh -c "$1"; }
asPG_B() { docker exec -u postgres "$B" sh -c "$1"; }

wait_ready() {
    c=$1
    for _ in $(seq 1 60); do
        docker exec "$c" pg_isready -U igh -d igh_platform >/dev/null 2>&1 && return 0
        sleep 1
    done
    return 1
}

cleanup
FAILED=0

step "准备：A 主库 + B 备库（同步复制）"
docker network create "$NET" >/dev/null
docker volume create igh-rejoin-va >/dev/null
docker volume create igh-rejoin-vb >/dev/null

docker run -d --name "$A" --network "$NET" --network-alias pg-a \
    -e POSTGRES_DB=igh_platform -e POSTGRES_USER=igh -e POSTGRES_PASSWORD=$PW \
    -e POSTGRES_INITDB_ARGS="-c wal_log_hints=on" \
    -p $PA:5432 -v igh-rejoin-va:/var/lib/postgresql/data \
    postgres:16-alpine \
    -c wal_level=replica -c wal_log_hints=on \
    -c max_wal_senders=10 -c max_replication_slots=10 -c hot_standby=on >/dev/null

wait_ready "$A" || { echo "A 未就绪"; exit 1; }

psqlA -c "CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD '$PW';" >/dev/null
docker exec "$A" sh -c 'printf "host replication replicator all scram-sha-256\nhost all igh all scram-sha-256\n" >> "$PGDATA/pg_hba.conf"'
docker exec -u postgres "$A" sh -c 'pg_ctl reload -D "$PGDATA"' >/dev/null

# B：先跑一个空壳容器，用它做基础备份（-R 生成 standby.signal 与 primary_conninfo）
docker run -d --name "$B" --network "$NET" --network-alias pg-b \
    -e PGPASSWORD=$PW -v igh-rejoin-vb:/var/lib/postgresql/data \
    postgres:16-alpine sleep 600 >/dev/null
sleep 2
asPG_B "rm -rf /var/lib/postgresql/data/* && pg_basebackup -h pg-a -p 5432 -U replicator -D /var/lib/postgresql/data -Fp -Xs -R" >/dev/null
docker rm -f "$B" >/dev/null

# 以真正的服务方式启动 B
docker run -d --name "$B" --network "$NET" --network-alias pg-b \
    -e POSTGRES_PASSWORD=$PW \
    -p $PB:5432 -v igh-rejoin-vb:/var/lib/postgresql/data \
    postgres:16-alpine -c hot_standby=on -c synchronous_commit=remote_apply >/dev/null

echo "  等待 B 进入恢复模式..."
for _ in $(seq 1 60); do
    [ "$(psqlB -c 'SELECT pg_is_in_recovery()')" = "t" ] && break
    sleep 1
done

psqlA -c "ALTER SYSTEM SET synchronous_commit='remote_apply'" >/dev/null
psqlA -c "ALTER SYSTEM SET synchronous_standby_names='*'" >/dev/null
psqlA -c "SELECT pg_reload_conf()" >/dev/null

SYNC=""
for _ in $(seq 1 30); do
    SYNC=$(psqlA -c "SELECT sync_state FROM pg_stat_replication" 2>/dev/null || true)
    [ "$SYNC" = "sync" ] && break
    sleep 1
done
[ "$SYNC" = "sync" ] && pass "同步复制建立（sync_state=sync）" || fail "同步复制未建立（$SYNC）"

step "第 1 步：写入旧数据"
psqlA -c "CREATE TABLE t (id int primary key, v text); INSERT INTO t VALUES (1,'old-1'),(2,'old-2');" >/dev/null
sleep 2
CNT_B=$(psqlB -c "SELECT count(*) FROM t")
[ "$CNT_B" = "2" ] && pass "旧数据已复制到 B（2 行）" || fail "B 未收到旧数据（$CNT_B）"

step "第 2 步：A 宕机"
docker stop "$A" >/dev/null
pass "A 已停止"

step "第 3 步：B 提升为新主库"
docker exec "$B" psql -U igh -d igh_platform -q -c "SELECT pg_promote(true, 60);" >/dev/null
sleep 3
IN_REC=$(psqlB -c "SELECT pg_is_in_recovery()")
[ "$IN_REC" = "f" ] && pass "B 已成为主库（可写）" || fail "B 提升失败"

step "第 4 步：B 在分叉后写入新数据"
psqlB -c "INSERT INTO t VALUES (3,'new-on-B');" >/dev/null
NEW_CNT=$(psqlB -c "SELECT count(*) FROM t")
[ "$NEW_CNT" = "3" ] && pass "B 主库写入成功（3 行）" || fail "B 写入失败（$NEW_CNT）"

step "第 5 步：A 重启 —— 危险的时刻"
docker start "$A" >/dev/null
wait_ready "$A" || { echo "A 未就绪"; exit 1; }

IN_REC_A=$(psqlA -c "SELECT pg_is_in_recovery()")
[ "$IN_REC_A" = "f" ] && pass "A 以**主库**身份启动 —— 这正是脑裂的起点" || fail "A 未以主库启动"

# 坏事一：旧的 synchronous_standby_names='*' 还留着，但备库已经走了。
#         写操作会卡在 SyncRep 上无限期等待 —— 注意 statement_timeout 管不住它
#         （实测挂了近 10 分钟，设 3s 也没用），所以用容器侧 timeout 硬掐。
SYNC_NAMES=$(psqlA -c "SHOW synchronous_standby_names")
[ "$SYNC_NAMES" = "*" ] && pass "A 仍带着 synchronous_standby_names='*'（备库已不在）" \
    || echo "  [info] A 的 synchronous_standby_names=$SYNC_NAMES"

if docker exec "$A" sh -c "timeout 5 psql -U igh -d igh_platform -c \"INSERT INTO t VALUES (4,'hung-on-A')\"" >/dev/null 2>&1; then
    fail "写操作居然直接成功了（预期应当挂住）"
else
    pass "A 的写操作卡在同步复制等待上并超时（SyncRep 挂死，非报错）"
fi

# 这里验证一个反直觉的事实：**杀掉客户端并不会回滚服务端的同步提交等待**。
# psql 被 timeout 掐断之后，那个后端仍挂在 SyncRep 上；一旦同步条件满足，
# 它照样会提交。这正是"围栏必须由服务端主动终止后端"的理由。
sleep 2
HUNG_ROWS=$(psqlA -c "SELECT count(*) FROM t WHERE id=4")
if [ "$HUNG_ROWS" = "1" ]; then
    echo "  [info] 客户端断开后，挂起的 INSERT 仍在服务端等待（未回滚）"
else
    echo "  [info] 客户端断开后，挂起的 INSERT 已被回滚"
fi

# 清掉挂住的连接（围栏的真实动作）
docker exec "$A" psql -U igh -d igh_platform -q -c "
    SELECT pg_terminate_backend(pid) FROM pg_stat_activity
    WHERE pid <> pg_backend_pid() AND backend_type = 'client backend';" >/dev/null 2>&1 || true
sleep 1

# 坏事二：解除同步复制后 A 就能写进分叉数据。
# 用 id=555 避免与上面那个挂起的事务撞主键。
# 这模拟"同步降级"或"多备库只确认了其中一个"的场景——正是 pg_rewind 要处理的。
psqlA -c "ALTER SYSTEM SET synchronous_standby_names = ''" >/dev/null
psqlA -c "SELECT pg_reload_conf()" >/dev/null
psqlA -c "INSERT INTO t VALUES (555,'diverged-on-A')" >/dev/null
A_CNT=$(psqlA -c "SELECT count(*) FROM t WHERE id=555")
[ "$A_CNT" = "1" ] && pass "A 接受了分叉数据（diverged-on-A 已落盘）" || fail "A 写入异常（$A_CNT）"

step "第 6 步：硬围栏 A（对齐 hactl hardFence 的顺序）"
# 顺序与 hardFence 完全一致：置只读 → 断连 → 清同步复制。
# 反过来（先清同步）会放行那些挂住的 INSERT，等于亲手造分叉数据。
psqlA -c "ALTER SYSTEM SET default_transaction_read_only = on;" >/dev/null
psqlA -c "SELECT pg_reload_conf();" >/dev/null
docker exec "$A" psql -U igh -d igh_platform -q -c "
    SELECT pg_terminate_backend(pid) FROM pg_stat_activity
    WHERE pid <> pg_backend_pid() AND backend_type = 'client backend';" >/dev/null 2>&1 || true

RO=$(docker exec "$A" psql -U igh -d igh_platform -t -A -c "SHOW default_transaction_read_only")
[ "$RO" = "on" ] && pass "A 已置为只读" || fail "只读未生效（$RO）"

if docker exec "$A" psql -U igh -d igh_platform -c "INSERT INTO t VALUES (5,'after-fence');" >/dev/null 2>&1; then
    fail "围栏后 A 仍能写入"
else
    pass "围栏后 A 写入被拒绝"
fi

step "第 7 步：pg_rewind 把 A 回滚到分叉点"
# A 容器的 PID 1 就是 postgres，pg_ctl stop 会让容器直接退出。
# 所以 rewind 不能"停库后再 exec 进同一个容器"——要用一个一次性容器
# 挂同一份数据卷。生产环境 hactl 与 PG 同机，不存在这个问题；
# 这里纯粹是模拟环境的形态差异。
docker rm -f "$A" >/dev/null 2>&1 || true

REWIND_SRC="host=pg-b port=5432 user=igh password=$PW dbname=igh_platform"
if docker run --rm -u postgres --network "$NET" \
        -v igh-rejoin-va:/var/lib/postgresql/data \
        -e PGPASSWORD=$PW \
        postgres:16-alpine \
        pg_rewind --target-pgdata=/var/lib/postgresql/data \
                 --source-server="$REWIND_SRC" --progress >/tmp/rewind.log 2>&1; then
    pass "pg_rewind 执行成功"
    head -3 /tmp/rewind.log | sed 's/^/    /'
else
    fail "pg_rewind 失败，日志："
    tail -15 /tmp/rewind.log | sed 's/^/    /'
    # rewind 没成功就不能继续：以备库身份启动一个数据不一致的目录，
    # 会得到"看起来正常、实际复制不动"的假象，比直接失败更难查。
    echo
    echo "════ pg_rewind 失败，中止后续步骤 ════"
    exit 1
fi

step "第 8 步：写成备库并启动"
# 先把 standby.signal 与 primary_conninfo 写进去（同样用一次性容器），
# 再以服务方式启动 A —— 这样 postgres 一启动就是备库身份。
docker run --rm -u postgres --network "$NET" \
    -v igh-rejoin-va:/var/lib/postgresql/data \
    postgres:16-alpine sh -c "
        touch /var/lib/postgresql/data/standby.signal &&
        printf \"\\nprimary_conninfo = '%s'\\n\" '$REWIND_SRC' >> /var/lib/postgresql/data/postgresql.auto.conf &&
        # rewind 后残留的只读标志必须清掉，否则备库无法重放 WAL
        sed -i '/default_transaction_read_only/d;/synchronous_standby_names/d' /var/lib/postgresql/data/postgresql.auto.conf
    " >/dev/null 2>&1

docker run -d --name "$A" --network "$NET" --network-alias pg-a \
    -e POSTGRES_PASSWORD=$PW \
    -p $PA:5432 -v igh-rejoin-va:/var/lib/postgresql/data \
    postgres:16-alpine -c hot_standby=on >/dev/null

echo "  等待 A 进入恢复模式..."
for _ in $(seq 1 60); do
    [ "$(psqlA -c 'SELECT pg_is_in_recovery()' 2>/dev/null)" = "t" ] && break
    sleep 1
done

IN_REC_A=$(psqlA -c "SELECT pg_is_in_recovery()")
[ "$IN_REC_A" = "t" ] && pass "A 已回退为备库（pg_is_in_recovery=true）" || fail "A 未进入恢复模式"

step "第 9 步：验证数据一致性"
sleep 3
A_ROWS=$(psqlA -c "SELECT string_agg(id || ':' || v, ',' ORDER BY id) FROM t")
B_ROWS=$(psqlB -c "SELECT string_agg(id || ':' || v, ',' ORDER BY id) FROM t")
echo "    A: $A_ROWS"
echo "    B: $B_ROWS"
[ "$A_ROWS" = "$B_ROWS" ] && pass "A 与 B 数据完全一致" || fail "A 与 B 数据不一致"

case "$A_ROWS" in
    *diverged-on-A*) fail "A 的分叉数据未被回滚（rewind 无效）" ;;
    *) pass "A 的分叉数据已被 pg_rewind 丢弃" ;;
esac

case "$A_ROWS" in
    *new-on-B*) pass "B 的新数据已同步到 A" ;;
    *) fail "B 的新数据未同步到 A" ;;
esac

echo
if [ "$FAILED" = "0" ]; then
    echo "════ 全部通过：旧主库可安全回退为备库 ════"
    exit 0
else
    echo "════ 存在失败项 ════"
    exit 1
fi
