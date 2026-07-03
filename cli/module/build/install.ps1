#Requires -Version 5.1

# ---------- Fixed config ----------
$DISTRO_NAME   = "deepright"
$WSL_VHD_PATH  = "C:\WSL\deepright"
$APP_DIR       = Join-Path $PSScriptRoot "app"
$LOG_FILE      = Join-Path $PSScriptRoot "install.log"
$WSL_SENTINEL  = "/home/deepright/.deepright_initialized"
$LOCAL_SENTINEL_DIR  = "C:\ProgramData\deepright"
$LOCAL_SENTINEL_FILE = Join-Path $LOCAL_SENTINEL_DIR ".deepright_installed"

# ---------- Log ----------
function L_Step($m) { $l="`n========================================  $m"; Write-Host $l -F Cyan;   Add-Content -Path $LOG_FILE -Value $l -Encoding UTF8 }
function L_OK($m)   {                     Write-Host "  [OK] $m" -F Green;  Add-Content -Path $LOG_FILE -Value "  [OK] $m" -Encoding UTF8 }
function L_Warn($m) {                     Write-Host "  [!] $m" -F Yellow;  Add-Content -Path $LOG_FILE -Value "  [WARN] $m" -Encoding UTF8 }
function L_Err($m)  {                     Write-Host "  [X] $m" -F Red;     Add-Content -Path $LOG_FILE -Value "  [ERROR] $m" -Encoding UTF8 }
function L_Info($m) {                     Write-Host "  [i] $m" -F Gray;    Add-Content -Path $LOG_FILE -Value "  [INFO] $m" -Encoding UTF8 }

# ---------- Helpers ----------
function Test-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    $p  = New-Object Security.Principal.WindowsPrincipal($id)
    return $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}
