package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/image/font/sfnt"
)

// FontStyle represents a specific variation within a family.
type FontStyle struct {
	Style    string `json:"style"`
	FilePath string `json:"filePath"`
	Format   string `json:"format"`
	IsActive bool   `json:"isActive"`
}

// FontFamily groups multiple styles of the same font.
type FontFamily struct {
	Name          string      `json:"name"`
	Styles        []FontStyle `json:"styles"`
	StylesCount   int         `json:"stylesCount"`
	RegularPath   string      `json:"regularPath"`   // Path of the most "regular" style for preview
	Format        string      `json:"format"`        // predominant format
	IsActive      bool        `json:"isActive"`      // True if at least one style is active
	IsFullyActive bool        `json:"isFullyActive"` // True if all styles are active
	Category      string      `json:"category"`      // serif/sans-serif/monospace/script/display/other
	WeightClass   int         `json:"weightClass"`   // 100-900 from OS/2
	HasItalic     bool        `json:"hasItalic"`     // true if family has italic styles
	IsManual      bool        `json:"isManual"`      // true if category was set manually by user
}

// FolderInfo adds status metadata to the persisted FolderConfig.
type FolderInfo struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Name   string `json:"name"`
	Status string `json:"status"` // "active", "partial", "inactive"
}

// App is the main application struct exposed to Wails.
type App struct {
	ctx         context.Context
	mu          sync.RWMutex
	activeFonts map[string]string // LOWERCASE ABSOLUTE PATH → original absolute path
	config      *Config
	familyCache map[string][]FontFamily // ID -> Cached Families
	watcher     *fsnotify.Watcher
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{
		activeFonts: make(map[string]string),
		familyCache: make(map[string][]FontFamily),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, err := loadConfig()
	if err != nil {
		cfg = &Config{
			Folders:           []FolderConfig{},
			CategoryOverrides: make(map[string]string),
		}
	}
	a.mu.Lock()
	a.config = cfg
	a.mu.Unlock()

	a.applyWindowTheme(cfg.effectiveTheme())

	// Hard-reset: remove ALL our registry entries, staged files, and evicted trash
	// left behind by a previous crash or locked-file deactivation.
	cleanupAllOurFonts()

	// Restore previously active fonts if enabled (default: true).
	if cfg.shouldRestoreOnStartup() && len(cfg.ActiveFonts) > 0 {
		seen := make(map[string]bool, len(cfg.ActiveFonts))
		var validPaths []string
		for _, path := range cfg.ActiveFonts {
			abs, err := filepath.Abs(path)
			if err != nil {
				continue
			}
			lower := strings.ToLower(abs)
			if seen[lower] {
				continue
			}
			if info, err := os.Stat(abs); err == nil && !info.IsDir() {
				seen[lower] = true
				validPaths = append(validPaths, abs)
			}
		}

		if len(validPaths) > 0 {
			succeeded := addFontsBatch(validPaths)
			a.mu.Lock()
			for _, path := range succeeded {
				a.activeFonts[strings.ToLower(path)] = path
			}
			a.mu.Unlock()
		}

		// Sync pruned list if some saved font files no longer exist on disk.
		if len(validPaths) != len(cfg.ActiveFonts) {
			a.saveConfig()
		}
	}

	a.startWatcher()

	// Listen for WM_ENDSESSION (Windows shutdown/restart) — clean up staged fonts
	// from Windows WITHOUT wiping ActiveFonts in config.json so they re-activate after reboot.
	startShutdownListener(func() {
		a.deactivateSystemFontsOnly()
	})
}

func (a *App) applyWindowTheme(theme string) {
	if a.ctx == nil {
		return
	}
	if theme == "light" {
		wailsruntime.WindowSetLightTheme(a.ctx)
		wailsruntime.WindowSetBackgroundColour(a.ctx, 253, 252, 249, 255)
	} else {
		wailsruntime.WindowSetDarkTheme(a.ctx)
		wailsruntime.WindowSetBackgroundColour(a.ctx, 17, 18, 20, 255)
	}
}

// startWatcher sets up fsnotify to watch all configured folders (and new subdirectories).
func (a *App) startWatcher() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	a.watcher = w

	a.mu.RLock()
	folders := make([]FolderConfig, len(a.config.Folders))
	copy(folders, a.config.Folders)
	a.mu.RUnlock()

	a.watchFolders(folders)

	go func() {
		var debounce *time.Timer
		triggerReload := func() {
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(500*time.Millisecond, func() {
				a.mu.Lock()
				a.familyCache = make(map[string][]FontFamily)
				a.mu.Unlock()
				if a.ctx != nil {
					wailsruntime.EventsEmit(a.ctx, "folders:changed")
				}
			})
		}

		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				// Dynamically watch newly created subdirectories (e.g. unzipped font packs).
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						a.watchPathTree(event.Name)
						triggerReload()
						continue
					}
				}
				ext := strings.ToLower(filepath.Ext(event.Name))
				if ext != ".ttf" && ext != ".otf" {
					continue
				}
				triggerReload()
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
}

