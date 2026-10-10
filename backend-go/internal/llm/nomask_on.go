//go:build testroutes

package llm

import "context"

type noMaskKey struct{}

// WithNoMask chỉ có ở bản dựng `testroutes`: cho route thử `_test/llm/chat` cũ (không gắn lớp) chạy không che.
// Bản dựng thường không có ký hiệu này (kiểm bằng `go tool nm`).
func WithNoMask(ctx context.Context) context.Context {
	return context.WithValue(ctx, noMaskKey{}, true)
}

func noMaskFrom(ctx context.Context) bool { v, _ := ctx.Value(noMaskKey{}).(bool); return v }
