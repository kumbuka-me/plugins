# Mermaid development

`mermaid/browser-build.sh` builds two **self-contained** assets from the exact pinned Mermaid **13.0.0-alpha.3** and esbuild versions:

- `mermaid/assets/headless.js`, a DOM-free renderer for Kumbuka's embedded pure-Go JavaScript runtime. The SDK's artifact contract stores its SVG output in PostgreSQL for live pages or writes files during static-site builds.
- `mermaid/assets/plugin.js`, a classic browser script used only for diagrams explicitly marked interactive. It does not require separate ESM requests or cross-origin headers inside Kumbuka's sandbox.

From the plugin repository root, run `make browser`, `make test-browser`, `make test`, or `make build-plugin PLUGIN=mermaid`. `scripts/build/browser.sh` delegates to the Mermaid plugin's specialized bundler; other browser plugins continue using the shared TypeScript compiler.

`mermaid/update-assets.sh` forces a rebuild by installing pinned dependencies into a disposable temporary directory. `browser-build.sh` checks both generated assets and their source timestamps to avoid unnecessary work. Generated JavaScript is Git-ignored but included in the `.kumbukaplugin` release. End users need no Node.js, Playwright, Chromium, or Mermaid CLI.

Before releasing, run the plugin's DOM-free rendering test **and** verify the packaged `headless.js` with the host's embedded JavaScript engine and SDK SVG validator. Mermaid v13 is experimental, and changes to its output can invalidate the SVG safety policy or runtime compatibility.
