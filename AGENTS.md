# Project constraints

- BadRename is a standalone application. Build and release a single executable per target platform; do not generate or publish installers, installation scripts, disk images, or bundled archives.
- Keep the UI native using MyGo `ui` controls; do not introduce a webview.
- Windows builds must retain the application icon and DPI-aware manifest.
- For local WSL development and Windows testing, see `wsl.md` if present (it is local and gitignored).
