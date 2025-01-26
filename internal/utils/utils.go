package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/filecoin-project/go-address"
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

// getStringActorId converts a signerActorId to its string representation,
// removes the first two characters ("f0" or "t0"), and returns the remaining string.
// It returns an error if the signerActorId does not start with "f0" or "t0" or is too short.
func GetStringActorId(signerActorId address.Address) (string, error) {
    s := signerActorId.String()
    if len(s) <= 2 {
        return "", fmt.Errorf("invalid signerActorId: too short")
    }

    prefix := s[:2]
    if prefix != "f0" && prefix != "t0" {
        return "", fmt.Errorf("invalid signerActorId prefix: expected 'f0' or 't0', got '%s'", prefix)
    }

    return s[2:], nil
}