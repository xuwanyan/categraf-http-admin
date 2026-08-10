package main

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"net/http"
	"sync"
	"time"
)

const (
	sessionCookieName = "ccsid"
	sessionMaxAge     = 24 * time.Hour
	tokenLen          = 32 // 256-bit 随机 token
)

var (
	authUser string
	authPass string
)

// sessionStore 服务端 session 存储
type sessionStore struct {
	mu      sync.RWMutex
	sessions map[string]struct {
		user     string
		expires  time.Time
	}
}

var sessions = sessionStore{
	sessions: make(map[string]struct {
		user     string
		expires  time.Time
	}),
}

// gcSessions 清理过期 session（每次新建时顺带清理，不过频）
func (s *sessionStore) gc() {
	now := time.Now()
	for tok, sess := range s.sessions {
		if sess.expires.Before(now) {
			delete(s.sessions, tok)
		}
	}
}

// InitAuth 初始化认证（程序启动时调用）
func InitAuth(user, pass string) {
	authUser = user
	authPass = pass
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

// 生成 session token（纯随机字符串，不携带任何明文信息）
func newSession() string {
	// 服务端过期清理
	sessions.mu.Lock()
	sessions.gc()
	sessions.mu.Unlock()

	// 生成 256-bit 随机 token
	buf := make([]byte, tokenLen)
	rand.Read(buf)
	token := hex.EncodeToString(buf)

	sessions.mu.Lock()
	sessions.sessions[token] = struct {
		user    string
		expires time.Time
	}{
		user:    authUser,
		expires: time.Now().Add(sessionMaxAge),
	}
	sessions.mu.Unlock()

	return token
}

// 验证 session token（检查是否存在、未过期、且用户名密码未变）
func validateSession(token string) bool {
	if token == "" {
		return false
	}

	sessions.mu.RLock()
	sess, ok := sessions.sessions[token]
	sessions.mu.RUnlock()

	if !ok {
		return false
	}

	// 过期校验
	if time.Now().After(sess.expires) {
		sessions.mu.Lock()
		delete(sessions.sessions, token)
		sessions.mu.Unlock()
		return false
	}

	// 用户名/密码未变
	return sess.user == authUser
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
		SameSite: http.SameSiteLaxMode,
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
