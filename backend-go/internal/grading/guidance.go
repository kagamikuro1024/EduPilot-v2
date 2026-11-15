package grading

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/store"
)

// GuidanceService là phía gateway của «Lưu ý chấm» (SRS FEAT-grading-guidance 6 #1–#8): cờ theo lớp, tổng quan, rút lưu ý, quyết định, áp dụng, số đo.
// Gateway không trạng thái (luật 10): trần / khoá idempotent ở Redis.
type GuidanceService struct {
	Pool  *pgxpool.Pool
	Redis *goredis.Client
	Jobs  *jobs.Service
	Flags *GuidanceFlags
	Det   PIIDetector
	Cfg   GuidanceConfig
}

// GuidanceActor là người gọi: ADMIN (JWT) hoặc nhân viên của lớp.
type GuidanceActor struct {
	UserID  uuid.UUID
	Admin   bool
	TraceID string
}

func (s *GuidanceService) on(course uuid.UUID) bool {
	return s.Cfg.Enabled && s.Flags.On(course, FlagGradingGuidance)
}

func disabled() *apierr.Error { return apierr.New(http.StatusConflict, apierr.GuidanceDisabled) }

func (s *GuidanceService) audit(ctx context.Context, q *store.Queries, a GuidanceActor, course uuid.UUID, entity, id, action string, before, after any) error {
	b, _ := json.Marshal(before)
	af, _ := json.Marshal(after)
	var trace *string
	if a.TraceID != "" {
		trace = &a.TraceID
	}
	_, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &a.UserID, Entity: entity, EntityID: id, Action: action, Before: b, After: af, TraceID: trace})
	return err
}

// ---- cờ (#1, #2) ----------------------------------------------------------------------------------------------------------------------------

// FlagView là một cờ trả cho giao diện; CanEdit false = công tắc khoá với người xem (REGRESSION_CAPTURE chỉ ADMIN).
type FlagView struct {
	Feature string `json:"feature"`
	Enabled bool   `json:"enabled"`
	Version int    `json:"version"`
	CanEdit bool   `json:"can_edit"`
}

func canEditFlag(feature string, admin bool) bool { return admin || feature != FlagRegressionCapture }

