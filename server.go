package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func newRouter(store *Store, port int) http.Handler {
	mux := http.NewServeMux()

	// Web 页面（需要登录）
	mux.HandleFunc("GET /", requireAuth(handleIndex(store, port)))
	mux.HandleFunc("POST /api/targets", requireAuth(handleAddTarget(store)))
	mux.HandleFunc("POST /api/targets/{id}/delete", requireAuth(handleDeleteTarget(store)))
	mux.HandleFunc("POST /api/targets/{id}/edit", requireAuth(handleEditTarget(store)))

	// API（公开）
	mux.HandleFunc("GET /api/targets", handleListTargets(store))
	mux.HandleFunc("DELETE /api/targets/{id}", handleDeleteTarget(store))
	mux.HandleFunc("GET /api/config/http_response", handleCategrafConfig(store))

	// 登录/登出
	if AuthEnabled() {
		mux.HandleFunc("GET /login", handleLogin)
		mux.HandleFunc("POST /login", handleLogin)
		mux.HandleFunc("GET /logout", handleLogout)
	} else {
		// 没开认证时 /login 直接跳首页
		mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusFound)
		})
	}

	// 健康检查（公开）
	mux.HandleFunc("GET /health", handleHealth(store))

	return mux
}

// ── 页面 ──

func handleIndex(store *Store, port int) http.HandlerFunc {
	funcMap := template.FuncMap{
		"eq":      func(a, b any) bool { return a == b },
		"isHTTPS": func(u string) bool { return strings.HasPrefix(strings.ToLower(u), "https://") },
		// escCtl 把真实控制字符转回 \r \n \t 可见形式，供编辑回显/列表展示（与 store.unescapeCtl 对称）
		"escCtl": func(s string) string {
			return strings.NewReplacer("\r", `\r`, "\n", `\n`, "\t", `\t`).Replace(s)
		},
	}
	tmpl := template.Must(template.New("page").Funcs(funcMap).Parse(pageHTML))

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		targets := store.All()

		jobSet := map[string]bool{}
		for _, t := range targets {
			if t.Job != "" {
				jobSet[t.Job] = true
			}
		}
		jobs := make([]string, 0, len(jobSet))
		for j := range jobSet {
			jobs = append(jobs, j)
		}

		errMsg := r.URL.Query().Get("error")

		data := struct {
			Targets     []Target
			TargetCount int
			TOML        string
			NetTOML     string
			Version     string
			Port        int
			Jobs        []string
			Error       string
		}{
			Targets:     targets,
			TargetCount: len(targets),
			TOML:        generateTOML(targets),
			NetTOML:     generateNetTOML(targets),
			Version:     store.ConfigVersion(),
			Port:        port,
			Jobs:        jobs,
			Error:       errMsg,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("E! render template: %v", err)
			http.Error(w, "internal error", 500)
		}
	}
}

// ── API ──

func handleListTargets(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targets := store.All()
		writeJSON(w, 200, map[string]any{
			"targets": targets,
			"total":   len(targets),
		})
	}
}

func handleAddTarget(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			var t Target
			if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
				writeJSON(w, 400, map[string]string{"error": "invalid JSON: " + err.Error()})
				return
			}
			if t.URL == "" {
				writeJSON(w, 400, map[string]string{"error": "url is required"})
				return
			}
			added, err := store.Add(t)
			if err != nil {
				writeJSON(w, 409, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "target": added})
			return
		}

		if err := r.ParseForm(); err != nil {
			writeJSON(w, 400, map[string]string{"error": "bad form"})
			return
		}
		t := targetFromForm(r)
		if _, err := store.Add(t); err != nil {
			http.Redirect(w, r, "/?error="+err.Error(), http.StatusFound)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func handleEditTarget(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "id required"})
			return
		}

		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			var t Target
			if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			updated, err := store.Update(id, t)
			if err != nil {
				writeJSON(w, 409, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "target": updated})
			return
		}

		if err := r.ParseForm(); err != nil {
			writeJSON(w, 400, map[string]string{"error": "bad form"})
			return
		}
		t := targetFromForm(r)
		if _, err := store.Update(id, t); err != nil {
			writeJSON(w, 409, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]bool{"ok": true})
	}
}

func handleDeleteTarget(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, 400, map[string]string{"error": "id required"})
			return
		}
		if !store.Delete(id) {
			writeJSON(w, 404, map[string]string{"error": "target not found"})
			return
		}
		if r.Method == "POST" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		writeJSON(w, 200, map[string]bool{"ok": true})
	}
}

// ── Categraf http_provider ──

func handleCategrafConfig(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targets := store.All()
		version := store.ConfigVersion()

		// 同一个 provider URL 同时下发 http_response 和 net_response 两个插件配置
		resp := map[string]any{
			"version": version,
			"configs": map[string]any{
				"http_response": map[string]any{
					version: map[string]string{
						"config": generateTOML(targets),
						"format": "toml",
					},
				},
				"net_response": map[string]any{
					version: map[string]string{
						"config": generateNetTOML(targets),
						"format": "toml",
					},
				},
			},
		}
		writeJSON(w, 200, resp)
	}
}

// ── 健康检查 ──

func handleHealth(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"status":  "ok",
			"targets": len(store.All()),
		})
	}
}

// ── 工具 ──

func targetFromForm(r *http.Request) Target {
	t := Target{
		Kind:            r.FormValue("kind"),
		URL:             strings.TrimSpace(r.FormValue("url")),
		Job:             strings.TrimSpace(r.FormValue("job")),
		ResponseTimeout: r.FormValue("response_timeout"),
	}
	if t.Kind == "" {
		t.Kind = KindHTTP
	}
	if t.Kind == KindNet {
		t.Protocol = r.FormValue("protocol")
		t.ReadTimeout = r.FormValue("read_timeout")
		t.Send = r.FormValue("send")
		t.Expect = r.FormValue("expect")
		return t
	}

	t.Method = r.FormValue("method")
	t.ExpectedStatusCodes = r.FormValue("expected_status_codes")
	t.Body = r.FormValue("body")
	t.FollowRedirects = r.FormValue("follow_redirects") == "true"
	t.TLSCA = strings.TrimSpace(r.FormValue("tls_ca"))
	t.InsecureSkipVerify = r.FormValue("insecure_skip_verify") == "true"
	// TLS 字段的互斥与 use_tls 推导统一在 store.normalizeTarget 处理
	if t.Method == "" {
		t.Method = "GET"
	}
	if t.ExpectedStatusCodes == "" {
		t.ExpectedStatusCodes = "200"
	}
	if hs := r.FormValue("headers"); hs != "" {
		var h []string
		if err := json.Unmarshal([]byte(hs), &h); err == nil {
			t.Headers = h
		}
	}
	return t
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func serverAddr(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}
