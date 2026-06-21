# Architecture & Cryptographic Design

[中文版](./architecture-zh.md)

The application implements a layered cryptographic pipeline to ensure zero-knowledge storage and secure access delegation.

## 1. Cryptographic Primitives

| Primitive | Algorithm | Usage |
|-----------|-----------|-------|
| Symmetric Encryption | AES-CTR (128-bit) | Authenticated encryption of file chunks, file metadata, and user structs |
| Message Authentication | HMAC-SHA512 | Integrity verification (encrypt-then-MAC) |
| Public-Key Encryption | RSA-OAEP (2048-bit) with SHA-512 | Secure key exchange for sharing invitations |
| Digital Signatures | RSA-PKCS1.5 (2048-bit) with SHA-512 | Non-repudiation and verification of sharing invites |
| Key Derivation | Argon2id | Master key derivation from user password |
| Key Diversification | HashKDF (HMAC-based) | Deriving purpose-specific sub-keys (encryption vs. MAC) from a single master key |
| Hashing | SHA-512 | Deterministic UUID generation, filename salting |

## 2. Key Hierarchy

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

## 3. Data Structures

- **File Blocking:** Files are split into 512-byte `FileBlock` chunks, each encrypted independently with a file-specific key derived from a random `FileKey`.
- **Inode:** Tracks total file size and an ordered list of block UUIDs. Encrypted and MAC'd as a single blob under the file key.
- **MailboxNode:** Per-user pointer containing `FileKey` and `InodeUUID`, encrypted with a mailbox-specific key. Each user (owner or sharee) has their own MailboxNode.
- **Access Record:** Maps a filename to the owner's MailboxNode UUID/key and maintains a sharing tree (`Children` map) of all direct sharees for revocation.
- **Invitation:** Encrypted payload containing a `MailboxUUID` and `MailboxKey`, transmitted via RSA-OAEP + digital signature to grant access.
- **User Struct:** Contains the username, RSA private key, DS signing key, Argon2-derived master key, and a map of known file access pointers. Encrypted under user's derived keys and stored in the Datastore.

## 4. Cryptographic Flow

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
  → Update sender's Access.Children

AcceptInvitation:
  → Verify signature + decrypt invitation (RSA-OAEP)
  → Create local Access pointing to received MailboxNode

RevokeAccess:
  → Generate new FileKey
  → Re-encrypt all blocks and inode with new key
  → Create new owner MailboxNode
  → Update remaining children's MailboxNodes with new FileKey
  → Remove revoked user from Children
```

## 5. Binary Protocol

The server implements a lightweight binary protocol over TCP/TLS:

| Opcode | Operation | Format |
|--------|-----------|--------|
| `0x01` | GET | `[op:1][keyLen:4][key]` → `[status:1][valLen:4][val]` |
| `0x02` | SET | `[op:1][keyLen:4][key][valLen:4][val]` → `[status:1]` |
| `0x03` | DELETE | `[op:1][keyLen:4][key]` → `[status:1]` |

Status codes: `0x00` OK, `0x01` Not Found, `0x02` Error.
