import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { after, test } from "node:test";
import { selectAppTemplateFiles } from "../lib/values-config.ts";

const schema =
  "# yaml-language-server: $schema=https://raw.githubusercontent.com/bjw-s-labs/helm-charts/app-template-4.4.0/charts/other/app-template/values.schema.json\n";
const root = mkdtempSync(join(tmpdir(), "homelab-tools-"));
after(() => rmSync(root, { recursive: true, force: true }));

for (const [path, text] of Object.entries({
  "apps/a/values.yaml": schema + "controllers: {}\n",
  "apps/b/values.yaml": "image: {}\n",
  "apps/c/helm-release.yaml": schema,
})) {
  mkdirSync(join(root, path, ".."), { recursive: true });
  writeFileSync(join(root, path), text);
}

test("selects only app-template values files from explicit paths", () => {
  assert.deepEqual(
    selectAppTemplateFiles(root, [
      join(root, "apps/a/values.yaml"),
      "apps/a/values.yaml",
      "apps/b/values.yaml",
      "apps/c/helm-release.yaml",
      "apps/missing/values.yaml",
      "../outside/values.yaml",
    ]),
    ["apps/a/values.yaml"],
  );
});
