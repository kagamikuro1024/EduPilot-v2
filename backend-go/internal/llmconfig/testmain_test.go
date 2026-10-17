package llmconfig_test

import (
	"testing"

	"github.com/edupilot/backend-go/internal/testutil"
)

// TestMain xoá database dùng chung của gói (testutil.SharedPostgresURL) khi tiến trình test kết thúc (BUG-S7-02).
func TestMain(m *testing.M) { testutil.Main(m) }
