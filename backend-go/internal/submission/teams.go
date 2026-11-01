package submission

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/submission/source"
)

const (
	// KindTeamsSync là loại việc nền `Đồng bộ từ Teams`.
	KindTeamsSync = "teams.sync"
	teamsSource   = "TEAMS"
	teamsLockTTL  = 5 * time.Minute
)

// Opener là phần của platform/crypto mà việc đọc client_secret cần.
type Opener interface {
	Decrypt(blob, aad []byte) ([]byte, error)
}

// TeamsSync đưa bài nộp từ Teams vào đường ống chung `Ingest` (SRS 4.11.3).
type TeamsSync struct {
	Pool        *pgxpool.Pool
	Redis       *appredis.Client
	Blob        BlobAPI
	Cipher      Opener
	Jobs        *jobs.Service
	Log         *slog.Logger
	Enabled     bool          // TEAMS_ENABLED
	Source      source.Config // GRAPH_* / TEAMS_*
	Timeout     time.Duration // TEAMS_SYNC_TIMEOUT (0 → 120 giây)
	MinInterval time.Duration // SYNC_NOW_MIN_INTERVAL (0 → 30 giây)
	Sleep       source.Sleeper
	MaxBytes    int64
}

// TeamsPayload là tải của việc `teams.sync`.
type TeamsPayload struct {
	CourseID     uuid.UUID `json:"course_id"`
	AssignmentID uuid.UUID `json:"assignment_id"`
}

// TeamsJob là phản hồi `202`.
type TeamsJob struct {
	JobID uuid.UUID `json:"job_id"`
}

// TeamsResult là kết quả đợt đồng bộ (SRS 4.11.3).
type TeamsResult struct {
	Fetched   int              `json:"fetched"`
	Created   int              `json:"created"`
	Deduped   int              `json:"deduped"`
	Unmatched int              `json:"unmatched"`
	Failed    []source.Failure `json:"failed"`
}

func teamsAAD(course uuid.UUID) []byte { return []byte(course.String() + ":TEAMS") }

func teamsRunningKey(asg uuid.UUID) string { return appredis.Key("teamssync", asg.String(), "job") }
func teamsLimitKey(asg uuid.UUID) string   { return appredis.Key("teamssync", asg.String(), "rl") }
func teamsLockKey(asg uuid.UUID) string    { return appredis.Key("lock", "teams", asg.String()) }

func (s *TeamsSync) credentials(ctx context.Context, q *store.Queries, course uuid.UUID) (source.Credentials, error) {
	r, err := q.IntegrationGet(ctx, store.IntegrationGetParams{CourseID: course, Kind: "TEAMS"})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !r.Enabled) {
		return source.Credentials{}, apierr.New(http.StatusConflict, apierr.TeamsNotConfigured)
	}
	if err != nil {
		return source.Credentials{}, fmt.Errorf("teams: đọc cấu hình: %w", err)
	}
	var pub struct {
		TenantID string `json:"tenant_id"`
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(r.ConfigPublic, &pub); err != nil {
		return source.Credentials{}, fmt.Errorf("teams: cấu hình hỏng: %w", err)
	}
	plain, err := s.Cipher.Decrypt(r.ConfigEnc, teamsAAD(course))
	if err != nil {
		return source.Credentials{}, fmt.Errorf("teams: giải mã client secret: %w", err)
	}
	var sec struct {
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(plain, &sec); err != nil {
		return source.Credentials{}, fmt.Errorf("teams: client secret hỏng: %w", err)
	}
	return source.Credentials{Tenant: pub.TenantID, ClientID: pub.ClientID, ClientSecret: sec.ClientSecret}, nil
}

// Enqueue xếp việc đồng bộ cho một bài (TA, TEACHER): TEAMS_ENABLED; kênh TEAMS (409 CHANNEL_NOT_ALLOWED); đã cấu hình; đang chạy → trả job_id cũ;
// ≤ 1 lần / MinInterval / bài (429 + retry_after).
func (s *TeamsSync) Enqueue(ctx context.Context, actor, course, asg uuid.UUID) (TeamsJob, error) {
	if !s.Enabled {
		return TeamsJob{}, apierr.New(http.StatusConflict, apierr.TeamsDisabled)
	}
	q := store.New(s.Pool)
	a, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: asg})
	if errors.Is(err, pgx.ErrNoRows) {
		return TeamsJob{}, apierr.New(http.StatusNotFound, apierr.AssignmentNotFound)
	}
	if err != nil {
		return TeamsJob{}, fmt.Errorf("teams: đọc bài tập: %w", err)
	}
	if a.TeamsClassID == nil || a.TeamsAssignmentID == nil || !hasChannel(a.Channels, "TEAMS") {
		return TeamsJob{}, apierr.New(http.StatusConflict, apierr.ChannelNotAllowed)
	}
	if _, err := s.credentials(ctx, q, course); err != nil {
		return TeamsJob{}, err
	}
	if v, err := s.Redis.Get(ctx, teamsRunningKey(asg)).Result(); err == nil {
		if id, perr := uuid.Parse(v); perr == nil {
			return TeamsJob{JobID: id}, nil
		}
	}
	min := s.MinInterval
	if min <= 0 {
		min = 30 * time.Second
	}
	ok, err := s.Redis.SetNX(ctx, teamsLimitKey(asg), "1", min).Result()
	if err != nil {
		return TeamsJob{}, fmt.Errorf("teams: giới hạn tần suất: %w", err)
	}
	if !ok {
		ttl, _ := s.Redis.TTL(ctx, teamsLimitKey(asg)).Result()
		return TeamsJob{}, apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(max(1, int(ttl.Seconds()+0.999)))
	}
	j, err := s.Jobs.Enqueue(ctx, actor, KindTeamsSync, TeamsPayload{CourseID: course, AssignmentID: asg})
	if err != nil {
		_ = s.Redis.Del(ctx, teamsLimitKey(asg)).Err()
		return TeamsJob{}, fmt.Errorf("teams: xếp việc: %w", err)
	}
	if err := s.Redis.Set(ctx, teamsRunningKey(asg), j.ID.String(), 15*time.Minute).Err(); err != nil {
		return TeamsJob{}, fmt.Errorf("teams: đánh dấu đang chạy: %w", err)
	}
	return TeamsJob{JobID: j.ID}, nil
}

