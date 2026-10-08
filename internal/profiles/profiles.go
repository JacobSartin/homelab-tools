// Package profiles selects the formatting rules for each YAML document from
// the embedded templates in templates/.
package profiles

import (
	"embed"
	"io/fs"

	"go.yaml.in/yaml/v3"

	"github.com/JacobSartin/homelab-tools/internal/rules"
)

//go:embed templates/*.yaml
var embedded embed.FS

// Default holds the embedded templates.
var Default = func() *Set {
	sub, err := fs.Sub(embedded, "templates")
	if err != nil {
		panic(err)
	}
	set, err := Load(sub)
	if err != nil {
		panic(err)
	}
	return set
}()

// For returns the rules for one document; documents no template matches get none.
func For(doc *yaml.Node) []rules.Rule {
	if t := Default.Match(doc); t != nil {
		return t.Rules
	}
	return nil
}