// courseExists: ADMIN không bị chặn bởi thẻ thành viên nên id lớp lạ phải trả 404 (không để khoá ngoại thành 500).
func (s *GuidanceService) courseExists(ctx context.Context, course uuid.UUID) error {
	ok, err := store.New(s.Pool).CourseExists(ctx, course)
	if err != nil {
		return fmt.Errorf("guidance: kiểm lớp: %w", err)
	}
	if !ok {
		return apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	return nil
}

// ListFlags trả đủ bốn cờ (hàng chưa có = tắt, version 1).
func (s *GuidanceService) ListFlags(ctx context.Context, a GuidanceActor, course uuid.UUID) ([]FlagView, error) {
	if err := s.courseExists(ctx, course); err != nil {
		return nil, err
	}
	rows, err := store.New(s.Pool).GuidanceFlagList(ctx, course)
	if err != nil {
		return nil, fmt.Errorf("guidance: đọc cờ: %w", err)
	}
	have := map[string]store.GuidanceFlagListRow{}
	for _, r := range rows {
		have[r.Feature] = r
	}
	out := make([]FlagView, 0, len(GuidanceFlagList))
	for _, f := range GuidanceFlagList {
		r, ok := have[f]
		if !ok {
			r = store.GuidanceFlagListRow{Feature: f, Version: 1}
		}
		out = append(out, FlagView{Feature: f, Enabled: r.Enabled, Version: int(r.Version), CanEdit: canEditFlag(f, a.Admin)})
	}
	return out, nil
}

// SetFlag đặt một cờ theo version (SRS 6 #2). Tính năng đang bị công tắc cứng tắt → 409 GUIDANCE_DISABLED; REGRESSION_CAPTURE chỉ ADMIN → 403.
func (s *GuidanceService) SetFlag(ctx context.Context, a GuidanceActor, course uuid.UUID, feature string, enabled bool, version int) (FlagView, error) {
	if !slices.Contains(GuidanceFlagList, feature) {
		return FlagView{}, apierr.Validation(apierr.FieldError{Field: "feature", Code: "invalid", Message: "Không có công tắc này."})
	}
	if !canEditFlag(feature, a.Admin) {
		return FlagView{}, apierr.New(http.StatusForbidden, apierr.Forbidden)
	}
	if err := s.courseExists(ctx, course); err != nil {
		return FlagView{}, err
	}
	if !s.Cfg.Enabled {
		return FlagView{}, disabled()
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return FlagView{}, fmt.Errorf("guidance: giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := store.New(tx)
	cur := store.GuidanceFlagListRow{Feature: feature, Version: 1}
	rows, err := q.GuidanceFlagList(ctx, course)
	if err != nil {
		return FlagView{}, fmt.Errorf("guidance: đọc cờ: %w", err)
	}
	for _, r := range rows {
		if r.Feature == feature {
			cur = r
		}
	}
	if int(cur.Version) != version {
		return FlagView{}, apierr.New(http.StatusConflict, apierr.VersionConflict)
	}
	got, err := q.GuidanceFlagSet(ctx, store.GuidanceFlagSetParams{CourseID: course, Feature: feature, Enabled: enabled, Actor: &a.UserID, Version: int32(version)}) //nolint:gosec // version nhỏ
	if errors.Is(err, pgx.ErrNoRows) {
		return FlagView{}, apierr.New(http.StatusConflict, apierr.VersionConflict)
	}
	if err != nil {
		return FlagView{}, fmt.Errorf("guidance: ghi cờ: %w", err)
	}
	if err := s.audit(ctx, q, a, course, "course_guidance_flags", course.String()+"/"+feature, "guidance_flag.set", map[string]any{"enabled": cur.Enabled}, map[string]any{"enabled": got.Enabled}); err != nil {
		return FlagView{}, fmt.Errorf("guidance: audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return FlagView{}, fmt.Errorf("guidance: commit: %w", err)
	}
	s.publishFlags(ctx)
	return FlagView{Feature: feature, Enabled: got.Enabled, Version: int(got.Version), CanEdit: true}, nil
}

// publishFlags nạp lại ảnh chụp tại chỗ rồi báo các bản khác qua Redis; mất thông báo thì TTL 60 s lo (SRS §8).
func (s *GuidanceService) publishFlags(ctx context.Context) {
	if s.Flags != nil {
		_ = s.Flags.Load(ctx)
	}
	if s.Redis != nil {
		_ = s.Redis.Publish(ctx, FlagsChannel, "1").Err()
	}
}

// ---- tổng quan (#3) ----------------------------------------------------------------------------------------------------------------------

// ItemView là một lưu ý trả cho giao diện nhân viên.
type ItemView struct {
	ID                string    `json:"id"`
	CriterionID       string    `json:"criterion_id"`
	Text              string    `json:"text"`
	TextOriginal      *string   `json:"text_original"`
	Direction         string    `json:"direction"`
	Support           int       `json:"support"`
	NeedsRecheck      bool      `json:"needs_recheck"`
	Status            string    `json:"status"`
	PendingRetire     bool      `json:"pending_retire"`
	ApprovedInVersion *int      `json:"approved_in_version"`
	RetiredInVersion  *int      `json:"retired_in_version"`
	Version           int       `json:"version"`
	CreatedAt         time.Time `json:"created_at"`
}

// StaleView đếm kết quả chưa công bố chấm theo bản cũ, ba nhóm (TL-1 mục 6).
type StaleView struct {
	Count    int   `json:"count"`    // nháp cũ chưa ai sửa: «Chấm lại» chọn đúng nhóm này
	Editing  int   `json:"editing"`  // nháp đang sửa dở (có tiêu chí sửa hoặc từng được duyệt)
	Reviewed int   `json:"reviewed"` // đã duyệt: miễn so phiên bản khi công bố
	Versions []int `json:"versions"` // các bản cũ còn dùng
}

// ReflectState cho nút `Rút lưu ý`.
type ReflectState struct {
	Eligible bool `json:"eligible"`
	Results  int  `json:"results"`
	Edits    int  `json:"edits"`
}

// OverviewView là phản hồi #3.
type OverviewView struct {
	Enabled        bool           `json:"enabled"`
	CurrentVersion int            `json:"current_version"`
	Items          []ItemView     `json:"items"`
	NextCursor     *string        `json:"next_cursor"`
	Counts         map[string]int `json:"counts"`
	Stale          StaleView      `json:"stale"`
	Reflect        ReflectState   `json:"reflect"`
}

func itemView(r store.GuidanceItemsRow) ItemView {
	v := ItemView{ID: r.ID.String(), CriterionID: r.CriterionID, Text: r.Text, TextOriginal: r.TextOriginal, Direction: r.Direction, Support: int(r.Support), NeedsRecheck: r.NeedsRecheck,
		Status: string(r.Status), PendingRetire: r.PendingRetire, Version: int(r.Version), CreatedAt: r.CreatedAt}
	if r.ApprovedInVersion != nil {
		x := int(*r.ApprovedInVersion)
		v.ApprovedInVersion = &x
	}
	if r.RetiredInVersion != nil {
		x := int(*r.RetiredInVersion)
		v.RetiredInVersion = &x
	}
	return v
}

// Overview trả danh sách lưu ý (phân trang con trỏ), phiên bản, bộ đếm, số bài chấm theo bản cũ và điều kiện rút lưu ý.
func (s *GuidanceService) Overview(ctx context.Context, course, assignment uuid.UUID, cursorAt *time.Time, cursorID *uuid.UUID, limit int) (OverviewView, error) {
	q := store.New(s.Pool)
	cur, err := q.GuidanceCurrentVersion(ctx, store.GuidanceCurrentVersionParams{CourseID: course, AssignmentID: assignment})
	if errors.Is(err, pgx.ErrNoRows) {
		return OverviewView{}, apierr.New(http.StatusNotFound, apierr.AssignmentNotFound)
	}
	if err != nil {
		return OverviewView{}, fmt.Errorf("guidance: đọc bài tập: %w", err)
	}
	out := OverviewView{Enabled: s.on(course) && cur.Type == "ESSAY", CurrentVersion: int(cur.GuidanceVersion), Items: []ItemView{}, Counts: map[string]int{}}
	rows, err := q.GuidanceItems(ctx, store.GuidanceItemsParams{CourseID: course, AssignmentID: assignment, CursorAt: cursorAt, CursorID: cursorID, PageLimit: int32(limit + 1)}) //nolint:gosec // limit ≤ 100
	if err != nil {
		return OverviewView{}, fmt.Errorf("guidance: đọc lưu ý: %w", err)
	}
	for i, r := range rows {
		if i == limit {
			c := httpx.EncodeCursor(rows[limit-1].CreatedAt, rows[limit-1].ID.String())
			out.NextCursor = &c
			break
		}
		out.Items = append(out.Items, itemView(r))
	}
	counts, err := q.GuidanceItemCounts(ctx, store.GuidanceItemCountsParams{CourseID: course, AssignmentID: assignment})
	if err != nil {
		return OverviewView{}, fmt.Errorf("guidance: đếm lưu ý: %w", err)
	}
	for _, c := range counts {
		key := countKey(string(c.Status), c.PendingRetire, c.Status == store.GuidanceItemStatusAPPROVED)
		out.Counts[key] += int(c.N)
	}
	if out.Stale, err = s.stale(ctx, q, course, assignment); err != nil {
		return OverviewView{}, err
	}
	cases, err := q.GuidanceCases(ctx, store.GuidanceCasesParams{CourseID: course, AssignmentID: assignment})
	if err != nil {
		return OverviewView{}, fmt.Errorf("guidance: đọc ca: %w", err)
	}
	res, edits := Eligibility(cases, s.Cfg.MinDelta)
	out.Reflect = ReflectState{Results: res, Edits: edits, Eligible: out.Enabled && res >= s.Cfg.MinResults && edits >= s.Cfg.MinEdits}
	return out, nil
}

// countKey gộp trạng thái thành nhãn của giao diện: draft | pending_apply | active | pending_retire | rejected | retired.
func countKey(status string, pendingRetire, approved bool) string {
	switch {
	case status == "DRAFT":
		return "draft"
	case status == "REJECTED":
		return "rejected"
	case status == "RETIRED":
		return "retired"
	case approved && pendingRetire:
		return "pending_retire"
	}
	return "approved"
}

func (s *GuidanceService) stale(ctx context.Context, q *store.Queries, course, assignment uuid.UUID) (StaleView, error) {
	rows, err := q.GuidanceStaleRows(ctx, store.GuidanceStaleRowsParams{CourseID: course, AssignmentID: assignment})
	if err != nil {
		return StaleView{}, fmt.Errorf("guidance: đếm bài cũ: %w", err)
	}
	v := StaleView{Versions: []int{}}
	for _, r := range rows {
		if gv := int(r.GuidanceVersion); !slices.Contains(v.Versions, gv) {
			v.Versions = append(v.Versions, gv)
		}
		switch {
		case r.Status == "REVIEWED":
			v.Reviewed++
		case r.FirstReviewedAt != nil || len(changedCriteriaOf(r.Criteria, r.AiScores)) > 0:
			v.Editing++
		default:
			v.Count++
		}
	}
	slices.Sort(v.Versions)
	return v, nil
}

// changedCriteriaOf: tiêu chí có điểm cuối khác điểm AI (như assignment.changedCriteria; so bằng decimal).
func changedCriteriaOf(criteria, ai json.RawMessage) []string {
	var out []string
	for _, d := range deltasOf(criteria, ai) {
		if !d.AI.Equal(d.Teacher) {
			out = append(out, d.ID)
		}
	}
	return out
}

// ---- rút lưu ý (#4) ----------------------------------------------------------------------------------------------------------------------

// ReflectEnqueued là phản hồi 202.
type ReflectEnqueued struct {
	JobID string `json:"job_id"`
}

// Reflect kiểm theo thứ tự rồi xếp việc (TL-1 mục 4.1): cờ → ESSAY → đủ điều kiện → khoá idempotent → trần giờ → xếp hàng.
// Hai lần bấm cùng dữ liệu (cùng max(reviewed_at), cùng bản) trả CÙNG job_id, không tạo việc mới và không tính vào trần.
func (s *GuidanceService) Reflect(ctx context.Context, a GuidanceActor, course, assignment uuid.UUID) (ReflectEnqueued, error) {
	if !s.on(course) {
		return ReflectEnqueued{}, disabled()
	}
	q := store.New(s.Pool)
	cur, err := q.GuidanceCurrentVersion(ctx, store.GuidanceCurrentVersionParams{CourseID: course, AssignmentID: assignment})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && cur.Type != "ESSAY") {
		return ReflectEnqueued{}, apierr.New(http.StatusNotFound, apierr.AssignmentNotFound)
	}
	if err != nil {
		return ReflectEnqueued{}, fmt.Errorf("guidance: đọc bài tập: %w", err)
	}
	cases, err := q.GuidanceCases(ctx, store.GuidanceCasesParams{CourseID: course, AssignmentID: assignment})
	if err != nil {
		return ReflectEnqueued{}, fmt.Errorf("guidance: đọc ca: %w", err)
	}
	if res, edits := Eligibility(cases, s.Cfg.MinDelta); res < s.Cfg.MinResults || edits < s.Cfg.MinEdits {
		return ReflectEnqueued{}, apierr.New(http.StatusConflict, apierr.NotEnoughData).WithMessage("Cần thêm bài đã duyệt.").WithDetails(map[string]int{"results": res, "edits": edits})
	}
	last, err := q.GuidanceMaxReviewedAt(ctx, store.GuidanceMaxReviewedAtParams{CourseID: course, AssignmentID: assignment})
	if err != nil {
		return ReflectEnqueued{}, fmt.Errorf("guidance: đọc mốc duyệt: %w", err)
	}
	stamp := last.UTC().Format(time.RFC3339Nano)
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%s|%d", assignment, stamp, cur.GuidanceVersion))
	idemKey := "ep:guidance:reflect:" + hex.EncodeToString(sum[:])
	if id, err := s.Redis.Get(ctx, idemKey).Result(); err == nil && id != "" {
		return ReflectEnqueued{JobID: id}, nil
	}
	rl := "ep:rl:reflect:" + assignment.String()
	n, err := s.Redis.Incr(ctx, rl).Result()
	if err != nil {
		return ReflectEnqueued{}, fmt.Errorf("guidance: trần giờ: %w", err)
	}
	if n == 1 {
		s.Redis.Expire(ctx, rl, time.Hour)
	}
	if int(n) > s.Cfg.ReflectMaxPerHour {
		return ReflectEnqueued{}, apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(int(time.Hour / time.Second))
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ReflectEnqueued{}, fmt.Errorf("guidance: giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	j, err := s.Jobs.EnqueueTx(ctx, tx, a.UserID, KindReflect, ReflectPayload{CourseID: course, AssignmentID: assignment, GuidanceVersion: int(cur.GuidanceVersion)})
	if err != nil {
		return ReflectEnqueued{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReflectEnqueued{}, fmt.Errorf("guidance: commit: %w", err)
	}
	s.Redis.SetNX(ctx, idemKey, j.ID.String(), time.Hour)
	return ReflectEnqueued{JobID: j.ID.String()}, nil
}
