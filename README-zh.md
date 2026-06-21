# ShareLock

[**English Version**](./README.md)

一个密码学安全的、去中心化信任的文件存储与共享系统，设计用于在不安全的云基础设施上安全运行。

**ShareLock** 将 UC Berkeley CS161 安全框架的核心架构原则扩展为端到端加密（E2EE）文件共享应用。即使服务器端被完全攻破，也能保证所有用户数据的机密性、完整性和真实性。

---

## 威胁模型与安全保证

系统针对**恶意主动攻击者**设计，该攻击者完全控制存储服务器和网络流量。

- **机密性：** 未授权用户（包括存储提供商）对文件内容、文件名或共享图一无所知。
- **完整性与真实性：** 任何未经授权的修改或篡改都会被立即检测到。
- **撤销效率：** 被撤销用户立即失去对文件所有未来更新的访问权。

---

## 关键特性

- **端到端加密（E2EE）：** 所有加密/解密严格在客户端执行。密钥永不以明文形式离开本地设备。
- **细粒度访问控制：** 通过加密邀请指针与特定用户共享文件。
- **即时撤销：** 动态密钥轮换隔离被撤销用户，透明更新剩余共享者。
- **追加优化：** 向大文件追加内容时无需重新加密整个结构。
- **先加密后 MAC：** 所有密文在存储前均通过 HMAC-SHA512 认证。
- **TLS 加密流式传输：** `read` 命令通过 TLS 加密的 TCP 连接下载文件。

---

## 快速开始

### 环境要求

- Go 1.20+
- 支持的操作系统：Linux, macOS, Windows

### 快速开始

```bash
git clone git@github.com:tuxnode/ShareLock.git
cd ShareLock
make all          # 构建二进制 + 生成开发 TLS 证书
```

### 快速演示

```bash
# 启动服务端
./sharelock-server -tls=false -address :8080 -dir ./data &

# 创建用户
./sharelock init -u alice -p pass123
./sharelock init -u bob   -p pass456

# 存储文件
echo "Hello from Alice!" > hello.txt
./sharelock store -f hello.txt

# 加载文件
./sharelock load -f hello.txt

# 分享给 Bob
invite=$(./sharelock share -f hello.txt -r bob)

# Bob 接受邀请
./sharelock accept -s alice -i "$invite" -f shared.txt
./sharelock load -f shared.txt

# Alice 撤销 Bob 的访问权限
./sharelock revoke -f hello.txt -r bob
```

---

## CLI 使用

### 命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `init` | 创建新用户 | `sharelock init -u alice -p secret` |
| `login` | 登录已有用户 | `sharelock login -u alice -p secret` |
| `store` | 从磁盘存储文件 | `sharelock store -f hello.txt` |
| `load` | 加载并打印文件内容 | `sharelock load -f hello.txt` |
| `append` | 追加内容到文件 | `sharelock append -f hello.txt -c " more"` |
| `share` | 分享文件给其他用户 | `sharelock share -f hello.txt -r bob` |
| `accept` | 接受文件分享邀请 | `sharelock accept -s alice -i <uuid> -f file.txt` |
| `revoke` | 撤销用户访问权限 | `sharelock revoke -f hello.txt -r bob` |
| `read` | 通过 TLS 流读取文件 | `sharelock read -f hello.txt -a localhost:8080` |
| `host` | 管理服务器连接 | `sharelock host add dev localhost:8080` |
| `help` | 显示帮助 | `sharelock help` |

运行 `sharelock <command> --help` 查看各命令的详细参数。

### 主机管理

```bash
./sharelock host add default localhost:8080
./sharelock host add dev localhost:8080 --tls=false
SHARELOCK_HOST=dev ./sharelock store -f ./hello.txt
./sharelock host list
```

---

## 服务端

KV 存储服务端（`sharelock-server`）使用 BadgerDB 提供加密的键值存储服务。所有加密操作在客户端执行 — 服务端为零知识。

### 启动参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-address` | `localhost:8080` | 监听地址 |
| `-dir` | `./data` | BadgerDB 数据目录 |
| `-cert` | — | TLS 证书文件（`-tls=true` 时必填） |
| `-key` | — | TLS 私钥文件（`-tls=true` 时必填） |
| `-tls` | `true` | 启用 TLS 加密 |

### 示例

```bash
# TLS 模式（生产环境）
./sharelock-server -address :8080 -dir ./data -cert cert.pem -key key.pem

# 明文 TCP 模式（开发环境）
./sharelock-server -tls=false -address :8080 -dir ./data
```

---

## 测试

```bash
make test              # 运行所有测试
make test-app          # 应用客户端测试
make test-encryption   # 加密集成测试
make test-unit         # 加密单元测试
make test-handler      # 处理协议测试
make test-store        # KV 存储测试
make test-integration  # 服务端 TLS 集成测试
make test-userlib      # userlib 测试
```

详见[测试说明文档](./docs/testing-zh.md)。

---

## 项目结构

```
.
├── cmd/
│   ├── client/main.go           # CLI 入口
│   └── server/main.go           # KV 存储服务端
├── internal/
│   ├── client/
│   │   ├── encryption/          # 密码学：User, FileService, InvitationService
│   │   ├── app/app.go           # 应用业务逻辑
│   │   ├── netstream/           # TLS 加密文件流式传输
│   │   └── config/              # .hosts 文件管理
│   ├── server/
│   │   ├── server.go            # TLS 监听，每个连接一个 goroutine
│   │   ├── store/store.go       # BadgerDB KV 存储
│   │   └── handler/handler.go   # 二进制协议处理
│   └── userlib/                 # 密码学原语 + 网络存储
├── docs/
│   ├── architecture-zh.md       # 密码学设计详情
│   ├── api-zh.md                # 用户库 & Service API 参考
│   └── testing-zh.md            # 测试说明
├── Makefile
├── go.mod
└── LICENSE
```

---

## 文档

- [架构与密码学设计](./docs/architecture-zh.md)
- [API 参考](./docs/api-zh.md)
- [测试说明](./docs/testing-zh.md)
- [English Version](./README.md)

---

## 许可证

本项目基于 UC Berkeley CS161（计算机安全）项目 2 的入门代码。保留所有权利。
