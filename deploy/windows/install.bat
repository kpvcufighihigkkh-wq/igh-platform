@echo off
setlocal

set SERVICE_NAME=igh-edge
set DISPLAY_NAME=IGH Edge Node
set INSTALL_DIR=%~dp0
set EXE_PATH=%INSTALL_DIR%igh-edge.exe
set CONFIG_DIR=%INSTALL_DIR%

echo ============================================
echo   IGH Edge Node - Windows Installation
echo ============================================
echo.

:: Check admin privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo ERROR: Please run this script as Administrator.
    pause
    exit /b 1
)

:: Create data directory
if not exist "%INSTALL_DIR%data" mkdir "%INSTALL_DIR%data"

:: Check if binary exists
if not exist "%EXE_PATH%" (
    echo ERROR: igh-edge.exe not found in %INSTALL_DIR%
    pause
    exit /b 1
)

:: Check if config exists
if not exist "%CONFIG_DIR%edge.yaml" (
    echo ERROR: edge.yaml not found in %CONFIG_DIR%
    echo Please copy and edit edge.yaml before installing.
    pause
    exit /b 1
)

:: Install as Windows service using sc
echo Installing %SERVICE_NAME% service...
sc create %SERVICE_NAME% binPath= "\"%EXE_PATH%\" --config \"%CONFIG_DIR%\"" DisplayName= "%DISPLAY_NAME%" start= auto
if %errorLevel% neq 0 (
    echo ERROR: Failed to create service. It may already exist.
    echo Run uninstall.bat first if you want to reinstall.
    pause
    exit /b 1
)

:: Set service description
sc description %SERVICE_NAME% "IGH Platform Edge Node - Local workstation for production line operations"

:: Set recovery options (restart on failure)
sc failure %SERVICE_NAME% reset= 86400 actions= restart/5000/restart/10000/restart/30000

:: Start the service
echo Starting %SERVICE_NAME% service...
sc start %SERVICE_NAME%

echo.
echo ============================================
echo   Installation complete!
echo   Service: %SERVICE_NAME%
echo   Config:  %CONFIG_DIR%edge.yaml
echo   Data:    %INSTALL_DIR%data\
echo.
echo   Edit edge.yaml to configure:
echo     - edge.id (unique node identifier)
echo     - edge.type (winding/sorting/packaging/warehouse)
echo     - edge.server.url (central server address)
echo     - edge.plc.devices (PLC connection settings)
echo ============================================
pause
