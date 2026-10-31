package mailin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/mail"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

const (
	KindIMAP = "IMAP"
	KindSMTP = "SMTP"
	// JobKind là loại việc nền của `Đồng bộ ngay`.
	JobKind = "imap.sync"

	defaultMailbox = "INBOX"
	minPoll        = 10
	maxPoll        = 3600
)

// Sealer là phần của platform/crypto mà cấu hình cần (AES-GCM, cùng khoá chủ với khoá nhà cung cấp LLM).
type Sealer interface {
	Encrypt(plain, aad []byte) ([]byte, error)
	Decrypt(blob, aad []byte) ([]byte, error)
}

// SMTPInfo là cấu hình SMTP của máy chủ (`SMTP_*`), chỉ đọc.
type SMTPInfo struct {
	Host     string
	Port     int
	Security string
	From     string
}

// Service là phần cấu hình / kiểm thử / đồng bộ ngay chạy ở gateway.
type Service struct {
	Pool        *pgxpool.Pool
	Cipher      Sealer
	Redis       *appredis.Client
	Jobs        *jobs.Service
	Production  bool
	SMTP        SMTPInfo
	TestTimeout time.Duration // 0 → 10 giây (SRS 4.5.2)
	MinSync     time.Duration // `SYNC_NOW_MIN_INTERVAL`; 0 → 30 giây
	Now         func() time.Time
}

// Public là phần không bí mật của cấu hình IMAP.
type Public struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Security    string `json:"security"`
	Username    string `json:"username"`
	Mailbox     string `json:"mailbox"`
	PollSeconds int    `json:"poll_seconds"`
}

// View là phản hồi `GET …/integrations/{kind}`: không bao giờ có mật khẩu hay đuôi mật khẩu.
type View struct {
	Kind       string     `json:"kind"`
	Configured bool       `json:"configured"`
	Enabled    bool       `json:"enabled"`
	Status     string     `json:"status"` // CONNECTED | NOT_CONNECTED | ERROR | PAUSED
	Config     any        `json:"config"`
	ReadOnly   bool       `json:"read_only"`
	LastOKAt   *time.Time `json:"last_ok_at"`
	LastSyncAt *time.Time `json:"last_sync_at"`
	LastError  *string    `json:"last_error"`
	ErrorSince *time.Time `json:"error_since"` // từ lúc nào lỗi ("Không kết nối được từ 09:12")
	Version    int32      `json:"version"`
}

// IMAPState là `integration_configs.state` của IMAP.
type IMAPState struct {
	UIDValidity uint32     `json:"uidvalidity"`
	LastUID     uint32     `json:"last_uid"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
	FailCount   int        `json:"fail_count,omitempty"`
	NextAt      *time.Time `json:"next_at,omitempty"`
	ErrorSince  *time.Time `json:"error_since,omitempty"`
	Paused      bool       `json:"paused,omitempty"`
}

func aad(course uuid.UUID, kind string) []byte { return []byte(course.String() + ":" + kind) }

func viewOf(r store.IntegrationConfig) (View, error) {
	v := View{Kind: r.Kind, Configured: true, Enabled: r.Enabled, LastOKAt: r.LastOkAt, LastError: r.LastError, Version: r.Version}
	var pub Public
	if err := json.Unmarshal(r.ConfigPublic, &pub); err != nil {
		return View{}, fmt.Errorf("mailin: cấu hình hỏng: %w", err)
	}
	v.Config = pub
	var st IMAPState
	_ = json.Unmarshal(r.State, &st)
	v.LastSyncAt, v.ErrorSince = st.LastSyncAt, st.ErrorSince
	switch {
	case st.Paused:
		v.Status = "PAUSED"
	case r.LastError != nil:
		v.Status = "ERROR"
	case r.LastOkAt != nil:
		v.Status = "CONNECTED"
	default:
		v.Status = "NOT_CONNECTED"
	}
	return v, nil
}

// Get trả cấu hình đã che. IMAP chưa cấu hình → {configured:false}; SMTP lấy từ `SMTP_*` (chỉ đọc); kind khác → 404.
func (s *Service) Get(ctx context.Context, course uuid.UUID, kind string) (View, error) {
	switch kind {
	case KindSMTP:
		ok := s.SMTP.Host != ""
		st := "NOT_CONNECTED"
		if ok {
			st = "CONNECTED"
		}
		return View{Kind: KindSMTP, Configured: ok, Enabled: ok, Status: st, ReadOnly: true, Version: 1,
			Config: map[string]any{"host": s.SMTP.Host, "port": s.SMTP.Port, "security": s.SMTP.Security, "from": s.SMTP.From}}, nil
	case KindIMAP:
	default:
		return View{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	r, err := store.New(s.Pool).IntegrationGet(ctx, store.IntegrationGetParams{CourseID: course, Kind: kind})
	if errors.Is(err, pgx.ErrNoRows) {
		return View{Kind: kind, Status: "NOT_CONNECTED", Config: nil}, nil
	}
	if err != nil {
		return View{}, fmt.Errorf("mailin: đọc cấu hình: %w", err)
	}
	return viewOf(r)
}

// PutIn là thân `PUT …/integrations/imap`. Password rỗng = giữ cũ.
type PutIn struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Security    string `json:"security"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Mailbox     string `json:"mailbox"`
	PollSeconds int    `json:"poll_seconds"`
	Enabled     bool   `json:"enabled"`
	Version     int32  `json:"version"`
}

