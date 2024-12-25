package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// getSize calculates the total size of the file or folder at the given path.
// It returns the size in bytes and any encountered error.
func GetSize(path string) (uint64, error) {
	var size uint64

	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("unable to access path: %v", err)
	}

	if info.IsDir() {
		err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			size += uint64(info.Size())
			return nil
		})
		if err != nil {
			return 0, fmt.Errorf("error walking through directory: %v", err)
		}
	} else {
		size = uint64(info.Size())
	}

	return size, nil
}

// Helper function to expand ~ to home directory
func ExpandPath(path string) (string, error) {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine home directory: %v", err)
		}
		return filepath.Join(homeDir, path[1:]), nil
	}
	return path, nil
}
