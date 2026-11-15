package quiz_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/quiz"
)

func sp(s string) *string { return &s }

// TestGradeShort — US-P9-01 AC2: chuẩn hoá giữ dấu tiếng Việt; ≥ 30 ca.
func TestGradeShort(t *testing.T) {
	t.Parallel()
	pts := decimal.RequireFromString("1")
	nfd := "Ha\u0300 No\u0323\u0302i" // "Hà Nội" dạng tổ hợp (NFD)
	cases := []struct {
		name string
		key  []string
		ans  *string
		want bool
	}{
		{"khớp đúng", []string{"Hà Nội"}, sp("Hà Nội"), true},
		{"hoa thường", []string{"Hà Nội"}, sp("hà nội"), true},
		{"HOA", []string{"Hà Nội"}, sp("HÀ NỘI"), true},
		{"khoảng trắng đầu cuối", []string{"Hà Nội"}, sp("  Hà Nội  "), true},
		{"khoảng trắng kép giữa", []string{"Hà Nội"}, sp("Hà   Nội"), true},
		{"tab và xuống dòng", []string{"Hà Nội"}, sp("Hà\t\nNội"), true},
		{"dấu chấm cuối", []string{"Hà Nội"}, sp("Hà Nội."), true},
		{"nhiều dấu câu cuối", []string{"Hà Nội"}, sp("Hà Nội!?."), true},
		{"dấu phẩy cuối", []string{"Hà Nội"}, sp("Hà Nội,"), true},
		{"NFD khớp NFC", []string{"Hà Nội"}, &nfd, true},
		{"NFC khớp NFD key", []string{nfd}, sp("Hà Nội"), true},
		{"không dấu KHÔNG khớp", []string{"Hà Nội"}, sp("ha noi"), false},
		{"thiếu dấu một chữ", []string{"Hà Nội"}, sp("Ha Nội"), false},
		{"sai dấu", []string{"Hà Nội"}, sp("Hả Nội"), false},
		{"nhiều đáp án chấp nhận: thứ nhất", []string{"HTTP", "Hypertext Transfer Protocol"}, sp("http"), true},
		{"nhiều đáp án chấp nhận: thứ hai", []string{"HTTP", "Hypertext Transfer Protocol"}, sp("hypertext transfer protocol."), true},
		{"không thuộc đáp án", []string{"HTTP", "HTTPS"}, sp("ftp"), false},
		{"trống nil", []string{"a"}, nil, false},
		{"trống chuỗi rỗng", []string{"a"}, sp(""), false},
		{"chỉ khoảng trắng", []string{"a"}, sp("   "), false},
		{"chỉ dấu câu", []string{"a"}, sp("..."), false},
		{"key có dấu chấm cuối", []string{"TCP."}, sp("tcp"), true},
		{"key có khoảng trắng thừa", []string{"  TCP  /  IP "}, sp("tcp / ip"), true},
		{"dấu câu GIỮA được giữ", []string{"TCP/IP"}, sp("TCP IP"), false},
		{"chữ số", []string{"256"}, sp("256."), true},
		{"chữ số khác", []string{"256"}, sp("255"), false},
		{"khoảng trắng không ngắt", []string{"Hà Nội"}, sp("Hà\u00a0Nội"), true},
		{"thừa chữ", []string{"Hà Nội"}, sp("Thủ đô Hà Nội"), false},
		{"thiếu chữ", []string{"Hà Nội"}, sp("Hà"), false},
		{"ký tự đặc biệt cuối", []string{"C++"}, sp("c++"), true}, // '+' là ký hiệu (Sm), không phải dấu câu: giữ
		{"đáp án rỗng bị bỏ qua, còn đáp án thật", []string{"", "SQL"}, sp("sql"), true},
		{"emoji khác", []string{"Hà Nội"}, sp("Hà Nội 🙂"), false},
	}
	require.GreaterOrEqual(t, len(cases), 30)
	for _, c := range cases {
		got, err := quiz.Grade(quiz.Short, pts, quiz.Key{Accepted: c.key}, answerText(c.ans), quiz.Partial)
		require.NoError(t, err, c.name)
		want := decimal.Zero
		if c.want {
			want = pts
		}
		require.True(t, want.Equal(got), "%s: want %s got %s", c.name, want, got)
	}
}

func answerText(s *string) *quiz.Answer {
	if s == nil {
		return nil
	}
	return &quiz.Answer{Text: s}
}

func TestGradeShortBadKey(t *testing.T) {
	t.Parallel()
	for _, acc := range [][]string{nil, {}, {""}, {"  "}, {"..."}} {
		_, err := quiz.Grade(quiz.Short, decimal.NewFromInt(1), quiz.Key{Accepted: acc}, answerText(sp("x")), quiz.Partial)
		require.ErrorIs(t, err, quiz.ErrBadKey, "%q", acc)
	}
}

func TestNormalizeShort(t *testing.T) {
	t.Parallel()
	require.Equal(t, "hà nội", quiz.NormalizeShort("  Hà   Nội. "))
	require.Equal(t, "", quiz.NormalizeShort(" ?! "))
}

// TestNoLLMImport — AC3: `internal/quiz` chấm bằng code thuần, KHÔNG import `internal/llm` (luật 5).
func TestNoLLMImport(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		af, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
		require.NoError(t, err)
		for _, im := range af.Imports {
			path, _ := strconv.Unquote(im.Path.Value)
			require.NotContains(t, path, "internal/llm", f)
		}
	}
}
