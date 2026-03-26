---
author: Gosunuts
title: end to end tls for public relay
---

## TL;DR

Portal is a permissionless relay network for exposing local services on public domains.

To preserve standard browser access without fully trusting the relay, Portal combines SNI-based routing, relay-backed keyless signing, and a built-in TLS self-probe that detects suspected relay-side TLS termination.

## Overview

On the web, it is surprisingly hard to get both public-domain convenience and strong end-to-end security at the same time.

This is because serving a public domain requires someone to terminate TLS for that domain, which typically pulls that party into your trust boundary.

The problem becomes more important in a permissionless relay network, where relays may be run by parties you do not know or fully trust.

Portal is designed around a different goal: keep the normal “open a URL in a browser” experience, but avoid giving the relay full visibility into tenant traffic.

Most tunnel systems solve this trade-off in one of three ways.

1. Terminate TLS at the relay (ngrok, Cloudflare Tunnel)

This is the most common design, used by many hosted tunnel and edge platforms.

It gives you clean public domains, simple setup, and convenient Layer 7 features such as caching, WAF, and HTTP-aware routing. But the cost is that the relay can see plaintext traffic, controls certificate private keys, and must be trusted as an active part of the security model.

2. Use a tunnel protocol (Tailscale, Openziti)

Protocols such as SSH or WireGuard preserve strong end-to-end encryption, but they change the access model. In practice, this usually means requiring a custom client, losing direct browser access, or making domain-based HTTPS exposure awkward. You keep strong encryption, but you move away from the standard web model.

3. Use a QUIC-native design (WebTransport-based)

QUIC provides built-in encryption and modern transport behavior, but it does not solve the relay trust problem by itself.

SNI is still exposed during the handshake, browser integration is narrower than plain HTTPS, UDP may be blocked in some environments, and operational complexity increases. QUIC can improve transport, but it does not automatically remove the relay from the trust boundary.

## How Portal approaches this differently

Portal tries to preserve the browser-native web model while moving tenant TLS termination back to the client side.

It does this by combining three ideas:

- SNI passthrough for routing
- Keyless signing for relay-backed domains
- MITM detection inside the client

These pieces are meant to work together, not independently. SNI passthrough lets the relay route connections without terminating tenant TLS. Keyless signing makes relay-backed public domains possible without moving private key material into the relay’s normal data path. The self-probe then checks whether the relay is actually preserving passthrough in practice.

In the normal data path, the flow looks like this:

1. The relay accepts a public TLS connection and inspects only the ClientHello information needed for SNI-based routing.
2. The SNI value is used to select the correct reverse session.
3. The relay forwards the encrypted connection as raw bytes over that reverse session.
4. The SDK or tunnel endpoint acts as the TLS server and terminates tenant TLS locally.
5. For relay-backed domains, the SDK requests certificate signatures through /v1/sign, using the relay as a keyless signing oracle.
6. TLS session keys are derived only on the SDK side.
7. After the handshake, the relay continues forwarding ciphertext without access to tenant plaintext or session keys.

The result is a model that aims to preserve several properties at once:

- no tenant plaintext at the relay
- no tenant session keys at the relay
- direct browser access over standard HTTPS
- public-domain routing without full relay-side TLS termination

## Detecting relay-side TLS termination (MITM)

Of course, this design only works if the relay actually preserves passthrough.

That is why Portal does not simply assume relay honesty. In addition to the normal forwarding path, the SDK performs a built-in self-probe to detect suspected relay-side TLS termination.

The probe works like this:

1. The client connects to its own public URL through the relay.
2. On the public-facing client side, it extracts TLS exporter material from the connection.
3. On the SDK side, where tenant TLS terminates locally, it reads the corresponding exporter material again.
4. If the two values match, the connection was observed as TLS passthrough.
5. If they differ, Portal treats that as suspected relay-side TLS termination.

This self-probe does not make the relay trustworthy by assumption. It tries to verify a concrete property of the connection path instead.

## Limits and open problems

This doesn’t eliminate trust completely:

The relay still participates in certificate signing for relay-backed domains, which means it is not reduced to a purely packet forwarder. The self-probe can confirm passthrough for the probe connection, but it does not prove that every user connection was handled the same way. The hardest remaining problem is still selective or adaptive MITM behavior.

So the point of Portal is not to claim that trust disappears. The goal is narrower and more practical: reduce unnecessary trust in the normal path, make violations more detectable, and create room for stronger relay verification models over time.

That is also why Portal’s long-term direction is not “trust one relay forever,” but stronger verification, witness-based checks, and better relay trust models for a permissionless network.

Feedback, especially on the MITM detection model and its limits, is very welcome.

GitHub: https://github.com/gosuda/portal

Domain: https://portal.thumbgo.kr