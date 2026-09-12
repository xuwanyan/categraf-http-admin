package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// TestRequireCategrafToken 验证 categraf 拉取端点的 token 校验：
// 仅接受 Authorization: Bearer <token> 请求头，拒绝 query token，未配 token 时公开。
func TestRequireCategrafToken(t *testing.T) {
	// 临时保存 / 恢复包级状态，避免污染其他测试
	orig := categrafToken
	defer func() { categrafToken = orig }()

	// 1. 未配 token：端点应公开（向后兼容）
	InitCategrafToken("")
	h := requireCategrafToken(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"ok": "1"})
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest("GET", "/api/config/http_response", nil))
	if rec.Code != 200 {
		t.Fatalf("未配 token 应公开放行，got %d", rec.Code)
	}

	// 2. 配了 token：只接受正确的 Bearer 头
	InitCategrafToken("sek")
	cases := []struct {
		name, auth, url string
		want            int
	}{
		{"无 header", "", "/api/config/http_response", 401},
		{"错误 token", "Bearer nope", "/api/config/http_response", 401},
		{"正确 token", "Bearer sek", "/api/config/http_response", 200},
		{"query token 不应接受", "", "/api/config/http_response?token=sek", 401},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", c.url, nil)
		if c.auth != "" {
			req.Header.Set("Authorization", c.auth)
		}
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: got %d, want %d", c.name, rec.Code, c.want)
		}
	}
}

// TestCategrafEndpointRouter 路由级验证：newRouter 挂载的
// /api/config/http_response 在启用 token 后按 Bearer 头校验，
// 且放行时响应体保持 categraf http_provider 期望的 JSON 契约
// （configs.<plugin> 的内层 key 必须等于顶层 version）。
func TestCategrafEndpointRouter(t *testing.T) {
	// 保存 / 恢复包级认证状态
	origTok := categrafToken
	origUser, origPass := authUser, authPass
	defer func() {
		categrafToken = origTok
		authUser, authPass = origUser, origPass
	}()

	InitCategrafToken("rtok")
	InitAuth("", "") // 关闭页面认证，避免 cookie 分支干扰

	store, err := NewStore(filepath.Join(t.TempDir(), "targets.json"))
	if err != nil {
		t.Fatal(err)
	}
	handler := newRouter(store, 5000)

	get := func(url, auth string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", url, nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	// 所有非 Bearer 途径都必须被拒
	denied := map[string]struct{ url, auth string }{
		"无 Authorization 头":   {"/api/config/http_response", ""},
		"错误 token":          {"/api/config/http_response", "Bearer nope"},
		"query token":         {"/api/config/http_response?token=rtok", ""},
		"Basic 头":            {"/api/config/http_response", "Basic cm90"},
		"Bearer 后多余空格":    {"/api/config/http_response", "Bearer rtok extra"},
	}
	for name, args := range denied {
		if rec := get(args.url, args.auth); rec.Code != 401 {
			t.Errorf("%s: got %d, want 401", name, rec.Code)
		}
	}

	// 正确 token 放行，校验响应契约
	rec := get("/api/config/http_response", "Bearer rtok")
	if rec.Code != 200 {
		t.Fatalf("正确 token: got %d, want 200", rec.Code)
	}
	var body struct {
		Version string                              `json:"version"`
		Configs map[string]map[string]map[string]string `json:"configs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	if body.Version == "" {
		t.Fatal("version 为空")
	}
	for _, plugin := range []string{"http_response", "net_response"} {
		entries, ok := body.Configs[plugin]
		if !ok {
			t.Errorf("configs 缺 %s", plugin)
			continue
		}
		if len(entries) != 1 {
			t.Errorf("%s: 应有 1 个 version 条目，got %d", plugin, len(entries))
			continue
		}
		for v, cfg := range entries {
			if v != body.Version {
				t.Errorf("%s: 内层 key %q 应等于顶层 version %q", plugin, v, body.Version)
			}
			if cfg["format"] != "toml" {
				t.Errorf("%s: format = %q, want toml", plugin, cfg["format"])
			}
			if cfg["config"] == "" {
				t.Errorf("%s: config 为空", plugin)
			}
		}
	}

	// /health 不受 token 影响，始终公开
	if rec := get("/health", ""); rec.Code != 200 {
		t.Errorf("/health: got %d, want 200", rec.Code)
	}
}
