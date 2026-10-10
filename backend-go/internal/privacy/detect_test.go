package privacy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type staticRoster struct {
	byCourse map[uuid.UUID][]Member
	calls    int
	err      error
}

func (s *staticRoster) Members(_ context.Context, id uuid.UUID) ([]Member, error) {
	s.calls++
	return s.byCourse[id], s.err
}

func newDetector(courses map[uuid.UUID][]Member) (*Detector, *staticRoster) {
	src := &staticRoster{byCourse: courses}
	return &Detector{Roster: &Roster{Src: src}}, src
}

var courseA, courseB = uuid.MustParse("00000000-0000-0000-0000-00000000000a"), uuid.MustParse("00000000-0000-0000-0000-00000000000b")

func spans(t *testing.T, text string, fs []Finding) []string {
	t.Helper()
	r := []rune(text)
	var out []string
	for _, f := range fs {
		out = append(out, string(f.Kind)+":"+string(r[f.Start:f.End]))
	}
	return out
}

func TestDetectRegexPositive(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(nil)
	for _, c := range []struct{ text, want string }{
		{"mssv của em 20201234 nhé", "MSSV:20201234"},
		{"MSSV: B21DCCN123", "MSSV:B21DCCN123"},
		{"mã số sinh viên b21dccn123", "MSSV:b21dccn123"},
		{"mssv: ABC12345", "MSSV:ABC12345"},
		{"B21DCCN123 điểm bao nhiêu", "MSSV:B21DCCN123"},
		{"mail em là an.nguyen+k@sv.ptit.edu.vn ạ", "EMAIL:an.nguyen+k@sv.ptit.edu.vn"},
		{"a@b.co", "EMAIL:a@b.co"},
		{"x_y-z@mail-server.example.com", "EMAIL:x_y-z@mail-server.example.com"},
		{"gọi 0912345678 giúp", "PHONE:0912345678"},
		{"sđt +84 912 345 678", "PHONE:+84 912 345 678"},
		{"sđt 0912.345.678", "PHONE:0912.345.678"},
		{"sđt 09 1234 5678", "PHONE:09 1234 5678"},
		{"sđt 0912-345-678", "PHONE:0912-345-678"},
		{"sđt 84912345678", "PHONE:84912345678"},
		{"0387654321", "PHONE:0387654321"},
		{"0587654321", "PHONE:0587654321"},
		{"0787654321", "PHONE:0787654321"},
		{"0887654321", "PHONE:0887654321"},
		{"cccd 001203012345", "CCCD:001203012345"},
		{"cccd 079 203 012 345", "CCCD:079 203 012 345"},
		{"cccd: 079203012345.", "CCCD:079203012345"},
		{"(20201234)", "MSSV:20201234"},
	} {
		fs, err := d.Detect(t.Context(), uuid.Nil, c.text)
		require.NoError(t, err)
		require.Equal(t, []string{c.want}, spans(t, c.text, fs), c.text)
	}
}

func TestDetectRegexNegative(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(nil)
	for _, text := range []string{
		"cổng 8080", "năm 2022", "IP 192.168.1.10", "CVE-2021-44228", "dãy 1234567890", "khoá 2048 bit", "hash a1b2c3d4e5f6a7b8",
		"sha 3f0123456789012a", "phiên bản 3.14.159", "giá 1.500.000đ", "lúc 14:30 ngày 12/10", "mssv của em ạ", "mã số sinh viên student",
		"mssv: hello", "chương 3 mục 4", "port 3306 và 5432", "id 12345678", "123456789012345", "email a@b",
		"[[ -f \"$f\" ]] && echo 1", "a[[i]] = 5",
	} {
		fs, err := d.Detect(t.Context(), uuid.Nil, text)
		require.NoError(t, err)
		require.Empty(t, spans(t, text, fs), text)
	}
}

func TestDetectOverlapAndOrder(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(nil)
	text := "mssv: 20201234, mail 20201234@sv.edu.vn, tel 0912345678"
	fs, err := d.Detect(t.Context(), uuid.Nil, text)
	require.NoError(t, err)
	require.Equal(t, []string{"MSSV:20201234", "EMAIL:20201234@sv.edu.vn", "PHONE:0912345678"}, spans(t, text, fs))
}

func TestDetectRuneOffsetsWithAccents(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(nil)
	text := "Chào thầy 😀 em ở lớp 3, mssv 20201234"
	fs, err := d.Detect(t.Context(), uuid.Nil, text)
	require.NoError(t, err)
	require.Equal(t, []string{"MSSV:20201234"}, spans(t, text, fs))
}

func TestRosterNameVariants(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(map[uuid.UUID][]Member{courseA: {{Name: "Nguyễn Văn An", Code: "B21DCCN001"}}})
	for _, text := range []string{
		"Nguyễn Văn An", "nguyen van an", "NGUYỄN VĂN AN", "An Nguyễn Văn", "An Nguyễn", "Nguyễn An", "nguyễn   văn  an", "Nguyen Van An",
	} {
		fs, err := d.Detect(t.Context(), courseA, "em là "+text+" ạ")
		require.NoError(t, err)
		require.Equal(t, []string{"NAME:" + text}, spans(t, "em là "+text+" ạ", fs), text)
	}
	// dạng NFD (dấu kết hợp) vẫn khớp, khoảng rune tính theo văn bản gốc
	nfd := "Nguye\u0302\u0303n Va\u0306n An"
	fs, err := d.Detect(t.Context(), courseA, nfd)
	require.NoError(t, err)
	require.Len(t, fs, 1)
	require.Equal(t, KindName, fs[0].Kind)
	// mã sinh viên của roster (chữ thường)
	fs, _ = d.Detect(t.Context(), courseA, "mã b21dccn001 là của ai")
	require.Equal(t, []string{"MSSV:b21dccn001"}, spans(t, "mã b21dccn001 là của ai", fs))
}

