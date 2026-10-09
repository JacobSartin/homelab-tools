package files

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func setup(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, path := range []string{
		"apps/a/values.yaml", "apps/a/ks.yml", "apps/a/README.md", "ignored/values.yaml",
		"nested/values.yaml",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("a: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{root, filepath.Join(root, "nested")} {
		if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	return root
}

func TestDiscoverSkipsIgnoredFilesAndNestedRepositories(t *testing.T) {
	got, err := Discover(setup(t))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"apps/a/ks.yml", "apps/a/values.yaml"}; !slices.Equal(got, want) {
		t.Errorf("got %q", got)
	}
}

func TestSelectKeepsExistingYAMLInsideRoot(t *testing.T) {
	root := setup(t)
	got := Select(root, []string{
		filepath.Join(root, "apps/a/values.yaml"),
		"apps/a/values.yaml",
		"apps/a/README.md",
		"apps/missing/values.yaml",
		"../outside/values.yaml",
	})
	if want := []string{"apps/a/values.yaml"}; !slices.Equal(got, want) {
		t.Errorf("got %q", got)
	}
}
