@echo off
REM IGH Platform Center Server - Windows 服务安装
REM
REM 本机仅安装"中心服务"；数据库（PostgreSQL）与 etcd 需另行部署。
REM 备机上的中心服务同样是常驻运行：切换靠 VIP 漂移，不靠启停进程。
REM
REM 用法（管理员权限）：
REM   install-server.bat
REM   install-server.bat uninstall
setlocal enabledelayedexpansion

set SERVICE_NAME=igh-server
set DISPLAY_NAME=IGH Center Server
set INSTALL_DIR=%~dp0
set CONFIG_DIR=%INSTALL_DIR%configs
set EXE_PATH=%INSTALL_DIR%igh-server.exe
set LOG_DIR=%INSTALL_DIR%logs

if /I "%~1"=="uninstall" goto uninstall

echo ============================================
echo   IGH Center Server - Windows Installation
echo ============================================

if not exist "%EXE_PATH%" (
    echo ERROR: igh-server.exe not found in %INSTALL_DIR%
    exit /b 1
)
if not exist "%CONFIG_DIR%\project.yaml" (
    echo ERROR: project.yaml not found in %CONFIG_DIR%
    echo Please copy and edit configs\project.yaml before installing.
    exit /b 1
)
if not exist "%LOG_DIR%" mkdir "%LOG_DIR%"

REM 数据库口令通过服务环境变量注入，避免写死进配置文件
REM 首次安装后在"服务 -> 属性 -> 环境"或系统环境变量里设置 IGH_DB_PASSWORD
echo Note: set system environment variable IGH_DB_PASSWORD before starting the service.

sc query %SERVICE_NAME% >nul 2>&1
if %errorlevel%==0 (
    echo Service already exists, stopping and removing...
    sc stop %SERVICE_NAME% >nul 2>&1
    timeout /t 3 /nobreak >nul
    sc delete %SERVICE_NAME% >nul 2>&1
)

sc create %SERVICE_NAME% binPath= "\"%EXE_PATH%\" --config \"%CONFIG_DIR%\"" DisplayName= "%DISPLAY_NAME%" start= auto
if errorlevel 1 (
    echo ERROR: failed to create service. Run this script as Administrator.
    exit /b 1
)

REM 进程异常退出后自动重启（第一次 5 秒后，后续 10/30 秒）
sc failure %SERVICE_NAME% reset= 86400 actions= restart/5000/restart/10000/restart/30000
sc description %SERVICE_NAME% "IGH Platform Center Server - API, Web UI, HA coordination"

sc start %SERVICE_NAME%

echo.
echo Installation complete.
echo   Service: %SERVICE_NAME%
echo   Config:  %CONFIG_DIR%\project.yaml
echo   Logs:    %LOG_DIR%
echo   Health:  http://127.0.0.1:8099/health
echo.
echo Remember to set IGH_DB_PASSWORD in system environment variables.
goto :eof

:uninstall
echo Stopping and removing %SERVICE_NAME%...
sc stop %SERVICE_NAME% >nul 2>&1
timeout /t 3 /nobreak >nul
sc delete %SERVICE_NAME% >nul 2>&1
echo Done.
