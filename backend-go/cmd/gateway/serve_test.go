package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// safeOut là stdout giả an toàn: go-redis ghi log từ goroutine nền trong lúc test đọc.
type safeOut struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *safeOut) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *safeOut) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func serveEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL":    "postgres://u:p@127.0.0.1:1/db",
		"REDIS_URL":       "redis://127.0.0.1:1/0",
		"JWT_SECRET_KEY":  "0123456789abcdef0123456789abcdef",
		"BLOB_ENDPOINT":   "127.0.0.1:1",
		"BLOB_BUCKET":     "b",
		"BLOB_ACCESS_KEY": "ak",
		"BLOB_SECRET_KEY": "sk",
		"HTTP_ADDR":       "127.0.0.1:0",
	}
}

func runGateway(t *testing.T, env map[string]string) (rc int, out string, elapsed time.Duration) {
	t.Helper()
	w := &safeOut{}
	start := time.Now()
	rc = run([]string{"serve"}, func(k string) string { return env[k] }, w, w)
	return rc, w.String(), time.Since(start)
}

func oneLine(t *testing.T, out string) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("số dòng stdout = %d, muốn đúng 1: %s", len(lines), out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil {
		t.Fatalf("dòng không phải JSON: %s", lines[0])
	}
	return m
}

// TestServe_MissingEnvExitsBeforeListening: thiếu biến bắt buộc → đúng một dòng JSON mức error có
// `missing` (chỉ tên), thoát 1 trong ≤ 1 s, không mở cổng (US-PG-01 AC1).
func TestServe_MissingEnvExitsBeforeListening(t *testing.T) {
	t.Parallel()
	env := serveEnv()
	env["HTTP_ADDR"] = "127.0.0.1:18079"
	delete(env, "JWT_SECRET_KEY")

	rc, out, elapsed := runGateway(t, env)
	if rc != 1 {
		t.Fatalf("rc = %d, muốn 1", rc)
	}
	if elapsed > time.Second {
		t.Fatalf("thoát sau %v, muốn ≤ 1 s", elapsed)
	}
	m := oneLine(t, out)
	if m["level"] != "ERROR" {
		t.Errorf("level = %v, muốn ERROR", m["level"])
	}
	missing, _ := m["missing"].([]any)
	if len(missing) != 1 || missing[0] != "JWT_SECRET_KEY" {
		t.Fatalf("missing = %v", m["missing"])
	}
	if strings.Contains(out, "postgres://u:p") {
		t.Fatalf("log in giá trị biến: %s", out)
	}
	if c, err := net.DialTimeout("tcp", "127.0.0.1:18079", 200*time.Millisecond); err == nil {
		_ = c.Close()
		t.Fatal("cổng vẫn mở dù cấu hình thiếu biến")
	}
}

// TestServe_InvalidEnvNamesVariable: giá trị sai → thoát 1, nêu TÊN biến, không in giá trị,
// không có dòng `config loaded` (US-PG-01 AC2).
func TestServe_InvalidEnvNamesVariable(t *testing.T) {
	t.Parallel()
	cases := []struct{ key, value, leak string }{
		{"JWT_SECRET_KEY", "short-secret-31-bytes-xxxxxxxxx", "short-secret"},
		{"DB_MAX_CONNS", "0", ""},
		{"JWT_EXPIRATION", "abc", "abc"},
		{"CORS_ORIGINS", "*", ""},
		{"APP_ENV", "staging", "staging"},
		{"BCRYPT_COST", "3", ""},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()
			env := serveEnv()
			env[tc.key] = tc.value

			rc, out, elapsed := runGateway(t, env)
			if rc != 1 {
				t.Fatalf("rc = %d, muốn 1", rc)
			}
			if elapsed > time.Second {
				t.Fatalf("thoát sau %v, muốn ≤ 1 s", elapsed)
			}
			m := oneLine(t, out)
			if m["level"] != "ERROR" {
				t.Errorf("level = %v", m["level"])
			}
			if msg, _ := m["msg"].(string); !strings.Contains(msg, tc.key) {
				t.Fatalf("msg = %q, muốn nêu %s", msg, tc.key)
			}
			// Chỉ so trong các trường do ứng dụng viết; `trace_id` / `instance` / `time` là chuỗi ngẫu nhiên và có thể
			// tình cờ chứa "abc" (BUG-PG-3).
			if tc.leak != "" {
				for k, v := range m {
					if k == "trace_id" || k == "instance" || k == "time" {
						continue
					}
					if strings.Contains(fmt.Sprint(v), tc.leak) {
						t.Fatalf("log lộ giá trị %q ở trường %s: %s", tc.leak, k, out)
					}
				}
			}
			if strings.Contains(out, "config loaded") {
				t.Fatalf("không được chạy tiếp sau cấu hình sai: %s", out)
			}
		})
	}
}

// TestServe_WaitsForDepsThenExits: cấu hình hợp lệ nhưng DB/Redis chưa lên → ghi `config loaded`,
// chờ tới STARTUP_TIMEOUT, log `dependency not ready` rồi thoát 1 (US-PG-01 AC14).
func TestServe_WaitsForDepsThenExits(t *testing.T) {
	t.Parallel()
	env := serveEnv()
	env["STARTUP_TIMEOUT"] = "2s"

	rc, out, elapsed := runGateway(t, env)
	if rc != 1 {
		t.Fatalf("rc = %d, muốn 1", rc)
	}
	if elapsed < 1500*time.Millisecond || elapsed > 9*time.Second {
		t.Fatalf("chạy %v, muốn ≈ STARTUP_TIMEOUT 2 s", elapsed)
	}

	var loaded, warns, errs int
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("dòng không phải JSON: %s", line)
		}
		switch {
		case m["msg"] == "config loaded":
			loaded++
			if m["secrets"] != "[redacted]" || m["db_via"] != "direct" {
				t.Errorf("dòng config loaded = %v", m)
			}
		case m["msg"] == "dependency not ready" && m["level"] == "WARN":
			warns++
			if m["dependency"] != "db" && m["dependency"] != "redis" {
				t.Errorf("dependency = %v", m["dependency"])
			}
		case m["msg"] == "dependency not ready" && m["level"] == "ERROR":
			errs++
		}
	}
	if loaded != 1 {
		t.Errorf("số dòng `config loaded` = %d, muốn 1", loaded)
	}
	if warns < 2 {
		t.Errorf("số dòng warn chờ phụ thuộc = %d, muốn ≥ 2 (mỗi giây một dòng)", warns)
	}
	if errs < 1 {
		t.Errorf("thiếu dòng error `dependency not ready`")
	}
	if strings.Contains(out, "0123456789abcdef") {
		t.Errorf("log lộ JWT_SECRET_KEY: %s", out)
	}
}
