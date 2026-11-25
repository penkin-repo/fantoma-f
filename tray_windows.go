package main

// tray_windows.go — Native Win32 system tray icon, no third-party library.
//
// Uses a hidden top-level WS_OVERLAPPED window (NOT HWND_MESSAGE) so Explorer's
// broadcast WM_TASKBARCREATED message actually reaches wndProc after shell restarts.

import (
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// ── Win32 constants ───────────────────────────────────────────

const (
	NIF_MESSAGE uint32 = 0x01
	NIF_ICON    uint32 = 0x02
	NIF_TIP     uint32 = 0x04

	NIM_ADD    uintptr = 0x00
	NIM_MODIFY uintptr = 0x01
	NIM_DELETE uintptr = 0x02

	WM_DESTROY  = 0x0002
	WM_CLOSE    = 0x0010
	WM_USER     = 0x0400
	WM_TRAYICON = WM_USER + 1

	WM_LBUTTONDBLCLK uint32 = 0x0203
	WM_RBUTTONUP     uint32 = 0x0205

	MF_STRING    uintptr = 0x0000
	MF_SEPARATOR uintptr = 0x0800

	TPM_RIGHTBUTTON uintptr = 0x0002
	TPM_RETURNCMD   uintptr = 0x0100

	IMAGE_ICON      uintptr = 1
	LR_LOADFROMFILE uintptr = 0x10

	// Menu command IDs
	cmdTrayShow          = 3001
	cmdTrayDeactivateAll = 3002
	cmdTrayQuit          = 3003
)

// ── Win32 structs ─────────────────────────────────────────────

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type winMsg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      [2]int32
}

type point struct{ X, Y int32 }

// ── Lazy Win32 procs ──────────────────────────────────────────

var (
	modShell32 = syscall.NewLazyDLL("shell32.dll")

	procShellNotifyIcon       = modShell32.NewProc("Shell_NotifyIconW")
	procLoadImageW            = user32.NewProc("LoadImageW")
	procCreateWindowExW       = user32.NewProc("CreateWindowExW")
	procDefWindowProcW        = user32.NewProc("DefWindowProcW")
	procRegisterClassExW      = user32.NewProc("RegisterClassExW")
	procGetMessageW           = user32.NewProc("GetMessageW")
	procTranslateMessage      = user32.NewProc("TranslateMessage")
	procDispatchMessageW      = user32.NewProc("DispatchMessageW")
	procCreatePopupMenu       = user32.NewProc("CreatePopupMenu")
	procAppendMenuW           = user32.NewProc("AppendMenuW")
	procTrackPopupMenu        = user32.NewProc("TrackPopupMenu")
	procDestroyMenu           = user32.NewProc("DestroyMenu")
	procGetCursorPos          = user32.NewProc("GetCursorPos")
	procSetForegroundWindow2  = user32.NewProc("SetForegroundWindow")
	procDestroyWindow2        = user32.NewProc("DestroyWindow")
	procPostQuitMessage       = user32.NewProc("PostQuitMessage")
	procRegisterWindowMessage = user32.NewProc("RegisterWindowMessageW")
	procGetModuleHandleW      = syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW")
)

// ── TrayAction ────────────────────────────────────────────────

type TrayAction int

const (
	TrayShow TrayAction = iota
	TrayDeactivateAll
	TrayQuit
)

// ── nativeTray ────────────────────────────────────────────────

type nativeTray struct {
	mu               sync.Mutex
	hwnd             uintptr
	hIcon            uintptr
	actionCh         chan TrayAction
	wmTaskbarCreated uint32
	nid              notifyIconData
	running          int32 // atomic
}

var globalNativeTray *nativeTray

// StartNativeTray starts the tray icon loop on a dedicated OS thread.
func StartNativeTray(iconData []byte) chan TrayAction {
	t := &nativeTray{
		actionCh: make(chan TrayAction, 8),
	}
	globalNativeTray = t
	go t.run(iconData)
	return t.actionCh
}

// StopNativeTray safely closes the tray window from any goroutine.
func StopNativeTray() {
	if globalNativeTray != nil && globalNativeTray.hwnd != 0 {
		// PostMessageW(WM_CLOSE) is thread-safe; DestroyWindow runs on the owning thread.
		_, _, _ = postMsgW.Call(globalNativeTray.hwnd, WM_CLOSE, 0, 0)
	}
}

