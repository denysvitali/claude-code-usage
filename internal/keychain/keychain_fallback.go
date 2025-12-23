//go:build !darwin

// Package keychain provides platform-specific credential storage.
// This fallback implementation returns an error on non-macOS platforms.
package keychain

import (
	"errors"
	"fmt"
	"runtime"
)

// ErrNotSupported indicates that keychain is not supported on this platform.
var ErrNotSupported = fmt.Errorf("keychain access is only supported on macOS, current platform: %s", runtime.GOOS)

// ErrNotFound indicates that credentials were not found in the keychain.
var ErrNotFound = errors.New("credentials not found in keychain")

// Load always returns an error on non-macOS platforms.
func Load() ([]byte, error) {
	return nil, ErrNotSupported
}
