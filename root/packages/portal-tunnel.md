---
id: 43210fd43f7878fd6c6b931640fb2175
author: gosunuts
title: Make localhost public with one curl — Portal Tunnel
description: Make your localhost public instantly with one curl command using Portal Tunnel—a decentralized, permissionless alternative to ngrok and cloudflared.
language: en
date: 2025-12-06T05:35:03.047352133Z
path: /blog/posts/make-localhost-public-with-one-curl-—-portal-tunnel-z2f33ae49
go_package: gosuda.org/portal
go_repourl: https://github.com/gosuda/portal.git
---

## Portal and Tunnel

We can create programs anywhere With AI.
But no matter how great a program is, it usually lives only on your own computer — on localhost.
![vibecon](/assets/images/portal/vibecon.webp)

To expose it to the outside world, you normally have to go through complicated steps such as router configuration, firewall rules, public IP setup, and tunnel configuration.

What if all of this could be solved with a single line of command?

With Portal’s tunnel, you can turn your local program into a public service with just one command.

## Make localhost public

1. First, run your program locally.

2. Then, this single line is all you need:
```bash
curl -fsSL portal.gosuda.org/tunnel | PORT=3000 NAME={app name} sh
```

3. Check that your app is now publicly accessible:
- {app name}.portal.gosuda.org

## Multi-tenancy

Portal is designed as an open network, not a single service. Anyone can operate a portal relay, and a single app can be connected to multiple portals simultaneously for redundancy or geographic distribution.

```bash
# Publish to multiple portal relays at once
curl -fsSL http://portal.gosuda.org/tunnel | \
PORT=3000 \
NAME={app_name} \
RELAY_URL=portal.thumbgo.kr,portal.iwanhae.kr,s-h.day,portal.lmmt.eu.org \
sh
```

A list of active public portals is maintained in the Portal List app (which is itself hosted on the Portal network):
https://portal-list.portal.gosuda.org/

This represents a truly permissionless publishing environment that is not dependent on any specific provider or infrastructure.

## Comparison with Other Services

Tools like ngrok and cloudflared are widely used to expose local services to the public internet.  
However, Portal is fundamentally different in both design philosophy and usage model.

ngrok and cloudflared are centralized, SaaS-based tunneling services.  
They require account creation, token issuance, binary installation, and configuration before use, and users are inevitably subject to service policies and pricing models.

In contrast, Portal Tunnel:

- Runs in one line without installation
- Publishes instantly without accounts or tokens
- Allows anyone to operate a relay
- Lets a single app connect to multiple portals simultaneously
- Is a pure network architecture without dependency on any specific vendor

These characteristics place Portal Tunnel in a completely different category from traditional tunneling services.