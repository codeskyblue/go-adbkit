# How to Write Tests

1. Capture data through socat:

```sh
socat -v -x TCP-LISTEN:5038,fork TCP-CONNECT:localhost:5037
```

2. Run command with adb:

```sh
adb -P 5038 devices
```

3. Copy the output to the test file. See [host_commands_test.go](adb/host_commands_test.go) for reference.