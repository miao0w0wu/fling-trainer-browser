package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// openInFileManager opens path in the OS file manager: directories open
// directly, files open with the file selected (Explorer/Finder reveal).
func openInFileManager(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		if info.IsDir() {
			command = exec.Command("explorer", path)
		} else {
			command = exec.Command("explorer", "/select,"+path)
		}
	case "darwin":
		if info.IsDir() {
			command = exec.Command("open", path)
		} else {
			command = exec.Command("open", "-R", path)
		}
	default:
		dir := path
		if !info.IsDir() {
			dir = filepath.Dir(path)
		}
		command = exec.Command("xdg-open", dir)
	}
	return command.Start()
}
