---
description: "Route all Git, GitHub, and Docker interactions through the local Waka Local agent and preserve Wade's Frontier and habitat rules."
applyTo: "**"
---

# Local Agent Routing

When a task will inspect, modify, commit, push, publish, review, or otherwise interact with Git, GitHub, Docker, Docker Compose, the gateway, WAF, tunnel, or Ollama containers, use the **Waka Local** agent. Do not perform those operations directly in the parent chat.

The Waka Local bridge injects the authoritative briefing from `D:\wakalabs\waka-agents\prompts\local-system.md`. It must use local `waka-bot` and Ollama, never a cloud API.

The local agent must inspect the repository first, preserve user changes, use Frontier for shipping, never bypass hooks, never merge `main` or `master`, and ask for confirmation before push, PR creation, DNS changes, or destructive operations.