# Plugin previews

`make previews` regenerates every plugin's canonical `assets/preview.png` from the current plugin checkout without starting the PostgreSQL-backed Kumbuka server.

The runner builds every local `.kumbukaplugin`, seeds those packages into an isolated Kumbuka CLI cache, renders each plugin README as a small static Kumbuka site, and captures one preview per plugin with Playwright.

For Markdown-facing plugins, the preview generator uses the first fenced `markdown` example from the README as a live rendered example when it is safe to render offline. Plugins that require application state, stored configuration, or server-only capabilities fall back to their rendered README. This keeps the preview source close to the documentation instead of maintaining separate preview fixtures.

The temporary static site also creates minimal support pages for wiki-link targets referenced by plugin READMEs. These pages exist only during preview generation so documentation examples such as `[[Page]]` satisfy the CLI's static-link validation without adding permanent fixtures to the repository.

The Kumbuka CLI version is pinned in the repository Makefile. The runner downloads the matching release binary and verifies the published checksum. Local plugin packages are supplied through the CLI's normal validated plugin cache, so previews never use published plugin packages in place of the current checkout.

Run:

```sh
make previews
```

To skip Playwright's Chromium installation when the browser is already available in its cache:

```sh
PREVIEW_SKIP_BROWSER_INSTALL=1 make previews
```

To use an installed Chrome channel instead of Playwright Chromium (which also skips the Chromium download):

```sh
PREVIEW_BROWSER_CHANNEL=chrome make previews
```

To verify generated previews are current:

```sh
make previews-check
```
