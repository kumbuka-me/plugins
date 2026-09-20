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

## Bump all plugins at once

```sh
make version-all BUMP=patch
git add '*/plugin.yaml'
git commit -m "chore: bump plugin versions"
make test lint build
make tag-all
git push origin main --tags
make release-missing
make check-releases
```

GitHub does not trigger tag push workflows when more than three tags are pushed
at once, so the explicit dispatch step is required for a bulk release. See
[GitHub’s push event documentation](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#push).

`release-missing` dispatches releases for the latest remote plugin tags that do
not have a release yet. Run it after pushing the tags so it selects the new
versions. `check-releases` reports any releases still pending or missing.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
