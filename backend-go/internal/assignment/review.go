package assignment

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// TopicResultChanged: kết quả chấm đổi (sửa / duyệt) → mục GRADING_REVIEW / GRADING_ERROR của nhân viên đổi. Dùng lại topic course-wide của submission để không thêm bên tiêu thụ.
const TopicResultChanged = "submission.resolved"

// QueueFilter là bộ lọc của hàng chờ (SRS 4.8.1). Status rỗng = mặc định DRAFT.
type QueueFilter struct {
	AssignmentID *uuid.UUID
	Status       string
	NeedsReview  *bool
	Flag         string
	Q            string
	Cursor       string
	Limit        int
}

func validQueueStatus(s string) bool {
	switch s {
	case "NONE", "RUNNING", "DRAFT", "REVIEWED", "PUBLISHED", "FAILED":
		return true
	}
	return false
}

// QueueStudent là sinh viên của dòng; Inactive gắn nhãn sinh viên đã nghỉ / bị loại khỏi lớp.
type QueueStudent struct {
	ID          uuid.UUID `json:"id"`
	FullName    string    `json:"full_name"`
	StudentCode string    `json:"student_code"`
	Inactive    bool      `json:"inactive"`
}

// QueueResult là kết quả hiện hành của dòng.
type QueueResult struct {
	ID          uuid.UUID `json:"id"`
	Status      string    `json:"status"`
	RunState    string    `json:"run_state"`
	Total       *string   `json:"total,omitempty"`
	Flags       []string  `json:"flags"`
	NeedsReview bool      `json:"needs_review"`
	ErrorCode   *string   `json:"error_code,omitempty"`
	Version     int32     `json:"version"`
}

// QueueRow là một dòng của hàng chờ.
type QueueRow struct {
	ID         uuid.UUID `json:"id"`
	Assignment struct {
		ID    uuid.UUID `json:"id"`
		Title string    `json:"title"`
	} `json:"assignment"`
	Student    *QueueStudent `json:"student"`
	Source     string        `json:"source"`
	Version    int32         `json:"version"`
	ReceivedAt time.Time     `json:"received_at"`
	LateDays   int           `json:"late_days"`
	Match      string        `json:"match_status"`
	Result     *QueueResult  `json:"result"`
}

