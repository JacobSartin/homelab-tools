package profiles

import (
	"slices"
	"strings"

	"github.com/JacobSartin/homelab-tools/internal/rules"
)

// Keys not listed keep their relative order after the listed ones, so
// unfamiliar fields are never shuffled.
var kubernetesCommon = []rules.Rule{
	{
		Path: "",
		Order: []string{
			"apiVersion", "kind", "metadata", "type", "immutable", "spec", "data", "stringData",
			"binaryData",
		},
	},
	{Path: "metadata", Order: []string{"name", "generateName", "namespace", "labels", "annotations"}},
	{Path: "metadata.labels", SortRest: true},
	{Path: "metadata.annotations", SortRest: true},
}

var kustomize = []rules.Rule{
	{
		Path: "",
		Order: []string{
			"apiVersion", "kind", "metadata", "namespace", "resources", "components", "patches",
			"replacements", "images", "generatorOptions", "configMapGenerator", "secretGenerator",
		},
	},
}

// Keyed by API group (without version) and kind.
var kubernetesByKind = map[string][]rules.Rule{
	"helm.toolkit.fluxcd.io/HelmRelease": {
		{
			Path: "spec",
			Order: []string{
				"interval", "chart", "chartRef", "releaseName", "targetNamespace",
				"storageNamespace", "serviceAccountName", "dependsOn", "suspend", "timeout",
				"maxHistory", "driftDetection", "install", "upgrade", "test", "rollback",
				"uninstall",
				// Inline values are merged over valuesFrom.
				"valuesFrom", "values", "postRenderers",
			},
		},
		{Path: "spec.chart.spec", Order: []string{"chart", "version", "sourceRef", "interval"}},
	},
	"kustomize.toolkit.fluxcd.io/Kustomization": {
		{
			Path: "spec",
			Order: []string{
				"dependsOn", "interval", "retryInterval", "timeout", "path", "prune", "sourceRef",
				"wait", "suspend", "targetNamespace", "serviceAccountName", "decryption",
				"postBuild", "components", "patches", "healthChecks", "healthCheckExprs",
			},
		},
	},
	"kustomize.config.k8s.io/Kustomization": kustomize,
	"kustomize.config.k8s.io/Component":     kustomize,
}

func kubernetesRules(apiVersion, kind string) []rules.Rule {
	group, _, _ := strings.Cut(apiVersion, "/")
	// Kind-specific rules come first so they win over the common root order.
	return slices.Concat(kubernetesByKind[group+"/"+kind], kubernetesCommon)
}
