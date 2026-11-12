# Mermaid development

`mermaid/browser-build.sh` generates a **single self-contained** `mermaid/assets/plugin.js` browser script from `mermaid/browser.ts`, bundling the pinned Mermaid **13.0.0-alpha.3** with esbuild. The generated script is classic JavaScript so it can execute in Kumbuka's opaque sandbox and on static sites without separate ESM requests or CORS configuration.

From this repository root, use `make browser`, `make test-browser`, or `make build-plugin PLUGIN=mermaid`. The shared `scripts/build/browser.sh` delegates to `browser-build.sh` when a plugin provides one. Other browser plugins continue to use the shared TypeScript compiler.

`mermaid/update-assets.sh` forces a rebuild. It installs the pinned Mermaid and esbuild packages into a temporary directory, rather than modifying the project dependencies. `browser-build.sh` skips rebuilding when the pinned bundle and source files have not changed.

`mermaid/assets/plugin.js` is generated and ignored by Git. It is packaged into the released `.kumbukaplugin`, so neither site visitors nor `kumbuka-cli` users install Node.js, npm, Playwright, Chromium, or Mermaid CLI.

The v13 release is experimental. Use the browser regression tests before publishing an updated plugin.
