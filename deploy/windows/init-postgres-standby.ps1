# ============================================================================
#  IGH 高可用集群 — PostgreSQL 备库初始化（在 server-b 执行一次）
#
#  用法（管理员 PowerShell）：
#    .\init-postgres-standby.ps1
#
#  前置：
#    1. PostgreSQL 16 已安装，但服务未运行或已停止
#    2. 主库已完成 init-postgres-primary.ps1
#    3. 备库的 D:\igh\pgsql\data 为空目录（或已被清空）
# ============================================================================

$ErrorActionPreference = "Stop"

# ---- 现场参数 ----
$PgHome    = "C:\Program Files\PostgreSQL\16"
$PgData    = "D:\igh\pgsql\data"
$PrimaryIP = if ($env:IGH_SERVER_A_IP) { $env:IGH_SERVER_A_IP } else { "192.168.10.1" }
$RepUser   = "replicator"
$Service   = "postgresql-x64-16"

$RepPassword = $env:IGH_REP_PASSWORD
if (-not $RepPassword) { throw "请先设置环境变量 IGH_REP_PASSWORD（与主库一致）" }

Write-Host "=== 1. 停止服务并清空数据目录 ===" -ForegroundColor Cyan
Stop-Service $Service -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

if (Test-Path $PgData) {
    $items = Get-ChildItem -Path $PgData -Force
    if ($items.Count -gt 0) {
        $backup = "$PgData.bak-$(Get-Date -Format yyyyMMddHHmmss)"
        Write-Host "数据目录非空，移动到 $backup"
        Move-Item -Path $PgData -Destination $backup
    }
}
New-Item -ItemType Directory -Path $PgData -Force | Out-Null
# pg_basebackup 要求目录为空且权限仅限服务账户
Write-Host "注意：如 pg_basebackup 报权限错误，请确保数据目录 ACL 允许服务账户写入。"

Write-Host "=== 2. 执行 pg_basebackup（含 -R 自动生成 standby.signal） ===" -ForegroundColor Cyan
$env:PGPASSWORD = $RepPassword
& "$PgHome\bin\pg_basebackup.exe" `
    -h $PrimaryIP -p 5432 -U $RepUser `
    -D $PgData -Fp -Xs -P -R -c fast

if ($LASTEXITCODE -ne 0) { throw "pg_basebackup 失败，退出码 $LASTEXITCODE" }

Write-Host "=== 3. 校验 standby.signal 与 primary_conninfo ===" -ForegroundColor Cyan
if (-not (Test-Path (Join-Path $PgData "standby.signal"))) {
    throw "缺少 standby.signal，备库不会以只读方式启动"
}
$conninfo = Select-String -Path (Join-Path $PgData "postgresql.auto.conf") -Pattern "primary_conninfo"
Write-Host $conninfo.Line

Write-Host "=== 4. 启动备库 ===" -ForegroundColor Cyan
Start-Service $Service
Start-Sleep -Seconds 5

Write-Host "=== 5. 验证复制状态 ===" -ForegroundColor Cyan
$env:PGPASSWORD = $env:IGH_DB_PASSWORD
& "$PgHome\bin\psql.exe" -h 127.0.0.1 -U postgres -c "SELECT pg_is_in_recovery();"
& "$PgHome\bin\psql.exe" -h 127.0.0.1 -U postgres -c "SELECT status, sender_host, sent_lsn, replay_lsn, sync_state FROM pg_stat_wal_receiver;" 2>$null

Write-Host ""
Write-Host "备库初始化完成。" -ForegroundColor Green
Write-Host "下一步：回到主库确认同步复制已生效：" -ForegroundColor Yellow
Write-Host "  psql -c `"SELECT application_name, sync_state, sync_priority FROM pg_stat_replication;`""
Write-Host "  期望 sync_state = 'sync'（若为 'async'，检查 synchronous_standby_names）"
