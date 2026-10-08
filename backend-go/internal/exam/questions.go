package exam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// TopicQuestionReviewed: outbox khi một câu đổi trạng thái duyệt (xoá cache "Hôm nay" của Staff — nguồn việc QUESTION_REVIEW).
const TopicQuestionReviewed = "question.reviewed"

// Option là một đáp án trong phản hồi cho Staff.
type Option struct {
	ID         uuid.UUID `json:"id"`
	Position   int       `json:"position"`
	Body       string    `json:"body"`
	PinnedLast bool      `json:"pinned_last"`
}

// ExamRef là một bài thi đang dùng câu hỏi.
type ExamRef struct {
	ID     uuid.UUID `json:"id"`
	Title  string    `json:"title"`
	Status string    `json:"status"`
}

// Reference là lời giải mẫu (chỉ Staff).
type Reference struct {
	Language string `json:"language"`
	Source   string `json:"source"`
}

// TestStats tóm tắt bộ test (chỉ test `approved`).
type TestStats struct {
	Total       int `json:"total"`
	Samples     int `json:"samples"`
	Hidden      int `json:"hidden"`
	TotalWeight int `json:"total_weight"`
}

// CodeDetail là cấu hình bài code kèm tóm tắt test.
type CodeDetail struct {
	Languages                []string        `json:"languages"`
	TimeLimitMS              int             `json:"time_limit_ms"`
	MemoryLimitMB            int             `json:"memory_limit_mb"`
	OutputLimitKB            int             `json:"output_limit_kb"`
	Checker                  string          `json:"checker"`
	FloatEps                 *string         `json:"float_eps"`
	StarterCode              json.RawMessage `json:"starter_code"`
	Reference                *Reference      `json:"reference"`
	ReferenceVerifiedVersion *int            `json:"reference_verified_version"`
	ReferenceVerifiedAt      *time.Time      `json:"reference_verified_at"`
	TestsVersion             int             `json:"tests_version"`
	Tests                    TestStats       `json:"tests"`
}

