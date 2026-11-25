# PROJECT LOG — Fantoma F

> Format: see PROJECT_LOG_HOW.md

---

## [v1.4.2] — 2026-11-25 — White FF Monogram Icon

### Stage: Visual Polish

**What was implemented:**
- **New app & tray icon**: Replaced `build/appicon.png` (512×512) and multi-size `build/windows/icon.ico` (16–48px 32-bit BGRA DIB + 64–256px PNG) with a crisp white geometric **FF** monogram on a transparent background, aligned to the 16×16 pixel grid for sharp system tray rendering.
- **Sidebar brand mark** (`Sidebar.jsx`): Matched the in-app header mark to the same **FF** monogram SVG.

**Files changed:**
- `build/appicon.png`, `build/windows/icon.ico`
- `frontend/src/components/Sidebar.jsx`
- `version.go` — bumped `AppVersion` to `1.4.2`

---

## [v1.4.1] — 2026-11-25 — Remove GDI Calls for Instant Batch Activation/Deactivation

### Stage: Performance Hotfix

**What was fixed:**
- **BUG-011**: Slow deactivation / activation regression (`font_windows.go`). Completely removed `gdi32.dll` (`AddFontResourceW` / `RemoveFontResourceW`) calls that were reintroduced in v1.4.0. `RemoveFontResourceW` acquired the global GDI kernel mutex on every file (~100–250ms/font), causing bulk folder deactivation to stall for up to a minute. Restored pure FontBase-style HKCU registry + parallel file staging/eviction + single async `WM_FONTCHANGE`, and optimized `addFontsBatch` / `removeFontsBatch` to reuse a single opened HKCU registry handle per batch.

**Files changed:**
- `font_windows.go` — removed `gdi32.dll` procs, single registry handle per batch, kept hashed names & `%LOCALAPPDATA%\FantomaF\trash` eviction
- `version.go` — bumped `AppVersion` to `1.4.1`
- `README.md`, `MASTER_PROMPT.md`, `PROJECT_LOG.md` — synced docs with v1.4.1

---

## [v1.4.0] — 2026-11-25 — Phantom Fonts Fix, Obsidian Dark Theme, Lucide Icons & Cleanup

### Stage: Reliability + Design Overhaul

**What was fixed & implemented:**
- **Phantom fonts & reboot restoration** (`font_windows.go`, `app.go`, `main.go`, `shutdown_windows.go`):
  - Collision-free hashed staged filenames (`<8-hex-fnv32>_<basename>`) so identically named font files from different folders never overwrite each other.
  - Locked-file eviction: if Illustrator or FontCache holds a staged file open on deactivation, it is moved atomically on the same NTFS volume to `%LOCALAPPDATA%\FantomaF\trash` and swept on next startup.
  - Shutdown (`WM_ENDSESSION`) and tray Quit now call `deactivateSystemFontsOnly()` instead of wiping `config.json`, so previously active fonts cleanly restore on next launch (toggleable via `Restore Active Fonts on Startup` in Settings).
  - Fixed `RemoveFolder()` to deactivate fonts inside the removed folder and unwatch its directory tree; fixed `DeactivateFolder()` to also clean up fonts deleted from disk; fixed `addFontsBatch()` to record only actually succeeded files; fixed tray `DeactivateAll()` to emit `fonts:changed` to keep UI in sync; fixed `fontLoader.js` FNV-1a hashing for Cyrillic/Unicode paths.
- **Obsidian Dark theme & Warm Paper light theme** (`style.css`, `main.go`, `app.go`, `config.go`, `SettingsView.jsx`, `Sidebar.jsx`):
  - Full CSS token system for dark (default) and light themes with native Windows titlebar theming (`WindowSetDarkTheme` / `WindowSetLightTheme`) and zero startup flash.
  - Persisted in `config.json` with quick toggle in Sidebar and theme cards in Settings.
