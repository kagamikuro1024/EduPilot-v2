package privacy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
)

// Hằng số của Mask (SRS 4.2.5).
const (
	MaskTTL        = 24 * time.Hour        // tính từ lần dùng gần nhất
	MaskTimeout    = 50 * time.Millisecond // quá hạn → ErrMaskFailed (không gọi provider)
	redisOpTimeout = 30 * time.Millisecond // Redis chậm / hỏng → dùng ánh xạ trong bộ nhớ (kết nối lạnh có thể mất > 15 ms — QC US-P3-02 ghi chú 1)
	rosterLoadCap  = 2 * time.Second       // nạp roster khi trượt cache (một truy vấn); không tính vào MaskTimeout
)

// ErrMaskFailed: bước che lỗi (panic, quá hạn, roster không nạp được). Người gọi KHÔNG được gửi payload tới provider.
var ErrMaskFailed = errors.New("MASK_FAILED")

// placeholderKind: loại → tên trong placeholder `[[SV_1]]`.
func placeholderKind(k Kind) string {
	switch k {
	case KindName:
		return "SV"
	case KindEmail:
		return "EMAIL"
	case KindPhone:
		return "SDT"
	case KindCCCD:
		return "CCCD"
	}
	return "MSSV"
}

// Session là phạm vi của ánh xạ placeholder. ID() == "" nghĩa là phạm vi yêu cầu: ánh xạ chỉ ở bộ nhớ, không tạo khoá Redis.
type Session interface {
	ID() string
	sess() *Sess
}

// Sess cài Session; giữ bản ánh xạ trong bộ nhớ (cho phạm vi yêu cầu và khi Redis hỏng).
type Sess struct {
	id string
	mu sync.Mutex
	ph map[string]string // placeholder → bản gốc
	rk map[string]string // sha256(kind|khoá) → placeholder
	n  map[string]int    // bộ đếm theo loại
}

// NewSession dựng phiên; id rỗng = phạm vi yêu cầu.
func NewSession(id string) *Sess {
	return &Sess{id: id, ph: map[string]string{}, rk: map[string]string{}, n: map[string]int{}}
}

// ID trả id phiên chat ("" = phạm vi yêu cầu).
func (s *Sess) ID() string { return s.id }

func (s *Sess) sess() *Sess { return s }

type sessionKey struct{}

// WithSession gắn phiên vào ctx cho hook che của internal/llm (do internal/chat gọi).
func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, s)
}

// SessionFrom lấy phiên từ ctx; nil nếu không có.
func SessionFrom(ctx context.Context) Session {
	s, _ := ctx.Value(sessionKey{}).(Session)
	return s
}

// MaskKey là khoá Redis của ánh xạ một phiên.
func MaskKey(sessionID string) string { return appredis.Key("mask", sessionID) }

// Masker che / khôi phục. Redis nil = luôn dùng bộ nhớ.
type Masker struct {
	Detector *Detector
	Redis    *appredis.Client
	Log      *slog.Logger
	Timeout  time.Duration // hạn che; 0 = MaskTimeout (PRIVACY_MASK_TIMEOUT_MS)
}

func (m *Masker) timeout() time.Duration {
	if m.Timeout > 0 {
		return m.Timeout
	}
	return MaskTimeout
}

const maskScript = `
local out = {}
for i = 2, #ARGV, 3 do
  local kind, h, orig = ARGV[i], ARGV[i+1], ARGV[i+2]
  local ph = redis.call('HGET', KEYS[1], 'r:' .. h)
  if not ph then
    local n = redis.call('HINCRBY', KEYS[1], 'n:' .. kind, 1)
    ph = '[[' .. kind .. '_' .. n .. ']]'
    redis.call('HSET', KEYS[1], 'r:' .. h, ph, 'p:' .. ph, orig)
  end
  out[#out + 1] = ph
end
redis.call('EXPIRE', KEYS[1], ARGV[1])
return out`

type entity struct{ kind, hash, orig string }

