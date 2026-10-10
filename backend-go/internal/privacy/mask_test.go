package privacy

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newMasker(t testing.TB) (*Masker, *bytes.Buffer) {
	t.Helper()
	d, _ := newDetector(map[uuid.UUID][]Member{
		courseA: {{Name: "Nguyễn Văn An", Code: "ZZ99887766"}, {Name: "Lê Thị Bình"}, {Name: "Trần Quốc Cường"}},
	})
	var logs bytes.Buffer
	return &Masker{Detector: d, Log: slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))}, &logs
}

func mask1(t testing.TB, m *Masker, s Session, text string) string {
	t.Helper()
	out, _, err := m.Mask(t.Context(), courseA, s, []string{text})
	require.NoError(t, err)
	return out[0]
}

func TestMaskStablePerSession(t *testing.T) {
	t.Parallel()
	m, _ := newMasker(t)
	s := NewSession("sess-1")
	require.Equal(t, "Em [[SV_1]] hỏi", mask1(t, m, s, "Em Nguyễn Văn An hỏi"))
	require.Equal(t, "[[SV_1]] / [[SV_2]]", mask1(t, m, s, "nguyen van an / Lê Thị Bình"), "khác hoa-thường / dấu vẫn cùng placeholder; thực thể khác tăng số")
	require.Equal(t, "[[SV_1]] và [[SV_1]]", mask1(t, m, s, "An Nguyễn Văn và NGUYỄN VĂN AN"), "dạng đảo cùng thực thể")
	require.Equal(t, "[[SV_3]]", mask1(t, m, s, "Trần Quốc Cường"), "số không đổi giữa các lượt, tăng theo thứ tự xuất hiện")
	out, n, err := m.Mask(t.Context(), courseA, s, []string{"Lê Thị Bình, Trần Quốc Cường", "Nguyễn Văn An"})
	require.NoError(t, err)
	require.Equal(t, []string{"[[SV_2]], [[SV_3]]", "[[SV_1]]"}, out)
	require.Equal(t, 3, n)
}

func TestMaskPlaceholderKinds(t *testing.T) {
	t.Parallel()
	m, _ := newMasker(t)
	s := NewSession("sess-k")
	got := mask1(t, m, s, "mssv 20201234, mail a@b.vn, tel 0912345678, cccd 001203012345, và zz99887766")
	require.Equal(t, "mssv [[MSSV_1]], mail [[EMAIL_1]], tel [[SDT_1]], cccd [[CCCD_1]], và [[MSSV_2]]", got)
	// cùng số điện thoại viết kiểu khác
	require.Equal(t, "[[SDT_1]] [[SDT_1]]", mask1(t, m, s, "+84 912 345 678 0912.345.678"))
	require.Equal(t, "[[EMAIL_1]]", mask1(t, m, s, "A@B.VN"))
}

func TestMaskIdentityWhenClean(t *testing.T) {
	t.Parallel()
	m, _ := newMasker(t)
	in := []string{"Cổng 8080 mở chưa ạ?", "Hoa Kỳ"}
	out, n, err := m.Mask(t.Context(), courseA, NewSession("s"), in)
	require.NoError(t, err)
	require.Equal(t, in, out)
	require.Zero(t, n)
}

func TestMaskSelfIdentity(t *testing.T) {
	t.Parallel()
	m, _ := newMasker(t)
	// sinh viên A (trong roster) gõ tên và MSSV của chính mình: bị che như mọi người, không có ngoại lệ chủ phiên
	got := mask1(t, m, NewSession("s"), "Em là Nguyễn Văn An, MSSV 20201234")
	require.NotContains(t, got, "An")
	require.NotContains(t, got, "20201234")
}

