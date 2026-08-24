# Hygiene (H) — AI-provenance marks

**Word:** `frontier hygiene`  
**Aliases:** `watermarks`, `marks`, letter **`H`**  
**Backend:** local [watermarks-remover](https://github.com/guillaumemeyer/watermarks-remover) HTTP service (`D:\wakalabs\watermarks-remover`)

Hygiene inspects the **changeset** for multi-vendor AI provenance marks (invisible Unicode, C2PA / EXIF / XMP, container metadata) and can strip them. It does **not** replace Guard.

## Pipeline order

```
  frontier scm …
  frontier learn      # L
  frontier guard      # G  security
  frontier hygiene    # H  provenance (this family)
  frontier slim       # S  planned
  frontier optimize   # O
  frontier plan → apply → push
```

## What it is

| Command | Effect |
|---------|--------|
| `frontier hygiene` / `inspect` | Inspect changed files (vs main + dirty). Advise if marks. |
| `frontier hygiene status` | Service health + capabilities |
| `frontier hygiene clean PATH` | Strip → `PATH` with `.cleaned` inserted (or `--in-place`) |
| `frontier watermarks` | Same command |

Default **disposition: advise**. Cleaning is explicit. `FRONTIER_HYGIENE_BLOCK=1` promotes suspicious findings to a **plan/apply block**.

## Service

Loopback only by default:

```text
http://127.0.0.1:8765
```

Env: `WATERMARKS_SERVICE_URL` or `FRONTIER_HYGIENE_URL`.

Start:

```powershell
python D:\wakalabs\watermarks-remover\service\scripts\server.py --host 127.0.0.1 --port 8765
# or: docker compose up -d   (in D:\wakalabs\watermarks-remover)
```

If the service is down, Hygiene **records** that fact and does not fail the ship (unless you set the block flag and expected it up).

## What it is not

- Not a vendor-oracle “proves human-written” stamp (Layer B statistical rewrite is best-effort).
- Not Guard (OWASP / secrets stay in `frontier guard`).
- Does not silently rewrite the tree on `plan` / `push`.
- Own content / hygiene / research only — see the upstream ethics notes.

## Evidence

Ledger actions: `hygiene.inspected`, `hygiene.cleaned`, `hygiene.service_down`.
