#!/bin/sh
# 备库启动脚本：数据目录为空时先做基础备份，否则直接启动。
#
# 用 shell 而不是更"正规"的 pg_basebackup 一次性 job，是因为备库需要
# 在每次容器重启后仍然以备库身份启动（standby.signal 由 -R 生成，
# 只要数据目录还在就会保留）。做成 job 则每次重启都要手工再跑一次。
set -e

PGDATA=/var/lib/postgresql/data

if [ ! -s "$PGDATA/PG_VERSION" ]; then
    echo "standby: 等待主库就绪..."
    until PGPASSWORD=replpass_2026 pg_isready -h pg-primary -p 5432 -U replicator; do
        sleep 1
    done

    echo "standby: 从主库拉取基础备份..."
    rm -rf "${PGDATA:?}"/*

    # -R 生成 standby.signal 与 primary_conninfo，容器重启后仍是备库
    PGPASSWORD=replpass_2026 pg_basebackup \
        -h pg-primary -p 5432 -U replicator \
        -D "$PGDATA" -Fp -Xs -P -R

    chmod 700 "$PGDATA"
    echo "standby: 基础备份完成"
fi

# 交给官方镜像的入口，它会处理权限与 postgres 启动
exec docker-entrypoint.sh postgres \
    -c hot_standby=on \
    -c synchronous_commit=remote_apply
