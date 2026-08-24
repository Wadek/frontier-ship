# Stack: Habitat edge (private Docker net + cloudflared)

## Detect

- `cloudflared/config.yml` with `ingress:` hostnames
- External Docker network for apps (no WAN bind)
- Optional scripts that publish DNS routes for the tunnel

## Standard publish flow (new public app)

```text
1. App joins the habitat network; prefer NO host ports
2. App has its own auth (or goes through gateway oauth2-proxy)
3. Edit cloudflared/config.yml — add hostname + service
4. Publish DNS:
   cloudflared tunnel route dns --overwrite-dns <tunnel-id> <hostname>
5. Restart the cloudflared compose service
6. Probe: https://<hostname>/api/health (or /)
7. Frontier: learn | guard on the app repo before inviting users
```

## Exposure audit (before first public hostname)

| Check | Why |
|-------|-----|
| `docker ps` Ports column | Spot `0.0.0.0` binds (LAN; dangerous if WAN-forwarded) |
| cloudflared ingress list | What is already on the public Internet |
| Direct vs gateway | Direct routes need strong app auth |
| Router port-forwards | Human confirms — agents cannot see ISP NAT |
| Cookie `Secure` | `Secure=true` breaks `http://127.0.0.1`; use env `COOKIE_SECURE` |

## Pitfalls learned

| Pitfall | Fix |
|---------|-----|
| `express-session` `secure: NODE_ENV===production` | Blocks sessions on localhost HTTP after register |
| Tunnel-route CLI argument order | `cloudflared tunnel route dns --overwrite-dns <id> <hostname>` |
| Stale paths in old scripts after a habitat move | Point scripts at the current habitat root |
| Committing sqlite under `data/` | gitignore runtime data |
| Cloudflare **Error 1033** | The connector is gone, not nginx. Docker Desktop must be running; confirm the cloudflared container logged `Registered tunnel connection`. Loopback can still work while the tunnel is down. |
| `python:3.12-alpine` + `cryptography`/`bcrypt` | Use `python:3.12-slim`. Alpine musl wheels stall the build. |
| `entrypoint.sh` CRLF from Windows | `sed -i 's/\\r$//' /app/entrypoint.sh` in the Dockerfile. |
| Public app behind GitHub oauth2-proxy | Phone users cannot log in. Public hostname, **app-owned auth**. |
| urllib/curl **403** on the public host | Cloudflare bot fight. Send a browser User-Agent; browsers are fine. |
| Project on the OS disk | Habitat apps live on the habitat volume, one tree per app. |
| Several parallel copies of the same product | One tree. Learn the old ones; ship in the habitat copy. |
| Hardcoded `SECRET_KEY` in the image | Compose/env only. Never bake demo secrets into `Dockerfile`. |
| Binding `0.0.0.0:<port>` | Loopback only: `127.0.0.1:<port>` for local tryout. |
| Nginx/ingress edited, still 1033 or old vhost | Restart **both** the gateway and cloudflared. Re-run DNS publish. |

## Guard / Optimize

- Run `frontier learn` + `frontier guard` on the app before enabling the tunnel.
- Add a stack playbook link in `.frontier/optimize/stack.md` when Optimize starts (this edge playbook plus the app’s language playbook).
