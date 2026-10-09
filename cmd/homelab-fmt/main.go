// Command homelab-fmt applies key-order and blank-line rules to the YAML in
// the homelab GitOps repositories.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/JacobSartin/homelab-tools/internal/files"
	"github.com/JacobSartin/homelab-tools/internal/format"
	"github.com/JacobSartin/homelab-tools/internal/templates"
)

// set by the release build.
var version = "dev"

const usage = `Usage: homelab-fmt [--check] [path...]
       homelab-fmt explain path...

Orders keys and blank lines in app-template values and Kubernetes manifests.
Without paths, formats every YAML file Git does not ignore. Paths that are not
YAML files are skipped. Layout is left to oxfmt.

explain prints the template each document matches and the rule for each mapping.

Exit status: 1 when --check finds files that need formatting, 2 on errors.

`

const (
	exitUnformatted = 1
	exitError       = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "explain" {
		return explain(args[1:], stdout, stderr)
	}
	flags := flag.NewFlagSet("homelab-fmt", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprint(stderr, usage)
		flags.PrintDefaults()
	}
	var checkMode, showVersion bool
	flags.BoolVar(&checkMode, "check", false, "report files that need formatting instead of writing them")
	flags.BoolVar(&showVersion, "version", false, "print the version")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return exitError
	}
	if showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitError
	}
	paths := flags.Args()
	targets := files.Select(root, paths)
	if len(paths) == 0 {
		if targets, err = files.Discover(root); err != nil {
			fmt.Fprintf(stderr, "listing files with git: %v\n", err)
			return exitError
		}
	}

	// TODO(#2): track affected files the same way in check and write mode so
	// both can report edit statistics.
	errored, unformatted, changed := false, false, 0
	for _, file := range targets {
		full := filepath.Join(root, file)
		data, err := os.ReadFile(full)
		if err != nil {
			fmt.Fprintln(stderr, err)
			errored = true
			continue
		}
		result, err := format.Format(file, string(data), templates.For)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", file, err)
			errored = true
			continue
		}
		for _, warning := range result.Warnings {
			fmt.Fprintf(stderr, "%s: %s\n", file, warning)
		}
		if !result.Changed {
			continue
		}
		changed++
		if checkMode {
			fmt.Fprintf(stdout, "needs formatting: %s\n", file)
			unformatted = true
			continue
		}
		if err := os.WriteFile(full, []byte(result.Output), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			errored = true
			continue
		}
		fmt.Fprintf(stdout, "formatted: %s\n", file)
	}
	if len(paths) == 0 {
		verb := "Formatted"
		if checkMode {
			verb = "Checked"
		}
		fmt.Fprintf(stdout, "%s %d files; %d changed.\n", verb, len(targets), changed)
	}
	switch {
	case errored:
		return exitError
	case unformatted:
		return exitUnformatted
	}
	return 0
}
