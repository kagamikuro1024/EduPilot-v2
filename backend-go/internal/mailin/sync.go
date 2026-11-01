package mailin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/notify"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/submission"
)

// Trạng thái / lý do của `mail_inbound`.
const (
	StatusAccepted = "ACCEPTED"
	StatusIgnored  = "IGNORED"
	StatusFailed   = "FAILED"

	ReasonAutoReply     = "AUTO_REPLY"
	ReasonNoAttachment  = "NO_ATTACHMENT"
	ReasonTooMany       = "TOO_MANY_ATTACHMENTS"
	ReasonAttachRejects = "ATTACHMENT_REJECTED"
	ReasonParseError    = "PARSE_ERROR"

	// TypeMailReceipt: chuông gửi sinh viên được khớp khi bài nộp qua thư (để thư giả mạo bị phát hiện).
	TypeMailReceipt = "SUBMISSION_MAIL_RECEIVED"

	maxPerSync = 500 // một lượt đồng bộ; phần còn lại để lượt sau (không giữ khoá quá lâu)
	lockTTL    = 2 * time.Minute
)

// Syncer đọc hộp thư IMAP của lớp và đưa thư vào `submission.Ingest` (worker).
type Syncer struct {
	Pool   *pgxpool.Pool
	Redis  *appredis.Client
	Cipher Sealer
	Blob   submission.BlobAPI
	Notify *notify.Service
	Log    *slog.Logger

	TrustedAuthservID string        // IMAP_TRUSTED_AUTHSERV_ID; rỗng → mọi thư UNMATCHED SENDER_UNVERIFIED
	SystemFrom        string        // địa chỉ gửi của hệ thống (thư của chính nó → AUTO_REPLY)
	MaxBytes          int64         // ASSIGN_MAX_BYTES
	DefaultPoll       time.Duration // IMAP_POLL_SECONDS khi lớp không đặt
	BackoffMax        time.Duration // IMAP_BACKOFF_MAX
	ConnTimeout       time.Duration // 0 → 30 giây
	Now               func() time.Time
}

func (y *Syncer) now() time.Time {
	if y.Now != nil {
		return y.Now()
	}
	return time.Now()
}

func (y *Syncer) maxBytes() int64 {
	if y.MaxBytes > 0 {
		return y.MaxBytes
	}
	return 20 << 20
}

// Result là kết quả một lượt đồng bộ (SRS 4.5.5).
type Result struct {
	Received  int `json:"received"`
	Created   int `json:"created"`
	Matched   int `json:"matched"`
	Unmatched int `json:"unmatched"`
	Ignored   int `json:"ignored"`
	Failed    int `json:"failed"`
}

// ErrBusy: lớp đang có một poller khác giữ khoá `ep:lock:imap:{course}`.
var ErrBusy = errors.New("mailin: lớp đang được đồng bộ")

func lockKey(course uuid.UUID) string { return appredis.Key("lock", "imap", course.String()) }

// lock giữ khoá một poller duy nhất cho lớp (gia hạn nền tới khi gọi unlock).
func (y *Syncer) lock(ctx context.Context, course uuid.UUID) (func(), error) {
	token := uuid.NewString()
	ok, err := y.Redis.SetNX(ctx, lockKey(course), token, lockTTL).Result()
	if err != nil {
		return nil, fmt.Errorf("mailin: khoá poller: %w", err)
	}
	if !ok {
		return nil, ErrBusy
	}
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(lockTTL / 3)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if v, err := y.Redis.Get(context.WithoutCancel(ctx), lockKey(course)).Result(); err == nil && v == token {
					_ = y.Redis.PExpire(context.WithoutCancel(ctx), lockKey(course), lockTTL).Err()
				}
			}
		}
	}()
	return func() {
		close(stop)
		bg := context.WithoutCancel(ctx)
		if v, err := y.Redis.Get(bg, lockKey(course)).Result(); err == nil && v == token {
			_ = y.Redis.Del(bg, lockKey(course)).Err()
		}
	}, nil
}