func hasChannel(list []string, c string) bool {
	for _, x := range list {
		if x == c {
			return true
		}
	}
	return false
}

func (s *TeamsSync) adapter() *source.Teams {
	cfg := s.Source
	if cfg.MaxFileBytes == 0 {
		cfg.MaxFileBytes = s.MaxBytes
	}
	return &source.Teams{Cfg: cfg, Sleep: s.Sleep}
}

// lock giữ khoá một đợt cho bài (token riêng, nhả khi xong).
func (s *TeamsSync) lock(ctx context.Context, asg uuid.UUID) (func(), error) {
	token := uuid.NewString()
	ok, err := s.Redis.SetNX(ctx, teamsLockKey(asg), token, teamsLockTTL).Result()
	if err != nil {
		return nil, fmt.Errorf("teams: khoá đợt: %w", err)
	}
	if !ok {
		return nil, errBusy
	}
	return func() {
		bg := context.WithoutCancel(ctx)
		if v, err := s.Redis.Get(bg, teamsLockKey(asg)).Result(); err == nil && v == token {
			_ = s.Redis.Del(bg, teamsLockKey(asg)).Err()
		}
	}, nil
}

var errBusy = errors.New("teams: bài đang được đồng bộ")

// Run chạy MỘT đợt: khoá `ep:lock:teams:{assignment}`, chuỗi Graph, mỗi tệp → blob + Ingest (kênh TEAMS). Mọi lỗi dừng đợt giữ nguyên bài đã về (idempotent, chạy lại được).
func (s *TeamsSync) Run(ctx context.Context, course, asg uuid.UUID, report func(done, total int)) (TeamsResult, error) {
	res := TeamsResult{Failed: []source.Failure{}}
	unlock, err := s.lock(ctx, asg)
	if err != nil {
		return res, err
	}
	defer unlock()
	to := s.Timeout
	if to <= 0 {
		to = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	q := store.New(s.Pool)
	a, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: asg})
	if err != nil {
		return res, fmt.Errorf("teams: đọc bài tập: %w", err)
	}
	if a.TeamsClassID == nil || a.TeamsAssignmentID == nil {
		return res, apierr.New(http.StatusConflict, apierr.ChannelNotAllowed)
	}
	cred, err := s.credentials(ctx, q, course)
	if err != nil {
		return res, err
	}
	env, err := LoadEnv(ctx, q, course, "TEAMS")
	if err != nil {
		return res, err
	}
	maxBytes := s.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 20 << 20
	}
	badFiles := 0
	onFile := func(ctx context.Context, f source.Fetched) error {
		res.Fetched++
		kind, ok := KindOf(strings.ToLower(path.Ext(f.FileName)))
		switch {
		case !ok:
			res.Failed = append(res.Failed, source.Failure{SubmissionID: f.SubmissionID, Code: apierr.FileTypeNotAllowed})
			badFiles++
			return nil
		case len(f.Data) == 0 || int64(len(f.Data)) > maxBytes || !kind.Sniff(ctx, memBytes(f.Data), "", int64(len(f.Data))):
			res.Failed = append(res.Failed, source.Failure{SubmissionID: f.SubmissionID, Code: apierr.FileInvalid})
			badFiles++
			return nil
		}
		sum := sha256.Sum256(f.Data)
		safe := SafeName(f.FileName)
		key := fmt.Sprintf("courses/%s/submissions/%s/%s", course, uuid.New(), safe)
		if err := s.Blob.Put(ctx, key, bytes.NewReader(f.Data), int64(len(f.Data)), kind.Mime); err != nil {
			return fmt.Errorf("teams: lưu tệp: %w", err)
		}
		aid := asg
		submitted := f.SubmittedAt
		var out Outcome
		err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
			var err error
			out, err = Ingest(ctx, tx, env, Raw{Source: teamsSource, ExternalRef: f.SubmissionID, ExternalTS: submitted, FromEmail: f.Email, SenderVerified: true, FileName: safe, BlobKey: key,
				FileMime: kind.Mime, SHA256: hex.EncodeToString(sum[:]), FileSize: int64(len(f.Data)), SubmittedAt: &submitted, AssignmentID: &aid}, nil)
			return err
		})
		if err != nil || out.Deduped {
			if derr := s.Blob.Delete(context.WithoutCancel(ctx), key); derr != nil && s.Log != nil {
				s.Log.WarnContext(ctx, "teams: không xoá được tệp thừa", "error", derr.Error())
			}
			return err
		}
		res.Created++
		if !out.Matched {
			res.Unmatched++
		}
		return nil
	}
	rep, err := s.adapter().Sync(ctx, cred, *a.TeamsClassID, *a.TeamsAssignmentID, onFile, report)
	res.Failed = append(res.Failed, rep.Failed...)
	res.Deduped = max(0, res.Fetched-badFiles-res.Created)
	return res, err
}

