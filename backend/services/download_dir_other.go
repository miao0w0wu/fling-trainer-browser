//go:build !windows

package services

import (
	"os"
	"path/filepath"
)

// userDownloadsRoot falls back to ~/Downloads on platforms without a
// configurable known-downloads location.
func userDownloadsRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Downloads")
}