// watchPathTree adds root and all subdirectories to the fsnotify watcher.
func (a *App) watchPathTree(root string) {
	if a.watcher == nil {
		return
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		_ = a.watcher.Add(path)
		return nil
	})
}

// unwatchPathTree removes root and all its subdirectories from the fsnotify watcher.
func (a *App) unwatchPathTree(root string) {
	if a.watcher == nil {
		return
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		_ = a.watcher.Remove(path)
		return nil
	})
}

func (a *App) watchFolders(folders []FolderConfig) {
	for _, f := range folders {
		a.watchPathTree(f.Path)
	}
}

// deactivateSystemFontsOnly removes all currently staged fonts from Windows
// WITHOUT clearing config.json's ActiveFonts list (used on app exit / Windows shutdown).
func (a *App) deactivateSystemFontsOnly() {
	a.mu.RLock()
	toRemove := make([]string, 0, len(a.activeFonts))
	for _, origPath := range a.activeFonts {
		toRemove = append(toRemove, origPath)
	}
	a.mu.RUnlock()

	if len(toRemove) > 0 {
		removeFontsBatch(toRemove)
	}
}

// shutdown is called when the app shuts down — clean up staged fonts from Windows
// while preserving ActiveFonts in config.json for the next launch.
func (a *App) shutdown(ctx context.Context) {
	a.deactivateSystemFontsOnly()
	if a.watcher != nil {
		_ = a.watcher.Close()
	}
}

// AddFolder opens a native folder-picker dialog and adds it to the managed list.
func (a *App) AddFolder() (*FolderInfo, error) {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Add Font Folder",
	})
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, nil // user cancelled
	}

	absDir, err := filepath.Abs(dir)
	if err == nil {
		dir = absDir
	}

	a.mu.Lock()
	for _, f := range a.config.Folders {
		if strings.EqualFold(f.Path, dir) {
			a.mu.Unlock()
			return nil, fmt.Errorf("folder already exists")
		}
	}
	folder := FolderConfig{
		ID:   uuid.NewString(),
		Path: dir,
		Name: filepath.Base(dir),
	}
	a.config.Folders = append(a.config.Folders, folder)
	delete(a.familyCache, "all")
	a.mu.Unlock()

	a.saveConfig()
	a.watchPathTree(folder.Path)

	return &FolderInfo{ID: folder.ID, Path: folder.Path, Name: folder.Name, Status: "inactive"}, nil
}

