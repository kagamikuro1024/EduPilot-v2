package main

import (
	"io"

	"github.com/edupilot/backend-go/internal/auth"
)

// runToken chạy `gateway token --role R [--sub UUID] [--email E] [--ttl D]` (FR-44) — công cụ dev in một JWT.
// Logic nằm ở internal/auth để test được độc lập với phần còn lại của cmd.
func runToken(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	return auth.RunTokenCommand(args, getenv, stdout, stderr)
}
