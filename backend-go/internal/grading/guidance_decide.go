package grading

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

func modelView(m store.GradingGuidanceItem) ItemView {
	v := ItemView{ID: m.ID.String(), CriterionID: m.CriterionID, Text: m.Text, TextOriginal: m.TextOriginal, Direction: m.Direction, Support: int(m.Support), NeedsRecheck: m.NeedsRecheck,
		Status: string(m.Status), PendingRetire: m.PendingRetire, Version: int(m.Version), CreatedAt: m.CreatedAt}
	if m.ApprovedInVersion != nil {
		x := int(*m.ApprovedInVersion)
		v.ApprovedInVersion = &x
	}
	if m.RetiredInVersion != nil {
		x := int(*m.RetiredInVersion)
		v.RetiredInVersion = &x
	}
	return v
}

func (s *GuidanceService) tx(ctx context.Context, fn func(q *store.Queries, tx pgx.Tx) error) error {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("guidance: giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(store.New(tx), tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("guidance: commit: %w", err)
	}
	return nil
}

func changed(ctx context.Context, tx pgx.Tx, course, assignment uuid.UUID) error {
	if _, err := outbox.Write(ctx, tx, TopicGuidanceChanged, map[string]any{"course_id": course, "assignment_id": assignment}); err != nil {
		return fmt.Errorf("guidance: outbox: %w", err)
	}
	return nil
}

// Resolve quyết định một lưu ý (SRS 4.2.9, #5): APPROVE / REJECT chỉ từ DRAFT (`text` sửa kèm theo phải qua kiểm từ khoá + PII), RETIRE chỉ từ mục đang hiệu lực (chờ nghỉ tới lần Áp dụng).
// Chỉ TEACHER (route). Mỗi quyết định ghi `audit_log` (chữ + ủng hộ, không id sinh viên).
func (s *GuidanceService) Resolve(ctx context.Context, a GuidanceActor, course, assignment, item uuid.UUID, decision string, text *string, version int) (ItemView, error) {
	if !s.on(course) {
		return ItemView{}, disabled()
	}
	if text != nil {
		if decision == "RETIRE" || decision == "RESET" {
			return ItemView{}, apierr.Validation(apierr.FieldError{Field: "text", Code: "not_allowed", Message: "Chỉ sửa chữ khi chưa duyệt."})
		}
		if ValidateText(ctx, s.Det, course, *text, s.Cfg.TextMax) != "" {
			return ItemView{}, apierr.New(http.StatusUnprocessableEntity, apierr.GuidanceTextRejected)
		}
	}
	var out ItemView
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		cur, err := q.GuidanceItemGet(ctx, store.GuidanceItemGetParams{CourseID: course, AssignmentID: assignment, ID: item})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusNotFound, apierr.NotFound)
		}
		if err != nil {
			return fmt.Errorf("guidance: đọc lưu ý: %w", err)
		}
		if decision == "RESET" && cur.Status == store.GuidanceItemStatusDRAFT { // idempotent: Hoàn tác bấm đúp / mục đã về nháp
			out = modelView(cur)
			return nil
		}
		if int(cur.Version) != version {
			return apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current": modelView(cur)})
		}
		before := map[string]any{"status": cur.Status, "text": cur.Text, "support": cur.Support}
		action := "grading_guidance." + map[string]string{"APPROVE": "approve", "REJECT": "reject", "RETIRE": "retire", "RESET": "reset"}[decision]
		var n int64
		switch decision {
		case "APPROVE", "REJECT":
			if cur.Status != store.GuidanceItemStatusDRAFT {
				return apierr.New(http.StatusConflict, apierr.ItemNotDraft)
			}
			st := store.GuidanceItemStatusAPPROVED
			if decision == "REJECT" {
				st = store.GuidanceItemStatusREJECTED
			}
			var newText *string
			if text != nil && *text != cur.Text {
				newText = text
				action = "grading_guidance.edit_" + map[string]string{"APPROVE": "approve", "REJECT": "reject"}[decision]
			}
			n, err = q.GuidanceDecide(ctx, store.GuidanceDecideParams{Status: st, Actor: &a.UserID, NewText: newText, CourseID: course, AssignmentID: assignment, ID: item, Version: int32(version)}) //nolint:gosec // version nhỏ
		case "RETIRE":
			if cur.Status != store.GuidanceItemStatusAPPROVED || cur.ApprovedInVersion == nil || cur.PendingRetire {
				return apierr.New(http.StatusConflict, apierr.ItemNotDraft)
			}
			n, err = q.GuidanceMarkRetire(ctx, store.GuidanceMarkRetireParams{CourseID: course, AssignmentID: assignment, ID: item, Version: int32(version)}) //nolint:gosec // version nhỏ
		case "RESET": // hoàn tác / đặt lại trước khi Áp dụng (P-106-8): chờ áp dụng / đã bỏ → DRAFT (giữ chữ đã sửa); chờ nghỉ → đang dùng
			action = "grading_guidance.reset"
			switch {
			case cur.Status == store.GuidanceItemStatusAPPROVED && cur.ApprovedInVersion != nil && cur.PendingRetire:
				n, err = q.GuidanceResetRetire(ctx, store.GuidanceResetRetireParams{CourseID: course, AssignmentID: assignment, ID: item, Version: int32(version)}) //nolint:gosec // version nhỏ
			case (cur.Status == store.GuidanceItemStatusAPPROVED || cur.Status == store.GuidanceItemStatusREJECTED) && cur.ApprovedInVersion == nil:
				n, err = q.GuidanceReset(ctx, store.GuidanceResetParams{CourseID: course, AssignmentID: assignment, ID: item, Version: int32(version)}) //nolint:gosec // version nhỏ
			default: // đã nằm trong một phiên bản đã áp dụng, hoặc đã nghỉ
				return apierr.New(http.StatusConflict, apierr.ItemNotResettable)
			}
			if err == nil && n == 0 { // Áp dụng thắng cuộc đua giữa hai giao dịch
				return apierr.New(http.StatusConflict, apierr.ItemNotResettable)
			}
		default:
			return apierr.Validation(apierr.FieldError{Field: "decision", Code: "invalid", Message: "Quyết định không hợp lệ."})
		}
		if err != nil {
			return fmt.Errorf("guidance: ghi quyết định: %w", err)
		}
		if n == 0 {
			return apierr.New(http.StatusConflict, apierr.VersionConflict)
		}
		now, err := q.GuidanceItemGet(ctx, store.GuidanceItemGetParams{CourseID: course, AssignmentID: assignment, ID: item})
		if err != nil {
			return fmt.Errorf("guidance: đọc lại lưu ý: %w", err)
		}
		if err := s.audit(ctx, q, a, course, "grading_guidance_items", item.String(), action, before, map[string]any{"status": now.Status, "text": now.Text, "support": now.Support, "pending_retire": now.PendingRetire}); err != nil {
			return fmt.Errorf("guidance: audit: %w", err)
		}
		if err := changed(ctx, tx, course, assignment); err != nil {
			return err
		}
		out = modelView(now)
		return nil
	})
	return out, err
}

