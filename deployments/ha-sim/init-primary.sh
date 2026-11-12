#!/bin/sh
# 主库首次初始化时执行（docker-entrypoint-initdb.d）。
#
# 只做两件事：
#   1. 建复制账号
#   2. 放开复制连接（pg_hba）
# 不改 synchronous_* 参数——它们由 compose 的 command 传给 postgres，
# 写在这里反而会被下一次启动覆盖掉，容易产生"配置看起来很对但实际没生效"。
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD 'replpass_2026';
EOSQL

# 容器网络的 172.16.0.0/12 足够覆盖 docker bridge。
# 生产环境应改成具体的备库地址，不要用网段。
echo "host replication replicator 0.0.0.0/0 scram-sha-256" >> "$PGDATA/pg_hba.conf"
echo "host all igh 0.0.0.0/0 scram-sha-256" >> "$PGDATA/pg_hba.conf"

echo "replication role created"
