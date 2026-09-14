---
name: waka-local
description: "Use when delegating coding, debugging, testing, or repository work to Wade's local waka-bot Ollama agent through the VS Code Waka Local profile."
---

# Waka Local

This skill connects VS Code chat to the local `waka-bot` CLI. The CLI is the implementation agent; VS Code chat supplies the task and reports the outcome.

Any Git, GitHub, Docker, Docker Compose, gateway, WAF, tunnel, or Ollama-container interaction must use this skill and the Waka Local agent.

## Invocation

Run the canonical bridge from PowerShell with the target repository as `-Cwd`:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File D:\wakalabs\waka-bot\agents\invoke-local.ps1 -Prompt "<task>" -Cwd "D:\wakalabs\<repo>"
```

The bridge uses `waka-bot -p --yolo` with the local Ollama model and exits when the task is complete. It does not create a watcher or polling loop. It resumes the latest session for the target CWD.
The runtime injects `D:\wakalabs\waka-bot\agents\prompts\local-system.md`, which contains the habitat, Frontier, GitHub, Docker, and project-management operating rules. Tool failures and non-zero command exits are recorded in the JSONL session and reloaded as debugging context.

## Prompt composition

Include:

- the concrete task and acceptance criteria;
- the repository path and relevant files or symbols;
- the requirement to inspect existing code before editing;
- the narrow validation command to run afterward.

Do not add cloud services, API keys, or broad unrelated cleanup.

## Safety

The local agent follows the habitat rules: no main/master merges, no `--no-verify`, no protected-path moves, and confirmation before push, email, tunnel DNS changes, or destructive data operations.