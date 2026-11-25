package main

// AppVersion is the current application version.
// Update this constant with every release.
const AppVersion = "1.4.2"

// GetAppVersion returns the current app version to the frontend.
func (a *App) GetAppVersion() string {
	return AppVersion
}
