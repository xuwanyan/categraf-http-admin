package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

const (
	sessionCookieName = "ccsid"
	sessionMaxAge     = 24 * time.Hour
)

var (
	authUser string
	authPass string
	secret   []byte
)

// InitAuth 初始化认证（程序启动时调用）
func InitAuth(user, pass string) {
	authUser = user
	authPass = pass
	// 生成随机签名密钥
	secret = make([]byte, 32)
	rand.Read(secret)
}

// AuthEnabled 返回是否启用了认证
func AuthEnabled() bool {
	return authUser != "" && authPass != ""
}

// requireAuth 返回一个中间件，只保护 Web 页面路由
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !AuthEnabled() {
			next(w, r)
			return
		}

		// 检查 session cookie
		cookie, err := r.Cookie(sessionCookieName)
		if err == nil && validateSession(cookie.Value) {
			next(w, r)
			return
		}

		// 未登录，重定向到登录页
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// 生成 session token
func newSession() string {
	data := fmt.Sprintf("%s|%d|%s", authUser, time.Now().UnixNano(), authPass)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	sig := hex.EncodeToString(mac.Sum(nil))
	return hex.EncodeToString([]byte(data)) + "." + sig
}

// 验证 session token
func validateSession(token string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	data, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	// 验证签名
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return false
	}
	// 验证用户名和密码未变
	pieces := strings.SplitN(string(data), "|", 3)
	if len(pieces) != 3 {
		return false
	}
	return pieces[0] == authUser && pieces[2] == authPass
}

// ── 登录页面 ──

const loginHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>登录 - Categraf 拨测管理</title>
<link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>📡</text></svg>">
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; display: flex; justify-content: center; align-items: center; min-height: 100vh; }
.card { background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); padding: 40px; width: 360px; }
h1 { font-size: 20px; margin-bottom: 24px; text-align: center; color: #1a1a1a; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 13px; font-weight: 500; margin-bottom: 4px; color: #555; }
.form-group input { width: 100%; padding: 10px 12px; border: 1px solid #dadce0; border-radius: 6px; font-size: 14px; }
.form-group input:focus { border-color: #1a73e8; outline: none; }
.btn { width: 100%; padding: 10px; background: #1a73e8; color: white; border: none; border-radius: 6px; font-size: 14px; cursor: pointer; }
.btn:hover { background: #1557b0; }
.error { background: #fce8e6; border: 1px solid #d93025; border-radius: 6px; padding: 10px 16px; margin-bottom: 16px; color: #c5221f; font-size: 14px; text-align: center; }
</style>
</head>
<body>
<div class="card">
<h1>🔐 Categraf 拨测管理</h1>
{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form action="/login" method="POST">
<div class="form-group">
<label>用户名</label>
<input type="text" name="username" required autofocus>
</div>
<div class="form-group">
<label>密码</label>
<input type="password" name="password" required>
</div>
<button type="submit" class="btn">登录</button>
</form>
</div>
</body>
</html>`

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 已经登录则跳回首页
		cookie, err := r.Cookie(sessionCookieName)
		if err == nil && validateSession(cookie.Value) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		data := struct{ Error string }{Error: ""}
		if r.URL.Query().Get("error") != "" {
			data.Error = "用户名或密码错误"
		}
		tmpl := template.Must(template.New("login").Parse(loginHTML))
		tmpl.Execute(w, data)
		return
	}

	// POST：验证登录
	r.ParseForm()
	user := r.FormValue("username")
	pass := r.FormValue("password")

	if user != authUser || pass != authPass {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}

	// 设置 session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    newSession(),
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(sessionMaxAge.Seconds()),
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}
