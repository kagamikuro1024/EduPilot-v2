// Package platform: cấu hình env và log dùng chung.
package platform

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Config là cấu hình gateway đọc từ env. P0 chỉ kiểm biến có mặt, chưa kết nối.
type Config struct {
	DatabaseURL string
	RedisURL    string
}

// ErrMissingEnv liệt kê mọi biến bắt buộc bị thiếu (chỉ tên, không giá trị).
type ErrMissingEnv struct{ Names []string }

func (e *ErrMissingEnv) Error() string {
	return "thiếu biến môi trường bắt buộc: " + strings.Join(e.Names, ", ")
}

// LoadConfig đọc env qua getenv; thiếu bất kỳ biến nào → *ErrMissingEnv chứa đủ tên.
func LoadConfig(getenv func(string) string) (Config, error) {
	var missing []string
	need := func(name string) string {
		v := getenv(name)
		if v == "" {
			missing = append(missing, name)
		}
		return v
	}
	c := Config{DatabaseURL: need("DATABASE_URL"), RedisURL: need("REDIS_URL")}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("load config: %w", &ErrMissingEnv{Names: missing})
	}
	return c, nil
}

// NewLogger trả slog JSON ra stdout.
func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
