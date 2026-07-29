#Requires -Version 5.1

param(
    [switch]$Force,
    [switch]$RemoveAll
)

$DISTRO_NAME = "deepright"
$WSL_VHD_PATH = "C:\WSL\deepright"
$PROGRAM_DATA_DIR = if ([string]::IsNullOrWhiteSpace($env:ProgramData)) { "C:\ProgramData" } else { $env:ProgramData }
$LOCAL_SENTINEL_FILE = Join-Path (Join-Path $PROGRAM_DATA_DIR "deepright") ".deepright_installed"
$LOCAL_CACHE_DIR = Join-Path $env:LOCALAPPDATA "DeepRight"
$SHORTCUT_NAME = "DeepRight"
$LOG_FILE = Join-Path $env:TEMP "deepright-uninstall.log"

function L_Step($m) { $l = "`n========================================  $m"; Write-Host $l -ForegroundColor Cyan; Add-Content -Path $LOG_FILE -Value $l -Encoding UTF8 }
function L_OK($m)   { Write-Host "  [OK] $m" -ForegroundColor Green; Add-Content -Path $LOG_FILE -Value "  [OK] $m" -Encoding UTF8 }
function L_Warn($m) { Write-Host "  [!] $m" -ForegroundColor Yellow; Add-Content -Path $LOG_FILE -Value "  [WARN] $m" -Encoding UTF8 }
function L_Err($m)  { Write-Host "  [X] $m" -ForegroundColor Red; Add-Content -Path $LOG_FILE -Value "  [ERROR] $m" -Encoding UTF8 }
function L_Info($m) { Write-Host "  [i] $m" -ForegroundColor Gray; Add-Content -Path $LOG_FILE -Value "  [INFO] $m" -Encoding UTF8 }
function L_Detail($m) { Write-Host "      $m" -ForegroundColor DarkGray; Add-Content -Path $LOG_FILE -Value "      $m" -Encoding UTF8 }

function Test-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($id)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Write-RemovalPreview([string]$Path, [string]$Label) {
    if ([string]::IsNullOrWhiteSpace($Path)) {
        L_Warn "$Label preview path is empty"
        return
    }

    if (-not (Test-Path -LiteralPath $Path)) {
        L_Info "$Label preview target not found: $Path"
        return
    }

    L_Info "$Label removal preview:"
    try {
        $item = Get-Item -LiteralPath $Path -Force -ErrorAction Stop
        L_Detail $item.FullName
        if ($item.PSIsContainer) {
            Get-ChildItem -LiteralPath $Path -Recurse -Force -ErrorAction SilentlyContinue |
                Sort-Object FullName |
                ForEach-Object { L_Detail $_.FullName }
        }
    } catch {
        L_Warn "$Label preview failed: $Path"
    }
}

function Remove-PathIfExists([string]$Path, [string]$Label, [switch]$AllowDelayedCleanup) {
    if (-not (Test-Path -LiteralPath $Path)) {
        L_Info "$Label not found: $Path"
        return
    }

    try {
        Write-RemovalPreview -Path $Path -Label $Label
        Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction Stop
        L_OK "$Label removed: $Path"
    } catch {
        if ($AllowDelayedCleanup) {
            L_Warn "$Label is still in use; cleanup will continue after the uninstaller exits: $Path"
            Start-DelayedDelete -Path $Path
            return
        }
        L_Err "$Label remove failed: $Path"
        throw
    }
}

function Remove-ShortcutIfExists([string]$ShortcutPath, [string]$Label) {
    if ([string]::IsNullOrWhiteSpace($ShortcutPath)) {
        L_Warn "Unable to resolve $Label shortcut path"
        return
    }

    if (-not (Test-Path -LiteralPath $ShortcutPath)) {
        L_Info "$Label shortcut not found: $ShortcutPath"
        return
    }

    try {
        Write-RemovalPreview -Path $ShortcutPath -Label "$Label shortcut"
        Remove-Item -LiteralPath $ShortcutPath -Force -ErrorAction Stop
        L_OK "$Label shortcut removed: $ShortcutPath"
    } catch {
        L_Err "$Label shortcut remove failed: $ShortcutPath"
        throw
    }
}

