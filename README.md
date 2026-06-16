# ShareLock

[**简体中文版本**](./README-zh.md)

A cryptographically secure, decentralized-trust file storage and sharing system designed to operate safely over untrusted cloud infrastructure.

**ShareLock** extends the core architectural principles of the UC Berkeley CS161 security framework into an end-to-end encrypted (E2EE) file sharing application. It guarantees confidentiality, integrity, and authenticity for all user data, even in the event of a total server-side compromise.

---

## Threat Model & Security Guarantees

The system is engineered against an **Active Malicious Adversary** who has full control over the storage server (Datastore) and network traffic.

### Security Goals

- **Confidentiality:** Unauthorized users (including the storage provider) learn absolutely nothing about file contents, file lengths, filenames, or the sharing graph topology.
- **Integrity & Authenticity:** Any unauthorized modification, tampering, or rollback of file data or sharing metadata by the server or an attacker is instantly detected.
- **Revocation Efficiency:** When a file owner revokes access from a user, that user immediately loses access to all future updates of the file, and their cryptographic access paths are completely invalidated.

---

## Architecture & Cryptographic Design

The application implements a layered cryptographic pipeline to ensure zero-knowledge storage and secure access delegation.

### 1. Cryptographic Primitives

| Primitive | Algorithm | Usage |
|-----------|-----------|-------|
| Symmetric Encryption | AES-CTR (128-bit) | Authenticated encryption of file chunks, file metadata, and user structs |
| Message Authentication | HMAC-SHA512 | Integrity verification (encrypt-then-MAC) |
| Public-Key Encryption | RSA-OAEP (2048-bit) with SHA-512 | Secure key exchange for sharing invitations |
| Digital Signatures | RSA-PKCS1.5 (2048-bit) with SHA-512 | Non-repudiation and verification of sharing invites |
| Key Derivation | Argon2id | Master key derivation from user password |
| Key Diversification | HashKDF (HMAC-based) | Deriving purpose-specific sub-keys (encryption vs. MAC) from a single master key |
| Hashing | SHA-512 | Deterministic UUID generation, filename salting |

### 2. Key Hierarchy

```
User Password
    └── Argon2id (salted by username)
            └── MasterKey
                    ├── HashKDF(..., "enc")      → Encryption Key  (AES-CTR)
                    ├── HashKDF(..., "mac")      → MAC Key         (HMAC-SHA512)
                    ├── HashKDF(..., filename)   → Personal Key
                    │       ├── HashKDF(..., "personal_enc") → Personal Encryption Key
                    │       └── HashKDF(..., "personal_mac") → Personal MAC Key
                    └── (RSA keypair, DS keypair)
```

### 3. Data Structures

- **File Blocking:** Files are split into 512-byte `FileBlock` chunks, each encrypted independently with a file-specific key derived from a random `FileKey`.
- **Inode:** Tracks total file size and an ordered list of block UUIDs. Encrypted and MAC'd as a single blob under the file key.
- **MailboxNode:** Per-user pointer containing `FileKey` and `InodeUUID`, encrypted with a mailbox-specific key. Each user (owner or sharee) has their own MailboxNode.
- **Access Record:** Maps a filename to the owner's MailboxNode UUID/key and maintains a sharing tree (`Chidren` map) of all direct sharees for revocation.
- **Invitation:** Encrypted payload containing a `MailboxUUID` and `MailboxKey`, transmitted via RSA-OAEP + digital signature to grant access.
- **User Struct:** Contains the username, RSA private key, DS signing key, Argon2-derived master key, and a map of known file access pointers. Encrypted under user's derived keys and stored in the Datastore.

### 4. Cryptographic Flow

