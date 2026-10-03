// Package outbox: transactional outbox (luật 14). `Write` ghi dòng outbox trong CÙNG transaction với việc nghiệp vụ;
// relay + consumer ở worker (agent US-PG-02 bổ sung: relay.go, consumer.go) chuyển dòng tới handler theo topic,
// at-least-once, retry 3 lần rồi dead-letter. Handler PHẢI idempotent.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/store"
)

var topicRE = regexp.MustCompile(`^[a-z][a-z0-9_.]{0,63}$`)

// Message là một dòng outbox đã nạp, đưa cho handler.
type Message struct {
	ID       uuid.UUID
	Topic    string
	Payload  json.RawMessage
	Attempts int // số lần handler đã chạy TRƯỚC lần này (0 ở lần đầu)
}

// Handler xử lý một tin; trả lỗi (hoặc panic) = thất bại → retry/backoff/dead-letter. Phải idempotent theo Message.ID.
type Handler func(ctx context.Context, m Message) error

// Registry là bảng topic → handler của một tiến trình worker. Không biến toàn cục: worker tự dựng một Registry.
type Registry struct {
	mu sync.RWMutex
	h  map[string]Handler
}

// NewRegistry dựng Registry rỗng.
func NewRegistry() *Registry { return &Registry{h: map[string]Handler{}} }

// Register gắn handler cho topic; topic sai định dạng hoặc đăng ký hai lần → panic (lỗi lập trình lúc khởi động).
func (r *Registry) Register(topic string, h Handler) {
	if !topicRE.MatchString(topic) {
		panic(fmt.Sprintf("outbox: topic %q sai định dạng", topic))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.h[topic]; dup {
		panic(fmt.Sprintf("outbox: topic %q đã đăng ký", topic))
	}
	r.h[topic] = h
}

// Lookup trả handler của topic.
func (r *Registry) Lookup(topic string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.h[topic]
	return h, ok
}

// Write ghi một dòng outbox bằng chính transaction `tx` của việc nghiệp vụ (commit cùng, rollback cùng).
func Write(ctx context.Context, tx pgx.Tx, topic string, payload any) (uuid.UUID, error) {
	if !topicRE.MatchString(topic) {
		return uuid.Nil, fmt.Errorf("outbox write: topic %q sai định dạng", topic)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("outbox write: marshal payload: %w", err)
	}
	row, err := store.New(tx).InsertOutbox(ctx, store.InsertOutboxParams{Topic: topic, Payload: raw})
	if err != nil {
		return uuid.Nil, fmt.Errorf("outbox write: insert: %w", err)
	}
	return row.ID, nil
}
