package auth

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// US-P2-12 AC8: mật khẩu mặc định của seed (`.env.example`) phải qua CHÍNH chính sách mật khẩu, với mọi email mẫu.
func TestSeedDefaultPasswordPassesPolicy(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("../../../.env.example")
	require.NoError(t, err)
	var pw string
	for _, l := range strings.Split(string(raw), "\n") {
		if v, ok := strings.CutPrefix(l, "SEED_DEFAULT_PASSWORD="); ok {
			pw = strings.TrimSpace(v)
		}
	}
	require.NotEmpty(t, pw, "SEED_DEFAULT_PASSWORD phải có trong .env.example")
	for _, email := range []string{"admin@edupilot.local", "teacher@edupilot.local", "ta@edupilot.local", "sv.gioi@edupilot.local", "sv.kha@edupilot.local", "sv.nguyco@edupilot.local", "sv.moi@edupilot.local", "sv31@edupilot.local"} {
		require.Empty(t, ValidatePasswordPolicy(pw, email), email)
	}
}