function Stop-Wsl() {
    L_Info "Shutting down WSL..."
    & wsl.exe --shutdown 2>&1 | Out-Null
    Start-Sleep -Seconds 2
    L_OK "WSL shutdown complete"
}

function Test-DistroExists([string]$DistroName) {
    L_Info "Checking distro: $DistroName"
    $listOutput = & wsl.exe -l -q 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "wsl -l -q:`n$listOutput" -Encoding UTF8
    $normalized = $listOutput -replace "`0", ""
    return (($normalized -split "[`r`n]+" | ForEach-Object { $_.Trim() }) -contains $DistroName)
}

function ConvertTo-BashSingleQuoted([string]$Value) {
    if ($null -eq $Value) {
        $Value = ""
    }
    # Close and reopen the bash single-quoted string around embedded single quotes.
    return "'" + $Value.Replace("'", "'""'""'") + "'"
}

function Remove-WslPathIfExists([string]$DistroName, [string]$Path, [string]$Label) {
    $quotedPath = ConvertTo-BashSingleQuoted $Path
    $probeOutput = & wsl.exe -d $DistroName -u root -- bash -c "if [ -e $quotedPath ]; then echo exists; else echo missing; fi" 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "probe ${Path}:`n$probeOutput" -Encoding UTF8
    if ($LASTEXITCODE -ne 0) {
        L_Err "$Label probe failed: $Path"
        throw "failed to probe WSL path: $Path"
    }

    if ($probeOutput.Trim() -ne "exists") {
        L_Info "$Label not found: $Path"
        return
    }

    $previewOutput = & wsl.exe -d $DistroName -u root -- bash -c "if [ -d $quotedPath ]; then find $quotedPath -print | LC_ALL=C sort; elif [ -e $quotedPath ]; then printf '%s\n' $quotedPath; fi" 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "preview ${Path}:`n$previewOutput" -Encoding UTF8
    if ($LASTEXITCODE -eq 0) {
        L_Info "$Label removal preview:"
        foreach ($line in ($previewOutput -split "[`r`n]+")) {
            $trimmed = $line.Trim()
            if ($trimmed -ne "") {
                L_Detail "[WSL] $trimmed"
            }
        }
    } else {
        L_Warn "$Label preview failed: $Path"
    }

    $removeOutput = & wsl.exe -d $DistroName -u root -- bash -c "rm -rf -- $quotedPath" 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "remove ${Path}:`n$removeOutput" -Encoding UTF8
    if ($LASTEXITCODE -ne 0) {
        L_Err "$Label remove failed: $Path"
        throw "failed to remove WSL path: $Path"
    }
    L_OK "$Label removed: $Path"
}

function Clear-DistroAppIfExists([string]$DistroName) {
    if (-not (Test-DistroExists -DistroName $DistroName)) {
        L_Info "WSL distro not found: $DistroName"
        return
    }

    Stop-Wsl
    Remove-WslPathIfExists -DistroName $DistroName -Path "/app" -Label "WSL app directory"
    Remove-WslPathIfExists -DistroName $DistroName -Path "/home/deepright/start-deepright.sh" -Label "WSL start wrapper"
    Remove-WslPathIfExists -DistroName $DistroName -Path "/home/deepright/.deepright_initialized" -Label "WSL sentinel"
    Stop-Wsl
}

