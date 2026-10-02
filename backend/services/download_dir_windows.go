//go:build windows

package services

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// userDownloadsRoot resolves the Downloads folder the user configured in
// Windows, which may live on any drive, instead of assuming ~/Downloads.
func userDownloadsRoot() string {
	if known, err := windows.KnownFolderPath(windows.FOLDERID_Downloads, windows.KF_FLAG_DONT_VERIFY); err == nil && known != "" {
		return known
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Downloads")
}
