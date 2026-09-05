---
id: 78787664f5c944556bd14c0346a41b41
author: Lemon Mint
title: Introducing GSTime and GSTimeAssure
description: GSTime & GSTimeAssure
language: en
date: 2026-09-05T11:44:59.861082619Z
path: /gstime
go_package: gosuda.org/gstime
go_repourl: https://github.com/gosuda/gstime.git
hidden: true
no_translate: true
---

Zero-dependency, fault-tolerant time synchronization and certification engine in Go. Provides continuous SI-nanosecond tracking, dual-track separation between statistical estimation and interval certification, RFC 8915 Network Time Security (NTS), bounded leap smearing, and lock-free publication.

## Requirements

Go 1.27.0 or newer. Zero external dependencies.

```bash
go get gosuda.org/gstime
```

## Upstream Sources Configuration

GSTime supports NTS-KE authenticated endpoints and standard NTPv4 pools across independent failure domains ($N \ge 2F+1$):

```go
cfg := config.Config{
	Assurance: config.AssuranceConfig{
		FaultBudget:       1,
		MinVotingDomains:  3,
		MinHonestCoverage: 2,
		MaxWidthNs:        32 * 1_000_000_000,
	},
	Raw: config.RawConfig{
		BackendProfile: "standard_monotonic",
		ScaleLowerPpm:  -200.0,
		ScaleUpperPpm:  200.0,
		ReadBoundNs:    1000,
	},
	Sources: []config.SourceConfig{
		{FaultDomainID: "cloudflare", Endpoint: "time.cloudflare.com:4460", NTS: true},
		{FaultDomainID: "google",     Endpoint: "time.google.com:123",      NTS: false},
		{FaultDomainID: "apple",      Endpoint: "time.apple.com:123",       NTS: false},
		{FaultDomainID: "meta",       Endpoint: "time.facebook.com:123",    NTS: false},
	},
}
cfgID, _ := cfg.ConfigID()
```

## Minimal Examples by Use Case

### Service & Background Sync Initialization

```go
rawClock := clock.NewSystemRawClock()
leapHistory, _ := core.NewLeapHistory(10, nil) // Configured GSTL1 leap table
svc := gstime.NewClockService(rawClock, leapHistory, cfgID, 32_000_000_000)

// Start background NTP/NTS synchronization engine
engine, err := gstime.NewSyncEngine(cfg, svc)
if err != nil {
	log.Fatal(err)
}

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

_ = engine.Start(ctx)
defer engine.Close() // Best Practice: Gracefully stops background worker with zero goroutine leaks

// Wait until initial synchronization is achieved
_ = engine.WaitSync(ctx)
```

### 1. PublicClock: Monotonic Presentation Time

For logging, APIs, and metrics. Guaranteed strictly non-decreasing ($P_{k+1} \ge P_k$) under OS clock steps, DST transitions, and VM snapshot rollbacks.

```go
pub := svc.NowPublicAssured()

fmt.Printf("Public Time: %d\n", pub.Center)                 // GstInstant (continuous SI-ns)
fmt.Printf("Symmetric Uncertainty: ±%d ns\n", pub.PublicSymmetricEpsilon)
fmt.Printf("Status: %s\n", pub.Status)                      // SYNCED, HOLDOVER, or DESYNC
```

### 2. GSTimeAssure: Distributed Transactions & CommitWait

For distributed databases requiring external consistency via certified interval and CommitWait.

```go
now := svc.Now()
if now.Interval != nil {
	// Certified interval [Earliest, Latest] enclosing true SI time
	fmt.Printf("Certified Range: [%d, %d]\n", now.Interval.Earliest, now.Interval.Latest)
}

// Tri-state causality check: CertainYes, CertainNo, or Unknown
decision, _, _ := svc.After(txTimestamp)

// Commit wait: blocks until certified lower watermark strictly exceeds commitTs
ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
defer cancel()

err := svc.CommitWait(ctx, commitTs, now.AssuranceEpochID, now.LeapHistoryID, now.ConfigID)
if err != nil {
	// Handle ErrDeadlineExceeded, ErrDesynchronized, or ErrConfigurationMismatch
}
```

