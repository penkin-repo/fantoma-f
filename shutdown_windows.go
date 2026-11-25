package main

// shutdown_windows.go — Intercepts WM_QUERYENDSESSION / WM_ENDSESSION so staged
// fonts are cleaned up when Windows shuts down or restarts without closing the app first.
//
// Runs on a locked OS thread so Win32 message dispatch is guaranteed reliable.

import (
	"log"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

const (
	WM_QUERYENDSESSION = 0x0011
	WM_ENDSESSION      = 0x0016
	WS_OVERLAPPED      = 0x00000000
)

var (
	shutdownCallback func()
	shutdownOnce     sync.Once
)

func shutdownWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_QUERYENDSESSION:
		// Approve Windows shutdown cleanly.
		return 1
	case WM_ENDSESSION:
		if wParam != 0 {
			log.Println("[FONT] WM_ENDSESSION received — cleaning up staged fonts")
			shutdownOnce.Do(func() {
				if shutdownCallback != nil {
					shutdownCallback()
				}
			})
		}
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return ret
}

// startShutdownListener creates a hidden top-level window on a locked OS thread
// that receives WM_QUERYENDSESSION and WM_ENDSESSION during Windows shutdown/restart.
func startShutdownListener(onShutdown func()) {
	shutdownCallback = onShutdown

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		hInstance, _, _ := procGetModuleHandleW.Call(0)

		className, _ := syscall.UTF16PtrFromString("FantomaFShutdownClass")
		wc := wndClassEx{
			CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
			LpfnWndProc:   syscall.NewCallback(shutdownWndProc),
			HInstance:     hInstance,
			LpszClassName: className,
		}
		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

		windowName, _ := syscall.UTF16PtrFromString("FantomaFShutdownWnd")
		hwnd, _, _ := procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(windowName)),
			WS_OVERLAPPED,
			0, 0, 0, 0,
			0, 0, hInstance, 0,
		)
		if hwnd == 0 {
			log.Println("[FONT] failed to create shutdown listener window")
			return
		}

		var msg winMsg
		for {
			ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if ret == 0 || ret == ^uintptr(0) {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}()
}
