package assignment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// PublishMaxBatch là PUBLISH_MAX_BATCH: số bài tối đa mỗi lần công bố.
const PublishMaxBatch = 200

// TopicResultPublished: một kết quả vừa được công bố → chuông + thư nhận xét cho người nộp, làm mới "Hôm nay".
const TopicResultPublished = "assignment.result_published"

// PublishIn là thân `POST …/grading-results/publish`: danh sách tường minh HOẶC bài tập + scope ALL_REVIEWED. `versions` (tuỳ chọn) khoá lạc quan từng bài.
type PublishIn struct {
	ResultIDs    []uuid.UUID      `json:"result_ids"`
	AssignmentID *uuid.UUID       `json:"assignment_id"`
	Scope        string           `json:"scope"`
	Versions     map[string]int32 `json:"versions"`
}

// ResultsPublishOut là phản hồi: số bài đã công bố và số bài bỏ qua (chỉ scope ALL_REVIEWED).
type ResultsPublishOut struct {
	Published int `json:"published"`
	Skipped   int `json:"skipped"`
}

func idsDetail(code string, status int, ids []uuid.UUID) *apierr.Error {
	return apierr.New(status, code).WithDetails(map[string]any{"ids": ids})
}

// PublishResults công bố hàng loạt (SRS 4.8.5) — NƠI DUY NHẤT đưa kết quả sang PUBLISHED. Một giao dịch mở đầu bằng `FOR SHARE` hàng sổ điểm (tuần tự với Chốt điểm);
// không ghi `grade_entries` (item ASSIGNMENT đọc thẳng điểm đã công bố); độ lệch AI ↔ giảng viên lưu lúc công bố; mỗi bài một `audit_log` + outbox `assignment.result_published`.
func (s *Service) PublishResults(ctx context.Context, a Actor, course uuid.UUID, in PublishIn) (ResultsPublishOut, error) {
	explicit := len(in.ResultIDs) > 0
	switch {
	case explicit && (in.AssignmentID != nil || in.Scope != ""):
		return ResultsPublishOut{}, apierr.Validation(apierr.FieldError{Field: "result_ids", Code: "exclusive", Message: "Chọn danh sách bài hoặc cả bài tập, không cả hai."})
	case !explicit && (in.AssignmentID == nil || in.Scope != "ALL_REVIEWED"):
		return ResultsPublishOut{}, apierr.Validation(apierr.FieldError{Field: "scope", Code: "required", Message: "Cần danh sách bài hoặc assignment_id kèm scope ALL_REVIEWED."})
	case len(in.ResultIDs) > PublishMaxBatch:
		return ResultsPublishOut{}, apierr.Validation(apierr.FieldError{Field: "result_ids", Code: "max", Message: "Tối đa 200 bài mỗi lần công bố."})
	}
	var out ResultsPublishOut
	var published []store.PublishLockManyRow
	err := s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := q.GradebookEnsure(ctx, course); err != nil {
			return fmt.Errorf("assignment: sổ điểm: %w", err)
		}
		if _, err := q.GradebookStatusShare(ctx, course); err != nil { // tuần tự với `Chốt điểm`; sổ LOCKED không chặn (không ghi vào sổ)
			return fmt.Errorf("assignment: khoá sổ điểm: %w", err)
		}
		ids := in.ResultIDs
		over := 0
		if !explicit {
			got, err := q.PublishReviewedIDs(ctx, store.PublishReviewedIDsParams{CourseID: course, AssignmentID: *in.AssignmentID, PageLimit: PublishMaxBatch + 1})
			if err != nil {
				return fmt.Errorf("assignment: bài đã duyệt: %w", err)
			}
			if len(got) > PublishMaxBatch {
				got, over = got[:PublishMaxBatch], 1
			}
			ids = got
		}
		if len(ids) == 0 {
			return nil
		}
		rows, err := q.PublishLockMany(ctx, store.PublishLockManyParams{CourseID: course, Ids: idStrs(ids)})
		if err != nil {
			return fmt.Errorf("assignment: khoá kết quả: %w", err)
		}
		byID := map[uuid.UUID]store.PublishLockManyRow{}
		for _, r := range rows {
			byID[r.ID] = r
		}
		var notReviewed, alreadyPublished, stale, recomputed []uuid.UUID
		var ok []store.PublishLockManyRow
		for _, id := range ids {
			r, found := byID[id]
			switch {
			case !found || r.Superseded || r.RunState != "DONE" || (r.Status != "REVIEWED" && r.Status != "PUBLISHED"):
				notReviewed = append(notReviewed, id)
				continue
			case r.Status == "PUBLISHED":
				alreadyPublished = append(alreadyPublished, id)
				continue
			}
			if v, has := in.Versions[id.String()]; has && v != r.Version {
				stale = append(stale, id)
				continue
			}
			if !lateStable(r) {
				recomputed = append(recomputed, id)
				continue
			}
			ok = append(ok, r)
		}
		if explicit { // danh sách tường minh: all-or-nothing
			switch {
			case len(notReviewed) > 0:
				return idsDetail(apierr.NotReviewed, http.StatusUnprocessableEntity, notReviewed)
			case len(alreadyPublished) > 0:
				return idsDetail(apierr.AlreadyPublished, http.StatusConflict, alreadyPublished)
			case len(stale) > 0:
				return idsDetail(apierr.VersionConflict, http.StatusConflict, stale)
			case len(recomputed) > 0:
				return idsDetail(apierr.LateRecomputed, http.StatusConflict, recomputed)
			}
		}
		for _, r := range ok {
			sum, _ := criteriaSum(r.Criteria)
			var dev decimal.Decimal
			if r.AiTotalBeforeLate.Valid {
				dev = sum.Sub(r.AiTotalBeforeLate.Decimal).Abs()
			}
			changed := changedCriteria(r.Criteria, r.AiScores)
			n, err := q.PublishMark(ctx, store.PublishMarkParams{Publisher: &a.UserID, Deviation: decimal.NullDecimal{Decimal: dev, Valid: true}, Changed: changed,
				LateDays: r.LateDays, LatePenalty: r.LatePenalty, CourseID: course, ID: r.ID, Version: r.Version})
			if err != nil {
				return fmt.Errorf("assignment: công bố: %w", err)
			}
			if n == 0 {
				return idsDetail(apierr.VersionConflict, http.StatusConflict, []uuid.UUID{r.ID})
			}
			if err := s.resultAudit(ctx, q, a, course, r.ID, "grading_result.publish", map[string]any{"status": "REVIEWED"}, map[string]any{"status": "PUBLISHED", "total": totalStr(r.Total), "deviation": dev.StringFixed(2), "changed": changed}); err != nil {
				return fmt.Errorf("assignment: audit: %w", err)
			}
			if _, err := outbox.Write(ctx, tx, TopicResultPublished, map[string]any{"course_id": course, "result_id": r.ID, "student_id": r.StudentID, "user_id": r.StudentID, "assignment_id": r.AssignmentID}); err != nil {
				return fmt.Errorf("assignment: outbox: %w", err)
			}
			published = append(published, r)
		}
		out.Published = len(published)
		out.Skipped = len(ids) - len(published) + over
		return nil
	})
	return out, err
}

