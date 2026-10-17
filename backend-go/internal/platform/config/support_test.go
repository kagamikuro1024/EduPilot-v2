package config_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/config"
)

// US-P4-07 AC6: SUPPORT_RESOURCES_VI ≤ 300 ký tự; dài hơn → gateway không khởi động, lỗi nêu tên biến; rỗng → câu mặc định (cấu hình rỗng).
func TestSupportResourcesConfigLimit(t *testing.T) {
	t.Parallel()
	env := full()
	env["SUPPORT_RESOURCES_VI"] = strings.Repeat("ê", 300)
	c, err := config.Load(getenv(env), config.Gateway)
	require.NoError(t, err)
	require.Len(t, []rune(c.SupportResourcesVI), 300, "300 ký tự (không phải byte) hợp lệ")
	env["SUPPORT_RESOURCES_VI"] = strings.Repeat("ê", 301)
	_, err = config.Load(getenv(env), config.Gateway)
	require.ErrorContains(t, err, "SUPPORT_RESOURCES_VI")
	delete(env, "SUPPORT_RESOURCES_VI")
	c, err = config.Load(getenv(env), config.Gateway)
	require.NoError(t, err)
	require.Empty(t, c.SupportResourcesVI)
}
