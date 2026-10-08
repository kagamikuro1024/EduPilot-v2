package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/judge"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

const tok = "0123456789abcdef"

// TestJudgeMemLimitCheck — JUDGE_MEM_LIMIT ≥ P×(512+256)+512 MiB, sai thì lỗi nêu tên biến.
func TestJudgeMemLimitCheck(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string // "" = ok
	}{
		{"P2 2g ok", map[string]string{"JUDGE_CONSUMER": "true", "JUDGE_TOKEN": tok, "JUDGE_PARALLELISM": "2", "JUDGE_MEM_LIMIT": "2g"}, ""},
		{"P2 đúng ngưỡng 2048m ok", map[string]string{"JUDGE_CONSUMER": "true", "JUDGE_TOKEN": tok, "JUDGE_PARALLELISM": "2", "JUDGE_MEM_LIMIT": "2048m"}, ""},
		{"P4 3584m ok", map[string]string{"JUDGE_CONSUMER": "true", "JUDGE_TOKEN": tok, "JUDGE_PARALLELISM": "4", "JUDGE_MEM_LIMIT": "3584m"}, ""},
		{"P2 1g thiếu", map[string]string{"JUDGE_CONSUMER": "true", "JUDGE_TOKEN": tok, "JUDGE_PARALLELISM": "2", "JUDGE_MEM_LIMIT": "1g"}, "JUDGE_MEM_LIMIT"},
		{"P4 2g thiếu", map[string]string{"JUDGE_CONSUMER": "true", "JUDGE_TOKEN": tok, "JUDGE_PARALLELISM": "4", "JUDGE_MEM_LIMIT": "2g"}, "JUDGE_MEM_LIMIT"},
		{"rác", map[string]string{"JUDGE_CONSUMER": "true", "JUDGE_TOKEN": tok, "JUDGE_MEM_LIMIT": "abc"}, "JUDGE_MEM_LIMIT"},
		{"token ngắn", map[string]string{"JUDGE_CONSUMER": "true", "JUDGE_TOKEN": "short"}, "JUDGE_TOKEN"},
		{"không consumer: bỏ qua token và bộ nhớ", map[string]string{"JUDGE_PARALLELISM": "4", "JUDGE_MEM_LIMIT": "1g"}, ""},
		{"P=0", map[string]string{"JUDGE_PARALLELISM": "0"}, "JUDGE_PARALLELISM"},
		{"lease < 2×renew", map[string]string{"JUDGE_LEASE": "15s", "JUDGE_LEASE_RENEW": "10s"}, "JUDGE_LEASE"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := judge.SettingsFromEnv(env(c.env))
			if c.want == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), c.want), err.Error())
		})
	}
}
