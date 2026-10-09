package templates

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"go.yaml.in/yaml/v3"
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

func parse(name, text string) (*Template, error) {
	dec := yaml.NewDecoder(strings.NewReader(text))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	if err := dec.Decode(new(yaml.Node)); err == nil {
		return nil, errors.New("a template holds one document")
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("a template is a mapping")
	}
	t := &Template{Name: name, Text: text}
	for _, d := range directives(headComments(&doc)...) {
		verb, arg, _ := strings.Cut(d, " ")
		switch verb {
		case "extends":
			t.Extends = strings.TrimSpace(arg)
		case "schema":
			t.schema = glob(strings.TrimSpace(arg))
		default:
			return nil, fmt.Errorf("unknown directive %q", d)
		}
	}
	if t.schema == nil {
		apiVersion, kind := selector(&doc)
		if apiVersion == "" || kind == "" {
			return nil, errors.New(`select documents with "schema" or with apiVersion and kind`)
		}
		t.group, t.kind = group(apiVersion), kind
	}
	root := doc.Content[0]
	if err := t.compile(strings.Split(text, "\n"), root, nil, nil, root.Line); err != nil {
		return nil, err
	}
	t.own = joinVariants(t.own)
	return t, nil
}
