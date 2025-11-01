package main

import (
	"os"

	"github.com/sabx/sabx/cmd/sabx/root"
)

func main() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