func (in *PutIn) validate(production bool) error {
	in.Host, in.Username, in.Mailbox = strings.TrimSpace(in.Host), strings.TrimSpace(in.Username), strings.TrimSpace(in.Mailbox)
	if in.Mailbox == "" {
		in.Mailbox = defaultMailbox
	}
	if in.PollSeconds == 0 {
		in.PollSeconds = 60
	}
	var fe []apierr.FieldError
	add := func(f, m string) { fe = append(fe, apierr.FieldError{Field: f, Code: "invalid", Message: m}) }
	if in.Host == "" || len(in.Host) > 255 || strings.ContainsAny(in.Host, " /\\") {
		add("host", "Máy chủ không hợp lệ.")
	}
	if in.Port < 1 || in.Port > 65535 {
		add("port", "Cổng từ 1 đến 65535.")
	}
	switch in.Security {
	case SecTLS, SecStartTLS:
	case SecNone:
		if production {
			add("security", "Không dùng kết nối không mã hoá ở môi trường thật.")
		}
	default:
		add("security", "Bảo mật là TLS, STARTTLS hoặc không.")
	}
	if in.Username == "" || len(in.Username) > 255 {
		add("username", "Tài khoản không hợp lệ.")
	}
	if len(in.Mailbox) > 255 {
		add("mailbox", "Tên hộp thư quá dài.")
	}
	if in.PollSeconds < minPoll || in.PollSeconds > maxPoll {
		add("poll_seconds", "Chu kỳ từ 10 đến 3.600 giây.")
	}
	if len(in.Password) > 1024 {
		add("password", "Mật khẩu quá dài.")
	}
	if len(fe) > 0 {
		return apierr.Validation(fe...)
	}
	return nil
}

// Put lưu cấu hình IMAP (chỉ ADMIN ở tầng route): `config_public` không bí mật + `config_enc` (mật khẩu, AES-GCM). Lần đầu cần mật khẩu; khoá lạc quan `version` (409 VERSION_CONFLICT).
// audit_log ghi phần không bí mật.
func (s *Service) Put(ctx context.Context, actor, course uuid.UUID, in PutIn) (View, error) {
	if err := in.validate(s.Production); err != nil {
		return View{}, err
	}
	pub, _ := json.Marshal(Public{Host: in.Host, Port: in.Port, Security: in.Security, Username: in.Username, Mailbox: in.Mailbox, PollSeconds: in.PollSeconds})
	var enc []byte
	if in.Password != "" {
		sec, _ := json.Marshal(map[string]string{"password": in.Password})
		var err error
		if enc, err = s.Cipher.Encrypt(sec, aad(course, KindIMAP)); err != nil {
			return View{}, fmt.Errorf("mailin: mã hoá mật khẩu: %w", err)
		}
	}
	var out store.IntegrationConfig
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.IntegrationGet(ctx, store.IntegrationGetParams{CourseID: course, Kind: KindIMAP})
		var before []byte
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			if enc == nil {
				return apierr.Validation(apierr.FieldError{Field: "password", Code: "required", Message: "Nhập mật khẩu."})
			}
			if out, err = q.IntegrationInsert(ctx, store.IntegrationInsertParams{CourseID: course, Kind: KindIMAP, ConfigPublic: pub, ConfigEnc: enc, Enabled: in.Enabled}); err != nil {
				return fmt.Errorf("mailin: ghi cấu hình: %w", err)
			}
			if out.ID == uuid.Nil {
				return apierr.New(http.StatusConflict, apierr.VersionConflict)
			}
		case err != nil:
			return fmt.Errorf("mailin: đọc cấu hình: %w", err)
		default:
			before = cur.ConfigPublic
			if in.Version != cur.Version {
				return apierr.New(http.StatusConflict, apierr.VersionConflict)
			}
			if out, err = q.IntegrationUpdate(ctx, store.IntegrationUpdateParams{CourseID: course, Kind: KindIMAP, ConfigPublic: pub, ConfigEnc: enc, Enabled: in.Enabled, Version: in.Version}); errors.Is(err, pgx.ErrNoRows) {
				return apierr.New(http.StatusConflict, apierr.VersionConflict)
			} else if err != nil {
				return fmt.Errorf("mailin: ghi cấu hình: %w", err)
			}
		}
		after, _ := json.Marshal(map[string]any{"config": json.RawMessage(pub), "enabled": in.Enabled, "password_changed": enc != nil})
		if before == nil {
			before = []byte(`null`)
		}
		_, err = q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &actor, Entity: "integration_configs", EntityID: out.ID.String(), Action: "imap.put", Before: before, After: after})
		return err
	})
	if err != nil {
		return View{}, err
	}
	return viewOf(out)
}

