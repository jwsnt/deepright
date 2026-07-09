# Windows Single-File Installer Design

## Goal

`build/exe-wrapper` provides the template used by `../build-windows-exe.sh` to turn an existing Linux/WSL release payload into a single Windows `.exe`.

The generated `.exe` is meant to be a self-extracting launcher:

- first run: extract payload, run `install.bat`, complete WSL2 installation, then start DeepRight
- later runs: extract check is reused by payload hash, and the launcher prefers `start.bat` when the local sentinel and WSL distro are healthy
- a separate uninstaller build extracts a minimal payload and runs `uninstall.bat`

## Why a new build path

The current `../build.sh` already produces the Windows-facing WSL payload directory under `release/linux/<target>/`, including:

- `install.bat`
- `install.ps1`
- `start.bat`
- `DeepRight.ico`
- Linux release files such as `integration`, `config/`, `site/`, `plugins/`, `helpers/`

This wrapper keeps that behavior unchanged and adds a separate packaging layer on top, instead of changing the existing release contract.

## Packaging flow

`../build-windows-exe.sh` does the following:

1. ensure `release/linux/x86` and/or `release/linux/arm` exists
2. copy the selected target payload into a temporary staging directory
3. zip the staged payload
4. copy `main.go.tmpl` into a temporary Go package as `main.go`
5. generate `config.go` with launcher-specific behavior
6. place `payload.zip` beside it
7. cross-compile the wrapper as a Windows `.exe`

## Runtime flow

When the generated `.exe` runs on Windows:

1. compute SHA-256 of embedded `payload.zip`
2. choose a stable extraction directory under the user cache directory
3. reuse the directory if the payload hash marker matches and required files exist
4. otherwise remove stale content and extract the payload again
5. launcher-specific default behavior:
   - installer: run `start.bat` when `C:\ProgramData\deepright\.deepright_installed` exists and `wsl.exe -d deepright -- echo ok` succeeds, otherwise run `install.bat`
   - uninstaller: run `uninstall.bat`

`install.bat` then keeps using the existing `install.ps1` logic, so WSL installation remains authoritative in one place. `uninstall.bat` similarly delegates to `uninstall.ps1` so the full Windows/WSL cleanup stays in one script.

## Design constraints

- no changes to the existing `../build.sh` release behavior
- no new runtime dependency on the target Windows machine
- payload extraction must defend against zip path traversal
- repeated runs must stay idempotent by reusing the extracted bundle and the existing sentinel-based install logic
- repeated runs should avoid unnecessary UAC prompts when the local installation is already healthy

## Current limitation

The generated `.exe` is a real Windows executable, but the build flow does not currently embed a PE icon resource into the installer executable itself.

What still uses the product icon today:

- the extracted payload includes `DeepRight.ico`
- installed desktop/start-menu shortcuts created by `install.ps1`

If needed later, a separate PE resource step can be added without changing this wrapper contract.
