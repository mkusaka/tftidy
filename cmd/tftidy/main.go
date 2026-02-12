package main

import (
	"os"

	"github.com/mkusaka/tftidy"
)

func main() {
	os.Exit(tftidy.Run(os.Args[1:], os.Stdout, os.Stderr))
}