func TestUnmaskTolerantForms(t *testing.T) {
	t.Parallel()
	m, _ := newMasker(t)
	s := NewSession("s")
	mask1(t, m, s, "nguyen van an hỏi về 20201234 và lê thị bình")
	for in, want := range map[string]string{
		"Chào [[SV_1]]!":       "Chào nguyen van an!",
		"Chào [[ SV_1 ]]!":     "Chào nguyen van an!",
		"Chào [[sv_1]]!":       "Chào nguyen van an!",
		"[[Sv_2]] và [[SV_1]]": "lê thị bình và nguyen van an",
		"MSSV [[mssv_1]]":      "MSSV 20201234",
		"[[SV_01]]":            "nguyen van an",
	} {
		require.Equal(t, want, m.Unmask(t.Context(), s, in), in)
	}
}

// TestMappingNotSharedAcrossSessions — AC13: placeholder của phiên khác coi là sót.
func TestMappingNotSharedAcrossSessions(t *testing.T) {
	t.Parallel()
	m, _ := newMasker(t)
	s1, s2 := NewSession("s1"), NewSession("s2")
	mask1(t, m, s1, "Nguyễn Văn An")
	require.Equal(t, "Chào bạn", m.Unmask(t.Context(), s2, "Chào [[SV_1]]"))
	require.Equal(t, "Chào Nguyễn Văn An", m.Unmask(t.Context(), s1, "Chào [[SV_1]]"))
}

func TestScannerUnknownPlaceholder(t *testing.T) {
	t.Parallel()
	m, logs := newMasker(t)
	s := NewSession("s")
	mask1(t, m, s, "Nguyễn Văn An")
	got := m.Unmask(t.Context(), s, "Chào [[SV_9]], [[SV_1]] và [[EMAIL_3]] và [[CCCD]] ạ")
	require.Equal(t, "Chào bạn, Nguyễn Văn An và bạn và bạn ạ", got)
	require.Equal(t, 1, strings.Count(logs.String(), "placeholder sót"), "MỘT dòng warn")
	require.Contains(t, logs.String(), `"count":3`)
	require.NotContains(t, logs.String(), "Nguyễn")
}

func TestScannerExpiredMapping(t *testing.T) {
	t.Parallel()
	m, _ := newMasker(t)
	require.Equal(t, "Chào bạn", m.Unmask(t.Context(), NewSession("hết-hạn"), "Chào [[ SV_1 ]]"))
}

var anyPlaceholder = regexp.MustCompile(`(?i)\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)(_\d*)?`)

func TestScannerUnclosedAtEnd(t *testing.T) {
	t.Parallel()
	m, _ := newMasker(t)
	s := NewSession("s")
	mask1(t, m, s, "Nguyễn Văn An")
	for _, in := range []string{"Chào [[SV", "Chào [[SV_", "Chào [[SV_1", "Chào [[SV_1]", "Chào [[ mssv", "Chào [[EMAIL_", "x [[SV]] y", "[[SV [[SV_1]]"} {
		got := m.Unmask(t.Context(), s, in)
		require.False(t, anyPlaceholder.MatchString(got), "%q → %q", in, got)
	}
	require.Equal(t, "Chào bạn", m.Unmask(t.Context(), s, "Chào [[SV_"))
	require.Equal(t, "bạn Nguyễn Văn An", m.Unmask(t.Context(), s, "[[SV [[SV_1]]"))
}

func TestScannerLeavesBashDoubleBracket(t *testing.T) {
	t.Parallel()
	m, _ := newMasker(t)
	s := NewSession("s")
	for _, in := range []string{`if [[ -f "$f" ]]; then echo ok; fi`, `a[[i]] = b[[j]]`, `[[`, `[`, `[[ x`, `]] [[`, `[[[`, `x[[ ]]`} {
		require.Equal(t, in, m.Unmask(t.Context(), s, in), in)
	}
}