// RemoveFolder deactivates fonts belonging to the folder and removes it from the managed list.
func (a *App) RemoveFolder(id string) error {
	a.mu.Lock()
	idx := -1
	var folderPath string
	for i, f := range a.config.Folders {
		if f.ID == id {
			idx = i
			folderPath = f.Path
			break
		}
	}
	if idx == -1 {
		a.mu.Unlock()
		return fmt.Errorf("folder not found")
	}
	a.config.Folders = append(a.config.Folders[:idx], a.config.Folders[idx+1:]...)
	delete(a.familyCache, id)
	delete(a.familyCache, "all")

	// Find any active fonts located inside the removed folder so they don't become phantoms.
	prefix := strings.ToLower(filepath.Clean(folderPath)) + string(os.PathSeparator)
	var toRemove []string
	var keysToDelete []string
	for lowerPath, origPath := range a.activeFonts {
		if strings.HasPrefix(lowerPath, prefix) {
			toRemove = append(toRemove, origPath)
			keysToDelete = append(keysToDelete, lowerPath)
		}
	}
	for _, k := range keysToDelete {
		delete(a.activeFonts, k)
	}
	a.mu.Unlock()

	a.unwatchPathTree(folderPath)
	if len(toRemove) > 0 {
		removeFontsBatch(toRemove)
	}

	a.saveConfig()
	return nil
}

// RefreshFolder invalidates the family cache for a folder so newly added files are picked up.
func (a *App) RefreshFolder(id string) error {
	a.mu.Lock()
	delete(a.familyCache, id)
	delete(a.familyCache, "all")
	var folderPath string
	for _, f := range a.config.Folders {
		if f.ID == id {
			folderPath = f.Path
			break
		}
	}
	a.mu.Unlock()

	if folderPath != "" {
		a.watchPathTree(folderPath)
	}
	return nil
}

// ClearCache clears the in-memory font metadata cache so folders are freshly re-scanned.
func (a *App) ClearCache() error {
	a.mu.Lock()
	a.familyCache = make(map[string][]FontFamily)
	a.mu.Unlock()
	return nil
}

// GetFolders returns the list of managed folders with their current status.
func (a *App) GetFolders() []FolderInfo {
	a.mu.RLock()
	folders := make([]FolderConfig, len(a.config.Folders))
	copy(folders, a.config.Folders)
	a.mu.RUnlock()

	infos := make([]FolderInfo, 0, len(folders))
	for _, f := range folders {
		status := a.calculateFolderStatus(f.Path)
		infos = append(infos, FolderInfo{
			ID:     f.ID,
			Path:   f.Path,
			Name:   f.Name,
			Status: status,
		})
	}
	return infos
}

// getFontFilePathsLightning returns paths to all fonts in a folder without parsing metadata.
func (a *App) getFontFilePathsLightning(folderPath string) []string {
	var paths []string
	_ = filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext == ".ttf" || ext == ".otf" {
			absPath, _ := filepath.Abs(path)
			paths = append(paths, absPath)
		}
		return nil
	})
	return paths
}

// parsedStyle carries per-style classification metadata before sorting & aggregation.
type parsedStyle struct {
	style       FontStyle
	weightClass int
	category    string
	isItalic    bool
}

