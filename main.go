package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aisk/ego/transpiler"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "transpile" {
		// Proxy all commands and args to go binary
		cmd := exec.Command("go", os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Handle transpile command
	args := os.Args[2:]
	if len(args) == 0 {
		// Find all .ego files recursively in current folder
		if err := transpileDirectory("."); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Transpile specific files
		for _, arg := range args {
			if !strings.HasSuffix(arg, ".ego") {
				fmt.Fprintf(os.Stderr, "Error: %s is not a .ego file\n", arg)
				os.Exit(1)
			}
			if err := transpileFile(arg); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
	}
}

func transpileFile(filename string) error {
	input, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer input.Close()

	outputName := strings.TrimSuffix(filename, ".ego") + ".go"
	output, err := os.Create(outputName)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", outputName, err)
	}
	defer output.Close()

	if err := transpiler.Transpile(input, output); err != nil {
		return fmt.Errorf("failed to transpile %s: %w", filename, err)
	}

	fmt.Printf("Transpiled: %s -> %s\n", filename, outputName)
	return nil
}

func transpileDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ego") {
			return transpileFile(path)
		}
		return nil
	})
}