func (t *nativeTray) run(iconData []byte) {
	// Win32 message loops MUST run on a single, locked OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInst, _, _ := procGetModuleHandleW.Call(0)

	// Register WM_TASKBARCREATED before the window exists so we never miss it.
	msg, _ := syscall.UTF16PtrFromString("TaskbarCreated")
	wmTC, _, _ := procRegisterWindowMessage.Call(uintptr(unsafe.Pointer(msg)))
	t.wmTaskbarCreated = uint32(wmTC)

	t.hIcon = loadIconFromBytes(iconData)

	clsName, _ := syscall.UTF16PtrFromString("FantomaFTray")
	wc := wndClassEx{
		LpfnWndProc:   syscall.NewCallback(t.wndProc),
		HInstance:     hInst,
		LpszClassName: clsName,
	}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	// Hidden top-level window (parent = 0, not shown) receives broadcast messages like WM_TASKBARCREATED.
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(clsName)),
		uintptr(unsafe.Pointer(clsName)),
		WS_OVERLAPPED,
		0, 0, 0, 0,
		0,
		0, hInst, 0,
	)
	t.hwnd = hwnd

	tooltip := "Fantoma F v" + AppVersion + " — Font Manager"
	t.addIcon(tooltip)
	atomic.StoreInt32(&t.running, 1)

	// Start heartbeat — refreshes the icon every 60 s as an extra safety net.
	go t.heartbeat()

	var m winMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 || r == ^uintptr(0) {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	t.removeIcon()
	atomic.StoreInt32(&t.running, 0)
}

func (t *nativeTray) wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case WM_TRAYICON:
		lp := uint32(lParam)
		if lp == WM_RBUTTONUP {
			t.showMenu(hwnd)
		} else if lp == WM_LBUTTONDBLCLK {
			select {
			case t.actionCh <- TrayShow:
			default:
			}
		}
		return 0

	case WM_CLOSE:
		procDestroyWindow2.Call(hwnd)
		return 0

	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}

	// Re-register icon when Explorer restarts.
	if t.wmTaskbarCreated != 0 && uint32(msg) == t.wmTaskbarCreated {
		t.addIcon("Fantoma F v" + AppVersion + " — Font Manager")
		return 0
	}

	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

func (t *nativeTray) addIcon(tooltip string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	nid := notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:             t.hwnd,
		UID:              1,
		UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		UCallbackMessage: WM_TRAYICON,
		HIcon:            t.hIcon,
	}
	tip := syscall.StringToUTF16(tooltip)
	n := len(tip)
	if n > 127 {
		n = 127
	}
	copy(nid.SzTip[:], tip[:n])
	t.nid = nid

	// Prefer MODIFY (idempotent); fall back to ADD on first call.
	r, _, _ := procShellNotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&t.nid)))
	if r == 0 {
		procShellNotifyIcon.Call(NIM_ADD, uintptr(unsafe.Pointer(&t.nid)))
	}
}

func (t *nativeTray) removeIcon() {
	t.mu.Lock()
	defer t.mu.Unlock()
	procShellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&t.nid)))
}

func (t *nativeTray) showMenu(hwnd uintptr) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	appendMenuItem(hMenu, cmdTrayShow, "Show Fantoma F")
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	appendMenuItem(hMenu, cmdTrayDeactivateAll, "Deactivate All")
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	appendMenuItem(hMenu, cmdTrayQuit, "Quit")

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow2.Call(hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(
		hMenu,
		TPM_RIGHTBUTTON|TPM_RETURNCMD,
		uintptr(pt.X), uintptr(pt.Y),
		0, hwnd, 0,
	)
	switch uint32(cmd) {
	case cmdTrayShow:
		t.actionCh <- TrayShow
	case cmdTrayDeactivateAll:
		t.actionCh <- TrayDeactivateAll
	case cmdTrayQuit:
		t.actionCh <- TrayQuit
	}
}

func appendMenuItem(hMenu uintptr, id uint32, label string) {
	s, _ := syscall.UTF16PtrFromString(label)
	procAppendMenuW.Call(hMenu, MF_STRING, uintptr(id), uintptr(unsafe.Pointer(s)))
}

func (t *nativeTray) heartbeat() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if atomic.LoadInt32(&t.running) == 0 {
			return
		}
		t.addIcon("Fantoma F v" + AppVersion + " — Font Manager")
	}
}

func loadIconFromBytes(data []byte) uintptr {
	tmp := os.TempDir() + "\\fantomaf_icon.ico"
	_ = os.WriteFile(tmp, data, 0600)
	pTmp, _ := syscall.UTF16PtrFromString(tmp)
	hIcon, _, _ := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(pTmp)),
		IMAGE_ICON,
		0, 0,
		LR_LOADFROMFILE,
	)
	return hIcon
}
