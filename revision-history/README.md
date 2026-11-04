# Revision History

Revision History adds a **Latest revision** summary to page details and contributes a **View changes** action and a page action that open Kumbuka's full revision-history fragment in a dialog.

Kumbuka core continues to own revision persistence, metadata snapshots, restore authorization, diffs, and the full history endpoint. Older revisions without snapshots remain Markdown-only. The plugin owns the optional page-details presentation.

## Permissions

- `pages:read` reads authorized public revision metadata for the current page.
