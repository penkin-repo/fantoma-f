# MASTER PROMPT — Fantoma F

> Updated: 2026-11-25 | Version: 1.4.1

## Project Name
**Fantoma F** — Lightweight temporary font manager for Windows.

## Stack
- **Backend**: Go 1.24 + Wails v2.11
- **Frontend**: React 18 (Vite 5) + CSS Modules + `@tanstack/react-virtual` + `lucide-react`
- **Package manager**: `pnpm` (mandatory everywhere, no npm)

---

## What the App Does
User manages one or more folders containing `.ttf` / `.otf` fonts (scanned recursively). The application groups fonts by family (via OpenType `sfnt` metadata), classifies them by type/weight/italic (via `OS/2`, PANOSE, and `post.isFixedPitch`), previews them live via local Wails AssetServer HTTP streaming, and activates/deactivates individual styles, families, or entire folders **temporarily** in Windows without admin elevation. On exit or Windows shutdown, staged fonts are cleanly unloaded from Windows while preserving `activeFonts` in `config.json` so they restore automatically on the next launch (configurable in Settings).

---

## Architecture

### Go Backend
- `main.go` — Wails entry point, SingleInstanceLock, custom `/font?path=...` AssetServer handler, initial theme detection, native tray loop, `SIGTERM`/`SIGINT` & tray Quit cleanup.
- `app.go` — `App` struct, folder management, parallel family scanning + `familyCache`, `fsnotify` recursive watcher, activation/deactivation methods, OS/2 + `post` classification, `TrimMemory`.
- `font_windows.go` — Fast FontBase-style per-user activation via HKCU registry (`Software\Microsoft\Windows NT\CurrentVersion\Fonts`) + staged copies in `%LOCALAPPDATA%\Microsoft\Windows\Fonts\fantomaf` with collision-free `<fnv32>_<basename>` names + single async `PostMessageW(HWND_BROADCAST, WM_FONTCHANGE)`. Avoids GDI `AddFontResourceW`/`RemoveFontResourceW` completely. Locked files (e.g. held by Illustrator) are atomically evicted on the same NTFS volume to `%LOCALAPPDATA%\FantomaF\trash` and swept on next startup.
- `shutdown_windows.go` — Hidden Win32 window on a locked OS thread intercepting `WM_QUERYENDSESSION` / `WM_ENDSESSION` to unload staged fonts during Windows shutdown/restart without wiping `config.json`.
- `tray_windows.go` — Native Win32 system tray icon (handles `WM_TASKBARCREATED` after Explorer restarts, 60s heartbeat, Show / Deactivate All / Quit menu).
- `autostart_windows.go` — `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` (`FantomaF` with `--minimized` and symlink resolution).
- `config.go` — Portable atomic JSON persistence (`config.json` next to `FantomaF.exe`).
- `version.go` — Single source of truth for `AppVersion` (`1.4.1`) and `GetAppVersion()`.

### Exported Wails Methods
- Folders: `AddFolder()`, `RemoveFolder(id)`, `RefreshFolder(id)`, `GetFolders()`, `GetFamiliesInFolder(id)`
- Activation: `ActivateFont(path)`, `DeactivateFont(path)`, `ActivateFamily(styles)`, `DeactivateFamily(styles)`, `ActivateFolder(id)`, `DeactivateFolder(id)`, `DeactivateAll()`
- Classification & Cache: `SetFontCategory(familyName, category)`, `ClearCache()`, `TrimMemory()`
- Settings & Info: `ToggleAutostart(enable)`, `IsAutostartEnabled()`, `GetTheme()`, `SetTheme(theme)`, `GetRestoreOnStartup()`, `SetRestoreOnStartup(enable)`, `GetAppVersion()`

