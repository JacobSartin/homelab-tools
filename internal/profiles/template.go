package profiles

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JacobSartin/homelab-tools/internal/rules"
	"github.com/JacobSartin/homelab-tools/internal/yamltext"
)

// A Template is an example document whose layout defines the rules for the
// documents it matches:
//
//   - Keys are ordered as in the template. Keys it does not list keep their
//     relative order after the listed ones.
//   - A "*" key stands for any key; a sequence's first item stands for every item.
//   - A blank line between two keys requires one after the first key.
//   - "# homelab-fmt: sort" on a key sorts the keys its mapping does not list,
//     "compact" removes blank lines between its entries, and "separate"
//     requires one between them.
//   - Header comments select documents: "schema <glob>" matches the
//     yaml-language-server $schema URL; otherwise the template's apiVersion
//     group and kind match ("*" matches any). "extends <template>" adds a
//     parent's rules for keys and paths the template does not cover.
type Template struct {
	Name    string
	Text    string
	Extends string
	// Chain lists this template followed by the templates it extends.
	Chain []string
	// Rules are this template's rules merged with its parents'.
	Rules []rules.Rule

	schema      *regexp.Regexp
	group, kind string
	own         []rules.Rule
}

// Set is a loaded collection of templates.
type Set struct {
	templates []*Template
}

var (
	directivePrefix = regexp.MustCompile(`^#\s*homelab-fmt:\s*(.*)$`)
	modeline        = regexp.MustCompile(`yaml-language-server:\s*\$schema=(\S+)`)
)

// Load reads every *.yaml template in fsys.
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
		t, err := parseTemplate(name, strings.ReplaceAll(string(data), "\r\n", "\n"))
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

// Templates returns the loaded templates sorted by name.
func (s *Set) Templates() []*Template { return s.templates }

func parseTemplate(name, text string) (*Template, error) {
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
	lines := strings.Split(text, "\n")
	if err := t.compile(lines, doc.Content[0], nil, nil, doc.Content[0].Line); err != nil {
		return nil, err
	}
	return t, nil
}

// compile turns the template's mappings into rules. flags are the directives
// on the key that holds node, and line is that key's line.
func (t *Template) compile(lines []string, node *yaml.Node, path, flags []string, line int) error {
	switch node.Kind {
	case yaml.SequenceNode:
		if len(flags) > 0 {
			return fmt.Errorf("line %d: directives apply to mappings", node.Line)
		}
		if len(node.Content) > 0 {
			return t.compile(lines, node.Content[0], append(slices.Clip(path), "*"), nil, node.Content[0].Line)
		}
		return nil
	case yaml.MappingNode:
	default:
		if len(flags) > 0 {
			return fmt.Errorf("line %d: directives apply to mappings", node.Line)
		}
		return nil
	}
	rule := rules.Rule{Path: strings.Join(path, "."), Source: fmt.Sprintf("%s:%d", t.Name, line)}
	for _, flag := range flags {
		switch flag {
		case "sort":
			rule.SortRest = true
		case "compact":
			rule.Spacing = rules.Compact
		case "separate":
			rule.Spacing = rules.Separate
		}
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if key := node.Content[i].Value; key != "*" {
			rule.Order = append(rule.Order, key)
		}
	}
	if m, ok := yamltext.Layout(lines, node, len(path) == 0); ok {
		gaps := m.Gaps(lines)
		for i := 1; i < len(m.Entries); i++ {
			if prev := m.Entries[i-1].Key; prev != "*" && slices.ContainsFunc(gaps[i], yamltext.IsBlank) {
				rule.BlankAfter = append(rule.BlankAfter, prev)
			}
		}
	}
	if rule.Orders() || rule.Spacing != rules.Keep || len(rule.BlankAfter) > 0 {
		t.own = append(t.own, rule)
	}
	// Named keys come before "*" so their rules win over the wildcard's.
	var wildcard []int
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "*" {
			wildcard = append(wildcard, i)
			continue
		}
		if err := t.compilePair(lines, node, i, path); err != nil {
			return err
		}
	}
	for _, i := range wildcard {
		if err := t.compilePair(lines, node, i, path); err != nil {
			return err
		}
	}
	return nil
}

func (t *Template) compilePair(lines []string, node *yaml.Node, i int, path []string) error {
	key, value := node.Content[i], node.Content[i+1]
	comments := []string{key.LineComment, value.LineComment, value.HeadComment}
	// The first key's head comment is the template header.
	if !(len(path) == 0 && i == 0) {
		comments = append(comments, key.HeadComment)
	}
	var flags []string
	for _, d := range directives(comments...) {
		switch d {
		case "sort", "compact", "separate":
			flags = append(flags, d)
		default:
			return fmt.Errorf("line %d: unknown directive %q", key.Line, d)
		}
	}
	return t.compile(lines, value, append(slices.Clip(path), key.Value), flags, key.Line)
}