type memBytes []byte

func (m memBytes) GetRange(_ context.Context, _ string, off, n int64) ([]byte, error) {
	if off >= int64(len(m)) {
		return nil, nil
	}
	return m[off:min(int64(len(m)), off+n)], nil
}

// teamsErr đổi lỗi của adapter / đợt thành UserError có mã + câu tiếng Việt cho người bấm.
func teamsErr(err error) error {
	var se *source.Error
	switch {
	case errors.As(err, &se):
		msg := map[string]string{
			source.CodeAuthFailed:      "Không đăng nhập được vào Teams. Kiểm tra lại Tenant, Client ID và Client secret.",
			source.CodeConsentRequired: "Cần quản trị Microsoft 365 của trường cấp quyền đọc bài tập và danh sách lớp.",
			source.CodeResponseInvalid: "Teams trả dữ liệu không đọc được. Thử lại sau ít phút.",
			source.CodeUnreachable:     "Không kết nối được Teams. Các bài đã lấy vẫn giữ nguyên.",
			source.CodeTimeout:         "Teams phản hồi quá chậm. Các bài đã lấy vẫn giữ nguyên.",
		}[se.Code]
		if msg == "" {
			msg = "Không đồng bộ được từ Teams."
		}
		return &jobs.UserError{Code: se.Code, Message: msg}
	case errors.Is(err, errBusy):
		return &jobs.UserError{Code: "BUSY", Message: "Bài này đang được đồng bộ."}
	}
	var ae *apierr.Error
	if errors.As(err, &ae) {
		return &jobs.UserError{Code: ae.Code, Message: ae.Message}
	}
	return err
}

// Register gắn việc `teams.sync`: tiến độ "Đã lấy k/n bài" (jobs.result {fetched, total} khi còn chạy), kết quả {fetched, created, deduped, unmatched, failed[]}.
func (s *TeamsSync) Register(r *jobs.Runner) {
	r.Register(KindTeamsSync, func(ctx context.Context, j jobs.JobCtx) (any, error) {
		var p TeamsPayload
		if err := json.Unmarshal(j.Payload, &p); err != nil {
			return nil, &jobs.UserError{Code: apierr.ValidationFailed, Message: "Yêu cầu đồng bộ không đọc được."}
		}
		defer s.Redis.Del(context.WithoutCancel(ctx), teamsRunningKey(p.AssignmentID)) //nolint:errcheck // khoá tự hết hạn
		res, err := s.Run(ctx, p.CourseID, p.AssignmentID, func(done, total int) {
			r.ReportDetail(ctx, j.ID, done*100/max(total, 1), map[string]int{"fetched": done, "total": total})
		})
		if err != nil {
			return nil, teamsErr(err)
		}
		return res, nil
	})
}
