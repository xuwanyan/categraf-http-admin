# categraf-http-admin

Categraf 拨测远程配置管理工具，支持两类拨测的在线管理：

- **网站拨测**（`http_response`）：HTTP/HTTPS 可用性、状态码、响应时间
- **端口拨测**（`net_response`）：TCP/UDP 端口连通性、send/expect 字符串交互校验

通过 Web 页面可视化增删改拨测目标，categraf 通过 `http_provider` 自动拉取配置，无需登录服务器改文件，categraf 侧零配置改动（两类插件共用同一个 provider URL）。

## 架构

```
用户浏览器 ──Web页面──→ categraf-http-admin ──持久化──→ targets.json
                            │
categraf ──http_provider──→ /api/config/http_response
        （一次拉取同时返回 http_response 与 net_response 两份 TOML）
```

- **配置版本**：version 是全部目标内容的 MD5 哈希（现场计算、不落盘），内容不变则 version 不变，categraf 不会误重启采集实例
- **生效延迟**：改配置后最坏等一个 `reload_interval` 周期生效（默认 60s，可调小）
- **categraf 侧不落盘**：http provider 拉到的配置只在 categraf 内存里；categraf 重启时会重新拉取，需保证管理端在线

## 部署

### 1. 上传文件

```bash
scp categraf-http-admin root@服务器:/etc/categraf/
scp categraf-http-admin.service root@服务器:/etc/systemd/system/
```

### 2. 创建认证配置

```bash
# 生成 categraf 拉取用的独立 token（32 字节 hex）
TOK=$(openssl rand -hex 32)
cat > /etc/categraf/.env << EOF
CONFIG_USER=admin
CONFIG_PASS=你的密码
CATEGRAF_TOKEN=$TOK
EOF
chmod 600 /etc/categraf/.env
```

> `CATEGRAF_TOKEN` 是 categraf 拉取配置端点的独立凭据，与登录账密解耦：登录密码只用于 Web 页面登录，categraf 的 `config.toml` 里只放这个 token。不设 `CATEGRAF_TOKEN` 则 `/api/config/http_response` 公开（不推荐）。

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
headers = ["Authorization", "Bearer <你的CATEGRAF_TOKEN>"]
timeout = 5
reload_interval = 60
```

> `headers` 走 `Authorization: Bearer <token>` 请求头，token 不进 URL / 日志。`<你的CATEGRAF_TOKEN>` 填 `.env` 里那串 hex。若你的 categraf 版本不支持 `http_provider.headers`，需升级或保留该端点公开。

然后重启 categraf：

```bash
systemctl restart categraf
```

> ⚠️ 迁移提醒：目标交给管理端后，删掉本地 `conf/input.http_response/`、`conf/input.net_response/` 中的同名目标，否则 local 与 http 两个 provider 会叠加，同一目标被拨测两遍。

## 使用

访问 `http://你的服务器IP:5000`，用 `.env` 中设置的用户名密码登录。

### 页面功能

- **拨测类型切换**：添加表单顶部选择"网站 HTTP/HTTPS"或"端口 TCP/UDP"，表单字段随类型切换
- **添加 / 编辑 / 删除目标**：列表内联编辑，编辑不允许更改拨测类型
- **TLS 配置**：填 `https://` 开头的 URL 时自动显示，支持私有 CA 证书路径与跳过证书校验
- **TOML 预览**：页面底部实时显示两个插件各自生成的配置，带"📋 复制 http / 📋 复制 net"一键复制（纯 HTTP 环境自动降级 execCommand，无需 HTTPS）

### API

| 方法 | 路径 | 说明 | 需认证 |
|------|------|------|--------|
| GET | `/api/targets` | 查看所有目标 | 是（cookie） |
| POST | `/api/targets` | 添加目标（JSON 或 Form） | 是（cookie） |
| DELETE | `/api/targets/{id}` | 删除目标 | 是（cookie） |
| POST | `/api/targets/{id}/edit` | 编辑目标 | 是（cookie） |
| POST | `/api/targets/{id}/delete` | 删除目标（表单） | 是（cookie） |
| GET | `/api/config/http_response` | categraf 配置拉取端点（含两插件） | 配了 `CATEGRAF_TOKEN` 需 `Authorization: Bearer` |
| GET | `/health` | 健康检查 | 否 |

#### categraf 拉取认证

