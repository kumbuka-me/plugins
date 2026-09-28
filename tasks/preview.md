# Release checklist

{{task id="release" text="Prepare release" assignee="@alice" due="2026-10-02"}}

{{task id="release-notes" parent="release" text="Publish release notes" assignee="@alice" due="2026-10-01" initial="done"}}

{{task id="production" parent="release" text="Deploy to production" assignee="@bob" due="2026-10-02"}}

{{task id="verify" parent="production" text="Verify production health" assignee="@bob"}}

{{task id="announce" text="Announce the release" assignee="@carol"}}
