# Testing Guide

[**中文版本**](./testing-zh.md)

## Overview

ShareLock uses [Ginkgo v2](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/) for both unit and integration testing. The test suite covers the cryptographic encryption layer, the application-level client, and the netstream file streaming module.

---

## Test Suites

| Suite | Package | Type | Location |
|-------|---------|------|----------|
| Encryption Unit Tests | `encryption` | White-box unit | `internal/client/encryption/encryption_unittest.go` |
| Encryption Integration Tests | `encryption_test` | Black-box integration | `internal/client/encryption_test/encryption_test.go` |
| App Client Tests | `app_test` | Black-box integration | `internal/client/app_test/app_test.go` |
| Netstream Tests | `netstream` | (none yet) | `internal/netstream/netstream.go` |
| KV Store Unit Tests | `store` | Unit | `internal/server/store/store_test.go` |
| Handler Protocol Tests | `handler` | Unit | `internal/server/handler/handler_test.go` |
| Server Integration Tests | `integration_test` | Integration (TLS) | `internal/integration_test/server_test.go` |

---

## Running Tests

```bash
# Run all tests
go test ./...

# Run encryption unit tests (white-box)
go test -v ./internal/client/encryption/...

# Run encryption integration tests (black-box)
go test -v ./internal/client/encryption_test/...

# Run app client integration tests (black-box)
go test -v ./internal/client/app_test/...

# Run KV store unit tests
go test -v ./internal/server/store/...

# Run handler protocol tests
go test -v ./internal/server/handler/...

# Run server TLS integration tests
go test -v ./internal/integration_test/...

# Run a specific test suite by name
go test -v -run "TestApp" ./...

# Run a specific spec by description
go test -v ./internal/client/app_test/ --ginkgo.focus="RevokeAccess"
```

### Running Benchmarks

```bash
# Run all benchmarks (all packages)
go test ./... -bench=. -benchtime=1s

# Run KV store benchmarks only
go test ./internal/server/store/ -bench=. -benchtime=1s

# Run handler protocol benchmarks
go test ./internal/server/handler/ -bench=. -benchtime=1s

# Run TLS end-to-end benchmarks
go test ./internal/integration_test/ -bench=. -benchtime=1s -timeout=120s

# Run with memory allocation stats
go test ./internal/server/store/ -bench=. -benchmem

# Filter benchmarks by name
go test ./internal/integration_test/ -bench=Parallel -benchtime=1s
```

---

## App Client Tests (`internal/client/app_test/app_test.go`)

This is the primary integration test suite for the application-layer `app.Client`. It tests all public methods through the complete crypto and storage pipeline.

### Test Groups

#### InitUser / GetUser (4 tests)
- **Single user init and get** — verifies basic lifecycle
- **Duplicate InitUser** — ensures re-initializing an existing user returns an error
- **Wrong password** — verifies `GetUser` rejects incorrect credentials
- **Non-existent user** — verifies `GetUser` returns an error for unknown users

#### StoreFile / LoadFile (4 tests)
- **Store and load** — basic round-trip
- **Empty content** — edge case for zero-length files
- **Non-existent file** — verifies `LoadFile` fails gracefully
- **Overwrite** — verifies that re-storing a file replaces the old content

#### AppendToFile (4 tests)
- **Append to existing file** — verifies content is correctly appended
- **Multiple appends** — verifies cumulative appends produce the expected composite content
- **Non-existent file** — verifies `AppendToFile` fails for missing files
- **Nil content** — verifies nil content is rejected

#### Invitations (4 tests)
- **Share file via invitation** — verifies `CreateInvitation` / `AcceptInvitation` flow
- **Shared user appends** — verifies that a sharee can modify the file
- **Invitation for non-existent file** — verifies error handling
- **Non-existent sender** — verifies acceptance fails for unknown sender

#### Multi-session Consistency (3 tests)
- **Cross-session read** — verifies data written by one session is visible to another
- **Cross-session append** — verifies appends propagate across sessions
- **Cross-session invitation** — verifies invitations created from one session work in another

#### RevokeAccess (5 tests)
- **Revoke direct sharee** — verifies revoked user loses access
- **Owner retains access** — verifies owner can still read after revocation
- **Cascade to indirect sharees** — verifies revocation propagates to sub-sharees
- **Revoked user cannot append** — verifies write operations are blocked
- **Owner can continue appending** — verifies owner's write capability is unaffected

---

## Encryption Integration Tests (`internal/client/encryption_test/encryption_test.go`)

