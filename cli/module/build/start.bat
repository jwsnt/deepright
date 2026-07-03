@echo off

echo ============================================
echo    Deepright - Start Integration
echo ============================================
echo.

:: Check if deepright distro is reachable
wsl -d deepright -- echo "ok" 2>nul >nul
if %errorlevel% neq 0 (
    echo [X] deepright distro not found or not running.
    echo.
    echo Please run install.bat first to set up the environment.
    echo ============================================
    pause
    exit /b 1
)

echo [i] Starting integration in deepright...
echo.

:: Run synchronously so browser has time to open.
:: The wrapper script sets TERM and uses setsid (required for browser auto-open).
wsl -d deepright -- /home/deepright/start-deepright.sh
if %errorlevel% neq 0 (
    echo [X] Integration failed to start.
    echo ============================================
    pause
    exit /b 1
)

timeout /t 3 >nul

echo [OK] Integration started (browser should open automatically).
echo ============================================
exit /b 0