// Tick đồng bộ mọi lớp đã tới hạn (mỗi lớp một lượt, tuần tự). Lỗi của một lớp không chặn lớp khác.
func (y *Syncer) Tick(ctx context.Context) {
	rows, err := store.New(y.Pool).IntegrationDue(ctx, store.IntegrationDueParams{Now: y.now(), PageLimit: 200})
	if err != nil {
		y.Log.ErrorContext(ctx, "mailin: không đọc được lớp tới hạn", "error", err.Error())
		return
	}
	for _, r := range rows {
		if ctx.Err() != nil {
			return
		}
		if _, err := y.SyncCourse(ctx, r.CourseID, nil); err != nil && !errors.Is(err, ErrBusy) {
			y.Log.WarnContext(ctx, "mailin: đồng bộ lớp lỗi", "course_id", r.CourseID.String(), "error", err.Error())
		}
	}
}

// Run chạy vòng poll tới khi ctx kết thúc; mỗi `every` kiểm các lớp tới hạn (chu kỳ riêng của lớp nằm ở `state.next_at`).
func (y *Syncer) Run(ctx context.Context, every time.Duration) error {
	if every <= 0 {
		every = 5 * time.Second
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			y.Tick(ctx)
		}
	}
}

// SyncCourse đọc thư mới của một lớp. `report(done, total)` (tuỳ chọn) báo tiến độ. Lỗi kết nối được ghi (last_error + lùi luỹ tiến / tạm dừng) và trả về.
func (y *Syncer) SyncCourse(ctx context.Context, course uuid.UUID, report func(done, total int)) (Result, error) {
	unlock, err := y.lock(ctx, course)
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	q := store.New(y.Pool)
	st, row, err := loadSettings(ctx, q, y.Cipher, course)
	if err != nil {
		return Result{}, err
	}
	var state IMAPState
	_ = json.Unmarshal(row.State, &state)
	var pub Public
	_ = json.Unmarshal(row.ConfigPublic, &pub)
	poll := time.Duration(pub.PollSeconds) * time.Second
	if poll <= 0 {
		poll = y.DefaultPoll
	}
	if poll <= 0 {
		poll = time.Minute
	}
	to := y.ConnTimeout
	if to <= 0 {
		to = 30 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, to)
	sess, oerr := open(cctx, st)
	cancel()
	if oerr != nil {
		return Result{}, y.fail(ctx, q, course, state, poll, oerr)
	}
	defer sess.close()

	if state.UIDValidity != sess.uidValidity {
		state.UIDValidity, state.LastUID = sess.uidValidity, 0 // đổi UIDVALIDITY: quét lại; UNIQUE (course, uidvalidity, uid) chống nhập đôi
	}
	uids, err := sess.uidsAfter(state.LastUID)
	if err != nil {
		return Result{}, y.fail(ctx, q, course, state, poll, &ConnError{Kind: KindUnreachable, Err: err})
	}
	if len(uids) > maxPerSync {
		uids = uids[:maxPerSync]
	}
	env, err := submission.LoadEnv(ctx, q, course, "IMAP")
	if err != nil {
		return Result{}, err
	}
	var res Result
	for i, uid := range uids {
		md, err := sess.fetchOne(uid)
		if err != nil {
			y.save(ctx, q, course, state)
			return res, y.fail(ctx, q, course, state, poll, &ConnError{Kind: KindUnreachable, Err: err})
		}
		if err := y.process(ctx, env, course, sess.uidValidity, md, &res); err != nil {
			y.save(ctx, q, course, state) // thư này chưa tính là đã xử lý: lượt sau thử lại
			return res, fmt.Errorf("mailin: xử lý thư %d: %w", uid, err)
		}
		state.LastUID = uid
		if (i+1)%20 == 0 {
			y.save(ctx, q, course, state)
		}
		if report != nil {
			report(i+1, len(uids))
		}
	}
	now := y.now()
	state.LastSyncAt, state.FailCount, state.ErrorSince = &now, 0, nil
	next := now.Add(poll)
	state.NextAt = &next
	patch, _ := json.Marshal(map[string]any{"uidvalidity": state.UIDValidity, "last_uid": state.LastUID, "last_sync_at": now, "fail_count": 0, "next_at": next, "error_since": nil, "paused": false})
	if err := q.IntegrationResult(ctx, store.IntegrationResultParams{CourseID: course, Kind: KindIMAP, StatePatch: patch, Ok: true}); err != nil {
		return res, fmt.Errorf("mailin: ghi trạng thái: %w", err)
	}
	return res, nil
}

