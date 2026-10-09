package templates

import (
	"fmt"
	"io/fs"
	"strings"
)

// reads every *.yaml file in fsys and resolves extends chains.
func Load(fsys fs.FS) (*Set, error) {
	names, err := fs.Glob(fsys, "*.yaml")
	if err != nil {
		return nil, err
	}
	set := &Set{}
	for _, name := range names {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		t, err := parse(name, strings.ReplaceAll(string(data), "\r\n", "\n"))
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", name, err)
		}
		set.templates = append(set.templates, t)
	}
	for _, t := range set.templates {
		if err := set.resolve(t, nil); err != nil {
			return nil, fmt.Errorf("template %s: %w", t.Name, err)
		}
		for _, other := range set.templates {
			if other != t && other.Name < t.Name && sameSelector(t, other) {
				return nil, fmt.Errorf("templates %s and %s select the same documents", other.Name, t.Name)
			}
		}
	}
	return set, nil
}
