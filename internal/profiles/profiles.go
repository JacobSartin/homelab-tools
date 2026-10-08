// Package profiles selects the formatting rules for each YAML document.
package profiles

import (
	"go.yaml.in/yaml/v3"

	"github.com/JacobSartin/homelab-tools/internal/rules"
)

// For returns the rules for one document of a file; documents without a known
// profile get none.
func For(fileText string, doc *yaml.Node) []rules.Rule {
	if IsAppTemplateValues(fileText) {
		return appTemplate
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil
	}
	var apiVersion, kind string
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		switch root.Content[i].Value {
		case "apiVersion":
			apiVersion = root.Content[i+1].Value
		case "kind":
			kind = root.Content[i+1].Value
		}
	}
	if apiVersion == "" || kind == "" {
		return nil
	}
	return kubernetesRules(apiVersion, kind)
}