// QueuePage là một trang hàng chờ.
type QueuePage struct {
	Items      []QueueRow `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

type queueCursor struct {
	R int       `json:"r"`
	T int64     `json:"t"`
	I uuid.UUID `json:"i"`
}

// Queue trả hàng chờ chấm theo SRS 4.8.1: mặc định MATCHED + DRAFT, `needs_review` lên đầu rồi `(received_at, id)`; keyset, không OFFSET.
func (s *Service) Queue(ctx context.Context, course uuid.UUID, f QueueFilter) (QueuePage, error) {
	status := f.Status
	if status == "" {
		status = "DRAFT"
	}
	if !validQueueStatus(status) {
		return QueuePage{}, apierr.Validation(apierr.FieldError{Field: "status", Code: "enum", Message: "Trạng thái không hợp lệ."})
	}
	limit := f.Limit
	if limit < 1 || limit > 100 {
		limit = 30
	}
	arg := store.AssignmentQueueParams{CourseID: course, AssignmentID: f.AssignmentID, Status: &status, NeedsReview: f.NeedsReview, PageLimit: int32(limit + 1)} //nolint:gosec // ≤ 101
	if f.Flag != "" {
		arg.Flag = &f.Flag
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		q = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q)
		arg.Q = &q
	}
	if f.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(f.Cursor)
		var c queueCursor
		if err != nil || json.Unmarshal(raw, &c) != nil {
			return QueuePage{}, apierr.Validation(apierr.FieldError{Field: "cursor", Code: "invalid", Message: "cursor không hợp lệ."})
		}
		at := time.UnixMicro(c.T).UTC()
		r := int32(c.R) //nolint:gosec // 0 hoặc 1
		arg.CursorRank, arg.CursorAt, arg.CursorID = &r, &at, &c.I
	}
	rows, err := store.New(s.Pool).AssignmentQueue(ctx, arg)
	if err != nil {
		return QueuePage{}, fmt.Errorf("assignment: hàng chờ: %w", err)
	}
	out := QueuePage{Items: make([]QueueRow, 0, len(rows))}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		raw, _ := json.Marshal(queueCursor{R: int(last.Rank), T: last.ReceivedAt.UnixMicro(), I: last.ID})
		c := base64.RawURLEncoding.EncodeToString(raw)
		out.NextCursor = &c
	}
	for _, r := range rows {
		row := QueueRow{ID: r.ID, Source: r.Source, Version: ptrOr(r.Version, 0), ReceivedAt: r.ReceivedAt, Match: "MATCHED"}
		row.Assignment.ID, row.Assignment.Title = ptrOr(r.AssignmentID, uuid.Nil), r.AssignmentTitle
		row.LateDays = grading.LateDays(r.SubmittedAt, r.DueAt)
		if r.StudentID != nil {
			row.Student = &QueueStudent{ID: *r.StudentID, FullName: r.StudentName, StudentCode: r.StudentCode, Inactive: r.EnrollmentStatus != "" && r.EnrollmentStatus != "ACTIVE"}
		}
		if r.ResultID != nil {
			qr := &QueueResult{ID: *r.ResultID, Status: deref(r.ResultStatus), RunState: deref(r.RunState), Flags: r.Flags, ErrorCode: r.ErrorCode}
			if qr.Flags == nil {
				qr.Flags = []string{}
			}
			if r.Total.Valid {
				t := r.Total.Decimal.StringFixed(2)
				qr.Total = &t
			}
			qr.NeedsReview = r.ResultNeedsReview != nil && *r.ResultNeedsReview
			if r.ResultVersion != nil {
				qr.Version = *r.ResultVersion
			}
			row.Result = qr
		}
		out.Items = append(out.Items, row)
	}
	return out, nil
}

func ptrOr[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// CriterionView là một tiêu chí ở màn duyệt.
type CriterionView struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Max      string             `json:"max"`
	Score    *string            `json:"score"`
	Comment  string             `json:"comment"`
	Edited   bool               `json:"edited"`
	Evidence []grading.Evidence `json:"evidence"`
	AIScore  *string            `json:"ai_score"`
}

// FlaggedRange là câu nghi chèn lệnh (UTF-16 code unit của `extracted_text`).
type FlaggedRange struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Code  string `json:"code"`
}

// LateView là khoản trừ nộp muộn đang áp.
type LateView struct {
	Days    int    `json:"days"`
	Penalty string `json:"penalty"`
	MaxDays int    `json:"max_days"`
	PerDay  string `json:"per_day"`
	Manual  bool   `json:"manual"`
}

// ResultView là chi tiết kết quả chấm cho nhân viên (SRS 4.8.2).
type ResultView struct {
	Result struct {
		ID              uuid.UUID  `json:"id"`
		Round           int16      `json:"round"`
		Status          string     `json:"status"`
		RunState        string     `json:"run_state"`
		ErrorCode       *string    `json:"error_code,omitempty"`
		Total           *string    `json:"total"`
		AITotal         *string    `json:"ai_total_before_late"`
		OverallFeedback string     `json:"overall_feedback"`
		OverallEdited   bool       `json:"overall_edited"`
		ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
		Version         int32      `json:"version"`
	} `json:"result"`
	Submission struct {
		ID            uuid.UUID      `json:"id"`
		Version       int32          `json:"version"`
		FileName      string         `json:"file_name"`
		ReceivedAt    time.Time      `json:"received_at"`
		LateDays      int            `json:"late_days"`
		Source        string         `json:"source"`
		ExtractedText string         `json:"extracted_text"`
		FlaggedRanges []FlaggedRange `json:"flagged_ranges"`
		Superseded    bool           `json:"superseded"`
	} `json:"submission"`
	Assignment struct {
		ID              uuid.UUID            `json:"id"`
		Title           string               `json:"title"`
		MaxScore        string               `json:"max_score"`
		Rubric          []grading.RubricItem `json:"rubric"`
		ReferenceAnswer *string              `json:"reference_answer,omitempty"`
	} `json:"assignment"`
	Criteria   []CriterionView `json:"criteria"`
	Flags      []string        `json:"flags"`
	PassDetail json.RawMessage `json:"pass_detail"`
	Late       LateView        `json:"late"`
	Appeal     *AppealView     `json:"appeal,omitempty"`   // chỉ ở GET chi tiết: phúc khảo của bài (US-P7-10)
	Guidance   *ResultGuidance `json:"guidance,omitempty"` // chỉ ở GET chi tiết khi lớp bật «Lưu ý chấm» (10.6)
}

// ResultGuidance là bản lưu ý ĐÃ GHIM cho kết quả (không phải bản hiện hành), theo tiêu chí.
type ResultGuidance struct {
	Version          int                 `json:"version"`
	CurrentVersion   int                 `json:"current_version"`
	Stale            bool                `json:"stale"`
	ItemsByCriterion map[string][]string `json:"items_by_criterion"`
}

type critRow struct {
	ID       string             `json:"id"`
	Score    *string            `json:"score"`
	Comment  string             `json:"comment"`
	Edited   bool               `json:"edited"`
	Evidence []grading.Evidence `json:"evidence"`
}

func viewOf(d store.ReviewLockRow) (ResultView, error) {
	var v ResultView
	var rubric []grading.RubricItem
	if err := json.Unmarshal(d.Rubric, &rubric); err != nil {
		return v, fmt.Errorf("assignment: rubric hỏng: %w", err)
	}
	var crits []critRow
	_ = json.Unmarshal(d.Criteria, &crits)
	byID := map[string]critRow{}
	for _, c := range crits {
		byID[c.ID] = c
	}
	ai := map[string]string{}
	if len(d.AiScores) > 0 {
		_ = json.Unmarshal(d.AiScores, &ai)
	}
	v.Result.ID, v.Result.Round, v.Result.Status, v.Result.RunState, v.Result.ErrorCode, v.Result.Version = d.ID, d.Round, d.Status, d.RunState, d.ErrorCode, d.Version
	v.Result.ReviewedAt = d.ReviewedAt
	if d.Total.Valid {
		t := d.Total.Decimal.StringFixed(2)
		v.Result.Total = &t
	}
	if d.AiTotalBeforeLate.Valid {
		t := d.AiTotalBeforeLate.Decimal.StringFixed(2)
		v.Result.AITotal = &t
	}
	if d.OverallFeedback != nil {
		v.Result.OverallFeedback = *d.OverallFeedback
	}
	var pd map[string]json.RawMessage
	_ = json.Unmarshal(d.PassDetail, &pd)
	v.Result.OverallEdited = string(pd["_overall_edited"]) == "true"
	v.Submission.ID, v.Submission.Version, v.Submission.FileName, v.Submission.ReceivedAt, v.Submission.Source = d.SubmissionID, ptrOr(d.SubVersion, 0), d.FileName, d.ReceivedAt, d.Source
	if d.ExtractedText != nil {
		v.Submission.ExtractedText = *d.ExtractedText
	}
	v.Submission.Superseded = d.Superseded
	v.Submission.FlaggedRanges = []FlaggedRange{}
	var hits []grading.InjectionHit
	_ = json.Unmarshal(pd["_injection"], &hits)
	for _, h := range hits {
		code := "INJECTION_SUSPECTED"
		if len(h.Codes) > 0 {
			code = h.Codes[0]
		}
		v.Submission.FlaggedRanges = append(v.Submission.FlaggedRanges, FlaggedRange{Start: h.Start, End: h.End, Code: code})
	}
	v.Assignment.ID, v.Assignment.Title, v.Assignment.MaxScore, v.Assignment.Rubric, v.Assignment.ReferenceAnswer = ptrOr(d.AssignmentID, uuid.Nil), d.Title, d.MaxScore.StringFixed(2), rubric, d.ReferenceAnswer
	for _, r := range rubric {
		c := byID[r.ID]
		cv := CriterionView{ID: r.ID, Name: r.Name, Max: r.MaxScore, Score: c.Score, Comment: c.Comment, Edited: c.Edited, Evidence: c.Evidence}
		if cv.Evidence == nil {
			cv.Evidence = []grading.Evidence{}
		}
		if a, ok := ai[r.ID]; ok {
			cv.AIScore = &a
		}
		v.Criteria = append(v.Criteria, cv)
	}
	v.Flags = d.Flags
	if v.Flags == nil {
		v.Flags = []string{}
	}
	v.PassDetail = d.PassDetail
	if len(v.PassDetail) == 0 {
		v.PassDetail = json.RawMessage("null")
	}
	days := grading.LateDays(d.SubmittedAt, d.DueAt)
	v.Submission.LateDays = days
	v.Late = LateView{Days: int(d.LateDays), Penalty: d.LatePenalty.StringFixed(2), MaxDays: int(d.LateMaxDays), PerDay: d.LatePenaltyPerDay.StringFixed(2), Manual: d.LatePenaltyManual}
	return v, nil
}

func detailAsLock(d store.ReviewDetailRow) store.ReviewLockRow { return store.ReviewLockRow(d) }

// ResultDetail trả chi tiết một kết quả (nhân viên).
// `id` là id kết quả HOẶC id bài nộp (màn `/grading/[submissionId]` mở theo bài nộp: lấy kết quả hiện hành của nó).
func (s *Service) ResultDetail(ctx context.Context, course, id uuid.UUID) (ResultView, error) {
	q := store.New(s.Pool)
	d, err := q.ReviewDetail(ctx, store.ReviewDetailParams{CourseID: course, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		if cur, cerr := q.GradingCurrent(ctx, store.GradingCurrentParams{CourseID: course, SubmissionID: id}); cerr == nil {
			d, err = q.ReviewDetail(ctx, store.ReviewDetailParams{CourseID: course, ID: cur.ID})
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ResultView{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return ResultView{}, fmt.Errorf("assignment: chi tiết kết quả: %w", err)
	}
	v, err := viewOf(detailAsLock(d))
	if err != nil {
		return v, err
	}
	if apid, aerr := q.GradeAppealIDByResult(ctx, store.GradeAppealIDByResultParams{CourseID: course, ResultID: v.Result.ID}); aerr == nil {
		ap, gerr := s.GetAppeal(ctx, course, apid)
		if gerr != nil {
			return v, gerr
		}
		v.Appeal = &ap
	} else if !errors.Is(aerr, pgx.ErrNoRows) {
		return v, fmt.Errorf("assignment: phúc khảo của kết quả: %w", aerr)
	}
	if s.Guidance.On(course, grading.FlagGradingGuidance) {
		g, gerr := q.GuidanceForResult(ctx, store.GuidanceForResultParams{CourseID: course, ID: v.Result.ID})
		if gerr != nil {
			return v, fmt.Errorf("assignment: lưu ý của kết quả: %w", gerr)
		}
		if g.CurrentVersion > 0 || g.GuidanceVersion > 0 {
			items, ierr := q.GuidanceSetAt(ctx, store.GuidanceSetAtParams{CourseID: course, AssignmentID: g.AssignmentID, V: g.GuidanceVersion})
			if ierr != nil {
				return v, fmt.Errorf("assignment: tập lưu ý đã ghim: %w", ierr)
			}
			rg := &ResultGuidance{Version: int(g.GuidanceVersion), CurrentVersion: int(g.CurrentVersion), ItemsByCriterion: map[string][]string{},
				Stale: g.GuidanceVersion < g.CurrentVersion && g.Status == "DRAFT" && !g.WasReviewed}
			for _, it := range items {
				rg.ItemsByCriterion[it.CriterionID] = append(rg.ItemsByCriterion[it.CriterionID], it.Text)
			}
			v.Guidance = rg
		}
	}
	return v, nil
}

// CriterionEdit là một tiêu chí được sửa; `nil` = giữ nguyên.
type CriterionEdit struct {
	ID      string  `json:"id"`
	Score   *Num    `json:"score"`
	Comment *string `json:"comment"`
}

// Num nhận điểm ở dạng chuỗi hoặc số JSON (`"2,25"`, `2.25`) — điểm luôn qua decimal, không float.
type Num string

func (n *Num) UnmarshalJSON(b []byte) error {
	var s string
	if len(b) > 0 && b[0] == '"' {
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
	} else {
		s = string(b)
	}
	*n = Num(s)
	return nil
}

// EditIn là thân `PUT …/grading-results/{id}`.
type EditIn struct {
	Version         int32           `json:"version"`
	Criteria        []CriterionEdit `json:"criteria"`
	OverallFeedback *string         `json:"overall_feedback"`
	LatePenalty     *Num            `json:"late_penalty"`
}

func scoreStep() decimal.Decimal { return decimal.New(25, -2) }

// parseScore: chấp nhận dấu phẩy; ≤ 2 chữ số thập phân (SCORE_PRECISION), trong [0; max] (SCORE_OUT_OF_RANGE), bội 0,25 (SCORE_STEP).
func parseScore(raw string, max decimal.Decimal, step bool) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(strings.Replace(strings.TrimSpace(raw), ",", ".", 1))
	if err != nil {
		return d, apierr.New(http.StatusUnprocessableEntity, apierr.ScoreOutOfRange)
	}
	if !d.Equal(d.Round(2)) {
		return d, apierr.New(http.StatusUnprocessableEntity, apierr.ScorePrecision)
	}
	if d.IsNegative() || d.GreaterThan(max) {
		return d, apierr.New(http.StatusUnprocessableEntity, apierr.ScoreOutOfRange)
	}
	if step && !d.Mod(scoreStep()).IsZero() {
		return d, apierr.New(http.StatusUnprocessableEntity, apierr.ScoreStep)
	}
	return d, nil
}

func (s *Service) lockResult(ctx context.Context, q *store.Queries, course, id uuid.UUID) (store.ReviewLockRow, error) {
	d, err := q.ReviewLock(ctx, store.ReviewLockParams{CourseID: course, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return d, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return d, fmt.Errorf("assignment: khoá kết quả: %w", err)
	}
	return d, nil
}

func (s *Service) resultAudit(ctx context.Context, q *store.Queries, a Actor, course, id uuid.UUID, action string, before, after any) error {
	b, _ := json.Marshal(before)
	af, _ := json.Marshal(after)
	var trace *string
	if a.TraceID != "" {
		trace = &a.TraceID
	}
	_, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &a.UserID, Entity: "grading_results", EntityID: id.String(), Action: action, Before: b, After: af, TraceID: trace})
	return err
}

func conflictWith(v ResultView) *apierr.Error {
	return apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current": v})
}

// guardEditable: PUBLISHED → RESULT_PUBLISHED; đang xếp hàng / chạy → NOT_READY; bài nộp đã bị thay → SUPERSEDED. FAILED sửa được (nhập điểm tay).
func guardEditable(d store.ReviewLockRow) error {
	switch {
	case d.Status == "PUBLISHED":
		return apierr.New(http.StatusConflict, apierr.ResultPublished)
	case d.RunState == "QUEUED" || d.RunState == "RUNNING":
		return apierr.New(http.StatusConflict, apierr.NotReady)
	case d.Superseded:
		return apierr.New(http.StatusConflict, apierr.Superseded)
	}
	return nil
}

// EditResult sửa điểm / nhận xét / khoản trừ muộn tại chỗ (SRS 4.8.3). Khoá hàng, kiểm version, tính lại tổng bằng decimal, REVIEWED → DRAFT, `audit_log` trước / sau, `ai_scores` giữ nguyên.
func (s *Service) EditResult(ctx context.Context, a Actor, course, id uuid.UUID, in EditIn) (ResultView, error) {
	var out ResultView
	err := s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		d, err := s.lockResult(ctx, q, course, id)
		if err != nil {
			return err
		}
		if d.Version != in.Version {
			cur, verr := viewOf(d)
			if verr != nil {
				return verr
			}
			return conflictWith(cur)
		}
		if err := guardEditable(d); err != nil {
			return err
		}
		var rubric []grading.RubricItem
		if err := json.Unmarshal(d.Rubric, &rubric); err != nil {
			return fmt.Errorf("assignment: rubric hỏng: %w", err)
		}
		var crits []critRow
		_ = json.Unmarshal(d.Criteria, &crits)
		byID := map[string]*critRow{}
		for i := range crits {
			byID[crits[i].ID] = &crits[i]
		}
		maxOf := map[string]decimal.Decimal{}
		for _, r := range rubric {
			m, _ := decimal.NewFromString(r.MaxScore)
			maxOf[r.ID] = m
			if byID[r.ID] == nil {
				crits = append(crits, critRow{ID: r.ID, Evidence: []grading.Evidence{}})
				byID = map[string]*critRow{}
				for i := range crits {
					byID[crits[i].ID] = &crits[i]
				}
			}
		}
		before := map[string]any{"criteria": snapshot(crits), "total": totalStr(d.Total), "late_penalty": d.LatePenalty.StringFixed(2), "overall_feedback": deref(d.OverallFeedback)}
		for _, e := range in.Criteria {
			c := byID[e.ID]
			if c == nil {
				return apierr.New(http.StatusUnprocessableEntity, apierr.CriterionUnknown)
			}
			if e.Score != nil {
				sc, err := parseScore(string(*e.Score), maxOf[e.ID], true)
				if err != nil {
					return err
				}
				ns := sc.StringFixed(2)
				if c.Score == nil || *c.Score != ns {
					c.Score, c.Edited = &ns, true
				}
			}
			if e.Comment != nil && *e.Comment != c.Comment {
				c.Comment, c.Edited = *e.Comment, true
			}
		}
		sum, complete := criteriaSum(mustJSON(crits))
		feedback := d.OverallFeedback
		overallEdited := false
		if in.OverallFeedback != nil && *in.OverallFeedback != deref(d.OverallFeedback) {
			feedback, overallEdited = in.OverallFeedback, true
		}
		pol := grading.LatePolicy{AllowLate: d.AllowLate, PerDay: d.LatePenaltyPerDay, MaxDays: int(d.LateMaxDays)}
		days := grading.LateDays(d.SubmittedAt, d.DueAt)
		penalty, manual := d.LatePenalty, d.LatePenaltyManual
		if in.LatePenalty != nil {
			p, err := parseScore(string(*in.LatePenalty), sum, false)
			if err != nil {
				return err
			}
			penalty, manual = p, true
		} else if !manual {
			penalty = grading.LatePenalty(days, pol, sum)
		}
		if !manual {
			d.LateDays = int16(days) //nolint:gosec // ≤ 14 + cắt ở LateDays
		}
		runState := d.RunState
		var total decimal.NullDecimal
		if complete {
			total = decimal.NullDecimal{Decimal: grading.TotalAfterLate(sum, penalty), Valid: true}
			runState = "DONE"
		} else {
			penalty = decimal.Zero
		}
		detail := markOverall(d.PassDetail, overallEdited || pdOverall(d.PassDetail))
		n, err := q.ReviewApplyEdit(ctx, store.ReviewApplyEditParams{Criteria: mustJSON(crits), OverallFeedback: feedback, Total: total, LateDays: d.LateDays, LatePenalty: penalty, LatePenaltyManual: manual,
			RunState: runState, PassDetail: detail, CourseID: course, ID: id, Version: in.Version})
		if err != nil {
			return fmt.Errorf("assignment: ghi bản sửa: %w", err)
		}
		if n == 0 {
			return conflictWith(ResultView{})
		}
		nd, err := s.lockResult(ctx, q, course, id)
		if err != nil {
			return err
		}
		after := map[string]any{"criteria": snapshot(crits), "total": totalStr(nd.Total), "late_penalty": nd.LatePenalty.StringFixed(2), "overall_feedback": deref(nd.OverallFeedback), "status": nd.Status}
		if err := s.resultAudit(ctx, q, a, course, id, "grading_result.edit", before, after); err != nil {
			return fmt.Errorf("assignment: audit: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicResultChanged, map[string]any{"course_id": course}); err != nil {
			return fmt.Errorf("assignment: outbox: %w", err)
		}
		out, err = viewOf(nd)
		return err
	})
	return out, err
}

func snapshot(cs []critRow) map[string]any {
	m := map[string]any{}
	for _, c := range cs {
		m[c.ID] = map[string]any{"score": c.Score, "comment": c.Comment}
	}
	return m
}

func totalStr(t decimal.NullDecimal) *string {
	if !t.Valid {
		return nil
	}
	s := t.Decimal.StringFixed(2)
	return &s
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func pdOverall(raw json.RawMessage) bool {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	return string(m["_overall_edited"]) == "true"
}

// markOverall ghi `_overall_edited` vào pass_detail (khoá riêng, chỉ nhân viên đọc).
func markOverall(raw json.RawMessage, edited bool) json.RawMessage {
	m := map[string]json.RawMessage{}
	_ = json.Unmarshal(raw, &m)
	if edited {
		m["_overall_edited"] = json.RawMessage("true")
	}
	if len(m) == 0 {
		return nil
	}
	return mustJSON(m)
}

// Review duyệt nháp (SRS 4.8.4): DRAFT → REVIEWED. Idempotent khi đã REVIEWED; thiếu điểm → SCORES_REQUIRED; đang chạy → NOT_READY; bản cũ → SUPERSEDED.
func (s *Service) Review(ctx context.Context, a Actor, course, id uuid.UUID, version int32) (ResultView, error) {
	var out ResultView
	err := s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		d, err := s.lockResult(ctx, q, course, id)
		if err != nil {
			return err
		}
		if d.Status == "REVIEWED" {
			out, err = viewOf(d)
			return err
		}
		if d.Version != version {
			cur, verr := viewOf(d)
			if verr != nil {
				return verr
			}
			return conflictWith(cur)
		}
		if err := guardEditable(d); err != nil {
			return err
		}
		if _, ok := criteriaSum(d.Criteria); !ok {
			return apierr.New(http.StatusUnprocessableEntity, apierr.ScoresRequired)
		}
		n, err := q.ReviewMark(ctx, store.ReviewMarkParams{Reviewer: &a.UserID, CourseID: course, ID: id, Version: version})
		if err != nil {
			return fmt.Errorf("assignment: duyệt: %w", err)
		}
		if n == 0 {
			return conflictWith(ResultView{})
		}
		nd, err := s.lockResult(ctx, q, course, id)
		if err != nil {
			return err
		}
		if err := s.resultAudit(ctx, q, a, course, id, "grading_result.review", map[string]any{"status": d.Status}, map[string]any{"status": nd.Status, "total": totalStr(nd.Total), "flags": nd.Flags}); err != nil {
			return fmt.Errorf("assignment: audit: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicResultChanged, map[string]any{"course_id": course}); err != nil {
			return fmt.Errorf("assignment: outbox: %w", err)
		}
		out, err = viewOf(nd)
		return err
	})
	return out, err
}

// SubmissionVersion là một phiên bản trong chi tiết bài nộp.
type SubmissionVersion struct {
	ID         uuid.UUID    `json:"id"`
	Version    *int32       `json:"version"`
	Source     string       `json:"source"`
	FileName   string       `json:"file_name"`
	FileSize   int64        `json:"file_size"`
	Match      string       `json:"match_status"`
	ReceivedAt time.Time    `json:"received_at"`
	LateDays   int          `json:"late_days"`
	Current    bool         `json:"current"` // bản mới nhất (bản được chấm)
	Result     *QueueResult `json:"result"`
}

// SubmissionDetail là chi tiết một bài nộp cho nhân viên: người nộp, bài tập và mọi phiên bản (SRS 6 hàng 15).
type SubmissionDetail struct {
	ID         uuid.UUID `json:"id"`
	Assignment *struct {
		ID    uuid.UUID `json:"id"`
		Title string    `json:"title"`
	} `json:"assignment"`
	Student         *QueueStudent       `json:"student"`
	Match           string              `json:"match_status"`
	UnmatchedReason *string             `json:"unmatched_reason,omitempty"`
	Versions        []SubmissionVersion `json:"versions"`
}

// SubmissionDetail trả bài nộp `id` cùng mọi phiên bản MATCHED của cùng (bài tập, sinh viên), mới nhất trước. Không thuộc lớp → 404. Chỉ nhân viên (route); không trả URL tệp (đã có `…/file`).
func (s *Service) SubmissionDetail(ctx context.Context, course, id uuid.UUID) (SubmissionDetail, error) {
	rows, err := store.New(s.Pool).SubmissionVersions(ctx, store.SubmissionVersionsParams{CourseID: course, ID: id})
	if err != nil {
		return SubmissionDetail{}, fmt.Errorf("assignment: các phiên bản bài nộp: %w", err)
	}
	var out SubmissionDetail
	for _, r := range rows {
		if !r.Requested {
			continue
		}
		out.ID, out.Match, out.UnmatchedReason = r.ID, r.MatchStatus, r.UnmatchedReason
		if r.AssignmentID != nil && r.AssignmentTitle != nil {
			out.Assignment = &struct {
				ID    uuid.UUID `json:"id"`
				Title string    `json:"title"`
			}{*r.AssignmentID, *r.AssignmentTitle}
		}
		if r.StudentID != nil {
			out.Student = &QueueStudent{ID: *r.StudentID, FullName: r.StudentName, StudentCode: r.StudentCode, Inactive: r.EnrollmentStatus != "" && r.EnrollmentStatus != "ACTIVE"}
		}
	}
	if out.ID == uuid.Nil {
		return SubmissionDetail{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	out.Versions = make([]SubmissionVersion, 0, len(rows))
	for i, r := range rows {
		v := SubmissionVersion{ID: r.ID, Version: r.Version, Source: r.Source, FileName: r.FileName, FileSize: r.FileSize, Match: r.MatchStatus, ReceivedAt: r.ReceivedAt, Current: i == 0}
		if r.DueAt != nil {
			v.LateDays = grading.LateDays(r.SubmittedAt, *r.DueAt)
		}
		if r.ResultID != nil {
			qr := &QueueResult{ID: *r.ResultID, Status: deref(r.ResultStatus), RunState: deref(r.RunState), Flags: r.Flags, ErrorCode: r.ErrorCode, NeedsReview: r.ResultNeedsReview != nil && *r.ResultNeedsReview}
			if qr.Flags == nil {
				qr.Flags = []string{}
			}
			if r.Total.Valid {
				t := r.Total.Decimal.StringFixed(2)
				qr.Total = &t
			}
			if r.ResultVersion != nil {
				qr.Version = *r.ResultVersion
			}
			v.Result = qr
		}
		out.Versions = append(out.Versions, v)
	}
	return out, nil
}