```
StoreFile:
  content → ByteToBlock (512B chunks)
          → encryptAndMAC(block, fEncKey, fMacKey) for each block
          → Inode{Size, BlockUUIDs} → encryptAndMAC → DatastoreSet
          → MailboxNode{FileKey, InodeUUID} → encryptAndMAC(mailbox keys) → DatastoreSet
          → Access{MymailboxUUID, MymailboxKey} → encryptAndMAC(personal keys) → DatastoreSet

LoadFile:
  Access UUID → DatastoreGet → decryptAndVerify(personal keys)
             → Access{MymailboxUUID, MymailboxKey}
             → MailboxNode → DatastoreGet → decryptAndVerify(mailbox keys)
             → MailboxNode{FileKey, InodeUUID}
             → Inode → DatastoreGet → decryptAndVerify(file keys)
             → Blocks → DatastoreGet → decryptAndVerify(file keys)
             → BlockYToByte → content

AppendToFile:
  Same as StoreFile block creation, but appends to existing inode's BlockUUIDs
  and updates Size. Does not re-encrypt existing blocks.

CreateInvitation:
  → Decrypt own MailboxNode
  → Create new MailboxNode for recipient (same FileKey/InodeUUID)
  → Encrypt invitation (RSA-OAEP) + sign (RSA-PKCS1.5)
  → Update sender's Access.Chidren

AcceptInvitation:
  → Verify signature + decrypt invitation (RSA-OAEP)
  → Create local Access pointing to received MailboxNode

RevokeAccess:
  → Generate new FileKey
  → Re-encrypt all blocks and inode with new key
  → Create new owner MailboxNode
  → Update remaining children's MailboxNodes with new FileKey
  → Remove revoked user from Chidren
```

---

## Key Features

- **End-to-End Encryption (E2EE):** All encryption/decryption occurs strictly client-side. Keys never leave the local device in plaintext.
- **Granular Access Control:** Seamlessly share files with specific users via encrypted invitation pointers backed by the MailboxNode sharing tree.
- **Instant Revocation:** Dynamic re-keying mechanism rotates the file key on revocation, isolates revoked users, and transparently updates remaining sharees without breaking their access.
- **Append Optimization:** Append to existing large files without downloading or re-encrypting the entire file structure.
- **Encrypt-Then-MAC:** All ciphertexts are authenticated with HMAC-SHA512 before storage, guaranteeing integrity.
- **TLS-Encrypted Streaming:** The `read` command downloads files over a TLS-encrypted TCP connection via the `netstream` module, ensuring confidentiality in transit.

---

## Implementation Status

| Component | Status |
|-----------|--------|
| `InitUser` | ✅ Implemented |
| `GetUser` | ✅ Implemented |
| `StoreFile` | ✅ Implemented |
| `LoadFile` | ✅ Implemented |
| `AppendToFile` | ✅ Implemented |
| `CreateInvitation` | ✅ Implemented |
| `AcceptInvitation` | ✅ Implemented |
| `RevokeAccess` | ✅ Implemented |
| `ReadFile` (TLS streaming) | ✅ Implemented |
| CLI (`cmd/client`) | ✅ Implemented |
| Netstream (`internal/netstream`) | ✅ Implemented |
| KV Store Server (`cmd/server`) | ✅ Implemented |
| BadgerDB Store (`internal/server/store`) | ✅ Implemented |
| Binary Protocol Handler (`internal/server/handler`) | ✅ Implemented |

---

## Getting Started

### Prerequisites

- Go 1.20+
- Supported OS: Linux, macOS, Windows

### Installation

```bash
# Clone the repository
git clone git@github.com:tuxnode/ShareLock.git
cd ShareLock

# Build all packages
go build ./...
```

### Running Tests

