package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// 拨测类型
const (
	KindHTTP = "http" // http_response 网站拨测
	KindNet  = "net"  // net_response TCP/UDP 端口拨测
)

// statusCodesRe 期望状态码格式：三位数字，多值用 | 分隔（categraf 为子串匹配，不支持 2xx 通配）
var statusCodesRe = regexp.MustCompile(`^\d{3}(\|\d{3})*$`)

func validateStatusCodes(s string) error {
	if s == "" {
		return nil
	}
	if !statusCodesRe.MatchString(s) {
		return fmt.Errorf("状态码格式非法: %s（应为三位数字，多值用|分隔，如 200 或 200|301）", s)
	}
	return nil
}

// Target 单个拨测目标
// Kind=http 时 URL 为网站地址；Kind=net 时 URL 为 host:port 端口地址
type Target struct {
	ID                  string   `json:"id"`
	Kind                string   `json:"kind,omitempty"`
	URL                 string   `json:"url"`
	Method              string   `json:"method,omitempty"`
	Job                 string   `json:"job"`
	ExpectedStatusCodes string   `json:"expected_status_codes,omitempty"`
	ResponseTimeout     string   `json:"response_timeout,omitempty"`
	FollowRedirects     bool     `json:"follow_redirects,omitempty"`
	Headers             []string `json:"headers,omitempty"`
	Body                string   `json:"body,omitempty"`
	UseTLS              bool     `json:"use_tls,omitempty"`
	TLSCA               string   `json:"tls_ca,omitempty"`
	InsecureSkipVerify  bool     `json:"insecure_skip_verify,omitempty"`
	// 端口拨测（net_response）专属字段
	Protocol    string `json:"protocol,omitempty"`     // tcp / udp
	ReadTimeout string `json:"read_timeout,omitempty"` // 配合 send/expect 的读超时
	Send        string `json:"send,omitempty"`         // 建连后发送的内容
	Expect      string `json:"expect,omitempty"`       // 期望响应包含的字符串
	CreatedAt   string `json:"created_at"`
}

// HeadersJSON 返回 Headers 的 JSON 字符串，供模板使用
func (t Target) HeadersJSON() string {
	if len(t.Headers) == 0 {
		return ""
	}
	b, _ := json.Marshal(t.Headers)
	return string(b)
}

// StoreData 持久化包装
type StoreData struct {
	Targets []Target `json:"targets"`
}

// Store 数据存储
type Store struct {
	mu      sync.RWMutex
	path    string
	targets []Target
}

// NewStore 从 JSON 文件加载数据
func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		s.targets = []Target{}
		return s, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read store file: %w", err)
	}

	// 兼容旧格式（纯数组）
	if len(data) > 0 && data[0] == '[' {
		if err := json.Unmarshal(data, &s.targets); err != nil {
			return nil, fmt.Errorf("parse store file: %w", err)
		}
		s.migrateKind()
		return s, nil
	}

	var sd StoreData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("parse store file: %w", err)
	}
	s.targets = sd.Targets
	s.migrateKind()
	return s, nil
}

// migrateKind 旧数据没有 kind 字段，统一补为 http
func (s *Store) migrateKind() {
	for i := range s.targets {
		if s.targets[i].Kind == "" {
			s.targets[i].Kind = KindHTTP
		}
	}
}

// All 返回所有目标副本
func (s *Store) All() []Target {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Target, len(s.targets))
	copy(out, s.targets)
	return out
}

// URLExists 检查 URL 是否已存在（排除指定 ID）
func (s *Store) URLExists(url, excludeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.targets {
		if t.URL == url && t.ID != excludeID {
			return true
		}
	}
	return false
}

// normalizeTLS 归一化 TLS 相关字段（表单与 JSON 两条入口共用）：
// 跳过校验与 CA 互斥（跳过优先，清空 CA）；use_tls 由是否需要 TLS 配置块自动推导。
func normalizeTLS(t *Target) {
	t.TLSCA = strings.TrimSpace(t.TLSCA)
	if t.InsecureSkipVerify {
		t.TLSCA = ""
	}
	t.UseTLS = t.InsecureSkipVerify || t.TLSCA != ""
}

// unescapeCtl 把表单里输入的 \r \n \t 转义序列还原成真实控制字符，
// 与手写 TOML 中 send = "\r\n" 的行为保持一致；JSON 入口传真实控制字符时不受影响。
func unescapeCtl(s string) string {
	return strings.NewReplacer(`\r`, "\r", `\n`, "\n", `\t`, "\t").Replace(s)
}

