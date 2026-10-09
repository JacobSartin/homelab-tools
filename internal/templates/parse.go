package templates

import (
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// reads one template's header directives and compiles its rules; extends
// chains are resolved later, once every template is parsed.
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