// QuestionDetail là bản đầy đủ dành cho Staff (đáp án, giải thích, test, lời giải mẫu).
type QuestionDetail struct {
	ID           uuid.UUID       `json:"id"`
	Type         string          `json:"type"`
	Title        string          `json:"title"`
	Topic        string          `json:"topic"`
	Difficulty   string          `json:"difficulty"`
	Stem         string          `json:"stem"`
	Explanation  *string         `json:"explanation"`
	Origin       string          `json:"origin"`
	ReviewStatus string          `json:"review_status"`
	Options      []Option        `json:"options"`
	AnswerKey    json.RawMessage `json:"answer_key"`
	Code         *CodeDetail     `json:"code,omitempty"`
	UsedInExams  []ExamRef       `json:"used_in_exams"`
	ReviewedBy   *uuid.UUID      `json:"reviewed_by"`
	ReviewedAt   *time.Time      `json:"reviewed_at"`
	ArchivedAt   *time.Time      `json:"archived_at"`
	AIJobID      *uuid.UUID      `json:"ai_job_id"`
	Version      int             `json:"version"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	// Warnings: cảnh báo không chặn khi tạo / sửa (đáp án kiểu "Tất cả các đáp án trên" chưa ghim cuối).
	Warnings []apierr.FieldError `json:"-"`
}

func lockingStatus(st string) bool {
	return st == "SCHEDULED" || st == "OPEN" || st == "CLOSED" || st == "PUBLISHED"
}

func (s *Service) detail(ctx context.Context, q *store.Queries, row store.QuestionBank) (QuestionDetail, error) {
	d := QuestionDetail{
		ID: row.ID, Type: string(row.Type), Title: row.Title, Topic: row.Topic, Difficulty: string(row.Difficulty), Stem: row.Stem, Explanation: row.Explanation,
		Origin: string(row.Origin), ReviewStatus: string(row.ReviewStatus), AnswerKey: row.AnswerKey, ReviewedBy: row.ReviewedBy, ReviewedAt: row.ReviewedAt,
		ArchivedAt: row.ArchivedAt, AIJobID: row.AiJobID, Version: int(row.Version), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Options: []Option{}, UsedInExams: []ExamRef{},
	}
	opts, err := q.QuestionOptions(ctx, store.QuestionOptionsParams{CourseID: row.CourseID, QuestionID: row.ID})
	if err != nil {
		return d, fmt.Errorf("exam: đọc đáp án: %w", err)
	}
	for _, o := range opts {
		d.Options = append(d.Options, Option{ID: o.ID, Position: int(o.Position), Body: o.Body, PinnedLast: o.PinnedLast})
	}
	use, err := q.QuestionUsage(ctx, store.QuestionUsageParams{CourseID: row.CourseID, QuestionID: row.ID})
	if err != nil {
		return d, fmt.Errorf("exam: đọc bài thi đang dùng: %w", err)
	}
	for _, u := range use {
		d.UsedInExams = append(d.UsedInExams, ExamRef{ID: u.ID, Title: u.Title, Status: string(u.Status)})
	}
	if row.Type == store.QuestionTypeCODE {
		cd, err := s.codeDetail(ctx, q, row.CourseID, row.ID)
		if err != nil {
			return d, err
		}
		d.Code = &cd
	}
	return d, nil
}

func (s *Service) codeDetail(ctx context.Context, q *store.Queries, courseID, qid uuid.UUID) (CodeDetail, error) {
	cp, err := q.CodeProblemGet(ctx, store.CodeProblemGetParams{CourseID: courseID, QuestionID: qid})
	if err != nil {
		return CodeDetail{}, fmt.Errorf("exam: đọc bài code: %w", err)
	}
	st, err := q.TestcaseStats(ctx, store.TestcaseStatsParams{CourseID: courseID, ProblemID: qid})
	if err != nil {
		return CodeDetail{}, fmt.Errorf("exam: thống kê test: %w", err)
	}
	cd := CodeDetail{
		Languages: cp.Languages, TimeLimitMS: int(cp.TimeLimitMs), MemoryLimitMB: int(cp.MemoryLimitMb), OutputLimitKB: int(cp.OutputLimitKb), Checker: string(cp.Checker),
		StarterCode: cp.StarterCode, ReferenceVerifiedAt: cp.ReferenceVerifiedAt, TestsVersion: int(cp.TestsVersion),
		Tests: TestStats{Total: int(st.Total), Samples: int(st.Samples), Hidden: int(st.Hidden), TotalWeight: int(st.TotalWeight)},
	}
	if cp.FloatEps.Valid {
		e := cp.FloatEps.Decimal.String()
		cd.FloatEps = &e
	}
	if cp.ReferenceVerifiedVersion != nil {
		v := int(*cp.ReferenceVerifiedVersion)
		cd.ReferenceVerifiedVersion = &v
	}
	if cp.ReferenceLanguage != nil && cp.ReferenceSource != nil {
		cd.Reference = &Reference{Language: *cp.ReferenceLanguage, Source: *cp.ReferenceSource}
	}
	return cd, nil
}

// Create tạo câu ở DRAFT (`origin=MANUAL`). Câu CODE tạo kèm dòng `code_problems` mặc định.
func (s *Service) Create(ctx context.Context, actor, courseID uuid.UUID, in QuestionIn) (QuestionDetail, error) {
	c, errs, warns := Validate(in)
	if len(errs) > 0 {
		return QuestionDetail{}, apierr.Validation(errs...)
	}
	var out QuestionDetail
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		row, err := s.insertQuestion(ctx, q, courseID, actor, c, store.QuestionOriginMANUAL, store.QuestionReviewStatusDRAFT, nil)
		if err != nil {
			return err
		}
		out, err = s.detail(ctx, q, row)
		return err
	})
	out.Warnings = warns
	return out, err
}

// insertQuestion ghi câu + đáp án (id sinh phía Go để `answer_key` trỏ đúng id) + dòng `code_problems` nếu là CODE.
func (s *Service) insertQuestion(ctx context.Context, q *store.Queries, courseID, actor uuid.UUID, c Checked, origin store.QuestionOrigin, status store.QuestionReviewStatus, aiJob *uuid.UUID) (store.QuestionBank, error) {
	ids := make([]uuid.UUID, len(c.Options))
	for i := range ids {
		id, err := uuid.NewV7()
		if err != nil {
			return store.QuestionBank{}, fmt.Errorf("exam: sinh id đáp án: %w", err)
		}
		ids[i] = id
	}
	row, err := q.QuestionInsert(ctx, store.QuestionInsertParams{
		CourseID: courseID, Type: store.QuestionType(c.Type), Title: c.Title, Topic: c.Topic, Difficulty: store.QuestionDifficulty(c.Difficulty), Stem: c.Stem,
		AnswerKey: AnswerKey(c, ids), Explanation: c.Explanation, Origin: origin, ReviewStatus: status, CreatedBy: actor, AiJobID: aiJob,
	})
	if err != nil {
		return store.QuestionBank{}, fmt.Errorf("exam: ghi câu hỏi: %w", err)
	}
	for i, o := range c.Options {
		if _, err := q.QuestionOptionInsert(ctx, store.QuestionOptionInsertParams{ID: ids[i], CourseID: courseID, QuestionID: row.ID, Position: int16(i + 1), Body: o.Body, PinnedLast: o.PinnedLast}); err != nil { //nolint:gosec // ≤ 8 đáp án
			return store.QuestionBank{}, fmt.Errorf("exam: ghi đáp án: %w", err)
		}
	}
	if c.Type == TypeCode {
		if _, err := q.CodeProblemInsert(ctx, store.CodeProblemInsertParams{CourseID: courseID, QuestionID: row.ID}); err != nil {
			return store.QuestionBank{}, fmt.Errorf("exam: ghi bài code: %w", err)
		}
	}
	return row, nil
}

// Get trả câu đầy đủ cho Staff; câu của lớp khác → 404.
func (s *Service) Get(ctx context.Context, courseID, id uuid.UUID) (QuestionDetail, error) {
	q := store.New(s.Pool)
	row, err := q.QuestionGet(ctx, store.QuestionGetParams{CourseID: courseID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return QuestionDetail{}, notFound()
	}
	if err != nil {
		return QuestionDetail{}, fmt.Errorf("exam: đọc câu hỏi: %w", err)
	}
	return s.detail(ctx, q, row)
}

// ListFilter là bộ lọc của danh sách (rỗng = không lọc).
type ListFilter struct {
	ReviewStatus, Topic, Difficulty, Type, Origin, Q string
	Archived                                         bool
}

// Cursor là con trỏ danh sách (created_at, id).
type Cursor struct {
	At time.Time
	ID uuid.UUID
}

// ListItem là một hàng của danh sách (không có đề / đáp án).
type ListItem struct {
	ID           uuid.UUID  `json:"id"`
	Type         string     `json:"type"`
	Title        string     `json:"title"`
	Topic        string     `json:"topic"`
	Difficulty   string     `json:"difficulty"`
	ReviewStatus string     `json:"review_status"`
	Origin       string     `json:"origin"`
	UsedInExams  int        `json:"used_in_exams"`
	Version      int        `json:"version"`
	ArchivedAt   *time.Time `json:"archived_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CreatedAt    time.Time  `json:"-"`
}

