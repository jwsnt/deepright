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
- `release/windows/arm/DeepRight-windows-arm-installer.exe`

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

- it extracts the embedded payload to a stable cache directory
- if the existing local sentinel and `deepright` WSL distro are healthy, it runs `start.bat`
- otherwise it runs `install.bat`
- `install.bat` continues into the existing WSL2 install/start flow

Because `install.ps1` is already idempotent, running the same `.exe` again does not rebuild the whole environment when the local sentinel and distro are healthy.

## Optional CLI flags for the generated `.exe`

- `--auto`
  - choose `start.bat` when the local install is healthy, otherwise `install.bat`
- `--install`
  - extract and run `install.bat`
- `--start`
  - extract and run `start.bat`
- `--extract-only`
  - only extract the bundle and print the extraction directory
