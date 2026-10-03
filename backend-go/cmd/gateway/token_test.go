package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

const (
	tokSecret = "0123456789abcdef0123456789abcdef"
	tokSub    = "00000000-0000-7000-8000-000000000001"
)

var jwtShape = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*$`)

// runTok chạy runToken với env giả, trả rc + stdout + stderr.
func runTok(env map[string]string, args ...string) (int, string, string) {
	var out, errBuf bytes.Buffer
	rc := runToken(args, func(k string) string { return env[k] }, &out, &errBuf)
	return rc, out.String(), errBuf.String()
}

func devEnv() map[string]string { return map[string]string{"JWT_SECRET_KEY": tokSecret} }

// payloadOf giải mã claim của token in ra stdout.
func payloadOf(t *testing.T, stdout string) map[string]any {
	t.Helper()
	line := strings.TrimSpace(stdout)
	parts := strings.Split(line, ".")
	if len(parts) != 3 {
		t.Fatalf("stdout không phải JWT 3 đoạn: %q", line)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("giải base64url payload: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("giải JSON payload: %v", err)
	}
	return m
}

func TestTokenCommand(t *testing.T) {
	t.Run("in đúng một dòng JWT", func(t *testing.T) {
		rc, out, errOut := runTok(devEnv(), "--role", "ADMIN", "--sub", tokSub)
		if rc != 0 {
			t.Fatalf("rc=%d, stderr=%q", rc, errOut)
		}
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) != 1 || !jwtShape.MatchString(lines[0]) {
			t.Fatalf("stdout=%q, muốn đúng 1 dòng JWT", out)
		}
		if strings.Contains(errOut, lines[0]) {
			t.Fatal("stderr chứa token")
		}
		p := payloadOf(t, out)
		if p["role"] != "ADMIN" || p["sub"] != tokSub || p["iss"] != "edupilot" || p["aud"] != "edupilot-api" {
			t.Fatalf("claim=%v", p)
		}
		if d := p["exp"].(float64) - p["iat"].(float64); d != 900 {
			t.Fatalf("exp-iat=%v, muốn 900 (mặc định 15 phút)", d)
		}
		if email, _ := p["email"].(string); !strings.Contains(email, "@") {
			t.Fatalf("email=%q", email)
		}
	})

	t.Run("bốn vai hợp lệ", func(t *testing.T) {
		for _, role := range []string{"ADMIN", "TEACHER", "TA", "STUDENT"} {
			rc, out, errOut := runTok(devEnv(), "--role", role)
			if rc != 0 {
				t.Fatalf("vai %s: rc=%d stderr=%q", role, rc, errOut)
			}
			if p := payloadOf(t, out); p["role"] != role {
				t.Fatalf("vai %s: claim role=%v", role, p["role"])
			}
		}
	})

	t.Run("--ttl đổi hạn", func(t *testing.T) {
		for ttl, want := range map[string]float64{"10m": 600, "1m": 60, "1h": 3600, "-1m": -60} {
			rc, out, errOut := runTok(devEnv(), "--role", "STUDENT", "--ttl", ttl)
			if rc != 0 {
				t.Fatalf("--ttl %s: rc=%d stderr=%q", ttl, rc, errOut)
			}
			p := payloadOf(t, out)
			if d := p["exp"].(float64) - p["iat"].(float64); d != want {
				t.Fatalf("--ttl %s: exp-iat=%v, muốn %v", ttl, d, want)
			}
		}
	})

	t.Run("bỏ --sub sinh uuid v7 ngẫu nhiên", func(t *testing.T) {
		v7 := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
		seen := map[string]bool{}
		for range 3 {
			rc, out, errOut := runTok(devEnv(), "--role", "TEACHER")
			if rc != 0 {
				t.Fatalf("rc=%d stderr=%q", rc, errOut)
			}
			p := payloadOf(t, out)
			sub, _ := p["sub"].(string)
			if !v7.MatchString(sub) {
				t.Fatalf("sub=%q, muốn uuid v7", sub)
			}
			if seen[sub] {
				t.Fatalf("sub lặp lại: %q", sub)
			}
			seen[sub] = true
			if jti, _ := p["jti"].(string); len(jti) != 22 {
				t.Fatalf("jti=%q dài %d, muốn 22", jti, len(jti))
			}
		}
	})

	t.Run("--email và --sub vào claim", func(t *testing.T) {
		rc, out, errOut := runTok(devEnv(), "--role", "TA", "--sub", tokSub, "--email", "qc-tc52@example.test")
		if rc != 0 {
			t.Fatalf("rc=%d stderr=%q", rc, errOut)
		}
		p := payloadOf(t, out)
		if p["email"] != "qc-tc52@example.test" || p["sub"] != tokSub {
			t.Fatalf("claim=%v", p)
		}
	})

	// Nhánh lỗi: rc=1, stdout rỗng, không nơi nào in token, thông báo nêu đúng cờ/biến.
	errTests := []struct {
		name    string
		env     map[string]string
		args    []string
		mention string
	}{
		{"APP_ENV=production", map[string]string{"APP_ENV": "production", "JWT_SECRET_KEY": tokSecret},
			[]string{"--role", "ADMIN"}, "APP_ENV"},
		{"thiếu JWT_SECRET_KEY", map[string]string{}, []string{"--role", "ADMIN"}, "JWT_SECRET_KEY"},
		{"--role ROOT", devEnv(), []string{"--role", "ROOT"}, "--role"},
		{"--role chữ thường", devEnv(), []string{"--role", "student"}, "--role"},
		{"thiếu --role", devEnv(), nil, "--role"},
		{"--ttl abc", devEnv(), []string{"--role", "ADMIN", "--ttl", "abc"}, "--ttl"},
		{"--ttl rỗng", devEnv(), []string{"--role", "ADMIN", "--ttl", ""}, "--ttl"},
		{"--sub không phải uuid", devEnv(), []string{"--role", "ADMIN", "--sub", "khong-phai-uuid"}, "--sub"},
		{"--sub thiếu ký tự", devEnv(), []string{"--role", "ADMIN", "--sub", "00000000-0000-7000-8000-00000000000"}, "--sub"},
		{"cờ lạ", devEnv(), []string{"--role", "ADMIN", "--khong-co"}, ""},
	}
	for _, tt := range errTests {
		t.Run(tt.name, func(t *testing.T) {
			rc, out, errOut := runTok(tt.env, tt.args...)
			if rc != 1 {
				t.Fatalf("rc=%d, muốn 1", rc)
			}
			if out != "" {
				t.Fatalf("stdout phải rỗng, có %q", out)
			}
			for _, line := range strings.Split(strings.TrimRight(errOut, "\n"), "\n") {
				if jwtShape.MatchString(line) {
					t.Fatalf("stderr có dòng dạng token: %q", line)
				}
			}
			if strings.TrimSpace(errOut) == "" {
				t.Fatal("không có thông báo lỗi")
			}
			if tt.mention != "" && !strings.Contains(errOut, tt.mention) {
				t.Fatalf("thông báo %q không nêu %q", errOut, tt.mention)
			}
			if strings.Contains(errOut, tokSecret) {
				t.Fatal("thông báo lộ giá trị JWT_SECRET_KEY")
			}
		})
	}
}
