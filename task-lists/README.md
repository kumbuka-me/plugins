# Task Lists

Task Lists renders GitHub-style Markdown task items with accessible visible checkbox state.

## Usage

```markdown
- [x] Create backup
- [ ] Run upgrade
```

The rendered checkboxes are presentation only; editing the Markdown remains the way to change their state. The plugin contributes the Task list action to the editor's block toolbar; Kumbuka's generic list editor continues list prefixes when Enter is pressed.

## Permissions

This plugin requests no Kumbuka capabilities. Kumbuka supplies only the standard `task-list` Markdown grammar adapter; this plugin's WASM module and filtered content stylesheet own the accessible checkbox presentation. Disabling the plugin therefore removes both parsing and presentation from new renders.

