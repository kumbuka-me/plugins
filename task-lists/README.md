# Checklists

Checklists renders GitHub-style Markdown task items as accessible controls that can be toggled directly in page view mode.

## Usage

```markdown
- [x] Create backup
- [ ] Run upgrade
```

Clicking a checkbox updates the canonical Markdown and creates a normal page revision. The write uses optimistic concurrency, so a stale page cannot silently overwrite a newer edit. The plugin also contributes the Checklist action to the editor's block toolbar; Kumbuka's generic list editor continues list prefixes when Enter is pressed.

## Permissions

This plugin requests sandboxed browser rendering plus page-content read and narrowly scoped page-content write capabilities. The write is available only for the page that emitted the command and preserves all page metadata. Kumbuka supplies the standard `task-list` Markdown grammar adapter; disabling the plugin removes both parsing and presentation from new renders.
