package main

// font_windows.go — Fast FontBase-style font activation via HKCU per-user registry
// + staged copies in %LOCALAPPDATA%\Microsoft\Windows\Fonts\fantomaf + async WM_FONTCHANGE.
//
// Deliberately avoids GDI AddFontResourceW/RemoveFontResourceW to prevent the global GDI
// kernel mutex bottleneck (~100-250ms/font). Prevents "phantom fonts" via collision-free
// hashed filenames, atomic same-volume eviction of locked files, and startup cleanup.

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	postMsgW = user32.NewProc("PostMessageW")
)

const (
	WM_FONTCHANGE  = 0x001D
	HWND_BROADCAST = 0xFFFF

	// Registry key for per-user fonts (no admin required).
	fontRegKey = `Software\Microsoft\Windows NT\CurrentVersion\Fonts`

	// Marker suffix appended to registry value names so we can identify
	// and clean up only our own entries.
	regMarker = " (FantomaF)"
)

// userFontsDir returns the staging directory inside the per-user fonts folder.
func userFontsDir() string {
	local := os.Getenv("LOCALAPPDATA")
	return filepath.Join(local, "Microsoft", "Windows", "Fonts", "fantomaf")
}

// fontTrashDir holds locked font files moved out of userFontsDir when an app
// (e.g. Illustrator) still has the file mapped during deactivation.
// Kept under LOCALAPPDATA (same NTFS volume as userFontsDir, outside Windows\Fonts)
// so os.Rename succeeds atomically even if %TEMP% is on another drive.
// Cleaned up on next startup before design apps launch.
func fontTrashDir() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return filepath.Join(os.TempDir(), "fantomaf_trash")
	}
	return filepath.Join(local, "FantomaF", "trash")
}

// stagedFilename builds a collision-free filename from the full path:
// "<8-hex-hash>_<basename>" so e.g. FontA/Regular.ttf and FontB/Regular.ttf never collide.
func stagedFilename(fontPath string) string {
	abs, err := filepath.Abs(fontPath)
	if err != nil {
		abs = fontPath
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(abs)))
	return fmt.Sprintf("%08x_%s", h.Sum32(), filepath.Base(fontPath))
}

// regValueName builds the unique registry value name for a font file.
func regValueName(fontPath string) string {
	return stagedFilename(fontPath) + regMarker
}

// stagedPath returns the absolute path where the staged copy lives.
func stagedPath(fontPath string) string {
	return filepath.Join(userFontsDir(), stagedFilename(fontPath))
}

// removeOrEvictFile tries to delete dst; if locked by Illustrator/FontCache,
// moves it atomically out of Windows\Fonts\fantomaf into %LOCALAPPDATA%\FantomaF\trash
// so Windows Font Cache stops indexing it immediately without any slow GDI calls.
func removeOrEvictFile(dst string) {
	if err := os.Remove(dst); err == nil || os.IsNotExist(err) {
		return
	}

	trashDir := fontTrashDir()
	_ = os.MkdirAll(trashDir, 0755)
	trashDst := filepath.Join(trashDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(dst)))
	if err := os.Rename(dst, trashDst); err != nil {
		log.Printf("[FONT] file locked & could not evict %s: %v", filepath.Base(dst), err)
	}
}

// stageFont copies fontPath into userFontsDir using a collision-free name.
func stageFont(fontPath string) error {
	dst := stagedPath(fontPath)
	if _, err := os.Stat(dst); err == nil {
		// Remove or evict any stale staged copy before writing fresh bytes.
		removeOrEvictFile(dst)
	}

	src, err := os.Open(fontPath)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}

	if _, err = io.Copy(out, src); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return fmt.Errorf("copy font: %w", err)
	}
	return out.Close()
}

// addFontToSystem activates a single font.
func addFontToSystem(path string) error {
	if err := os.MkdirAll(userFontsDir(), 0755); err != nil {
		return err
	}
	if err := stageFont(path); err != nil {
		return err
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, fontRegKey, registry.SET_VALUE)
	if err != nil {
		removeOrEvictFile(stagedPath(path))
		return err
	}
	err = k.SetStringValue(regValueName(path), stagedPath(path))
	_ = k.Close()
	if err != nil {
		removeOrEvictFile(stagedPath(path))
		return err
	}
	broadcastFontChange()
	return nil
}

// removeFontFromSystem deactivates a single font.
func removeFontFromSystem(path string) error {
	if k, err := registry.OpenKey(registry.CURRENT_USER, fontRegKey, registry.SET_VALUE); err == nil {
		_ = k.DeleteValue(regValueName(path))
		_ = k.DeleteValue(filepath.Base(path) + regMarker)
		_ = k.Close()
	}
	removeOrEvictFile(stagedPath(path))
	broadcastFontChange()
	return nil
}