// resolve merges the rules of the templates t extends into t.Rules.
func (s *Set) resolve(t *Template, seen []string) error {
	if slices.Contains(seen, t.Name) {
		return fmt.Errorf("extends cycle: %s", strings.Join(append(seen, t.Name), " -> "))
	}
	t.Chain = []string{t.Name}
	t.Rules = slices.Clone(t.own)
	if t.Extends != "" {
		i := slices.IndexFunc(s.templates, func(p *Template) bool { return p.Name == t.Extends })
		if i < 0 {
			return fmt.Errorf("extends unknown template %s", t.Extends)
		}
		parent := s.templates[i]
		if err := s.resolve(parent, append(seen, t.Name)); err != nil {
			return err
		}
		t.Chain = append(t.Chain, parent.Chain...)
		t.Rules = merge(t.own, parent.Rules)
	}
	// Literal paths win over wildcard paths for the same mapping.
	slices.SortStableFunc(t.Rules, func(a, b rules.Rule) int {
		return strings.Count(a.Path, "*") - strings.Count(b.Path, "*")
	})
	return nil
}

// merge layers child rules over parent rules: for the same path, the child's
// keys come first, then the parent's keys the child does not list.
func merge(child, parent []rules.Rule) []rules.Rule {
	merged := slices.Clone(child)
	for _, p := range parent {
		i := slices.IndexFunc(merged, func(c rules.Rule) bool { return c.Path == p.Path })
		if i < 0 {
			merged = append(merged, p)
			continue
		}
		c := &merged[i]
		for _, key := range p.Order {
			if !slices.Contains(c.Order, key) {
				c.Order = append(c.Order, key)
			}
		}
		c.SortRest = c.SortRest || p.SortRest
		if c.Spacing == rules.Keep {
			c.Spacing = p.Spacing
		}
		for _, key := range p.BlankAfter {
			if !slices.Contains(c.BlankAfter, key) {
				c.BlankAfter = append(c.BlankAfter, key)
			}
		}
		c.Source += ", " + p.Source
	}
	return merged
}

// Match returns the most specific template for a document, or nil.
func (s *Set) Match(doc *yaml.Node) *Template {
	var schemas []string
	for _, comment := range headComments(doc) {
		for _, m := range modeline.FindAllStringSubmatch(comment, -1) {
			schemas = append(schemas, m[1])
		}
	}
	apiVersion, kind := selector(doc)
	var best *Template
	bestScore := -1
	for _, t := range s.templates {
		score := -1
		switch {
		case t.schema != nil:
			if slices.ContainsFunc(schemas, t.schema.MatchString) {
				score = 3
			}
		case apiVersion != "" && kind != "" &&
			(t.group == "*" || t.group == group(apiVersion)) && (t.kind == "*" || t.kind == kind):
			score = 0
			if t.group != "*" {
				score++
			}
			if t.kind != "*" {
				score++
			}
		}
		if score > bestScore {
			best, bestScore = t, score
		}
	}
	return best
}

func sameSelector(a, b *Template) bool {
	if a.schema != nil || b.schema != nil {
		return a.schema != nil && b.schema != nil && a.schema.String() == b.schema.String()
	}
	return a.group == b.group && a.kind == b.kind
}

// headComments returns the comments above a document's first key.
func headComments(doc *yaml.Node) []string {
	comments := []string{doc.HeadComment}
	if len(doc.Content) > 0 {
		root := doc.Content[0]
		comments = append(comments, root.HeadComment)
		if root.Kind == yaml.MappingNode && len(root.Content) > 0 {
			comments = append(comments, root.Content[0].HeadComment)
		}
	}
	return comments
}

func directives(comments ...string) []string {
	var found []string
	for _, comment := range comments {
		for _, line := range strings.Split(comment, "\n") {
			m := directivePrefix.FindStringSubmatch(strings.TrimSpace(line))
			if m == nil {
				continue
			}
			for _, d := range strings.Split(m[1], ",") {
				if d = strings.Join(strings.Fields(d), " "); d != "" {
					found = append(found, d)
				}
			}
		}
	}
	return found
}

func selector(doc *yaml.Node) (apiVersion, kind string) {
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return "", ""
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i+1].Kind != yaml.ScalarNode {
			continue
		}
		switch root.Content[i].Value {
		case "apiVersion":
			apiVersion = root.Content[i+1].Value
		case "kind":
			kind = root.Content[i+1].Value
		}
	}
	return apiVersion, kind
}

// group returns the API group of an apiVersion: "" for the core group.
func group(apiVersion string) string {
	if apiVersion == "*" {
		return "*"
	}
	g, _, found := strings.Cut(apiVersion, "/")
	if !found {
		return ""
	}
	return g
}

// glob compiles a pattern where "*" matches any text, including "/".
func glob(pattern string) *regexp.Regexp {
	return regexp.MustCompile("^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*") + "$")
}