// save ghi vị trí đã đọc tới đâu (không đổi trạng thái lỗi / hạn poll).
func (y *Syncer) save(ctx context.Context, q *store.Queries, course uuid.UUID, st IMAPState) {
	patch, _ := json.Marshal(map[string]any{"uidvalidity": st.UIDValidity, "last_uid": st.LastUID})
	if err := q.IntegrationResult(context.WithoutCancel(ctx), store.IntegrationResultParams{CourseID: course, Kind: KindIMAP, StatePatch: patch, Ok: false, LastError: nil}); err != nil {
		y.Log.WarnContext(ctx, "mailin: không ghi được vị trí đọc", "course_id", course.String(), "error", err.Error())
	}
}

// fail ghi lỗi kết nối: lùi luỹ tiến (poll × 2^n ≤ BackoffMax); sai mật khẩu → tạm dừng poll (cần nhập lại), không thử dồn dập.
func (y *Syncer) fail(ctx context.Context, q *store.Queries, course uuid.UUID, st IMAPState, poll time.Duration, cause error) error {
	var ce *ConnError
	kind := KindUnreachable
	if errors.As(cause, &ce) {
		kind = ce.Kind
	}
	now := y.now()
	if st.FailCount == 0 || st.ErrorSince == nil {
		st.ErrorSince = &now
	}
	st.FailCount++
	max := y.BackoffMax
	if max <= 0 {
		max = 15 * time.Minute
	}
	wait := poll
	for i := 1; i < st.FailCount && wait < max; i++ {
		wait *= 2
	}
	wait = min(wait, max)
	next := now.Add(wait)
	p := map[string]any{"uidvalidity": st.UIDValidity, "last_uid": st.LastUID, "fail_count": st.FailCount, "next_at": next, "error_since": st.ErrorSince}
	if kind == KindAuth {
		p["paused"] = true
	}
	patch, _ := json.Marshal(p)
	if err := q.IntegrationResult(context.WithoutCancel(ctx), store.IntegrationResultParams{CourseID: course, Kind: KindIMAP, StatePatch: patch, Ok: false, LastError: &kind}); err != nil {
		y.Log.WarnContext(ctx, "mailin: không ghi được lỗi kết nối", "course_id", course.String(), "error", err.Error())
	}
	return cause
}

type memReader struct{ b []byte }

func (m memReader) GetRange(_ context.Context, _ string, off, n int64) ([]byte, error) {
	if off >= int64(len(m.b)) {
		return nil, nil
	}
	return m.b[off:min(int64(len(m.b)), off+n)], nil
}

var errDuplicate = errors.New("mailin: thư đã xử lý")

