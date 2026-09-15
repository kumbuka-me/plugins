# Coding Ligatures

Coding Ligatures shows common programming operator sequences as typographic ligatures in rendered Kumbuka content while keeping the Markdown source editor literal.

## Examples

Sequences such as `!=`, `===`, `=>`, `->`, `>=`, `<=`, `:=`, `&&`, and `||` use Fira Code ligatures when the plugin is enabled.

Coding Ligatures also declares the semantic `preserve-programming-operators` render policy. Typographer honors that policy when both plugins are enabled, so programming-oriented punctuation remains available to the ligature font instead of being converted first.

## Permissions

This plugin does not request any Kumbuka capabilities. Its stylesheet is filtered and limited to safe typography properties inside rendered `.prose` content.