// GetFamiliesInFolder returns fonts grouped by family, filtering out WOFF/WOFF2.
func (a *App) GetFamiliesInFolder(id string) ([]FontFamily, error) {
	var folderPaths []string
	a.mu.RLock()
	if id == "all" {
		for _, f := range a.config.Folders {
			folderPaths = append(folderPaths, f.Path)
		}
	} else {
		for _, f := range a.config.Folders {
			if f.ID == id {
				folderPaths = append(folderPaths, f.Path)
				break
			}
		}
	}
	cached, ok := a.familyCache[id]
	a.mu.RUnlock()

	if len(folderPaths) == 0 && id != "all" {
		return nil, fmt.Errorf("folder not found")
	}

	if ok {
		// Deep-copy families & styles so we never mutate shared cache backing arrays.
		a.mu.RLock()
		result := make([]FontFamily, len(cached))
		for i, fam := range cached {
			stylesCopy := make([]FontStyle, len(fam.Styles))
			anyActive := false
			allActive := len(fam.Styles) > 0
			for j, s := range fam.Styles {
				_, active := a.activeFonts[strings.ToLower(s.FilePath)]
				s.IsActive = active
				stylesCopy[j] = s
				if active {
					anyActive = true
				} else {
					allActive = false
				}
			}
			fam.Styles = stylesCopy
			fam.IsActive = anyActive
			fam.IsFullyActive = allActive
			result[i] = fam
		}
		a.mu.RUnlock()
		return result, nil
	}

	// Phase 1: Collect all unique valid font file paths.
	seenFiles := make(map[string]bool)
	var filesToProcess []string
	for _, folderPath := range folderPaths {
		_ = filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if ext == ".ttf" || ext == ".otf" {
				absPath, _ := filepath.Abs(path)
				lower := strings.ToLower(absPath)
				if !seenFiles[lower] {
					seenFiles[lower] = true
					filesToProcess = append(filesToProcess, absPath)
				}
			}
			return nil
		})
	}

	type familyAcc struct {
		name   string
		format string
		styles []parsedStyle
	}

	familyMap := make(map[string]*familyAcc)
	var mapMu sync.Mutex

	// Phase 2: Worker pool
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	pathsCh := make(chan string, len(filesToProcess))
	for _, p := range filesToProcess {
		pathsCh <- p
	}
	close(pathsCh)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for absPath := range pathsCh {
				filename := filepath.Base(absPath)
				ext := strings.ToLower(filepath.Ext(filename))
				lowerPath := strings.ToLower(absPath)

				familyName, styleName := parseFontMeta(absPath, filename)

				a.mu.RLock()
				_, isActive := a.activeFonts[lowerPath]
				a.mu.RUnlock()

				meta := parseFontClassification(absPath)
				isItalic := isItalicStyleName(styleName)

				ps := parsedStyle{
					style: FontStyle{
						Style:    styleName,
						FilePath: absPath,
						Format:   strings.TrimPrefix(ext, "."),
						IsActive: isActive,
					},
					weightClass: meta.weightClass,
					category:    meta.category,
					isItalic:    isItalic,
				}

				mapMu.Lock()
				acc, exists := familyMap[familyName]
				if !exists {
					acc = &familyAcc{
						name:   familyName,
						format: strings.TrimPrefix(ext, "."),
					}
					familyMap[familyName] = acc
				}
				acc.styles = append(acc.styles, ps)
				mapMu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Snapshot category overrides once under RLock.
	a.mu.RLock()
	overrides := make(map[string]string, len(a.config.CategoryOverrides))
	for k, v := range a.config.CategoryOverrides {
		overrides[k] = v
	}
	a.mu.RUnlock()

	families := make([]FontFamily, 0, len(familyMap))
	for _, acc := range familyMap {
		// Deterministic sort: Regular/Book/Roman first, then by weight, then alphabetically.
		sort.Slice(acc.styles, func(i, j int) bool {
			si := strings.ToLower(acc.styles[i].style.Style)
			sj := strings.ToLower(acc.styles[j].style.Style)
			isRegI := si == "regular" || si == "book" || si == "roman" || si == "normal"
			isRegJ := sj == "regular" || sj == "book" || sj == "roman" || sj == "normal"
			if isRegI != isRegJ {
				return isRegI
			}
			if acc.styles[i].isItalic != acc.styles[j].isItalic {
				return !acc.styles[i].isItalic
			}
			if acc.styles[i].weightClass != acc.styles[j].weightClass {
				return acc.styles[i].weightClass < acc.styles[j].weightClass
			}
			return si < sj
		})

		styles := make([]FontStyle, len(acc.styles))
		hasItalic := false
		anyActive := false
		allActive := len(acc.styles) > 0
		primaryWeight := acc.styles[0].weightClass
		primaryCategory := acc.styles[0].category

		for i, ps := range acc.styles {
			styles[i] = ps.style
			if ps.isItalic {
				hasItalic = true
			}
			if ps.style.IsActive {
				anyActive = true
			} else {
				allActive = false
			}
			// If primary style had "other", pick a more specific category from any sibling style.
			if primaryCategory == "other" && ps.category != "other" {
				primaryCategory = ps.category
			}
		}

		isManual := false
		if override, ok := overrides[acc.name]; ok && override != "" {
			primaryCategory = override
			isManual = true
		}

		families = append(families, FontFamily{
			Name:          acc.name,
			Styles:        styles,
			StylesCount:   len(styles),
			RegularPath:   styles[0].FilePath,
			Format:        acc.format,
			IsActive:      anyActive,
			IsFullyActive: allActive,
			Category:      primaryCategory,
			WeightClass:   primaryWeight,
			HasItalic:     hasItalic,
			IsManual:      isManual,
		})
	}

	sort.Slice(families, func(i, j int) bool {
		return strings.ToLower(families[i].Name) < strings.ToLower(families[j].Name)
	})

	a.mu.Lock()
	a.familyCache[id] = families
	a.mu.Unlock()

	return families, nil
}

