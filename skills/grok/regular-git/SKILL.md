---
name: regular-git
description: >-
  Keep work on git and GitHub as a regular habit: private repos, commits, and confirming a push. Use when the user says "we need to make sure we are regularly using git and github", "create private repo on github and commit", or "is this pushed to github?". Does not replace the Frontier pre-push pipeline.
user-invocable: true
---

# regular-git

Use git and GitHub in the same session as the work. Do not leave repos uncommitted or unpushed.

## Steps

1. Create a private GitHub repo when the work has none.
2. Commit on a feature branch. Never commit or push main or master.
3. When asked "is this pushed to github?", check remote tracking and say yes or no with the branch name.
4. In-session tests do not replace Frontier (`frontier plan` / `frontier apply`) before GitHub. Never skip the local git hooks.
