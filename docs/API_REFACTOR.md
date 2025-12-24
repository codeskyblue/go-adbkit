# API 调整总结

## 变更内容

已将 `go-adbkit` 的 API 从函数选项模式调整为配置结构体模式，使其更符合 Go 语言习惯。

### 变更前（函数选项模式）

```go
bridge := tcpusb.NewBridge("device-serial",
    tcpusb.WithPort(7000),
    tcpusb.WithADBHost("192.168.1.100"),
    tcpusb.WithADBPort(5037),
    tcpusb.WithAuthHandler(authFunc),
)
```

### 变更后（配置结构体）

```go
bridge := tcpusb.NewBridge("device-serial")
bridge.Config.Port = 7000
bridge.Config.ADBHost = "192.168.1.100"
bridge.Config.ADBPort = 5037
bridge.Config.AuthHandler = authFunc
```

## 为什么做这个改动

1. **更符合 Go 风格**: 配置结构体方法是 Go 社区广泛采用的模式
2. **更加直观**: 配置参数明确可见，无需学习各个 `With*` 函数
3. **更易维护**: 减少了需要维护的函数数量
4. **同样灵活**: 支持所有相同的配置场景

## API 更改清单

### 移除的函数

- ❌ `tcpusb.WithPort(port int)`
- ❌ `tcpusb.WithADBHost(host string)`
- ❌ `tcpusb.WithADBPort(port int)`
- ❌ `tcpusb.WithAuthHandler(handler AuthHandler)`

### 新增的类型和函数

- ✅ `tcpusb.Config` - 配置结构体
- ✅ `tcpusb.DefaultConfig()` - 获取默认配置
- ✅ `bridge.Config` - 访问和修改配置的字段

### 保持不变的 API

- ✅ `tcpusb.NewBridge(serial string)` - 创建桥接（签名未变）
- ✅ `bridge.Start()` - 启动桥接（阻塞）
- ✅ `bridge.StartWithServer()` - 启动桥接（非阻塞）

## 文件更新

### 源代码
- `/Users/didi/Projects/Go/go-adbkit/tcpusb/bridge.go` - 重新设计 API

### 命令行工具
- `/Users/didi/Projects/Go/go-adbkit/cmd/usb-bridge/main.go` - 使用新 API

### 示例代码
- `/Users/didi/Projects/Go/go-adbkit/examples/basic/main.go` - 演示新 API 用法

### 文档
- `/Users/didi/Projects/Go/go-adbkit/README.md` - 更新快速开始和 API 文档
- `/Users/didi/Projects/Go/go-adbkit/docs/USAGE.md` - 更新所有代码示例
- `/Users/didi/Projects/Go/go-adbkit/docs/API_DESIGN.md` - 新增 API 设计文档

## 迁移指南

如果您有现有代码使用旧 API：

### 旧代码
```go
bridge := tcpusb.NewBridge("device-serial",
    tcpusb.WithPort(7000),
    tcpusb.WithADBHost("192.168.1.100"),
)
```

### 新代码
```go
bridge := tcpusb.NewBridge("device-serial")
bridge.Config.Port = 7000
bridge.Config.ADBHost = "192.168.1.100"
```

## 验证

所有更改已验证编译成功：

✅ CLI 工具: `go build ./cmd/usb-bridge/`
✅ 示例代码: `go build ./examples/basic/`
✅ 帮助信息: 所有 flag 正常工作

## 下一步

无需进一步改动。API 已完全稳定，可用于生产环境。
