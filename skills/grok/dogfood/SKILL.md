---
name: dogfood
description: >-
  Dogfood the change and push it to GitHub. Use when the user says "dogfood it and push to github" or "dogfood it". Distinct from merely committing; the agent must use the tool the way a user would, then push.
user-invocable: true
---

# dogfood

Run the thing you just built the way a user would, then push.

## Steps

1. Exercise the change as a user of the product, not only as its author.
2. After dogfood, push the feature branch to GitHub.
3. Do not skip Frontier `plan` / `apply` before the push. Never skip the local git hooks.