// Mask che mọi PII trong texts bằng placeholder ổn định theo phiên. Văn bản không có PII trả về cùng chuỗi byte.
// Người đang chat cũng bị che (họ nằm trong roster). Lỗi / panic / quá hạn → ErrMaskFailed.
func (m *Masker) Mask(ctx context.Context, courseID uuid.UUID, s Session, texts []string) (masked []string, n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			masked, n, err = nil, 0, fmt.Errorf("%w: panic", ErrMaskFailed)
		}
	}()
	if s == nil {
		s = NewSession("")
	}
	// Roster: nạp (có thể một truy vấn DB khi trượt cache) với hạn riêng, rồi mới đếm hạn MaskTimeout cho phần che.
	var idx *rosterIndex
	if m.Detector != nil && m.Detector.Roster != nil && courseID != uuid.Nil {
		lctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rosterLoadCap)
		var ierr error
		idx, ierr = m.Detector.Roster.index(lctx, courseID)
		cancel()
		if ierr != nil {
			return nil, 0, fmt.Errorf("%w: %w", ErrMaskFailed, ierr)
		}
	}
	deadline := time.Now().Add(m.timeout())

	type found struct{ ms []match }
	per := make([]found, len(texts))
	var ents []entity
	for i, t := range texts {
		if time.Now().After(deadline) {
			return nil, 0, fmt.Errorf("%w: quá hạn", ErrMaskFailed)
		}
		per[i].ms = detectIn(t, idx)
		for _, f := range per[i].ms {
			ents = append(ents, entityOf(t, f))
		}
	}
	if len(ents) == 0 {
		return texts, 0, nil
	}
	phs := m.resolve(ctx, s, ents)
	masked = make([]string, len(texts))
	k := 0
	for i, t := range texts {
		if len(per[i].ms) == 0 {
			masked[i] = t
			continue
		}
		runes := []rune(t)
		var b strings.Builder
		pos := 0
		for _, f := range per[i].ms {
			b.WriteString(string(runes[pos:f.Start]))
			b.WriteString(phs[k])
			k++
			pos = f.End
		}
		b.WriteString(string(runes[pos:]))
		masked[i] = b.String()
		n += len(per[i].ms)
	}
	return masked, n, nil
}

func entityOf(text string, f match) entity {
	runes := []rune(text)
	sum := sha256.Sum256([]byte(f.key))
	return entity{kind: placeholderKind(f.Kind), hash: hex.EncodeToString(sum[:]), orig: string(runes[f.Start:f.End])}
}

// resolve trả placeholder cho từng thực thể (đúng thứ tự ents). Redis lỗi → bộ nhớ của phiên, ghi warn không kèm nội dung.
func (m *Masker) resolve(ctx context.Context, s Session, ents []entity) []string {
	ss := s.sess()
	if s.ID() != "" && m.Redis != nil {
		args := make([]any, 0, 1+3*len(ents))
		args = append(args, int(MaskTTL.Seconds()))
		for _, e := range ents {
			args = append(args, e.kind, e.hash, e.orig)
		}
		rctx, cancel := context.WithTimeout(ctx, redisOpTimeout)
		res, err := redis.NewScript(maskScript).Run(rctx, m.Redis, []string{MaskKey(s.ID())}, args...).StringSlice()
		cancel()
		if err == nil && len(res) == len(ents) {
			return res
		}
		m.warn("mask: Redis không dùng được, dùng ánh xạ trong bộ nhớ")
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	out := make([]string, len(ents))
	for i, e := range ents {
		rk := e.kind + "|" + e.hash
		ph, ok := ss.rk[rk]
		if !ok {
			ss.n[e.kind]++
			ph = "[[" + e.kind + "_" + strconv.Itoa(ss.n[e.kind]) + "]]"
			ss.rk[rk] = ph
			ss.ph[ph] = e.orig
		}
		out[i] = ph
	}
	return out
}

// original tra bản gốc của một placeholder chuẩn (`[[SV_1]]`): bộ nhớ trước, rồi Redis.
func (m *Masker) original(ctx context.Context, s Session, ph string) (string, bool) {
	ss := s.sess()
	ss.mu.Lock()
	v, ok := ss.ph[ph]
	ss.mu.Unlock()
	if ok {
		return v, true
	}
	if s.ID() == "" || m.Redis == nil {
		return "", false
	}
	rctx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()
	v, err := m.Redis.HGet(rctx, MaskKey(s.ID()), "p:"+ph).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			m.warn("unmask: Redis không đọc được")
		}
		return "", false
	}
	return v, true
}

// Unmask khôi phục mọi placeholder trong text (chịu khoảng trắng / hoa-thường), rồi quét sót: placeholder không có trong ánh xạ → "bạn".
// Dùng chung máy trạng thái với StreamUnmasker nên Unmask(toàn bộ) == nối các lần Write + Flush.
func (m *Masker) Unmask(ctx context.Context, s Session, text string) string {
	u := m.NewStreamUnmasker(ctx, s)
	return u.Write(text) + u.Flush()
}

func (m *Masker) warn(msg string, args ...any) {
	if m.Log != nil {
		m.Log.Warn(msg, args...)
	}
}
