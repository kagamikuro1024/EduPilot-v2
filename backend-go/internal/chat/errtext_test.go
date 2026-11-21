package chat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// DEF-R6-U5 (TC-P3-02-50, SRS FEAT-private-chat-pii 4.2): che lỗi (MASK_FAILED) báo «Chưa gửi được tin nhắn. Thử lại.», không phải câu lỗi nhà cung cấp.
func TestErrorTextMaskFailed(t *testing.T) {
	require.Equal(t, "Chưa gửi được tin nhắn. Thử lại.", errorText("MASK_FAILED"))
	require.NotEqual(t, errorText("PROVIDER_ERROR"), errorText("MASK_FAILED"))
}
