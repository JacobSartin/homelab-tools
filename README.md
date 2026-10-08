# Homelab tools

`homelab-fmt` keeps the YAML in the homelab GitOps repositories in a
consistent key order and blank-line layout. It is a single static binary with
no runtime. Indentation, wrapping and quoting are left to
[oxfmt](https://oxc.rs/docs/guide/usage/formatter) in the editor.

| Profile | Selected by | Rules |
| --- | --- | --- |
| app-template values | the bjw-s app-template `$schema` modeline | [`internal/profiles/apptemplate.go`](internal/profiles/apptemplate.go) |
| Kubernetes objects | `apiVersion` and `kind` | [`internal/profiles/kubernetes.go`](internal/profiles/kubernetes.go): top-level fields and `metadata` for every kind, plus Flux `HelmRelease`, Flux `Kustomization` and kustomize `Kustomization`/`Component` |

How it edits:

- It moves whole lines, so scalars, quoting, indentation and comments are
  never re-printed. Output that oxfmt accepted stays accepted.
- Comments directly above a key move with it; blank lines and detached
  comments stay where they were. The comment at the top of a document (such
  as the schema modeline) stays on top.
- Keys a profile does not list keep their relative order after the listed ones.
- SOPS files (`*.sops.*`, or a top-level `sops:` key) are never touched.
- A mapping stays unsorted, with a warning, when sorting would move a YAML
  alias above its anchor.
- The result must parse to the same data as the input, or the file is
  reported as an error and left unchanged.

## Usage

```sh
homelab-fmt              # format every YAML file Git does not ignore
homelab-fmt --check      # list files that need formatting; exit 1 if any
homelab-fmt a.yaml b.md  # format only these; non-YAML paths are skipped
```

## Installing in a consumer repository

Releases publish binaries for Linux, macOS and Windows with build provenance
attestations, which mise verifies. Pin a release in the repository's
`mise.toml`:

```toml
[tools]
"github:JacobSartin/homelab-tools" = "1.0.0"
```

To format on save in VS Code, keep oxfmt as the YAML formatter and run
`homelab-fmt` after each save with the
[Run on Save](https://marketplace.visualstudio.com/items?itemName=emeraldwalk.RunOnSave)
extension. VS Code must see mise's shims on `PATH`.

```json
"emeraldwalk.runonsave": {
  "commands": [{ "match": "\\.ya?ml$", "cmd": "homelab-fmt \"${file}\"" }]
}
```

## Layout

- [`cmd/homelab-fmt`](cmd/homelab-fmt): command-line interface.
- [`internal/format`](internal/format): `Format(path, text)`; `mapping.go` maps
  a block mapping to line ranges and re-renders it.
- [`internal/rules`](internal/rules): the rule model (key path, `Order`,
  `SortRest`, `Spacing`, `BlankAfter`), path matching and natural sort.
- [`internal/profiles`](internal/profiles): which rules apply to which document.
- [`internal/files`](internal/files): discovery through `git ls-files` and
  explicit-path selection.

To support another document type, add its rules to a profile and a fixture
pair `internal/format/testdata/<name>.in.yaml` / `<name>.out.yaml`.

## Development

```sh
go test ./...
go run ./cmd/homelab-fmt --check   # from a consumer repository root
```

Pushing a `v*` tag runs [the release workflow](.github/workflows/release.yaml),
which tests, builds with GoReleaser and attests the archives.
