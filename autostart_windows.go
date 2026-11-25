//go:build windows

package main

import (
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	autostartRegKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
	autostartAppName = "FantomaF"
)

// setAutostart adds or removes the app from Windows startup via HKCU registry.
func setAutostart(enable bool) error {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		autostartRegKey,
		registry.SET_VALUE|registry.QUERY_VALUE,
	)
	if err != nil {
		return err
	}
	defer key.Close()

	if enable {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		// Resolve symlinks — os.Executable may return a symlinked path on some setups
		if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
			exePath = resolved
		}
		value := `"` + exePath + `" --minimized`
		log.Printf("[AUTOSTART] registering: %s", value)
		return key.SetStringValue(autostartAppName, value)
	}

	return key.DeleteValue(autostartAppName)
}

// isAutostartEnabled checks whether the registry key exists.
func isAutostartEnabled() bool {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		autostartRegKey,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(autostartAppName)
	return err == nil
}