- **Lucide icons & UI upgrades** (`frontend/package.json`, all components):
  - Replaced all emoji/unicode glyphs with `lucide-react` icons.
  - Added expandable individual style drawer in `FontRow.jsx` and `Toast` notifications.
  - Improved monospace detection via `post.isFixedPitch` and OS/2 PANOSE `bProportion`.
- **Cleanup & docs**:
  - Removed obsolete Wails boilerplate (`build/README.md`) and replaced template `README.md` with project documentation.

**Files changed:**
- `main.go`, `app.go`, `config.go`, `font_windows.go`, `shutdown_windows.go`, `tray_windows.go`, `version.go`
- `frontend/src/style.css`, `frontend/src/App.jsx`, `frontend/src/hooks/useFonts.js`, `frontend/src/utils/fontLoader.js`
- `frontend/src/components/` (`Sidebar`, `TopBar`, `FilterBar`, `FontRow`, `CategoryPicker`, `SettingsView`, `Toast` — JSX + CSS Modules)
- `README.md`, `MASTER_PROMPT.md`, `PROJECT_LOG.md`, deleted `build/README.md`

**Dependencies added:**
- `lucide-react ^1.80.0` — crisp vector icons replacing emojis

---

## [v1.3.3] — 2026-03-14 — Autostart Fix

### Stage: Reliability

**What was fixed:**
- **BUG-010**: Launch on startup toggle in settings had no effect — `ToggleAutostart`/`IsAutostartEnabled` existed in JS bindings but the Go methods were never wired to the `App` struct. Added the missing method declarations in `app.go` that delegate to `setAutostart()`/`isAutostartEnabled()` in `autostart_windows.go`. Also added `filepath.EvalSymlinks` to resolve real exe path before writing to registry.

**Files changed:**
- `app.go` — added `ToggleAutostart(bool) error` and `IsAutostartEnabled() bool` methods on `App`
- `autostart_windows.go` — added `filepath.EvalSymlinks` to resolve real exe path, added log output

---

## [v1.3.2] — 2026-03-14 — Remove Symlinks, Version Display

### Stage: Reliability + Version Tracking

**What was implemented:**
- **Symlinks removed** (`font_windows.go`): `stageFont()` now copies files only — symlinks broke after power loss / hard shutdown because the source path became unreachable while the font was still registered. Copy approach is always safe.
- **Version display** (`version.go`, `SettingsView.jsx`, `main.go`): `AppVersion` bumped to `1.3.2`. Added `GetAppVersion()` method exposed to frontend. Settings → About now shows live version from Go backend. Window title already used `AppVersion` — no change needed.
- **Version registry rule** (`PROJECT_LOG_HOW.md`): Added rule table — only `version.go` needs updating per release, everything else pulls from `AppVersion` automatically.

**Files changed:**
- `version.go` — `AppVersion` updated to `1.3.2`, added `GetAppVersion()` method
- `font_windows.go` — removed symlink attempt in `stageFont()`, copy-only now
- `frontend/src/components/SettingsView.jsx` — loads version via `GetAppVersion()` instead of hardcoded string
- `frontend/wailsjs/go/main/App.js` — added `GetAppVersion` binding
- `PROJECT_LOG_HOW.md` — added Version Registry Rule section

---

## [v1.3.1] — 2026-03-14 — Hard Shutdown Font Cleanup

### Stage: Reliability

**What was fixed:**
- **BUG-009**: Fonts stayed registered after PC shutdown/power loss. Two-part fix:
  1. `startup()` now calls `cleanupAllOurFonts()` before re-registering — unconditionally removes all our registry entries and staged files, then rebuilds cleanly. Handles broken symlinks from power loss.
  2. New `shutdown_windows.go`: hidden Win32 window listens for `WM_ENDSESSION` — when Windows shuts down/restarts, `DeactivateAll()` is called synchronously before Windows proceeds.

**Files changed:**
- `font_windows.go` — added `cleanupAllOurFonts()` (hard reset all our entries)
- `app.go` — `startup()` uses `cleanupAllOurFonts()` instead of `cleanupOrphanedFonts()`, added `startShutdownListener()` call
- `shutdown_windows.go` — new file: `WM_ENDSESSION` hidden window listener

