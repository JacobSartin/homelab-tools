import assert from "node:assert/strict";
import { test } from "node:test";
import { goRegExp, isSopsEncrypted, requiresEncryption, sopsPathRules } from "../lib/sops.ts";

const encrypted = `apiVersion: v1
kind: Secret
stringData:
  token: ENC[AES256_GCM,data:abc,type:str]
sops:
  age:
    - recipient: age1example
  mac: ENC[AES256_GCM,data:def,type:str]
  version: 3.13.3
`;

test("reads path rules and skips rules without a path", () => {
  const rules = sopsPathRules(`creation_rules:
  - path_regex: ^clusters.*kubernetes.*\\.secret.ya?ml
    age: age1example
  - path_regex: (?i)clusterenv.yaml
  - age: age1catchall
`);
  assert.equal(rules.length, 2);
  assert.ok(requiresEncryption("clusters/main/kubernetes/app/config.secret.yaml", rules));
  assert.ok(requiresEncryption("clusters/main/ClusterEnv.yaml", rules));
  assert.ok(!requiresEncryption("clusters/main/kubernetes/app/session-secrets.yaml", rules));
});

test("treats .sops.yaml names as secrets without any rules", () => {
  assert.ok(requiresEncryption("apps/mise/mise.secret.sops.yaml", []));
  assert.ok(requiresEncryption("apps/x/credentials.sops.yml", []));
  assert.ok(!requiresEncryption(".sops.yaml", []));
  assert.ok(!requiresEncryption("apps/x/values.yaml", []));
});

test("translates the Go case-insensitive flag", () => {
  assert.ok(goRegExp("(?i)secret").test("SECRET"));
  assert.ok(!goRegExp("secret").test("SECRET"));
});

test("accepts encrypted documents and rejects plaintext", () => {
  assert.ok(isSopsEncrypted(encrypted));
  assert.ok(isSopsEncrypted(`# comment\n---\n${encrypted}---\n${encrypted}`));
  assert.ok(!isSopsEncrypted(encrypted.replace(/^sops:[\s\S]*/m, "")));
  assert.ok(!isSopsEncrypted(`${encrypted}---\napiVersion: v1\nkind: Secret\n`));
  assert.ok(!isSopsEncrypted(encrypted.replace("mac: ENC[", "mac: plain[")));
  assert.ok(!isSopsEncrypted(""));
});
