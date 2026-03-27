package utils

import (
	"os"
	"path/filepath"
)

// ExpandPath expands ~ to user's home directory
func ExpandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		// Handle both / and \ as path separators
		if len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
			return filepath.Join(home, path[2:])
		}
		return home
	}
	path, _ = filepath.Abs(path)
	return path
}
