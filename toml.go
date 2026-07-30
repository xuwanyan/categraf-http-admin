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
	InsecureSkipVerify  bool   `json:"insecure_skip_verify"`
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
		InsecureSkipVerify:  t.InsecureSkipVerify,
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

// generateTOML 将 HTTP 拨测目标渲染为 http_response.toml
func generateTOML(targets []Target) string {
	targets = filterKind(targets, KindHTTP)
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
					InsecureSkipVerify:  t.InsecureSkipVerify,
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
		// 状态码必须显式落盘：categraf 不配置时不做任何状态码检查，
		// 5xx 也会报 result_code=0，不能省略 200
		if p.ExpectedStatusCodes != "" {
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
		if p.UseTLS || p.InsecureSkipVerify {
			// insecure_skip_verify 必须配合 use_tls = true 才生效
			b.WriteString("use_tls = true\n")
			if p.TLSCA != "" {
				b.WriteString(fmt.Sprintf("tls_ca = %q\n", p.TLSCA))
			}
			if p.InsecureSkipVerify {
				b.WriteString("insecure_skip_verify = true\n")
			}
		}
	}

	return b.String()
}

func filterKind(targets []Target, kind string) []Target {
	out := make([]Target, 0, len(targets))
	for _, t := range targets {
		k := t.Kind
		if k == "" {
			k = KindHTTP
		}
		if k == kind {
			out = append(out, t)
		}
	}
	return out
}

// netProfile 端口拨测配置画像——完全相同才合并到同一个 [[instances]]
type netProfile struct {
	Protocol        string `json:"protocol"`
	ResponseTimeout string `json:"timeout"`
	ReadTimeout     string `json:"read_timeout"`
	Send            string `json:"send"`
	Expect          string `json:"expect"`
}

// generateNetTOML 将端口拨测目标渲染为 net_response.toml
func generateNetTOML(targets []Target) string {
	targets = filterKind(targets, KindNet)
	if len(targets) == 0 {
		return "# (no targets configured)\n"
	}

	type group struct {
		profile netProfile
		addrs   []string
	}
	groups := make(map[string]*group)
	keys := make([]string, 0)

	for _, t := range targets {
		p := netProfile{
			Protocol:        t.Protocol,
			ResponseTimeout: t.ResponseTimeout,
			ReadTimeout:     t.ReadTimeout,
			Send:            t.Send,
			Expect:          t.Expect,
		}
		kb, _ := json.Marshal(p)
		k := string(kb)
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
			groups[k] = &group{profile: p}
		}
		groups[k].addrs = append(groups[k].addrs, t.URL)
	}

	var b strings.Builder

	// [mappings] 段：给每个 target 打上 job 标签
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

	for gi, k := range keys {
		g := groups[k]
		sort.Strings(g.addrs)

		if gi > 0 {
			b.WriteString("\n")
		}

		b.WriteString("[[instances]]\n")
		b.WriteString("targets = [\n")
		for i, a := range g.addrs {
			comma := ","
			if i == len(g.addrs)-1 {
				comma = ""
			}
			b.WriteString(fmt.Sprintf("    %q%s\n", a, comma))
		}
		b.WriteString("]\n")

		p := g.profile
		// tcp 为 categraf 默认值，仅 udp 时显式落盘
		if p.Protocol == "udp" {
			b.WriteString("protocol = \"udp\"\n")
		}
		if p.ResponseTimeout != "" {
			b.WriteString(fmt.Sprintf("timeout = %q\n", p.ResponseTimeout))
		}
		if p.ReadTimeout != "" {
			b.WriteString(fmt.Sprintf("read_timeout = %q\n", p.ReadTimeout))
		}
		if p.Send != "" {
			b.WriteString(fmt.Sprintf("send = %q\n", p.Send))
		}
		if p.Expect != "" {
			b.WriteString(fmt.Sprintf("expect = %q\n", p.Expect))
		}
	}

	return b.String()
}