The project uses [Ginkgo v2](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/) for testing. See the [Testing Guide](./docs/testing.md) for detailed documentation, including [benchmarks](./docs/testing.md#benchmarks).

```bash
# Run all tests
go test ./...

# Run encryption unit tests (white-box)
go test -v ./internal/client/encryption/...

# Run client integration tests (black-box)
go test -v ./internal/client/encryption_test/...

# Run user library tests
go test -v ./project2-userlib/...

# Run a specific test suite
go test -v -run "TestSetupAndExecution" ./...
```

### CLI Usage

```bash
# Build the client binary
go build -o sharelock ./cmd/client

# Initialize a user
./sharelock inituser -username alice -password secret

# Store a file
./sharelock storefile -filename hello.txt -content "Hello, World!"

# Load a file
./sharelock loadfile -filename hello.txt

# Share a file
invite=$(./sharelock createinvitation -filename hello.txt -recipient bob)
./sharelock acceptinvitation -sender alice -invitation $invite -filename hello.txt

# Revoke access
./sharelock revokeaccess -filename hello.txt -recipient bob

# Read a file via TLS-encrypted stream
./sharelock read -filename hello.txt -address localhost:8080

# Build and run the KV Store server (requires TLS cert)
go build -o sharelock-server ./cmd/server
./sharelock-server -address :8080 -dir ./data -cert cert.pem -key key.pem
```

### Linting

```bash
go vet ./...
```

---

## Project Structure

```
.
├── cmd/
│   ├── client/
│   │   └── main.go              # CLI entry point (subcommand dispatch)
│   └── server/
│       └── main.go              # KV Store server (TLS + BadgerDB)
├── internal/
│   ├── client/
│   │   ├── encryption/
│   │   │   ├── access.go        # Data structures: MailboxNode, Access, Invitation, ChildrenInfo
│   │   │   ├── encryption.go    # Core client: User struct, InitUser, GetUser, StoreFile, etc.
│   │   │   ├── encryption_unittest.go  # White-box unit tests (Ginkgo/Gomega)
│   │   │   ├── File.go          # File block splitting/merging utilities
│   │   │   └── utils.go         # Cryptographic helpers: encryptAndMAC, decryptAndVerify, key derivation
│   │   ├── app/
│   │   │   └── app.go           # Application-level client business logic layer
│   │   └── netstream/
│   │       └── netstream.go     # TLS-encrypted file streaming (FileSeander / FileReceiver)
│   ├── client/encryption_test/
│   │   └── encryption_test.go   # Black-box integration tests
│   ├── client/app_test/
│   │   └── app_test.go          # App client integration tests
│   ├── netstream/
│   │   └── netstream.go         # TLS-encrypted file streaming (FileSeander / FileReceiver)
│   ├── server/
│   │   ├── server.go            # TLS listener loop, goroutine-per-conn
│   │   ├── store/
│   │   │   └── store.go         # BadgerDB KV store (Get, Set, Delete, Exists)
│   │   └── handler/
│   │       └── handler.go       # Binary protocol handler (GET 0x01 / SET 0x02 / DELETE 0x03)
│   └── integration_test/
│       └── server_test.go       # Server TLS integration tests
├── project2-userlib/            # Cryptographic library (Datastore, Keystore, primitives)
│   ├── userlib.go               # Core crypto primitives and storage interfaces
│   ├── userlib_test.go          # Library tests
│   └── go.mod
├── go.mod                       # Module definition
├── go.sum                       # Dependency checksums
├── CHANGELOG.md
├── project2-spec.pdf            # Original project specification
├── proj2.excalidraw             # Architecture diagram (Excalidraw format)
├── LICENSE
├── README.md
└── README-zh.md
```

---

## User Library API

The project relies on `github.com/cs161-staff/project2-userlib` which provides:

| Function | Purpose |
|----------|---------|
| `SymEnc(key, iv, plaintext)` | AES-CTR encryption |
| `SymDec(key, ciphertext)` | AES-CTR decryption |
| `PKEKeyGen()` | RSA-OAEP key pair generation |
| `PKEEnc(pk, plaintext)` | RSA-OAEP encryption |
| `PKEDec(sk, ciphertext)` | RSA-OAEP decryption |
| `DSKeyGen()` | RSA-PKCS1.5 signing key pair generation |
| `DSign(sk, msg)` | RSA-PKCS1.5 signing |
| `DSVerify(pk, msg, sig)` | RSA-PKCS1.5 signature verification |
| `Argon2Key(password, salt, keyLen)` | Argon2id key derivation |
| `HashKDF(key, context)` | HMAC-based key derivation |
| `Hash(data)` | SHA-512 hashing |
| `HMACEval(key, data)` | HMAC-SHA512 computation |
| `HMACEqual(a, b)` | Constant-time HMAC comparison |
| `RandomBytes(n)` | Cryptographically secure random bytes |
| `DatastoreGet(key)` | Retrieve from untrusted storage |
| `DatastoreSet(key, value)` | Store to untrusted storage |
| `DatastoreDelete(key)` | Delete from untrusted storage |
| `KeystoreGet(key)` | Retrieve from trusted public-key store |
| `KeystoreSet(key, value)` | Store to trusted public-key store |
| `DatastoreGetBandwidth()` | Measure storage bandwidth (testing) |

---

## License

This project is based on starter code for UC Berkeley CS161 (Computer Security) Project 2. All rights reserved.