// normalizeTarget 按拨测类型归一化并校验字段（表单与 JSON 两条入口共用）
func normalizeTarget(t *Target) error {
	t.URL = strings.TrimSpace(t.URL)
	if t.Kind == "" {
		t.Kind = KindHTTP
	}
	switch t.Kind {
	case KindHTTP:
		// 清理端口拨测专属字段
		t.Protocol, t.ReadTimeout, t.Send, t.Expect = "", "", "", ""
		normalizeTLS(t)
	case KindNet:
		if t.Protocol == "" {
			t.Protocol = "tcp"
		}
		if t.Protocol != "tcp" && t.Protocol != "udp" {
			return fmt.Errorf("协议非法: %s（仅支持 tcp/udp）", t.Protocol)
		}
		if t.URL != "" {
			host, port, err := net.SplitHostPort(t.URL)
			if err != nil || host == "" || port == "" {
				return fmt.Errorf("目标地址格式非法: %s（应为 host:port，如 10.0.0.1:22）", t.URL)
			}
		}
		// 清理 HTTP 拨测专属字段
		t.Method, t.ExpectedStatusCodes, t.Body = "", "", ""
		t.Headers = nil
		t.FollowRedirects = false
		t.UseTLS, t.TLSCA, t.InsecureSkipVerify = false, "", false
		t.Send = unescapeCtl(t.Send)
		t.Expect = unescapeCtl(t.Expect)
	default:
		return fmt.Errorf("拨测类型非法: %s", t.Kind)
	}
	return nil
}

// Add 添加目标（含重复校验）
func (s *Store) Add(t Target) (Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := normalizeTarget(&t); err != nil {
		return Target{}, err
	}

	if t.URL == "" {
		return Target{}, fmt.Errorf("url is required")
	}
	if t.Job == "" {
		return Target{}, fmt.Errorf("job is required")
	}
	for _, existing := range s.targets {
		if existing.URL == t.URL {
			return Target{}, fmt.Errorf("duplicate url: %s", t.URL)
		}
		if existing.Job == t.Job {
			return Target{}, fmt.Errorf("duplicate job: %s", t.Job)
		}
	}

	t.ID = fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s-%d", t.URL, time.Now().UnixNano()))))[:12]
	t.CreatedAt = time.Now().Format(time.RFC3339)

	if t.Kind == KindHTTP {
		if t.Method == "" {
			t.Method = "GET"
		}
		if t.ExpectedStatusCodes == "" {
			t.ExpectedStatusCodes = "200"
		}
		if err := validateStatusCodes(t.ExpectedStatusCodes); err != nil {
			return Target{}, err
		}
	}

	s.targets = append(s.targets, t)
	return t, s.persist()
}

// Update 修改目标（拨测类型不允许修改，始终沿用已存目标的 Kind）
func (s *Store) Update(id string, t Target) (Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 先找到已存目标，按其类型归一化
	idx := -1
	for i, existing := range s.targets {
		if existing.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return Target{}, fmt.Errorf("target not found: %s", id)
	}

	t.Kind = s.targets[idx].Kind
	if err := normalizeTarget(&t); err != nil {
		return Target{}, err
	}
	if t.Kind == KindHTTP {
		if err := validateStatusCodes(t.ExpectedStatusCodes); err != nil {
			return Target{}, err
		}
	}

	// 检查重复
	for _, existing := range s.targets {
		if existing.ID != id {
			if t.URL != "" && existing.URL == t.URL {
				return Target{}, fmt.Errorf("duplicate url: %s", t.URL)
			}
			if t.Job != "" && existing.Job == t.Job {
				return Target{}, fmt.Errorf("duplicate job: %s", t.Job)
			}
		}
	}

	// Job 不允许为空
	if t.Job == "" {
		return Target{}, fmt.Errorf("job is required")
	}

	cur := &s.targets[idx]
	if t.URL != "" {
		cur.URL = t.URL
	}
	cur.Job = t.Job
	switch t.Kind {
	case KindHTTP:
		if t.Method != "" {
			cur.Method = t.Method
		}
		if t.ExpectedStatusCodes != "" {
			cur.ExpectedStatusCodes = t.ExpectedStatusCodes
		}
		cur.ResponseTimeout = t.ResponseTimeout
		cur.Body = t.Body
		cur.FollowRedirects = t.FollowRedirects
		cur.UseTLS = t.UseTLS
		cur.TLSCA = t.TLSCA
		cur.InsecureSkipVerify = t.InsecureSkipVerify
		if t.Headers != nil {
			cur.Headers = t.Headers
		}
	case KindNet:
		cur.Protocol = t.Protocol
		cur.ResponseTimeout = t.ResponseTimeout
		cur.ReadTimeout = t.ReadTimeout
		cur.Send = t.Send
		cur.Expect = t.Expect
	}
	return *cur, s.persist()
}

// Delete 删除目标
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, t := range s.targets {
		if t.ID == id {
			s.targets = append(s.targets[:i], s.targets[i+1:]...)
			s.persist()
			return true
		}
	}
	return false
}

// ConfigVersion 计算当前配置 MD5
func (s *Store) ConfigVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := md5.New()
	// 生成器版本盐：TOML 生成逻辑变更时递增，强制 categraf 重新拉取配置
	fmt.Fprintf(h, "schema:v4|")
	sorted := make([]Target, len(s.targets))
	copy(sorted, s.targets)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	for _, t := range sorted {
		fmt.Fprintf(h, "%s|%s|%s|%s|%s|%s|%s|%v|%v|%s|%v|%s|%s|%s|%s", t.ID, t.Kind, t.URL, t.Method, t.Job, t.ExpectedStatusCodes, t.ResponseTimeout, t.FollowRedirects, t.UseTLS, t.TLSCA, t.InsecureSkipVerify, t.Protocol, t.ReadTimeout, t.Send, t.Expect)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *Store) persist() error {
	sd := StoreData{
		Targets: s.targets,
	}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
