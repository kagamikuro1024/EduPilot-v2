package main

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/platform/config"
)

// TestStartupRequiresMasker — US-P3-03 AC7: dựng cổng LLM mà không có Masker là lỗi cấu hình (tiến trình không khởi động), không phải chạy không che.
func TestStartupRequiresMasker(t *testing.T) {
	rt, err := llmrt.NewWithMasker(context.Background(), config.Config{}, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if err == nil || rt != nil {
		t.Fatalf("thiếu Masker phải trả lỗi cấu hình, nhận rt=%v err=%v", rt, err)
	}
}
