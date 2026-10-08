package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn sự kiện liêm chính (US-PE-07 AC3).
const (
	eventsBatchMax = 50  // một yêu cầu nhận tối đa 50 sự kiện đầu
	metaMaxBytes   = 300 // `exam_events.meta` ≤ 300 byte
)

// IntegrityConfig: EXAM_EVENTS_MAX (500 sự kiện / lượt) và SIMILARITY_MIN (0,60 = 600‰, số nguyên: không dùng số thực). Số 0 = mặc định.
type IntegrityConfig struct {
	EventsMax             int
	SimilarityMinPermille int
}

func (c IntegrityConfig) eventsMax() int {
	if c.EventsMax <= 0 {
		return 500
	}
	return c.EventsMax
}

// EventIn là một sự kiện máy khách gửi lên. `type` ngoài danh sách cho phép bị bỏ (không lỗi: sự kiện không bao giờ làm hỏng việc lưu bài).
type EventIn struct {
	Type     string                     `json:"type"`
	ClientAt *time.Time                 `json:"client_at"`
	Meta     map[string]json.RawMessage `json:"meta"`
}

// EventsIn là thân `POST …/attempts/{aid}/events`.
type EventsIn struct {
	Events []EventIn `json:"events" validate:"required"`
}

func allowedClientEvent(t string) bool {
	switch t {
	case "TAB_HIDDEN", "TAB_VISIBLE", "PASTE", "OFFLINE", "ONLINE":
		return true
	}
	return false
}

// cleanMeta chỉ giữ khoá cho phép `duration_ms`, `chars` (số nguyên ≥ 0) và `item_id` (uuid); khoá lạ / giá trị sai bị bỏ; quá 300 byte → `{}`. Không bao giờ giữ nội dung.
func cleanMeta(in map[string]json.RawMessage) map[string]any {
	out := map[string]any{}
	for k, raw := range in {
		switch k {
		case "duration_ms", "chars":
			var n int64
			if json.Unmarshal(raw, &n) == nil && n >= 0 && n <= 1_000_000_000 { // số nguyên; số thực / chữ → bỏ
				out[k] = n
			}
		case "item_id":
			var str string
			if json.Unmarshal(raw, &str) == nil {
				if id, err := uuid.Parse(str); err == nil {
					out[k] = id.String()
				}
			}
		}
	}
	if b, _ := json.Marshal(out); len(b) > metaMaxBytes {
		return map[string]any{}
	}
	return out
}