func idStrs(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// lateStable: tính lại khoản trừ muộn lần cuối theo hạn / quy tắc HIỆN TẠI; tổng đổi so với bản đã duyệt → không công bố (LATE_RECOMPUTED).
func lateStable(r store.PublishLockManyRow) bool {
	if r.LatePenaltyManual {
		return true
	}
	sum, ok := criteriaSum(r.Criteria)
	if !ok || !r.Total.Valid {
		return false
	}
	pol := grading.LatePolicy{AllowLate: r.AllowLate, PerDay: r.LatePenaltyPerDay, MaxDays: int(r.LateMaxDays)}
	days := grading.LateDays(r.SubmittedAt, r.DueAt)
	return grading.TotalAfterLate(sum, grading.LatePenalty(days, pol, sum)).Equal(r.Total.Decimal)
}

// changedCriteria: tiêu chí có điểm cuối khác điểm AI.
func changedCriteria(criteria, ai json.RawMessage) []string {
	var cs []critRow
	_ = json.Unmarshal(criteria, &cs)
	aiScores := map[string]string{}
	_ = json.Unmarshal(ai, &aiScores)
	out := []string{}
	for _, c := range cs {
		if c.Score == nil {
			continue
		}
		a, has := aiScores[c.ID]
		fin, e1 := decimal.NewFromString(*c.Score)
		prev, e2 := decimal.NewFromString(a)
		if !has || e1 != nil || e2 != nil || !fin.Equal(prev) {
			out = append(out, c.ID)
		}
	}
	return out
}

// Stats là số liệu của bài (SRS 4.8.8). Các tỷ lệ là decimal dạng chuỗi, `null` khi mẫu số 0.
type Stats struct {
	Submitted          int     `json:"submitted"`
	Graded             int     `json:"graded"`
	Reviewed           int     `json:"reviewed"`
	Published          int     `json:"published"`
	MeanAbsDeviation   *string `json:"mean_abs_deviation"`
	TeacherChangedRate *string `json:"teacher_changed_rate"`
	Appealed           int     `json:"appealed"`
	AppealRate         *string `json:"appeal_rate"`
	Adjusted           int     `json:"adjusted"`
	AppealSuccessRate  *string `json:"appeal_success_rate"`
}

// Stats trả số liệu của bài cho nhân viên (M13-A: độ lệch AI ↔ giảng viên).
func (s *Service) Stats(ctx context.Context, course, id uuid.UUID) (Stats, error) {
	q := store.New(s.Pool)
	if _, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: id}); errors.Is(err, pgx.ErrNoRows) {
		return Stats{}, notFound()
	} else if err != nil {
		return Stats{}, fmt.Errorf("assignment: đọc bài: %w", err)
	}
	r, err := q.AssignmentStats(ctx, store.AssignmentStatsParams{CourseID: course, AssignmentID: id})
	if err != nil {
		return Stats{}, fmt.Errorf("assignment: số liệu: %w", err)
	}
	out := Stats{Submitted: int(r.Submitted), Graded: int(r.Graded), Reviewed: int(r.Reviewed), Published: int(r.Published), Appealed: int(r.Appealed), Adjusted: int(r.Adjusted)}
	if r.Published > 0 { // trung bình / tỷ lệ chỉ có nghĩa khi đã có bài công bố
		m, c := r.MeanAbsDeviation.StringFixed(2), r.TeacherChangedRate.StringFixed(4)
		out.MeanAbsDeviation, out.TeacherChangedRate = &m, &c
	}
	if r.Published > 0 {
		v := decimal.NewFromInt(int64(r.Appealed)).Div(decimal.NewFromInt(int64(r.Published))).StringFixed(4)
		out.AppealRate = &v
	}
	if r.Appealed > 0 {
		v := decimal.NewFromInt(int64(r.Adjusted)).Div(decimal.NewFromInt(int64(r.Appealed))).StringFixed(4)
		out.AppealSuccessRate = &v
	}
	return out, nil
}

// VersionCount trả số phiên bản đã khớp của một sinh viên ở bài (đúng lớp: bài ngoài lớp → 404).
func (s *Service) VersionCount(ctx context.Context, course, id, student uuid.UUID) (int, error) {
	q := store.New(s.Pool)
	if _, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: id}); errors.Is(err, pgx.ErrNoRows) {
		return 0, notFound()
	} else if err != nil {
		return 0, fmt.Errorf("assignment: đọc bài: %w", err)
	}
	n, err := q.SubmissionVersionCount(ctx, store.SubmissionVersionCountParams{CourseID: course, AssignmentID: &id, StudentID: &student})
	if err != nil {
		return 0, fmt.Errorf("assignment: đếm phiên bản: %w", err)
	}
	return int(n), nil
}
