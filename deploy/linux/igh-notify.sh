#!/bin/sh
# keepalived 角色变更通知：把本机角色写进状态文件。
#
# 它**不启停任何服务**。双机热备采用"两头常驻 + VIP 漂移"：
# 两台服务器上的 igh-server 一直都在跑，谁持有 VIP 谁对外服务。
# 启停进程会在切换瞬间产生几秒空窗，而且备机进程没起来时的排错成本很高。
#
# igh-server 每 2 秒读一次这个文件，通过 /health 的 ha.role 如实上报。
# 这样运维不用登 keepalived 机器就能知道当前谁是 master。
#
# 安装：
#   install -m 755 igh-notify.sh /etc/keepalived/igh-notify.sh
#   mkdir -p /var/lib/igh && chown igh:igh /var/lib/igh
set -eu

STATE_DIR="/var/lib/igh"
STATE_FILE="${STATE_DIR}/keepalived.state"

role="${1:-unknown}"
case "$role" in
    master|backup|fault|stop) ;;
    *) echo "unknown role: $role" >&2; exit 2 ;;
esac

mkdir -p "$STATE_DIR"

# 先写临时文件再 mv：igh-server 可能正好在这个瞬间读，
# 直接覆盖会让它读到空文件，误判成"角色未知"。
tmp="${STATE_FILE}.tmp.$$"
printf '%s\n' "$role" > "$tmp"
mv -f "$tmp" "$STATE_FILE"

# 文件属主是 root（keepalived 以 root 运行），igh-server 以 igh 用户读取，
# 需要放开读权限，否则它读不到、只能一直显示 unknown。
chmod 644 "$STATE_FILE"

echo "keepalived role -> ${role}"
