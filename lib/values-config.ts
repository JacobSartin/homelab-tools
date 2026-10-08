import { existsSync, readdirSync, readFileSync } from "node:fs";
import { basename, isAbsolute, join, relative, resolve } from "node:path";
import yml from "eslint-plugin-yml";
import type { RuleDefinition } from "@eslint/core";
import type { Linter } from "eslint";
import type { YAMLSourceCode } from "eslint-plugin-yml";
import type { AST } from "yaml-eslint-parser";

type YAMLRule = RuleDefinition<{
  LangOptions: Record<string, unknown>;
  Code: YAMLSourceCode;
  RuleOptions: [];
  Visitor: { YAMLMapping(node: AST.YAMLMapping): void };
  Node: AST.YAMLNode;
  MessageIds:
    | "gap"
    | "afterProbes"
    | "afterResources"
    | "afterImage"
    | "afterPersistence"
    | "betweenVolumes";
  ExtRuleDocs: Record<string, unknown>;
}>;

function keyValue(key: AST.YAMLNode | null): string | undefined {
  return key?.type === "YAMLScalar" && typeof key.value === "string" ? key.value : undefined;
}

// Only opt in files that identify the bjw-s app-template schema.
// New chart versions are discovered automatically; other charts remain untouched.
export function isAppTemplateValues(text: string): boolean {
  return /^#\s*yaml-language-server:\s*\$schema=.*bjw-s(?:-labs)?\/helm-charts\/app-template[^\r\n]*\/values\.schema\.json/m.test(
    text,
  );
}

export function appTemplateFiles(dir: string, prefix = ""): string[] {
  return readdirSync(dir, { withFileTypes: true })
    .flatMap((entry) => {
      if (entry.name === ".git" || entry.name === "node_modules") return [];
      const relative = prefix + entry.name;
      if (entry.isDirectory()) return appTemplateFiles(join(dir, entry.name), relative + "/");
      if (!entry.isFile() || entry.name !== "values.yaml") return [];
      return isAppTemplateValues(readFileSync(join(dir, entry.name), "utf8")) ? [relative] : [];
    })
    .sort();
}

// Narrow explicit paths (absolute or relative to root) to app-template values files,
// so hooks can pass every staged file without knowing which ones opt in.
export function selectAppTemplateFiles(root: string, paths: string[]): string[] {
  return [
    ...new Set(
      paths
        .map((path) => relative(root, resolve(root, path)).replaceAll("\\", "/"))
        .filter(
          (path) =>
            basename(path) === "values.yaml" &&
            !path.startsWith("../") &&
            !isAbsolute(path) &&
            existsSync(join(root, path)) &&
            isAppTemplateValues(readFileSync(join(root, path), "utf8")),
        ),
    ),
  ].sort();
}

const alphabetic = { keyPattern: ".*", order: { type: "asc", natural: true } };
const ordered = (pathPattern: string, keys: string[]) => ({
  pathPattern,
  order: [...keys, alphabetic],
  allowLineSeparatedGroups: false,
});
const entry = '(?:\\.[^.\\[]+|\\["[^"\\]]+"\\])';
const controller = "^controllers" + entry;
const container = controller + "\\.containers" + entry;