// ApplyOut là phản hồi #6: phiên bản mới và số bài chấm theo bản cũ (ba nhóm).
type ApplyOut struct {
	Version int       `json:"version"`
	Stale   StaleView `json:"stale"`
}

// Apply gom các mục đã duyệt chờ áp dụng (và chờ nghỉ) thành phiên bản N+1 (SRS 4.2.10, #6). Một giao dịch; KHÔNG tự chấm lại.
// Đây là nơi DUY NHẤT đổi assignments.guidance_version (TestGuidanceVersionOnlyViaApply).
func (s *GuidanceService) Apply(ctx context.Context, a GuidanceActor, course, assignment uuid.UUID, expected int) (ApplyOut, error) {
	if !s.on(course) {
		return ApplyOut{}, disabled()
	}
	var out ApplyOut
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		cur, err := q.GuidanceLockAssignment(ctx, store.GuidanceLockAssignmentParams{CourseID: course, AssignmentID: assignment})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusNotFound, apierr.AssignmentNotFound)
		}
		if err != nil {
			return fmt.Errorf("guidance: khoá bài tập: %w", err)
		}
		if int(cur) != expected {
			return apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current_version": int(cur)})
		}
		next := cur + 1
		n1, err := q.GuidanceApplyApprove(ctx, store.GuidanceApplyApproveParams{NewVersion: next, CourseID: course, AssignmentID: assignment})
		if err != nil {
			return fmt.Errorf("guidance: áp dụng mục duyệt: %w", err)
		}
		n2, err := q.GuidanceApplyRetire(ctx, store.GuidanceApplyRetireParams{NewVersion: next, CourseID: course, AssignmentID: assignment})
		if err != nil {
			return fmt.Errorf("guidance: áp dụng mục nghỉ: %w", err)
		}
		if n1+n2 == 0 {
			return apierr.New(http.StatusConflict, apierr.NothingToApply)
		}
		active, err := q.GuidanceSetAt(ctx, store.GuidanceSetAtParams{CourseID: course, AssignmentID: assignment, V: next})
		if err != nil {
			return fmt.Errorf("guidance: đếm mục hiệu lực: %w", err)
		}
		if err := q.GuidanceVersionInsert(ctx, store.GuidanceVersionInsertParams{CourseID: course, AssignmentID: assignment, Version: next, ItemCount: int16(len(active)), Actor: a.UserID}); err != nil { //nolint:gosec // ≤ 50
			return fmt.Errorf("guidance: ghi phiên bản: %w", err)
		}
		if err := q.GuidanceSetVersion(ctx, store.GuidanceSetVersionParams{NewVersion: next, CourseID: course, AssignmentID: assignment}); err != nil {
			return fmt.Errorf("guidance: đổi phiên bản: %w", err)
		}
		if err := s.audit(ctx, q, a, course, "assignments", assignment.String(), "grading_guidance.apply", map[string]any{"version": cur}, map[string]any{"version": next, "approved": n1, "retired": n2, "active": len(active)}); err != nil {
			return fmt.Errorf("guidance: audit: %w", err)
		}
		if err := changed(ctx, tx, course, assignment); err != nil {
			return err
		}
		stale, err := s.stale(ctx, q, course, assignment)
		if err != nil {
			return err
		}
		out = ApplyOut{Version: int(next), Stale: stale}
		return nil
	})
	return out, err
}

