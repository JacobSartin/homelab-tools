package format

import (
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JacobSartin/homelab-tools/internal/rules"
	"github.com/JacobSartin/homelab-tools/internal/templates"
)

// lists the template each document matches and the rule for each of its mappings.
func Explain(path, text string, set *templates.Set) (string, error) {
	if IsSOPS(path, text) {
		return path + ": SOPS file, never formatted\n", nil
	}
	docs, err := parse(strings.ReplaceAll(text, "\r\n", "\n"))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for d, doc := range docs {
		b.WriteString(path + ", document " + strconv.Itoa(d+1) + ": ")
		t := set.Match(doc)
		if t == nil || len(doc.Content) == 0 {
			b.WriteString("no template matches; left as is\n")
			continue
		}
		b.WriteString("template " + strings.Join(t.Chain, " extends ") + "\n")
		walk(doc.Content[0], nil, func(_ *yaml.Node, keys []string) bool {
			if rule, ok := rules.First(t.Rules, keys); ok {
				b.WriteString("  " + pathName(keys) + ": " + rule.Summarize() + "\n")
			}
			return false
		})
	}
	return b.String(), nil
}