function Unregister-DistroIfExists([string]$DistroName) {
    Stop-Wsl

    if (-not (Test-DistroExists -DistroName $DistroName)) {
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
    if ([string]::IsNullOrWhiteSpace($Path) -or -not (Test-Path -LiteralPath $Path)) {
        return
    }

    try {
        Write-RemovalPreview -Path $Path -Label "Delayed cleanup target"
        $escapedPath = $Path.Replace("'", "''")
        $cleanupScript = @"
Start-Sleep -Seconds 2
`$targetPath = '$escapedPath'
for (`$attempt = 0; `$attempt -lt 60; `$attempt++) {
    try {
        if (-not (Test-Path -LiteralPath `$targetPath)) { exit 0 }
        Remove-Item -LiteralPath `$targetPath -Recurse -Force -ErrorAction Stop
        exit 0
    } catch {
        Start-Sleep -Seconds 1
    }
}
exit 1
"@
        $encodedScript = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($cleanupScript))
        Start-Process -FilePath "powershell.exe" -WindowStyle Hidden -ArgumentList @("-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-EncodedCommand", $encodedScript)
        L_Info "Scheduled deferred cleanup (up to 60 seconds): $Path"
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

L_Step "DeepRight Uninstall"
L_Info "Log file: $LOG_FILE"

if (-not (Test-Admin)) {
    L_Err "This script requires Administrator privileges"
    exit 1
}

if (-not $Force) {
    Write-Host ""
    if ($RemoveAll) {
        Write-Host "This will fully remove DeepRight from Windows and WSL." -ForegroundColor Yellow
        Write-Host "It will delete shortcuts, local cache, the deepright WSL distro, C:\WSL\deepright, and all data under that distro." -ForegroundColor Yellow
    } else {
        Write-Host "This will remove DeepRight application files and Windows shortcuts, but keep the existing deepright WSL distro and agent-dir contents." -ForegroundColor Yellow
        Write-Host "Use uninstall.ps1 -RemoveAll if you also want to delete the distro and all WSL data." -ForegroundColor Yellow
    }
    $confirmation = Read-Host "Type YES or Y to continue"
    if (-not (Test-UninstallConfirmation -InputValue $confirmation)) {
        L_Warn "Uninstall cancelled by user"
        exit 0
    }
}

try {
    if ($RemoveAll) {
        L_Step "Step 1/4: Remove shortcuts"
    } else {
        L_Step "Step 1/3: Remove shortcuts"
    }
    Remove-ShortcutIfExists -ShortcutPath (Join-Path ([Environment]::GetFolderPath("DesktopDirectory")) "$SHORTCUT_NAME.lnk") -Label "Desktop"
    Remove-ShortcutIfExists -ShortcutPath (Join-Path ([Environment]::GetFolderPath("Programs")) "$SHORTCUT_NAME.lnk") -Label "Start Menu"

    if ($RemoveAll) {
        L_Step "Step 2/4: Remove Windows install state and extracted payload"
    } else {
        L_Step "Step 2/3: Remove Windows install state and extracted payload"
    }
    Remove-PathIfExists -Path $LOCAL_SENTINEL_FILE -Label "Install sentinel file"
    # The installer, Explorer, or a just-closed launcher can retain a handle
    # under this cache briefly. Do not abort an otherwise valid uninstall;
    # the detached cleanup process retries after this script has exited.
    Remove-PathIfExists -Path $LOCAL_CACHE_DIR -Label "Local extracted payload directory" -AllowDelayedCleanup

    if ($RemoveAll) {
        L_Step "Step 3/4: Remove WSL distro"
        Unregister-DistroIfExists -DistroName $DISTRO_NAME

        L_Step "Step 4/4: Remove WSL VHD directory"
        Remove-PathIfExists -Path $WSL_VHD_PATH -Label "WSL VHD directory"
    } else {
        L_Step "Step 3/3: Remove WSL application files (keep agent-dir)"
        Clear-DistroAppIfExists -DistroName $DISTRO_NAME
    }

    $launcherBundleRoot = Get-LauncherBundleRoot
    if ($launcherBundleRoot -and ($launcherBundleRoot -ne $LOCAL_CACHE_DIR)) {
        Start-DelayedDelete -Path $launcherBundleRoot
    }

    L_Step "Uninstall completed"
    if ($RemoveAll) {
        L_OK "DeepRight has been fully removed"
    } else {
        L_OK "DeepRight application files have been removed; existing agent-dir data was preserved"
    }
    exit 0
} catch {
    L_Err "Uninstall failed: $_"
    exit 1
}
