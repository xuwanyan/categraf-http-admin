# Build

每次编译都必须产出两份二进制：

- Windows: `go build -o categraf-http-admin.exe .`
- Linux（静态编译）: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o categraf-http-admin .`

本机存在 `C:\vscode\go.work` 会干扰 `go build`，执行前先 `$env:GOWORK='off'`。