These tests exercise the raw `encryption` package directly without the `app.Client` wrapper. They verify the same cryptographic flows as the app client tests but at a lower level.

### Test Groups

- **InitUser / GetUser** — basic user lifecycle
- **Single User Store/Load/Append** — full CRUD on a single user's file
- **Create/Accept Invite with Multi-session** — sharing across multiple client instances
- **Revoke Functionality** — revocation with cascading to sub-sharees

---

## Encryption Unit Tests (`internal/client/encryption/encryption_unittest.go`)

White-box tests that have access to internal struct fields. The `encryption_unittest.go` file explicitly states that it will **not** be graded — it exists for developers to validate internal implementation details.

---

## Writing New Tests

### Adding a test to the app client suite

```go
// In internal/client/app_test/app_test.go

It("should do something specific", func() {
    alice := &app.Client{}
    err := alice.InitUser("alice", "password")
    Expect(err).To(BeNil())

    err = alice.StoreFile("test.txt", []byte("data"))
    Expect(err).To(BeNil())

    data, err := alice.LoadFile("test.txt")
    Expect(err).To(BeNil())
    Expect(data).To(Equal([]byte("data")))
})
```

### Adding a new test file

Create a new test file in the appropriate `*_test` directory or alongside the package being tested. Follow the Ginkgo/Gomega convention:

```go
package <package>_test

import (
    "testing"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestSuite(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Suite Description")
}
```

---

## Benchmarks

Performance benchmarks measure three layers of the stack on real hardware.

### Benchmark Suites

| Suite | File | What it measures |
|-------|------|------------------|
| Store | `internal/server/store/store_bench_test.go` | Raw BadgerDB throughput: GET, SET, DELETE, parallel ops, value size scaling |
| Handler | `internal/server/handler/handler_bench_test.go` | In-process protocol cost: op dispatch, key/value encoding |
| TLS Integration | `internal/integration_test/server_bench_test.go` | End-to-end TLS: single-op latency, parallel clients, pipelining, large values |

### Reference Results (i5-11500 @ 2.70GHz)

**Store (BadgerDB, ZeroMQ LSM):**
| Benchmark | Time/op | Notes |
|-----------|---------|-------|
| `StoreGet` | ~713 ns | Near-memory speed |
| `StoreSet` | ~5.9 µs | fsync-bound; disable SyncWrites for ~1 µs |
| `StoreGetParallel` | ~601 ns | RLock scales well |
| `StoreSetParallel` | ~4.0 µs | Write lock contention visible |
| `StoreValueSize/64KB` | ~136 µs | Throughput: ~480 MB/s |

**Handler (in-process, no network):**
| Benchmark | Time/op | Notes |
|-----------|---------|-------|
| `HandlerGet` | ~747 ns | Negligible protocol overhead vs raw store |
| `HandlerSet` | ~5.9 µs | Protocol + store combined |
| `HandlerSetValueSize/64KB` | ~85 µs | Pure store overhead dominates |

**TLS End-to-End (127.0.0.1, self-signed cert):**
| Benchmark | Time/op | Notes |
|-----------|---------|-------|
| `TLS_Get` | ~13 µs | TLS handshake + encryption + round trip |
| `TLS_Set` | ~20 µs | |
| `TLS_SetGet` | ~36 µs | Combined read + write |
| `TLS_GetParallel` | ~1.8 µs | Connection pool scales |
| `TLS_SetParallel` | ~6.7 µs | |
| `TLS_Pipeline` | ~10 µs / op | 100-batch pipeline amortizes overhead |
| `TLS_ValueSize/64KB` | ~260 µs | Throughput: ~250 MB/s |

### Interpreting Results

- **Store** benchmarks reflect raw database performance; any improvement here benefits all layers.
- **Handler** overhead is <5% for most operations — the protocol is not a bottleneck.
- **TLS** adds 12–15 µs per operation vs in-process handler, dominated by crypto + network round trip.
- For maximum throughput, use **pipelining** (batch writes before reading responses) or **parallel connections**.

```bash
# Reproduce results on your hardware
go test ./internal/server/store/ -bench=. -benchtime=1s
go test ./internal/server/handler/ -bench=. -benchtime=1s
go test ./internal/integration_test/ -bench=. -benchtime=1s -timeout=120s
```

---

## Code Quality

```bash
# Run the Go vet checker
go vet ./...

# List all tests without running them
go test -list ".*" ./...
```
