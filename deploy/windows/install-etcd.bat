@echo off
setlocal enabledelayedexpansion
REM ============================================================================
REM  IGH 高可用集群 — etcd 成员安装（在 3 台机器上分别执行）
REM
REM  用法（管理员 PowerShell 或 cmd）：
REM    install-etcd.bat server-a
REM    install-etcd.bat server-b
REM    install-etcd.bat witness
REM
REM  前置条件：
REM    1. etcd 二进制已解压到 %ETCD_HOME%（默认 C:\igh\etcd）
REM    2. configs\ha\cluster.yaml 中的 IP 与 cluster_token 已改为现场值
REM ============================================================================

set "NODE_ID=%~1"
if "%NODE_ID%"=="" (
    echo 用法: install-etcd.bat ^<server-a^|server-b^|witness^>
    exit /b 1
)

REM ---- 现场参数（必须与 cluster.yaml 保持一致） ----
if "%IGH_SERVER_A_IP%"=="" set "IGH_SERVER_A_IP=192.168.10.1"
if "%IGH_SERVER_B_IP%"=="" set "IGH_SERVER_B_IP=192.168.10.2"
if "%IGH_WITNESS_IP%"==""  set "IGH_WITNESS_IP=192.168.10.31"
if "%IGH_CLUSTER_TOKEN%"=="" set "IGH_CLUSTER_TOKEN=igh-hx-jj-2026"
if "%ETCD_HOME%"=="" set "ETCD_HOME=C:\igh\etcd"
set "ETCD_DATA=%ETCD_HOME%\data"

REM ---- 解析本机 IP ----
if /i "%NODE_ID%"=="server-a" set "THIS_IP=%IGH_SERVER_A_IP%"
if /i "%NODE_ID%"=="server-b" set "THIS_IP=%IGH_SERVER_B_IP%"
if /i "%NODE_ID%"=="witness"  set "THIS_IP=%IGH_WITNESS_IP%"
if "%THIS_IP%"=="" (
    echo 未知节点: %NODE_ID%
    exit /b 1
)

net session >nul 2>&1
if errorlevel 1 (
    echo 错误：请以管理员身份运行。
    exit /b 1
)

if not exist "%ETCD_HOME%\etcd.exe" (
    echo 错误：找不到 %ETCD_HOME%\etcd.exe
    exit /b 1
)
if not exist "%ETCD_DATA%" mkdir "%ETCD_DATA%"

set "CLUSTER=server-a=http://%IGH_SERVER_A_IP%:2380,server-b=http://%IGH_SERVER_B_IP%:2380,witness=http://%IGH_WITNESS_IP%:2380"

set "ARGS=--name %NODE_ID%"
set "ARGS=%ARGS% --data-dir "%ETCD_DATA%""
set "ARGS=%ARGS% --listen-client-urls http://%THIS_IP%:2379 --advertise-client-urls http://%THIS_IP%:2379"
set "ARGS=%ARGS% --listen-peer-urls http://%THIS_IP%:2380 --initial-advertise-peer-urls http://%THIS_IP%:2380"
set "ARGS=%ARGS% --initial-cluster "%CLUSTER%""
set "ARGS=%ARGS% --initial-cluster-state new --initial-cluster-token %IGH_CLUSTER_TOKEN%"
set "ARGS=%ARGS% --logger zap --log-outputs "%ETCD_HOME%\logs\etcd.log""

echo.
echo ============================================
echo   etcd 成员安装
echo   节点   : %NODE_ID%
echo   地址   : %THIS_IP%
echo   集群   : %CLUSTER%
echo ============================================
echo.

REM ---- 服务化（sc 的 binPath 需要短路径；用 etcd.exe 的绝对路径即可） ----
sc query igh-etcd >nul 2>&1
if not errorlevel 1 (
    echo 服务已存在，先停止并删除...
    sc stop igh-etcd >nul 2>&1
    timeout /t 3 /nobreak >nul
    sc delete igh-etcd >nul 2>&1
)

sc create igh-etcd binPath= "\"%ETCD_HOME%\etcd.exe\" %ARGS%" DisplayName= "IGH etcd (%NODE_ID%)" start= auto
if errorlevel 1 (
    echo 错误：创建服务失败。
    exit /b 1
)
sc description igh-etcd "IGH HA cluster etcd member - %NODE_ID%"
sc failure igh-etcd reset= 86400 actions= restart/5000/restart/10000/restart/30000

sc start igh-etcd
timeout /t 3 /nobreak >nul

REM ---- 验证 ----
"%ETCD_HOME%\etcdctl.exe" --endpoints=http://%THIS_IP%:2379 endpoint health
if errorlevel 1 (
    echo.
    echo 警告：etcd 未在 3 秒内健康。请检查 2379/2380 端口与防火墙。
    echo 查看日志：%ETCD_HOME%\logs\etcd.log
)

echo.
echo 完成。下一步在任意一台执行：
echo   etcdctl --endpoints=http://%THIS_IP%:2379 member list
echo 应看到 3 个成员。
pause