// streamCases phủ: 0–5 placeholder, hai placeholder liền nhau, đầu / cuối chuỗi, `[` và `[[` thường, tiếng Việt có dấu và emoji.
var streamCases = []string{
	"Xin chào bạn",
	"[[SV_1]]",
	"Chào [[SV_1]], điểm của bạn tốt",
	"[[SV_1]][[SV_2]]",
	"[[SV_1]] và [[SV_2]] và [[MSSV_1]]",
	"Đầu [[SV_1]]",
	"[[SV_2]] cuối",
	"😀[[SV_1]]😀 ạ",
	`if [[ -f "$f" ]]; then a[[i]]`,
	"[[ SV_1 ]] x [[sv_2]]",
	"Tiếng Việt: Nguyễn [[SV_1]] đã [ ] nộp [[ ]]",
	"[[SV_9]] bịa [[EMAIL_1]] [[SDT_1]] [[CCCD_1]]",
	"[[[SV_1]]]",
	"a [[ SV_1  ]]b [ [SV_1]] c",
}

func init() {
	// thêm chuỗi dài ≤ 60 rune ghép từ các mảnh để số tổ hợp cắt đạt ≥ 50.000 (mỗi chuỗi 60 rune ≈ 1.900 cách cắt hai điểm)
	frags := []string{"[[SV_1]]", "😀", " nộp bài ", "[[", "]]", "[", "ạ", "[[ SV_2 ]]", "ế", "a[[i]]", "[[MSSV_1]]", "\n", " [[ -f x ]] ", "[[SV_9]]", "[[EMAIL_1]]", "[[S"}
	for i := range 40 {
		var b strings.Builder
		for k := 0; utf8.RuneCountInString(b.String()) < 52; k++ {
			b.WriteString(frags[(i*7+k*5+k*k)%len(frags)])
		}
		r := []rune(b.String())
		streamCases = append(streamCases, string(r[:min(len(r), 60)]))
	}
}

func newStreamFixture(t testing.TB) (*Masker, Session) {
	t.Helper()
	m, _ := newMasker(t)
	s := NewSession("stream")
	mask1(t, m, s, "Nguyễn Văn An, Lê Thị Bình, mssv 20201234, a@b.vn, 0912345678, 001203012345")
	return m, s
}

// TestUnmaskStream — AC9: cắt token ở MỌI vị trí, ra đúng Unmask(toàn bộ) và không bao giờ phát nửa placeholder.
func TestUnmaskStream(t *testing.T) {
	t.Parallel()
	m, s := newStreamFixture(t)
	combos := 0
	run := func(text string, cuts []int) {
		runes := []rune(text)
		want := m.Unmask(t.Context(), s, text)
		u := m.NewStreamUnmasker(t.Context(), s)
		var got strings.Builder
		prev := 0
		for _, c := range append(cuts, len(runes)) {
			got.WriteString(u.Write(string(runes[prev:c])))
			prev = c
			require.True(t, strings.HasPrefix(want, got.String()), "phát trước hạn: %q (cắt %v) → %q, đích %q", text, cuts, got.String(), want)
		}
		got.WriteString(u.Flush())
		require.Equal(t, want, got.String(), "%q cắt %v", text, cuts)
		combos++
	}
	for _, text := range streamCases {
		n := utf8.RuneCountInString(text)
		require.LessOrEqual(t, n, 60, text)
		for i := 0; i <= n; i++ {
			run(text, []int{i})
			for j := i; j <= n; j++ {
				run(text, []int{i, j})
			}
		}
		every := make([]int, 0, n)
		for i := 1; i < n; i++ {
			every = append(every, i)
		}
		run(text, every) // từng rune
	}
	t.Logf("số tổ hợp đã chạy: %d", combos)
	require.GreaterOrEqual(t, combos, 50000)
}

func TestUnmaskStreamNoDelayWithoutBracket(t *testing.T) {
	t.Parallel()
	m, s := newStreamFixture(t)
	u := m.NewStreamUnmasker(t.Context(), s)
	require.Equal(t, "Xin chào Việt Nam 😀", u.Write("Xin chào Việt Nam 😀"))
	require.Equal(t, "a", u.Write("a"))
	require.Equal(t, "x ", u.Write("x [")[:2]) // phát ngay phần trước `[`
	require.Equal(t, "", u.Flush()[:0])
}

