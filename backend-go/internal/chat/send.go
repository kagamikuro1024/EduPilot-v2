package chat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

const busyTTL = 130 * time.Second

const releaseLuaSrc = `if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) end return 0`

// precheck gom các kiểm tra theo thứ tự SRS 4.7.1 bước 3–4 (khoá giờ thi TRƯỚC tốc độ / phân loại / nhúng / provider).
func (s *Service) precheck(ctx context.Context, a Actor) error {
	lk, locked, err := s.Lock.IsLocked(ctx, a.UserID)
	if err != nil {
		if s.Log != nil {
			s.Log.ErrorContext(ctx, "chat: tra khoá giờ thi lỗi, từ chối chat", "error", err)
		}
		return unavailable()
	}
	if locked {
		if err := s.Lock.RecordChatBlocked(ctx, a.UserID, lk.AttemptID); err != nil && s.Log != nil {
			s.Log.WarnContext(ctx, "chat: ghi CHAT_BLOCKED lỗi", "error", err)
		}
		return apierr.New(http.StatusConflict, apierr.ExamInProgress).WithDetails(map[string]any{"until": lk.Until.UTC().Format(time.RFC3339)})
	}
	return s.rate(ctx, a.UserID)
}

// rate: cửa sổ cố định theo phút đồng hồ; retry_after = số giây còn lại tới hết phút. Redis lỗi → bỏ qua giới hạn (không chặn chat).
func (s *Service) rate(ctx context.Context, uid uuid.UUID) error {
	if s.Redis == nil {
		return nil
	}
	now := s.now()
	k := fmt.Sprintf("ep:rl:chat:%s:%d", uid, now.Unix()/60)
	n, err := s.Redis.Incr(ctx, k).Result()
	if err != nil {
		if s.Log != nil {
			s.Log.WarnContext(ctx, "chat: giới hạn tốc độ lỗi Redis, bỏ qua", "error", err)
		}
		return nil
	}
	if n == 1 {
		_ = s.Redis.Expire(ctx, k, 70*time.Second).Err()
	}
	if int(n) > s.c().RatePerMin {
		return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(int(60 - now.Unix()%60))
	}
	return nil
}

// takeBusy: SET NX; trả (true, "") khi giành được; (false, giá trị đang giữ) khi không. Redis lỗi → coi như giành được (không chặn chat).
func (s *Service) takeBusy(ctx context.Context, uid uuid.UUID, val string) (bool, string) {
	if s.Redis == nil {
		return true, ""
	}
	ok, err := s.Redis.SetNX(ctx, busyKey(uid), val, busyTTL).Result()
	if err != nil {
		if s.Log != nil {
			s.Log.WarnContext(ctx, "chat: khoá CHAT_BUSY lỗi Redis, bỏ qua", "error", err)
		}
		return true, ""
	}
	if ok {
		return true, ""
	}
	cur, _ := s.Redis.Get(ctx, busyKey(uid)).Result()
	return false, cur
}

// releaseBusy nhả khoá CHAT_BUSY nếu còn đúng của lượt này (so sánh giá trị, không nhả nhầm lượt khác).
func (s *Service) releaseBusy(ctx context.Context, uid uuid.UUID, val string) {
	if s.Redis == nil || val == "" {
		return
	}
	if err := goredis.NewScript(releaseLuaSrc).Run(ctx, s.Redis, []string{busyKey(uid)}, val).Err(); err != nil && s.Log != nil {
		s.Log.WarnContext(ctx, "chat: nhả CHAT_BUSY lỗi", "error", err)
	}
}

func (s *Service) validate(content string) (string, error) {
	t := strings.TrimSpace(content)
	if t == "" {
		return "", apierr.Validation(apierr.FieldError{Field: "content", Code: "required", Message: "Hãy nhập nội dung."})
	}
	if utf8.RuneCountInString(t) > s.c().MaxInput {
		return "", apierr.New(http.StatusUnprocessableEntity, apierr.MessageTooLong).WithDetails(map[string]any{"limit": s.c().MaxInput})
	}
	return t, nil
}

