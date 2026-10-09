# Callouts

Callouts adds styled note and warning blocks to Kumbuka Markdown pages.

## Usage

<!-- prettier-ignore -->
```markdown
!!! warning
    Back up the database before continuing.
```

Supported kinds are `note`, `info`, `tip`, `success`, `warning`, `danger`, and `error`. Indent the entire body by four spaces (or one tab); blank lines are allowed, and the callout ends at the first non-empty unindented line. The body is rendered as Markdown without a title. Callout syntax inside fenced code blocks stays literal.

## Visual editor

In Visual mode, callouts render as their colored panel with the selected kind and content. Select a callout to edit its type and body or switch to source editing.

## Permissions

This plugin does not request any Kumbuka capabilities.
