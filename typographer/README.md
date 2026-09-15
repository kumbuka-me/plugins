# Typographer

Typographer converts common ASCII punctuation sequences into typographic characters while Kumbuka renders Markdown.

## Example

```markdown
"Kumbuka" -- documentation...
```

Depending on the text, Typographer can produce curly quotes, typographic dashes, and ellipses. The plugin is disabled by default because these substitutions intentionally change rendered punctuation.

When **Coding Ligatures** is enabled as well, Kumbuka preserves programming-oriented operator sequences such as `-->`, `<<`, and `>>` so the ligature font can render them instead of Typographer consuming them first.

## Permissions

This plugin requests no Kumbuka capabilities. It activates Kumbuka's public `typographer` render policy.
