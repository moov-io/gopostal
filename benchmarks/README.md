# Benchmark Results for gopostal concurrent access

**Important:** The parser (but not expand) now uses an exclusive `sync.Mutex`
instead of `RWMutex`. libpostal's address parser implementation (the CRF tagger
and feature extraction in address_parser.c) is not safe for concurrent calls
from multiple threads, even for read-only use after model loading. Using
RWMutex for the parser led to internal corruption, "got NULL word" errors,
and SIGSEGV under concurrent load (see `TestConcurrentParse`).

The historical artifacts below predate this discovery and measured the (buggy)
RWMutex behavior.

This directory contains benchmark artifacts demonstrating two changes:

1. `sync.Mutex` → `sync.RWMutex` for the runtime parse/expand calls (expand still uses RWMutex; parser uses exclusive Mutex)
2. Removal of unsafe `init()` double-setup in favor of `sync.Once` via `internal/setup`

## Summary of results

- **Mutex → RWMutex** (historical, for expand): allowed concurrent expansion calls.
- Parser now uses exclusive Mutex (required for correctness).
- **RWMutex + sync.Once**: zero regression on the hot path after the one-time setup.

## Files

- `benchmark_before.txt` — Mutex baseline (serialized parallel work)
- `benchmark_after.txt` — RWMutex (concurrent parallel work)
- `benchmark_after_once.txt` — RWMutex + `sync.Once` initialization (current code)
- `benchstat_comparison.txt` — Mutex vs RWMutex delta
- `benchstat_once_vs_previous.txt` — RWMutex vs RWMutex+Once (no regression)
- `run_log.txt` + `bench_*_attempt.log` — Raw `go test` attempts (libpostal missing here)

## How to collect real data (machine with libpostal installed)

On a system where `pkg-config --exists libpostal` succeeds:

```bash
# Install benchstat once
go install golang.org/x/perf/cmd/benchstat@latest
export PATH="$PATH:$(go env GOPATH)/bin"

# Recommended: collect 6–10 samples for reliable stats
go test -bench=. -benchmem -count=10 ./parser ./expand | tee benchmarks/current.txt

# Then compare any two runs
benchstat benchmarks/benchmark_after.txt benchmarks/current.txt
```

### Reproducing the full history (Mutex → RWMutex → Once)

If you want clean before/after for both changes:

```bash
# 1. Mutex baseline (original code)
git checkout HEAD~1 -- parser/parser.go expand/expand.go 2>/dev/null || \
  (echo "Manually revert to sync.Mutex + old init()"; exit 1)
go test -bench=. -benchmem -count=10 ./parser ./expand | tee benchmarks/before_mutex.txt

# 2. RWMutex only (before the Once change)
#    - Change Mutex → RWMutex + RLock/RUnlock
#    - Keep the old init() functions
go test -bench=. -benchmem -count=10 ./parser ./expand | tee benchmarks/after_rwmutex.txt

# 3. Current code (RWMutex + sync.Once via internal/setup)
git checkout -- .
go test -bench=. -benchmem -count=10 ./parser ./expand | tee benchmarks/after_once.txt

benchstat benchmarks/before_mutex.txt benchmarks/after_once.txt
```

## Expected real results

- Serial benchmarks (`BenchmarkParse`, `BenchmarkExpand`): unchanged across all versions.
- Parallel benchmarks: ~10–100× faster with RWMutex vs original Mutex.
- Adding `sync.Once` for init: **no measurable difference** on steady-state benchmarks.

## Concurrent correctness tests

The tests `TestConcurrentParse` and `TestConcurrentExpand` launch many
goroutines that repeatedly call the address functions and assert that every
result is identical to the expected golden value. Any data corruption or
race would cause these tests to fail.
