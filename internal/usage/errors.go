package usage

import "errors"

// ErrNoValidCredentials is returned when no valid credentials are found
var ErrNoValidCredentials = errors.New("no valid credentials in keychain")
