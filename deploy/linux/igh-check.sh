#!/bin/sh
# keepalived 健康检查：本节点能否承接流量。
#
# 判断两件事，缺一不可：
#   1. igh-server 进程活着（/health 能回）
#   2. 它连的数据库可写、探测健康（/health 的 database.healthy=true）
#
# 为什么第 2 条必须查：数据库被围栏（只读）的节点上，进程本身是正常的，
# 但它接到的每一个写请求都会失败。只看进程存活会把 VIP 留在一个
# 根本写不进数据的节点上。
#
# 返回值：0 = 健康，可以持有 VIP；非 0 = 不健康，让出 VIP。
set -eu

PORT="${IGH_HEALTH_PORT:-8101}"
URL="http://127.0.0.1:${PORT}/health"

body=$(curl -sf --max-time 2 "$URL" 2>/dev/null) || {
    echo "unhealthy: /health unreachable on port ${PORT}"
    exit 1
}

# 提取 database.healthy。用 sed 而不是 jq：现场机器不一定装了 jq，
# 而这份脚本要在最小化的服务器上跑。
db_healthy=$(printf '%s' "$body" | sed -n 's/.*"healthy":\(true\|false\).*/\1/p' | head -1)

if [ "$db_healthy" = "false" ]; then
    echo "unhealthy: database probe failed"
    exit 1
fi

echo "healthy"
exit 0