func TestRosterNoCommonWordFalsePositive(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(map[uuid.UUID][]Member{courseA: {{Name: "Nguyễn Văn An"}, {Name: "Lê Minh"}, {Name: "Trần Hoa"}, {Name: "Phạm Anh"}}})
	for _, text := range []string{"An", "anh", "Hoa Kỳ", "minh chứng", "Minh ơi", "hoa hồng", "Anh Nguyễn Văn", "văn bản", "anh em"} {
		fs, err := d.Detect(t.Context(), courseA, text)
		require.NoError(t, err)
		require.Empty(t, spans(t, text, fs), text)
	}
}

func TestRosterWordBoundary(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(map[uuid.UUID][]Member{courseA: {{Name: "Nguyễn Văn An"}}})
	for _, text := range []string{"Nguyễn Văn Anh", "Nguyễn Văn Ann", "xNguyễn Văn An", "Nguyễn, Văn An", "Nguyễn-Văn An"} {
		fs, err := d.Detect(t.Context(), courseA, text)
		require.NoError(t, err)
		require.Empty(t, spans(t, text, fs), text)
	}
	fs, _ := d.Detect(t.Context(), courseA, "Nguyễn Văn An, Nguyễn Văn Anh")
	require.Equal(t, []string{"NAME:Nguyễn Văn An"}, spans(t, "Nguyễn Văn An, Nguyễn Văn Anh", fs))
}

func TestRosterScopedToCourse(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(map[uuid.UUID][]Member{courseA: {{Name: "Nguyễn Văn An"}}, courseB: {{Name: "Lê Thị Bình", Code: "ZZ99887766"}}})
	fs, _ := d.Detect(t.Context(), courseA, "Lê Thị Bình và zz99887766")
	require.Empty(t, fs, "sinh viên chỉ ở lớp B không bị bắt ở lớp A")
	fs, _ = d.Detect(t.Context(), courseB, "Lê Thị Bình và zz99887766")
	require.Len(t, fs, 2)
	fs, _ = d.Detect(t.Context(), uuid.Nil, "Nguyễn Văn An")
	require.Empty(t, fs, "không có lớp thì chỉ regex")
}

func TestRosterLoadError(t *testing.T) {
	t.Parallel()
	d, src := newDetector(nil)
	src.err = context.DeadlineExceeded
	_, err := d.Detect(t.Context(), courseA, "xin chào")
	require.Error(t, err)
}

func TestRedact(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(map[uuid.UUID][]Member{courseA: {{Name: "Nguyễn Văn An"}}})
	text := "Em Nguyễn Văn An, mssv 20201234, sđt 0912345678 xin hỏi"
	fs, err := d.Detect(t.Context(), courseA, text)
	require.NoError(t, err)
	got := Redact(text, fs)
	require.Equal(t, "Em [đã ẩn], mssv [đã ẩn], sđt [đã ẩn] xin hỏi", got)
	require.NotContains(t, got, "20201234")

	// idempotent
	fs2, _ := d.Detect(t.Context(), courseA, got)
	require.Equal(t, got, Redact(got, fs2))

	// hai khoảng chạm nhau gộp thành một dấu
	require.Equal(t, "a[đã ẩn]z", Redact("a12345z", []Finding{{Start: 1, End: 3}, {Start: 3, End: 6}}))
	// khoảng chồng nhau
	require.Equal(t, "a[đã ẩn]z", Redact("a12345z", []Finding{{Start: 1, End: 4}, {Start: 2, End: 6}}))
}

func TestRedactIdentityWhenClean(t *testing.T) {
	t.Parallel()
	s := "Cổng 8080 có mở không ạ?"
	require.Equal(t, s, Redact(s, nil))
	d, _ := newDetector(nil)
	fs, _ := d.Detect(t.Context(), uuid.Nil, s)
	require.Equal(t, s, Redact(s, fs))
}

// TestDetectLinearTime — AC14: không backtracking mũ; 100.000 ký tự lặp xử lý ≤ 50 ms.
func TestDetectLinearTime(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(map[uuid.UUID][]Member{courseA: {{Name: "Nguyễn Văn An"}}})
	for name, text := range map[string]string{
		"a1":     strings.Repeat("a1", 50000),
		"paren":  strings.Repeat("(", 100000),
		"digits": strings.Repeat("0", 100000),
		"at":     strings.Repeat("a@", 50000),
		"names":  strings.Repeat("Nguyễn ", 14000),
		"mssv":   strings.Repeat("mssv ", 20000),
	} {
		start := time.Now()
		_, err := d.Detect(t.Context(), courseA, text)
		require.NoError(t, err)
		require.Less(t, time.Since(start), 50*time.Millisecond*raceSlowdown, name)
	}
}

func TestDetectWindowedLongText(t *testing.T) {
	t.Parallel()
	d, _ := newDetector(nil)
	text := strings.Repeat("x ", 15000) + "mssv 20201234 " + strings.Repeat("y ", 15000)
	fs, err := d.Detect(t.Context(), uuid.Nil, text)
	require.NoError(t, err)
	require.Equal(t, []string{"MSSV:20201234"}, spans(t, text, fs))
}
