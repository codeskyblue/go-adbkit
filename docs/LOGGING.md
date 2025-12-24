# 日志级别说明

## 日志类型

USB-to-TCP Bridge 使用结构化日志 (slog)，提供两种日志级别：

### Info 级别（默认）
显示关键操作信息，适合正常使用：
- 服务启动/停止
- 设备连接信息
- 服务调用（shell 命令等）
- 错误信息

```bash
./usb-bridge -serial <device-serial>
```

示例输出：
```
time=2025-12-10T18:52:27.000+08:00 level=INFO msg="Starting USB-to-TCP bridge" device=192.168.0.95:4575 port=6174
time=2025-12-10T18:52:27.000+08:00 level=INFO msg="TCP-USB bridge listening" address=:6174
time=2025-12-10T18:52:27.000+08:00 level=INFO msg="Connect with" command="adb connect :6174"
time=2025-12-10T18:52:27.000+08:00 level=INFO msg="Calling service" name="shell:pwd"
```

### Debug 级别（详细模式）
显示所有 ADB 协议细节，适合调试和开发：
- 所有 Info 级别的日志
- ADB 协议包详情（A_SYNC, A_CNXN, A_OPEN, A_OKAY, A_WRTE, A_CLSE, A_AUTH）
- 认证过程详情
- 数据传输详情
- 服务生命周期

```bash
./usb-bridge -serial <device-serial> --verbose
```

示例输出：
```
time=2025-12-10T18:52:27.000+08:00 level=INFO msg="Starting USB-to-TCP bridge" device=192.168.0.95:4575 port=6174
time=2025-12-10T18:52:27.000+08:00 level=INFO msg="TCP-USB bridge listening" address=:6174
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="I:A_CNXN" packet="CNXN arg0=16777217 arg1=65535 length=218"
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="Created challenge" token="WiV2LDJYzFaX09hiyzxivTGhWyc="
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="O:A_AUTH"
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="I:A_AUTH" packet="AUTH arg0=2 arg1=0 length=256"
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="Received signature" signature="Nte8rF1eWtrN..."
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="O:A_AUTH"
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="I:A_AUTH" packet="AUTH arg0=3 arg1=0 length=712"
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="Received RSA public key"
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="O:A_CNXN"
time=2025-12-10T18:52:27.000+08:00 level=INFO msg="Calling service" name="shell:pwd"
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="Services active" count=1
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="I:A_OPEN" packet="OPEN arg0=3012628 arg1=0 length=10"
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="O:A_OKAY"
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="O:A_WRTE"
time=2025-12-10T18:52:27.000+08:00 level=DEBUG msg="I:A_OKAY" packet="OKAY arg0=3012628 arg1=2 length=0"
```

## ADB 协议包说明

当使用 `--verbose` 模式时，你会看到这些协议包：

- **A_SYNC**: 同步包，用于握手
- **A_CNXN**: 连接包，建立 ADB 连接
- **A_AUTH**: 认证包，设备认证过程
- **A_OPEN**: 打开服务包，开始一个新的服务（如 shell 命令）
- **A_OKAY**: 确认包，确认收到数据
- **A_WRTE**: 写数据包，传输实际数据
- **A_CLSE**: 关闭包，关闭服务连接

包格式示例：`WRTE arg0=3012628 arg1=2 length=10`
- arg0/arg1: 协议参数（通常是连接 ID）
- length: 数据长度

## 使用场景

### 正常使用
```bash
./usb-bridge -serial <device-serial>
```
适用于日常使用，只显示必要信息。

### 调试问题
```bash
./usb-bridge -serial <device-serial> --verbose
```
当遇到以下情况时使用：
- 连接问题
- 命令执行失败
- 数据传输异常
- 性能分析
- 协议开发和测试

### 日志重定向
```bash
# 只保存错误日志
./usb-bridge -serial <device-serial> 2>error.log

# 保存所有详细日志
./usb-bridge -serial <device-serial> --verbose 2>debug.log

# 同时显示和保存
./usb-bridge -serial <device-serial> --verbose 2>&1 | tee bridge.log
```

## 程序化使用

如果你在代码中使用 tcpusb 包，可以通过 slog 配置日志级别：

```go
import "log/slog"

// 设置为 Debug 级别
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
slog.SetDefault(logger)

// 设置为 Info 级别（默认）
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
slog.SetDefault(logger)

// 完全静默（只显示错误和警告）
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelWarn,
}))
slog.SetDefault(logger)
```
