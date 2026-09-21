# Plugin previews

`make previews` regenerates every plugin's canonical `assets/preview.png` from the current checkout without starting the PostgreSQL-backed Kumbuka server.

Every plugin owns a small `preview.md` beside its `README.md`. The README remains the complete documentation; `preview.md` contains only the representative content that belongs in the preview image. Do not add a page title, introduction, permissions, usage explanation, or other documentation copy to `preview.md`.

The runner builds the current `.kumbukaplugin` packages, prepares one isolated static Kumbuka site per plugin, renders that plugin's `preview.md` with the pinned Kumbuka CLI, and captures only the rendered `.prose` element with Playwright. The generated image therefore excludes the Kumbuka page title, breadcrumbs, top bar, navigation sidebar, table of contents, footer, and floating controls.

Some plugins need small static support pages while rendering their preview. These are generated only in the temporary build directory. Includes receives a source page to transclude and Subpages receives a few child pages. They are not additional checked-in fixtures.

Playwright is a pinned development dependency, matching the documentation repository's screenshot setup. `npm ci` installs the Playwright package. Local preview generation installs Playwright's matching Chromium browser when needed:

```sh
make previews
```

The Kumbuka CLI uses an isolated temporary home and cache, but the Playwright install runs in the caller's normal environment. Playwright's normal browser cache is therefore reused between runs instead of downloading Chromium every time.

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
