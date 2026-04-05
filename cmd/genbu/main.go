package main

import (
	"os"

	"github.com/dreadnought-inc/genbu/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