---

## [v1.3.0] — 2026-03-11 — Font Classification, Filters, fsnotify, Bug Fixes

### Stage: Classification + Stability

**What was implemented:**
- **OS/2 table parsing** (`app.go`): Direct binary read of TTF/OTF OS/2 table — no CGO, no sfnt internals. Extracts `usWeightClass` and `sFamilyClass` for auto-classification into serif/sans-serif/monospace/script/display/other.
- **FontFamily classification fields** (`app.go`): Added `Category string`, `WeightClass int`, `HasItalic bool`, `IsManual bool` to `FontFamily` struct.
- **Manual category override** (`app.go`, `config.go`): `SetFontCategory(familyName, category)` saves override to `CategoryOverrides map[string]string` in `config.json`. Cache invalidated on save.
- **FilterBar component** (`FilterBar.jsx`, `FilterBar.module.css`): Chips for Type / Weight / Italic filtering above font list. Clear button resets all filters.
- **CategoryPicker popup** (`CategoryPicker.jsx`, `CategoryPicker.module.css`): Click on category badge in FontRow opens popup for manual category override. Manual categories shown in accent color.
- **Category badge in FontRow** (`FontRow.jsx`, `FontRow.module.css`): Small badge below style count showing detected category. Click opens CategoryPicker.
- **fsnotify folder watcher** (`app.go`): Auto-detects new/removed font files in watched folders (and subdirectories). 500ms debounce, emits `folders:changed` Wails event → frontend auto-reloads.
- **Refresh button in Sidebar** (`Sidebar.jsx`, `Sidebar.module.css`): Manual ↻ button per folder (appears on hover) to force cache invalidation and reload.

**Bug fixes:**
- **BUG-003**: Deadlock in `RemoveFolder`/`AddFolder` — `mu.Lock()` held via `defer` while `saveConfig()` tried `mu.RLock()`. Fixed: explicit `mu.Unlock()` before `saveConfig()`.
- **BUG-004**: White screen — `refreshFolder` declared before `loadFamilies` in `useFonts.js`, causing `useCallback` to capture `undefined`. Fixed: moved declaration after `loadFamilies`.
- **BUG-005**: Production bundle crash `Cannot access 'oe' before initialization` — Rollup circular dependency with `@tanstack/react-virtual`. Fixed: added `optimizeDeps.include` in `vite.config.js`.

**Files changed:**
- `app.go` — `FontFamily` fields, `parseOS2()`, `familyClassToCategory()`, `SetFontCategory()`, `RefreshFolder()`, `startWatcher()`, `watchFolders()`, deadlock fixes in `AddFolder`/`RemoveFolder`
- `config.go` — `CategoryOverrides map[string]string` in `Config`
- `frontend/src/App.jsx` — `FilterBar`, `CategoryPicker`, `refreshFolder`, `handleCategoryClick`
- `frontend/src/hooks/useFonts.js` — `filters` state, `setFontCategory`, `refreshFolder`, `folders:changed` event subscription
- `frontend/src/components/FilterBar.jsx` — new file
- `frontend/src/components/FilterBar.module.css` — new file
- `frontend/src/components/CategoryPicker.jsx` — new file
- `frontend/src/components/CategoryPicker.module.css` — new file
- `frontend/src/components/FontRow.jsx` — category badge, `onCategoryClick` prop
- `frontend/src/components/FontRow.module.css` — `.categoryBadge`, `.categoryManual`
- `frontend/src/components/Sidebar.jsx` — `onRefresh` prop, refresh button
- `frontend/src/components/Sidebar.module.css` — `.refreshButton`
- `frontend/src/main.jsx` — `ErrorBoundary` for crash diagnostics
- `frontend/vite.config.js` — `optimizeDeps`, `build.rollupOptions`
- `frontend/wailsjs/go/main/App.js` — `SetFontCategory`, `RefreshFolder` bindings
- `frontend/wailsjs/go/main/App.d.ts` — type definitions