- **Web 管理 API**（增删改查目标）走 session cookie：先 `POST /login` 拿 cookie 再带 cookie 调用，浏览器登录后天然带。
- **categraf 拉取端点** `/api/config/http_response` 走独立的 `CATEGRAF_TOKEN`，用 `Authorization: Bearer <token>` 请求头校验（categraf 是后台进程，不走登录）。
- 仅接受请求头，不接受 query——避免 token 落进 categraf 日志 / access log / URL。
- token 比对用常量时间比较（`crypto/subtle`），防时序侧信道。
- 页面"配置方式"提示行展示占位符 `Bearer <CATEGRAF_TOKEN>`，**不回显 token 明文**，从 `.env` 取值填入 categraf `config.toml`。

### 网站拨测（http_response）字段

| 字段 | 说明 |
|------|------|
| URL | 拨测地址（http:// 或 https://） |
| 请求方法 | GET / POST / PUT / DELETE / HEAD |
| 名称 | 用于 `[mappings]` 中的 job 标签 |
| 期望状态码 | 默认 200，支持 `200\|301` 多值 |
| 超时时长 | 空则不写，categraf 用默认 3s |
| 跟随重定向 | 勾选后跟随 301/302 跳转 |
| 请求头 / Body | 仅在 POST 时显示 |
| TLS CA 证书路径 | https 目标可填私有 CA（categraf 所在机器上的路径） |
| 跳过证书校验 | 自签名/过期证书场景，勾选后不校验证书链 |

### 端口拨测（net_response）字段

| 字段 | 说明 |
|------|------|
| 目标地址 | `host:port` 格式，如 `10.0.0.1:22` |
| 协议 | TCP / UDP |
| 名称 | 用于 `[mappings]` 中的 job 标签 |
| 连接超时 | 对应 `timeout`，空则用 categraf 默认 1s |
| 发送内容 (send) | 建连后发送的字符串，支持 `\r` `\n` `\t` 转义（输入 `\r\n` 即真实回车换行） |
| 期望响应包含 (expect) | 响应中需包含的字符串，不填则仅检测连通 |
| 读超时 (read_timeout) | 配合 send/expect 的读超时，空则用默认 3s |

> ⚠️ UDP 是无连接协议，不配 send/expect 时"发包不报错即算成功"，判活不可靠；UDP 目标请务必配置 send/expect。

**告警建议**：net_response 用 `result_code != 0` 告警（0=成功 1=超时 2=连接失败 3=读失败 4=expect 不匹配）。

## TOML 合并规则

配置画像相同的目标自动合并到同一个 `[[instances]]`，`job` 名称不参与分组（放在 `[mappings]` 里逐目标打标）：

- **http_response 画像**：方法 + 状态码 + 超时 + Body + 请求头 + 重定向 + TLS
- **net_response 画像**：协议 + 连接超时 + read_timeout + send + expect

```toml
# http_response
[mappings]
"https://site-a.com" = { job = "站点A" }
"https://site-b.com" = { job = "站点B" }

[[instances]]
targets = [
    "https://site-a.com",
    "https://site-b.com"
]
```

```toml
# net_response
[mappings]
"10.0.0.1:22" = { job = "跳板机SSH" }
"192.0.2.20:52522" = { job = "手持机测试251" }

[[instances]]                # 画像：tcp + 3s
targets = [
    "10.0.0.1:22"
]
timeout = "3s"

[[instances]]                # 画像：tcp + 3s + send
targets = [
    "192.0.2.20:52522"
]
timeout = "3s"
send = "\r\n"
```

默认值不落盘：`protocol = "tcp"`、默认状态码 200 等与 categraf 默认行为一致的配置会自动省略。

## 数据与兼容

- 唯一持久化文件是 `targets.json`（与二进制同目录，可用 `-data` 指定路径），备份它即可
- 旧版本（仅 http_response 时期）的 `targets.json` 直接兼容，加载时自动识别为"网站"类型

## 常见操作

### 查看运行状态

```bash
systemctl status categraf-http-admin
journalctl -u categraf-http-admin -f
```

### 修改密码 / 轮换 token

编辑 `.env` 文件后重启：

```bash
vim /etc/categraf/.env
systemctl restart categraf-http-admin
# 轮换 CATEGRAF_TOKEN 后，同步更新 categraf conf/config.toml 里的 headers 并重启 categraf
```

### 不启用页面认证

不创建 `.env` 文件即可，所有页面公开访问（不推荐生产环境使用）。

### 命令行参数

| 参数 | 说明 |
|------|------|
| `-data` | targets.json 路径（默认与二进制同目录） |
| `-env-file` | .env 文件路径 |
| `-user` / `-pass` | 直接指定登录账号（优先级高于 .env） |
| `-categraf-token` | 直接指定 categraf 拉取 token（优先级高于 .env） |

端口固定 5000。

## 编译

需要 Go 1.22+：

```bash
# Linux amd64（静态编译）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o categraf-http-admin .

# 本地 Windows
go build -o categraf-http-admin.exe .
```
