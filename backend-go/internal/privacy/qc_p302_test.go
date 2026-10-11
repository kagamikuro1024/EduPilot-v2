package privacy_test

// QC US-P3-02 (hộp đen theo SRS 4.2). Dữ liệu tấn công do QC soạn từ US/SRS, không lấy từ test của dev.
// Chạy: go test -count=1 ./internal/privacy -run TestQC -v   (QC_REDIS=redis://localhost:6380 cho TC-35).

import (
	"context"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/google/uuid"
)

type qcSrc struct{ ms []privacy.Member }

func (s qcSrc) Members(context.Context, uuid.UUID) ([]privacy.Member, error) { return s.ms, nil }

var qcCourse = uuid.New()

func qcDet() *privacy.Detector {
	return &privacy.Detector{Roster: &privacy.Roster{Src: qcSrc{[]privacy.Member{
		{Name: "Vũ Hoàng Giang", Code: "20229001"}, {Name: "Bùi Thanh Khải", Code: "20229002"},
		{Name: "Ngô Ngọc Cẩm", Code: "20229003"}, {Name: "Nguyễn Văn An", Code: "20229005"},
	}}}}
}

func kinds(fs []privacy.Finding) []privacy.Kind {
	var k []privacy.Kind
	for _, f := range fs {
		k = append(k, f.Kind)
	}
	return k
}

func TestQCPositive(t *testing.T) { // TC-01..06, 11..15, 26
	d := qcDet()
	cases := map[string]privacy.Kind{
		"Bài lab 3 của 20221234 nộp chưa?": privacy.KindMSSV, "Mã B21DCAT123 có trong danh sách không?": privacy.KindMSSV,
		"MSSV 123456 là ai": privacy.KindMSSV, "mã số sinh viên 1234567890123 ạ": privacy.KindMSSV, "mssv: abc123xyz": privacy.KindMSSV,
		"nam.nt@edupilot.local": privacy.KindEmail, "abc.xyz+tag@gmail.com": privacy.KindEmail, "sv@cntt.hust.edu.vn": privacy.KindEmail,
		"0912345678": privacy.KindPhone, "+84 912 345 678": privacy.KindPhone, "0912.345.678": privacy.KindPhone, "09 1234 5678": privacy.KindPhone,
		"001203004567": privacy.KindCCCD, "001 203 004 567": privacy.KindCCCD,
		"Bùi Thanh Khải được mấy điểm lab?": privacy.KindName, "bui thanh khai lam bai nay chua": privacy.KindName,
		"BÙI THANH KHẢI": privacy.KindName, "Khải Bùi Thanh nộp bài chưa": privacy.KindName, "Khải Bùi có đi học không": privacy.KindName,
		"Bùi Khải": privacy.KindName,
	}
	for in, want := range cases {
		fs, err := d.Detect(context.Background(), qcCourse, in)
		if err != nil || len(fs) == 0 {
			t.Errorf("LỌT %q → %v %v", in, fs, err)
			continue
		}
		ok := false
		for _, f := range fs {
			ok = ok || f.Kind == want
		}
		if !ok {
			t.Errorf("sai loại %q: muốn %s, có %v", in, want, kinds(fs))
		}
	}
}

func TestQCNegative(t *testing.T) { // TC-07, 16, 17, 24(giảng viên không có trong roster fake)
	d := qcDet()
	for _, in := range []string{"Mở cổng 8080", "Đề thi năm 2022", "IP nội bộ 192.168.1.10", "CVE-2021-44228 là lỗi gì",
		"dãy 1234567890 có ý nghĩa gì", "khoá RSA 2048 bit", "Bài 2022 trong giáo trình", "hash 4f3c2b1a9d8e7f6a5b4c3d2e",
		"Khải", "An", "anh hiểu sai chỗ này", "Hoa Kỳ dùng chuẩn nào", "cần minh chứng cho luận điểm", "MIT công bố bài báo",
		"Mai thi lại được không", "Nguyễn Văn Anh", "Gọi văn phòng khoa 0241234567"} {
		fs, _ := d.Detect(context.Background(), qcCourse, in)
		if len(fs) != 0 {
			t.Errorf("CHẶN NHẦM %q → %v", in, fs)
		}
	}
	if fs, _ := d.Detect(context.Background(), qcCourse, "Nguyễn Văn An"); len(fs) == 0 {
		t.Error("TC-17: 'Nguyễn Văn An' phải bị bắt")
	}
}

