package submission

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Env là dữ liệu lớp nạp MỘT LẦN cho cả lô (ZIP nhiều mục không truy vấn danh sách lại cho từng mục).
type Env struct {
	CourseID uuid.UUID
	Roster   []RosterEntry
	Asgs     []AssignmentRef
}

// LoadEnv nạp danh sách sinh viên ACTIVE và các bài đã công bố có kênh `channel` bật.
func LoadEnv(ctx context.Context, q *store.Queries, course uuid.UUID, channel string) (Env, error) {
	rs, err := q.SubmissionRoster(ctx, course)
	if err != nil {
		return Env{}, fmt.Errorf("submission: danh sách lớp: %w", err)
	}
	as, err := q.SubmissionAssignmentRefs(ctx, store.SubmissionAssignmentRefsParams{CourseID: course, Channel: channel})
	if err != nil {
		return Env{}, fmt.Errorf("submission: bài tập của lớp: %w", err)
	}
	env := Env{CourseID: course}
	for _, r := range rs {
		env.Roster = append(env.Roster, RosterEntry{StudentID: r.UserID, Email: r.Email, EmailVerified: r.EmailVerified, Code: r.Code, JoinedVia: r.JoinedVia})
	}
	for _, a := range as {
		env.Asgs = append(env.Asgs, AssignmentRef{ID: a.ID, Code: a.SubmissionCode})
	}
	return env, nil
}

// Raw là một bài nộp từ nguồn ngoài (tệp lẻ, mục trong ZIP, thư, Teams) đã nằm sẵn trong object storage.
type Raw struct {
	Source         string // UPLOAD | IMAP | TEAMS
	ExternalRef    string
	ExternalTS     time.Time
	FromEmail      string
	SenderVerified bool
	Subject        string
	FileName       string
	MatchName      string // tên dùng để khớp khi khác tên lưu (mục ZIP: đường dẫn đầy đủ trong ZIP, thư mục cũng có thể chứa MSSV)
	BlobKey        string
	FileMime       string
	SHA256         string
	FileSize       int64
	ReceivedAt     time.Time
	SubmittedAt    *time.Time // nguồn có giờ nộp thật (Teams); UPLOAD / IMAP để nil → không bao giờ muộn qua bước này
	AssignmentID   *uuid.UUID // nguồn gọi theo bài
}

// Outcome là kết quả một lần Ingest.
type Outcome struct {
	ID      uuid.UUID
	Deduped bool
	Matched bool
	Reason  string // lý do UNMATCHED
	Version int
}

// Ingest là đường ống CHUNG của mọi nguồn ngoài (SRS 4.4.6), chạy trong giao dịch của người gọi:
// (1) trùng (course, source, external_ref, external_ts) → deduped; (2) Match; (3) khớp mà bản mới nhất của (bài, sinh viên) đã REVIEWED / PUBLISHED → KHÔNG đè: UNMATCHED
// `ALREADY_REVIEWED` kèm `candidate_student_id`; (4) cấp `version` dưới khoá nộp của (bài, sinh viên); (5) ghi `submissions` + outbox `submission.created`.
func Ingest(ctx context.Context, tx pgx.Tx, env Env, raw Raw, actor *uuid.UUID) (Outcome, error) {
	q := store.New(tx)
	if raw.ReceivedAt.IsZero() {
		raw.ReceivedAt = time.Now().UTC()
	}
	v := Match(Signals{FromEmail: raw.FromEmail, Subject: raw.Subject, FileName: firstNonEmpty(raw.MatchName, raw.FileName), SenderVerified: raw.SenderVerified, AssignmentID: raw.AssignmentID}, env.Roster, env.Asgs)
	p := store.SubmissionIngestParams{CourseID: env.CourseID, Source: raw.Source, BlobKey: raw.BlobKey, FileName: raw.FileName, FileSize: raw.FileSize, FileMime: raw.FileMime, Sha256: raw.SHA256,
		SenderVerified: raw.SenderVerified, SubmittedAt: raw.SubmittedAt, ReceivedAt: raw.ReceivedAt, AssignmentID: v.AssignmentID, CandidateStudentID: v.Candidate}
	if raw.ExternalRef != "" {
		ref, ts := raw.ExternalRef, raw.ExternalTS.UTC()
		p.ExternalRef, p.ExternalTs = &ref, &ts
	}
	if raw.FromEmail != "" {
		p.FromEmail = &raw.FromEmail
	}
	if raw.Subject != "" {
		p.Subject = &raw.Subject
	}
	if v.Matched() {
		if err := q.SubmissionLockStudent(ctx, store.SubmissionLockStudentParams{AssignmentID: v.AssignmentID.String(), StudentID: v.StudentID.String()}); err != nil {
			return Outcome{}, fmt.Errorf("submission: khoá nộp: %w", err)
		}
		cur, err := q.SubmissionLatestMatched(ctx, store.SubmissionLatestMatchedParams{CourseID: env.CourseID, AssignmentID: *v.AssignmentID, StudentID: *v.StudentID})
		switch {
		case err == nil && (cur.ResultStatus == "REVIEWED" || cur.ResultStatus == "PUBLISHED"):
			v = Verdict{AssignmentID: v.AssignmentID, Candidate: v.StudentID, Reason: ReasonAlreadyReviewed}
			p.AssignmentID, p.CandidateStudentID = v.AssignmentID, v.Candidate
		case err != nil && !errors.Is(err, pgx.ErrNoRows):
			return Outcome{}, fmt.Errorf("submission: bản mới nhất: %w", err)
		}
	}
	if v.Matched() {
		p.StudentID, p.MatchStatus, p.MatchMethod = v.StudentID, "MATCHED", &v.Method
	} else {
		p.MatchStatus, p.UnmatchedReason = "UNMATCHED", &v.Reason
	}
	row, err := q.SubmissionIngest(ctx, p)
	if errors.Is(err, pgx.ErrNoRows) {
		return Outcome{Deduped: true}, nil
	}
	if err != nil {
		return Outcome{}, fmt.Errorf("submission: ghi bài nộp: %w", err)
	}
	out := Outcome{ID: row.ID, Matched: v.Matched(), Reason: v.Reason}
	if row.Version != nil {
		out.Version = int(*row.Version)
	}
	after, _ := json.Marshal(map[string]any{"source": raw.Source, "match_status": p.MatchStatus, "reason": v.Reason, "version": out.Version})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &env.CourseID, ActorID: actor, Entity: "submissions", EntityID: row.ID.String(), Action: "create", After: after}); err != nil {
		return Outcome{}, fmt.Errorf("submission: audit: %w", err)
	}
	pay := map[string]any{"course_id": env.CourseID.String(), "submission_id": row.ID.String(), "version": out.Version, "source": raw.Source, "matched": out.Matched}
	if v.AssignmentID != nil {
		pay["assignment_id"] = v.AssignmentID.String()
	}
	if v.StudentID != nil {
		pay["student_id"], pay["user_id"] = v.StudentID.String(), v.StudentID.String()
	}
	if _, err := outbox.Write(ctx, tx, TopicCreated, pay); err != nil {
		return Outcome{}, fmt.Errorf("submission: outbox: %w", err)
	}
	return out, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
