package privacy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/text/unicode/norm"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// RosterTTL là lưới an toàn của cache roster; vô hiệu thật sự đi theo sự kiện outbox.
const RosterTTL = time.Hour

// Member là một sinh viên ACTIVE của lớp. Giảng viên / TA không vào từ điển (Q8).
type Member struct {
	Name string `json:"n"`
	Code string `json:"c,omitempty"` // student_code_snapshot
}

// RosterSource nạp roster một lớp từ DB.
type RosterSource interface {
	Members(ctx context.Context, courseID uuid.UUID) ([]Member, error)
}

// Roster giữ từ điển theo lớp: Redis `ep:roster:{course_id}` (JSON, TTL 1 giờ), trượt thì dựng lại bằng một truy vấn.
// ponytail: dựng chỉ mục (vài chục biến thể tên) mỗi lần gọi thay vì cache trong tiến trình — cache cục bộ sẽ làm chậm vô hiệu theo sự kiện; thêm khi lớp > ~500 sinh viên.
type Roster struct {
	Src   RosterSource
	Redis *appredis.Client // nil = không cache
	Log   *slog.Logger
	// OpTimeout: hạn mỗi lệnh Redis; 0 = redisOpTimeout (30 ms, quá hạn thì dựng lại từ DB). Test đặt dài để kết quả không phụ thuộc tải máy (#19).
	OpTimeout time.Duration
}

func (r *Roster) opTimeout() time.Duration {
	if r.OpTimeout > 0 {
		return r.OpTimeout
	}
	return redisOpTimeout
}

// RosterKey là khoá Redis của một lớp.
func RosterKey(courseID uuid.UUID) string { return appredis.Key("roster", courseID.String()) }

func (r *Roster) members(ctx context.Context, courseID uuid.UUID) ([]Member, error) {
	key := RosterKey(courseID)
	if r.Redis != nil {
		rctx, cancel := context.WithTimeout(ctx, r.opTimeout())
		raw, err := r.Redis.Get(rctx, key).Bytes()
		cancel()
		switch {
		case err == nil:
			var ms []Member
			if json.Unmarshal(raw, &ms) == nil {
				return ms, nil
			}
		case !errors.Is(err, redis.Nil):
			r.warn("roster: không đọc được cache")
		}
	}
	ms, err := r.Src.Members(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("privacy: nạp roster: %w", err)
	}
	if r.Redis != nil {
		if raw, err := json.Marshal(ms); err == nil {
			rctx, cancel := context.WithTimeout(ctx, r.opTimeout())
			if err := r.Redis.Set(rctx, key, raw, RosterTTL).Err(); err != nil {
				r.warn("roster: không ghi được cache")
			}
			cancel()
		}
	}
	return ms, nil
}

func (r *Roster) warn(msg string) {
	if r.Log != nil {
		r.Log.Warn(msg)
	}
}

func (r *Roster) index(ctx context.Context, courseID uuid.UUID) (*rosterIndex, error) {
	ms, err := r.members(ctx, courseID)
	if err != nil {
		return nil, err
	}
	return buildIndex(ms), nil
}

