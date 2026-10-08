package profiles

import (
	"regexp"

	"github.com/JacobSartin/homelab-tools/internal/rules"
)

// Only files that declare the bjw-s app-template schema opt in; new chart
// versions match automatically and other charts' values stay untouched.
var appTemplateSchema = regexp.MustCompile(
	`(?m)^#\s*yaml-language-server:\s*\$schema=.*bjw-s(?:-labs)?/helm-charts/app-template[^\r\n]*/values\.schema\.json`,
)

// IsAppTemplateValues reports whether a file declares the app-template values schema.
func IsAppTemplateValues(text string) bool { return appTemplateSchema.MatchString(text) }

const container = "controllers.*.containers.*"

var appTemplate = []rules.Rule{
	{
		Path:     "",
		SortRest: true,
		Order: []string{
			"global", "defaultPodOptions", "controllers", "service", "ingress", "route",
			"persistence", "configMaps", "secrets", "serviceAccount", "rbac", "rawResources",
		},
		BlankAfter: []string{"persistence"},
	},
	{
		Path:     "controllers.*",
		SortRest: true,
		Order: []string{
			"enabled", "type", "forceRename", "annotations", "labels", "replicas", "strategy",
			"rollingUpdate", "podDisruptionBudget", "revisionHistoryLimit", "pod",
			"initContainers", "containers",
		},
	},
	{
		Path:     "controllers.*.pod",
		SortRest: true,
		Order: []string{
			"annotations", "labels", "serviceAccountName", "nodeSelector", "affinity",
			"tolerations", "securityContext", "terminationGracePeriodSeconds",
		},
	},
	{
		Path:     container,
		SortRest: true,
		Order: []string{
			"image", "command", "args", "env", "envFrom", "ports", "probes", "securityContext",
			"resources",
		},
		BlankAfter: []string{"image", "probes", "resources"},
	},
	{
		Path:     container + ".image",
		SortRest: true,
		Order:    []string{"repository", "tag", "pullPolicy"},
		Spacing:  rules.None,
	},
	{Path: container + ".resources", SortRest: true, Order: []string{"requests", "limits"}},
	{
		Path:     container + ".probes",
		SortRest: true,
		Order:    []string{"startup", "readiness", "liveness"},
		Spacing:  rules.None,
	},
	{Path: container + ".probes.*", SortRest: true, Order: []string{"enabled", "custom", "spec"}},
	{
		Path:     "service.*",
		SortRest: true,
		Order: []string{
			"enabled", "forceRename", "controller", "type", "loadBalancerIP", "loadBalancerClass",
			"loadBalancerSourceRanges", "allocateLoadBalancerNodePorts", "externalTrafficPolicy",
			"annotations", "labels", "ports",
		},
		Spacing: rules.None,
	},
	{Path: "persistence", Spacing: rules.All},
	{
		Path:     "persistence.*",
		SortRest: true,
		Order: []string{
			"enabled", "type", "identifier", "name", "existingClaim", "storageClass", "accessMode",
			"size", "retain", "hostPath", "server", "path", "medium", "sizeLimit", "globalMounts",
			"advancedMounts",
		},
		Spacing: rules.None,
	},
	{Path: "**.labels", SortRest: true},
	{Path: "**.annotations", SortRest: true},
}