// Send nhận một tin: ghi USER + ASSISTANT(STREAMING) trong một giao dịch rồi khởi động G. Trả hàng ASSISTANT để người gọi `Tail`.
// Gửi lại cùng clientMsgID → phát lại hàng cũ, không tạo thêm.
func (s *Service) Send(ctx context.Context, a Actor, sid, clientMsgID uuid.UUID, content string) (store.ChatMessage, error) {
	sess, err := s.session(ctx, a, sid)
	if err != nil {
		return store.ChatMessage{}, err
	}
	if err := s.archived(ctx, sess.CourseID); err != nil {
		return store.ChatMessage{}, err
	}
	text, err := s.validate(content)
	if err != nil {
		return store.ChatMessage{}, err
	}
	if err := s.precheck(ctx, a); err != nil {
		return store.ChatMessage{}, err
	}
	q := store.New(s.Pool)
	if m, ok, err := s.replay(ctx, q, sid, clientMsgID); err != nil || ok {
		return m, err
	}
	val := clientMsgID.String()
	if got, cur := s.takeBusy(ctx, a.UserID, val); !got {
		if cur == val { // bản đầu của chính tin này còn chạy: phát lại
			if m, ok, err := s.replayWait(ctx, q, sid, clientMsgID); err != nil || ok {
				return m, err
			}
		}
		return store.ChatMessage{}, apierr.New(http.StatusConflict, apierr.ChatBusy)
	}
	user, asst, err := s.insertPair(ctx, sess, a, clientMsgID, text)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" { // cuộc đua cùng khoá: bản đã commit thắng — phát lại, KHÔNG nhả khoá của nó
			if m, ok, rerr := s.replay(ctx, q, sid, clientMsgID); rerr != nil || ok {
				return m, rerr
			}
		}
		s.releaseBusy(ctx, a.UserID, val)
		return store.ChatMessage{}, fmt.Errorf("chat: ghi tin: %w", err)
	}
	if err := s.start(ctx, a, &run{sess: sess, user: user, asst: asst, text: text, busy: val}); err != nil {
		return store.ChatMessage{}, err
	}
	return asst, nil
}

func (s *Service) replay(ctx context.Context, q *store.Queries, sid, clientMsgID uuid.UUID) (store.ChatMessage, bool, error) {
	u, err := q.GetChatUserMessageByClientID(ctx, store.GetChatUserMessageByClientIDParams{SessionID: sid, ClientMsgID: &clientMsgID})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ChatMessage{}, false, nil
	}
	if err != nil {
		return store.ChatMessage{}, false, fmt.Errorf("chat: tra tin theo khoá: %w", err)
	}
	m, err := q.GetChatAssistantFor(ctx, store.GetChatAssistantForParams{SessionID: sid, ReplyTo: &u.ID})
	if err != nil {
		return store.ChatMessage{}, false, fmt.Errorf("chat: tra câu trả lời: %w", err)
	}
	return m, true, nil
}

