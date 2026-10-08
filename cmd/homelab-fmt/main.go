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
)

// Set by the release build.
var version = "dev"

const usage = `Usage: homelab-fmt [--check] [path...]

Orders keys and blank lines in app-template values and Kubernetes manifests.
Without paths, formats every YAML file Git does not ignore. Paths that are not
YAML files are skipped. Layout is left to oxfmt.

`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("homelab-fmt", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprint(stderr, usage)
		flags.PrintDefaults()
	}
	check := flags.Bool("check", false, "report files that need formatting instead of writing them")
	showVersion := flags.Bool("version", false, "print the version")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	paths := flags.Args()
	var targets []string
	if len(paths) > 0 {
		targets = files.Select(root, paths)
	} else if targets, err = files.Discover(root); err != nil {
		fmt.Fprintf(stderr, "listing files with git: %v\n", err)
		return 1
	}

	failed, changed := false, 0
	for _, file := range targets {
		full := filepath.Join(root, file)
		data, err := os.ReadFile(full)
		if err != nil {
			fmt.Fprintln(stderr, err)
			failed = true
			continue
		}
		text := string(data)
		result, err := format.Format(file, text)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", file, err)
			failed = true
			continue
		}
		for _, warning := range result.Warnings {
			fmt.Fprintf(stderr, "%s: %s\n", file, warning)
		}
		if result.Output == text {
			continue
		}
		changed++
		if *check {
			fmt.Fprintf(stdout, "needs formatting: %s\n", file)
			failed = true
			continue
		}
		if err := os.WriteFile(full, []byte(result.Output), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			failed = true
			continue
		}
		fmt.Fprintf(stdout, "formatted: %s\n", file)
	}
	if len(paths) == 0 {
		verb := "Formatted"
		if *check {
			verb = "Checked"
		}
		fmt.Fprintf(stdout, "%s %d files; %d changed.\n", verb, len(targets), changed)
	}
	if failed {
		return 1
	}
	return 0
}
