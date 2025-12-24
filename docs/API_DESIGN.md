# API 设计文档

## 概述

`go-adbkit` 提供了一个简洁而符合 Go 语言习惯的 API，用于创建和管理 USB-to-TCP 桥接。

## 设计哲学

API 设计遵循以下原则：

1. **简洁性**: 最常见的用法应该最简单
2. **显式优于隐式**: 配置参数明确可见，易于理解
3. **Go 习惯**: 遵循标准 Go 库的设计模式
4. **灵活性**: 支持简单和复杂的配置场景

## 核心数据结构

### Config 结构体

```go
type Config struct {
    Port        int          // TCP 端口 (默认: 6174)
    ADBHost     string       // ADB 服务器地址 (默认: 127.0.0.1)
    ADBPort     int          // ADB 服务器端口 (默认: 5037)
    AuthHandler AuthHandler  // 认证处理器函数
}
```

### Bridge 结构体

```go
type Bridge struct {
    Serial string // 设备序列号
    Config Config // 配置信息
}
```

## API 使用模式

### 1. 默认配置（最简单的情况）

```go
bridge := tcpusb.NewBridge("device-serial")
if err := bridge.Start(); err != nil {
    log.Fatal(err)
}
```

**说明**: 使用默认配置启动桥接，最小化代码。

### 2. 自定义单个配置项

```go
bridge := tcpusb.NewBridge("device-serial")
bridge.Config.Port = 7000
bridge.Config.ADBHost = "192.168.1.100"

if err := bridge.Start(); err != nil {
    log.Fatal(err)
}
```

**说明**: 显式修改所需的配置项，其他项保持默认值。

### 3. 完全自定义配置

```go
bridge := tcpusb.NewBridge("device-serial")
bridge.Config = tcpusb.Config{
    Port:    7000,
    ADBHost: "192.168.1.100",
    ADBPort: 5037,
    AuthHandler: func(publicKey []byte) error {
        // 自定义认证逻辑
        return nil
    },
}

if err := bridge.Start(); err != nil {
    log.Fatal(err)
}
```

**说明**: 一次性设置所有配置参数。

### 4. 非阻塞启动

```go
bridge := tcpusb.NewBridge("device-serial")
server, err := bridge.StartWithServer()
if err != nil {
    log.Fatal(err)
}
defer server.Close()

// 在这里进行其他操作
// server 继续在后台处理连接
```

**说明**: 获得 server 实例的控制权，可以在适当的时机关闭。

## DefaultConfig 函数

```go
func DefaultConfig() Config {
    return Config{
        Port:    6174,
        ADBHost: "127.0.0.1",
        ADBPort: 5037,
        AuthHandler: func(publicKey []byte) error {
            return nil  // 接受所有连接
        },
    }
}
```

用于获取默认配置，也可用于重置配置：

```go
bridge.Config = tcpusb.DefaultConfig()
```

## 与函数选项模式的对比

### 之前的设计（函数选项模式）

```go
bridge := tcpusb.NewBridge("device-serial",
    tcpusb.WithPort(7000),
    tcpusb.WithADBHost("192.168.1.100"),
)
```

**优点**:
- 支持可变参数
- 易于链式调用

**缺点**:
- 需要定义多个函数
- 参数不够直观
- `With` 前缀不够 Go 风格

### 现有设计（配置结构体）

```go
bridge := tcpusb.NewBridge("device-serial")
bridge.Config.Port = 7000
bridge.Config.ADBHost = "192.168.1.100"
```

**优点**:
- 更加简洁直观
- 符合 Go 语言习惯
- 配置参数清晰可见
- 易于理解和维护

**缺点**:
- 无法在创建时立即指定所有参数（但通常这不是问题）

## 最佳实践

### 1. 单设备桥接

```go
bridge := tcpusb.NewBridge("device-serial")
if err := bridge.Start(); err != nil {
    log.Fatal(err)
}
```

### 2. 多设备桥接

```go
devices := []string{"device1", "device2", "device3"}

for i, serial := range devices {
    bridge := tcpusb.NewBridge(serial)
    bridge.Config.Port = 6174 + i  // 为每个设备分配不同端口
    
    go func(b *tcpusb.Bridge) {
        if err := b.Start(); err != nil {
            log.Printf("Error for %s: %v", b.Serial, err)
        }
    }(bridge)
}

select {} // 保持运行
```

### 3. 远程 ADB 服务器

```go
bridge := tcpusb.NewBridge("device-serial")
bridge.Config.ADBHost = "remote-adb-server.example.com"
bridge.Config.ADBPort = 5037

if err := bridge.Start(); err != nil {
    log.Fatal(err)
}
```

### 4. 自定义认证

```go
bridge := tcpusb.NewBridge("device-serial")
bridge.Config.AuthHandler = func(publicKey []byte) error {
    // 实现认证逻辑
    if isAuthorized(publicKey) {
        return nil
    }
    return errors.New("unauthorized")
}

if err := bridge.Start(); err != nil {
    log.Fatal(err)
}
```

## 总结

当前 API 设计采用了**配置结构体**的方法，这是一种更符合 Go 语言习惯的设计方式。它在简洁性和灵活性之间取得了很好的平衡，使得代码易于理解和使用。
