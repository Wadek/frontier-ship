---
name: cloudflare
description: >-
  Own-WAF override of the Cloudflare plugin. Use when the user says they do not want Cloudflare, want their own WAF, or "i dont want to use cloudflare". Do not load the plugin waf/ tree in that case.
user-invocable: true
---

# Cloudflare (user override)

This file overrides the installed Cloudflare plugin skill of the same name for WAF routing only. Do not edit `installed-plugins`.

## Security routing

Need security? Only if the user is on Cloudflare **and** asked for Cloudflare WAF → load plugin `waf/`. If they want their own WAF / not Cloudflare, **do not load this tree**.

Do not default to Cloudflare WAF, Tunnel, or Workers because a "WAF" or "edge" word appeared. Confirm they are on Cloudflare first.