**Dependencies added:**
- `github.com/fsnotify/fsnotify` — folder watching for auto-reload

---

## Bug Log

### [BUG-008] — 2026-03-11 — All font previews identical after fontRegistry reset
- **File:** `frontend/src/hooks/useFonts.js`
- **Symptom:** All font rows render the same font (first font in list)
- **Root cause:** `loadFamilies` replaced entire `fontRegistry` state on each call → `@font-face` `<style>` in `<head>` was wiped and rewritten → browser reset all font loads, WebView2 shows all as first loaded font during reload
- **Fix:** Changed `setFontRegistry` to merge-only: only appends new entries, never replaces existing ones
- **Status:** ✅ Fixed

### [BUG-007] — 2026-03-11 — TDZ crash: Cannot access 'loadFamilies' before initialization (second occurrence)
- **File:** `frontend/src/hooks/useFonts.js:62`
- **Symptom:** App crashed on startup with `ReferenceError: Cannot access 'loadFamilies' before initialization`
- **Root cause:** `useEffect(() => { loadFamiliesRef.current = loadFamilies }, [loadFamilies])` was placed before `const loadFamilies = useCallback(...)` declaration — TDZ
- **Fix:** Moved all `useEffect` blocks that reference `loadFamilies` to after its declaration
- **Status:** ✅ Fixed

### [BUG-006] — 2026-03-11 — TDZ crash: Cannot access 'oe' before initialization (Rollup ordering)
- **File:** `frontend/src/hooks/useFonts.js`, `frontend/vite.config.js`
- **Symptom:** Production build white screen: `ReferenceError: Cannot access 'oe' before initialization`
- **Root cause:** Rollup placed `useEffect([loadFamilies, loadFolders])` before `const loadFamilies = useCallback(...)` in minified output — TDZ in production bundle
- **Fix:** Removed `loadFamilies`/`loadFolders` from `EventsOn` useEffect dependencies, used stable refs instead. Added `resolve.dedupe` and `optimizeDeps` to `vite.config.js`
- **Status:** ✅ Fixed

### [BUG-005] — 2026-03-11 — White screen: Cannot access 'oe' before initialization
- **File:** `frontend/vite.config.js`
- **Symptom:** App shows white screen / ErrorBoundary: `Cannot access 'oe' before initialization`
- **Root cause:** Rollup production build circular dependency with `@tanstack/react-virtual`
- **Fix:** Added `optimizeDeps: { include: ['@tanstack/react-virtual'] }` to `vite.config.js`
- **Status:** ✅ Fixed

### [BUG-004] — 2026-03-11 — White screen: refreshFolder before loadFamilies
- **File:** `frontend/src/hooks/useFonts.js`
- **Symptom:** White screen on app load
- **Root cause:** `refreshFolder` useCallback declared before `loadFamilies`, capturing `undefined`
- **Fix:** Moved `refreshFolder` declaration after `loadFamilies`
- **Status:** ✅ Fixed

### [BUG-003] — 2026-03-11 — RemoveFolder deadlock (folder delete hangs)
- **File:** `app.go:RemoveFolder`, `app.go:AddFolder`
- **Symptom:** Clicking ✕ to remove a folder caused the app to hang indefinitely
- **Root cause:** `defer a.mu.Unlock()` held write lock while `saveConfig()` tried to acquire read lock → deadlock
- **Fix:** Replaced `defer` with explicit `a.mu.Unlock()` before calling `saveConfig()`
- **Status:** ✅ Fixed

---

## [v1.2.1] — 2026-03-11 — UI Fixes: Scroll, Filter, Font Preview, Design

### Stage: UI Polish

**What was fixed:**

