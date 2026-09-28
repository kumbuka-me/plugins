# Tasks

Tasks adds persistent actionable task lists to Kumbuka pages. A task can have an assignee, due date, configurable workflow state, and any number of nested subtasks. State changes are stored separately from page Markdown, so checking a task does not create a page revision.

## Visual editor

The preferred way to create tasks is the **Tasks** block in the Visual editor. One block owns the entire task tree and opens one modal with a **Tasks and subtasks** table.

Each row contains:

- **Task** — the visible task text.
- **ID** — the stable task identity used for persisted state.
- **Parent task ID** — leave empty for a top-level task, or enter the ID of an earlier row to make this row its child.
- **Assignee** — an optional canonical Kumbuka mention such as `@alice`.
- **Due** — an optional `YYYY-MM-DD` date.
- **Initial state** — an optional workflow state ID. When omitted, the first configured state is used.

All tasks, subtasks, sub-subtasks, and deeper levels are edited in that same modal. Parent rows must appear before their children. Nesting is supported up to 16 levels.

A newly inserted block starts with one row:

```markdown
{{tasks texts="Describe the task" ids="task-id"}}
```

When multiple rows are edited visually, Kumbuka stores the table columns as list attributes inside the same `{{tasks ...}}` declaration. Those list separators are an implementation detail; source-mode users can continue to use the readable single-task syntax below.

## Source-mode task syntax

The existing `{{task ...}}` form remains supported and adjacent declarations are rendered as one task list:

```markdown
{{task id="release" text="Prepare release" assignee="@alice" due="2026-10-01"}}
{{task id="release-notes" parent="release" text="Publish release notes" assignee="@alice"}}
{{task id="production" parent="release" text="Deploy to production" assignee="@bob"}}
{{task id="verify" parent="production" text="Verify production health"}}
```

This produces the hierarchy:

```text
Prepare release
├─ Publish release notes
└─ Deploy to production
   └─ Verify production health
```

Only `id` and `text` are required for a single-task declaration. IDs support letters, numbers, `.`, `_`, `-`, `/`, and `:`. Keep an ID unchanged when editing or moving a task because persisted state is keyed by that ID.

Task declarations inside fenced code blocks or inline code remain literal. Invalid declarations render a visible **Task error** instead of silently disappearing.

## Workflow states

Open **Administration → Plugin settings → Tasks → Workflow** to configure the ordered state list. Each state has:

- a stable lower-case **ID**, such as `todo`, `in-progress`, or `done`;
- a user-facing **Label**;
- a **Color** used by the interactive state control;
- a **Completed** flag used for completed-task presentation and completion/reopen notifications.

When no states are configured, Tasks uses the backward-compatible `open` and `done` workflow. Existing persisted `open`/`done` values and the previous boolean JSON state format are mapped to the closest configured incomplete/completed state.

Completing every child does not automatically complete its parent. A parent remains an independent task and must be moved to a completed state explicitly.

## Interaction

Rendered task lists use compact rows instead of large cards. Nested tasks are shown below their parent with a hierarchy line. The circular control at the left changes state; the current state, assignee, and due date remain visible as secondary metadata. Completed tasks are muted and struck through, and multi-task lists show a completed/total counter.

State changes are sent through Kumbuka's host-mediated page-details command handler. Before persisting a change, Tasks rereads the current page and verifies that both the task and requested state are still valid.

The **Tasks** section in page details shows the current state and provides one non-JavaScript **Move to …** action per task that advances to the next configured state.

When an assignee is initially set or changed, Tasks creates an attributed Kumbuka in-app notification from the committed old/new page Markdown. When an assigned task changes state, Tasks also notifies the assignee. Crossing into a completed state produces a completion notification; leaving one produces a reopened notification; other transitions produce a state-change notification.

Notification keys are deterministic, so retries do not create duplicate in-app notifications or duplicate `notification.created` webhook events. Rendering a page never creates notifications.

## Permissions

- `browser:render` renders the interactive state controls in Kumbuka's isolated browser-module frame.
- `settings:read` reads the administrator-configured task workflow.
- `storage:read` reads persisted task state.
- `storage:write` persists validated task changes.
- `pages:content` rereads the current page before accepting a task command.
- `users:read` resolves canonical assignee mentions without exposing contact details.
- `notifications:send` creates core-owned in-app notifications after committed assignment and state mutations.
