# Tasks

Tasks adds persistent actionable tasks to Kumbuka pages. Tasks support Kumbuka user assignees, due dates, administrator-configurable workflow states, and in-app notifications without rewriting page Markdown when a state changes.

## Usage

```markdown
{{task id="deploy-api" text="Deploy API to production" assignee="@alice" due="2026-10-01"}}
```

Only `id` and `text` are required. `id` is the stable identity used for persisted task state, so keep it unchanged when moving a task. IDs support letters, numbers, `.`, `_`, `-`, `/`, and `:`.

Optional attributes are:

- `assignee` — canonical Kumbuka user mention responsible for the task, such as `@alice`.
- `due` — due date in `YYYY-MM-DD` form.
- `initial` — initial workflow state ID. When omitted, Tasks uses the first configured state.

Task declarations inside fenced code blocks or inline code remain literal. Invalid declarations render a visible **Task error** instead of silently disappearing.

## Workflow states

Open **Administration → Plugin settings → Tasks → Workflow** to configure the ordered state list. Each state has:

- a stable lower-case **ID**, such as `todo`, `in-progress`, or `done`;
- a user-facing **Label**;
- a **Color** used by the interactive state selector;
- a **Completed** flag used for completed-task presentation and completion/reopen notifications.

When no states are configured, Tasks keeps the backward-compatible `open` and `done` workflow. Existing persisted `open`/`done` values and the previous boolean JSON state format are migrated in memory to the closest configured incomplete/completed state.

For a task that should start in a non-default state:

```markdown
{{task id="review-runbook" text="Review the production runbook" initial="in-progress"}}
```

## Interaction

On rendered pages, the browser module shows the configured workflow states in a selector. State changes are sent through Kumbuka's host-mediated page-details command handler. Before persisting a change, Tasks rereads the current page and verifies that both the task and requested state are still valid.

The **Tasks** section in page details shows the current state and provides one non-JavaScript **Move to …** action per task that advances to the next configured state.

When an assignee is initially set or changed, Tasks creates an attributed Kumbuka in-app notification from the committed old/new page Markdown. When an assigned task changes state, Tasks also notifies the assignee. Crossing into a completed state produces a completion notification; leaving one produces a reopened notification; other transitions produce a state-change notification.

Notification keys are deterministic, so retries do not create duplicate in-app notifications or duplicate `notification.created` webhook events. Rendering a page never creates notifications.

## Visual editor

In Visual mode, tasks render as task cards. The assignee uses Kumbuka's mention picker and the due date uses the native date picker. `Initial state ID` accepts configured state IDs; `open` and `done` remain suggested for the default workflow.

## Permissions

- `browser:render` renders the interactive state selector in Kumbuka's isolated browser-module frame.
- `settings:read` reads the administrator-configured task workflow.
- `storage:read` reads persisted task state.
- `storage:write` persists validated task changes.
- `pages:content` rereads the current page before accepting a task command.
- `users:read` resolves canonical assignee mentions without exposing contact details.
- `notifications:send` creates core-owned in-app notifications after committed assignment and state mutations.