// replayWait: bản đầu đã giành khoá nhưng có thể chưa commit — chờ ≤ 2 s.
func (s *Service) replayWait(ctx context.Context, q *store.Queries, sid, clientMsgID uuid.UUID) (store.ChatMessage, bool, error) {
	for range 20 {
		if m, ok, err := s.replay(ctx, q, sid, clientMsgID); err != nil || ok {
			return m, ok, err
		}
		select {
		case <-ctx.Done():
			return store.ChatMessage{}, false, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return store.ChatMessage{}, false, nil
}

func (s *Service) insertPair(ctx context.Context, sess store.ChatSession, a Actor, clientMsgID uuid.UUID, text string) (user, asst store.ChatMessage, err error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return user, asst, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	user, err = q.InsertChatMessage(ctx, store.InsertChatMessageParams{
		CourseID: sess.CourseID, SessionID: sess.ID, UserID: a.UserID, Role: store.ChatRoleUSER, Content: text,
		StreamStatus: store.ChatStreamStatusDONE, ClientMsgID: &clientMsgID,
	})
	if err != nil {
		return user, asst, err
	}
	asst, err = q.InsertChatMessage(ctx, store.InsertChatMessageParams{
		CourseID: sess.CourseID, SessionID: sess.ID, UserID: a.UserID, Role: store.ChatRoleASSISTANT,
		StreamStatus: store.ChatStreamStatusSTREAMING, ReplyTo: &user.ID,
	})
	if err != nil {
		return user, asst, err
	}
	if err := q.TouchChatSession(ctx, sess.ID); err != nil {
		return user, asst, err
	}
	if sess.Title == nil {
		if err := q.SetChatTitle(ctx, store.SetChatTitleParams{ID: sess.ID, Title: ptr(title60(text))}); err != nil {
			return user, asst, err
		}
	}
	return user, asst, tx.Commit(ctx)
}

func ptr[T any](v T) *T { return &v }

// Retry đặt lại CÙNG hàng ASSISTANT (FAILED / CANCELLED) rồi sinh lại; không tạo bong bóng người dùng thứ hai.
func (s *Service) Retry(ctx context.Context, a Actor, mid uuid.UUID) (store.ChatMessage, error) {
	m, err := s.message(ctx, a, mid)
	if err != nil {
		return store.ChatMessage{}, err
	}
	if m.Role != store.ChatRoleASSISTANT || (m.StreamStatus != store.ChatStreamStatusFAILED && m.StreamStatus != store.ChatStreamStatusCANCELLED) {
		return store.ChatMessage{}, apierr.New(http.StatusConflict, apierr.MessageNotRetryable)
	}
	sess, err := s.session(ctx, a, m.SessionID)
	if err != nil {
		return store.ChatMessage{}, err
	}
	if err := s.archived(ctx, sess.CourseID); err != nil {
		return store.ChatMessage{}, err
	}
	if err := s.precheck(ctx, a); err != nil {
		return store.ChatMessage{}, err
	}
	q := store.New(s.Pool)
	if m.ReplyTo == nil {
		return store.ChatMessage{}, apierr.New(http.StatusConflict, apierr.MessageNotRetryable)
	}
	user, err := q.GetChatUserMessageOf(ctx, *m.ReplyTo)
	if err != nil {
		return store.ChatMessage{}, fmt.Errorf("chat: nạp tin người dùng: %w", err)
	}
	val := mid.String()
	if got, _ := s.takeBusy(ctx, a.UserID, val); !got {
		return store.ChatMessage{}, apierr.New(http.StatusConflict, apierr.ChatBusy)
	}
	asst, err := q.RetryChatMessage(ctx, store.RetryChatMessageParams{ID: mid, UserID: a.UserID})
	if errors.Is(err, pgx.ErrNoRows) {
		s.releaseBusy(ctx, a.UserID, val)
		return store.ChatMessage{}, apierr.New(http.StatusConflict, apierr.MessageNotRetryable)
	}
	if err != nil {
		s.releaseBusy(ctx, a.UserID, val)
		return store.ChatMessage{}, fmt.Errorf("chat: đặt lại tin: %w", err)
	}
	if err := s.start(ctx, a, &run{sess: sess, user: user, asst: asst, text: user.Content, busy: val}); err != nil {
		return store.ChatMessage{}, err
	}
	return asst, nil
}

// Cancel: PUBLISH huỷ; không ai nghe (không có G sống) mà DB còn STREAMING → ghi CANCELLED thẳng. Idempotent; không phải của mình → 404.
func (s *Service) Cancel(ctx context.Context, a Actor, mid uuid.UUID) error {
	m, err := s.message(ctx, a, mid)
	if err != nil {
		return err
	}
	if m.StreamStatus != store.ChatStreamStatusSTREAMING {
		return nil
	}
	if s.Redis != nil {
		if n, perr := s.Redis.Publish(ctx, cancelChan(mid), "1").Result(); perr == nil && n > 0 {
			return nil
		}
	}
	n, err := store.New(s.Pool).CancelChatMessage(ctx, store.CancelChatMessageParams{ID: mid, Attempt: m.Attempt})
	if err != nil {
		return fmt.Errorf("chat: dừng: %w", err)
	}
	if n > 0 {
		s.releaseBusyOf(ctx, a.UserID, m)
	}
	return nil
}

// releaseBusyOf nhả khoá của lượt `m` (giá trị = client_msg_id của tin người dùng, hoặc id tin trả lời khi retry).
func (s *Service) releaseBusyOf(ctx context.Context, uid uuid.UUID, m store.ChatMessage) {
	s.releaseBusy(ctx, uid, m.ID.String())
	if m.ReplyTo != nil {
		if u, err := store.New(s.Pool).GetChatUserMessageOf(ctx, *m.ReplyTo); err == nil && u.ClientMsgID != nil {
			s.releaseBusy(ctx, uid, u.ClientMsgID.String())
		}
	}
}
