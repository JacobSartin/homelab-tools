#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
import { isAbsolute, join, relative, resolve } from "node:path";
import { isSopsEncrypted, loadSopsPathRules, requiresEncryption } from "../lib/sops.ts";
const root = process.cwd();

// Reject plaintext secrets before they reach Git history. Paths come from the
// hook (staged files) or the command line; without paths, nothing is checked.
const rules = loadSopsPathRules(root);
const plaintext = process.argv
  .slice(2)
  .map((path) => relative(root, resolve(root, path)).replaceAll("\\", "/"))
  .filter(
    (path) =>
      !path.startsWith("../") &&
      !isAbsolute(path) &&
      requiresEncryption(path, rules) &&
      existsSync(join(root, path)) &&
      !isSopsEncrypted(readFileSync(join(root, path), "utf8")),
  );
if (plaintext.length) {
  console.error("These files must be SOPS-encrypted before committing:");
  for (const path of plaintext) console.error(`  ${path}`);
  console.error("Encrypt each with: sops --encrypt --in-place <file>");
  process.exit(1);
}
