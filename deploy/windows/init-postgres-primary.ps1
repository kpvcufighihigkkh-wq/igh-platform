# ============================================================================
#  IGH 高可用集群 — PostgreSQL 主库初始化（在 server-a 执行一次）
#
#  用法（管理员 PowerShell）：
#    .\init-postgres-primary.ps1
#
#  前置：PostgreSQL 16 已安装，服务名为 postgresql-x64-16
# ============================================================================

$ErrorActionPreference = "Stop"

# ---- 现场参数 ----
$PgHome   = "C:\Program Files\PostgreSQL\16"
$PgData   = "D:\igh\pgsql\data"
$DbName   = "igh_platform"
$DbUser   = "igh"
$RepUser  = "replicator"
$StandbyIP = if ($env:IGH_SERVER_B_IP) { $env:IGH_SERVER_B_IP } else { "192.168.10.2" }
$Service  = "postgresql-x64-16"

$DbPassword  = $env:IGH_DB_PASSWORD
$RepPassword = $env:IGH_REP_PASSWORD
if (-not $DbPassword)  { throw "请先设置环境变量 IGH_DB_PASSWORD" }
if (-not $RepPassword) { throw "请先设置环境变量 IGH_REP_PASSWORD" }

Write-Host "=== 1. 备份并修改 postgresql.conf ===" -ForegroundColor Cyan
$conf = Join-Path $PgData "postgresql.conf"
if (-not (Test-Path $conf)) { throw "找不到 $conf，请确认数据目录" }
Copy-Item $conf "$conf.bak-$(Get-Date -Format yyyyMMddHHmmss)"

$settings = @"
# ---- IGH HA (added $(Get-Date)) ----
listen_addresses = '*'
wal_level = replica
max_wal_senders = 10
max_replication_slots = 10
wal_keep_size = 1GB
hot_standby = on
wal_log_hints = on
synchronous_commit = remote_apply
synchronous_standby_names = '*'
full_page_writes = on
fsync = on
"@

Add-Content -Path $conf -Value $settings -Encoding UTF8

Write-Host "=== 2. 配置 pg_hba.conf（允许备库复制） ===" -ForegroundColor Cyan
$hba = Join-Path $PgData "pg_hba.conf"
Copy-Item $hba "$hba.bak-$(Get-Date -Format yyyyMMddHHmmss)"
Add-Content -Path $hba -Value @"

# ---- IGH HA (added $(Get-Date)) ----
host    replication     $RepUser        $StandbyIP/32           scram-sha-256
host    all             $DbUser         192.168.10.0/24         scram-sha-256
"@ -Encoding UTF8

Write-Host "=== 3. 重启 PostgreSQL ===" -ForegroundColor Cyan
Restart-Service $Service
Start-Sleep -Seconds 3

Write-Host "=== 4. 创建应用库、用户、复制用户 ===" -ForegroundColor Cyan
$env:PGPASSWORD = $DbPassword
& "$PgHome\bin\psql.exe" -U postgres -c "CREATE USER $DbUser WITH PASSWORD '$DbPassword';" 2>$null
& "$PgHome\bin\psql.exe" -U postgres -c "CREATE DATABASE $DbName OWNER $DbUser;" 2>$null
& "$PgHome\bin\psql.exe" -U postgres -c "CREATE USER $RepUser WITH REPLICATION LOGIN PASSWORD '$RepPassword';"

Write-Host "=== 5. 验证同步复制配置 ===" -ForegroundColor Cyan
& "$PgHome\bin\psql.exe" -U postgres -c "SHOW synchronous_commit; SHOW synchronous_standby_names; SHOW wal_level;"

Write-Host ""
Write-Host "主库初始化完成。" -ForegroundColor Green
Write-Host "下一步：在备库($StandbyIP)执行 init-postgres-standby.ps1" -ForegroundColor Yellow
Write-Host "注意：此时 synchronous_standby_names='*' 要求至少一个备库在线，否则写入会被阻塞。"
Write-Host "      首次建备库前，可临时注释该行，或先以 standalone 方式启动。"
