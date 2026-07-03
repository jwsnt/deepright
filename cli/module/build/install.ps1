#Requires -Version 5.1

[CmdletBinding()]
param(
  [string]$DistroName = "deepright",
  [string]$TargetDirName = "deepright",
  [switch]$SkipLaunch
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$ManagedUserName = "deepright"
$InstallerVersion = "2026-07-03-build-module-sync-1"
$LogFile = Join-Path $PSScriptRoot "install.log"
$RootfsRelease = "noble"

function Get-ExpectedWSLArchitecture {
  $processorArch = [string]$env:PROCESSOR_ARCHITECTURE
  switch ($processorArch.ToUpperInvariant()) {
    "ARM64" { return "aarch64" }
    default { return "x86_64" }
  }
}

function Get-RootfsArchTag {
  switch (Get-ExpectedWSLArchitecture) {
    "aarch64" { return "arm64" }
    default { return "amd64" }
  }
}

function Write-LogLine {
  param([string]$Message)

  $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
  Add-Content -Path $LogFile -Value ("[{0}] {1}" -f $timestamp, $Message) -Encoding UTF8
}

function Write-Info {
  param([string]$Message)

  Write-Host "[DeepRight WSL2] $Message"
  Write-LogLine $Message
}

function Test-IsAdministrator {
  $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
  $principal = New-Object Security.Principal.WindowsPrincipal($identity)
  return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Ensure-Administrator {
  if (-not (Test-IsAdministrator)) {
    throw "This script requires Administrator privileges. Please launch via install.bat."
  }
}

function Invoke-NativeCommand {
  param(
    [string]$FilePath,
    [string[]]$Arguments = @(),
    [switch]$IgnoreExitCode
  )

  Write-LogLine ("EXEC " + $FilePath + " " + ($Arguments -join " "))

  $escapedArguments = @()
  foreach ($arg in $Arguments) {
    $text = [string]$arg
    if ($text -match '[\s"]') {
      $text = '"' + ($text -replace '(\\*)"', '$1$1\"' -replace '(\\+)$', '$1$1') + '"'
    }
    $escapedArguments += $text
  }

  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = $FilePath
  $psi.Arguments = ($escapedArguments -join " ")
  $psi.UseShellExecute = $false
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true

  $process = New-Object System.Diagnostics.Process
  $process.StartInfo = $psi
  [void]$process.Start()
  $stdout = $process.StandardOutput.ReadToEnd()
  $stderr = $process.StandardError.ReadToEnd()
  $process.WaitForExit()
  $exitCode = $process.ExitCode
  $output = (($stdout.TrimEnd()) + [Environment]::NewLine + ($stderr.TrimEnd())).Trim()

  if (-not [string]::IsNullOrWhiteSpace($output)) {
    Write-LogLine $output
  }

  if (-not $IgnoreExitCode -and $exitCode -ne 0) {
    if ([string]::IsNullOrWhiteSpace($output)) {
      throw "$FilePath failed with exit code $exitCode"
    }
    throw "$FilePath failed with exit code $exitCode`n$output"
  }

  return @{
    Output = $output
    ExitCode = $exitCode
  }
}

function Test-RestartRequired {
  param($CommandResult)

  if ($null -eq $CommandResult) {
    return $false
  }
  if ($CommandResult.ExitCode -eq 3010) {
    return $true
  }
  $text = ""
  if ($CommandResult.ContainsKey("Output")) {
    $text = [string]$CommandResult.Output
  }
  return $text -match "(?im)restart|reboot"
}

function Get-WindowsBuildNumber {
  $version = Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion"
  return [int]$version.CurrentBuildNumber
}

function Assert-WSL2Supported {
  $build = Get-WindowsBuildNumber
  if ($build -lt 19041) {
    throw "Current Windows build $build does not support WSL2. Windows 10 2004 (Build 19041) or later is required."
  }
}

function Ensure-WSLInstalled {
  $status = Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--status") -IgnoreExitCode
  if ($status.ExitCode -eq 0) {
    Write-Info "WSL is already available."
  } else {
    Write-Info "WSL is not available. Installing WSL2 components."
    $install = Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--install", "--no-distribution") -IgnoreExitCode
    if ($install.ExitCode -ne 0) {
      Write-Info "wsl --install failed. Trying DISM fallback."
      $featureWSL = Invoke-NativeCommand -FilePath "dism.exe" -Arguments @("/online", "/enable-feature", "/featurename:Microsoft-Windows-Subsystem-Linux", "/all", "/norestart") -IgnoreExitCode
      $featureVM = Invoke-NativeCommand -FilePath "dism.exe" -Arguments @("/online", "/enable-feature", "/featurename:VirtualMachinePlatform", "/all", "/norestart") -IgnoreExitCode
      if ($featureWSL.ExitCode -ne 0 -or $featureVM.ExitCode -ne 0) {
        throw "Failed to enable required WSL Windows features."
      }
      $update = Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--update") -IgnoreExitCode
      if ($update.ExitCode -ne 0) {
        Write-Info "wsl --update returned a non-zero exit code. Continuing."
      }
      if ((Test-RestartRequired $featureWSL) -or (Test-RestartRequired $featureVM) -or (Test-RestartRequired $install)) {
        throw "WSL2 components were enabled, but Windows must be restarted. Please reboot and run install.bat again."
      }
    } elseif (Test-RestartRequired $install) {
      throw "WSL2 installation started, but Windows must be restarted. Please reboot and run install.bat again."
    }
  }

  $defaultVersion = Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--set-default-version", "2") -IgnoreExitCode
  if ($defaultVersion.ExitCode -ne 0) {
    Write-Info "Failed to set the default WSL version to 2. Continuing with validation."
  }
}

function Get-WSLDistros {
  $result = Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("-l", "-q") -IgnoreExitCode
  if ($result.ExitCode -ne 0) {
    return @()
  }
  return @(
    $result.Output -split "`r?`n" |
      ForEach-Object { $_.Trim() } |
      Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
  )
}

function Get-WSLVerboseListLines {
  $result = Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("-l", "-v") -IgnoreExitCode
  if ($result.ExitCode -ne 0) {
    return @()
  }
  return @(
    $result.Output -split "`r?`n" |
      ForEach-Object { $_.Trim() } |
      Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
  )
}

function Find-UbuntuDistro {
  param([string]$PreferredDistro)

  $distros = Get-WSLDistros
  foreach ($name in $distros) {
    if ($name -eq $PreferredDistro) {
      return $name
    }
  }
  foreach ($name in $distros) {
    if ($name -match '^Ubuntu($|-)') {
      return $name
    }
  }
  return ""
}

function Test-DistroExists {
  param([string]$Name)

  if ([string]::IsNullOrWhiteSpace($Name)) {
    return $false
  }
  foreach ($entry in (Get-WSLDistros)) {
    if ($entry -eq $Name) {
      return $true
    }
  }
  return $false
}

function Get-DistroVersion {
  param([string]$Name)

  foreach ($line in (Get-WSLVerboseListLines)) {
    $text = $line.Trim()
    if ([string]::IsNullOrWhiteSpace($text)) {
      continue
    }
    if ($text.StartsWith("*")) {
      $text = $text.Substring(1).Trim()
    }
    if ($text -match "^(?<name>.+?)\s+\S+\s+(?<version>\d+)$") {
      if ($Matches["name"].Trim() -eq $Name) {
        return [int]$Matches["version"]
      }
    }
  }
  return 0
}

function Try-PrimeDistro {
  param([string]$Name)

  if ([string]::IsNullOrWhiteSpace($Name)) {
    return
  }
  Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("-d", $Name, "--", "sh", "-lc", "printf ready") -IgnoreExitCode | Out-Null
}

function Ensure-DistroVersion2 {
  param([string]$Name)

  $version = Get-DistroVersion -Name $Name
  if ($version -eq 0) {
    Try-PrimeDistro -Name $Name
    $version = Get-DistroVersion -Name $Name
  }
  if ($version -eq 2 -or $version -eq 0) {
    return
  }

  Write-Info "Switching distro $Name to WSL2."
  $setVersion = Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--set-version", $Name, "2") -IgnoreExitCode
  if ($setVersion.ExitCode -ne 0) {
    $message = $setVersion.Output.Trim()
    if ([string]::IsNullOrWhiteSpace($message)) {
      throw "Failed to switch distro $Name to WSL2."
    }
    throw "Failed to switch distro $Name to WSL2.`n$message"
  }
}

function Ensure-MirroredNetworking {
  $configPath = Join-Path $env:USERPROFILE ".wslconfig"
  $lines = New-Object System.Collections.Generic.List[string]
  if (Test-Path $configPath) {
    foreach ($line in Get-Content -Path $configPath) {
      $lines.Add([string]$line)
    }
  }

  $sectionStart = -1
  $sectionEnd = $lines.Count
  for ($i = 0; $i -lt $lines.Count; $i++) {
    if ($lines[$i] -match "^\s*\[wsl2\]\s*$") {
      $sectionStart = $i
      continue
    }
    if ($sectionStart -ge 0 -and $i -gt $sectionStart -and $lines[$i] -match "^\s*\[[^\]]+\]\s*$") {
      $sectionEnd = $i
      break
    }
  }

  $changed = $false
  if ($sectionStart -lt 0) {
    if ($lines.Count -gt 0 -and -not [string]::IsNullOrWhiteSpace($lines[$lines.Count - 1])) {
      $lines.Add("")
    }
    $lines.Add("[wsl2]")
    $lines.Add("networkingMode=mirrored")
    $changed = $true
  } else {
    $found = $false
    for ($i = $sectionStart + 1; $i -lt $sectionEnd; $i++) {
      if ($lines[$i] -match "^\s*networkingMode\s*=") {
        $found = $true
        if ($lines[$i].Trim() -ne "networkingMode=mirrored") {
          $lines[$i] = "networkingMode=mirrored"
          $changed = $true
        }
        break
      }
    }
    if (-not $found) {
      $lines.Insert($sectionStart + 1, "networkingMode=mirrored")
      $changed = $true
    }
  }

  if ($changed) {
    Write-Info "Updating .wslconfig to enable mirrored networking."
    Set-Content -Path $configPath -Value ($lines -join "`r`n") -Encoding Ascii
    Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--shutdown") -IgnoreExitCode | Out-Null
  } else {
    Write-Info ".wslconfig already has mirrored networking enabled."
  }
}

function Invoke-WSLCommand {
  param(
    [string]$Distro,
    [string[]]$CommandArgs,
    [switch]$AsRoot,
    [switch]$IgnoreExitCode
  )

  $args = @("-d", $Distro)
  if ($AsRoot) {
    $args += @("-u", "root")
  }
  $args += "--"
  $args += $CommandArgs
  return Invoke-NativeCommand -FilePath "wsl.exe" -Arguments $args -IgnoreExitCode:$IgnoreExitCode
}

function Convert-ToWSLShellLiteral {
  param([string]$Value)

  if ($null -eq $Value) {
    return "''"
  }
  $singleQuote = [string][char]39
  $replacement = $singleQuote + [char]34 + $singleQuote + [char]34 + $singleQuote
  return $singleQuote + $Value.Replace($singleQuote, $replacement) + $singleQuote
}

function Ensure-DirectoryPath {
  param([string]$Path)

  if ([string]::IsNullOrWhiteSpace($Path)) {
    throw "A directory path is required."
  }
  New-Item -ItemType Directory -Force -Path $Path | Out-Null
}

function Remove-PathWithRetry {
  param([string]$Path)

  if ([string]::IsNullOrWhiteSpace($Path) -or -not (Test-Path $Path)) {
    return
  }

  for ($attempt = 1; $attempt -le 8; $attempt++) {
    try {
      Remove-Item -Recurse -Force $Path
      return
    } catch {
      if ($attempt -eq 8) {
        throw
      }
      Start-Sleep -Seconds 2
    }
  }
}

function Get-ManagedWSLBaseDir {
  return "C:\WSL"
}

function Get-ManagedRootfsCachePath {
  $cacheDir = Join-Path (Get-ManagedWSLBaseDir) "cache"
  Ensure-DirectoryPath -Path $cacheDir
  return Join-Path $cacheDir ("ubuntu-" + $RootfsRelease + "-" + (Get-RootfsArchTag) + ".wsl")
}

function Get-ManagedDistroInstallDir {
  param([string]$Name)

  $baseDir = Get-ManagedWSLBaseDir
  Ensure-DirectoryPath -Path $baseDir
  return Join-Path $baseDir $Name
}

function Get-RootfsDownloadCandidates {
  $archTag = Get-RootfsArchTag
  return @(
    "https://cdimages.ubuntu.com/ubuntu-wsl/$RootfsRelease/daily-live/current/$RootfsRelease-wsl-$archTag.wsl",
    "https://cloud-images.ubuntu.com/wsl/releases/noble/current/ubuntu-noble-wsl-$archTag-wsl.rootfs.tar.gz"
  )
}

function Ensure-RootfsArchive {
  $archivePath = Get-ManagedRootfsCachePath
  if (Test-Path $archivePath) {
    Write-Info "Using cached Ubuntu rootfs: $archivePath"
    return $archivePath
  }

  $errors = New-Object System.Collections.Generic.List[string]
  foreach ($url in (Get-RootfsDownloadCandidates)) {
    try {
      Write-Info "Downloading Ubuntu rootfs from $url"
      if ($PSVersionTable.PSVersion.Major -ge 6) {
        Invoke-WebRequest -Uri $url -OutFile $archivePath
      } else {
        Invoke-WebRequest -Uri $url -OutFile $archivePath -UseBasicParsing
      }
      if ((Get-Item $archivePath).Length -gt 0) {
        return $archivePath
      }
      $errors.Add("Downloaded empty file from $url")
    } catch {
      $errors.Add(($_.Exception.Message + " @ " + $url))
    }
    Remove-Item -Force -ErrorAction SilentlyContinue $archivePath
  }

  throw ("Failed to download an Ubuntu WSL rootfs archive.`n" + ($errors -join "`n"))
}

function Reset-ManagedDistroState {
  param(
    [string]$Name,
    [string]$InstallDir
  )

  if (-not [string]::IsNullOrWhiteSpace($Name)) {
    Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--terminate", $Name) -IgnoreExitCode | Out-Null
    Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--unregister", $Name) -IgnoreExitCode | Out-Null
  }
  Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--shutdown") -IgnoreExitCode | Out-Null
  if (-not [string]::IsNullOrWhiteSpace($InstallDir) -and (Test-Path $InstallDir)) {
    Remove-PathWithRetry -Path $InstallDir
  }
}

function Ensure-ManagedUbuntuDistro {
  param([string]$Name)

  $installDir = Get-ManagedDistroInstallDir -Name $Name
  if (Test-DistroExists -Name $Name) {
    $probe = Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("-d", $Name, "-u", "root", "--", "sh", "-lc", "printf ready") -IgnoreExitCode
    if ($probe.ExitCode -eq 0 -and $probe.Output.Trim() -eq "ready") {
      Write-Info "Managed distro already exists: $Name"
      return
    }
    Write-Info "Existing managed distro is not healthy. Recreating $Name."
    Reset-ManagedDistroState -Name $Name -InstallDir $installDir
  }

  $archivePath = Ensure-RootfsArchive
  if (Test-Path $installDir) {
    Reset-ManagedDistroState -Name $Name -InstallDir $installDir
  }
  Ensure-DirectoryPath -Path $installDir

  Write-Info "Importing Ubuntu rootfs as distro $Name"
  $import = Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--import", $Name, $installDir, $archivePath, "--version", "2") -IgnoreExitCode
  if ($import.ExitCode -eq 0) {
    return
  }

  $firstMessage = $import.Output.Trim()
  Write-Info "Initial import failed. Retrying once after cleanup."
  Reset-ManagedDistroState -Name $Name -InstallDir $installDir
  Ensure-DirectoryPath -Path $installDir

  $importRetry = Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--import", $Name, $installDir, $archivePath, "--version", "2") -IgnoreExitCode
  if ($importRetry.ExitCode -eq 0) {
    return
  }

  $message = $importRetry.Output.Trim()
  Reset-ManagedDistroState -Name $Name -InstallDir $installDir
  if ([string]::IsNullOrWhiteSpace($message)) {
    $message = $firstMessage
  }
  if ([string]::IsNullOrWhiteSpace($message)) {
    throw "Failed to import WSL distro $Name."
  }
  throw "Failed to import WSL distro $Name.`n$message"
}

function Wait-ForDistroReady {
  param([string]$Name)

  for ($attempt = 1; $attempt -le 15; $attempt++) {
    $result = Invoke-WSLCommand -Distro $Name -AsRoot -CommandArgs @("sh", "-lc", "printf ready") -IgnoreExitCode
    if ($result.ExitCode -eq 0 -and $result.Output.Trim() -eq "ready") {
      return
    }
    Start-Sleep -Seconds 2
  }
  throw "The Ubuntu distro is not ready yet. Please rerun install.bat."
}

function Ensure-WSLPackages {
  param([string]$Name)

  $missing = Invoke-WSLCommand -Distro $Name -AsRoot -CommandArgs @(
    "sh",
    "-lc",
    'missing=""; for name in git npm python3 bwrap; do command -v "$name" >/dev/null 2>&1 || missing="$missing $name"; done; printf "%s" "$missing"'
  )
  $packages = @(
    $missing.Output.Trim() -split "\s+" |
      Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
  )
  if ($packages.Count -eq 0) {
    Write-Info "git, npm, python3, and bubblewrap are already installed."
    return
  }

  Write-Info ("Installing WSL packages: " + ($packages -join ", "))
  $installArgs = @(
    "sh",
    "-lc",
    'export DEBIAN_FRONTEND=noninteractive; apt-get update && apt-get install -y "$@"',
    "sh"
  ) + $packages
  Invoke-WSLCommand -Distro $Name -AsRoot -CommandArgs $installArgs | Out-Null
}

function Convert-WindowsPathToWSL {
  param([string]$WindowsPath)

  $normalized = [string]$WindowsPath
  if ([string]::IsNullOrWhiteSpace($normalized)) {
    throw "A Windows path is required."
  }
  $normalized = $normalized.Trim()

  if ($normalized.StartsWith("/")) {
    return $normalized
  }
  if ($normalized -match '^(?<drive>[A-Za-z]):[\\/]*(?<rest>.*)$') {
    $drive = $Matches["drive"].ToLowerInvariant()
    $rest = [string]$Matches["rest"]
    $rest = $rest -replace '[\\/]+', '/'
    if ([string]::IsNullOrWhiteSpace($rest)) {
      return "/mnt/$drive"
    }
    return "/mnt/$drive/$rest"
  }
  throw "Unsupported Windows path format for WSL conversion: $WindowsPath"
}

function Get-WSLUserHome {
  param([string]$Name)

  $userLiteral = Convert-ToWSLShellLiteral $ManagedUserName
  $result = Invoke-WSLCommand -Distro $Name -AsRoot -CommandArgs @(
    "sh",
    "-lc",
    "getent passwd $userLiteral | cut -d: -f6"
  )
  return $result.Output.Trim()
}

function Join-WSLPath {
  param(
    [string]$BasePath,
    [string]$Leaf
  )

  if ([string]::IsNullOrWhiteSpace($BasePath)) {
    return $Leaf
  }
  if ($BasePath.EndsWith("/")) {
    return $BasePath + $Leaf
  }
  return $BasePath + "/" + $Leaf
}

function Assert-WSLArchitecture {
  param([string]$Name)

  $arch = Invoke-WSLCommand -Distro $Name -CommandArgs @("uname", "-m")
  $resolved = $arch.Output.Trim()
  $expected = Get-ExpectedWSLArchitecture
  if ($resolved -ne $expected) {
    throw "The Ubuntu architecture is $resolved, but this Windows host expects $expected."
  }
}

function Ensure-WSLBootstrapUser {
  param([string]$Name)

  $userLiteral = Convert-ToWSLShellLiteral $ManagedUserName
  $scriptBody = @'
set -eu
user_name=__USER_LITERAL__
if ! id -u "$user_name" >/dev/null 2>&1; then
  useradd -m -s /bin/bash "$user_name"
fi
if getent group sudo >/dev/null 2>&1; then
  usermod -aG sudo "$user_name"
fi
if command -v sudo >/dev/null 2>&1; then
  install -d -m 755 /etc/sudoers.d
  printf '%s ALL=(ALL) NOPASSWD:ALL\n' "$user_name" > "/etc/sudoers.d/99-$user_name"
  chmod 440 "/etc/sudoers.d/99-$user_name"
fi
cat > /etc/wsl.conf <<EOF
[user]
default=$user_name
EOF
'@
  $scriptBody = $scriptBody.Replace("__USER_LITERAL__", $userLiteral)
  Invoke-WSLCommand -Distro $Name -AsRoot -CommandArgs @("sh", "-lc", $scriptBody) | Out-Null
  Invoke-NativeCommand -FilePath "wsl.exe" -Arguments @("--terminate", $Name) -IgnoreExitCode | Out-Null
}

function Resolve-ReleaseSourceRoot {
  $appDir = Join-Path $PSScriptRoot "app"
  if (Test-Path $appDir) {
    $items = Get-ChildItem -Path $appDir -Force -ErrorAction SilentlyContinue
    if ($items.Count -gt 0) {
      return $appDir
    }
  }
  return $PSScriptRoot
}

function Assert-ReleasePackageLayout {
  param([string]$SourceRoot)

  $requiredPaths = @(
    "integration",
    "config",
    "site"
  )
  foreach ($relativePath in $requiredPaths) {
    $fullPath = Join-Path $SourceRoot $relativePath
    if (-not (Test-Path $fullPath)) {
      throw "Release payload is incomplete. Missing: $fullPath"
    }
  }
}

function Copy-ReleasePackage {
  param(
    [string]$Name,
    [string]$SourceWindowsDir,
    [string]$TargetDir
  )

  $sourceWSLDir = Convert-WindowsPathToWSL -WindowsPath $SourceWindowsDir
  $targetDirLiteral = Convert-ToWSLShellLiteral $TargetDir
  $sourceDirLiteral = Convert-ToWSLShellLiteral $sourceWSLDir
  $userLiteral = Convert-ToWSLShellLiteral $ManagedUserName
  $scriptBody = @'
set -e
target_dir=__TARGET_DIR_LITERAL__
source_dir=__SOURCE_DIR_LITERAL__
user_name=__USER_LITERAL__
rm -rf "$target_dir"
mkdir -p "$target_dir"
cp -R "$source_dir/." "$target_dir/"
rm -f "$target_dir/install.bat" "$target_dir/install.ps1" "$target_dir/start.bat" "$target_dir/install.log" "$target_dir/install-wsl2.ps1" "$target_dir/install-wsl2.cmd"
chmod 755 "$target_dir/integration"
if [ -d "$target_dir/helpers" ]; then
  find "$target_dir/helpers" -type f -name 'CLI_SANDBOX' -exec chmod 755 {} +
fi
if [ -d "$target_dir/plugins" ]; then
  find "$target_dir/plugins" -mindepth 1 -maxdepth 1 -type f -exec chmod 755 {} +
fi
chown -R "$user_name:$user_name" "$target_dir"
'@
  $scriptBody = $scriptBody.Replace("__TARGET_DIR_LITERAL__", $targetDirLiteral)
  $scriptBody = $scriptBody.Replace("__SOURCE_DIR_LITERAL__", $sourceDirLiteral)
  $scriptBody = $scriptBody.Replace("__USER_LITERAL__", $userLiteral)
  Invoke-WSLCommand -Distro $Name -AsRoot -CommandArgs @("sh", "-lc", $scriptBody) | Out-Null
}

function Install-IntegrationShim {
  param(
    [string]$Name,
    [string]$TargetDirLeaf
  )

  $targetDirLiteral = Convert-ToWSLShellLiteral $TargetDirLeaf
  $userLiteral = Convert-ToWSLShellLiteral $ManagedUserName
  $scriptBody = @'
set -e
target_dir_name=__TARGET_DIR_LITERAL__
user_name=__USER_LITERAL__
home_dir="$(getent passwd "$user_name" | cut -d: -f6)"
cat > "$home_dir/.integration" <<EOF
#!/usr/bin/env sh
set -eu
BASE_DIR="$HOME/$target_dir_name"
if [ "${1:-}" = "--start" ]; then
  shift
  cd "$BASE_DIR"
  exec "$BASE_DIR/integration" start "$@"
fi
if [ "$#" -eq 0 ]; then
  cd "$BASE_DIR"
  exec "$BASE_DIR/integration" start
fi
exec "$BASE_DIR/integration" "$@"
EOF
cat > "$home_dir/start-deepright.sh" <<EOF
#!/usr/bin/env sh
set -eu
exec "$HOME/.integration" --start "$@"
EOF
chmod 755 "$home_dir/.integration" "$home_dir/start-deepright.sh"
chown "$user_name:$user_name" "$home_dir/.integration" "$home_dir/start-deepright.sh"
'@
  $scriptBody = $scriptBody.Replace("__TARGET_DIR_LITERAL__", $targetDirLiteral)
  $scriptBody = $scriptBody.Replace("__USER_LITERAL__", $userLiteral)
  Invoke-WSLCommand -Distro $Name -AsRoot -CommandArgs @("sh", "-lc", $scriptBody) | Out-Null
}

function Start-Integration {
  param([string]$Name)

  Write-Info "Starting ~/.integration --start"
  & wsl.exe -d $Name -u $ManagedUserName -- sh -lc 'exec "$HOME/.integration" --start'
  exit $LASTEXITCODE
}

Write-LogLine ("START installer version " + $InstallerVersion)
Write-Info ("Installer version: " + $InstallerVersion)

Ensure-Administrator
Assert-WSL2Supported
Ensure-WSLInstalled
Ensure-MirroredNetworking
Ensure-ManagedUbuntuDistro -Name $DistroName
Ensure-DistroVersion2 -Name $DistroName
Wait-ForDistroReady -Name $DistroName
Ensure-WSLBootstrapUser -Name $DistroName
Wait-ForDistroReady -Name $DistroName
Ensure-WSLPackages -Name $DistroName
Assert-WSLArchitecture -Name $DistroName

$packageRoot = (Resolve-Path -Path (Resolve-ReleaseSourceRoot)).Path
Assert-ReleasePackageLayout -SourceRoot $packageRoot
Write-Info ("Using release payload: " + $packageRoot)

$userHome = Get-WSLUserHome -Name $DistroName
if ([string]::IsNullOrWhiteSpace($userHome)) {
  throw "Failed to resolve the WSL user home directory."
}
$targetDir = Join-WSLPath -BasePath $userHome -Leaf $TargetDirName

Write-Info ("Copying release files to " + $targetDir)
Copy-ReleasePackage -Name $DistroName -SourceWindowsDir $packageRoot -TargetDir $targetDir
Install-IntegrationShim -Name $DistroName -TargetDirLeaf $TargetDirName

if ($SkipLaunch) {
  Write-Info 'Installation finished. Launch skipped. You can run "~/.integration --start" inside WSL.'
  exit 0
}

Start-Integration -Name $DistroName
