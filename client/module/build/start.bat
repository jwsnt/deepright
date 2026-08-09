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

:: Check if the WSL app files still exist
wsl -d deepright -- bash -c "test -x /app/integration && test -x /home/deepright/start-deepright.sh" 2>nul >nul
if %errorlevel% neq 0 (
    echo [X] DeepRight app files are missing inside WSL.
    echo.
    echo Please run install.bat first to restore the application files.
    echo ============================================
    pause
    exit /b 1
)

echo [i] Starting integration in deepright...
echo.

:: Run synchronously until the service is ready.
:: Use root for compatibility with existing installations whose integration log
:: was created by the root-owned integration service.
:: The wrapper suppresses its WSL-side browser opener; this script opens it
:: below from the interactive Windows desktop session.
wsl -d deepright -u root -- /home/deepright/start-deepright.sh
if %errorlevel% neq 0 (
    echo [X] Integration failed to start.
    echo ============================================
    pause
    exit /b 1
)

timeout /t 3 >nul

:: Open the page from the interactive Windows desktop session. The Integration
:: process itself runs as root inside WSL, which cannot reliably activate a
:: browser in the user's Windows session.
set "DEEPRIGHT_PORT=8080"
for /f "tokens=2 delims=:" %%P in ('wsl -d deepright -u root -- grep -m1 port /app/config/config.json 2^>nul') do set "DEEPRIGHT_PORT=%%P"
set "DEEPRIGHT_PORT=%DEEPRIGHT_PORT:,=%"
for /f "tokens=* delims= " %%P in ("%DEEPRIGHT_PORT%") do set "DEEPRIGHT_PORT=%%P"
start "" "http://localhost:%DEEPRIGHT_PORT%/launch"

echo [OK] Integration started. Opening http://localhost:%DEEPRIGHT_PORT%/launch
echo ============================================
exit /b 0