// TestResult là phản hồi `POST …/integrations/imap/test`.
type TestResult struct {
	OK   bool    `json:"ok"`
	Kind *string `json:"kind,omitempty"`
}

// Test kết nối + đăng nhập + mở hộp thư trong ≤ 10 giây bằng cấu hình đã lưu; ghi `last_ok_at` / `last_error` (chữ ngắn, không bí mật).
func (s *Service) Test(ctx context.Context, course uuid.UUID) (TestResult, error) {
	st, _, err := loadSettings(ctx, store.New(s.Pool), s.Cipher, course)
	if err != nil {
		return TestResult{}, err
	}
	to := s.TestTimeout
	if to <= 0 {
		to = 10 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	sess, oerr := open(cctx, st)
	q := store.New(s.Pool)
	if oerr != nil {
		var ce *ConnError
		kind := KindUnreachable
		if errors.As(oerr, &ce) {
			kind = ce.Kind
		}
		patch := []byte(`{}`)
		if kind == KindAuth {
			patch = []byte(`{"paused":true}`)
		}
		if err := q.IntegrationResult(ctx, store.IntegrationResultParams{CourseID: course, Kind: KindIMAP, StatePatch: patch, Ok: false, LastError: &kind}); err != nil {
			return TestResult{}, fmt.Errorf("mailin: ghi kết quả: %w", err)
		}
		return TestResult{Kind: &kind}, nil
	}
	sess.close()
	if err := q.IntegrationResult(ctx, store.IntegrationResultParams{CourseID: course, Kind: KindIMAP, StatePatch: []byte(`{"paused":false,"fail_count":0}`), Ok: true}); err != nil {
		return TestResult{}, fmt.Errorf("mailin: ghi kết quả: %w", err)
	}
	return TestResult{OK: true}, nil
}

// loadSettings đọc + giải mã cấu hình IMAP của lớp. Chưa cấu hình → 404.
func loadSettings(ctx context.Context, q *store.Queries, c Sealer, course uuid.UUID) (Settings, store.IntegrationConfig, error) {
	r, err := q.IntegrationGet(ctx, store.IntegrationGetParams{CourseID: course, Kind: KindIMAP})
	if errors.Is(err, pgx.ErrNoRows) {
		return Settings{}, r, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return Settings{}, r, fmt.Errorf("mailin: đọc cấu hình: %w", err)
	}
	var pub Public
	if err := json.Unmarshal(r.ConfigPublic, &pub); err != nil {
		return Settings{}, r, fmt.Errorf("mailin: cấu hình hỏng: %w", err)
	}
	plain, err := c.Decrypt(r.ConfigEnc, aad(course, KindIMAP))
	if err != nil {
		return Settings{}, r, fmt.Errorf("mailin: giải mã mật khẩu: %w", err)
	}
	var sec struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(plain, &sec); err != nil {
		return Settings{}, r, fmt.Errorf("mailin: mật khẩu hỏng: %w", err)
	}
	return Settings{Host: pub.Host, Port: pub.Port, Security: pub.Security, Username: pub.Username, Password: sec.Password, Mailbox: pub.Mailbox}, r, nil
}

// SyncJob là phản hồi `202` của `sync-now`.
type SyncJob struct {
	JobID uuid.UUID `json:"job_id"`
}

// SyncPayload là tải của việc `imap.sync`.
type SyncPayload struct {
	CourseID uuid.UUID `json:"course_id"`
}

func runningKey(course uuid.UUID) string { return appredis.Key("imapsync", course.String(), "job") }
func limitKey(course uuid.UUID) string   { return appredis.Key("imapsync", course.String(), "rl") }

// SyncNow xếp việc `imap.sync` cho lớp: đang có một lần chạy → trả lại `job_id` đó (không chạy chồng); ≤ 1 lần mỗi `MinSync` / lớp (429 + retry_after).
func (s *Service) SyncNow(ctx context.Context, actor, course uuid.UUID) (SyncJob, error) {
	if _, _, err := loadSettings(ctx, store.New(s.Pool), s.Cipher, course); err != nil {
		return SyncJob{}, err
	}
	if v, err := s.Redis.Get(ctx, runningKey(course)).Result(); err == nil {
		if id, perr := uuid.Parse(v); perr == nil {
			return SyncJob{JobID: id}, nil
		}
	}
	min := s.MinSync
	if min <= 0 {
		min = 30 * time.Second
	}
	ok, err := s.Redis.SetNX(ctx, limitKey(course), "1", min).Result()
	if err != nil {
		return SyncJob{}, fmt.Errorf("mailin: giới hạn tần suất: %w", err)
	}
	if !ok {
		ttl, _ := s.Redis.TTL(ctx, limitKey(course)).Result()
		return SyncJob{}, apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(max(1, int(ttl.Seconds()+0.999)))
	}
	j, err := s.Jobs.Enqueue(ctx, actor, JobKind, SyncPayload{CourseID: course})
	if err != nil {
		_ = s.Redis.Del(ctx, limitKey(course)).Err()
		return SyncJob{}, fmt.Errorf("mailin: xếp việc đồng bộ: %w", err)
	}
	if err := s.Redis.Set(ctx, runningKey(course), j.ID.String(), 15*time.Minute).Err(); err != nil {
		return SyncJob{}, fmt.Errorf("mailin: đánh dấu đang chạy: %w", err)
	}
	return SyncJob{JobID: j.ID}, nil
}

// SMTPTest xếp một thư thử tới địa chỉ của người bấm (Q7); thư đi qua `mail_outbox` như mọi thư khác.
func (s *Service) SMTPTest(ctx context.Context, user uuid.UUID) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		u, err := store.New(tx).GetUser(ctx, user)
		if err != nil {
			return fmt.Errorf("mailin: đọc người dùng: %w", err)
		}
		_, _, err = mail.Enqueue(ctx, tx, mail.Message{To: u.Email, Template: "smtp_test", Payload: map[string]any{"full_name": u.FullName}, DedupeKey: "smtp_test:" + uuid.NewString()})
		if err != nil {
			return fmt.Errorf("mailin: xếp thư thử: %w", err)
		}
		return nil
	})
}

