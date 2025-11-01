package auth

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	keyring "github.com/zalando/go-keyring"
)

const keyringService = "sabx"

// ErrNotFound is returned when no key is stored for the profile/baseURL pair.
var ErrNotFound = keyring.ErrNotFound

func keyFor(profile, baseURL string) string {
	h := sha1.New()
	h.Write([]byte(baseURL))
	return fmt.Sprintf("%s:%s", profile, hex.EncodeToString(h.Sum(nil)))
}

// SaveAPIKey stores a SABnzbd API key in the OS keyring.
func SaveAPIKey(profile, baseURL, apiKey string) error {
	return keyring.Set(keyringService, keyFor(profile, baseURL), apiKey)
}

// LoadAPIKey retrieves the API key from the keyring.
func LoadAPIKey(profile, baseURL string) (string, error) {
	return keyring.Get(keyringService, keyFor(profile, baseURL))
}

// DeleteAPIKey removes the API key from the keyring (primarily for logout/testing).
func DeleteAPIKey(profile, baseURL string) error {
	return keyring.Delete(keyringService, keyFor(profile, baseURL))
}
