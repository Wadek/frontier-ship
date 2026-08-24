# Alpha release walkthrough (SLSA Go releaser)

What you saw in GitHub (“SLSA Go releaser”, “SLSA Generic generator”) are reusable workflows from [slsa-github-generator](https://github.com/slsa-framework/slsa-github-generator).

We use the **Go builder** (`builder_go_slsa3.yml`): it compiles our binaries **and** attaches SLSA provenance. The **generic** generator is for when you already built artifacts yourself; we do not need it for this alpha.

---

## One-time: what is already in this repo

| Path | Purpose |
|------|---------|
| `.github/workflows/slsa-goreleaser.yml` | Release on tag `v*` |
| `.github/workflows/ci.yml` | Test/build on push to main |
| `.slsa-goreleaser/*.yml` | One config per binary × OS/arch |

Assets produced on a tag (examples):

- `frontier-git-linux-amd64`
- `frontier-git-windows-amd64`
- `frontier-git-darwin-arm64`
- `frontier-linux-amd64` / `frontier-windows-amd64`
- matching `*.intoto.jsonl` provenance files

Alpha tags (`v0.1.0-alpha.1`) create a **prerelease**.

---

## Walkthrough: cut the alpha

### 1. Push release wiring to `main`

(Already done when this file landed; if not: merge/push the workflow files.)

### 2. Create and push the tag

From a clean tree (or with `FRONTIER_SOFT=1` if your Frontier `git` blocks tagging habits):

```powershell
cd C:\Users\waka\src\frontier-ship
# use real git for the tag push if the shim fights you:
$env:FRONTIER_GIT_BIN = "C:\Program Files\Git\cmd\git.exe"

git tag -a v0.1.0-alpha.1 -m "Frontier alpha: git shim + CLI with SLSA3 provenance"
git push origin v0.1.0-alpha.1
```

### 3. Watch GitHub Actions

```text
https://github.com/Wadek/frontier-ship/actions
```

Open **SLSA go releaser** for the tag. Wait until all matrix builds are green.

### 4. Confirm the release page

```text
https://github.com/Wadek/frontier-ship/releases/tag/v0.1.0-alpha.1
```

You should see binaries + `.intoto.jsonl` files. Marked **Pre-release**.

### 5. Pull into another project’s workflow

Example: install `frontier-git` on a Linux CI runner and use it as `git`:

```yaml
# .github/workflows/example-use-frontier.yml
name: Use Frontier alpha

on:
  workflow_dispatch:

permissions:
  contents: read

jobs:
  demo:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download frontier-git (alpha)
        env:
          GH_TOKEN: ${{ github.token }}
        run: |
          gh release download v0.1.0-alpha.1 \
            --repo Wadek/frontier-ship \
            --pattern 'frontier-git-linux-amd64*' \
            --dir /tmp/frontier
          chmod +x /tmp/frontier/frontier-git-linux-amd64
          sudo mv /tmp/frontier/frontier-git-linux-amd64 /usr/local/bin/frontier-git

      # Optional: verify SLSA provenance (install slsa-verifier once)
      # - name: Verify provenance
      #   run: |
      #     # see https://github.com/slsa-framework/slsa-verifier
      #     slsa-verifier verify-artifact /usr/local/bin/frontier-git \
      #       --provenance-path /tmp/frontier/frontier-git-linux-amd64.intoto.jsonl \
      #       --source-uri github.com/Wadek/frontier-ship \
      #       --source-tag v0.1.0-alpha.1

      - name: Point FRONTIER_GIT_BIN at system git; put frontier first if desired
        run: |
          echo "FRONTIER_GIT_BIN=$(which git)" >> "$GITHUB_ENV"
          # Or invoke explicitly:
          frontier-git frontier explain
          frontier-git frontier demo
```

For **Windows** runners, download `frontier-git-windows-amd64.exe` instead.

---

## Manual re-run without a new tag

Actions → **SLSA go releaser** → **Run workflow**.  
That builds; asset upload to a Release is mainly for **tags**.

---

## Verify provenance locally (optional)

```text
# install: https://github.com/slsa-framework/slsa-verifier/releases
slsa-verifier verify-artifact ./frontier-git-linux-amd64 \
  --provenance-path ./frontier-git-linux-amd64.intoto.jsonl \
  --source-uri github.com/Wadek/frontier-ship \
  --source-tag v0.1.0-alpha.1
```

---

## Notes (honest)

- The SLSA Go builder is in maintenance mode; GitHub also pushes **artifact attestations**. For alpha, SLSA Go releaser matches what the GitHub UI offered you.
- Private repos posting to public Rekor can leak the repo name; this project is **public**, so that is fine.
- Minimality: we release only the binaries people need (`frontier-git`, `frontier`), not a framework zoo.
