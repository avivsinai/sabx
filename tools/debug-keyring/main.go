package main

import (
	"fmt"
	"os"

	"github.com/99designs/keyring"
)

func main() {
	cfg := keyring.Config{
		ServiceName:     "sabx-debug",
		AllowedBackends: []keyring.BackendType{keyring.KeychainBackend},
	}

	fmt.Printf("Attempting to open keyring with backend: %v\n", cfg.AllowedBackends)

	kr, err := keyring.Open(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening keyring: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully opened keyring!\n")

	// Try to set a test key
	err = kr.Set(keyring.Item{
		Key:  "test-key",
		Data: []byte("test-value"),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting test key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully wrote test key to keyring!")

	// Try to read it back
	item, err := kr.Get("test-key")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading test key: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully read test key: %s\n", string(item.Data))

	// Clean up
	_ = kr.Remove("test-key")
	fmt.Println("Test completed successfully!")
}
