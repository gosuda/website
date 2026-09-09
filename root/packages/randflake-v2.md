---
author: Lemon Mint
title: Randflake v2
description: Randflake v2 Go module with half-open leases, configured clocks, and encrypted 64-bit IDs.
language: en
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
