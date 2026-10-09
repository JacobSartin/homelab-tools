package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/JacobSartin/homelab-tools/internal/files"
	"github.com/JacobSartin/homelab-tools/internal/format"
	"github.com/JacobSartin/homelab-tools/internal/templates"
)

func explain(paths []string, stdout, stderr io.Writer) int {
	if len(paths) == 0 {
		fmt.Fprint(stderr, usage)
		return exitError
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitError
	}
	errored := false
	for _, file := range files.Select(root, paths) {
		data, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			fmt.Fprintln(stderr, err)
			errored = true
			continue
		}
		text, err := format.Explain(file, string(data), templates.Default)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", file, err)
			errored = true
			continue
		}
		fmt.Fprint(stdout, text)
	}
	if errored {
		return exitError
	}
	return 0
}
