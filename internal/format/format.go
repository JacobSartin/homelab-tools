// Package format applies key-order and blank-line rules to YAML text.
//
// Rules move whole lines, the way an editor would: comments travel with the
// key below them, blank lines stay where they were, and scalars, quoting and
// indentation are never re-printed. Layout (indentation, wrapping, quotes)
// is left to oxfmt.
package format

import (
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JacobSartin/homelab-tools/internal/profiles"
	"github.com/JacobSartin/homelab-tools/internal/rules"
)

// Result is a formatted file.
type Result struct {
	Output   string
	Warnings []string
}

var sopsKey = regexp.MustCompile(`(?m)^sops:\s*$`)

// IsSOPS reports whether a file is SOPS-encrypted; such files are never
// rewritten because their MAC covers the document structure.
func IsSOPS(path, text string) bool {
	return strings.Contains(filepath.Base(path), ".sops.") || sopsKey.MatchString(text)
}

// Format applies the profile rules to every document of a YAML file. It fails
// rather than return output that parses to different data.
func Format(path, text string) (Result, error) {
	return formatWith(path, text, profiles.For)
}

// RulesFor picks the rules for one document of a file.
type RulesFor func(fileText string, doc *yaml.Node) []rules.Rule

func formatWith(path, text string, rulesFor RulesFor) (Result, error) {
	if IsSOPS(path, text) {
		return Result{Output: text}, nil
	}
	crlf := strings.Contains(text, "\r\n")
	src := strings.ReplaceAll(text, "\r\n", "\n")
	before, err := decode(src)
	if err != nil {
		return Result{}, err
	}
	var warnings []string
	blocked := map[string]bool{}
	// Each pass fixes the first mapping that breaks a rule and re-parses, so
	// line numbers are always current.
	for pass := 0; ; pass++ {
		if pass > 10000 {
			return Result{}, errors.New("formatting did not settle")
		}
		docs, err := parse(src)
		if err != nil {
			return Result{}, err
		}
		next, ok := nextEdit(src, docs, rulesFor, blocked)
		if !ok {
			break
		}
		if _, err := decode(next.text); err != nil {
			if !next.moves {
				return Result{}, err
			}
			// Reordering moved an alias above its anchor.
			blocked[next.mapping] = true
			warnings = append(warnings,
				next.path+": left unsorted because an alias would come before its anchor")
			continue
		}
		src = next.text
	}
	after, err := decode(src)
	if err != nil || !reflect.DeepEqual(before, after) {
		return Result{}, errors.New("formatting would change the parsed data")
	}
	if crlf {
		src = strings.ReplaceAll(src, "\n", "\r\n")
	}
	return Result{Output: src, Warnings: warnings}, nil
}

func parse(text string) ([]*yaml.Node, error) {
	var docs []*yaml.Node
	dec := yaml.NewDecoder(strings.NewReader(text))
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); errors.Is(err, io.EOF) {
			return docs, nil
		} else if err != nil {
			return nil, err
		}
		docs = append(docs, &doc)
	}
}

func decode(text string) ([]any, error) {
	var values []any
	dec := yaml.NewDecoder(strings.NewReader(text))
	for {
		var value any
		if err := dec.Decode(&value); errors.Is(err, io.EOF) {
			return values, nil
		} else if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
}

type edit struct {
	text string
	path string
	// mapping identifies the edited mapping as document index and key path.
	mapping string
	moves   bool
}

func nextEdit(text string, docs []*yaml.Node, rulesFor RulesFor, blocked map[string]bool) (edit, bool) {
	lines := strings.Split(text, "\n")
	for d, doc := range docs {
		if len(doc.Content) == 0 {
			continue
		}
		found, ok := visit(lines, doc.Content[0], nil, rulesFor(text, doc), true,
			func(path []string) bool { return blocked[strconv.Itoa(d)+":"+strings.Join(path, ".")] })
		if ok {
			found.mapping = strconv.Itoa(d) + ":" + found.mapping
			return found, true
		}
	}
	return edit{}, false
}

func visit(lines []string, node *yaml.Node, path []string, set []rules.Rule, top bool,
	blocked func([]string) bool) (edit, bool) {
	switch node.Kind {
	case yaml.MappingNode:
		if e, ok := fix(lines, node, path, set, top, blocked); ok {
			return e, true
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			child := append(slices.Clip(path), node.Content[i].Value)
			if e, ok := visit(lines, node.Content[i+1], child, set, false, blocked); ok {
				return e, true
			}
		}
	case yaml.SequenceNode:
		for i, item := range node.Content {
			child := append(slices.Clip(path), strconv.Itoa(i))
			if e, ok := visit(lines, item, child, set, false, blocked); ok {
				return e, true
			}
		}
	}
	return edit{}, false
}

// fix returns the edit that makes one mapping follow its rules, if it needs one.
func fix(lines []string, node *yaml.Node, path []string, set []rules.Rule, top bool,
	blocked func([]string) bool) (edit, bool) {
	var matching []rules.Rule
	for _, rule := range set {
		if rules.Match(rule.Path, path) {
			matching = append(matching, rule)
		}
	}
	if len(matching) == 0 {
		return edit{}, false
	}
	m, ok := layout(lines, node, top)
	if !ok {
		return edit{}, false
	}
	name := strings.Join(path, ".")
	if name == "" {
		name = "(root)"
	}
	keys := m.keys()
	gaps := m.gaps(lines)
	if i := slices.IndexFunc(matching, rules.Rule.Orders); i >= 0 && !blocked(path) {
		if sorted := matching[i].Sorted(keys); !slices.Equal(sorted, keys) {
			text := strings.Join(m.replace(lines, m.render(lines, sorted, gaps)), "\n")
			return edit{text: text, path: name, mapping: strings.Join(path, "."), moves: true}, true
		}
	}
	spaced := slices.Clone(gaps)
	for _, rule := range matching {
		for i := 1; i < len(keys); i++ {
			blank := slices.ContainsFunc(spaced[i], func(l string) bool { return indentOf(l) < 0 })
			switch {
			case slices.Contains(rule.BlankAfter, keys[i-1]) || rule.Spacing == rules.All:
				if !blank {
					spaced[i] = append([]string{""}, spaced[i]...)
				}
			case rule.Spacing == rules.None:
				spaced[i] = slices.DeleteFunc(slices.Clone(spaced[i]),
					func(l string) bool { return indentOf(l) < 0 })
			}
		}
	}
	if slices.EqualFunc(gaps, spaced, slices.Equal) {
		return edit{}, false
	}
	return edit{text: strings.Join(m.replace(lines, m.render(lines, keys, spaced)), "\n"), path: name}, true
}
