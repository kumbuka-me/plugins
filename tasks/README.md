# Tasks

Tasks adds persistent actionable checkboxes to Kumbuka pages. A task can carry an assignee and due date, while its completion state is stored separately from the page Markdown so checking a task does not rewrite the page.

## Usage

```markdown
{{task id="deploy-api" text="Deploy API to production" assignee="@alice" due="2026-10-01"}}
```

Only `id` and `text` are required. `id` is the stable identity used for persisted completion state, so keep it unchanged when moving a task. IDs support letters, numbers, `.`, `_`, `-`, `/`, and `:`.

Optional attributes are:

- `assignee` — canonical Kumbuka user mention responsible for the task, such as `@alice`.
- `due` — due date in `YYYY-MM-DD` form.
- `initial` — initial state, either `open` (default) or `done`.

For example:

```markdown
{{task id="review-runbook" text="Review the production runbook" assignee="@alice" due="2026-10-15" initial="done"}}
```

Task declarations inside fenced code blocks or inline code remain literal. Invalid declarations render a visible **Task error** instead of silently disappearing.

## Interaction

On rendered pages, the checkbox is provided by Kumbuka's isolated browser-module boundary. Changes are sent through the host-mediated page-details command handler. Before persisting a change, the plugin rereads the current page and verifies that the task ID still exists in its Markdown.

When an assigned task is completed or reopened, Tasks resolves the canonical mention and creates an attributed Kumbuka in-app notification. Delivery is mutation-only and retry-safe; rendering a page never creates a notification.

The **Tasks** section in page details lists the current page's tasks and provides host-rendered **Mark done** / **Reopen** actions as a non-JavaScript fallback.

## Visual editor

In Visual mode, tasks render as task cards. Select a task to edit its ID, text, assignee, due date, or initial state. Markdown remains the canonical saved representation.

## Permissions

- `browser:render` renders the interactive checkbox in Kumbuka's isolated browser-module frame.
- `storage:read` reads persisted task completion state.
- `storage:write` persists validated task changes.
- `pages:content` rereads the current page before accepting a task command.
- `users:read` resolves canonical assignee mentions without exposing contact details.
- `notifications:send` creates core-owned in-app notifications after task state mutations.
