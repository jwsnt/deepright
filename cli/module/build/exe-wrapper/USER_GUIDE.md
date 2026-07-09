# Windows Single-File Installer Guide

## Build

From `cli/module/`:

```sh
./build-windows-exe.sh
```

Optional target selection:

```sh
./build-windows-exe.sh x86
./build-windows-exe.sh arm
```

Repository target naming follows the existing release layout:

- `x86` = Windows `amd64`
- `arm` = Windows `arm64`

## Output

Artifacts are written to:

- `release/windows/x86/DeepRight-windows-x86-installer.exe`
- `release/windows/x86/DeepRight-windows-x86-uninstaller.exe`
- `release/windows/arm/DeepRight-windows-arm-installer.exe`
- `release/windows/arm/DeepRight-windows-arm-uninstaller.exe`

If `shasum` is available, a matching `.sha256` file is also generated.

## Build options

- `DEEPRIGHT_SKIP_LINUX_BUILD=1`
  - reuse the existing `release/linux/<target>` payload instead of calling `build.sh linux`
- `DEEPRIGHT_KEEP_WINDOWS_EXE_TMP=1`
  - keep the temporary staging directory for debugging

Example:

```sh
DEEPRIGHT_SKIP_LINUX_BUILD=1 ./build-windows-exe.sh x86
```

## Runtime behavior on Windows

Double-click the generated `.exe`:

- installer:
  - extracts the embedded payload to a stable cache directory
  - if the existing local sentinel and `deepright` WSL distro are healthy, it runs `start.bat`
  - otherwise it runs `install.bat`
  - `install.bat` continues into the existing WSL2 install/start flow
- uninstaller:
  - extracts a minimal payload to a separate cache namespace
  - runs `uninstall.bat`, which elevates and calls `uninstall.ps1`
  - `uninstall.ps1` removes shortcuts, Windows cache/sentinel data, unregisters the `deepright` WSL distro, and deletes `C:\WSL\deepright`

Because `install.ps1` is already idempotent, running the same `.exe` again does not rebuild the whole environment when the local sentinel and distro are healthy.

## Optional CLI flags for the generated `.exe`

- installer:
  - `--auto`
  - `--install`
  - `--start`
  - `--extract-only`
- uninstaller:
  - `--uninstall`
  - `--extract-only`
