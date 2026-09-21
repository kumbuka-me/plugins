# Kumbuka Plugins

First-party plugins for Kumbuka.

## Install a plugin

1. Download the plugin's `.kumbukaplugin` file from the matching GitHub Release.
2. In Kumbuka, open **Administration → Plugins**.
3. Under **Install a plugin**, drop the package into the upload area or choose the file.
4. Select **Install and enable**.

Bundled plugins already appear on the same page and only need to be enabled when they are disabled.

## Update catalog

Published releases are collected into `catalog.json`. The release workflow regenerates the catalog after a plugin release succeeds, commits the canonical copy to this repository, and synchronizes it to `kumbuka-me/docs` so it is served as `https://kumbuka.me/plugins/catalog.json`.

Configure the `DOCS_REPOSITORY_TOKEN` Actions secret with contents write access to `kumbuka-me/docs`. Catalog entries are published only for non-draft, non-prerelease plugin releases that have both the versioned `.kumbukaplugin` asset and its `.sha256` asset.

Regenerate the catalog locally from existing GitHub releases with:

```sh
GH_TOKEN=$(gh auth token) make catalog
```

## Build a plugin

See the [Kumbuka plugin development guide](https://kumbuka.me/plugins/).

## Release plugins

Use the interactive release wizard for normal releases:

```sh
make release
```

The wizard lists all plugins and lets you select one, several, or all of them. You can apply the same patch, minor, or major bump to every selected plugin, or choose the bump separately for each plugin. Before changing Git history it shows the complete release plan and asks for confirmation.

After confirmation, the wizard:

1. updates the selected `plugin.yaml` versions;
2. runs `make test` and `make lint`;
3. builds every selected plugin with `make build-plugin`;
4. creates one version commit and one `plugin/vX.Y.Z` tag per plugin;
5. pushes `main`;
6. pushes every release tag separately so GitHub emits a release workflow event for every plugin.

The working tree must be clean and the current branch must be `main`. Set `RELEASE_REF` or `RELEASE_REMOTE` when releasing from a different branch or remote:

```sh
make release RELEASE_REF=main RELEASE_REMOTE=origin
```

The lower-level versioning and tagging targets remain available for automation and recovery:

```sh
make version-plugin PLUGIN=status-dropdowns BUMP=minor
make tag-plugin PLUGIN=status-dropdowns
make release-missing
make check-releases
```

`release-missing` dispatches release workflows for latest remote plugin tags that do not have a release yet. `check-releases` reports releases that are still pending or missing.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
