package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExitCodes(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ordered.yaml", "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n")
	write("unordered.yaml", "kind: ConfigMap\napiVersion: v1\nmetadata:\n  name: x\n")
	write("invalid.yaml", "a: [\n")
	t.Chdir(dir)

	for _, tc := range []struct {
		name string
		args []string
		want int
	}{
		{"formatted file", []string{"--check", "ordered.yaml"}, exitOK},
		{"file needs formatting", []string{"--check", "unordered.yaml"}, exitUnformatted},
		{"runtime error wins over needs formatting", []string{"--check", "unordered.yaml", "invalid.yaml"}, exitRuntime},
		{"unknown flag", []string{"--nope"}, exitUsage},
		{"explain without paths", []string{"explain"}, exitUsage},
		{"explain invalid YAML", []string{"explain", "invalid.yaml"}, exitRuntime},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := run(tc.args, io.Discard, io.Discard); got != tc.want {
				t.Errorf("exit %d, want %d", got, tc.want)
			}
		})
	}
}