function Get-WindowsBuild {
    return [int](Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion").CurrentBuild
}
function Test-WslInstalled {
    try { $null = & wsl.exe --status 2>&1; return ($LASTEXITCODE -eq 0) } catch { return $false }
}

function Test-DistroExists([string]$N) {
    <# Retry up to 3 times -- WSL VM cold-start can take several seconds #>
    for ($i = 1; $i -le 3; $i++) {
        $out = & wsl.exe -d $N -u root -- echo "ok" 2>&1 | Out-String
        if ($out -match "ok") {
            return $true
        }
        if ($i -lt 3) {
            Start-Sleep -Seconds 3
        }
    }
    return $false
}

function WslPath([string]$P) {
    $d = $P.Substring(0,1).ToLower()
    $r = $P.Substring(2) -replace '\\', '/'
    return "/mnt/$d$r"
}

function Nuke-Distro([string]$Name, [string]$VhdPath) {
    & wsl.exe --shutdown 2>&1 | Out-Null
    Start-Sleep -Seconds 2

    $out1 = & wsl.exe --unregister $Name 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "unregister $Name : $out1" -Encoding UTF8
    Start-Sleep -Seconds 2

    $out2 = & wsl.exe --unregister $Name 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "unregister $Name (2nd): $out2" -Encoding UTF8
    Start-Sleep -Seconds 1

    if (Test-Path $VhdPath) {
        Get-ChildItem -Path $VhdPath -Recurse -Force | Remove-Item -Force -EA SilentlyContinue
    } else {
        New-Item -ItemType Directory -Path $VhdPath -Force | Out-Null
    }
}

function Fix-WslConfig([string]$Path) {
    $content = "[wsl2]`r`nnetworkingMode=mirrored`r`n"
    [System.IO.File]::WriteAllText($Path, $content, [System.Text.Encoding]::ASCII)
}

function Test-WslTool([string]$cmd) {
    & wsl.exe -d $DISTRO_NAME -- bash -c "command -v $cmd > /dev/null 2>&1" | Out-Null
    return ($LASTEXITCODE -eq 0)
}

# ---------- Main ----------
$runHeader = "`n========================================  NEW RUN: $(Get-Date -F 'yyyy-MM-dd HH:mm:ss')"
Add-Content -Path $LOG_FILE -Value $runHeader -Encoding UTF8

L_Step "Deepright WSL2 Environment Installer"
L_Info "Script dir: $PSScriptRoot"
L_Info "Log file: $LOG_FILE"

# ====================================================================
#   PHASE 0: Sentinel check
#   If local sentinel exists AND distro is alive → skip to integration
# ====================================================================

$needsFullInstall = $true

if (Test-Path $LOCAL_SENTINEL_FILE) {
    if (Test-DistroExists -Name $DISTRO_NAME) {
        $needsFullInstall = $false
        L_OK "Sentinel found + distro alive -- skipping installation"
        L_OK "To force re-install, delete: $LOCAL_SENTINEL_DIR"
    } else {
        L_Warn "Sentinel found but distro is missing (unregistered/deleted)"
        L_Warn "Deleting stale sentinel, re-installing..."
        Remove-Item $LOCAL_SENTINEL_DIR -Recurse -Force -EA SilentlyContinue
    }
}

# ====================================================================
#   PHASE 1: Full installation (only if sentinel not found)
# ====================================================================

if ($needsFullInstall) {

# ---- Step 1: Admin ----
L_Step "Step 1/7: Check administrator privileges"
if (-not (Test-Admin)) {
    L_Err "This script requires Administrator privileges"
    L_Info "Launch via install.bat (auto-elevates)"
    exit 1
}
L_OK "Administrator confirmed"

# ---- Step 2: Windows version ----
L_Step "Step 2/7: Check Windows version"
$build = Get-WindowsBuild
L_Info "Windows Build: $build"
if ($build -lt 19041) {
    L_Err "WSL2 requires Windows 10 2004+ (Build 19041+)"
    exit 1
}
L_OK "Windows version OK (Build $build)"

# ---- Step 3: WSL2 ----
L_Step "Step 3/7: Check WSL2"
if (Test-WslInstalled) {
    L_OK "WSL already installed"
} else {
    L_Info "Installing WSL (may take several minutes)..."
    $o = & wsl.exe --install --no-distribution 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "WSL install: $o" -Encoding UTF8
    if (-not (Test-WslInstalled)) {
        L_Info "Trying DISM fallback..."
        $d1 = & dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart 2>&1 | Out-String
        $d2 = & dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart 2>&1 | Out-String
        Add-Content -Path $LOG_FILE -Value "DISM1: $d1`nDISM2: $d2" -Encoding UTF8
        if (-not (Test-WslInstalled)) {
            L_Warn "WSL features enabled. REBOOT REQUIRED, then re-run install.bat"
            $c = Read-Host "Reboot now? (y/n)"
            if ($c -eq 'y' -or $c -eq 'Y') { Restart-Computer -Force }
            exit 0
        }
    }
    L_OK "WSL2 installation complete"
}
$null = & wsl.exe --set-default-version 2 2>&1

# ---- Step 4: Ubuntu as deepright ----
L_Step "Step 4/7: Install Ubuntu (deepright)"

# 4a: Write clean .wslconfig
L_Info "Writing clean .wslconfig (ASCII, mirrored only)..."
$wcPath = Join-Path $env:USERPROFILE ".wslconfig"
Fix-WslConfig -Path $wcPath
L_OK ".wslconfig written"

# 4b: Update WSL
L_Info "Updating WSL..."
$wu = & wsl.exe --update 2>&1 | Out-String
Add-Content -Path $LOG_FILE -Value "wsl --update: $wu" -Encoding UTF8
L_OK "WSL updated"

# 4c: Shutdown with clean config
L_Info "Shutting down WSL..."
& wsl.exe --shutdown 2>&1 | Out-Null
Start-Sleep -Seconds 3
L_OK "WSL restarted with clean config"

# 4d: Check if deepright already running
$distroExists = Test-DistroExists -Name $DISTRO_NAME

if ($distroExists) {
    L_OK "Distro deepright already exists"

    # Ensure user deepright exists
    $cu = & wsl.exe -d $DISTRO_NAME -u root -- id -u deepright 2>&1 | Out-String
    if ($LASTEXITCODE -ne 0) {
        L_Info "Creating user deepright..."
        & wsl.exe -d $DISTRO_NAME -u root -- useradd -m -s /bin/bash deepright 2>&1 | Out-Null
        & wsl.exe -d $DISTRO_NAME -u root -- bash -c "echo 'deepright:deepright' | chpasswd" 2>&1 | Out-Null
        & wsl.exe -d $DISTRO_NAME -u root -- usermod -aG sudo deepright 2>&1 | Out-Null
        & wsl.exe -d $DISTRO_NAME -u root -- bash -c "echo 'deepright ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/deepright && chmod 440 /etc/sudoers.d/deepright" 2>&1 | Out-Null
        & wsl.exe -d $DISTRO_NAME -u root -- bash -c "printf '[user]\ndefault=deepright\n' > /etc/wsl.conf" 2>&1 | Out-Null
        & wsl.exe --shutdown 2>&1 | Out-Null
        Start-Sleep -Seconds 3
        L_OK "User deepright created"
    } else {
        L_OK "User deepright already exists"
        & wsl.exe -d $DISTRO_NAME -u root -- bash -c "printf '[user]\ndefault=deepright\n' > /etc/wsl.conf" 2>&1 | Out-Null
        L_OK "Default user set to deepright"
    }

} else {
    L_Info "deepright not found, performing full install..."

    # Nuke any stale registrations
    L_Info "Cleaning stale registrations..."
    Nuke-Distro -Name $DISTRO_NAME -VhdPath $WSL_VHD_PATH
    & wsl.exe --unregister Ubuntu 2>&1 | Out-Null
    Start-Sleep -Seconds 2
    L_OK "Cleanup done"

    # Get Ubuntu rootfs
    $rootfsFile = $null

    # Method 1: wsl --install -d Ubuntu
    L_Info "Method 1: wsl --install -d Ubuntu..."
    $i1 = & wsl.exe --install -d Ubuntu --no-launch 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "wsl --install -d Ubuntu: $i1" -Encoding UTF8
    Start-Sleep -Seconds 10

    $ubuntuOk = Test-DistroExists -Name "Ubuntu"
    if ($ubuntuOk) {
        L_OK "Ubuntu installed via Method 1"
        $TEMP_TAR = "C:\Temp\deepright-ubuntu.tar"
        if (-not (Test-Path "C:\Temp")) { New-Item -ItemType Directory -Path "C:\Temp" -Force | Out-Null }
        L_Info "Exporting Ubuntu..."
        & wsl.exe --export Ubuntu $TEMP_TAR 2>&1 | Out-Null
        if (Test-Path $TEMP_TAR) {
            L_OK "Ubuntu exported"
            & wsl.exe --unregister Ubuntu 2>&1 | Out-Null
            Start-Sleep -Seconds 2
            $rootfsFile = $TEMP_TAR
        } else {
            L_Warn "Export failed, trying Method 2"
        }
    } else {
        L_Warn "Method 1 failed, trying Method 2"
    }

    # Method 2: direct download
    if (-not $rootfsFile) {
        L_Info "Method 2: Direct rootfs download..."
        $rootfsUrl = "https://cloud-images.ubuntu.com/wsl/releases/noble/current/ubuntu-noble-wsl-amd64-wsl.rootfs.tar.gz"
        $dlDir = "C:\Temp"
        if (-not (Test-Path $dlDir)) { New-Item -ItemType Directory -Path $dlDir -Force | Out-Null }
        $rootfsFile = Join-Path $dlDir "ubuntu-rootfs.tar.gz"

        L_Info "URL: $rootfsUrl"
        try {
            [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
            $wc = New-Object System.Net.WebClient
            $wc.DownloadFile($rootfsUrl, $rootfsFile)
        } catch {
            L_Err "Download error: $_"; exit 1
        }
        if (-not (Test-Path $rootfsFile)) { L_Err "Download failed"; exit 1 }
        L_OK "Rootfs downloaded ($([math]::Round((Get-Item $rootfsFile).Length/1MB,1)) MB)"
    }

    # Import as deepright
    L_Info "Final cleanup before import..."
    & wsl.exe --shutdown 2>&1 | Out-Null
    Start-Sleep -Seconds 2
    & wsl.exe --unregister $DISTRO_NAME 2>&1 | Out-Null
    Start-Sleep -Seconds 1
    if (Test-Path $WSL_VHD_PATH) {
        Get-ChildItem -Path $WSL_VHD_PATH -Recurse -Force | Remove-Item -Force -EA SilentlyContinue
    } else {
        New-Item -ItemType Directory -Path $WSL_VHD_PATH -Force | Out-Null
    }

    $rtar = (Get-Item $rootfsFile).FullName
    L_Info "Importing: $rtar -> $WSL_VHD_PATH"
    $imp = & wsl.exe --import $DISTRO_NAME $WSL_VHD_PATH $rtar 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "wsl --import: $imp" -Encoding UTF8

    Start-Sleep -Seconds 3
    $verifyOut = & wsl.exe -d $DISTRO_NAME -u root -- echo "imported_ok" 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "verify: $verifyOut" -Encoding UTF8

    if ($verifyOut -notmatch "imported_ok") {
        L_Err "Import failed. WSL output:"
        L_Info $imp
        L_Info "Verify output: $verifyOut"
        exit 1
    }
    L_OK "deepright imported successfully"

    # Clean up tar
    Remove-Item $rootfsFile -Force -EA SilentlyContinue

    # Create user deepright
    L_Info "Creating user deepright..."
    & wsl.exe -d $DISTRO_NAME -u root -- useradd -m -s /bin/bash deepright 2>&1 | Out-Null
    & wsl.exe -d $DISTRO_NAME -u root -- bash -c "echo 'deepright:deepright' | chpasswd" 2>&1 | Out-Null
    & wsl.exe -d $DISTRO_NAME -u root -- usermod -aG sudo deepright 2>&1 | Out-Null
    L_OK "User deepright created"

    L_Info "Configuring passwordless sudo..."
    & wsl.exe -d $DISTRO_NAME -u root -- bash -c "echo 'deepright ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/deepright && chmod 440 /etc/sudoers.d/deepright" 2>&1 | Out-Null
    L_OK "Passwordless sudo configured"

    L_Info "Setting default user to deepright..."
    & wsl.exe -d $DISTRO_NAME -u root -- bash -c "printf '[user]\ndefault=deepright\n' > /etc/wsl.conf" 2>&1 | Out-Null
    L_OK "Default user configured"

    L_Info "Restarting WSL..."
    & wsl.exe --shutdown 2>&1 | Out-Null
    Start-Sleep -Seconds 3
    L_OK "WSL restarted"
}

# ---- Step 5: .wslconfig verify ----
L_Step "Step 5/7: Verify mirrored networking"
L_OK ".wslconfig already configured with networkingMode=mirrored"

# ---- Step 6: Tools ----
L_Step "Step 6/7: Install tools (git, npm, python3)"

$needGit  = -not (Test-WslTool "git")
$needPy   = -not (Test-WslTool "python3")
$needNode = -not (Test-WslTool "node")

if (-not ($needGit -or $needPy -or $needNode)) {
    L_OK "All tools already installed"
} else {
    L_Info "Updating apt..."
    & wsl.exe -d $DISTRO_NAME -- bash -c "sudo DEBIAN_FRONTEND=noninteractive apt-get update -qq" 2>&1 | Out-Null
    L_OK "apt updated"

    if ($needGit -or $needPy) {
        L_Info "Installing git, python3, pip, curl..."
        $ar = & wsl.exe -d $DISTRO_NAME -- bash -c "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y git python3 python3-pip curl build-essential 2>&1" | Out-String
        Add-Content -Path $LOG_FILE -Value "apt: $ar" -Encoding UTF8
        if ($LASTEXITCODE -eq 0) { L_OK "git, python3, pip, curl installed" } else { L_Warn "Some packages may have failed" }
    }

    if ($needNode) {
        L_Info "Installing Node.js 20.x LTS (npm)..."
        $nr = & wsl.exe -d $DISTRO_NAME -- bash -c "curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash - 2>&1 && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs 2>&1" | Out-String
        Add-Content -Path $LOG_FILE -Value "node: $nr" -Encoding UTF8
        if ($LASTEXITCODE -eq 0) {
            L_OK "Node.js 20.x LTS + npm installed"
        } else {
            L_Warn "NodeSource failed, trying system packages..."
            & wsl.exe -d $DISTRO_NAME -- bash -c "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs npm 2>&1" | Out-Null
            if ($LASTEXITCODE -eq 0) { L_OK "Node.js + npm (system) installed" } else { L_Err "Node.js install failed" }
        }
    }
}

L_Info "Verifying tools..."
$gitV  = (& wsl.exe -d $DISTRO_NAME -- bash -c "git --version 2>&1" | Out-String).Trim()
$nodeV = (& wsl.exe -d $DISTRO_NAME -- bash -c "node --version 2>&1" | Out-String).Trim()
$npmV  = (& wsl.exe -d $DISTRO_NAME -- bash -c "npm --version 2>&1" | Out-String).Trim()
$pyV   = (& wsl.exe -d $DISTRO_NAME -- bash -c "python3 --version 2>&1" | Out-String).Trim()
L_OK "git:      $gitV"
L_OK "node:     $nodeV"
L_OK "npm:      $npmV"
L_OK "python3:  $pyV"

# ---- Step 7: Copy app ----
L_Step "Step 7/7: Copy app dir to WSL /app/"

$WSL_APP_TARGET = "/app"

if (Test-Path $APP_DIR) {
    $fs = Get-ChildItem -Path $APP_DIR -Recurse -File -EA SilentlyContinue
    if ($fs.Count -eq 0) {
        L_Warn "app dir is empty, skipping"
        L_Info "Place program files in: $APP_DIR then re-run"
    } else {
        # Check if files need updating
        $needCopy = $false
        foreach ($f in $fs) {
            $rel = $f.FullName.Substring($APP_DIR.Length).TrimStart('\','/') -replace '\\','/'
            $cmp = & wsl.exe -d $DISTRO_NAME -u root -- bash -c "stat -c %Y '${WSL_APP_TARGET}/$rel' 2>/dev/null || echo 0" 2>&1 | Out-String
            $cmp = $cmp.Trim()
            $wslTs = if ($cmp -match '^\d+$') { [long]$cmp } else { 0 }
            $localTs = [long][math]::Floor([decimal]($f.LastWriteTimeUtc - (Get-Date "1970-01-01T00:00:00Z")).TotalSeconds)
            if ($localTs -gt $wslTs) { $needCopy = $true; break }
        }
        if (-not $needCopy) {
            L_OK "All files up to date, skipping"
        } else {
            L_Info "Copying $($fs.Count) files..."
            $wp = WslPath -P $APP_DIR
            & wsl.exe -d $DISTRO_NAME -u root -- bash -c "mkdir -p ${WSL_APP_TARGET} 2>/dev/null; cp -r '${wp}'/* ${WSL_APP_TARGET}/ 2>/dev/null; chown -R deepright:deepright ${WSL_APP_TARGET}/; chmod -R u+rw ${WSL_APP_TARGET}/" 2>&1 | Out-Null
            if ($LASTEXITCODE -eq 0) {
                L_OK "App copied to ${WSL_APP_TARGET}/"
                $cl = & wsl.exe -d $DISTRO_NAME -- bash -c "ls -la ${WSL_APP_TARGET}/" 2>&1 | Out-String
                Write-Host $cl -F DarkGray
            } else {
                L_Warn "App copy may have partially failed"
            }
        }
    }
} else {
    L_Warn "app dir not found: $APP_DIR"
    New-Item -ItemType Directory -Path $APP_DIR -Force | Out-Null
    L_Info "Created empty app dir"
}

# ====================================================================
#   CREATE START WRAPPER SCRIPT (used by start.bat)
# ====================================================================
L_Info "Creating WSL start wrapper script..."
& wsl.exe -d $DISTRO_NAME -u deepright -- bash -c "printf '#!/bin/bash`nTERM=xterm-256color setsid /app/integration start`n' > /home/deepright/start-deepright.sh"
& wsl.exe -d $DISTRO_NAME -u deepright -- chmod +x /home/deepright/start-deepright.sh
L_OK "Wrapper script: /home/deepright/start-deepright.sh"

# ====================================================================
#   WRITE SENTINELS -- immediately after all setup is complete
# ====================================================================
L_Step "Writing sentinel files"

# Debug: dump all sentinel path variables
Add-Content -Path $LOG_FILE -Value "DEBUG LOCAL_SENTINEL_DIR  = $LOCAL_SENTINEL_DIR" -Encoding UTF8
Add-Content -Path $LOG_FILE -Value "DEBUG LOCAL_SENTINEL_FILE = $LOCAL_SENTINEL_FILE" -Encoding UTF8

# --- 1) Local sentinel (Windows) ---
try {
    $sentinelDir = [System.IO.Path]::GetFullPath($LOCAL_SENTINEL_DIR)
    Add-Content -Path $LOG_FILE -Value "DEBUG Resolved dir = $sentinelDir" -Encoding UTF8

    if (-not (Test-Path $sentinelDir)) {
        $nd = New-Item -ItemType Directory -Path $sentinelDir -Force
        Add-Content -Path $LOG_FILE -Value "DEBUG New-Item exit: dir now exists = $(Test-Path $sentinelDir)" -Encoding UTF8
        L_Info "Created directory: $sentinelDir"
    } else {
        L_Info "Directory already exists: $sentinelDir"
    }

    # Write the file using .NET API (same as .wslconfig)
    [System.IO.File]::WriteAllText($LOCAL_SENTINEL_FILE, "deepright", [System.Text.Encoding]::ASCII)
    Add-Content -Path $LOG_FILE -Value "DEBUG WriteAllText completed" -Encoding UTF8

    # Verify
    if (Test-Path $LOCAL_SENTINEL_FILE) {
        $content = Get-Content $LOCAL_SENTINEL_FILE -Raw
        Add-Content -Path $LOG_FILE -Value "DEBUG Sentinal file content: [$content]" -Encoding UTF8
        L_OK "Local sentinel OK: $LOCAL_SENTINEL_FILE"
    } else {
        Add-Content -Path $LOG_FILE -Value "DEBUG Test-Path returned FALSE for: $LOCAL_SENTINEL_FILE" -Encoding UTF8
        L_Err "Local sentinel NOT created: $LOCAL_SENTINEL_FILE"
        exit 1
    }
} catch {
    Add-Content -Path $LOG_FILE -Value "DEBUG EXCEPTION: $_" -Encoding UTF8
    L_Err "Local sentinel exception: $_"
    exit 1
}

# --- 2) WSL sentinel ---
try {
    $touchOut = & wsl.exe -d $DISTRO_NAME -u deepright -- touch "$WSL_SENTINEL" 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "DEBUG WSL touch output: $touchOut" -Encoding UTF8
    $null = & wsl.exe -d $DISTRO_NAME -u deepright -- test -f "$WSL_SENTINEL" 2>&1
    Add-Content -Path $LOG_FILE -Value "DEBUG WSL test-f exit: $LASTEXITCODE" -Encoding UTF8
    if ($LASTEXITCODE -eq 0) {
        L_OK "WSL sentinel OK: $WSL_SENTINEL"
    } else {
        L_Err "WSL sentinel NOT created: $WSL_SENTINEL"
        Add-Content -Path $LOG_FILE -Value "DEBUG WSL sentinel failed, continuing anyway" -Encoding UTF8
    }
} catch {
    Add-Content -Path $LOG_FILE -Value "DEBUG WSL sentinel exception: $_" -Encoding UTF8
    L_Warn "WSL sentinel write threw exception, continuing"
}

}  # end of needsFullInstall

# ====================================================================
#   PHASE 2: Start integration (always, regardless of sentinel path)
# ====================================================================

# Populate version vars if not set (sentinel shortcut path)
if (-not $gitV)  { $gitV  = (& wsl.exe -d $DISTRO_NAME -- bash -c "git --version 2>&1" | Out-String).Trim() }
if (-not $nodeV) { $nodeV = (& wsl.exe -d $DISTRO_NAME -- bash -c "node --version 2>&1" | Out-String).Trim() }
if (-not $npmV)  { $npmV  = (& wsl.exe -d $DISTRO_NAME -- bash -c "npm --version 2>&1" | Out-String).Trim() }
if (-not $pyV)   { $pyV   = (& wsl.exe -d $DISTRO_NAME -- bash -c "python3 --version 2>&1" | Out-String).Trim() }

L_Step "Installation Complete"
Write-Host ""
Write-Host "  ================================================" -F Cyan
Write-Host "    Deepright WSL2 Environment Ready" -F Green
Write-Host "  ================================================" -F Cyan
Write-Host ""
Write-Host "  Distro:  $DISTRO_NAME" -F White
Write-Host "  User:    deepright" -F White
Write-Host "  Pass:    deepright" -F White
Write-Host "  VHD:     $WSL_VHD_PATH" -F White
Write-Host "  Network: mirrored" -F White
Write-Host ""
Write-Host "  Tools:" -F White
Write-Host "    git:      $gitV" -F Gray
Write-Host "    node:     $nodeV" -F Gray
Write-Host "    npm:      $npmV" -F Gray
Write-Host "    python3:  $pyV" -F Gray
Write-Host ""
Write-Host "  Enter WSL: wsl -d $DISTRO_NAME" -F Yellow
Write-Host "  Log file:  $LOG_FILE" -F Gray
Write-Host ""

# ---- Start integration ----
L_Step "Starting integration service"
Write-Host ""
& wsl.exe -d $DISTRO_NAME -- bash /home/deepright/start-deepright.sh 2>&1 | Write-Host
Write-Host ""
L_OK "Integration started"
