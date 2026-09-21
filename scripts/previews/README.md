# Plugin previews

`make previews` regenerates every plugin's canonical `assets/preview.png` from the current plugin checkout without starting the PostgreSQL-backed Kumbuka server.

The runner builds every local `.kumbukaplugin`, seeds those packages into an isolated Kumbuka CLI cache, renders each plugin README as a small static Kumbuka site, and captures one preview per plugin with Playwright.

For Markdown-facing plugins, the preview generator uses the first fenced `markdown` example from the README as a live rendered example when it is safe to render offline. Plugins that require application state, stored configuration, or server-only capabilities fall back to their rendered README. This keeps the preview source close to the documentation instead of maintaining separate preview fixtures.

The temporary static site also creates minimal support pages for wiki-link targets referenced by plugin READMEs. These pages exist only during preview generation so documentation examples such as `[[Page]]` satisfy the CLI's static-link validation without adding permanent fixtures to the repository.

The Kumbuka CLI version is pinned in the repository Makefile. Local plugin packages are supplied through the CLI's normal validated plugin cache, so previews never use published plugin packages in place of the current checkout.

Playwright is a pinned development dependency in this repository, matching the documentation repository's screenshot setup. `npm ci` installs the Playwright package. Local preview generation installs Playwright's matching Chromium browser when needed:

```sh
make previews
```

CI installs Chromium with its system dependencies once and then skips the runner's browser-install step:

```sh
npm ci
npx playwright install --with-deps chromium
make previews PREVIEW_SKIP_BROWSER_INSTALL=1
```

`PREVIEW_BROWSER_CHANNEL` can select a Playwright browser channel when needed. `PREVIEW_SKIP_BROWSER_INSTALL=1` is intended for environments that already installed the required browser.

To verify generated previews are current:

```sh
make previews-check
```