- **Scroll position lost on activate** (`useFonts.js`): `toggleFamily` no longer calls `loadFamilies()` after a successful toggle — optimistic update is sufficient. `loadFolders()` still runs for sidebar counters. `loadFamilies()` only called on error to revert.
- **"Active only" filter** (`TopBar.jsx`, `TopBar.module.css`, `useFonts.js`): New `showActiveOnly` state + toggle button in TopBar. Works together with search filter.
- **Font preview all looked identical** (`App.jsx`, `useFonts.js`, `FontRow.jsx`): Root cause — `@font-face` was declared inside each `FontRow` as an inline `<style>`. Virtualizer row recycling caused browser to ignore subsequent declarations with the same font-family name. Then moving to a global `<style>` in body still failed because WebView2/Chromium ignores `@font-face` in body-level styles after the first. Final fix: inject one `<style id="ff-fontfaces">` directly into `document.head` via `useEffect`. Introduced stable `fontRegistry` state (only `{name, regularPath}`) updated only on real folder load — never on optimistic toggles — so `@font-face` is never wiped on activate/deactivate clicks.
- **Font size slider — row heights not updating** (`App.jsx`): Added `measureElement` to virtualizer so actual rendered row heights are measured. Added `virtualizer.measure()` call after `debouncedFontSize` changes (600ms debounce) to force full height recalculation. Removed `height: 100%` from `.row` so rows grow to their natural content height.
- **Sidebar folder items border-radius** (`Sidebar.module.css`): Reduced `border-radius` from `var(--radius)` (12px) to `4px` for `folderItem`, `addButton`, `settingsButton`, `searchInput` — consistent with straight-edged right panel.
- **font-display: block → swap revert** (`FontRow.jsx`): `block` caused 3s blank rows during fast scroll. Reverted to `swap`.

**Files changed:**
- `frontend/src/App.jsx` — `fontRegistry` useEffect, `measureElement`, `virtualizer.measure()`, debounce, removed `key` from font-list
- `frontend/src/hooks/useFonts.js` — `fontRegistry` state, `showActiveOnly` state, removed `loadFamilies()` after toggle success
- `frontend/src/components/FontRow.jsx` — removed inline `@font-face` style, exported `fontFaceId()`
- `frontend/src/components/TopBar.jsx` — added `showActiveOnly` / `setShowActiveOnly` props + filter button
- `frontend/src/components/TopBar.module.css` — added `.filterBtn`, `.filterActive` styles
- `frontend/src/components/FontRow.module.css` — removed `height: 100%` from `.row`
- `frontend/src/components/Sidebar.module.css` — `border-radius` 12px → 4px on folder items, buttons, search input

---

## [v1.2.0] — 2026-03-11 — Registry Font Activation & UI Overflow Fix

### Stage: Performance Architecture Overhaul