func TestQCRedact(t *testing.T) { // TC-26..28
	d := qcDet()
	in := "Bùi Thanh Khải 20229002 bui.khai@edupilot.local 0912345678"
	fs, _ := d.Detect(context.Background(), qcCourse, in)
	out := privacy.Redact(in, fs)
	if out != "[đã ẩn] [đã ẩn] [đã ẩn] [đã ẩn]" {
		t.Errorf("TC-26: %q", out)
	}
	fs2, _ := d.Detect(context.Background(), qcCourse, out)
	if o2 := privacy.Redact(out, fs2); o2 != out || len(fs2) != 0 {
		t.Errorf("TC-27: %q %v", o2, fs2)
	}
	clean := "So sánh AES-GCM với ChaCha20 🙂\n`[[ -f \"$f\" ]]`"
	fs3, _ := d.Detect(context.Background(), qcCourse, clean)
	if privacy.Redact(clean, fs3) != clean || len(fs3) != 0 {
		t.Errorf("TC-28: %v", fs3)
	}
}

func qcMasker(rd *appredis.Client) *privacy.Masker {
	return &privacy.Masker{Detector: qcDet(), Redis: rd, Log: slog.New(slog.NewTextHandler(os.Stderr, nil))}
}

var qcLeft = regexp.MustCompile(`(?i)\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)`)

func TestQCMask(t *testing.T) { // TC-30..34, 38, 39, 53
	m := qcMasker(nil)
	ctx := context.Background()
	s := privacy.NewSession("qc-s1")
	o, n, err := m.Mask(ctx, qcCourse, s, []string{"Bùi Thanh Khải học nhóm với em", "bui thanh khai có nộp bài chưa", "BÙI THANH KHẢI"})
	if err != nil || n != 3 {
		t.Fatalf("mask: n=%d err=%v", n, err)
	}
	for i, x := range o {
		if !strings.Contains(x, "[[SV_1]]") || strings.Contains(strings.ToLower(x), "khải") || strings.Contains(strings.ToLower(x), "khai") {
			t.Errorf("TC-30/32 [%d] %q", i, x)
		}
	}
	o, _, _ = m.Mask(ctx, qcCourse, privacy.NewSession("qc-s2"), []string{"Bùi Thanh Khải 20229002, Ngô Ngọc Cẩm, a@b.vn, 0912345678, 001203004567"})
	for _, w := range []string{"[[SV_1]]", "[[SV_2]]", "[[MSSV_1]]", "[[EMAIL_1]]", "[[SDT_1]]", "[[CCCD_1]]"} {
		if !strings.Contains(o[0], w) {
			t.Errorf("TC-31 thiếu %s trong %q", w, o[0])
		}
	}
	o, n, _ = m.Mask(ctx, qcCourse, privacy.NewSession("qc-s3"), []string{"Thuật toán RSA dựa trên bài toán nào?"})
	if n != 0 || o[0] != "Thuật toán RSA dựa trên bài toán nào?" {
		t.Errorf("TC-33 %q %d", o[0], n)
	}
	o, _, _ = m.Mask(ctx, qcCourse, privacy.NewSession("qc-s4"), []string{"Em là Vũ Hoàng Giang, MSSV 20229001, cho em xem điểm danh"})
	if strings.Contains(o[0], "Giang") || strings.Contains(o[0], "20229001") {
		t.Errorf("TC-34 chủ phiên lọt: %q", o[0])
	}
	// TC-38/39: khôi phục bản gốc lần thấy đầu tiên, dạng lỏng
	s5 := privacy.NewSession("qc-s5")
	_, _, _ = m.Mask(ctx, qcCourse, s5, []string{"bui thanh khai"})
	if got := m.Unmask(ctx, s5, "Chào [[SV_1]] và [[ SV_1 ]] và [[sv_1]]"); got != "Chào bui thanh khai và bui thanh khai và bui thanh khai" {
		t.Errorf("TC-38/39: %q", got)
	}
	// TC-53: không chéo phiên
	if got := m.Unmask(ctx, privacy.NewSession("qc-s6"), "Chào [[SV_1]]"); got != "Chào bạn" {
		t.Errorf("TC-53: %q", got)
	}
}

func TestQCUnmaskEdges(t *testing.T) { // TC-42, 44, 46, 47
	m := qcMasker(nil)
	ctx := context.Background()
	s := privacy.NewSession("qc-e1")
	_, _, _ = m.Mask(ctx, qcCourse, s, []string{"Bùi Thanh Khải"})
	bash := "Dùng `[[ -f \"$f\" ]]` để kiểm tệp, và a[[i]] là chỉ mục lồng"
	if got := m.Unmask(ctx, s, bash); got != bash {
		t.Errorf("TC-42: %q", got)
	}
	if got := m.Unmask(ctx, s, "Chào [[SV_9]] nhé"); got != "Chào bạn nhé" {
		t.Errorf("TC-44: %q", got)
	}
	u := m.NewStreamUnmasker(ctx, s)
	got := u.Write("… gửi cho [[MSSV_") + u.Flush()
	if qcLeft.MatchString(got) || strings.Contains(got, "[[MSSV") {
		t.Errorf("TC-46 lọt phần mở dở: %q", got)
	}
	t.Logf("TC-46 → %q", got)
}

