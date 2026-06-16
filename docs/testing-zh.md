# 测试说明

## 概述

ShareLock 使用 [Ginkgo v2](https://onsi.github.io/ginkgo/) 和 [Gomega](https://onsi.github.io/gomega/) 进行单元测试和集成测试。测试套件覆盖加密层、应用层客户端以及 netstream 文件流式传输模块。

---

## 测试套件

| 套件 | 包 | 类型 | 位置 |
|------|-----|------|------|
| 加密单元测试 | `encryption` | 白盒单元测试 | `internal/client/encryption/encryption_unittest.go` |
| 加密集成测试 | `encryption_test` | 黑盒集成测试 | `internal/client/encryption_test/encryption_test.go` |
| 应用客户端测试 | `app_test` | 黑盒集成测试 | `internal/client/app_test/app_test.go` |
| Netstream 测试 | `netstream` | （暂无） | `internal/netstream/netstream.go` |
| KV 存储单元测试 | `store` | 单元测试 | `internal/server/store/store_test.go` |
| 处理协议测试 | `handler` | 单元测试 | `internal/server/handler/handler_test.go` |
| 服务端集成测试 | `integration_test` | 集成测试 (TLS) | `internal/integration_test/server_test.go` |

---

## 运行测试

```bash
# 运行所有测试
go test ./...

# 运行加密单元测试（白盒）
go test -v ./internal/client/encryption/...

# 运行加密集成测试（黑盒）
go test -v ./internal/client/encryption_test/...

# 运行应用客户端集成测试（黑盒）
go test -v ./internal/client/app_test/...

# 运行 KV 存储单元测试
go test -v ./internal/server/store/...

# 运行处理协议测试
go test -v ./internal/server/handler/...

# 运行服务端 TLS 集成测试
go test -v ./internal/integration_test/...

# 按测试套件名称运行
go test -v -run "TestApp" ./...

# 按描述过滤测试用例
go test -v ./internal/client/app_test/ --ginkgo.focus="RevokeAccess"
```

---

## 应用客户端测试（`internal/client/app_test/app_test.go`）

这是应用层 `app.Client` 的主要集成测试套件，覆盖所有公开方法，贯穿完整的加密与存储流水线。

### 测试分组

#### InitUser / GetUser（4 个用例）
- **单用户初始化与获取** — 验证基本生命周期
- **重复初始化** — 确保重复初始化已存在的用户返回错误
- **错误密码** — 验证 `GetUser` 拒绝错误凭据
- **不存在的用户** — 验证 `GetUser` 对未知用户返回错误

#### StoreFile / LoadFile（4 个用例）
- **存取文件** — 基本读写往返
- **空内容** — 零长度文件的边界情况
- **文件不存在** — 验证 `LoadFile` 优雅处理缺失文件
- **覆盖写入** — 验证重新存储文件会替换旧内容

#### AppendToFile（4 个用例）
- **追加到已存在文件** — 验证内容被正确追加
- **多次追加** — 验证累积追加产生预期的组合内容
- **文件不存在** — 验证 `AppendToFile` 对缺失文件返回错误
- **空内容** — 验证 nil 内容被拒绝

#### 邀请（4 个用例）
- **通过邀请共享文件** — 验证 `CreateInvitation` / `AcceptInvitation` 流程
- **共享者追加** — 验证共享者可以修改文件
- **对不存在的文件创建邀请** — 验证错误处理
- **发送者不存在** — 验证接受邀请时对未知发送者返回错误

#### 多会话一致性（3 个用例）
- **跨会话读取** — 验证一个会话写入的数据对另一个会话可见
- **跨会话追加** — 验证追加操作跨会话传播
- **跨会话邀请** — 验证一个会话创建的邀请在另一个会话中生效

#### 撤销访问（5 个用例）
- **撤销直接共享者** — 验证被撤销用户失去访问权限
- **所有者保留访问** — 验证撤销后所有者仍可读取
- **级联撤销间接共享者** — 验证撤销传播到子共享者
- **被撤销用户无法追加** — 验证写操作被阻止
- **所有者可继续追加** — 验证所有者的写能力不受影响

---

## 加密集成测试（`internal/client/encryption_test/encryption_test.go`）

这些测试直接操作原始 `encryption` 包，绕过 `app.Client` 包装层。它们验证与客户端测试相同的加密流程，但在更低的层级上。

### 测试分组

- **InitUser / GetUser** — 基本用户生命周期
- **单用户存储/加载/追加** — 单用户文件的完整增删改查
- **创建/接受邀请与多会话** — 跨多个客户端实例的共享
- **撤销功能** — 撤销操作及其对子共享者的级联效果

---

## 加密单元测试（`internal/client/encryption/encryption_unittest.go`）

白盒测试，可以访问内部结构体字段。该文件明确声明其内容**不会被评分**——仅供开发者在开发过程中验证内部实现细节。

---

## 编写新测试

### 在客户端套件中添加测试

```go
// 在 internal/client/app_test/app_test.go 中

It("应该完成某个特定功能", func() {
    alice := &app.Client{}
    err := alice.InitUser("alice", "password")
    Expect(err).To(BeNil())

    err = alice.StoreFile("test.txt", []byte("data"))
    Expect(err).To(BeNil())

    data, err := alice.LoadFile("test.txt")
    Expect(err).To(BeNil())
    Expect(data).To(Equal([]byte("data")))
})
```

### 添加新的测试文件

在相应的 `*_test` 目录或待测包的同级目录下创建新测试文件。遵循 Ginkgo/Gomega 惯例：

```go
package <包名>_test

import (
    "testing"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestSuite(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "套件描述")
}
```

---

## 代码质量

```bash
# 运行 Go vet 检查器
go vet ./...

# 列出所有测试而不运行
go test -list ".*" ./...
```