// Invalidate là outbox.Handler cho `course.member_changed` và `roster.imported`: xoá cache của lớp trong payload. Idempotent.
// Nối vào dòng đăng ký sẵn có bằng outbox.Chain (đăng ký trùng topic sẽ panic).
func (r *Roster) Invalidate(ctx context.Context, m outbox.Message) error {
	if r == nil || r.Redis == nil {
		return nil
	}
	var p struct {
		CourseID string `json:"course_id"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil {
		return fmt.Errorf("privacy: payload %s: %w", m.Topic, err)
	}
	id, err := uuid.Parse(p.CourseID)
	if err != nil {
		return nil // sự kiện không thuộc một lớp: không có gì để xoá
	}
	if err := r.Redis.Del(ctx, RosterKey(id)).Err(); err != nil {
		return fmt.Errorf("privacy: xoá cache roster: %w", err)
	}
	return nil
}

// rosterIndex: biến thể tên đã chuẩn hoá → họ tên đầy đủ chuẩn hoá (khoá thực thể); mã sinh viên chữ thường.
type rosterIndex struct {
	names  map[string]string
	forms  map[string][][]string // biến thể (đã bỏ dấu) → các dạng CÓ DẤU của nó (từng âm tiết, chữ thường NFC), để kiểm «dấu là tập con» (P-106-11)
	codes  map[string]string
	maxTok int
}

// buildIndex sinh biến thể cho mỗi tên ≥ 2 âm tiết: nguyên; đảo (tên trước họ: "An Nguyễn Văn", "An Nguyễn"); rút ("Nguyễn An").
// Mỗi biến thể lưu cả dạng có dấu (forms) — find chỉ nhận đoạn trong bài khi «dấu là tập con» của dạng đó.
func buildIndex(ms []Member) *rosterIndex {
	idx := &rosterIndex{names: map[string]string{}, forms: map[string][][]string{}, codes: map[string]string{}}
	for _, m := range ms {
		if c := strings.ToLower(strings.TrimSpace(m.Code)); c != "" {
			idx.codes[c] = c
		}
		acc := strings.Fields(strings.ToLower(norm.NFC.String(m.Name)))
		toks := make([]string, len(acc))
		for i, a := range acc {
			toks[i] = auth.Fold(a)
		}
		if len(toks) < 2 {
			continue
		}
		full := strings.Join(toks, " ")
		n := len(toks)
		orders := [][]int{seq(n), append([]int{n - 1}, seq(n-1)...), {n - 1, 0}, {0, n - 1}}
		if n >= 3 && n <= 4 { // mọi thứ tự của 3–4 âm tiết (người gõ đảo tuỳ ý: "Giang Ngọc Bùi"); 2 âm tiết đã đủ hai chiều ở trên
			orders = append(orders, permutationsOf(n)...)
		}
		for _, o := range orders {
			key, form := make([]string, len(o)), make([]string, len(o))
			for k, i := range o {
				key[k], form[k] = toks[i], acc[i]
			}
			v := strings.Join(key, " ")
			if _, taken := idx.names[v]; !taken {
				idx.names[v] = full
			}
			idx.forms[v] = append(idx.forms[v], form)
			idx.maxTok = max(idx.maxTok, len(o))
		}
	}
	return idx
}

func seq(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

// permutationsOf trả mọi hoán vị của 0..n-1 (n ≤ 4 → ≤ 24 phần tử).
func permutationsOf(n int) [][]int {
	var out [][]int
	var rec func(cur, rest []int)
	rec = func(cur, rest []int) {
		if len(rest) == 0 {
			out = append(out, cur)
			return
		}
		for i := range rest {
			next := append(append([]int{}, rest[:i]...), rest[i+1:]...)
			rec(append(append([]int{}, cur...), rest[i]), next)
		}
	}
	rec(nil, seq(n))
	return out
}

// diacriticSubset (P-106-11, «dấu là tập con»): từ trong bài khớp âm tiết của tên nếu trùng hẳn HOẶC là dạng không dấu của nó; khác dấu (ụ ≠ ũ) thì không.
// Giữ bắt tên gõ không dấu / thiếu dấu một phần ("Dung Nguyen", "Dũng Nguyen"), bỏ cụm thường có dấu khác ("áp dụng nguyên tắc" ≠ "Dũng Nguyễn").
func diacriticSubset(textWord, nameWord string) bool {
	t, n := []rune(strings.ToLower(norm.NFC.String(textWord))), []rune(nameWord)
	if len(t) != len(n) {
		return false
	}
	for i := range t {
		if t[i] != n[i] && auth.Fold(string(n[i])) != string(t[i]) {
			return false
		}
	}
	return true
}

func (idx *rosterIndex) accentOK(key string, runes []rune, ws []word) bool {
	for _, form := range idx.forms[key] {
		ok := len(form) == len(ws)
		for k := 0; ok && k < len(ws); k++ {
			ok = diacriticSubset(string(runes[ws[k].start:ws[k].end]), form[k])
		}
		if ok {
			return true
		}
	}
	return false
}

type word struct {
	start, end int // rune
	folded     string
}

func (idx *rosterIndex) hasCode(k string) bool { _, ok := idx.codes[k]; return ok }

// find tìm tên và mã sinh viên của roster theo ranh giới từ, không phân biệt hoa-thường / dấu; ưu tiên cụm dài nhất.
func (idx *rosterIndex) find(text string) []match {
	if len(idx.names) == 0 && len(idx.codes) == 0 {
		return nil
	}
	runes := []rune(text)
	var ws []word
	for i := 0; i < len(runes); {
		if !isWordRune(runes[i]) {
			i++
			continue
		}
		j := i
		for j < len(runes) && isWordRune(runes[j]) {
			j++
		}
		ws = append(ws, word{start: i, end: j, folded: auth.Fold(string(runes[i:j]))})
		i = j
	}
	var out []match
	for i := 0; i < len(ws); {
		if _, ok := idx.codes[ws[i].folded]; ok {
			out = append(out, match{Finding: Finding{Kind: KindMSSV, Start: ws[i].start, End: ws[i].end}, key: ws[i].folded})
			i++
			continue
		}
		matched := 0
		for n := min(idx.maxTok, len(ws)-i); n >= 2; n-- {
			if !spaceOnlyGaps(runes, ws[i:i+n]) {
				continue
			}
			parts := make([]string, n)
			for k := range parts {
				parts[k] = ws[i+k].folded
			}
			key := strings.Join(parts, " ")
			if full, ok := idx.names[key]; ok && idx.accentOK(key, runes, ws[i:i+n]) {
				out = append(out, match{Finding: Finding{Kind: KindName, Start: ws[i].start, End: ws[i+n-1].end}, key: full})
				matched = n
				break
			}
		}
		i += max(matched, 1)
	}
	return out
}

// spaceOnlyGaps: giữa hai từ liền nhau chỉ có khoảng trắng (không dấu phẩy, gạch…).
func spaceOnlyGaps(runes []rune, ws []word) bool {
	for k := 1; k < len(ws); k++ {
		if ws[k].start == ws[k-1].end {
			return false
		}
		for _, r := range runes[ws[k-1].end:ws[k].start] {
			if !unicode.IsSpace(r) {
				return false
			}
		}
	}
	return true
}