func (s *Service) logger() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// RecordEvents: `POST …/attempts/{aid}/events` — ghi sự kiện rời tab / dán (không nội dung). Lượt của người khác → 404; lượt không IN_PROGRESS → bỏ qua (204);
// lỗi ghi KHÔNG trả ra ngoài (không bao giờ làm lượt thất bại). Thời điểm lưu là giờ máy chủ.
func (s *Service) RecordEvents(ctx context.Context, userID, courseID, examID, attemptID uuid.UUID, in EventsIn) error {
	q := store.New(s.Pool)
	a, err := q.AttemptOwn(ctx, store.AttemptOwnParams{CourseID: courseID, ExamID: examID, ID: attemptID, StudentID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		s.logger().WarnContext(ctx, "đọc lượt để ghi sự kiện lỗi", "attempt_id", attemptID, "error", err.Error())
		return nil
	}
	if a.Status != store.AttemptStatusINPROGRESS || len(in.Events) == 0 {
		return nil
	}
	evs := in.Events
	if len(evs) > eventsBatchMax {
		s.logger().WarnContext(ctx, "lô sự kiện liêm chính vượt giới hạn: bỏ phần dư", "attempt_id", attemptID, "dropped", len(evs)-eventsBatchMax)
		evs = evs[:eventsBatchMax]
	}
	have, err := q.ExamEventCount(ctx, store.ExamEventCountParams{CourseID: courseID, AttemptID: attemptID})
	if err != nil {
		s.logger().WarnContext(ctx, "đếm sự kiện lỗi", "attempt_id", attemptID, "error", err.Error())
		return nil
	}
	room := max(0, s.Integrity.eventsMax()-int(have))
	rows := make([]map[string]any, 0, len(evs))
	for _, e := range evs {
		if !allowedClientEvent(e.Type) {
			continue
		}
		if len(rows) >= room {
			s.countDropped(ctx, attemptID)
			continue
		}
		rows = append(rows, map[string]any{"type": e.Type, "client_at": e.ClientAt, "meta": cleanMeta(e.Meta)})
	}
	if len(rows) == 0 {
		return nil
	}
	raw, _ := json.Marshal(rows)
	if err := q.ExamEventsInsert(ctx, store.ExamEventsInsertParams{CourseID: courseID, ExamID: examID, AttemptID: attemptID, StudentID: userID, At: s.now(), Rows: raw}); err != nil {
		s.logger().WarnContext(ctx, "ghi sự kiện liêm chính lỗi", "attempt_id", attemptID, "error", err.Error())
	}
	return nil
}

// countDropped tăng bộ đếm sự kiện bị bỏ vì quá EXAM_EVENTS_MAX (chỉ thông tin; Redis lỗi bỏ qua).
func (s *Service) countDropped(ctx context.Context, attemptID uuid.UUID) {
	if s.Redis == nil {
		return
	}
	k := appredis.Key("exam", "events", "dropped", attemptID.String())
	if s.Redis.Incr(ctx, k).Err() == nil {
		_ = s.Redis.Expire(ctx, k, 30*24*time.Hour).Err()
	}
}

// ---- Giảng viên đọc log ---------------------------------------------------------------------------------------------------

// EventView là một sự kiện như Giảng viên thấy.
type EventView struct {
	ID         uuid.UUID       `json:"id"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	ClientAt   *time.Time      `json:"client_at"`
	Meta       json.RawMessage `json:"meta"`
}

// IntegritySummary là tóm tắt MỘT lượt: tín hiệu để tham khảo, không phải kết luận.
type IntegritySummary struct {
	TabHiddenCount   int   `json:"tab_hidden_count"`
	TabHiddenMS      int64 `json:"tab_hidden_ms"`
	PasteCount       int   `json:"paste_count"`
	PasteChars       int64 `json:"paste_chars"`
	OfflineCount     int   `json:"offline_count"`
	TakeoverCount    int   `json:"takeover_count"`
	ChatBlockedCount int   `json:"chat_blocked_count"`
}

// EventsPage là phản hồi `GET …/events`.
type EventsPage struct {
	Summary IntegritySummary `json:"summary"`
	Items   []EventView      `json:"items"`
}

// Summary tính tóm tắt của một lượt (dùng cả ở trang kết quả — US-PE-08).
func (s *Service) Summary(ctx context.Context, courseID, attemptID uuid.UUID) (IntegritySummary, error) {
	rows, err := store.New(s.Pool).ExamEventsSummary(ctx, store.ExamEventsSummaryParams{CourseID: courseID, AttemptID: attemptID})
	if err != nil {
		return IntegritySummary{}, fmt.Errorf("exam: tóm tắt sự kiện: %w", err)
	}
	var out IntegritySummary
	for _, r := range rows {
		switch r.Type {
		case store.ExamEventTypeTABHIDDEN:
			out.TabHiddenCount, out.TabHiddenMS = int(r.N), r.DurationMs
		case store.ExamEventTypePASTE:
			out.PasteCount, out.PasteChars = int(r.N), r.Chars
		case store.ExamEventTypeOFFLINE:
			out.OfflineCount = int(r.N)
		case store.ExamEventTypeTABTAKEOVER:
			out.TakeoverCount = int(r.N)
		case store.ExamEventTypeCHATBLOCKED:
			out.ChatBlockedCount = int(r.N)
		}
	}
	return out, nil
}

// ListEvents: `GET …/events?attempt=` — CHỈ Giảng viên (kiểm ở guard). Lượt không thuộc bài → 404. Mới nhất trước, cursor.
func (s *Service) ListEvents(ctx context.Context, courseID, examID, attemptID uuid.UUID, cur *Cursor, fetch int) (EventsPage, error) {
	q := store.New(s.Pool)
	if _, err := q.AttemptOfExam(ctx, store.AttemptOfExamParams{CourseID: courseID, ExamID: examID, ID: attemptID}); errors.Is(err, pgx.ErrNoRows) {
		return EventsPage{}, notFound()
	} else if err != nil {
		return EventsPage{}, fmt.Errorf("exam: đọc lượt làm: %w", err)
	}
	sum, err := s.Summary(ctx, courseID, attemptID)
	if err != nil {
		return EventsPage{}, err
	}
	arg := store.ExamEventsListParams{CourseID: courseID, ExamID: examID, AttemptID: attemptID, RowLimit: int32(fetch)} //nolint:gosec // ≤ 101
	if cur != nil {
		arg.CursorAt, arg.CursorID = &cur.At, &cur.ID
	}
	rows, err := q.ExamEventsList(ctx, arg)
	if err != nil {
		return EventsPage{}, fmt.Errorf("exam: danh sách sự kiện: %w", err)
	}
	items := make([]EventView, 0, len(rows))
	for _, r := range rows {
		items = append(items, EventView{ID: r.ID, Type: string(r.Type), OccurredAt: r.OccurredAt, ClientAt: r.ClientAt, Meta: r.Meta})
	}
	return EventsPage{Summary: sum, Items: items}, nil
}
