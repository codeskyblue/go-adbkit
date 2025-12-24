# USB-to-TCP Bridge 使用指南

## 快速开始

### 1. 作为命令行工具使用

```bash
# 构建命令行工具
cd cmd/usb-bridge
go build

# 查看设备列表
adb devices

# 启动 bridge (将 <serial> 替换为你的设备序列号)
./usb-bridge -serial <device-serial> -port 6174

# 启用详细日志 (显示协议调试信息)
./usb-bridge -serial <device-serial> -port 6174 --verbose

# 从其他机器连接
adb connect <your-ip>:6174
```

#### 命令行参数

- `-serial <serial>`: 设备序列号（必需）
- `-port <port>`: 监听端口（默认: 6174）
- `-adb-host <host>`: ADB 服务器地址（默认: 127.0.0.1）
- `-adb-port <port>`: ADB 服务器端口（默认: 5037）
- `--verbose`: 启用详细日志，显示 ADB 协议调试信息

### 2. 作为 Go 库使用

#### 简单示例

```go
package main

import (
    "log"
    "github.com/codeskyblue/go-adbkit/tcpusb"
)

func main() {
    // 创建一个简单的 bridge
    bridge := tcpusb.NewBridge("your-device-serial")
    
    // 启动 bridge (阻塞调用)
    if err := bridge.Start(); err != nil {
        log.Fatal(err)
    }
}
```

#### 自定义配置

```go
package main

import (
    "log"
    "github.com/codeskyblue/go-adbkit/tcpusb"
)

func main() {
    // 创建 bridge
    bridge := tcpusb.NewBridge("device-serial")
    
    // 配置参数
    bridge.Config.Port = 7000                // 自定义端口
    bridge.Config.ADBHost = "192.168.1.100" // 远程 ADB 服务器
    bridge.Config.ADBPort = 5037             // ADB 端口
    
    // 非阻塞启动,获取 server 实例进行控制
    server, err := bridge.StartWithServer()
    if err != nil {
        log.Fatal(err)
    }
    defer server.Close()
    
    // 你的其他代码...
    
    log.Println("Bridge is running...")
    select {} // 保持运行
}
```

#### 自定义认证

```go
package main

import (
    "errors"
    "log"
    "github.com/codeskyblue/go-adbkit/tcpusb"
)

func main() {
    // 自定义认证处理器
    authHandler := func(publicKey []byte) error {
        // 实现你的认证逻辑
        log.Printf("Device attempting to connect with key: %x", publicKey[:20])
        
        // 例如:检查公钥是否在白名单中
        if isAuthorized(publicKey) {
            return nil  // 接受连接
        }
        return errors.New("unauthorized device")  // 拒绝连接
    }
    
    bridge := tcpusb.NewBridge("device-serial")
    bridge.Config.AuthHandler = authHandler
    
    bridge.Start()
}

func isAuthorized(publicKey []byte) bool {
    // 你的验证逻辑
    return true
}
```

#### 多设备 Bridge

```go
package main

import (
    "log"
    "github.com/codeskyblue/go-adbkit/tcpusb"
)

func main() {
    devices := []string{"device1-serial", "device2-serial", "device3-serial"}
    
    // 为每个设备创建一个 bridge
    for i, serial := range devices {
        bridge := tcpusb.NewBridge(serial)
        bridge.Config.Port = 6174 + i  // 为不同设备分配不同端口
        
        go func(b *tcpusb.Bridge, s string) {
            log.Printf("Starting bridge for device %s on port %d", s, b.Config.Port)
            if err := b.Start(); err != nil {
                log.Printf("Bridge error for %s: %v", s, err)
            }
        }(bridge, serial)
        
        port++
    }
    
    // 保持主进程运行
    select {}
}
```

## API 文档

### Bridge 选项

| 函数 | 说明 | 默认值 |
|------|------|--------|
| `WithPort(port int)` | 设置 TCP 监听端口 | 6174 |
| `WithADBHost(host string)` | 设置 ADB 服务器地址 | "127.0.0.1" |
| `WithADBPort(port int)` | 设置 ADB 服务器端口 | 5037 |
| `WithAuthHandler(handler AuthHandler)` | 设置自定义认证处理器 | 总是接受 |

### Bridge 方法

#### `NewBridge(serial string, options ...BridgeOption) *Bridge`
创建一个新的 bridge 实例。

**参数:**
- `serial`: 设备序列号
- `options`: 可选配置项

**返回:**
- `*Bridge`: Bridge 实例

#### `Start() error`
启动 bridge 服务器(阻塞调用)。

**返回:**
- `error`: 错误信息

#### `StartWithServer() (*Server, error)`
启动 bridge 并返回 server 实例(非阻塞)。

**返回:**
- `*Server`: Server 实例,可用于控制
- `error`: 错误信息

### Server 方法

#### `Close() error`
关闭服务器和所有连接。

**返回:**
- `error`: 错误信息

#### `Addr() net.Addr`
返回监听地址。

**返回:**
- `net.Addr`: 监听地址

## 使用场景

### 1. 远程开发
通过网络访问 USB 连接的 Android 设备进行远程开发和测试。

```bash
# 在有设备的机器上
./usb-bridge -serial device-serial -port 6174

# 在开发机器上
adb connect <server-ip>:6174
adb shell
```

### 2. CI/CD 集成
将物理设备集成到 CI/CD 流程中。

```go
// 在 CI 服务器上
server, err := bridge.StartWithServer()
if err != nil {
    log.Fatal(err)
}

// 运行测试
runTests()

// 清理
server.Close()
```

### 3. 设备农场
创建设备农场,在不同端口暴露多个设备。

```go
devices := []string{"device1", "device2", "device3"}
port := 6174

for _, serial := range devices {
    bridge := tcpusb.NewBridge(serial, 
        tcpusb.WithPort(port),
    )
    go bridge.Start()
    port++
}
```

## 故障排除

### Bridge 无法启动
- 确保 ADB 服务器正在运行: `adb start-server`
- 检查端口是否被占用: `lsof -i :<port>`

### 无法从远程机器连接
- 确保防火墙允许 bridge 端口的连接
- 验证设备序列号是否正确: `adb devices`

### 设备未授权
- 接受 Android 设备上的授权提示
- 或实现自定义认证处理器自动接受

## 架构说明

实现遵循 ADB 协议规范:

```
TCP 客户端 (adb) <-> Socket <-> Service <-> ADB 服务器 <-> USB 设备
```

1. **Packet Layer**: 处理 ADB 协议数据包 (SYNC, CNXN, OPEN, OKAY, WRTE, CLSE, AUTH)
2. **Socket Layer**: 管理客户端连接和认证
3. **Service Layer**: 在 TCP 客户端和 USB 设备之间转发数据
4. **Server Layer**: 接受 TCP 连接并创建 socket 处理器

## 性能提示

1. **调整缓冲区大小**: maxPayload 默认为 4096 字节,可以根据需要调整
2. **并发连接**: 服务器可以处理多个并发连接
3. **网络延迟**: 在高延迟网络上使用时可能会影响性能

## 安全建议

1. **使用自定义认证**: 在生产环境中实现适当的认证
2. **防火墙规则**: 限制对 bridge 端口的访问
3. **加密通信**: 考虑使用 VPN 或 SSH 隧道加密通信

## 示例代码

更多示例请参考 `examples/` 目录。
