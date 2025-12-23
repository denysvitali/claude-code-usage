//go:build darwin

// Package keychain provides platform-specific credential storage.
// On macOS, it uses the native Keychain Services API.
package keychain

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework Security

#import <Foundation/Foundation.h>
#import <Security/Security.h>

// GetCredentials retrieves the "Claude Code-credentials" from the keychain.
// Returns a JSON string containing the credentials, or an error message.
// The caller must free the returned string using free().
char* GetCredentials() {
	NSDictionary* query = @{
		(__bridge id)kSecClass: (__bridge id)kSecClassGenericPassword,
		(__bridge id)kSecAttrService: @"Claude Code-credentials",
		(__bridge id)kSecMatchLimit: (__bridge id)kSecMatchLimitOne,
		(__bridge id)kSecReturnData: @YES,
	};

	CFTypeRef result = NULL;
	OSStatus status = SecItemCopyMatching((__bridge CFDictionaryRef)query, &result);

	if (status != errSecSuccess) {
		if (status == errSecItemNotFound) {
			// Return empty string for "not found" to distinguish from errors
			return strdup("");
		}
		char* err_msg = malloc(128);
		snprintf(err_msg, 128, "keychain error: %d", (int)status);
		return err_msg;
	}

	NSData* data = (__bridge_transfer NSData*)result;
	NSString* jsonString = [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];

	if (!jsonString) {
		return strdup("keychain error: failed to decode data");
	}

	return strdup([jsonString UTF8String]);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

// ErrNotFound indicates that credentials were not found in the keychain.
var ErrNotFound = errors.New("credentials not found in keychain")

// Load retrieves credentials JSON from the macOS keychain.
// It looks for the "Claude Code-credentials" entry which contains
// a JSON with the claudeAiOauth field.
// Returns the raw JSON bytes or an error.
func Load() ([]byte, error) {
	cJson := C.GetCredentials()
	defer C.free(unsafe.Pointer(cJson))

	jsonStr := C.GoString(cJson)

	// Empty string means item not found
	if jsonStr == "" {
		return nil, ErrNotFound
	}

	// Check for error messages that start with "keychain error:"
	if len(jsonStr) > 14 && jsonStr[:14] == "keychain error:" {
		return nil, fmt.Errorf("failed to read from keychain: %s", jsonStr)
	}

	return []byte(jsonStr), nil
}
