package main

import (
	"os"

	"github.com/aisk/ego/transpiler"
)

func main() {
	err := transpiler.Transpile(os.Stdin, os.Stdout)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
}
