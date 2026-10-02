//go:build testroutes

package main

import (
	"strings"
	"testing"
	"time"
)

// TestTestBinary_RefusesProduction: bản dựng có build tag `testroutes` từ chối chạy khi
// APP_ENV=production — thoát 1 ngay, log nêu APP_ENV (03-AC18, SRS 6.3).
func TestTestBinary_RefusesProduction(t *testing.T) {
	t.Parallel()
	env := serveEnv()
	env["APP_ENV"] = "production"
	env["STARTUP_TIMEOUT"] = "2s"

	rc, out, elapsed := runGateway(t, env)
	if rc != 1 {
		t.Fatalf("rc = %d, muốn 1\n%s", rc, out)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("thoát sau %v, muốn ≤ 5 s", elapsed)
	}
	if !strings.Contains(out, "APP_ENV") {
		t.Fatalf("log không nêu APP_ENV: %s", out)
	}

	// Thoát TRƯỚC khi chạm tới phụ thuộc nào: không có dòng `config loaded`, không mở cổng.
	if strings.Contains(out, "config loaded") {
		t.Fatalf("đã khởi động dù APP_ENV=production: %s", out)
	}
}
