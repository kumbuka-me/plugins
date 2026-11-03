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
