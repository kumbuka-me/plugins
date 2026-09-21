# Status Dropdowns

Status Dropdowns adds compact, configurable status dropdowns to Kumbuka pages. On a rendered page, select a new value directly from the status control; Kumbuka sends the choice through its host-mediated plugin command boundary, validates it against the page's current Markdown and configured status set, persists it, and reloads the page.

The plugin is inspired by the status workflow of Confluence apps such as Handy Status while keeping browser code isolated from Kumbuka's DOM, credentials, and storage APIs.

## Usage

Insert a status anywhere normal Markdown text is allowed, including table cells:

```markdown
{{status id="release-readiness" set="workflow" prefix="Release"}}
```

The `id` is the persistent identity of the status. Keep it stable when moving the declaration and make it globally unique across the Kumbuka instance. Two declarations with the same `id` intentionally share one stored value.

Available attributes are:

- `id` — required stable identifier. Letters, numbers, `.`, `_`, `-`, `/`, and `:` are supported.
- `set` — required status-set name.
- `initial` — optional initial value. If omitted, the first status in the set is used.
- `prefix` — optional text displayed before the current value.
- `style` — `solid` (default) or `outline`.

Example inside a table:

```markdown
| Service | Readiness                                                               |
| ------- | ----------------------------------------------------------------------- |
| API     | {{status id="release/api" set="workflow" prefix="API"}}                 |
| Web     | {{status id="release/web" set="workflow" prefix="Web" style="outline"}} |
```

Status declarations inside fenced code blocks or inline code stay literal.

## Change a status

On a rendered page, open the status dropdown and select another value. The dropdown runs inside Kumbuka's isolated browser-module frame. A trusted user change is relayed by Kumbuka to the plugin's existing page-details widget command; plugin JavaScript cannot directly access Kumbuka cookies, DOM, HTTP APIs, or storage.

The command handler does not trust the submitted action alone. It rereads the current page, finds the current declaration, reloads the configured set, and only then accepts a matching status choice before writing plugin storage.

The **Status controls** section in page details remains available as a non-JavaScript fallback for the same actions.

Editor previews deliberately do not submit status commands. Change persisted statuses from the rendered page.

## Built-in status sets

`workflow` is available without configuration:

```text
To do|gray
In progress|blue
Blocked|red
Done|green
```

`approval` is also built in:

```text
Draft|gray
In review|yellow
Approved|green
Rejected|red
```

## Custom status sets

Open **Administration → Plugin settings → Status Dropdowns → Status sets** and add a record. Enter one status per line using `Label|color`.

Example `deployment` set:

```text
Planned|gray
Deploying|blue
Verifying|yellow
Live|green
Rolled back|red
```

Supported colors are `gray`, `blue`, `green`, `yellow`, `orange`, `red`, `purple`, and `teal`. A custom set named `workflow` or `approval` overrides the corresponding built-in set.

## Permissions

- `browser:render` runs the dropdown UI in Kumbuka's isolated browser-module frame.
- `settings:read` reads administrator-managed status sets.
- `storage:read` reads the persisted value for each status ID.
- `storage:write` changes a status after a validated host-mediated command.
- `pages:content` rereads the current page so status commands can be validated against its current Markdown.
