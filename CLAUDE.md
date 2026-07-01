# ShareLock

End-to-end encrypted file storage and sharing system built in Go.

## Project Structure

```
.
├── cmd/
│   ├── client/main.go           # CLI entry point
│   └── server/main.go           # KV Store server
├── internal/
│   ├── client/
│   │   ├── app/app.go           # Application business logic
│   │   ├── encryption/          # Crypto services & utils
│   │   │   ├── service.go       # UserService, FileService, InvitationService
│   │   │   ├── encryption.go    # User/Inode struct definitions
│   │   │   ├── access.go        # MailboxNode, Access, Invitation structs
│   │   │   ├── utils.go         # encryptAndMAC, decryptAndVerify, key derivation
│   │   │   ├── memory.go        # In-memory storage for testing
│   │   │   └── File.go          # 512-byte block splitting
│   │   ├── config/config.go     # .hosts file management
│   │   └── netstream/           # TLS file streaming
│   ├── server/                  # BadgerDB KV store server
│   └── userlib/                 # Crypto primitives
├── docs/
└── Makefile
```

## Build

```bash
make all          # build binaries + generate dev TLS certificate
go build ./...    # compile all packages
```

## Test

```bash
make test                    # run all tests
make test-app               # app client tests
make test-encryption        # encryption integration tests
make test-unit              # encryption unit tests
make test-handler           # handler protocol tests
make test-store             # KV store tests
make test-integration       # server TLS integration tests

# Or directly:
go test ./internal/client/... -v
go test ./internal/client/encryption/ -v
```

## CLI Commands

```bash
./sharelock init -u alice -p secret      # Create user
./sharelock login -u alice -p secret     # Login
./sharelock store -f hello.txt           # Store file
./sharelock load -f hello.txt            # Load file
./sharelock append -f hello.txt -c "more" # Append
./sharelock share -f hello.txt -r bob    # Share (returns UUID)
./sharelock accept -s alice -i <uuid> -f shared.txt  # Accept share
./sharelock revoke -f hello.txt -r bob   # Revoke access
./sharelock list                         # List files
./sharelock host add dev localhost:8080   # Add server
```

## Architecture

### Client Layers
1. **CLI** (`cmd/client/main.go`): Command parsing, output formatting
2. **App** (`internal/client/app/app.go`): Business logic coordination
3. **Services** (`internal/client/encryption/service.go`): UserService, FileService, InvitationService
4. **Crypto Utils** (`internal/client/encryption/utils.go`): Encryption, MAC, key derivation

### Crypto Design
- **Symmetric**: AES + HMAC-SHA512 (Encrypt-Then-MAC)
- **Key Derivation**: Argon2 + HashKDF
- **Public Key**: PKE for invitation encryption, DS for signing
- **Key Hierarchy**: MasterKey → PersonalKey → FileKey → MailboxKey

### Data Structures
- `User`: Username, PKEPrivateKey, DSSignKey, MasterKey, Files map
- `Access`: MailboxUUID, MailboxKey, Children (sharing tree)
- `MailboxNode`: FileKey, InodeUUID
- `Inode`: Size, BlockUUIDs
- `FileBlock`: 512-byte encrypted data chunks

## Key Files

- `internal/client/encryption/service.go` - All service implementations
- `internal/client/encryption/utils.go` - Core crypto functions
- `internal/client/encryption/memory.go` - StorageService/KeyStoreService interfaces
- `cmd/client/main.go` - CLI entry point with all commands

## Testing Patterns

- Uses Ginkgo/Gomega for BDD-style tests
- `app_test/app_test.go` - Integration tests (27+ specs)
- `encryption_test/` - Service-level tests
- `encryption/utils_test.go` - Unit tests for crypto functions
- `netstream/netstream_test.go` - File transfer tests

## Conventions

- Error messages include context (username, filename)
- Services use interface-based storage for testability
- All crypto operations happen client-side (zero-knowledge server)
- Files are encrypted with random keys, stored as linked blocks
