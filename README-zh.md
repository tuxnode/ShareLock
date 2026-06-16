# ShareLock

[**English Version**](./README.md)

一个密码学安全的、去中心化信任的文件存储与共享系统，设计用于在不安全的云基础设施上安全运行。

**ShareLock** 将 UC Berkeley CS161 安全框架的核心架构原则扩展为端到端加密（E2EE）文件共享应用。即使服务器端被完全攻破，也能保证所有用户数据的机密性、完整性和真实性。

---

## 威胁模型与安全保证

系统针对**恶意主动攻击者**设计，该攻击者完全控制存储服务器（Datastore）和网络流量。

### 安全目标

- **机密性：** 未授权用户（包括存储提供商）对文件内容、文件长度、文件名或共享图拓扑结构一无所知。
- **完整性与真实性：** 任何对文件数据或共享元数据的未经授权修改、篡改或回滚，都会被立即检测到。
- **撤销效率：** 当文件所有者撤销某用户的访问权限时，该用户立即失去对文件所有未来更新的访问权，其密码学访问路径被完全无效化。

---

## 架构与密码学设计

应用实现了一个分层的密码学流水线，以确保零知识存储和安全访问委托。

### 1. 密码学原语

| 原语 | 算法 | 用途 |
|-----------|-----------|-------|
| 对称加密 | AES-CTR (128-bit) | 文件块、文件元数据和用户结构的认证加密 |
| 消息认证 | HMAC-SHA512 | 完整性验证（先加密后 MAC） |
| 公钥加密 | RSA-OAEP (2048-bit) with SHA-512 | 共享邀请的安全密钥交换 |
| 数字签名 | RSA-PKCS1.5 (2048-bit) with SHA-512 | 共享邀请的不可否认性和验证 |
| 密钥派生 | Argon2id | 从用户密码派生出主密钥 |
| 密钥多样化 | HashKDF (基于 HMAC) | 从单个主密钥派生出特定用途的子密钥（加密 vs MAC） |
| 哈希 | SHA-512 | 确定性 UUID 生成、文件名加盐 |

### 2. 密钥层次

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

### 3. 数据结构

- **文件分块：** 文件被分割为 512 字节的 `FileBlock` 块，每块使用从随机 `FileKey` 派生的文件特定密钥独立加密。
- **Inode：** 跟踪文件总大小和有序的块 UUID 列表。作为单个数据块在文件密钥下加密和 MAC 保护。
- **MailboxNode：** 每个用户（所有者或共享者）的指针，包含 `FileKey` 和 `InodeUUID`，使用邮箱特定密钥加密。每个用户拥有自己的 MailboxNode。
- **访问记录（Access）：** 将文件名映射到所有者的 MailboxNode UUID/密钥，并维护一个共享树（`Chidren` 映射），记录所有直接共享者，用于撤销操作。
- **邀请（Invitation）：** 包含 `MailboxUUID` 和 `MailboxKey` 的加密载荷，通过 RSA-OAEP + 数字签名传输以授予访问权限。
- **用户结构（User Struct）：** 包含用户名、RSA 私钥、DS 签名密钥、Argon2 派生主密钥以及已知文件访问指针的映射。在用户派生密钥下加密后存储在 Datastore 中。

### 4. 密码学流程

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
  → 更新发送者的 Access.Chidren

AcceptInvitation (接受邀请):
  → 验证签名 + 解密邀请 (RSA-OAEP)
  → 创建指向收到的 MailboxNode 的本地 Access

RevokeAccess (撤销访问):
  → 生成新的 FileKey
  → 用新密钥重新加密所有块和 inode
  → 创建所有者的新 MailboxNode
  → 用新 FileKey 更新剩余子节点的 MailboxNode
  → 从 Chidren 中移除被撤销的用户
```

---

## 关键特性

- **端到端加密（E2EE）：** 所有加密/解密严格在客户端执行。密钥永不以明文形式离开本地设备。
- **细粒度访问控制：** 通过基于 MailboxNode 共享树的加密邀请指针，无缝与特定用户共享文件。
- **即时撤销：** 动态密钥轮换机制在撤销时更换文件密钥，隔离被撤销用户，并透明更新剩余共享者而不中断其访问。
- **追加优化：** 向现有大文件追加内容时，无需下载或重新加密整个文件结构。
- **先加密后 MAC：** 所有密文在存储前均通过 HMAC-SHA512 认证，保证完整性。
- **TLS 加密流式传输：** `read` 命令通过 `netstream` 模块经 TLS 加密的 TCP 连接下载文件，保证传输中的机密性。

---

## 实现状态

| 组件 | 状态 |
|-----------|--------|
| `InitUser` | ✅ 已完成 |
| `GetUser` | ✅ 已完成 |
| `StoreFile` | ✅ 已完成 |
| `LoadFile` | ✅ 已完成 |
| `AppendToFile` | ✅ 已完成 |
| `CreateInvitation` | ✅ 已完成 |
| `AcceptInvitation` | ✅ 已完成 |
| `RevokeAccess` | ✅ 已完成 |
| `ReadFile` (TLS 流式传输) | ✅ 已完成 |
| 命令行 (`cmd/client`) | ✅ 已完成 |
| Netstream (`internal/netstream`) | ✅ 已完成 |
| KV 存储服务端 (`cmd/server`) | ✅ 已完成 |
| BadgerDB 存储 (`internal/server/store`) | ✅ 已完成 |
| 二进制协议处理 (`internal/server/handler`) | ✅ 已完成 |

---

## 快速开始

### 环境要求

- Go 1.20+
- 支持的操作系统：Linux, macOS, Windows

### 快速开始

```bash
# 克隆仓库
git clone git@github.com:tuxnode/ShareLock.git
cd ShareLock