// Keep named services, volumes, and probe groups compact without touching scalars.
const layout = {
  rules: {
    "compact-settings": {
      meta: {
        type: "layout",
        fixable: "whitespace",
        schema: [],
        messages: {
          gap: "Remove blank lines between settings within this entry.",
          afterProbes: "Add a blank line after the complete probes section.",
          afterResources: "Add a blank line after the complete resources section.",
          afterImage: "Add a blank line after the complete image section.",
          afterPersistence: "Add a blank line after the complete persistence section.",
          betweenVolumes: "Add a blank line between persistence entries.",
        },
      },
      create(context) {
        return {
          YAMLMapping(node) {
            const path: unknown[] = [];
            for (
              let parent: AST.YAMLNode | null | undefined = node.parent;
              parent;
              parent = parent.parent
            ) {
              if (parent.type === "YAMLPair") path.unshift(keyValue(parent.key));
            }
            const inContainer = path[0] === "controllers" && path[2] === "containers";
            if (node.parent?.type === "YAMLDocument") {
              for (let i = 0; i < node.pairs.length - 1; i++) {
                if (keyValue(node.pairs[i].key) !== "persistence") continue;
                const range: [number, number] = [
                  node.pairs[i].range[1],
                  node.pairs[i + 1].range[0],
                ];
                const gap = context.sourceCode.text.slice(...range);
                if (/\r?\n/.test(gap) && !/\r?\n[ \t]*\r?\n/.test(gap))
                  context.report({
                    node: node.pairs[i + 1],
                    messageId: "afterPersistence",
                    fix: (fixer) => fixer.replaceTextRange(range, gap.replace(/\r?\n/, "\n\n")),
                  });
              }
              return;
            }
            if (path.length === 1 && path[0] === "persistence") {
              for (let i = 1; i < node.pairs.length; i++) {
                const range: [number, number] = [
                  node.pairs[i - 1].range[1],
                  node.pairs[i].range[0],
                ];
                const gap = context.sourceCode.text.slice(...range);
                if (!/\r?\n[ \t]*\r?\n/.test(gap))
                  context.report({
                    node: node.pairs[i],
                    messageId: "betweenVolumes",
                    fix: (fixer) => fixer.replaceTextRange(range, "\n" + gap),
                  });
              }
              return;
            }
            if (inContainer && path.length === 4) {
              for (const pair of node.pairs) {
                if (!["resources", "image"].includes(keyValue(pair.key) ?? "")) continue;
                const next = context.sourceCode.getTokenAfter(pair);
                if (!next) continue;
                const range: [number, number] = [pair.range[1], next.range[0]];
                const gap = context.sourceCode.text.slice(...range);
                if (/\r?\n/.test(gap) && !/\r?\n[ \t]*\r?\n/.test(gap))
                  context.report({
                    node: pair,
                    messageId: keyValue(pair.key) === "image" ? "afterImage" : "afterResources",
                    fix: (fixer) => fixer.replaceTextRange(range, gap.replace(/\r?\n/, "\n\n")),
                  });
              }
              for (let i = 0; i < node.pairs.length - 1; i++) {
                if (keyValue(node.pairs[i].key) !== "probes") continue;
                const range: [number, number] = [
                  node.pairs[i].range[1],
                  node.pairs[i + 1].range[0],
                ];
                const gap = context.sourceCode.text.slice(...range);
                if (!/\r?\n[ \t]*\r?\n/.test(gap))
                  context.report({
                    node: node.pairs[i + 1],
                    messageId: "afterProbes",
                    fix: (fixer) => fixer.replaceTextRange(range, "\n" + gap),
                  });
              }
              return;
            }
            const probeGroup = inContainer && path.length === 5 && path[4] === "probes";
            const imageSettings = inContainer && path.length === 5 && path[4] === "image";
            const sectionPair = node.parent?.parent?.parent;
            const namedSettings =
              sectionPair?.type === "YAMLPair" &&
              ["service", "persistence"].includes(keyValue(sectionPair.key) ?? "") &&
              sectionPair.parent?.parent?.type === "YAMLDocument";
            if (!namedSettings && !probeGroup && !imageSettings) return;
            for (let i = 1; i < node.pairs.length; i++) {
              const range: [number, number] = [node.pairs[i - 1].range[1], node.pairs[i].range[0]];
              const gap = context.sourceCode.text.slice(...range);
              const compact = gap.replace(/\r?\n[ \t]*(?=\r?\n)/g, "");
              if (gap !== compact)
                context.report({
                  node: node.pairs[i],
                  messageId: "gap",
                  fix: (fixer) => fixer.replaceTextRange(range, compact),
                });
            }
          },
        };
      },
    } satisfies YAMLRule,
  },
};

export function createValuesConfig(root: string): Linter.Config[] {
  const files = appTemplateFiles(root);
  return files.length
    ? [
        {
          files,
          plugins: { yml, layout },
          language: "yml/yaml",
          rules: {
            "layout/compact-settings": "error",
            "yml/sort-keys": [
              "error",
              ordered("^$", [
                "global",
                "defaultPodOptions",
                "controllers",
                "service",
                "ingress",
                "route",
                "persistence",
                "configMaps",
                "secrets",
                "serviceAccount",
                "rbac",
                "rawResources",
              ]),
              ordered(controller + "$", [
                "enabled",
                "type",
                "forceRename",
                "annotations",
                "labels",
                "replicas",
                "strategy",
                "rollingUpdate",
                "podDisruptionBudget",
                "revisionHistoryLimit",
                "pod",
                "initContainers",
                "containers",
              ]),
              ordered(controller + "\\.pod$", [
                "annotations",
                "labels",
                "serviceAccountName",
                "nodeSelector",
                "affinity",
                "tolerations",
                "securityContext",
                "terminationGracePeriodSeconds",
              ]),
              ordered(container + "$", [
                "image",
                "command",
                "args",
                "env",
                "envFrom",
                "ports",
                "probes",
                "securityContext",
                "resources",
              ]),
              ordered(container + "\\.image$", ["repository", "tag", "pullPolicy"]),
              ordered(container + "\\.resources$", ["requests", "limits"]),
              ordered(container + "\\.probes$", ["startup", "readiness", "liveness"]),
              ordered(container + "\\.probes" + entry + "$", ["enabled", "custom", "spec"]),
              ordered("^service" + entry + "$", [
                "enabled",
                "forceRename",
                "controller",
                "type",
                "loadBalancerIP",
                "loadBalancerClass",
                "loadBalancerSourceRanges",
                "allocateLoadBalancerNodePorts",
                "externalTrafficPolicy",
                "annotations",
                "labels",
                "ports",
              ]),
              ordered("^persistence" + entry + "$", [
                "enabled",
                "type",
                "identifier",
                "name",
                "existingClaim",
                "storageClass",
                "accessMode",
                "size",
                "retain",
                "hostPath",
                "server",
                "path",
                "medium",
                "sizeLimit",
                "globalMounts",
                "advancedMounts",
              ]),
              {
                pathPattern: "(?:^|\\.)(?:labels|annotations)$",
                order: { type: "asc", natural: true },
              },
            ],
          },
        },
      ]
    : [];
}
