package templates

import (
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

var modeline = regexp.MustCompile(`yaml-language-server:\s*\$schema=(\S+)`)

// the most specific template for doc, or nil: a schema match first, then
// apiVersion group and kind, then templates with wildcards.
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

// comments above the document's first key.
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

// "" for the core group (apiVersion "v1").
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

// "*" matches any text, including "/".
func glob(pattern string) *regexp.Regexp {
	return regexp.MustCompile("^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*") + "$")
}
