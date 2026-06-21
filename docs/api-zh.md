# API 参考

[English](./api.md)

## 用户库 API

项目依赖 `internal/userlib`，提供以下功能：

| 函数 | 用途 |
|----------|---------|
| `SymEnc(key, iv, plaintext)` | AES-CTR 加密 |
| `SymDec(key, ciphertext)` | AES-CTR 解密 |
| `PKEKeyGen()` | RSA-OAEP 密钥对生成 |
| `PKEEnc(pk, plaintext)` | RSA-OAEP 加密 |
| `PKEDec(sk, ciphertext)` | RSA-OAEP 解密 |
| `DSKeyGen()` | RSA-PKCS1.5 签名密钥对生成 |
| `DSign(sk, msg)` | RSA-PKCS1.5 签名 |
| `DSVerify(pk, msg, sig)` | RSA-PKCS1.5 签名验证 |
| `Argon2Key(password, salt, keyLen)` | Argon2id 密钥派生 |
| `HashKDF(key, context)` | 基于 HMAC 的密钥派生 |
| `Hash(data)` | SHA-512 哈希 |
| `HMACEval(key, data)` | HMAC-SHA512 计算 |
| `HMACEqual(a, b)` | 常量时间 HMAC 比较 |
| `RandomBytes(n)` | 密码学安全随机字节 |
| `DatastoreGet(key)` | 从不可信存储中检索 |
| `DatastoreSet(key, value)` | 存储到不可信存储 |
| `DatastoreDelete(key)` | 从不可信存储中删除 |
| `KeystoreGet(key)` | 从可信公钥存储中检索 |
| `KeystoreSet(key, value)` | 存储到可信公钥存储 |
| `DatastoreGetBandwidth()` | 测量存储带宽（测试用） |

## Service API

项目提供基于接口分离的 Service API，提升可测试性和灵活性。

### 接口

| 接口 | 方法 | 用途 |
|------|------|------|
| `StorageService` | `Get`, `Set`, `Delete` | 抽象存储操作 |
| `KeyStoreService` | `Get`, `Set` | 抽象公钥存储 |

### 服务

| 服务 | 方法 | 用途 |
|------|------|------|
| `UserService` | `InitUser`, `GetUser` | 用户生命周期管理 |
| `FileService` | `StoreFile`, `LoadFile`, `AppendToFile` | 文件操作 |
| `InvitationService` | `CreateInvitation`, `AcceptInvitation`, `RevokeAccess` | 共享与撤销 |

### 使用示例

```go
// 创建存储实现
storage := encryption.NewUserlibStorage()    // 或 NewMemoryStorage() 用于测试
keyStore := encryption.NewUserlibKeyStore()  // 或 NewMemoryKeyStore() 用于测试

// 创建服务
userService := encryption.NewUserService(storage, keyStore)
fileService := encryption.NewFileService(storage, keyStore)
invitationService := encryption.NewInvitationService(storage, keyStore)

// 初始化用户
user, err := userService.InitUser("alice", "password")

// 存储文件
err = fileService.StoreFile(user, "hello.txt", []byte("Hello, World!"))

// 加载文件
content, err := fileService.LoadFile(user, "hello.txt")

// 共享文件
invPtr, err := invitationService.CreateInvitation(user, "hello.txt", "bob")

// 接受邀请
err = invitationService.AcceptInvitation(bobUser, "alice", invPtr, "hello.txt")

// 撤销访问
err = invitationService.RevokeAccess(user, "hello.txt", "bob")
```
