package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	cacheDirName  = "k0s_version"
	cacheFileName = "known_versions.txt"
)

// File returns the absolute path to the known versions cache file.
// The base directory honors XDG_CACHE_HOME when set; otherwise it
// uses os.UserCacheDir for a platform-aware default.
func File() (string, error) {
	if base := os.Getenv("XDG_CACHE_HOME"); base != "" {
		return filepath.Join(base, cacheDirName, cacheFileName), nil
	}

	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("determine cache directory: %w", err)
	}

	if base == "" {
		return "", errors.New("cache base directory is empty")
	}

	return filepath.Join(base, cacheDirName, cacheFileName), nil
}
