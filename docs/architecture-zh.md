# 架构与密码学设计

[English](./architecture.md)

应用实现了一个分层的密码学流水线，以确保零知识存储和安全访问委托。

## 1. 密码学原语

| 原语 | 算法 | 用途 |
|-----------|-----------|-------|
| 对称加密 | AES-CTR (128-bit) | 文件块、文件元数据和用户结构的认证加密 |
| 消息认证 | HMAC-SHA512 | 完整性验证（先加密后 MAC） |
| 公钥加密 | RSA-OAEP (2048-bit) with SHA-512 | 共享邀请的安全密钥交换 |
| 数字签名 | RSA-PKCS1.5 (2048-bit) with SHA-512 | 共享邀请的不可否认性和验证 |
| 密钥派生 | Argon2id | 从用户密码派生出主密钥 |
| 密钥多样化 | HashKDF (基于 HMAC) | 从单个主密钥派生出特定用途的子密钥（加密 vs MAC） |
| 哈希 | SHA-512 | 确定性 UUID 生成、文件名加盐 |

## 2. 密钥层次

```
用户密码
    └── Argon2id (以用户名为盐)
            └── 主密钥 (MasterKey)
                    ├── HashKDF(..., "enc")      → 加密密钥 (AES-CTR)
                    ├── HashKDF(..., "mac")      → MAC 密钥 (HMAC-SHA512)
                    ├── HashKDF(..., filename)   → 个人密钥
                    │       ├── HashKDF(..., "personal_enc") → 个人加密密钥
                    │       └── HashKDF(..., "personal_mac") → 个人 MAC 密钥
                    └── (RSA 密钥对, DS 密钥对)
```

## 3. 数据结构

- **文件分块：** 文件被分割为 512 字节的 `FileBlock` 块，每块使用从随机 `FileKey` 派生的文件特定密钥独立加密。
- **Inode：** 跟踪文件总大小和有序的块 UUID 列表。作为单个数据块在文件密钥下加密和 MAC 保护。
- **MailboxNode：** 每个用户（所有者或共享者）的指针，包含 `FileKey` 和 `InodeUUID`，使用邮箱特定密钥加密。每个用户拥有自己的 MailboxNode。
- **访问记录（Access）：** 将文件名映射到所有者的 MailboxNode UUID/密钥，并维护一个共享树（`Children` 映射），记录所有直接共享者，用于撤销操作。
- **邀请（Invitation）：** 包含 `MailboxUUID` 和 `MailboxKey` 的加密载荷，通过 RSA-OAEP + 数字签名传输以授予访问权限。
- **用户结构（User Struct）：** 包含用户名、RSA 私钥、DS 签名密钥、Argon2 派生主密钥以及已知文件访问指针的映射。在用户派生密钥下加密后存储在 Datastore 中。

## 4. 密码学流程

```
StoreFile (存储文件):
  content → ByteToBlock (512B 块)
          → encryptAndMAC(block, fEncKey, fMacKey) 对每个块
          → Inode{Size, BlockUUIDs} → encryptAndMAC → DatastoreSet
          → MailboxNode{FileKey, InodeUUID} → encryptAndMAC(mailbox keys) → DatastoreSet
          → Access{MymailboxUUID, MymailboxKey} → encryptAndMAC(personal keys) → DatastoreSet

LoadFile (加载文件):
  Access UUID → DatastoreGet → decryptAndVerify(personal keys)
             → Access{MymailboxUUID, MymailboxKey}
             → MailboxNode → DatastoreGet → decryptAndVerify(mailbox keys)
             → MailboxNode{FileKey, InodeUUID}
             → Inode → DatastoreGet → decryptAndVerify(file keys)
             → Blocks → DatastoreGet → decryptAndVerify(file keys)
             → BlockYToByte → content

AppendToFile (追加文件):
  与 StoreFile 的块创建相同，但追加到现有 inode 的 BlockUUIDs
  并更新 Size。不重新加密现有块。

CreateInvitation (创建邀请):
  → 解密自己的 MailboxNode
  → 为接收者创建新的 MailboxNode（相同 FileKey/InodeUUID）
  → 加密邀请 (RSA-OAEP) + 签名 (RSA-PKCS1.5)
  → 更新发送者的 Access.Children

AcceptInvitation (接受邀请):
  → 验证签名 + 解密邀请 (RSA-OAEP)
  → 创建指向收到的 MailboxNode 的本地 Access

RevokeAccess (撤销访问):
  → 生成新的 FileKey
  → 用新密钥重新加密所有块和 inode
  → 创建所有者的新 MailboxNode
  → 用新 FileKey 更新剩余子节点的 MailboxNode
  → 从 Children 中移除被撤销的用户
```

## 5. 二进制协议

服务端实现了一套轻量级二进制协议，运行在 TCP/TLS 之上：

| 操作码 | 操作 | 格式 |
|--------|------|------|
| `0x01` | GET | `[op:1][keyLen:4][key]` → `[status:1][valLen:4][val]` |
| `0x02` | SET | `[op:1][keyLen:4][key][valLen:4][val]` → `[status:1]` |
| `0x03` | DELETE | `[op:1][keyLen:4][key]` → `[status:1]` |

状态码：`0x00` 成功，`0x01` 未找到，`0x02` 错误。
