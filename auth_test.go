package main

import (
	"net/http"
	"net/http/httptest"
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
		name, auth, url  string
		want             int
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
