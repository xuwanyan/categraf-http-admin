package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

// Target 单个拨测目标
type Target struct {
	ID                   string            `json:"id"`
	URL                  string            `json:"url"`
	Method               string            `json:"method"`
	Job                  string            `json:"job"`
	ExpectedStatusCodes  string            `json:"expected_status_codes"`
	ResponseTimeout      string            `json:"response_timeout,omitempty"`
	FollowRedirects      bool              `json:"follow_redirects"`
	Headers              []string          `json:"headers,omitempty"`
	Body                 string            `json:"body,omitempty"`
	UseTLS               bool              `json:"use_tls"`
	TLSCA                string            `json:"tls_ca,omitempty"`
	CreatedAt            string            `json:"created_at"`
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
		return s, nil
	}

	var sd StoreData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("parse store file: %w", err)
	}
	s.targets = sd.Targets
	return s, nil
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

// Add 添加目标（含重复校验）
func (s *Store) Add(t Target) (Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	if t.Method == "" {
		t.Method = "GET"
	}
	if t.ExpectedStatusCodes == "" {
		t.ExpectedStatusCodes = "200"
	}

	s.targets = append(s.targets, t)
	return t, s.persist()
}

// Update 修改目标
func (s *Store) Update(id string, t Target) (Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	for i, existing := range s.targets {
		if existing.ID == id {
			if t.URL != "" {
				s.targets[i].URL = t.URL
			}
			if t.Method != "" {
				s.targets[i].Method = t.Method
			}
			if t.ExpectedStatusCodes != "" {
				s.targets[i].ExpectedStatusCodes = t.ExpectedStatusCodes
			}
			// Job 不允许为空
			if t.Job == "" {
				return Target{}, fmt.Errorf("job is required")
			}
			s.targets[i].Job = t.Job
			s.targets[i].Body = t.Body
			s.targets[i].FollowRedirects = t.FollowRedirects
			s.targets[i].UseTLS = t.UseTLS
			s.targets[i].TLSCA = t.TLSCA
			if t.Headers != nil {
				s.targets[i].Headers = t.Headers
			}
			return s.targets[i], s.persist()
		}
	}
	return Target{}, fmt.Errorf("target not found: %s", id)
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
	sorted := make([]Target, len(s.targets))
	copy(sorted, s.targets)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	for _, t := range sorted {
		fmt.Fprintf(h, "%s|%s|%s|%s|%s|%s|%v|%v|%s", t.ID, t.URL, t.Method, t.Job, t.ExpectedStatusCodes, t.ResponseTimeout, t.FollowRedirects, t.UseTLS, t.TLSCA)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *Store) persist() error {
	sd := StoreData{
		Targets:       s.targets,
	}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}