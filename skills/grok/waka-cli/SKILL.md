---
name: waka-cli
description: >-
  Build and operate waka-cli as a grok-cli-shaped local coding CLI (Ollama).
  Use when the user says "model waka-cli after grok-cli", "powershell-cli similar
  to grok", "forget --yolo", "--always-approve", "vim like behavior in grok cli",
  "deepseektokens", "waka-cli + deepseek tokens",
  or wants the local coding agent on PATH as waka-cli.
user-invocable: true
---

# waka-cli

Canonical local coding CLI. Source: `D:\wakalabs\ai_learning\waka-cli`.
Command on PATH: **`waka-cli`** only. Do not install or use `waka` or `waka-bot`
as CLI aliases — those names caused confusion with the separate chat/agent trees.

## Steps

1. Copy grok-cli behavior. Do not invent a different CLI shape.
2. Put **`waka-cli`** on PATH (`install.ps1`). Never re-add `waka` / `waka-bot` launchers.
3. Do not add `--yolo`. `--always-approve` is the allowed auto-approve flag (`--yolo` may remain as a hidden legacy alias in code only).
4. In PowerShell, use `&` only as the call operator. Do not put `&` inside a URL.
5. Vim-like navigation (gt between sessions, g, gg, ctrl+p) is an accepted customization, not a second product.
6. Do not poll the user or sit in monitor loops. The run notifies when done.
7. Prefer a local model when choosing what waka-cli runs (see local-first).
8. Advanced planning: `waka-cli plan` (DeepSeek after egress). Atomic edits: local Ollama via `waka-cli -p`. If the local coder errors or writes no files, `-p` falls back to DeepSeek `codegen` on the same prompt.
9. Session `/model` choices: `qwen2.5-coder:64k` (local), `deepseek-flash` (cloud), `deepseek-v4-pro` (cloud). Aliases: `qwen`/`local`, `flash`/`deepseek-flast`, `pro`/`v4-pro`.

## Not this product

- `D:\wakalabs\waka-bot` — older agent/chat consolidation tree; its `cli/` is stale. Point runners at `ai_learning\waka-cli`.
- `waka-net/waka-bot` / `waka-chat` — web chat UIs, not the coding CLI.
- Ollama alias `waka-coder` — model tag only, not a shell command.
