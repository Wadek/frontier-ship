# DEPRECATED: do not consolidate toward D:\wakalabs\waka-bot\cli.
# Canonical coding CLI is D:\wakalabs\ai_learning\waka-cli (command: waka-cli).
# This script is kept only as historical reference.
<#!
.SYNOPSIS
    Create the canonical local waka-bot product tree.

.DESCRIPTION
    Migration is copy-first: existing ai_learning and waka-agents trees are
    preserved until the new tree has passed validation.
#>

$ErrorActionPreference = "Stop"
$root = "D:\wakalabs"
$target = Join-Path $root "waka-bot"
$cliSource = Join-Path $root "ai_learning\waka-cli"
$agentsSource = Join-Path $root "waka-agents"

New-Item -ItemType Directory -Force -Path $target, "$target\cli", "$target\agents", "$target\docs" | Out-Null

Remove-Item -LiteralPath "$target\cli", "$target\agents" -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path "$target\cli", "$target\agents" | Out-Null

Get-ChildItem -LiteralPath $cliSource -Force | Where-Object { $_.Name -notin @(".git", "__pycache__") } | ForEach-Object {
    Copy-Item -LiteralPath $_.FullName -Destination "$target\cli\$($_.Name)" -Recurse -Force
}

Get-ChildItem -LiteralPath $agentsSource -Force | Where-Object { $_.Name -notin @(".git", "logs") } | ForEach-Object {
    Copy-Item -LiteralPath $_.FullName -Destination "$target\agents\$($_.Name)" -Recurse -Force
}

$invoke = Join-Path $target "agents\invoke-local.ps1"
if (Test-Path -LiteralPath $invoke) {
    $text = Get-Content -LiteralPath $invoke -Raw -Encoding UTF8
    $text = $text.Replace('D:\wakalabs\ai_learning\waka-cli\run.py', 'D:\wakalabs\waka-bot\cli\run.py')
        $text = $text.Replace('D:\wakalabs\waka-agents', 'D:\wakalabs\waka-bot\agents')
    Set-Content -LiteralPath $invoke -Value $text -Encoding UTF8
}

$readme = @'
# waka-bot

Canonical local agent product for Wade's habitat.

## Layout

- `cli/` - local Ollama coding CLI and tests.
- `agents/` - Waka Local bridge, prompts, health agents, and repository actions.
- `docs/` - project-management and operating workflows.

The source trees `D:\wakalabs\ai_learning\waka-cli` and
`D:\wakalabs\waka-agents` remain as compatibility sources during migration.
New code should target this directory.

## Local runtime

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File D:\wakalabs\waka-bot\agents\invoke-local.ps1 -Prompt "<task>" -Cwd "D:\wakalabs\<repo>"
```

The runtime uses local Ollama only, keeps web search available, captures errors
into the current session, and never bypasses Frontier or merges `main`.
'@
Set-Content -LiteralPath "$target\README.md" -Value $readme -Encoding UTF8

$projectManagement = @'
# Waka Local project management

Waka Local is responsible for the complete local development loop, not only
code edits. It may inspect repositories, decompose work, run tests, diagnose
failures, manage feature branches, prepare commits, operate Docker Compose,
and create pull requests.

## Standard loop

1. Identify the repository and read its README and applicable `AGENTS.md`.
2. Inspect `git status --short --branch` and preserve existing user changes.
3. Break the request into the smallest verifiable tasks.
4. Work on a `frontier/*` feature branch; never edit or merge `main` directly.
5. Run the narrowest test after each meaningful change, then the required verbose suite.
6. For shipping, run `frontier hygiene`, `frontier plan`, and `frontier apply`.
7. Push through normal hooks and create a PR with `gh pr create --base main --head <branch> --fill`.
8. Report the exact commit, tests, PR, and remaining risks. A human merges.

## Error handling

Preserve exact non-secret stderr, exit codes, failed test names, Docker logs,
and GitHub CLI errors. Waka Local records failures in the current JSONL session
so another Waka client can resume diagnosis.

## Docker

Read compose configuration before changing services. Use the external
`waka-net` network, bind new host ports to loopback, and recreate a service
after Python changes with `docker compose up -d --force-recreate <service>`.

## Confirmation boundaries

Ask before push, PR creation, DNS changes, email, destructive data operations,
or changes to protected paths. Never use `--no-verify`, hook bypasses,
`FRONTIER_SOFT=1`, GitHub MCP write tools, or direct merges to `main`.
'@
Set-Content -LiteralPath "$target\docs\project-management.md" -Value $projectManagement -Encoding UTF8

Get-ChildItem -LiteralPath $target -Recurse -File | Where-Object { $_.Extension -in @('.ps1', '.py', '.md', '.txt', '.json', '.cmd') } | ForEach-Object {
    $content = Get-Content -LiteralPath $_.FullName -Raw -Encoding UTF8
    $content = $content.Replace('D:\wakalabs\waka-agents', 'D:\wakalabs\waka-bot\agents')
    $content = $content.Replace('D:\wakalabs\ai_learning\waka-cli', 'D:\wakalabs\waka-bot\cli')
    Set-Content -LiteralPath $_.FullName -Value $content -Encoding UTF8
}

Write-Host "Created canonical waka-bot tree at $target"
Get-ChildItem -LiteralPath $target -Directory | Select-Object -ExpandProperty Name