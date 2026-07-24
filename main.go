package main

import (
	"os"

	"github.com/latif-essam/app-dev-clean/internal/cli"
	_ "github.com/latif-essam/app-dev-clean/internal/detectors" // register detectors
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], version))
}
