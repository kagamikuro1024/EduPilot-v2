package llmconfig_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// DEF-R3-04 (US-P1-01 AC8): `DecryptKey(` — hàm duy nhất trả khoá API dạng rõ — chỉ được gọi trong internal/llm và internal/llmconfig.
func TestDecryptKeyOnlyInLLMPackages(t *testing.T) {
	root := filepath.Join("..", "..")
	var bad []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		slash := filepath.ToSlash(p)
		if strings.Contains(slash, "/internal/llm/") || strings.Contains(slash, "/internal/llmconfig/") {
			return nil
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		if strings.Contains(string(b), "DecryptKey(") {
			bad = append(bad, slash)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) > 0 {
		t.Fatalf("DecryptKey( ngoài internal/llm, internal/llmconfig: %v", bad)
	}
}
