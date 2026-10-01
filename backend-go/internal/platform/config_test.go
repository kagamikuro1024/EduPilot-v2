package platform

import (
	"errors"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		missing []string
	}{
		{"đủ biến", map[string]string{"DATABASE_URL": "pg", "REDIS_URL": "rd"}, nil},
		{"thiếu DATABASE_URL", map[string]string{"REDIS_URL": "rd"}, []string{"DATABASE_URL"}},
		{"thiếu REDIS_URL", map[string]string{"DATABASE_URL": "pg"}, []string{"REDIS_URL"}},
		{"thiếu cả hai", map[string]string{}, []string{"DATABASE_URL", "REDIS_URL"}},
		{"chỉ có khoảng trắng coi như thiếu", map[string]string{"DATABASE_URL": "pg", "REDIS_URL": "  \t"}, []string{"REDIS_URL"}},
		{"rỗng coi như thiếu", map[string]string{"DATABASE_URL": "", "REDIS_URL": "rd"}, []string{"DATABASE_URL"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadConfig(func(k string) string { return tt.env[k] })
			if tt.missing == nil {
				if err != nil || cfg.DatabaseURL != "pg" || cfg.RedisURL != "rd" {
					t.Fatalf("cfg=%+v err=%v", cfg, err)
				}
				return
			}
			var me *ErrMissingEnv
			if !errors.As(err, &me) || !reflect.DeepEqual(me.Names, tt.missing) {
				t.Fatalf("err=%v, want missing %v", err, tt.missing)
			}
		})
	}
}
