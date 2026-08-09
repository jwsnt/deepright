@echo off
setlocal

echo ============================================
echo    DeepRight Uninstaller
echo ============================================
echo.

net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Requesting administrator privileges...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

echo Running as Administrator
echo.

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0uninstall.ps1"
set "EXIT_CODE=%errorlevel%"

echo.
if %EXIT_CODE% equ 0 (
    echo ============================================
    echo Uninstallation finished. Press any key to exit.
    echo ============================================
) else (
    echo ============================================
    echo Uninstallation failed. Press any key to exit.
    echo ============================================
)
pause >nul
exit /b %EXIT_CODE%
