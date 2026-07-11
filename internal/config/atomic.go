package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// atomicWriteFile writes data to a temporary file next to path, fsyncs it,
// and then atomically renames it into place with the requested permissions.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(tmp)
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("fsync temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}

	// Best-effort fsync of the containing directory for durability.
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}

	return nil
}
