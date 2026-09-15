# Kumbuka Plugins

First-party plugins for Kumbuka. Each top-level plugin directory contains a `plugin.yaml` manifest, a `README.md`, and any Go/WASI or browser assets required by that plugin.

The repository is a single Go module. Executable plugins import the public Kumbuka SDK from `github.com/kumbuka-me/sdk`; declarative plugins contain no unnecessary Go module or guest code.

## Development

Install dependencies and local development tools:

```sh
make download
```

Format, test, and lint the repository:

```sh
make fmt
make test
make lint
```

Build every plugin as a deterministic, versioned package:

```sh
make build
```

Artifacts are written to `dist/` using the manifest version, for example:

```text
dist/tables-1.0.0.kumbukaplugin
dist/tables-1.0.0.kumbukaplugin.sha256
```

Build one plugin with:

```sh
make build-plugin PLUGIN=tables
```

Browser TypeScript is compiled with the repository-pinned TypeScript version before packaging. Vendored third-party assets such as Mermaid remain plugin-owned and are refreshed with their plugin-specific update script.

## Plugin identity

First-party manifests use:

```yaml
provider: Kumbuka
id: me.kumbuka.<plugin-directory>
```

The manifest `version` is the source of truth for the plugin package version.

## Releases

Plugins are released independently from this monorepo. Tag a release as `<plugin>/v<version>` and make the tag match that plugin's manifest exactly:

```sh
git tag tables/v1.1.0
git push origin tables/v1.1.0
```

The release workflow validates the tag against `tables/plugin.yaml`, runs the repository tests, builds only that plugin, and creates a GitHub Release containing:

```text
tables-1.1.0.kumbukaplugin
tables-1.1.0.kumbukaplugin.sha256
```

Normal pushes and pull requests run all tests, lint checks, and package builds. CI also uploads all built plugin packages as workflow artifacts.

## Local SDK development

For simultaneous SDK and plugin work, use a local Go workspace rather than committing a `replace` directive:

```sh
go work init .
go work use ../kumbuka-sdk
```

`go.work` and `go.work.sum` are ignored. Release builds use the SDK version pinned in `go.mod`.
