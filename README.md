# BadRename

A small batch rename tool. Add files, pick a rule, see every new name
before anything touches the disk. Built with Go and MyGo native UI.

## Rules

Four rule pages, one active at a time:

- **Whole** — build the name from a template. `*` inserts the original
  name, `#` inserts a serial number or letter (`a, b, c...`). Start,
  step, digit width, zero padding, and extension are configurable.
- **Replace** — replace every occurrence of a string in the name.
- **Add/Delete** — prefix, suffix, insert at a position, delete a
  string, or delete a character range. Acts on the name without the
  extension.
- **Regex** — replace a regular expression match; `$1` and `${name}`
  group references work in the replacement. The expression is checked
  as you type.

Start and step accept signed 64-bit integers; serial calculations remain exact
even when a later number exceeds that range. Delete counts must be non-negative.
The application blocks renaming and shows a message for invalid values.

Every page can lowercase or uppercase the name, the extension, or both.
On a name conflict you choose: ask, overwrite, skip, or auto-rename
(appends `(2)`, `(3)`, ...). Generated names must be single file names, not
paths. Existing directory entries, including dangling symbolic links, count as
conflicts. Unless you explicitly choose overwrite, the application creates the
new name without replacing an existing entry. This uses hard links: on file
systems or with file types that do not support them, the rename fails safely
rather than falling back to a potentially destructive overwrite. If creating
the new name succeeds but removing the old one fails, both names are retained
and the operation is reported as failed.

## Build

Requires Go 1.27.1 or later. The MyGo runtime and CLI are pinned together
in `go.mod` at `v0.5.6-0.20261017195139-b7f6946547d9`. The native UI
does not require Node.js or a webview.

```powershell
go test ./...
go tool mygo vet ./internal/ui
go build -trimpath -tags mygo_noinspector -ldflags="-s -w -H=windowsgui -X github.com/egoist/mygo.production=1 -X github.com/egoist/mygo.packageName=BadRename -X github.com/egoist/mygo.packageVersion=0.1.0 -X github.com/egoist/mygo.packageIdentifier=com.badrename.app" -o dist/windows-amd64/BadRename.exe .
```

The Windows build produces one standalone executable; no installer is needed.
The release workflow also produces individual Linux and macOS executables.
Linux requires GTK 3, Windows requires Windows 10 or 11, and macOS requires
macOS 12 or later. Linux and macOS downloads may need `chmod +x` before use.
The macOS executable is unsigned; distribution to other Macs requires an
appropriate signature and notarization.

Windows icon and DPI manifest resources are checked in as
`internal/winres/rsrc_windows_amd64.syso`. After changing the icon or
`build/windows/badrename.exe.manifest`, regenerate them with:

```sh
go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico build/windows/icon.ico -manifest build/windows/badrename.exe.manifest -o internal/winres/rsrc_windows_amd64.syso
```

## License

MIT. See `LICENSE`.
