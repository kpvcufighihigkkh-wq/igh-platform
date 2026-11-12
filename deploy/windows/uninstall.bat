@echo off
setlocal

set SERVICE_NAME=igh-edge

echo ============================================
echo   IGH Edge Node - Windows Uninstallation
echo ============================================
echo.

:: Check admin privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo ERROR: Please run this script as Administrator.
    pause
    exit /b 1
)

:: Stop service
echo Stopping %SERVICE_NAME% service...
sc stop %SERVICE_NAME% >nul 2>&1
timeout /t 3 /nobreak >nul

:: Delete service
echo Removing %SERVICE_NAME% service...
sc delete %SERVICE_NAME%

echo.
echo Service removed. Data directory was NOT deleted.
echo Remove it manually if no longer needed.
pause
