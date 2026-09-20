# Simple Icons

Adds the [Simple Icons](https://simpleicons.org/) brand-logo catalog to Kumbuka's normal icon picker.

Icons keep the existing `-simple` identifiers, for example `github-simple`, so pages and navigation items that already use Simple Icons continue to work when this plugin is enabled.

The plugin registers an SDK `icon-resource` module backed by `assets/icons.json`. Kumbuka validates the resource and renders the SVG structure itself; plugin-provided markup is not injected directly into the page.

## Permissions

This plugin requests no Kumbuka capabilities.
