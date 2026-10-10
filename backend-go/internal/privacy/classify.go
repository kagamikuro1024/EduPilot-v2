package privacy

import (
	"bufio"
	"context"
	_ "embed" // personal-exemplars.txt
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

//go:embed personal-exemplars.txt
var exemplarsTxt string

// Ngưỡng khởi điểm (SRS 4.1): chỉnh trên tập `dev` của E1, không trên `test`.
const (
	PersonalSimHigh = 0.78
	// PersonalSimLow = 0.55: ≤ ngưỡng này → công khai; vùng giữa không coi là cá nhân nên chỉ cần nhớ ngưỡng cao (hằng thấp dành cho hiệu chỉnh E1).
	// minEmbedChars: văn bản ngắn hơn không nhúng để phân loại.
	minEmbedChars = 20
)

// Channel là kênh mà câu hỏi thuộc về.
type Channel string

// Hai kênh.
const (
	ChannelPrivate Channel = "PRIVATE"
	ChannelPublic  Channel = "PUBLIC"
)

// Reason là lý do kết luận "cá nhân" (không chứa nội dung).
type Reason string

// Các lý do.
const (
	ReasonPIIMSSV   Reason = "PII_MSSV"
	ReasonPIIEmail  Reason = "PII_EMAIL"
	ReasonPIIPhone  Reason = "PII_PHONE"
	ReasonPIICCCD   Reason = "PII_CCCD"
	ReasonPIIName   Reason = "PII_NAME"
	ReasonPattern   Reason = "PERSONAL_PATTERN"
	ReasonSimilarty Reason = "PERSONAL_SIMILARITY"
)

// Result là kết quả phân loại kênh.
type Result struct {
	Channel       Channel
	Personal      bool
	Reasons       []Reason
	UsedEmbedding bool
	Similarity    float64 // cosine lớn nhất với mẫu cá nhân khi đã nhúng; chỉ để hiệu chỉnh, không bao giờ trả cho sinh viên
}

// personalPatterns: câu hỏi cá nhân tiếng Việt, khớp trên văn bản đã bỏ dấu + chữ thường (nên phủ cả có dấu lẫn không dấu).
var (
	rePers0 = regexp.MustCompile(`\bdiem( [a-z]+){0,3} (cua )?(em|minh|toi|tao)\b`)
	rePers1 = regexp.MustCompile(`\b(em|minh|toi) (co )?(da |bi |dang |con )?(vang|nghi|duoc|thieu|bi tru|bi cam|bi canh bao)\b`)
	rePers2 = regexp.MustCompile(`\b(so buoi |buoi )?(vang|nghi|diem danh|chuyen can) (cua )?(em|minh|toi)\b`)
	rePers3 = regexp.MustCompile(`\blich thi (cua )?(em|minh|toi)\b`)
	rePers4 = regexp.MustCompile(`\bdiem cong (cua )?(em|minh|toi)\b`)
	rePers5 = regexp.MustCompile(`\bquy che\b.{0,50}\b(ap|danh) (vao|cho) (em|minh|toi)\b`)
	rePers6 = regexp.MustCompile(`\b(phuc khao|khieu nai)\b.{0,50}\bdiem (cua )?(em|minh|toi)\b`)
	rePers7 = regexp.MustCompile(`\b(bai|diem|ket qua)\b.{0,30}\bcua (em|minh|toi)\b`)
)

// IsPersonalPattern cho biết văn bản khớp mẫu câu cá nhân (dùng cả ở agent để chọn "ý định cá nhân").
func IsPersonalPattern(text string) bool {
	f := auth.Fold(text)
	for _, re := range []*regexp.Regexp{rePers0, rePers1, rePers2, rePers3, rePers4, rePers5, rePers6, rePers7} {
		if re.MatchString(f) {
			return true
		}
	}
	return false
}

// Prototypes là tập vectơ mẫu câu hỏi cá nhân (đã nhúng).
type Prototypes struct{ vecs [][]float32 }

// Exemplars trả các câu mẫu cá nhân (seed/privacy/personal-exemplars.txt, ≥ 40 câu).
func Exemplars() []string {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(exemplarsTxt))
	for sc.Scan() {
		if t := strings.TrimSpace(sc.Text()); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// NewPrototypes nhúng tập mẫu bằng embedMany (người gọi cắm cache `ep:cls:proto:{model}` quanh nó).
func NewPrototypes(ctx context.Context, embedMany func(ctx context.Context, inputs []string) ([][]float32, error)) (*Prototypes, error) {
	ex := Exemplars()
	vecs, err := embedMany(ctx, ex)
	if err != nil {
		return nil, fmt.Errorf("privacy: nhúng mẫu cá nhân: %w", err)
	}
	if len(vecs) != len(ex) {
		return nil, fmt.Errorf("privacy: nhận %d vectơ cho %d mẫu", len(vecs), len(ex))
	}
	return &Prototypes{vecs: vecs}, nil
}

// NewPrototypesFromVectors dựng từ vectơ đã có (cache / test).
func NewPrototypesFromVectors(v [][]float32) *Prototypes { return &Prototypes{vecs: v} }

// Vectors trả các vectơ mẫu (để người gọi ghi cache).
func (p *Prototypes) Vectors() [][]float32 { return p.vecs }

func (p *Prototypes) maxCosine(v []float32) float64 {
	best := -1.0
	for _, q := range p.vecs {
		if c := cosine(v, q); c > best {
			best = c
		}
	}
	return best
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// Classifier phân loại kênh: luật (PII, mẫu câu) rồi — chỉ khi luật chưa quyết và văn bản đủ dài — độ tương đồng embedding.
// KHÔNG có bước LLM sinh chữ (D47 mục 2). Chạy đúng MỘT lần mỗi tin nhắn / yêu cầu; vectơ đã nhúng trả ra cho rag dùng lại.
type Classifier struct {
	Detector *Detector
	Protos   *Prototypes // nil = chỉ luật (hoặc dùng SetPrototypes khi nạp muộn)
	lazy     atomic.Pointer[Prototypes]
	High     float64 // 0 = PersonalSimHigh; ≤ ngưỡng thấp (PersonalSimLow, khởi điểm 0,55) và vùng giữa đều KHÔNG coi là cá nhân
	Log      *slog.Logger
}

// Classify trả kết quả và vectơ đã nhúng (nil nếu chưa nhúng). embed chỉ được gọi khi luật chưa quyết.
func (c *Classifier) Classify(ctx context.Context, courseID uuid.UUID, text string, embed func(context.Context) ([]float32, error)) (Result, []float32, error) {
	res := Result{Channel: ChannelPublic}
	fs, err := c.Detector.Detect(ctx, courseID, text)
	if err != nil {
		return res, nil, fmt.Errorf("privacy: phân loại: %w", err)
	}
	for _, f := range fs {
		res.Reasons = appendReason(res.Reasons, map[Kind]Reason{KindMSSV: ReasonPIIMSSV, KindEmail: ReasonPIIEmail, KindPhone: ReasonPIIPhone, KindCCCD: ReasonPIICCCD, KindName: ReasonPIIName}[f.Kind])
	}
	if IsPersonalPattern(text) {
		res.Reasons = appendReason(res.Reasons, ReasonPattern)
	}
	if len(res.Reasons) > 0 { // luật quyết định rõ → không nhúng
		res.Personal, res.Channel = true, ChannelPrivate
		return res, nil, nil
	}
	if embed == nil || c.protos() == nil || utf8.RuneCountInString(strings.TrimSpace(text)) < minEmbedChars {
		return res, nil, nil
	}
	v, err := embed(ctx)
	if err != nil { // hạ cấp: chỉ luật
		if c.Log != nil {
			c.Log.WarnContext(ctx, "phân loại: nhúng lỗi, chỉ dùng luật")
		}
		return res, nil, nil
	}
	res.UsedEmbedding = true
	res.Similarity = c.protos().maxCosine(v)
	hi := c.High
	if hi == 0 {
		hi = PersonalSimHigh
	}
	if res.Similarity >= hi {
		res.Personal, res.Channel = true, ChannelPrivate
		res.Reasons = appendReason(res.Reasons, ReasonSimilarty)
	}
	return res, v, nil
}

func appendReason(rs []Reason, r Reason) []Reason {
	for _, x := range rs {
		if x == r {
			return rs
		}
	}
	return append(rs, r)
}

// SetPrototypes đặt mẫu cá nhân nạp muộn (gateway khởi động trước khi nhà cung cấp nhúng được cấu hình); an toàn khi đang phân loại.
func (c *Classifier) SetPrototypes(p *Prototypes) { c.lazy.Store(p) }

func (c *Classifier) protos() *Prototypes {
	if c.Protos != nil {
		return c.Protos
	}
	return c.lazy.Load()
}

// HasPrototypes cho biết đã có mẫu câu cá nhân để so nhúng.
func (c *Classifier) HasPrototypes() bool { return c.protos() != nil }