// ActivateFont temporarily installs a single font style.
func (a *App) ActivateFont(fullPath string) error {
	absPath, _ := filepath.Abs(fullPath)
	lowerPath := strings.ToLower(absPath)

	a.mu.RLock()
	_, already := a.activeFonts[lowerPath]
	a.mu.RUnlock()
	if already {
		return nil
	}

	if err := addFontToSystem(absPath); err != nil {
		return err
	}

	a.mu.Lock()
	a.activeFonts[lowerPath] = absPath
	a.mu.Unlock()

	a.saveConfig()
	return nil
}

// DeactivateFont removes a single temporarily installed font style.
func (a *App) DeactivateFont(fullPath string) error {
	absPath, _ := filepath.Abs(fullPath)
	lowerPath := strings.ToLower(absPath)

	a.mu.RLock()
	_, active := a.activeFonts[lowerPath]
	a.mu.RUnlock()
	if !active {
		return nil
	}

	if err := removeFontFromSystem(absPath); err != nil {
		return err
	}

	a.mu.Lock()
	delete(a.activeFonts, lowerPath)
	a.mu.Unlock()

	a.saveConfig()
	return nil
}

// ActivateFamily activates all styles in a family BATCHED.
func (a *App) ActivateFamily(styles []FontStyle) error {
	var toAdd []string
	a.mu.RLock()
	for _, s := range styles {
		absPath, _ := filepath.Abs(s.FilePath)
		lowerPath := strings.ToLower(absPath)
		if _, ok := a.activeFonts[lowerPath]; !ok {
			toAdd = append(toAdd, absPath)
		}
	}
	a.mu.RUnlock()

	if len(toAdd) == 0 {
		return nil
	}

	succeeded := addFontsBatch(toAdd)
	if len(succeeded) == 0 {
		return fmt.Errorf("failed to activate font files (check if files exist or are locked)")
	}

	a.mu.Lock()
	for _, path := range succeeded {
		a.activeFonts[strings.ToLower(path)] = path
	}
	a.mu.Unlock()

	a.saveConfig()
	return nil
}

// DeactivateFamily deactivates all styles in a family BATCHED.
func (a *App) DeactivateFamily(styles []FontStyle) error {
	var toRemove []string
	a.mu.RLock()
	for _, s := range styles {
		absPath, _ := filepath.Abs(s.FilePath)
		lowerPath := strings.ToLower(absPath)
		if orig, ok := a.activeFonts[lowerPath]; ok {
			toRemove = append(toRemove, orig)
		}
	}
	a.mu.RUnlock()

	if len(toRemove) == 0 {
		return nil
	}

	removeFontsBatch(toRemove)

	a.mu.Lock()
	for _, path := range toRemove {
		delete(a.activeFonts, strings.ToLower(path))
	}
	a.mu.Unlock()

	a.saveConfig()
	return nil
}

