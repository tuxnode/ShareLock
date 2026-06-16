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
| Netstream Tests | `netstream` | (none yet) | `internal/client/netstream/netstream.go` |

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

# Run a specific test suite by name
go test -v -run "TestApp" ./...

# Run a specific spec by description
go test -v ./internal/client/app_test/ --ginkgo.focus="RevokeAccess"
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

## Code Quality

```bash
# Run the Go vet checker
go vet ./...

# List all tests without running them
go test -list ".*" ./...
```