### React Frontend (`frontend/src/`)
- `App.jsx` — Root layout, `@tanstack/react-virtual` list, `CategoryPicker`, `Toast`, tray `TrimMemory` hook.
- `hooks/useFonts.js` — Folders, families, optimistic toggles, filters, settings, Wails `folders:changed` & `fonts:changed` event subscriptions.
- `utils/fontLoader.js` — Lazy deduplicated `FontFace` loader with FNV-1a hashing (Cyrillic/Unicode-safe) streaming from `http://wails.localhost/font`.
- `components/`:
  - `Sidebar.jsx` — Search, folder list with active/partial/inactive dots, bulk ON/OFF, refresh/remove, Settings & quick theme toggle.
  - `TopBar.jsx` — Custom preview text, font size slider (12–120px), Active-only filter, family counter.
  - `FilterBar.jsx` — Category, weight, and italic filter chips.
  - `FontRow.jsx` — Family row with live preview, category badge, activation toggle, and expandable individual style drawer.
  - `CategoryPicker.jsx` — Manual category override popup.
  - `SettingsView.jsx` — Theme selector (`Obsidian Dark` / `Warm Paper`), Launch on Startup, Restore Active Fonts on Startup, Clear Cache, About.
  - `Toast.jsx` — Auto-dismissing status notifications.

---

## Key Technical Decisions

| Topic | Decision |
|---|---|
| Elevation | None — per-user HKCU registry + `%LOCALAPPDATA%\Microsoft\Windows\Fonts\fantomaf` |
| Activation engine | Pure HKCU registry + copy + single async `PostMessageW(WM_FONTCHANGE)` (NO GDI `AddFontResourceW`/`RemoveFontResourceW` — avoids ~100–250ms/font kernel mutex) |
| Staged filenames | `<8-hex-fnv32>_<basename>` prevents collisions across folders; copy-only (no symlinks) survives hard power loss |
| Locked file eviction | If Illustrator holds a font open on deactivation, `os.Rename` moves it on the same NTFS volume to `%LOCALAPPDATA%\FantomaF\trash` and `cleanupAllOurFonts()` sweeps it on next start |
| Shutdown & Reboot | `WM_ENDSESSION` + tray Quit call `deactivateSystemFontsOnly()` so Windows stays clean while `config.json` preserves `activeFonts` for next launch (`restoreOnStartup`, default `true`) |
| Folder scan | Recursive scan (`.ttf`/`.otf` only, WOFF/WOFF2 excluded) + `fsnotify` live directory watcher |
| Font preview | Local AssetServer `/font?path=...` + `FontFace` API (replaces base64) |
| Themes & Icons | CSS custom properties (`Obsidian Dark` default + `Warm Paper` light), synced with Windows titlebar (`WindowSetDarkTheme`/`WindowSetLightTheme`); `lucide-react` icons |
| Config persistence | Portable `config.json` next to `FantomaF.exe` (atomic `.tmp` + rename) |
| System Tray | Native Win32 tray; `✕` hides to tray and triggers `TrimMemory()`; Quit unloads fonts and exits |
| Platform | **Windows ONLY** (`windows/amd64`) |

---

## Project Structure
```
litefont/
├── main.go                    # Wails entry point, /font handler, tray, signals
├── app.go                     # Core App methods, scanner, fsnotify, OS/2 parser
├── font_windows.go            # HKCU registry + staged copy + locked-file eviction
├── shutdown_windows.go        # WM_QUERYENDSESSION / WM_ENDSESSION listener
├── tray_windows.go            # Native Win32 system tray icon
├── autostart_windows.go       # HKCU Run autostart
├── config.go                  # Atomic JSON config persistence
├── version.go                 # Single source of truth for AppVersion
├── wails.json                 # Wails + pnpm build config
├── go.mod / go.sum
├── README.md
├── MASTER_PROMPT.md
├── PROJECT_LOG.md
├── PROJECT_LOG_HOW.md
├── build/
│   ├── appicon.png
│   ├── bin/FantomaF.exe       # Built portable binary (+ config.json)
│   └── windows/               # icon.ico, info.json, manifest, installer
└── frontend/
    ├── src/
    │   ├── App.jsx
    │   ├── main.jsx
    │   ├── style.css
    │   ├── hooks/useFonts.js
    │   ├── utils/fontLoader.js
    │   └── components/
    │       ├── Sidebar.jsx / .module.css
    │       ├── TopBar.jsx / .module.css
    │       ├── FilterBar.jsx / .module.css
    │       ├── FontRow.jsx / .module.css
    │       ├── CategoryPicker.jsx / .module.css
    │       ├── SettingsView.jsx / .module.css
    │       └── Toast.jsx / .module.css
    ├── index.html
    ├── vite.config.js
    └── package.json
```

---

## Build Commands
```powershell
cd litefont

# Dev (hot reload)
wails dev

# Production portable .exe
wails build

# Run minimized (autostart mode)
./build/bin/FantomaF.exe --minimized
```