// process xử lý MỘT thư (SRS 4.5.4): lỗi trả về chỉ khi hạ tầng hỏng (thư chưa được tính); thư xấu thành dòng IGNORED / FAILED.
func (y *Syncer) process(ctx context.Context, env submission.Env, course uuid.UUID, uv uint32, md MessageData, res *Result) error {
	res.Received++
	p, perr := Parse(md.Raw, y.maxBytes())
	if perr != nil {
		res.Failed++
		return y.record(ctx, course, uv, md, p, StatusFailed, ReasonParseError, nil)
	}
	if p.MessageID != "" {
		if seen, err := store.New(y.Pool).MailInboundSeen(ctx, store.MailInboundSeenParams{CourseID: course, MessageID: &p.MessageID}); err != nil {
			return fmt.Errorf("mailin: kiểm thư đã nhận: %w", err)
		} else if seen {
			return nil // thư đã nhận ở UIDVALIDITY cũ (hộp thư tạo lại): không nhập lại
		}
	}
	switch {
	case p.AutoReply || (y.SystemFrom != "" && p.From == y.SystemFrom):
		return y.ignored(ctx, course, uv, md, p, ReasonAutoReply, res)
	case p.Attachments == 0:
		return y.ignored(ctx, course, uv, md, p, ReasonNoAttachment, res)
	case len(p.Valid) >= 2:
		return y.ignored(ctx, course, uv, md, p, ReasonTooMany, res)
	case len(p.Valid) == 0:
		return y.ignored(ctx, course, uv, md, p, ReasonAttachRejects, res)
	}
	at := p.Valid[0]
	if !at.Kind.Sniff(ctx, memReader{at.Data}, "", int64(len(at.Data))) {
		return y.ignored(ctx, course, uv, md, p, ReasonAttachRejects, res)
	}
	sum := sha256.Sum256(at.Data)
	safe := submission.SafeName(at.Name)
	key := fmt.Sprintf("courses/%s/submissions/%s/%s", course, uuid.New(), safe)
	if err := y.Blob.Put(ctx, key, bytes.NewReader(at.Data), int64(len(at.Data)), at.Kind.Mime); err != nil {
		return fmt.Errorf("mailin: lưu đính kèm: %w", err)
	}
	when := md.Date.UTC()
	raw := submission.Raw{Source: "IMAP", ExternalRef: strconv.FormatUint(uint64(uv), 10) + ":" + strconv.FormatUint(uint64(md.UID), 10), ExternalTS: when, FromEmail: p.From,
		SenderVerified: SenderVerified(p.AuthResults, p.From, y.TrustedAuthservID), Subject: p.Subject, FileName: safe, BlobKey: key, FileMime: at.Kind.Mime,
		SHA256: hex.EncodeToString(sum[:]), FileSize: int64(len(at.Data)), ReceivedAt: when, SubmittedAt: &when}
	var out submission.Outcome
	var announce []notify.Row
	err := pgx.BeginFunc(ctx, y.Pool, func(tx pgx.Tx) error {
		var err error
		if out, err = submission.Ingest(ctx, tx, env, raw, nil); err != nil {
			return err
		}
		sid := out.ID
		var subID *uuid.UUID
		if !out.Deduped {
			subID = &sid
		}
		if err := insertInbound(ctx, tx, course, uv, md, p, StatusAccepted, "", subID); err != nil {
			return err
		}
		if out.Matched && !out.Deduped {
			row, err := y.receipt(ctx, store.New(tx), course, out.ID, when)
			if err != nil {
				return err
			}
			announce = append(announce, row...)
		}
		return nil
	})
	if err != nil {
		y.drop(ctx, key)
		if errors.Is(err, errDuplicate) {
			res.Received--
			return nil
		}
		return err
	}
	if out.Deduped {
		y.drop(ctx, key)
	} else {
		res.Created++
		if out.Matched {
			res.Matched++
		} else {
			res.Unmatched++
		}
	}
	if y.Notify != nil {
		y.Notify.Announce(ctx, announce...)
	}
	return nil
}

func (y *Syncer) ignored(ctx context.Context, course uuid.UUID, uv uint32, md MessageData, p Parsed, reason string, res *Result) error {
	res.Ignored++
	return y.record(ctx, course, uv, md, p, StatusIgnored, reason, nil)
}

func (y *Syncer) drop(ctx context.Context, key string) {
	if err := y.Blob.Delete(context.WithoutCancel(ctx), key); err != nil {
		y.Log.WarnContext(ctx, "mailin: không xoá được tệp thừa", "error", err.Error())
	}
}

