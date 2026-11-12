# Mermaid development

`mermaid/generate.sh` bundles the **pinned Mermaid v13 alpha** into `mermaid/assets/mermaid.min.js` during plugin builds, and `mermaid/browser.ts` is compiled to `mermaid/assets/plugin.js`. The resulting `.kumbukaplugin` package contains both scripts, so users install only the Kumbuka plugin.

From this repository root, use `make browser`, `make test-browser`, or `make build-plugin PLUGIN=mermaid`. Plugin developers need npm to produce a release; CLI users and site visitors do not.

`mermaid/update-assets.sh` forces a rebuild of the vendored library. It installs its build tools into a temporary directory, not the user's machine. The bundled JavaScript is generated, ignored by Git, and rebuilt from the pinned Mermaid and esbuild versions.