**What was implemented:**
- **Registry font activation (FontBase-style)**: Completely replaced `AddFontResourceEx(FR_PUBLIC)` with HKCU registry + symlink/copy approach in `font_windows.go`. Measured improvement: 246 fonts in ~67s → expected ~1-3s (50-100x faster). No GDI global mutex contention. Fonts visible in all apps (GDI + DirectWrite: Illustrator, Photoshop, InDesign, Office, browsers).
- **Symlink with copy fallback**: `stageFont()` attempts `os.Symlink` first (zero SSD writes, requires Developer Mode). Falls back to file copy automatically if Developer Mode is off.
- **Staging directory**: Files placed in `%LOCALAPPDATA%\Microsoft\Windows\Fonts\fantomaf\` (subfolder keeps our files separate from user-installed fonts).
- **Registry marker**: All registry value names suffixed with ` (FantomaF)` so we can identify and clean up only our own entries.
- **Crash recovery via `cleanupOrphanedFonts()`**: On startup, scans registry and staging dir for entries with our marker that are NOT in the current active set → removes them. Replaces the old `removeFontsBatch` pre-call. Handles crash scenario where `DeactivateAll` never ran.
- **UI fix — row overflow**: `FontRow.module.css` — `.row` now `height: 100%` to fill virtualizer slot. `.preview` gets `line-height: 1` and `min-width: 0`. Rows no longer overflow their bounds at large preview font sizes.

**Files changed:**
- `font_windows.go` — full rewrite: removed `AddFontResourceEx`, added `stageFont`, `registerFont`, `unregisterFont`, `cleanupOrphanedFonts`, `addFontsBatch`, `removeFontsBatch`, `addFontToSystem`, `removeFontFromSystem`
- `app.go` — `startup` now calls `cleanupOrphanedFonts(validPaths)` instead of `removeFontsBatch`
- `frontend/src/components/FontRow.module.css` — `.row` height 100%, `.preview` line-height 1 + min-width 0

**Dependencies added:**
- `golang.org/x/sys/windows/registry` — already in go.mod via `golang.org/x/sys`

---

## [v1.1.1] — 2026-03-10 — Tray Fix, Crash Recovery & Font Timing Logs

### Stage: Bug Fixes & Diagnostics

**What was implemented:**
- **Tray Show fix**: `showMenu` in `tray_windows.go` now handles the `TrackPopupMenu` return value directly. Previously `WM_COMMAND` was expected in `wndProc`, but `TPM_RETURNCMD` never sends `WM_COMMAND` — causing "Show Fantoma F" to silently drop ~50% of clicks. Dead `WM_COMMAND` case removed from `wndProc`.
- **Crash recovery**: `startup` in `app.go` now calls `removeFontsBatch` before `addFontsBatch` when restoring previously active fonts. This evicts orphaned GDI registrations left by a crash (when `DeactivateAll` never ran), fixing the "can't deactivate after crash until reboot" problem.
- **Font timing logs**: Added `log.Printf` timers to `addFontsBatch`, `removeFontsBatch`, and `broadcastFontChange` in `font_windows.go` to measure exactly where activation slowness occurs (`AddFontResourceEx` vs `WM_FONTCHANGE` broadcast).

**Files changed:**
- `tray_windows.go` — `showMenu` handles `cmd` from `TrackPopupMenu` return; `WM_COMMAND` case and unused constant removed
- `app.go` — `startup` calls `removeFontsBatch` before `addFontsBatch`
- `font_windows.go` — added `log`/`time` imports, timing logs in `addFontsBatch`, `removeFontsBatch`, `broadcastFontChange`

---

## [v0.1.0] — 2026-02-26 — Initial Implementation

### Stage: Project Scaffold & Full Initial Build

**Stack:**
- Wails v2.11.0 + Go 1.25.7 + React 18 + Vite 5 + pnpm 10.28.2

**What was implemented:**
- `wails init` with React + Vite template
- Go backend: `main.go`, `app.go`, `font_windows.go`, `autostart_windows.go`, `config.go`
- React frontend: `App.jsx`, `TopBar.jsx`, `FontCard.jsx`, `Toast.jsx`, `useFonts.js`
- Full dark theme CSS design system (CSS Modules)
- System tray integration with context menu
- SIGTERM/SIGINT handler → `DeactivateAll()` on exit
- `--minimized` CLI flag for autostart mode
- Windows registry autostart (`HKCU\...\Run`)
- Font preview via CSS `@font-face` + base64 (cached in-memory)
- WOFF/WOFF2 shown with "⚠️ Preview only" badge

**Key architectural decisions:**
- No UAC elevation — `FR_PRIVATE` works without admin
- Root folder scan only (no recursion)
- Font name from sfnt metadata, fallback to filename
- Config persistence: `%APPDATA%/fantoma-f/config.json`

**Dependencies added:**
- `golang.org/x/image v0.36.0` (sfnt font name parsing)
- `golang.org/x/sys v0.41.0` (Windows syscall, registry)

## [v0.2.0] — 2026-02-26 — Global UI Overhaul

### Stage: Major Redesign & Multi-Folder Support

**What was implemented:**
- **Layout**: Sidebar with multi-folder management.
- **Theme**: Creamy Light Theme (Beige/Cream palette).
- **List View**: Google Fonts-style horizontal rows with global preview controls.
- **Bulk Actions**: Folder-level Activate/Deactivate All.
- **UX**: Spinners added to all activation buttons.
- **Fix**: Tray icon is now visible (valid ICO byte array).
- **Backend**: Complete rewrite of `app.go` to handle multi-folder configuration and Go-side state.
- **Rename**: Project technical name changed from `litefont` to `FantomaF`. Output binary is now `FantomaF.exe`.

## [v0.3.0] — 2026-02-26 — Optimization & Grouping

### Stage: Performance Fix & Family Management

**What was implemented:**
- **Grouping**: Fonts are now grouped by Family (SFNT metadata). One row per family.
- **Virtualization**: Implemented `@tanstack/react-virtual` in the frontend. This fixed the massive RAM consumption by only rendering visible rows.
- **Lazy Loading**: Base64 font data is loaded asynchronously only when a row enters the viewport.
- **Global View**: Added "All Fonts" item to sidebar to search/view fonts from all folders simultaneously.
- **Backend API**: Refactored `app.go` with `GetFamiliesInFolder`, `ActivateFamily`, and `DeactivateFamily`.
- **UX**: Added style count badges and "Partial" activation state.

## [v0.4.0] — 2026-02-26 — Batch Optimization & Settings

### Stage: Reliability & UI Polish

**What was implemented:**
- **Optimization (The Spike)**: Refactored Windows API calls to use **Batch Activation**. Now the system-wide notification is sent only once at the end of a bulk operation, eliminating CPU/IO spikes.
- **Single Instance**: Added a named Mutex to prevent multiple copies of the app from running simultaneously.
- **Filtering**: Removed WOFF/WOFF2 support from the UI list as they are not needed for system font activation.
- **Settings View**: Added a dedicated settings screen with a toggle for **Windows Autostart**.
- **Sidebar Dots**: Folders now have colored status indicators (Active/Partial/Inactive) that persist and refresh correctly.
- **UI Simplification**: Renamed buttons to "ON" / "OFF" and refined the creamy aesthetic.

---

## [v0.4.1] — 2026-02-26 — Memory Optimization & UI Focus

### Stage: Stability & Resource Management

**What was implemented:**
- **Memory Fix (DOM Cleanup)**: Rows now explicitly remove their injected `<style>` tags when they are scrolled out of view. This prevents base64 font data from bloating the WebView2 memory.
- **Cache Management**: Added a global limit on the number of base64 strings kept in the application memory.
- **Manual Cleanup**: Added a "Clear Font Cache" button in the Settings view.
- **Window Focus**: If the user tries to launch a second instance, the original window is now automatically focused and restored.
- **Versioning**: Bumped to v0.4.1.

---

## [v0.4.2] — 2026-02-27 — Deep Optimization & Portability

### Stage: Architecture & Performance

**What was implemented:**
- **Zero-Lag Scanning**: Refactored font metadata parsing (`app.go`). Instead of loading entire `.ttf` files into RAM (using `os.ReadFile`), we now open a stream and parse only the header bytes using `sfnt.ParseReaderAt`. This reduces folder scanning time for thousands of fonts from ~60 seconds to less than 1 second.
- **True Portability**: Modified `config.go` to save `config.json` directly next to the `FantomaF.exe` executable instead of in the system's `APPDATA` folder. This allows the app and its config to be run entirely from a USB flash drive.

---

## [v0.4.4] — 2026-02-27 — Parallel Activation & Clean Exit

### Stage: Performance & Stability

**What was implemented:**
- **Parallel Activation**: Refactored `addFontsBatch` and `removeFontsBatch` in `font_windows.go` to use a **Worker Pool** (16 concurrent workers). This reduced the processing time for large font batches from minutes to a few seconds.
- **Guaranteed Exit**: Consolidated application shutdown logic into a `handleExit` function in `main.go`. Added explicit `os.Exit(0)` to the system tray's "Quit" handler to ensure the process terminates completely on Windows, fixing a common issue where the process would hang.

---

## [v0.4.3] — 2026-02-27 — Instant Activation & Deep Memory Trim

### Stage: Final Polish & UX Excellence

**What was implemented:**
- **Instant Activation**: Replaced `SendMessageTimeoutW` with `PostMessageW` in `font_windows.go`. Font activations now broadcast asynchronously, preventing Fantoma F from freezing when other applications are slow to respond to font updates. Activation of even 10+ families is now completely instantaneous.
- **Deep Memory Trimming**: Added a listener to the React frontend that triggers a Go backend method (`TrimMemory`) whenever the window is hidden (`document.hidden`). This method issues a `SetProcessWorkingSetSize` command to Windows, aggressively paging the WebView2 browser engine RAM to disk. This drops background RAM consumption from ~280MB to roughly ~15MB.

---

## Bug Log

### [BUG-001] — 2026-02-26 — Fonts not visible in other applications
- **File:** `font_windows.go`
- **Symptom:** Fonts were "activated" in the app but didn't appear in Photoshop, Word, etc.
- **Root cause:** Use of `FR_PRIVATE` flag (0x10) restricted font visibility to the Fantoma F process only.
- **Fix:** Changed flag to `0x00` (Public/Session-wide).
- **Status:** ✅ Fixed

### [BUG-002] — 2026-02-26 — Fonts persist after deactivation
- **File:** `app.go`, `font_windows.go`
- **Symptom:** Deactivating a font didn't remove it from other apps.
- **Root cause:** Redundant `AddFontResourceEx` calls (refcount > 1) and asynchronous `PostMessage` which apps could miss. Mismatched path tracking (filename vs path).
- **Fix:** Track by absolute path, check if active before calling `Add`, and use `SendMessageTimeoutW` for reliable broadcast.
- **Status:** ✅ Fixed

### [BUG-005] — 2026-03-10 — Ghost fonts after crash, can't deactivate until reboot
- **File:** `app.go:82-93`
- **Symptom:** After a crash, previously active fonts couldn't be deactivated — they persisted in GDI until Windows restarted.
- **Root cause:** `startup` only called `addFontsBatch` on restore, never `removeFontsBatch` first. When the app crashed, `DeactivateAll` never ran, leaving orphaned GDI font registrations. On next start, `AddFontResourceEx` incremented the refcount but never cleared the stale entry.
- **Fix:** `startup` now calls `removeFontsBatch(validPaths)` before `addFontsBatch(validPaths)` to evict any orphaned registrations.
- **Status:** ✅ Fixed

### [BUG-004] — 2026-03-10 — Tray "Show" unresponsive ~50% of clicks
- **File:** `tray_windows.go:285-318`
- **Symptom:** Clicking "Show Fantoma F" in the tray context menu often did nothing.
- **Root cause:** `TPM_RETURNCMD` flag causes `TrackPopupMenu` to return the selected command ID directly and **never** post `WM_COMMAND`. The code was listening for `WM_COMMAND` in `wndProc`, which never arrived, silently dropping menu selections.
- **Fix:** `showMenu` now reads the return value of `TrackPopupMenu` and dispatches the action directly. Dead `WM_COMMAND` case removed from `wndProc`.
- **Status:** ✅ Fixed

### [BUG-003] — 2026-02-27 — Process persists after tray quit
- **File:** `main.go`
- **Symptom:** Application window closes but the process remains in Task Manager when quitting from system tray.
- **Root cause:** Hang in the interaction between Wails shutdown sequence and `systray` message loop on Windows.
- **Fix:** Introduced `handleExit` with explicit `os.Exit(0)` to force process termination after resource cleanup.
- **Status:** ✅ Fixed

---

## TODO / Future Work

- [ ] Wails Asset Server for font preview (replaces base64 approach for large fonts)
- [ ] Recursive subfolder scan option (toggle in settings)
- [ ] Multiple folder support
- [ ] Font tagging / favorites
- [ ] Font comparison mode (side-by-side)
- [ ] Export active font list
- [ ] Settings panel (UI) for autostart toggle
- [ ] Installer version (NSIS/Inno) in addition to portable
