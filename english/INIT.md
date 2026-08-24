# How to start Frontier

Read this in order. Simple English only here.

## What you are installing

1. **Laws** (English + Haskell) — what is allowed  
2. **`git` + `frontier`** (Go) — how you talk to the machine every day  

You do **not** install a new operating system.

---

## Step 0 — Tools on your PC

- Git for Windows (real engine)  
- Go 1.22+ (to build the runtime)  
- Optional: GHC 9.x (to check the Haskell laws)

---

## Step 1 — Get the code

```text
git clone https://github.com/Wadek/frontier-ship.git
cd frontier-ship
```

If Frontier `git` is already on your PATH, and it blocks cloning quirks, use the real engine once:

```text
& "C:\Program Files\Git\cmd\git.exe" clone https://github.com/Wadek/frontier-ship.git
```

---

## Step 2 — Build the Go runtime

```text
go build -o D:\frontier\bin\git.exe ./cmd/frontier-git
go build -o D:\frontier\bin\frontier.exe ./cmd/frontier
```

Or:

```text
powershell -File scripts\install-git-interface.ps1
```

Then open a **new** terminal.

Check:

```text
Get-Command git
git frontier explain
git --version
```

You should see `D:\frontier\bin\git.exe` and a normal git version string.

---

## Step 3 — Learn mode (first week)

```text
$env:FRONTIER_SOFT = "1"
```

This warns instead of blocking. Turn it off when ready:

```text
$env:FRONTIER_SOFT = "0"
```

---

## Step 4 — First safe drill (Terraform-like)

```text
cd <some-repo>
git checkout -b frontier/first
# edit a file
git add -A
git commit -m "frontier: first sealed change"

git frontier demo
git frontier learn         # classify project (L)
git frontier guard         # security exam (G)
git frontier plan          # must succeed — fail closed
git frontier apply         # only works after plan.passed
git push -u origin HEAD    # only works after apply/gate.passed
```

If `plan` fails, fix reasons; do not expect apply/push to work.  
Ledger is state (like Terraform state).

---

## Step 5 — Optional Haskell check

```text
cd haskell
cabal test
```

If you have no GHC yet, skip this. The English + Go path still works. Install GHC later so F5 (English + Haskell + Go) is complete on your machine.

---

## Step 6 — Agents use the same CLI

Coding agents run `frontier` and `git` like a human. There is no separate host protocol.

Corporate laptop with no install: paste `teach/CORPORATE_AI_TRIAL_PROMPT.md`.

---

## Where to read next

| File | Why |
|------|-----|
| `english/LANGUAGE.md` | Why English + Haskell + Go |
| `english/AXIOMS.md` | The laws in English |
| `teach/LEARN_GIT.md` | Practice with `git` |
| `haskell/` | The same laws as pure code |
| `english/WHY_NOT_A_CUSTOM_OS.md` | Why we do not write an OS |