# 构建所有内容并生成开发 TLS 证书
make all

# 运行所有测试
make test

# 查看可用命令
make help
```

### 运行测试

项目使用 [Ginkgo v2](https://onsi.github.io/ginkgo/) 和 [Gomega](https://onsi.github.io/gomega/) 进行测试。详细文档请参阅[测试说明文档](./docs/testing-zh.md)，包含[基准测试](./docs/testing-zh.md#性能基准测试)。

```bash
# 运行所有测试
make test

# 运行特定测试套件
go test -v -run "TestSetupAndExecution" ./...
```

### CLI 使用

```bash
# 构建二进制
make build

# 初始化用户
./sharelock inituser -username alice -password secret

# 存储文件
./sharelock storefile -filename hello.txt -content "Hello, World!"

# 加载文件
./sharelock loadfile -filename hello.txt

# 共享文件
invite=$(./sharelock createinvitation -filename hello.txt -recipient bob)
./sharelock acceptinvitation -sender alice -invitation $invite -filename hello.txt

# 撤销访问
./sharelock revokeaccess -filename hello.txt -recipient bob

# 通过 TLS 加密流读取文件
./sharelock read -filename hello.txt -address localhost:8080

# 运行 KV 存储服务端（make all 已生成证书）
./sharelock-server -address :8080 -dir ./data -cert cert.pem -key key.pem

# 明文 TCP 模式（开发用，无需证书）
./sharelock-server -tls=false -address :8080 -dir ./data
./sharelock read -filename hello.txt -address localhost:8080 -tls=false
```

### 代码检查

```bash
make vet
```

---

## 项目结构

```
.
├── cmd/
│   ├── client/
│   │   └── main.go              # CLI 入口（子命令分发）
│   └── server/
│       └── main.go              # KV 存储服务端 (TLS + BadgerDB)
├── internal/
│   ├── client/
│   │   ├── encryption/
│   │   │   ├── access.go        # 数据结构：MailboxNode, Access, Invitation, ChildrenInfo
│   │   │   ├── encryption.go    # 核心客户端：User 结构、InitUser、GetUser、StoreFile 等
│   │   │   ├── encryption_unittest.go  # 白盒单元测试 (Ginkgo/Gomega)
│   │   │   ├── File.go          # 文件块分割/合并工具
│   │   │   └── utils.go         # 密码学辅助函数：encryptAndMAC、decryptAndVerify、密钥派生
│   │   ├── app/
│   │   │   └── app.go           # 应用层客户端业务逻辑
│   │   └── netstream/
│   │       └── netstream.go     # TLS 加密文件流式传输 (FileSeander / FileReceiver)
│   ├── client/encryption_test/
│   │   └── encryption_test.go   # 黑盒集成测试
│   ├── client/app_test/
│   │   └── app_test.go          # 应用客户端集成测试
│   ├── netstream/
│   │   └── netstream.go         # TLS 加密文件流式传输 (FileSeander / FileReceiver)
│   ├── server/
│   │   ├── server.go            # TLS 监听循环，每个连接一个 goroutine
│   │   ├── store/
│   │   │   └── store.go         # BadgerDB KV 存储 (Get, Set, Delete, Exists)
│   │   └── handler/
│   │       └── handler.go       # 二进制协议处理 (GET 0x01 / SET 0x02 / DELETE 0x03)
│   └── integration_test/
│       └── server_test.go       # 服务端 TLS 集成测试
├── internal/
│   ├── client/
│   │   ├── encryption/
│   │   │   ├── access.go        # 数据结构：MailboxNode, Access, Invitation, ChildrenInfo
│   │   │   ├── encryption.go    # 核心客户端：User 结构、InitUser、GetUser、StoreFile 等
│   │   │   ├── encryption_unittest.go  # 白盒单元测试 (Ginkgo/Gomega)
│   │   │   ├── File.go          # 文件块分割/合并工具
│   │   │   └── utils.go         # 密码学辅助函数：encryptAndMAC、decryptAndVerify、密钥派生
│   │   ├── app/
│   │   │   └── app.go           # 应用层客户端业务逻辑
│   │   └── netstream/
│   │       └── netstream.go     # TLS 加密文件流式传输 (FileSeander / FileReceiver)
│   ├── client/encryption_test/
│   │   └── encryption_test.go   # 黑盒集成测试
│   └── server/                  # (空目录, 占位)
├── internal/userlib/            # 密码学库 (Datastore, Keystore, 原语)
│   ├── userlib.go               # 核心密码学原语 + 网络 KV 存储后端
│   └── userlib_test.go          # 库测试
├── go.mod                       # 模块定义
├── go.sum                       # 依赖校验和
├── CHANGELOG.md
├── project2-spec.pdf            # 原始项目规范
├── proj2.excalidraw             # 架构图 (Excalidraw 格式)
├── LICENSE
├── README.md
└── README-zh.md
```

---

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

---

## 许可证

本项目基于 UC Berkeley CS161（计算机安全）项目 2 的入门代码。保留所有权利。
