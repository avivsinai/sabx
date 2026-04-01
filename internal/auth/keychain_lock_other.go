//go:build !darwin

package auth

func withKeychainLock(fn func() error) error {
	return fn()
}
