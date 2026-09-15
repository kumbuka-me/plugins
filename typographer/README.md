# Typographer

Typographer converts common ASCII punctuation sequences into typographic characters while Kumbuka renders Markdown.

## Example

```markdown
"Kumbuka" -- documentation...
```

Depending on the text, Typographer can produce curly quotes, typographic dashes, and ellipses. The plugin is disabled by default because these substitutions intentionally change rendered punctuation.

When **Coding Ligatures** is enabled as well, Kumbuka preserves programming-oriented operator sequences such as `-->`, `<<`, and `>>` so the ligature font can render them instead of Typographer consuming them first.

## Permissions

This plugin requests no Kumbuka capabilities. Its WASM module transforms rendered text before Kumbuka's central sanitizer. It leaves code blocks and inline code unchanged and honors the generic `preserve-programming-operators` render policy when another active plugin declares it.
