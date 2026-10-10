package agent

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/store"
)

// RedisKV cài RedisKV trên Redis thật.
type redisKV struct{ r *appredis.Client }

// NewRedisKV bọc client Redis cho AnswerCache.
func NewRedisKV(r *appredis.Client) RedisKV { return redisKV{r} }

func (k redisKV) GetString(ctx context.Context, key string) (string, bool, error) {
	v, err := k.r.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("redis get: %w", err)
	}
	return v, true, nil
}

func (k redisKV) SetString(ctx context.Context, key, val string, ttlSec int) error {
	if err := k.r.Set(ctx, key, val, time.Duration(ttlSec)*time.Second).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// EmbedTTL là hạn cache nhúng `ep:emb:{sha256(chuẩn hoá(chữ thô))}` — giá trị là vectơ nên KHÔNG chứa chữ (SRS 5.7).
const EmbedTTL = 10 * time.Minute

// NewEmbedder trả hàm nhúng một câu hỏi: làn INTERACTIVE, qua cổng llm (hook che chạy ở đó), cache theo băm chữ thô. rdb nil = không cache.
// Cùng khoá dùng cho worker AI Threads để không nhúng lần hai (TLR-16).
func NewEmbedder(gw llm.Client, rdb *appredis.Client) func(ctx context.Context, text string) ([]float32, error) {
	lane := llm.LaneInteractive
	return func(ctx context.Context, text string) ([]float32, error) {
		norm := strings.Join(strings.Fields(auth.Fold(text)), " ")
		sum := sha256.Sum256([]byte(norm))
		key := appredis.Key("emb", hex.EncodeToString(sum[:]))
		if rdb != nil {
			if raw, err := rdb.Get(ctx, key).Bytes(); err == nil && len(raw) == 4*llm.EmbedDims {
				return decodeVec(raw), nil
			}
		}
		vecs, err := gw.Embed(ctx, llm.EmbedRequest{Inputs: []string{text}, Lane: &lane})
		if err != nil {
			return nil, fmt.Errorf("agent: nhúng: %w", err)
		}
		if len(vecs) != 1 {
			return nil, fmt.Errorf("agent: nhúng trả %d vectơ", len(vecs))
		}
		if rdb != nil {
			_ = rdb.Set(ctx, key, encodeVec(vecs[0]), EmbedTTL).Err() // cache hỏng không làm hỏng lượt hỏi
		}
		return vecs[0], nil
	}
}

func encodeVec(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(x))
	}
	return b
}

func decodeVec(b []byte) []float32 {
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return v
}

// StoreSelf tra danh tính của người trong TrustedContext từ Postgres.
type StoreSelf struct{ Pool *pgxpool.Pool }

// Self cài SelfSource. Không có ghi danh → lỗi (người gọi coi như không xác định được: không hỏi hộ được).
func (s StoreSelf) Self(ctx context.Context, tc TrustedContext) (Self, error) {
	row, err := store.New(s.Pool).AgentSelf(ctx, store.AgentSelfParams{UserID: tc.UserID, CourseID: tc.CourseID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Self{}, nil
	}
	if err != nil {
		return Self{}, fmt.Errorf("agent: danh tính người hỏi: %w", err)
	}
	code := ""
	if row.StudentCodeSnapshot != nil {
		code = *row.StudentCodeSnapshot
	}
	return NewSelf(row.FullName, code, row.Email), nil
}

// StoreEvents ghi pii_events (chỉ loại + số lượng, không nội dung).
type StoreEvents struct{ Pool *pgxpool.Pool }

// Record cài EventSink: pii là giá trị enum pii_kind (vd. OTHER_PERSON), action là pii_action (vd. BLOCKED).
func (e StoreEvents) Record(ctx context.Context, tc TrustedContext, pii, action string, channel privacy.Channel, count int) error {
	var sid *uuid.UUID
	if tc.SessionID != uuid.Nil {
		sid = &tc.SessionID
	}
	err := store.New(e.Pool).InsertPIIEvent(ctx, store.InsertPIIEventParams{
		CourseID: tc.CourseID, SessionID: sid, UserID: tc.UserID, Channel: store.ChatChannel(channel),
		PiiType: store.PiiKind(pii), Count: int32(count), Action: store.PiiAction(action), //nolint:gosec // đếm nhỏ
	})
	if err != nil {
		return fmt.Errorf("agent: ghi pii_events: %w", err)
	}
	return nil
}