func nz(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ptrOf trả con trỏ tới giá trị đã đổi kiểu, hoặc nil khi chuỗi rỗng (lọc không đặt).
func ptrOf[T ~string](s string) *T {
	if s == "" {
		return nil
	}
	v := T(s)
	return &v
}

// List: `rows` đã lấy limit+1 dòng (người gọi cắt bằng httpx.Paginate).
func (s *Service) List(ctx context.Context, courseID uuid.UUID, f ListFilter, cur *Cursor, fetch int) ([]ListItem, error) {
	arg := store.QuestionListParams{
		CourseID: courseID, Archived: f.Archived, MaxRows: int32(fetch), //nolint:gosec // ≤ 101
		ReviewStatus: ptrOf[store.QuestionReviewStatus](f.ReviewStatus), Topic: nz(f.Topic), Difficulty: ptrOf[store.QuestionDifficulty](f.Difficulty),
		Type: ptrOf[store.QuestionType](f.Type), Origin: ptrOf[store.QuestionOrigin](f.Origin), Q: nz(f.Q),
	}
	if cur != nil {
		arg.CursorAt, arg.CursorID = &cur.At, &cur.ID
	}
	rows, err := store.New(s.Pool).QuestionList(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("exam: danh sách câu hỏi: %w", err)
	}
	out := make([]ListItem, len(rows))
	for i, r := range rows {
		out[i] = ListItem{ID: r.ID, Type: string(r.Type), Title: r.Title, Topic: r.Topic, Difficulty: string(r.Difficulty), ReviewStatus: string(r.ReviewStatus), Origin: string(r.Origin),
			UsedInExams: int(r.UsedInExams), Version: int(r.Version), ArchivedAt: r.ArchivedAt, UpdatedAt: r.UpdatedAt, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

func inUse(use []store.QuestionUsageRow) []store.QuestionUsageRow {
	return slices.DeleteFunc(slices.Clone(use), func(u store.QuestionUsageRow) bool { return !lockingStatus(string(u.Status)) })
}

func inUseError(use []store.QuestionUsageRow) *apierr.Error {
	exams := make([]map[string]any, len(use))
	for i, u := range use {
		exams[i] = map[string]any{"id": u.ID, "title": u.Title}
	}
	return apierr.New(http.StatusConflict, apierr.QuestionInUse).WithDetails(map[string]any{"exams": exams})
}

// optionsSame: danh sách đáp án (thứ tự, nội dung, ghim) và tập đáp án đúng không đổi.
func optionsSame(old []store.QuestionOptionsRow, key json.RawMessage, c Checked) bool {
	if len(old) != len(c.Options) {
		return false
	}
	for i, o := range old {
		if o.Body != c.Options[i].Body || o.PinnedLast != c.Options[i].PinnedLast {
			return false
		}
	}
	var k struct {
		OptionIDs []string `json:"option_ids"`
	}
	_ = json.Unmarshal(key, &k)
	got := map[string]bool{}
	for _, id := range k.OptionIDs {
		got[id] = true
	}
	if len(got) != len(c.CorrectIdx) {
		return false
	}
	for _, i := range c.CorrectIdx {
		if !got[old[i].ID.String()] {
			return false
		}
	}
	return true
}

func sameValue(old json.RawMessage, v *bool) bool {
	var k struct {
		Value *bool `json:"value"`
	}
	_ = json.Unmarshal(old, &k)
	return k.Value != nil && v != nil && *k.Value == *v
}

// Update sửa câu (khoá lạc quan theo `version`). Câu đang được bài thi SCHEDULED+ dùng chỉ cho sửa chủ đề / độ khó / giải thích (409 QUESTION_IN_USE).
// Sửa đề / đáp án đưa câu APPROVED / REJECTED về DRAFT.
func (s *Service) Update(ctx context.Context, courseID, id uuid.UUID, in QuestionIn, version int) (QuestionDetail, error) {
	c, errs, warns := Validate(in)
	var out QuestionDetail
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		row, err := q.QuestionLock(ctx, store.QuestionLockParams{CourseID: courseID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: khoá câu hỏi: %w", err)
		}
		if string(row.Type) != c.Type && c.Type != "" {
			errs = append(errs, apierr.FieldError{Field: "type", Code: "TYPE_IMMUTABLE", Message: "Không đổi được loại câu hỏi."})
		}
		if int(row.Version) != version {
			cur, derr := s.detail(ctx, q, row)
			if derr != nil {
				return derr
			}
			return &VersionConflict{Version: int(row.Version), Current: cur}
		}
		if len(errs) > 0 {
			return apierr.Validation(errs...)
		}
		old, err := q.QuestionOptions(ctx, store.QuestionOptionsParams{CourseID: courseID, QuestionID: id})
		if err != nil {
			return fmt.Errorf("exam: đọc đáp án: %w", err)
		}
		same := false
		switch row.Type {
		case store.QuestionTypeMCQSINGLE, store.QuestionTypeMCQMULTI:
			same = optionsSame(old, row.AnswerKey, c)
		case store.QuestionTypeTRUEFALSE:
			same = sameValue(row.AnswerKey, c.Value)
		default:
			same = true
		}
		contentChanged := !same || c.Title != row.Title || c.Stem != row.Stem
		if contentChanged {
			use, err := q.QuestionUsage(ctx, store.QuestionUsageParams{CourseID: courseID, QuestionID: id})
			if err != nil {
				return fmt.Errorf("exam: đọc bài thi đang dùng: %w", err)
			}
			if u := inUse(use); len(u) > 0 {
				return inUseError(u)
			}
		}
		key := json.RawMessage(row.AnswerKey)
		if !same && (row.Type == store.QuestionTypeMCQSINGLE || row.Type == store.QuestionTypeMCQMULTI) {
			if err := q.QuestionOptionsDelete(ctx, store.QuestionOptionsDeleteParams{CourseID: courseID, QuestionID: id}); err != nil {
				return fmt.Errorf("exam: xoá đáp án cũ: %w", err)
			}
			ids := make([]uuid.UUID, len(c.Options))
			for i, o := range c.Options {
				ids[i], _ = uuid.NewV7()
				if _, err := q.QuestionOptionInsert(ctx, store.QuestionOptionInsertParams{ID: ids[i], CourseID: courseID, QuestionID: id, Position: int16(i + 1), Body: o.Body, PinnedLast: o.PinnedLast}); err != nil { //nolint:gosec // ≤ 8
					return fmt.Errorf("exam: ghi đáp án: %w", err)
				}
			}
			key = AnswerKey(c, ids)
		} else if !same && row.Type == store.QuestionTypeTRUEFALSE {
			key = AnswerKey(c, nil)
		}
		upd, err := q.QuestionUpdateContent(ctx, store.QuestionUpdateContentParams{
			CourseID: courseID, ID: id, ExpectedVersion: row.Version, Title: c.Title, Topic: c.Topic, Difficulty: store.QuestionDifficulty(c.Difficulty), Stem: c.Stem,
			AnswerKey: key, Explanation: c.Explanation, ResetReview: contentChanged,
		})
		if err != nil {
			return fmt.Errorf("exam: sửa câu hỏi: %w", err)
		}
		out, err = s.detail(ctx, q, upd)
		return err
	})
	out.Warnings = warns
	return out, err
}

// Archive lưu trữ câu (không xoá cứng; câu đang dùng vẫn lưu trữ được, lịch sử giữ nguyên).
func (s *Service) Archive(ctx context.Context, actor, courseID, id uuid.UUID) (QuestionDetail, error) {
	var out QuestionDetail
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		before, err := q.QuestionLock(ctx, store.QuestionLockParams{CourseID: courseID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: khoá câu hỏi: %w", err)
		}
		row, err := q.QuestionArchive(ctx, store.QuestionArchiveParams{CourseID: courseID, ID: id})
		if err != nil {
			return fmt.Errorf("exam: lưu trữ câu hỏi: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "question", id, "question.archive",
			map[string]any{"archived": before.ArchivedAt != nil, "version": before.Version}, map[string]any{"archived": true, "version": row.Version}); err != nil {
			return err
		}
		if before.ReviewStatus == store.QuestionReviewStatusPENDING {
			if _, err := outbox.Write(ctx, tx, TopicQuestionReviewed, map[string]any{"course_id": courseID}); err != nil {
				return fmt.Errorf("exam: outbox: %w", err)
			}
		}
		out, err = s.detail(ctx, q, row)
		return err
	})
	return out, err
}
