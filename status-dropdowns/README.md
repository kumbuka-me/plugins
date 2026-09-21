# Status Dropdowns

Status Dropdowns adds compact, configurable status dropdowns to Kumbuka pages. Choices can live directly in the page Markdown or come from reusable administrator-managed sets. On a rendered page, selecting a new value goes through Kumbuka's host-mediated plugin command boundary, is validated against the page's current declaration, persisted, and then shown everywhere that uses the same status ID.

The plugin is inspired by the status workflow of Confluence apps such as Handy Status while keeping browser code isolated from Kumbuka's DOM, credentials, and storage APIs.

## Page-local statuses

For ordinary page-specific workflows, put the choices directly in the status declaration:

```markdown
{{status id="release-readiness" options="To do;In progress;Blocked;Done" prefix="Release"}}
```

The `options` list accepts semicolon-separated values. Commas are also accepted when no semicolon is present. When `colors` is omitted, Status Dropdowns assigns a stable default palette.

Specify colors only when the page needs them explicitly:

```markdown
{{status id="risk" options="Low;Medium;High;Critical" colors="#64748b;#2563eb;#ea580c;#dc2626" style="outline"}}
```

Named colors `gray`, `blue`, `green`, `yellow`, `orange`, `red`, `purple`, `teal`, and `pink` are accepted in `colors` as a shorthand for their built-in hexadecimal values.

The `id` is the persistent identity of the status. Keep it stable when moving the declaration and make it globally unique across the Kumbuka instance. Two declarations with the same `id` intentionally share one stored value.

Available attributes are:

- `id` — required stable identifier. Letters, numbers, `.`, `_`, `-`, `/`, and `:` are supported.
- `options` — page-local choices. Use this or `set`, never both.
- `colors` — optional page-local colors parallel to `options`.
- `set` — optional reusable built-in or administrator-managed set name instead of `options`.
- `initial` — optional initial value. If omitted, the first configured status is used.
- `prefix` — optional text displayed before the current value.
- `style` — `solid` (default) or `outline`.

Status declarations inside fenced code blocks or inline code stay literal.

## Reusable status sets

For workflows shared by many pages, open **Administration → Plugin settings → Status Dropdowns → Status sets** and create a set. The status editor presents one row per choice with separate **Status** and **Color** columns; the color uses the browser's color picker. Add and remove rows without writing `Label|color` syntax.

Then reference the reusable set:

```markdown
{{status id="production-deployment" set="deployment" prefix="Production"}}
```

Two built-in sets work without administration setup. `workflow` contains **To do**, **In progress**, **Blocked**, and **Done**; `approval` contains **Draft**, **In review**, **Approved**, and **Rejected**. An administrator-managed set with either name overrides its built-in counterpart.

## Errors

Malformed declarations and invalid reusable sets render a visible inline **Status error** with the concrete reason instead of silently disappearing or becoming a generic unavailable badge. Host-side administrator validation also rejects malformed structured rows and invalid colors before they are stored.

## Change a status

On a rendered page, open the status dropdown and select another value. The dropdown runs inside Kumbuka's isolated browser-module frame. A trusted user change is relayed by Kumbuka to the plugin's page-details widget command; plugin JavaScript cannot directly access Kumbuka cookies, DOM, HTTP APIs, or storage.

The command handler does not trust the submitted action alone. It rereads the current page, finds the current declaration, resolves its current page-local or reusable choices, and only then accepts a matching status choice before writing plugin storage.

The **Status controls** section in page details remains available as a non-JavaScript fallback for the same actions. Editor previews deliberately do not submit status commands; change persisted statuses from the rendered page.

## Permissions

- `browser:render` runs the dropdown UI in Kumbuka's isolated browser-module frame.
- `settings:read` reads administrator-managed reusable status sets.
- `storage:read` reads the persisted value for each status ID.
- `storage:write` changes a status after a validated host-mediated command.
- `pages:content` rereads the current page so status commands can be validated against its current Markdown.
