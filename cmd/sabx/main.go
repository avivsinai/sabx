package main

import (
	"os"

	"github.com/avivsinai/sabx/cmd/sabx/root"
)

func main() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