func TestQCStreamCutEverywhere(t *testing.T) { // TC-41 (đối chiếu Unmask(toàn) và 0 khung chứa nửa placeholder)
	m := qcMasker(nil)
	ctx := context.Background()
	s := privacy.NewSession("qc-c1")
	_, _, _ = m.Mask(ctx, qcCourse, s, []string{"Bùi Thanh Khải, 20229002, Ngô Ngọc Cẩm"})
	full := "Xin chào [[SV_1]], MSSV [[MSSV_1]] 🙂 và [[SV_2]][[SV_2]] ở đây [ [[ [[x]] ✓"
	want := m.Unmask(ctx, s, full)
	rs := []rune(full)
	cnt := 0
	cut := func(parts ...string) {
		u := m.NewStreamUnmasker(ctx, s)
		var sb strings.Builder
		for _, p := range parts {
			f := u.Write(p)
			if regexp.MustCompile(`\[\[\s*(SV|MSSV)[^\]]*$|\[\[SV|\[\[MSSV`).MatchString(f) {
				t.Fatalf("khung chứa nửa placeholder: %q", f)
			}
			sb.WriteString(f)
		}
		sb.WriteString(u.Flush())
		cnt++
		if sb.String() != want {
			t.Fatalf("lệch tại %q: %q != %q", parts, sb.String(), want)
		}
	}
	for i := 0; i <= len(rs); i++ {
		cut(string(rs[:i]), string(rs[i:]))
		for j := i; j <= len(rs); j++ {
			cut(string(rs[:i]), string(rs[i:j]), string(rs[j:]))
		}
	}
	var each []string
	for _, r := range rs {
		each = append(each, string(r))
	}
	cut(each...)
	t.Logf("TC-41: %d tổ hợp cắt, đều bằng Unmask(toàn)=%q", cnt, want)
	if !utf8.ValidString(want) {
		t.Error("utf8")
	}
}

func TestQCRedisMapping(t *testing.T) { // TC-35, 45, 37
	url := os.Getenv("QC_REDIS")
	if url == "" {
		t.Skip("QC_REDIS không đặt")
	}
	rd, err := appredis.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	m := qcMasker(rd)
	ctx := context.Background()
	sid := "qc-redis-" + uuid.NewString()
	s := privacy.NewSession(sid)
	defer rd.Del(ctx, privacy.MaskKey(sid))
	_, _, _ = m.Mask(ctx, qcCourse, s, []string{"Bùi Thanh Khải"})
	ttl := rd.TTL(ctx, privacy.MaskKey(sid)).Val()
	if ttl <= 0 || ttl > 24*time.Hour {
		t.Errorf("TC-35 TTL %v", ttl)
	}
	keys := rd.HKeys(ctx, privacy.MaskKey(sid)).Val()
	t.Logf("TC-35 TTL=%v khoá HASH=%d (%v)", ttl, len(keys), keys)
	rd.Del(ctx, privacy.MaskKey(sid)) // TC-45: ánh xạ hết hạn giữa chừng
	if got := m.Unmask(ctx, s, "Chào [[SV_1]]"); got == "Chào Bùi Thanh Khải" {
		t.Logf("TC-45: còn nhớ trong bộ nhớ phiên (Sess giữ bản trong RAM): %q", got)
	} else if got != "Chào bạn" {
		t.Errorf("TC-45: %q", got)
	}
	n0 := len(rd.Keys(ctx, "ep:mask:*").Val())
	_, _, _ = m.Mask(ctx, qcCourse, privacy.NewSession(""), []string{"Bùi Thanh Khải"})
	if n1 := len(rd.Keys(ctx, "ep:mask:*").Val()); n1 != n0 {
		t.Errorf("TC-37: %d → %d", n0, n1)
	}
}

func TestQCLongInput(t *testing.T) { // TC-56, 57
	d := qcDet()
	for _, n := range []int{1000, 4000, 8000, 100000} {
		for _, pat := range []string{"a1", "((("} {
			s := strings.Repeat(pat, n/len(pat))
			t0 := time.Now()
			_, _ = d.Detect(context.Background(), qcCourse, s)
			t.Logf("n=%d %q: %v", n, pat, time.Since(t0))
		}
	}
	long := strings.Repeat("x ", 12500) + "liên hệ 0912345678 nhé"
	if fs, _ := d.Detect(context.Background(), qcCourse, long); len(fs) != 1 || fs[0].Kind != privacy.KindPhone {
		t.Errorf("TC-57: %v", fs)
	}
}

func TestQCNoDoubleCount(t *testing.T) { // TC-08: một khoảng không vừa PHONE vừa CCCD
	d := qcDet()
	for _, in := range []string{"CCCD của tôi 0912345678901 và 0912345678", "0912345678901"} {
		fs, _ := d.Detect(context.Background(), qcCourse, in)
		for i := 1; i < len(fs); i++ {
			if fs[i].Start < fs[i-1].End {
				t.Errorf("TC-08 chồng lấn %q: %v", in, fs)
			}
		}
		t.Logf("TC-08 %q → %v", in, fs)
	}
}
