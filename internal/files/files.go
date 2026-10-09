// Package files finds the YAML files to format.
package files

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var yamlFile = regexp.MustCompile(`\.ya?ml$`)

// Discover lists tracked and untracked, non-ignored YAML files under root.
// Git leaves out nested repositories and worktrees.
//
// TODO: Discover should not fail when root is outside a Git repository.
func Discover(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z",
		"--", "*.yaml", "*.yml")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var found []string
	for _, path := range strings.Split(string(out), "\x00") {
		if yamlFile.MatchString(path) && exists(filepath.Join(root, path)) {
			found = append(found, path)
		}
	}
	slices.Sort(found)
	return slices.Compact(found), nil
}

// Select narrows explicit paths (absolute or relative to root) to existing
// YAML files inside root, so hooks and editors can pass any file.
func Select(root string, paths []string) []string {
	var found []string
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if yamlFile.MatchString(rel) && exists(path) {
			found = append(found, filepath.ToSlash(rel))
		}
	}
	slices.Sort(found)
	return slices.Compact(found)
}

func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
