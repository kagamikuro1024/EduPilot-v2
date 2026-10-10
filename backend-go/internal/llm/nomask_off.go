//go:build !testroutes

package llm

import "context"

// noMaskFrom: bản dựng thường KHÔNG có đường bỏ che theo ctx — chỉ Options.NoMask (test / ping).
func noMaskFrom(context.Context) bool { return false }
