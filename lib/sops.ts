import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { parse } from "yaml";

// Files named for SOPS must be encrypted even where no creation rule covers them.
const sopsName = /[^/]\.sops\.ya?ml$/;

// SOPS creation rules use Go regular expressions; translate the leading
// case-insensitive flag, which JavaScript only accepts as a RegExp flag.
export function goRegExp(pattern: string): RegExp {
  return pattern.startsWith("(?i)") ? new RegExp(pattern.slice(4), "i") : new RegExp(pattern);
}

export function sopsPathRules(configText: string): RegExp[] {
  const config = parse(configText) as { creation_rules?: { path_regex?: unknown }[] } | null;
  return (config?.creation_rules ?? []).flatMap((rule) =>
    typeof rule?.path_regex === "string" ? [goRegExp(rule.path_regex)] : [],
  );
}

export function loadSopsPathRules(root: string): RegExp[] {
  const path = join(root, ".sops.yaml");
  return existsSync(path) ? sopsPathRules(readFileSync(path, "utf8")) : [];
}

// Paths are relative to the repository root with forward slashes, as SOPS matches them.
export function requiresEncryption(path: string, rules: RegExp[]): boolean {
  return sopsName.test(path) || rules.some((rule) => rule.test(path));
}

// Encrypted YAML carries top-level SOPS metadata with an encrypted MAC in every document.
export function isSopsEncrypted(text: string): boolean {
  const documents = text
    .split(/^---[ \t]*$/m)
    .filter((document) => /^[^#\s]/m.test(document));
  return (
    documents.length > 0 &&
    documents.every(
      (document) => /^sops:[ \t]*$/m.test(document) && /^[ \t]+mac:[ \t]*ENC\[/m.test(document),
    )
  );
}
