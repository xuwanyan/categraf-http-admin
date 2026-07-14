# categraf-http-admin

Categraf HTTP 拨测（http_response）远程配置管理工具。

通过 Web 页面可视化增删拨测目标，categraf 通过 `http_provider` 自动拉取配置，无需登录服务器改文件。

## 架构

```
用户浏览器 ──Web页面──→ categraf-http-admin
                            │
categraf ──http_provider──→ /api/config/http_response
```

## 部署

### 1. 上传文件

```bash
scp categraf-http-admin root@服务器:/etc/categraf/
scp categraf-http-admin.service root@服务器:/etc/systemd/system/
```

### 2. 创建认证配置

```bash
cat > /etc/categraf/.env << 'EOF'
CONFIG_USER=admin
CONFIG_PASS=你的密码
EOF
chmod 600 /etc/categraf/.env
```

### 3. 启动服务

```bash
systemctl daemon-reload
systemctl enable --now categraf-http-admin
```

### 4. 配置 categraf

在 `/etc/categraf/conf/config.toml` 中修改：

```toml
providers = ["local", "http"]

[http_provider]
remote_url = "http://你的服务器IP:5000/api/config/http_response"
timeout = 5
reload_interval = 60
```

然后重启 categraf：

```bash
systemctl restart categraf
```

## 使用

访问 `http://你的服务器IP:5000`，用 `.env` 中设置的用户名密码登录。

### 页面功能

- **添加目标**：填写 URL、网站名称、请求方法、状态码等
- **编辑目标**：点击列表中的"编辑"按钮
- **删除目标**：点击"删除"按钮
- **全局超时**：每个目标独立设置超时时长
- **TLS 配置**：填 `https://` 开头的 URL 时自动显示
- **TOML 预览**：页面底部实时显示生成的配置

### API

| 方法 | 路径 | 说明 | 需认证 |
|------|------|------|--------|
| GET | `/api/targets` | 查看所有目标 | 否 |
| POST | `/api/targets` | 添加目标（JSON 或 Form） | 是 |
| DELETE | `/api/targets/{id}` | 删除目标 | 否 |
| POST | `/api/targets/{id}/edit` | 编辑目标 | 是 |
| POST | `/api/targets/{id}/delete` | 删除目标（表单） | 是 |
| GET | `/api/config/http_response` | categraf 配置拉取端点 | 否 |
| GET | `/health` | 健康检查 | 否 |

### 字段说明

| 字段 | 说明 |
|------|------|
| URL | 拨测地址 |
| 请求方法 | GET / POST / PUT / DELETE / HEAD |
| 网站名称 | 用于 [mappings] 中的 job 标签 |
| 期望状态码 | 默认 200，支持 `200\|301` 多值 |
| 超时时长 | 空则不写，categraf 用默认 3s |
| 跟随重定向 | 勾选后跟随 301/302 跳转 |
| 请求头 / Body | 仅在 POST 时显示 |
| 使用 TLS | 填 https:// 时自动显示 |
| TLS CA 证书路径 | 勾选"使用 TLS"后显示 |

## TOML 合并规则

相同配置（方法 + 状态码 + 超时 + Body + 请求头 + 重定向 + TLS）的目标自动合并到同一个 `[[instances]]`：

```toml
[mappings]
"https://site-a.com" = { job = "站点A" }
"https://site-b.com" = { job = "站点B" }

[[instances]]
targets = [
    "https://site-a.com",
    "https://site-b.com"
]

[[instances]]
targets = ["https://api.example.com"]
method = "POST"
response_timeout = "15s"
body = """..."""
```

## 常见操作

### 查看运行状态

```bash
systemctl status categraf-http-admin
journalctl -u categraf-http-admin -f
```

### 修改密码

编辑 `.env` 文件后重启：

```bash
vim /etc/categraf/.env
systemctl restart categraf-http-admin
```

### 不启用页面认证

不创建 `.env` 文件即可，所有页面公开访问（不推荐生产环境使用）。

## 编译

需要 Go 1.22+：

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o categraf-http-admin .

# 本地 Windows
go build -o categraf-http-admin.exe .
```