### 3. Civil UTC with Leap Seconds

For civil calendrical display supporting positive leap seconds (`SecondOfDay = 86400`).

```go
earliest, latest, est, status, err := svc.NowUtc(leapHistory.ID)
if est != nil {
	fmt.Printf("UTC: %s\n", est.String())             // e.g. 2026-09-05T06:56:38.922254833Z
	fmt.Printf("SecondOfDay: %d\n", est.SecondOfDay)   // 0..86399 (or 86400 on leap seconds)
}
```

### 4. POSIX / Unix Projection

For compatibility with legacy Unix millisecond/nanosecond APIs.

```go
proj, err := svc.NowUnixProjection()
fmt.Printf("UnixNanos: %d, InLeapSecond: %v\n", proj.Nanos, proj.IsLeapSecond)
```

### 5. Distributed Lock & Lease Validation (After / Before)

Non-blocking causality check to safely verify whether a distributed lock lease has expired.

```go
decision, status, reason := svc.After(leaseDeadline)
if status != gstime.StatusSynced || decision != gstime.CertainNo {
	// Lease may have expired or clock desynchronized: abort write
}
```

### 6. VM Migration & Discontinuity Fail-Fast

Hardware counters detect suspend/resume and snapshot rollbacks, transitioning to `StatusDesync`.

```go
now := svc.Now()
if now.Status == gstime.StatusDesync {
	// Reason: ReasonBoundTooOld (VM paused) or ReasonRawDiscontinuity (snapshot rollback)
}
```

## Running Examples and Tests

Execute all Go Example tests:

```bash
go test -v -run Example .
```

Run the standalone executable example:

```bash
go run ./examples/main.go
```

### Deterministic Simulation Testing (DST)

GSTime includes a deterministic simulation testing (DST) harness (`dst_test.go`) inspired by FoundationDB and Antithesis. It runs discrete time simulation with pseudorandom fault injection across PRNG seeds:
- **VM Snapshot Rollbacks**: Hardware counter rewinds and continuity token changes (verifying fail-fast `StatusDesync`).
- **Hypervisor Freezes / Suspends**: VM pause/migration for 10s–60s across validity horizons.
- **OS Clock Shaking**: Oscillator frequency wander (up to ±180 ppm) and sampling noise.
- **Byzantine Upstreams**: Outlier sources (+1 hour offsets) filtered by Marzullo/Hull consensus.
- **Strict Invariants**: Proves public clock monotonicity ($P_{k+1} \ge P_k$), true time containment ($P \pm \epsilon$ and $[L, U]$), and 100% bitwise trace reproducibility across identical seeds.

```bash
go test -v -run TestDST .
```

## Package Layout

```
gosuda.org/gstime
├── core/         # Semantic types, Q16.48 fixed-point math, GSTL1 leap codec
├── ntp/          # NTPv4 wire framing, 2036 era unfolding, reachability bitmap
├── nts/          # NTS-KE client, AEAD 15/30 (RFC 5297 / 8452), cookie lifecycle
├── source/       # Weighted regression, DP runs test, N-F consensus sweep (App. B)
├── clock/        # RawClock drivers, EstimateClock, slew planner, smear engine, clamp
├── assurance/    # Absolute anchor propagation, holdover tracking, status machine
├── publish/      # Lock-free atomic snapshot publication, publication guard
├── config/       # Canonical JSON configuration and RFC 8785 SHA-256 config hashing
├── telemetry/    # Atomic metrics collectors (offsets, errors, smear, re-anchors)
├── conformance/  # Conformance test suites (Levels A-F) and property verifications (P1-P14)
└── service.go    # Unified ClockService facade
```

## Verification

```bash
go test -v -race -count=1 ./...
gojgp check
```
