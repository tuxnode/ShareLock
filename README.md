# ShareLock

[**简体中文版本**](./README-zh.md)

A cryptographically secure, decentralized-trust file storage and sharing system designed to operate safely over untrusted cloud infrastructure.

**ShareLock** extends the core architectural principles of the UC Berkeley CS161 security framework into an end-to-end encrypted (E2EE) file sharing application. It guarantees confidentiality, integrity, and authenticity for all user data, even in the event of a total server-side compromise.

---

## Threat Model & Security Guarantees

The system is engineered against an **Active Malicious Adversary** who has full control over the storage server (Datastore) and network traffic.

- **Confidentiality:** Unauthorized users (including the storage provider) learn nothing about file contents, filenames, or the sharing graph.
- **Integrity & Authenticity:** Any unauthorized modification or tampering is instantly detected.
- **Revocation Efficiency:** Revoked users immediately lose access to all future file updates.

---

## Key Features

- **End-to-End Encryption (E2EE):** All encryption/decryption occurs client-side. Keys never leave the local device.
- **Granular Access Control:** Share files with specific users via encrypted invitation pointers.
- **Instant Revocation:** Dynamic re-keying isolates revoked users and transparently updates remaining sharees.
- **Append Optimization:** Append to large files without re-encrypting the entire structure.
- **Encrypt-Then-MAC:** All ciphertexts are authenticated with HMAC-SHA512 before storage.
- **TLS-Encrypted Streaming:** The `read` command downloads files over a TLS-encrypted TCP connection.

---

## Getting Started

### Prerequisites

- Go 1.20+
- Supported OS: Linux, macOS, Windows

### Quick Start

```bash
git clone git@github.com:tuxnode/ShareLock.git
cd ShareLock
make all          # build binaries + generate dev TLS certificate
```

### Quick Demo

```bash
# Start the server
./sharelock-server -tls=false -address :8080 -dir ./data &

# Create users
./sharelock init -u alice -p pass123
./sharelock init -u bob   -p pass456

# Store a file
echo "Hello from Alice!" > hello.txt
./sharelock store -f hello.txt

# Load it back
./sharelock load -f hello.txt

# Share with Bob
invite=$(./sharelock share -f hello.txt -r bob)

# Bob accepts
./sharelock accept -s alice -i "$invite" -f shared.txt
./sharelock load -f shared.txt

# Alice revokes Bob's access
./sharelock revoke -f hello.txt -r bob
```

---

## CLI Usage

### Commands

| Command | Description | Example |
|---------|-------------|---------|
| `init` | Create a new user account | `sharelock init -u alice -p secret` |
| `login` | Login as existing user | `sharelock login -u alice -p secret` |
| `store` | Store a file from disk | `sharelock store -f hello.txt` |
| `load` | Load and print file contents | `sharelock load -f hello.txt` |
| `append` | Append content to a file | `sharelock append -f hello.txt -c " more"` |
| `share` | Share a file with another user | `sharelock share -f hello.txt -r bob` |
| `accept` | Accept a file sharing invitation | `sharelock accept -s alice -i <uuid> -f file.txt` |
| `revoke` | Revoke a user's access | `sharelock revoke -f hello.txt -r bob` |
| `read` | Read file via TLS streaming | `sharelock read -f hello.txt -a localhost:8080` |
| `host` | Manage server connections | `sharelock host add dev localhost:8080` |
| `help` | Show help | `sharelock help` |

Run `sharelock <command> --help` for detailed flags on any command.

### Host Management

```bash
./sharelock host add default localhost:8080
./sharelock host add dev localhost:8080 --tls=false
SHARELOCK_HOST=dev ./sharelock store -f ./hello.txt
./sharelock host list
```

---

## Server

The KV Store server (`sharelock-server`) provides encrypted key-value storage using BadgerDB. All encryption is performed client-side — the server is zero-knowledge.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-address` | `localhost:8080` | Listen address |
| `-dir` | `./data` | Data directory for BadgerDB |
| `-cert` | — | TLS certificate file (required when `-tls=true`) |
| `-key` | — | TLS private key file (required when `-tls=true`) |
| `-tls` | `true` | Enable TLS encryption |

### Examples

```bash
# TLS mode (production)
./sharelock-server -address :8080 -dir ./data -cert cert.pem -key key.pem

# Plain TCP mode (development)
./sharelock-server -tls=false -address :8080 -dir ./data
```

---

## Testing

```bash
make test              # run all tests
make test-app          # app client tests
make test-encryption   # encryption integration tests
make test-unit         # encryption unit tests
make test-handler      # handler protocol tests
make test-store        # KV store tests
make test-integration  # server TLS integration tests
make test-userlib      # userlib tests
```

See [Testing Guide](./docs/testing.md) for detailed documentation and benchmarks.

---

## Project Structure

```
.
├── cmd/
│   ├── client/main.go           # CLI entry point
│   └── server/main.go           # KV Store server
├── internal/
│   ├── client/
│   │   ├── encryption/          # Crypto: User, FileService, InvitationService
│   │   ├── app/app.go           # Application business logic
│   │   ├── netstream/           # TLS-encrypted file streaming
│   │   └── config/              # .hosts file management
│   ├── server/
│   │   ├── server.go            # TLS listener, goroutine-per-conn
│   │   ├── store/store.go       # BadgerDB KV store
│   │   └── handler/handler.go   # Binary protocol handler
│   └── userlib/                 # Crypto primitives + network storage
├── docs/
│   ├── architecture.md          # Cryptographic design details
│   ├── api.md                   # User Library & Service API reference
│   └── testing.md               # Testing guide
├── Makefile
├── go.mod
└── LICENSE
```

---

## Documentation

- [Architecture & Cryptographic Design](./docs/architecture.md)
- [API Reference](./docs/api.md)
- [Testing Guide](./docs/testing.md)
- [Chinese Version (中文)](./README-zh.md)

---

## License

This project is based on starter code for UC Berkeley CS161 (Computer Security) Project 2. All rights reserved.