// ---- chấm lại lô chưa sửa (#7) ------------------------------------------------------------------------------------------------------------

// RegradeSkipped là số bài bị bỏ qua: đã duyệt, đang sửa dở, hoặc đã đủ 20 vòng chấm.
type RegradeSkipped struct {
	Reviewed int `json:"reviewed"`
	Edited   int `json:"edited"`
	MaxRound int `json:"max_round"`
}

// RegradeOut là phản hồi #7: JobID nil khi không có bài nào để chấm (200).
type RegradeOut struct {
	JobID   *uuid.UUID     `json:"job_id,omitempty"`
	Queued  int            `json:"queued"`
	Skipped RegradeSkipped `json:"skipped"`
}

// RegradeStale xếp chấm lại (round + 1) CÁC BÀI NHÁP bản cũ chưa ai sửa / duyệt (SRS 4.2.11, #7). Không bao giờ đụng bài đã duyệt, đã sửa tay hay đã công bố: điểm người đã chốt không bị ghi đè.
// Tối đa RegradeMaxBatch bài mỗi lần; idempotent nhờ UNIQUE (course, submission, round); việc đi làn BATCH như chấm thường.
func (s *GuidanceService) RegradeStale(ctx context.Context, a GuidanceActor, course, assignment uuid.UUID) (RegradeOut, error) {
	if !s.on(course) {
		return RegradeOut{}, disabled()
	}
	var out RegradeOut
	b := &Batches{Pool: s.Pool, Jobs: s.Jobs}
	e, err := b.enqueue(ctx, a.UserID, course, func(q *store.Queries) ([]uuid.UUID, error) {
		if _, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: assignment}); errors.Is(err, pgx.ErrNoRows) {
			return nil, apierr.New(http.StatusNotFound, apierr.AssignmentNotFound)
		} else if err != nil {
			return nil, err
		}
		stale, err := q.GuidanceStaleRows(ctx, store.GuidanceStaleRowsParams{CourseID: course, AssignmentID: assignment})
		if err != nil {
			return nil, err
		}
		for _, r := range stale {
			switch {
			case r.Status == "REVIEWED":
				out.Skipped.Reviewed++
			case r.FirstReviewedAt != nil || len(changedCriteriaOf(r.Criteria, r.AiScores)) > 0:
				out.Skipped.Edited++
			case r.Round >= 20:
				out.Skipped.MaxRound++
			}
		}
		cands, err := q.GuidanceStaleCandidates(ctx, store.GuidanceStaleCandidatesParams{CourseID: course, AssignmentID: assignment, PageLimit: int32(s.Cfg.RegradeMaxBatch)}) //nolint:gosec // ≤ 1000
		if err != nil {
			return nil, err
		}
		var ids []uuid.UUID
		for _, c := range cands {
			if len(changedCriteriaOf(c.Criteria, c.AiScores)) > 0 { // so decimal: bài có tiêu chí sửa không bao giờ bị chấm lại
				continue
			}
			id, err := q.GradingInsertQueued(ctx, store.GradingInsertQueuedParams{CourseID: course, SubmissionID: c.SubmissionID, Round: c.Round + 1})
			if errors.Is(err, pgx.ErrNoRows) {
				continue // hai người cùng bấm: dòng đã có
			}
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		return ids, nil
	})
	if err != nil {
		return RegradeOut{}, err
	}
	out.JobID, out.Queued = e.JobID, e.Queued
	return out, nil
}

