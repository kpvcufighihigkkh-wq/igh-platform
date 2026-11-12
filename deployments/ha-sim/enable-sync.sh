#!/bin/sh
# 备库开始流复制之后，把主库切到"同步复制 + 零丢失"。
#
# 为什么不在 compose 的 command 里直接设：
# 官方镜像初始化阶段会起一个只监听 localhost 的临时服务执行 CREATE DATABASE，
# 那时还没有任何备库。synchronous_standby_names='*' 会让每个提交无限期等待，
# 初始化直接卡死（第一次搭建时就踩了这个坑，库都建不出来）。
#
# 所以顺序是：主库先异步起来 → 备库做完基础备份开始追 → 再执行本脚本。
#
# 用法：
#   docker exec igh-ha-pg-primary /enable-sync.sh
set -e

psql -U igh -d postgres <<-EOSQL
    ALTER SYSTEM SET synchronous_commit = 'remote_apply';
    ALTER SYSTEM SET synchronous_standby_names = '*';
    SELECT pg_reload_conf();
EOSQL

echo "--- 同步复制已开启，当前状态： ---"
psql -U igh -d postgres -c "SHOW synchronous_commit;"
psql -U igh -d postgres -c "SHOW synchronous_standby_names;"
psql -U igh -d postgres -c "SELECT client_addr, state, sync_state FROM pg_stat_replication;"
