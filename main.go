package main

import (
	"os"

	"github.com/tuliofaria/asaas-cli/cmd"
)

// version is overridden at build time:
//
//	go build -ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	os.Exit(cmd.Execute(version))
}
