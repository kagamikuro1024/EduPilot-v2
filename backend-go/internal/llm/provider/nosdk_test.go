package provider_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestDecideNoSDK: gói provider (và cả internal/llm) không kéo SDK Google (D60: một hàm net/http, không SDK).
func TestDecideNoSDK(t *testing.T) {
	t.Parallel()
	out, err := exec.Command("go", "list", "-deps", "github.com/edupilot/backend-go/internal/llm/...").CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	for _, bad := range []string{"google.golang.org/genai", "cloud.google.com/go/ai", "github.com/google/generative-ai-go"} {
		if strings.Contains(string(out), bad) {
			t.Fatalf("internal/llm phụ thuộc %s", bad)
		}
	}
}
