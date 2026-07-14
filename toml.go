package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// configProfile 目标配置画像——只有完全相同的配置才能合并到同一个 [[instances]]
type configProfile struct {
	Method              string `json:"method"`
	ExpectedStatusCodes string `json:"expected_status_codes"`
	ResponseTimeout     string `json:"response_timeout"`
	Body                string `json:"body"`
	Headers             string `json:"headers"`
	FollowRedirects     bool   `json:"follow_redirects"`
	UseTLS              bool   `json:"use_tls"`
	TLSCA               string `json:"tls_ca"`
}

func profileKey(t Target) string {
	h := configProfile{
		Method:              t.Method,
		ExpectedStatusCodes: t.ExpectedStatusCodes,
		ResponseTimeout:     t.ResponseTimeout,
		Body:                t.Body,
		Headers:             headerKey(t.Headers),
		FollowRedirects:     t.FollowRedirects,
		UseTLS:              t.UseTLS,
		TLSCA:               t.TLSCA,
	}
	b, _ := json.Marshal(h)
	return string(b)
}

func headerKey(h []string) string {
	if len(h) == 0 {
		return ""
	}
	sorted := make([]string, len(h))
	copy(sorted, h)
	sort.Strings(sorted)
	return strings.Join(sorted, "\x00")
}

// generateTOML 将 targets 渲染为 http_response.toml
func generateTOML(targets []Target) string {
	if len(targets) == 0 {
		return "# (no targets configured)\n"
	}

	// 按完整配置分组合并
	type group struct {
		profile configProfile
		urls    []string
	}
	groups := make(map[string]*group)
	keys := make([]string, 0)

	for _, t := range targets {
		k := profileKey(t)
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
			groups[k] = &group{
				profile: configProfile{
					Method:              t.Method,
					ExpectedStatusCodes: t.ExpectedStatusCodes,
					ResponseTimeout:     t.ResponseTimeout,
					Body:                t.Body,
					Headers:             headerKey(t.Headers),
					FollowRedirects:     t.FollowRedirects,
					UseTLS:              t.UseTLS,
					TLSCA:               t.TLSCA,
				},
				urls: make([]string, 0),
			}
		}
		groups[k].urls = append(groups[k].urls, t.URL)
	}

	var b strings.Builder

	// [mappings] 段
	hasMapping := false
	for _, t := range targets {
		if t.Job != "" {
			if !hasMapping {
				b.WriteString("[mappings]\n")
				hasMapping = true
			}
			b.WriteString(fmt.Sprintf("%q = { job = %q }\n", t.URL, t.Job))
		}
	}
	if hasMapping {
		b.WriteString("\n")
	}

	// [[instances]] 段
	for gi, k := range keys {
		g := groups[k]
		sort.Strings(g.urls)

		if gi > 0 {
			b.WriteString("\n")
		}

		b.WriteString("[[instances]]\n")
		b.WriteString("targets = [\n")
		for i, u := range g.urls {
			comma := ","
			if i == len(g.urls)-1 {
				comma = ""
			}
			b.WriteString(fmt.Sprintf("    %q%s\n", u, comma))
		}
		b.WriteString("]\n")

		p := g.profile

		// 超时：为空则不写，categraf 用默认值
		if p.ResponseTimeout != "" {
			b.WriteString(fmt.Sprintf("response_timeout = %q\n", p.ResponseTimeout))
		}

		if p.Method != "" && p.Method != "GET" {
			b.WriteString(fmt.Sprintf("method = %q\n", p.Method))
		}
		if p.ExpectedStatusCodes != "" && p.ExpectedStatusCodes != "200" {
			b.WriteString(fmt.Sprintf("expect_response_status_codes = %q\n", p.ExpectedStatusCodes))
		}
		if p.FollowRedirects {
			b.WriteString("follow_redirects = true\n")
		}
		if p.Headers != "" {
			parts := make([]string, 0, len(strings.Split(p.Headers, "\x00")))
			for _, h := range strings.Split(p.Headers, "\x00") {
				parts = append(parts, fmt.Sprintf("%q", h))
			}
			b.WriteString(fmt.Sprintf("headers = [%s]\n", strings.Join(parts, ", ")))
		}
		if p.Body != "" {
			b.WriteString(fmt.Sprintf("body = \"\"\"\n%s\n\"\"\"\n", p.Body))
		}
		if p.UseTLS {
			b.WriteString("use_tls = true\n")
			if p.TLSCA != "" {
				b.WriteString(fmt.Sprintf("tls_ca = %q\n", p.TLSCA))
			}
		}
	}

	return b.String()
}