// ---- số đo (#8) ----------------------------------------------------------------------------------------------------------------------------

// VersionStats là số đo của kết quả hiện hành đã duyệt / công bố, gom theo phiên bản ghim. Điểm là chuỗi decimal; n < 5 → các số trung bình là null.
type VersionStats struct {
	Version         int     `json:"version"`
	N               int     `json:"n"`
	MaeCriterion    *string `json:"mae_criterion"`
	MaeTotal        *string `json:"mae_total"`
	BiasTotal       *string `json:"bias_total"`
	EditedCriteria  int     `json:"edited_criteria"`
	EditedRate      *string `json:"edited_rate"`
	CriteriaCounted int     `json:"criteria_counted"`
}

// StatsView là phản hồi #8.
type StatsView struct {
	Items []VersionStats `json:"items"`
}

// minStatsN là cỡ mẫu tối thiểu để công bố số trung bình (SRS 4.2.13).
const minStatsN = 5

// Stats tính MAE tiêu chí / tổng và lệch trung bình (có dấu: dương = AI cho điểm cao hơn giảng viên) theo phiên bản ghim, bằng decimal.
func (s *GuidanceService) Stats(ctx context.Context, course, assignment uuid.UUID) (StatsView, error) {
	q := store.New(s.Pool)
	if _, err := q.GuidanceCurrentVersion(ctx, store.GuidanceCurrentVersionParams{CourseID: course, AssignmentID: assignment}); errors.Is(err, pgx.ErrNoRows) {
		return StatsView{}, apierr.New(http.StatusNotFound, apierr.AssignmentNotFound)
	} else if err != nil {
		return StatsView{}, fmt.Errorf("guidance: đọc bài tập: %w", err)
	}
	rows, err := q.GuidanceStatsRows(ctx, store.GuidanceStatsRowsParams{CourseID: course, AssignmentID: assignment})
	if err != nil {
		return StatsView{}, fmt.Errorf("guidance: đọc số đo: %w", err)
	}
	type acc struct {
		n, crit, edited          int
		sumCrit, sumTot, sumBias decimal.Decimal
	}
	by := map[int]*acc{}
	for _, r := range rows {
		a := by[int(r.GuidanceVersion)]
		if a == nil {
			a = &acc{}
			by[int(r.GuidanceVersion)] = a
		}
		a.n++
		var finTotal decimal.Decimal
		for _, d := range deltasOf(r.Criteria, r.AiScores) {
			a.crit++
			a.sumCrit = a.sumCrit.Add(d.AI.Sub(d.Teacher).Abs())
			finTotal = finTotal.Add(d.Teacher)
			if !d.AI.Equal(d.Teacher) {
				a.edited++
			}
		}
		if r.AiTotalBeforeLate.Valid {
			diff := r.AiTotalBeforeLate.Decimal.Sub(finTotal)
			a.sumTot = a.sumTot.Add(diff.Abs())
			a.sumBias = a.sumBias.Add(diff)
		}
	}
	vers := make([]int, 0, len(by))
	for v := range by {
		vers = append(vers, v)
	}
	slices.Sort(vers)
	out := StatsView{Items: []VersionStats{}}
	for _, v := range vers {
		a := by[v]
		it := VersionStats{Version: v, N: a.n, EditedCriteria: a.edited, CriteriaCounted: a.crit}
		if a.n >= minStatsN && a.crit > 0 {
			n, c := decimal.NewFromInt(int64(a.n)), decimal.NewFromInt(int64(a.crit))
			f := func(d decimal.Decimal) *string { x := d.StringFixed(2); return &x }
			it.MaeCriterion, it.MaeTotal, it.BiasTotal = f(a.sumCrit.Div(c)), f(a.sumTot.Div(n)), f(a.sumBias.Div(n))
			it.EditedRate = f(decimal.NewFromInt(int64(a.edited)).Div(c))
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}
