# Revision History

Revision History adds a compact **Latest revision** summary to page details. The entire summary is clickable and opens the color-coded comparison for that exact revision. The widget shows the author, revision number, timestamp, and added/removed line counts. Metadata-only revisions display changed field names when Kumbuka supplies them.

The plugin owns its compact layout in `assets/revision-history.css`, contributed through the existing `content-style` module. Kumbuka sanitizes the stylesheet and permits only safe declarations under `.prose`; plugin-specific class names scope these rules to this widget. The host still supplies the accessible full-card comparison dialog action. No revision-specific styles are required in Kumbuka core.

The plugin also contributes a **Revision history** page action that opens the full revision list.

Kumbuka core owns revision persistence, metadata snapshots, restore authorization, diffs, and the comparison dialog. Older revisions without snapshots remain Markdown-only. The plugin owns the optional page-details presentation.

## Permissions

- `pages:read` reads authorized public revision metadata for the current page.