func TestUnmaskStreamBufferCap(t *testing.T) {
	t.Parallel()
	m, s := newStreamFixture(t)
	u := m.NewStreamUnmasker(t.Context(), s)
	// `[[` + khoảng trắng dài: vẫn là tiền tố hợp lệ nhưng không giữ quá 32 rune
	in := "[[" + strings.Repeat(" ", 100) + "xong"
	var out strings.Builder
	maxHeld := 0
	for _, r := range in {
		out.WriteString(u.Write(string(r)))
		maxHeld = max(maxHeld, len(u.held))
	}
	out.WriteString(u.Flush())
	require.Equal(t, in, out.String())
	require.LessOrEqual(t, maxHeld, 32)
}

func TestMaskRedisDownFallsBackInMemory(t *testing.T) {
	t.Parallel()
	m, logs := newMasker(t)
	m.Redis = deadRedis(t)
	s := NewSession("sess-down")
	got := mask1(t, m, s, "Nguyễn Văn An, 20201234")
	require.Equal(t, "[[SV_1]], [[MSSV_1]]", got, "payload vẫn đã che")
	require.Equal(t, "Chào Nguyễn Văn An", m.Unmask(t.Context(), s, "Chào [[SV_1]]"))
	require.Contains(t, logs.String(), `"level":"WARN"`)
	require.NotContains(t, logs.String(), "Nguyễn")
	require.NotContains(t, logs.String(), "20201234")
}

type panicRoster struct{}

func (panicRoster) Members(context.Context, uuid.UUID) ([]Member, error) { panic("boom") }

func TestMaskFailsClosed(t *testing.T) {
	t.Parallel()
	// nạp roster lỗi
	d, src := newDetector(nil)
	src.err = fmt.Errorf("db down")
	m := &Masker{Detector: d}
	out, _, err := m.Mask(t.Context(), courseA, NewSession("s"), []string{"Nguyễn Văn An"})
	require.ErrorIs(t, err, ErrMaskFailed)
	require.Nil(t, out)
	// panic bắt được
	m = &Masker{Detector: &Detector{Roster: &Roster{Src: panicRoster{}}}}
	out, _, err = m.Mask(t.Context(), courseA, NewSession("s"), []string{"x"})
	require.ErrorIs(t, err, ErrMaskFailed)
	require.Nil(t, out)
	// quá hạn: văn bản khổng lồ không xong trong MaskTimeout
	d2, _ := newDetector(nil)
	m = &Masker{Detector: d2}
	big := make([]string, 4000)
	for i := range big {
		big[i] = strings.Repeat("abc 20201234 ", 400)
	}
	start := time.Now()
	_, _, err = m.Mask(t.Context(), uuid.Nil, NewSession("s"), big)
	require.ErrorIs(t, err, ErrMaskFailed)
	require.Less(t, time.Since(start), 5*MaskTimeout+time.Second)
}

func TestSessionlessMaskUnmaskInRequestScope(t *testing.T) {
	t.Parallel()
	m, _ := newMasker(t)
	s := NewSession("")
	require.Equal(t, "[[SV_1]]", mask1(t, m, s, "Lê Thị Bình"))
	require.Equal(t, "Lê Thị Bình", m.Unmask(t.Context(), s, "[[SV_1]]"))
}

// BenchmarkMask4k — AC14: tin nhắn 4.000 ký tự, từ điển 60 tên, Detect + Mask ≤ 5 ms.
func BenchmarkMask4k(b *testing.B) {
	var members []Member
	for i := range 60 {
		members = append(members, Member{Name: fmt.Sprintf("Nguyễn Văn Ten%c%c", 'a'+i%26, 'a'+i/26), Code: fmt.Sprintf("ZZ9988%04d", i)})
	}
	d, _ := newDetector(map[uuid.UUID][]Member{courseA: members})
	m := &Masker{Detector: d}
	text := strings.Repeat("Em hỏi về chương 3 và cổng 8080, nguyen van tenaa gửi mail a@b.vn lúc 14:30. ", 50)[:4000]
	b.ResetTimer()
	for range b.N {
		if _, _, err := m.Mask(context.Background(), courseA, NewSession("b"), []string{text}); err != nil {
			b.Fatal(err)
		}
	}
}
