#!/usr/bin/env node
import { ESLint } from "eslint";
import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import { openSync, closeSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { appTemplateFiles, createValuesConfig } from "../lib/values-config.ts";
const root = process.cwd();

const check = process.argv.includes("--check");
const files = appTemplateFiles(root);
if (!files.length) {
  console.log("No bjw-s app-template values files found.");
  process.exit(0);
}
const eslint = new ESLint({
  cwd: root,
  fix: !check,
  overrideConfigFile: true,
  overrideConfig: createValuesConfig(root),
});
// Check every destination before concurrent fixes can modify any files.
// Opening with r+ checks write access without truncating the file.
if (!check) {
  for (const file of files) closeSync(openSync(join(root, file), "r+"));
}
const results = await eslint.lintFiles(files);
if (!check) await ESLint.outputFixes(results);
const report = await (await eslint.loadFormatter("stylish")).format(results);
if (report) console.log(report);
const failed = results.some((result) => result.errorCount || result.warningCount);
const require = createRequire(import.meta.url);
const formatter = spawnSync(
  process.execPath,
  [
    join(dirname(require.resolve("oxfmt/package.json")), "bin/oxfmt"),
    "--config",
    fileURLToPath(new URL("../../oxfmt.config.json", import.meta.url)),
    "--disable-nested-config",
    check ? "--check" : "--write",
    ...files,
  ],
  { cwd: root, stdio: "inherit" },
);
console.log(`${check ? "Checked" : "Formatted"} ${files.length} app-template values files.`);
process.exit(failed || formatter.status !== 0 ? 1 : 0);
