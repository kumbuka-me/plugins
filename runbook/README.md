# Runbook

Render a decision procedure as a compact, top-to-bottom graph. Every action and branch is visible. Two routes can point to the same handoff, so its instructions appear once.

## Usage

```runbook
alert: Receive alert
  -> priority
priority: What is the priority?
  High -> page
  Medium -> notify
page: Page the on-call engineer
  -> impact
notify: Notify the team
  -> impact
impact: Does this need attention?
  Yes -> marco
  No -> count
count: More than 5 alerts in 10 minutes?
  Yes -> marco
  No -> ack
ack: Acknowledge the alert
marco: Contact Marco
```

Write one `id: Title` per step in display order. Indent each route beneath its step. Use `-> id` to continue or `Answer -> id` for a named choice. Give every branch of a decision a distinct label. A step with no routes ends that path.

IDs start with a letter and can contain letters, digits, underscores, and hyphens. They are case-sensitive. Start with the entry point and place destinations below their callers. Every step must be reachable. A block can contain up to 512 lines. A malformed graph shows a diagnostic and the original source.

## Permissions

The plugin uses `browser:render` to draw inside Kumbuka’s isolated browser frame. Without the browser module, readers see the fenced source.
