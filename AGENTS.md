# ShareLock

End-to-end encrypted file storage and sharing (Go 1.20, BadgerDB). Module: `github.com/cs161-staff/project2-starter-code`.

## Commands

```bash
make all           # build binaries + generate dev TLS cert
make build         # build binaries only
make vet           # go vet ./...
make clean         # rm binaries, cert.pem, key.pem
```

## Test

```bash
make test                    # go test ./... -count=1 (disables cache)
make test-app                # internal/client/app_test/ (Ginkgo)
make test-encryption         # internal/client/encryption_test/ (Ginkgo)
make test-unit               # internal/client/encryption/... (standard testing)
make test-handler            # internal/server/handler/
make test-store              # internal/server/store/
make test-integration        # internal/integration_test/ -timeout=120s
make test-userlib            # internal/userlib/
```

Focused verify: `go test -v -count=1 ./internal/client/encryption/`

## Run

```bash
./sharelock-server -tls=false -address :8080 -dir ./data &   # dev server
./sharelock init -u alice -p secret                           # create + login
./sharelock login -u alice -p secret                          # login (session cached)
./sharelock store -f hello.txt                                # store file from disk
./sharelock load -f hello.txt                                 # load + print
./sharelock share -f hello.txt -r bob                         # returns UUID
./sharelock accept -s alice -i <uuid> -f shared.txt
./sharelock revoke -f hello.txt -r bob
./sharelock host add dev localhost:8080                       # add server in .hosts
SHARELOCK_HOST=dev ./sharelock store -f hello.txt             # select host
```

## Architecture

- **Entrypoints**: `cmd/client/main.go`, `cmd/server/main.go`
- **Client layers**: CLI -> `internal/client/app/app.go` -> `internal/client/encryption/service.go` (UserService, FileService, InvitationService) -> `internal/userlib/` (network/crypto primitives)
- **Server**: BadgerDB KV store (`internal/server/`). Zero-knowledge — all crypto client-side.
- **Storage abstraction**: Services depend on `StorageService`/`KeyStoreService` interfaces (`internal/client/encryption/memory.go`). `MemoryStorage`/`MemoryKeyStore` for tests; `UserlibStorage`/`UserlibKeyStore` for production (wraps `userlib.Datastore*`/`Keystore*`).
- **Key hierarchy**: MasterKey -> PersonalKey (per-file, via `getPersonalKey`) -> FileKey (per-file, via `getFileKeys`) -> MailboxKey (per-pointer, via `getMailKeys`)

## Key files

| File | Purpose |
|------|---------|
| `internal/client/encryption/service.go` | All three service implementations |
| `internal/client/encryption/utils.go` | encryptAndMAC, decryptAndVerify, key derivation helpers |
| `internal/client/encryption/memory.go` | StorageService/KeyStoreService interfaces + test impls |
| `internal/client/encryption/encryption.go` | User, Inode struct defs |
| `internal/client/encryption/access.go` | MailboxNode, Access, Invitation structs |
| `internal/client/encryption/File.go` | 512-byte block splitting |
| `internal/client/app/app.go` | Client struct wiring services together |
| `cmd/client/main.go` | CLI flag parsing, session management |
| `cmd/server/main.go` | Server flags: -address, -dir, -cert, -key, -tls |

## Conventions

- All crypto client-side; server stores opaque encrypted blobs
- Encrypt-then-MAC: ciphertext + 64-byte HMAC-SHA512
- Error messages include context (username, filename)
- Session cached to disk (cleared via `sharelock logout`)
- Integration tests (app_test, encryption_test) use Ginkgo/Gomega; unit tests (encryption/*) use standard `testing` package
- `UserlibKeyStore` caches keys per-instance + falls through to global `userlib.Keystore*`. In tests, isolate with `userlib.KeystoreClear()` or use `MemoryKeyStore` to avoid cross-test key contamination.
