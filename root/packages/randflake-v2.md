---
id: 642af9de8681e69b3f7a259faf4a04b0
author: Lemon Mint
title: 'Randflake v2: 64-Bit Encrypted ID Generator with Half-Open Node Leases'
description: Go module (gosuda.org/randflake/v2) providing Sparx64-encrypted 64-bit unique ID generation with whole-second Unix timestamps and half-open node lease validation.
language: en
date: 2026-09-09T00:49:37.732996276Z
path: /randflake/v2
go_package: gosuda.org/randflake/v2
go_repourl: https://github.com/gosuda/randflake.git
hidden: true
no_translate: true
---

Randflake v2 provides configured generators with whole Unix seconds, half-open node leases, and encrypted 64-bit IDs. Go 1.23 or newer is required.

```bash
go get gosuda.org/randflake/v2@v2.1.0
```

The deprecated constructor API is available at `gosuda.org/randflake/v2/legacy`. Existing IDs retain their wire format.

[Source and usage examples](https://github.com/gosuda/randflake)
