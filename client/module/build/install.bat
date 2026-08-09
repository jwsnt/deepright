@echo off
setlocal enabledelayedexpansion

echo ============================================
echo    Deepright WSL2 Installer
echo ============================================
echo.

:: Check admin
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Requesting administrator privileges...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

echo Running as Administrator
echo.

:: Run PowerShell script
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0install.ps1"

echo.
echo ============================================
echo Installation finished. Press any key to exit.
echo ============================================
pause >nul