// ActivateFolder activates all TTF/OTF fonts in a managed folder BATCHED.
func (a *App) ActivateFolder(id string) error {
	a.mu.RLock()
	var folderPath string
	for _, f := range a.config.Folders {
		if f.ID == id {
			folderPath = f.Path
			break
		}
	}
	a.mu.RUnlock()

	if folderPath == "" {
		return nil
	}

	allPaths := a.getFontFilePathsLightning(folderPath)

	var toAdd []string
	a.mu.RLock()
	for _, path := range allPaths {
		lowerPath := strings.ToLower(path)
		if _, ok := a.activeFonts[lowerPath]; !ok {
			toAdd = append(toAdd, path)
		}
	}
	a.mu.RUnlock()

	if len(toAdd) == 0 {
		return nil
	}

	succeeded := addFontsBatch(toAdd)
	if len(succeeded) == 0 {
		return fmt.Errorf("failed to activate fonts in folder")
	}

	a.mu.Lock()
	for _, path := range succeeded {
		a.activeFonts[strings.ToLower(path)] = path
	}
	a.mu.Unlock()

	a.saveConfig()
	return nil
}

// DeactivateFolder deactivates all fonts in a managed folder BATCHED.
func (a *App) DeactivateFolder(id string) error {
	a.mu.RLock()
	var folderPath string
	for _, f := range a.config.Folders {
		if f.ID == id {
			folderPath = f.Path
			break
		}
	}
	a.mu.RUnlock()

	if folderPath == "" {
		return nil
	}

	filesInFolder := a.getFontFilePathsLightning(folderPath)
	pathMap := make(map[string]bool, len(filesInFolder))
	for _, p := range filesInFolder {
		pathMap[strings.ToLower(p)] = true
	}
	prefix := strings.ToLower(filepath.Clean(folderPath)) + string(os.PathSeparator)

	var toRemove []string
	a.mu.RLock()
	for lowerPath, origPath := range a.activeFonts {
		if pathMap[lowerPath] || strings.HasPrefix(lowerPath, prefix) {
			toRemove = append(toRemove, origPath)
		}
	}
	a.mu.RUnlock()

	if len(toRemove) == 0 {
		return nil
	}

	removeFontsBatch(toRemove)

	a.mu.Lock()
	for _, path := range toRemove {
		delete(a.activeFonts, strings.ToLower(path))
	}
	a.mu.Unlock()

	a.saveConfig()
	return nil
}

// DeactivateAll removes all active fonts from Windows AND clears ActiveFonts in config.json.
// Emits "fonts:changed" so the frontend stays in sync (e.g. when invoked from the system tray).
func (a *App) DeactivateAll() error {
	a.mu.RLock()
	toRemove := make([]string, 0, len(a.activeFonts))
	for _, origPath := range a.activeFonts {
		toRemove = append(toRemove, origPath)
	}
	a.mu.RUnlock()

	if len(toRemove) > 0 {
		removeFontsBatch(toRemove)
	}

	a.mu.Lock()
	a.activeFonts = make(map[string]string)
	a.mu.Unlock()

	a.saveConfig()

	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "fonts:changed")
	}
	return nil
}

// ToggleAutostart enables or disables Windows autostart.
func (a *App) ToggleAutostart(enable bool) error {
	return setAutostart(enable)
}

// IsAutostartEnabled returns autostart status.
func (a *App) IsAutostartEnabled() bool {
	return isAutostartEnabled()
}

// GetTheme returns the saved UI theme ("dark" or "light").
func (a *App) GetTheme() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.effectiveTheme()
}

// SetTheme updates the UI theme, native Windows titlebar theme, and persists it.
func (a *App) SetTheme(theme string) error {
	if theme != "light" {
		theme = "dark"
	}
	a.mu.Lock()
	a.config.Theme = theme
	a.mu.Unlock()

	a.applyWindowTheme(theme)
	a.saveConfig()
	return nil
}

// GetRestoreOnStartup returns whether active fonts are automatically re-activated after reboot.
func (a *App) GetRestoreOnStartup() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.shouldRestoreOnStartup()
}