// record ghi dòng mail_inbound của thư không thành bài nộp.
func (y *Syncer) record(ctx context.Context, course uuid.UUID, uv uint32, md MessageData, p Parsed, status, reason string, sub *uuid.UUID) error {
	err := pgx.BeginFunc(ctx, y.Pool, func(tx pgx.Tx) error { return insertInbound(ctx, tx, course, uv, md, p, status, reason, sub) })
	if errors.Is(err, errDuplicate) {
		return nil
	}
	return err
}

func insertInbound(ctx context.Context, tx pgx.Tx, course uuid.UUID, uv uint32, md MessageData, p Parsed, status, reason string, sub *uuid.UUID) error {
	var rs *string
	if reason != "" {
		rs = &reason
	}
	var mid *string
	if p.MessageID != "" {
		mid = &p.MessageID
	}
	_, err := store.New(tx).MailInboundInsert(ctx, store.MailInboundInsertParams{CourseID: course, Uidvalidity: int64(uv), MessageUid: int64(md.UID), MessageID: mid, FromEmail: p.From, Subject: clip(p.Subject, 500),
		ReceivedAt: md.Date.UTC(), Status: status, Reason: rs, SubmissionID: sub})
	if errors.Is(err, pgx.ErrNoRows) {
		return errDuplicate
	}
	if err != nil {
		return fmt.Errorf("mailin: ghi thư nhận: %w", err)
	}
	return nil
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// receipt tạo chuông cho sinh viên được khớp: "Đã nhận bài nộp qua thư lúc HH:mm dd/MM" (để thư giả mạo bị phát hiện).
func (y *Syncer) receipt(ctx context.Context, q *store.Queries, course, sub uuid.UUID, at time.Time) ([]notify.Row, error) {
	r, err := q.SubmissionStudentOf(ctx, store.SubmissionStudentOfParams{CourseID: course, ID: sub})
	if err != nil || r.StudentID == nil {
		return nil, fmt.Errorf("mailin: sinh viên của bài nộp: %w", err)
	}
	link := "/assignments/" + r.AssignmentID.String()
	row, created, err := notify.SendTx(ctx, q, notify.Spec{UserID: *r.StudentID, CourseID: &course, Type: TypeMailReceipt, Title: "Đã nhận bài nộp qua thư lúc " + at.In(time.FixedZone("ICT", 7*3600)).Format("15:04 02/01"),
		Link: &link, DedupeKey: "imap.receipt:" + sub.String()})
	if err != nil {
		return nil, err
	}
	if !created {
		return nil, nil
	}
	return []notify.Row{row}, nil
}

// Register gắn việc `imap.sync` (Đồng bộ ngay) vào jobs.Runner: tiến độ "Đã nhận n/total thư", kết quả {received, created, matched, unmatched, ignored}.
func (y *Syncer) Register(r *jobs.Runner) {
	r.Register(JobKind, func(ctx context.Context, j jobs.JobCtx) (any, error) {
		var p SyncPayload
		if err := json.Unmarshal(j.Payload, &p); err != nil {
			return nil, &jobs.UserError{Code: "VALIDATION_FAILED", Message: "Yêu cầu đồng bộ không đọc được."}
		}
		defer y.Redis.Del(context.WithoutCancel(ctx), runningKey(p.CourseID)) //nolint:errcheck // khoá tự hết hạn
		var res Result
		var err error
		deadline := time.Now().Add(time.Minute) // poller định kỳ đang giữ khoá: chờ tối đa 1 phút rồi đọc
		for {
			res, err = y.SyncCourse(ctx, p.CourseID, func(done, total int) { j.Progress(done * 100 / max(total, 1)) })
			if !errors.Is(err, ErrBusy) || time.Now().After(deadline) {
				break
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Second):
			}
		}
		var ce *ConnError
		if errors.As(err, &ce) {
			return nil, &jobs.UserError{Code: "IMAP_" + ce.Kind, Message: "Không đồng bộ được hộp thư."}
		}
		if err != nil {
			return nil, err
		}
		return res, nil
	})
}
