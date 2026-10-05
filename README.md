# Homelab tools

Requires Node.js 24 or newer. Keep this checkout beside the consumer
repositories so local file dependencies resolve:

```text
homelab/
  homelab-tools/
  cluster/
  media-cluster/
  shared-cluster/
```

The shared implementation is TypeScript. Build the tooling checkout first:

```sh
cd homelab-tools
npm ci
npm run build
npm run typecheck
```

Then, from a consumer repository root:

```sh
npm ci
npm run format:values
npm run check:values
```

Keep the bjw-s app-template schema comment when adding a values file;
it opts the file into formatting. Formatting does not validate Helm schemas.

The consumers use `file:../homelab-tools` with `install-links=true` in `.npmrc`.
They install package copies containing compiled JavaScript. Node does not strip
TypeScript types inside installed packages.

Shared source and settings:

- `lib/values-config.ts`: file discovery, ESLint key order, and layout rules.
- `bin/format-values.ts`: command that runs ESLint followed by oxfmt.
- `oxfmt.config.json`: spacing and wrapping settings.
- `package.json`: shared tool versions.

After editing this package, run `npm run build` here and `npm ci` in each consumer
to refresh its installed copy.
After changing dependencies or the package version, regenerate the tooling
and consumer lockfiles and run each consumer's check command.

Once this repository has a Git remote or a registry release, replace each local
file dependency with a pinned Git commit/tag or package version and regenerate
its lockfile. Until then, standalone checkouts need the sibling tooling repository.
