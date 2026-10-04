package today

import goredis "github.com/redis/go-redis/v9"

// appredisNil là lỗi "không có khoá" của Redis (không phải sự cố).
func appredisNil() error { return goredis.Nil }