// addFontsBatch stages files in parallel, writes HKCU registry values with a single
// opened key handle, sends one async WM_FONTCHANGE, and returns ONLY succeeded paths.
func addFontsBatch(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	if err := os.MkdirAll(userFontsDir(), 0755); err != nil {
		log.Printf("[FONT] mkdir error: %v", err)
		return nil
	}

	t0 := time.Now()

	// 1. Copy files in parallel.
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		staged = make([]string, 0, len(paths))
		sem    = make(chan struct{}, 32)
	)
	for _, path := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := stageFont(p); err != nil {
				log.Printf("[FONT] stage error %s: %v", filepath.Base(p), err)
				return
			}
			mu.Lock()
			staged = append(staged, p)
			mu.Unlock()
		}(path)
	}
	wg.Wait()

	if len(staged) == 0 {
		return nil
	}

	// 2. Write all registry entries using a single opened key handle.
	k, _, err := registry.CreateKey(registry.CURRENT_USER, fontRegKey, registry.SET_VALUE)
	if err != nil {
		log.Printf("[FONT] registry open error: %v", err)
		for _, p := range staged {
			removeOrEvictFile(stagedPath(p))
		}
		return nil
	}
	succeeded := make([]string, 0, len(staged))
	for _, p := range staged {
		if err := k.SetStringValue(regValueName(p), stagedPath(p)); err != nil {
			log.Printf("[FONT] registry error %s: %v", filepath.Base(p), err)
			removeOrEvictFile(stagedPath(p))
			continue
		}
		succeeded = append(succeeded, p)
	}
	_ = k.Close()

	log.Printf("[FONT] addFontsBatch x%d done in %v (ok=%d)", len(paths), time.Since(t0), len(succeeded))

	if len(succeeded) > 0 {
		broadcastFontChange()
	}
	return succeeded
}

// removeFontsBatch unregisters all HKCU values using one key handle, deletes/evicts
// staged files in parallel, and sends one async WM_FONTCHANGE at the end.
func removeFontsBatch(paths []string) int {
	if len(paths) == 0 {
		return 0
	}

	t0 := time.Now()

	// 1. Delete all registry entries using a single opened key handle.
	if k, err := registry.OpenKey(registry.CURRENT_USER, fontRegKey, registry.SET_VALUE); err == nil {
		for _, p := range paths {
			_ = k.DeleteValue(regValueName(p))
			_ = k.DeleteValue(filepath.Base(p) + regMarker)
		}
		_ = k.Close()
	}

	// 2. Delete or evict staged files in parallel.
	var wg sync.WaitGroup
	sem := make(chan struct{}, 32)
	for _, path := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			removeOrEvictFile(stagedPath(p))
		}(path)
	}
	wg.Wait()

	log.Printf("[FONT] removeFontsBatch x%d done in %v", len(paths), time.Since(t0))

	broadcastFontChange()
	return len(paths)
}

// cleanupAllOurFonts removes ALL registry entries, staged files, and trash leftovers
// created by Fantoma F. Called on startup before re-registering.
func cleanupAllOurFonts() {
	cleaned := false

	// 1. Clean all HKCU registry entries with our marker.
	k, err := registry.OpenKey(registry.CURRENT_USER, fontRegKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err == nil {
		names, _ := k.ReadValueNames(-1)
		for _, name := range names {
			if strings.HasSuffix(name, regMarker) {
				_ = k.DeleteValue(name)
				cleaned = true
				log.Printf("[FONT] cleanup registry: %s", name)
			}
		}
		_ = k.Close()
	}

	// 2. Clean all staged files in userFontsDir.
	dir := userFontsDir()
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			full := filepath.Join(dir, e.Name())
			removeOrEvictFile(full)
			cleaned = true
			log.Printf("[FONT] cleanup file: %s", e.Name())
		}
	}

	// 3. Sweep any previously evicted locked files from trash (current & legacy %TEMP%).
	for _, trashDir := range []string{fontTrashDir(), filepath.Join(os.TempDir(), "fantomaf_trash")} {
		if entries, err := os.ReadDir(trashDir); err == nil {
			for _, e := range entries {
				_ = os.Remove(filepath.Join(trashDir, e.Name()))
			}
		}
	}

	if cleaned {
		broadcastFontChange()
	}
}

func broadcastFontChange() {
	t0 := time.Now()
	_, _, _ = postMsgW.Call(
		uintptr(HWND_BROADCAST),
		uintptr(WM_FONTCHANGE),
		0,
		0,
	)
	log.Printf("[FONT] WM_FONTCHANGE broadcast done in %v", time.Since(t0))
}

func init() {
	_ = os.MkdirAll(userFontsDir(), 0755)
}
