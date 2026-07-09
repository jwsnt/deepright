#Requires -Version 5.1

param(
    [switch]$Force
)

$DISTRO_NAME = "deepright"
$WSL_VHD_PATH = "C:\WSL\deepright"
$LOCAL_SENTINEL_DIR = "C:\ProgramData\deepright"
$LOCAL_CACHE_DIR = Join-Path $env:LOCALAPPDATA "DeepRight"
$SHORTCUT_NAME = "DeepRight"
$LOG_FILE = Join-Path $env:TEMP "deepright-uninstall.log"

function L_Step($m) { $l = "`n========================================  $m"; Write-Host $l -ForegroundColor Cyan; Add-Content -Path $LOG_FILE -Value $l -Encoding UTF8 }
function L_OK($m)   { Write-Host "  [OK] $m" -ForegroundColor Green; Add-Content -Path $LOG_FILE -Value "  [OK] $m" -Encoding UTF8 }
function L_Warn($m) { Write-Host "  [!] $m" -ForegroundColor Yellow; Add-Content -Path $LOG_FILE -Value "  [WARN] $m" -Encoding UTF8 }
function L_Err($m)  { Write-Host "  [X] $m" -ForegroundColor Red; Add-Content -Path $LOG_FILE -Value "  [ERROR] $m" -Encoding UTF8 }
function L_Info($m) { Write-Host "  [i] $m" -ForegroundColor Gray; Add-Content -Path $LOG_FILE -Value "  [INFO] $m" -Encoding UTF8 }

function Test-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($id)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Remove-PathIfExists([string]$Path, [string]$Label) {
    if (-not (Test-Path $Path)) {
        L_Info "$Label not found: $Path"
        return
    }

    try {
        Remove-Item -Path $Path -Recurse -Force -ErrorAction Stop
        L_OK "$Label removed: $Path"
    } catch {
        L_Err "$Label remove failed: $Path"
        throw
    }
}

function Remove-ShortcutIfExists([string]$ShortcutPath, [string]$Label) {
    if ([string]::IsNullOrWhiteSpace($ShortcutPath)) {
        L_Warn "Unable to resolve $Label shortcut path"
        return
    }

    if (-not (Test-Path $ShortcutPath)) {
        L_Info "$Label shortcut not found: $ShortcutPath"
        return
    }

    try {
        Remove-Item -Path $ShortcutPath -Force -ErrorAction Stop
        L_OK "$Label shortcut removed: $ShortcutPath"
    } catch {
        L_Err "$Label shortcut remove failed: $ShortcutPath"
        throw
    }
}

function Unregister-DistroIfExists([string]$DistroName) {
    L_Info "Shutting down WSL..."
    & wsl.exe --shutdown 2>&1 | Out-Null
    Start-Sleep -Seconds 2
    L_OK "WSL shutdown complete"

    L_Info "Checking distro: $DistroName"
    $listOutput = & wsl.exe -l -q 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "wsl -l -q:`n$listOutput" -Encoding UTF8
    $normalized = $listOutput -replace "`0", ""
    $distroExists = ($normalized -split "[`r`n]+" | ForEach-Object { $_.Trim() }) -contains $DistroName

    if (-not $distroExists) {
        L_Info "WSL distro not found: $DistroName"
        return
    }

    L_Info "Unregistering distro: $DistroName"
    $unregisterOutput = & wsl.exe --unregister $DistroName 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "wsl --unregister ${DistroName}:`n$unregisterOutput" -Encoding UTF8
    if ($LASTEXITCODE -ne 0) {
        L_Err "Failed to unregister distro: $DistroName"
        throw "wsl --unregister failed"
    }
    L_OK "WSL distro removed: $DistroName"
}

function Start-DelayedDelete([string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Path) -or -not (Test-Path $Path)) {
        return
    }

    try {
        Start-Process -FilePath "cmd.exe" -WindowStyle Hidden -ArgumentList "/c", "ping 127.0.0.1 -n 4 >nul && rmdir /s /q `"$Path`""
        L_Info "Scheduled cleanup: $Path"
    } catch {
        L_Warn "Failed to schedule cleanup: $Path"
    }
}

function Get-LauncherBundleRoot() {
    $windowsExeDir = Split-Path -Parent $PSScriptRoot
    if ([string]::IsNullOrWhiteSpace($windowsExeDir)) {
        return $null
    }
    return Split-Path -Parent $windowsExeDir
}

function Test-UninstallConfirmation([string]$InputValue) {
    if ($null -eq $InputValue) {
        return $false
    }

    $normalized = $InputValue.Trim().ToLowerInvariant()
    return $normalized -eq "y" -or $normalized -eq "yes"
}

$runHeader = "`n========================================  NEW RUN: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
Add-Content -Path $LOG_FILE -Value $runHeader -Encoding UTF8

L_Step "DeepRight Full Uninstall"
L_Info "Log file: $LOG_FILE"

if (-not (Test-Admin)) {
    L_Err "This script requires Administrator privileges"
    exit 1
}

if (-not $Force) {
    Write-Host ""
    Write-Host "This will fully remove DeepRight from Windows and WSL." -ForegroundColor Yellow
    Write-Host "It will delete shortcuts, local cache, the deepright WSL distro, and C:\WSL\deepright." -ForegroundColor Yellow
    $confirmation = Read-Host "Type YES or Y to continue"
    if (-not (Test-UninstallConfirmation -InputValue $confirmation)) {
        L_Warn "Uninstall cancelled by user"
        exit 0
    }
}

try {
    L_Step "Step 1/4: Remove shortcuts"
    Remove-ShortcutIfExists -ShortcutPath (Join-Path ([Environment]::GetFolderPath("DesktopDirectory")) "$SHORTCUT_NAME.lnk") -Label "Desktop"
    Remove-ShortcutIfExists -ShortcutPath (Join-Path ([Environment]::GetFolderPath("Programs")) "$SHORTCUT_NAME.lnk") -Label "Start Menu"

    L_Step "Step 2/4: Remove Windows install state and extracted payload"
    Remove-PathIfExists -Path $LOCAL_SENTINEL_DIR -Label "Install sentinel directory"
    Remove-PathIfExists -Path $LOCAL_CACHE_DIR -Label "Local extracted payload directory"

    L_Step "Step 3/4: Remove WSL distro"
    Unregister-DistroIfExists -DistroName $DISTRO_NAME

    L_Step "Step 4/4: Remove WSL VHD directory"
    Remove-PathIfExists -Path $WSL_VHD_PATH -Label "WSL VHD directory"

    $launcherBundleRoot = Get-LauncherBundleRoot
    if ($launcherBundleRoot -and ($launcherBundleRoot -ne $LOCAL_CACHE_DIR)) {
        Start-DelayedDelete -Path $launcherBundleRoot
    }

    L_Step "Uninstall completed"
    L_OK "DeepRight has been fully removed"
    exit 0
} catch {
    L_Err "Uninstall failed: $_"
    exit 1
}
