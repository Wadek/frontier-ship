---
name: inbox-watcher
description: >-
  Use the existing Python mailbox monitor and its PowerShell hook that spawns a Grok session. Use when the user says "mailbox monitor", "powershell hook to spawn a grok session", "restart inbox watcher", or "inbox watcher". Do not replace it with a Grok poll loop.
user-invocable: true
---

# inbox-watcher

The mailbox is event-driven. An existing Python method already monitors the mailbox. A PowerShell hook already spawns a Grok session. Reuse those. Do not poll.

## Steps

1. Find and reuse the existing Python mailbox monitor. Do not rewrite it.
2. Keep the PowerShell hook that spawns a Grok session on mail.
3. On "restart inbox watcher", restart that existing watcher only.
4. Do not put Grok on a 1-minute scheduler or a 30s poll loop for this job.
