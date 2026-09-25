# Includes

Includes transcludes Markdown from another Kumbuka page before the normal Markdown renderer runs.

## Usage

Include a complete page:

```markdown
{{include:operations/shared-warning}}
```

Or select one ATX heading and its descendants through the next heading at the same or a higher level:

```markdown
{{include:operations/postgres#Restore from backup}}
```

Includes may nest. Kumbuka limits nesting depth and rejects recursive include chains. Include syntax inside fenced code blocks stays literal.

Rendered includes show a breadcrumb before the transcluded content so readers can see which page, and for section includes which heading, the content came from.

Variables and Snippets are processed after Includes, so macros already present in an included page can resolve normally. Content inserted later by Variables or Snippets is not rescanned as an Include.

The plugin receives page Markdown only through Kumbuka's authorized `pages.content` capability. It cannot bypass the current render scope's page access policy.

## Visual editor

In Visual mode, include macros render as compact Include references. Select a reference to edit the target page or heading while keeping the saved Markdown syntax unchanged.