// InboundRow là một thư đã nhận (tab `Chưa khớp`, mục "Thư không xử lý được").
type InboundRow struct {
	ID           uuid.UUID  `json:"id"`
	FromEmail    string     `json:"from_email"`
	Subject      string     `json:"subject"`
	ReceivedAt   time.Time  `json:"received_at"`
	Status       string     `json:"status"`
	Reason       *string    `json:"reason"`
	SubmissionID *uuid.UUID `json:"submission_id"`
}

// InboundPage là một trang danh sách.
type InboundPage struct {
	Items      []InboundRow `json:"items"`
	NextCursor *string      `json:"next_cursor"`
}

// ListInbound trả thư đã nhận của lớp (lọc `status` ACCEPTED | IGNORED | FAILED), mới nhất trước, con trỏ (received_at, id).
func (s *Service) ListInbound(ctx context.Context, course uuid.UUID, status string, p httpx.PageParams) (InboundPage, error) {
	if status != "" && status != StatusAccepted && status != StatusIgnored && status != StatusFailed {
		return InboundPage{}, apierr.Validation(apierr.FieldError{Field: "status", Code: "enum", Message: "status là ACCEPTED, IGNORED hoặc FAILED."})
	}
	arg := store.MailInboundListParams{CourseID: course, PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if status != "" {
		arg.Status = &status
	}
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := store.New(s.Pool).MailInboundList(ctx, arg)
	if err != nil {
		return InboundPage{}, fmt.Errorf("mailin: danh sách thư: %w", err)
	}
	out := InboundPage{Items: make([]InboundRow, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].ReceivedAt, rows[len(rows)-1].ID.String())
		out.NextCursor = &c
	}
	for _, r := range rows {
		out.Items = append(out.Items, InboundRow{ID: r.ID, FromEmail: r.FromEmail, Subject: r.Subject, ReceivedAt: r.ReceivedAt, Status: r.Status, Reason: r.Reason, SubmissionID: r.SubmissionID})
	}
	return out, nil
}