// SetRestoreOnStartup toggles automatic re-activation of previously active fonts on startup.
func (a *App) SetRestoreOnStartup(enable bool) error {
	val := enable
	a.mu.Lock()
	a.config.RestoreOnStartup = &val
	a.mu.Unlock()
	a.saveConfig()
	return nil
}

// calculateFolderStatus checks if all/some/none fonts in a path are active.
func (a *App) calculateFolderStatus(folderPath string) string {
	var total, active int
	a.mu.RLock()
	defer a.mu.RUnlock()

	_ = filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".ttf" && ext != ".otf" {
			return nil
		}
		total++
		abs, _ := filepath.Abs(path)
		if _, ok := a.activeFonts[strings.ToLower(abs)]; ok {
			active++
		}
		return nil
	})

	if total == 0 || active == 0 {
		return "inactive"
	}
	if active == total {
		return "active"
	}
	return "partial"
}

// SetFontCategory sets a manual category override for a font family and persists it.
func (a *App) SetFontCategory(familyName, category string) error {
	a.mu.Lock()
	if a.config.CategoryOverrides == nil {
		a.config.CategoryOverrides = make(map[string]string)
	}
	if category == "" {
		delete(a.config.CategoryOverrides, familyName)
	} else {
		a.config.CategoryOverrides[familyName] = category
	}
	a.familyCache = make(map[string][]FontFamily)
	a.mu.Unlock()
	a.saveConfig()
	return nil
}

// saveConfig safely snapshots Config under RLock and writes it atomically.
func (a *App) saveConfig() {
	a.mu.RLock()
	active := make([]string, 0, len(a.activeFonts))
	for _, origPath := range a.activeFonts {
		active = append(active, origPath)
	}
	sort.Strings(active)

	foldersCopy := make([]FolderConfig, len(a.config.Folders))
	copy(foldersCopy, a.config.Folders)

	overridesCopy := make(map[string]string, len(a.config.CategoryOverrides))
	for k, v := range a.config.CategoryOverrides {
		overridesCopy[k] = v
	}

	var restoreCopy *bool
	if a.config.RestoreOnStartup != nil {
		v := *a.config.RestoreOnStartup
		restoreCopy = &v
	}

	snapshot := Config{
		Folders:           foldersCopy,
		ActiveFonts:       active,
		CategoryOverrides: overridesCopy,
		Theme:             a.config.Theme,
		RestoreOnStartup:  restoreCopy,
	}
	a.mu.RUnlock()

	_ = saveConfigSnapshot(snapshot)
}

// fontClassification holds fields extracted from OS/2 and post OpenType tables.
type fontClassification struct {
	weightClass int
	category    string
}

