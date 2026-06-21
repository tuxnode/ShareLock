# API Reference

[中文版](./api-zh.md)

## User Library API

The project relies on `internal/userlib` which provides:

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

## Service API

The project provides a service-based API with interface separation for better testability and flexibility.

### Interfaces

| Interface | Methods | Purpose |
|-----------|---------|---------|
| `StorageService` | `Get`, `Set`, `Delete` | Abstracts storage operations |
| `KeyStoreService` | `Get`, `Set` | Abstracts public key storage |

### Services

| Service | Methods | Purpose |
|---------|---------|---------|
| `UserService` | `InitUser`, `GetUser` | User lifecycle management |
| `FileService` | `StoreFile`, `LoadFile`, `AppendToFile` | File operations |
| `InvitationService` | `CreateInvitation`, `AcceptInvitation`, `RevokeAccess` | Sharing and revocation |

### Usage Example

```go
// Create storage implementations
storage := encryption.NewUserlibStorage()    // or NewMemoryStorage() for testing
keyStore := encryption.NewUserlibKeyStore()  // or NewMemoryKeyStore() for testing

// Create services
userService := encryption.NewUserService(storage, keyStore)
fileService := encryption.NewFileService(storage, keyStore)
invitationService := encryption.NewInvitationService(storage, keyStore)

// Initialize user
user, err := userService.InitUser("alice", "password")

// Store file
err = fileService.StoreFile(user, "hello.txt", []byte("Hello, World!"))

// Load file
content, err := fileService.LoadFile(user, "hello.txt")

// Share file
invPtr, err := invitationService.CreateInvitation(user, "hello.txt", "bob")

// Accept invitation
err = invitationService.AcceptInvitation(bobUser, "alice", invPtr, "hello.txt")

// Revoke access
err = invitationService.RevokeAccess(user, "hello.txt", "bob")
```
