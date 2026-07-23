package main

import (
	"os"

	"github.com/latif-essam/app-dev-clean/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], version))
}