// parseFontClassification reads OS/2 (weight, sFamilyClass, PANOSE) and post (isFixedPitch)
// directly from TTF/OTF binary tables.
func parseFontClassification(fullPath string) fontClassification {
	ext := strings.ToLower(filepath.Ext(fullPath))
	if ext != ".ttf" && ext != ".otf" {
		return fontClassification{weightClass: 400, category: "other"}
	}
	f, err := os.Open(fullPath)
	if err != nil {
		return fontClassification{weightClass: 400, category: "other"}
	}
	defer f.Close()

	header := make([]byte, 12)
	if _, err := f.ReadAt(header, 0); err != nil {
		return fontClassification{weightClass: 400, category: "other"}
	}
	numTables := int(binary.BigEndian.Uint16(header[4:6]))
	if numTables <= 0 || numTables > 256 {
		return fontClassification{weightClass: 400, category: "other"}
	}

	records := make([]byte, numTables*16)
	if _, err := f.ReadAt(records, 12); err != nil {
		return fontClassification{weightClass: 400, category: "other"}
	}

	var os2Offset, os2Length uint32
	var postOffset, postLength uint32
	for i := 0; i < numTables; i++ {
		rec := records[i*16 : i*16+16]
		tag := string(rec[0:4])
		switch tag {
		case "OS/2":
			os2Offset = binary.BigEndian.Uint32(rec[8:12])
			os2Length = binary.BigEndian.Uint32(rec[12:16])
		case "post":
			postOffset = binary.BigEndian.Uint32(rec[8:12])
			postLength = binary.BigEndian.Uint32(rec[12:16])
		}
	}

	weightClass := 400
	category := "other"
	isMono := false

	// Check post.isFixedPitch (offset 12, uint32 != 0) for monospace detection.
	if postOffset > 0 && postLength >= 16 {
		postHead := make([]byte, 16)
		if _, err := f.ReadAt(postHead, int64(postOffset)); err == nil {
			if binary.BigEndian.Uint32(postHead[12:16]) != 0 {
				isMono = true
			}
		}
	}

	if os2Offset > 0 && os2Length >= 32 && os2Length <= 1024 {
		raw := make([]byte, os2Length)
		if _, err := f.ReadAt(raw, int64(os2Offset)); err == nil {
			wc := int(binary.BigEndian.Uint16(raw[4:6]))
			if wc > 0 {
				weightClass = wc
			}
			// PANOSE starts at offset 32 (10 bytes): byte 3 is bProportion (9 = Monospaced)
			if len(raw) >= 42 && raw[35] == 9 {
				isMono = true
			}
			familyClass := int(raw[30])
			category = familyClassToCategory(familyClass)
		}
	}

	if isMono {
		category = "monospace"
	}

	return fontClassification{weightClass: weightClass, category: category}
}

func familyClassToCategory(cls int) string {
	switch cls {
	case 1, 2, 3, 4, 5, 7: // Old Style/Transitional/Modern/Clarendon/Slab/Freeform Serifs
		return "serif"
	case 8: // Sans Serif
		return "sans-serif"
	case 9: // Ornamentals
		return "display"
	case 10: // Scripts
		return "script"
	case 12: // Symbolic
		return "display"
	default:
		return "other"
	}
}

func isItalicStyleName(style string) bool {
	s := strings.ToLower(style)
	return strings.Contains(s, "italic") || strings.Contains(s, "oblique") || strings.Contains(s, "slanted")
}

func parseFontMeta(fullPath, filename string) (family, style string) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".ttf" || ext == ".otf" {
		file, err := os.Open(fullPath)
		if err == nil {
			defer file.Close()
			f, err := sfnt.ParseReaderAt(file)
			if err == nil {
				buf := &sfnt.Buffer{}
				// 1. Typographic Family (ID 16)
				famVal, err := f.Name(buf, sfnt.NameIDTypographicFamily)
				if err == nil && famVal != "" {
					styleVal, _ := f.Name(buf, sfnt.NameIDTypographicSubfamily)
					if styleVal == "" {
						styleVal = "Regular"
					}
					return strings.TrimSpace(famVal), strings.TrimSpace(styleVal)
				}

				// 2. Standard Family (ID 1)
				famVal, err = f.Name(buf, sfnt.NameIDFamily)
				if err == nil && famVal != "" {
					styleVal, _ := f.Name(buf, sfnt.NameIDSubfamily)
					if styleVal == "" {
						styleVal = "Regular"
					}
					return strings.TrimSpace(famVal), strings.TrimSpace(styleVal)
				}

				// 3. PostScript name
				ps, err := f.Name(buf, sfnt.NameIDPostScript)
				if err == nil && ps != "" {
					return strings.TrimSpace(ps), "Regular"
				}
			}
		}
	}
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.SplitN(base, "-", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return base, "Regular"
}

// TrimMemory explicitly empties the process working set when hidden in tray.
func (a *App) TrimMemory() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setProcessWorkingSetSize := kernel32.NewProc("SetProcessWorkingSetSize")
	currentProcess, _ := syscall.GetCurrentProcess()
	setProcessWorkingSetSize.Call(uintptr(currentProcess), ^uintptr(0), ^uintptr(0))
}
