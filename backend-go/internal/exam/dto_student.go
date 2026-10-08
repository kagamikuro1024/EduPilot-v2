package exam

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/store"
)

// DTO CỦA SINH VIÊN (SRS 6.4): struct riêng, KHÔNG tái dùng struct của Staff và KHÔNG dùng `omitempty` để che trường nhạy cảm — trường nhạy cảm không tồn tại ở đây.
// Thêm trường mới bắt buộc sửa `testdata/student_dto_allowlist.json` (TestStudentDTOAllowlist) để có người rà.

// MyAttemptView là lượt làm của CHÍNH sinh viên trong danh sách / chi tiết bài.
type MyAttemptView struct {
	ID          uuid.UUID  `json:"id"`
	Status      string     `json:"status"`
	DeadlineAt  time.Time  `json:"deadline_at"`
	SubmittedAt *time.Time `json:"submitted_at"`
}

// ExamStudentView là bài thi nhìn từ sinh viên. Không có `items`, `override`, số câu theo loại; `my_score` chỉ khi PUBLISHED.
type ExamStudentView struct {
	ID              uuid.UUID      `json:"id"`
	Title           string         `json:"title"`
	Instructions    *string        `json:"instructions"`
	Kind            string         `json:"kind"`
	OpensAt         *time.Time     `json:"opens_at"`
	ClosesAt        *time.Time     `json:"closes_at"`
	DurationMinutes *int           `json:"duration_minutes"`
	MaxScore        string         `json:"max_score"`
	Status          string         `json:"status"`
	MyAttempt       *MyAttemptView `json:"my_attempt"`
	MyScore         *string        `json:"my_score"`
}

// OptionView là một đáp án như sinh viên thấy (không có `pinned_last`, không có đúng / sai).
type OptionView struct {
	ID   uuid.UUID `json:"id"`
	Body string    `json:"body"`
}

// SampleView là một test MẪU (đầu vào và đầu ra mong đợi của test mẫu được phép hiển thị).
type SampleView struct {
	Name     string `json:"name"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

// CodeItemView là phần bài code. Không có test ẩn, trọng số, `tests_version`, lời giải mẫu.
type CodeItemView struct {
	Languages     []string          `json:"languages"`
	TimeLimitMS   int               `json:"time_limit_ms"`
	MemoryLimitMB int               `json:"memory_limit_mb"`
	StarterCode   map[string]string `json:"starter_code"`
	Samples       []SampleView      `json:"samples"`
}

// ItemView là một mục của bài như sinh viên thấy. Không có: answer_key, correct, is_correct, explanation, override, original_position, pinned_last.
type ItemView struct {
	ItemID   uuid.UUID     `json:"item_id"`
	Position int           `json:"position"`
	Type     string        `json:"type"`
	Points   string        `json:"points"`
	Stem     string        `json:"stem"`
	Options  []OptionView  `json:"options"`
	Code     *CodeItemView `json:"code"`
	// Answer là phần CHÍNH sinh viên này đã lưu (`{"option_ids":[…]}` / `{"value":true}`); null khi chưa trả lời hoặc ở xem trước.
	Answer json.RawMessage `json:"answer"`
}

// PreviewView là phản hồi của "xem trước": cùng DTO của lượt làm sinh viên, `preview:true`, không lượt làm, không đáp án đã chọn.
type PreviewView struct {
	Preview bool            `json:"preview"`
	Exam    ExamStudentView `json:"exam"`
	Items   []ItemView      `json:"items"`
}

// studentView dựng DTO sinh viên từ một hàng ExamList (đã lọc theo sinh viên).
func studentView(r store.ExamListRow) ExamStudentView {
	v := ExamStudentView{ID: r.ID, Title: r.Title, Instructions: r.Instructions, Kind: string(r.Kind), OpensAt: r.OpensAt, ClosesAt: r.ClosesAt,
		DurationMinutes: durationOf(r.DurationMinutes), MaxScore: r.MaxScore.StringFixed(2), Status: string(r.EffectiveStatus)}
	if r.MyAttemptID != nil && r.MyAttemptStatus != nil && r.MyDeadlineAt != nil {
		v.MyAttempt = &MyAttemptView{ID: *r.MyAttemptID, Status: string(*r.MyAttemptStatus), DeadlineAt: *r.MyDeadlineAt, SubmittedAt: r.MySubmittedAt}
	}
	if r.Status == store.ExamStatusPUBLISHED { // điểm chỉ hiện sau khi công bố
		switch {
		case r.MyAdjustedScore.Valid:
			sc := r.MyAdjustedScore.Decimal.StringFixed(2)
			v.MyScore = &sc
		case r.MyAutoScore.Valid:
			sc := r.MyAutoScore.Decimal.StringFixed(2)
			v.MyScore = &sc
		}
	}
	return v
}

// StudentListRow là một hàng danh sách của sinh viên; `CreatedAt` chỉ để dựng con trỏ, không ra JSON.
type StudentListRow struct {
	ExamStudentView
	CreatedAt time.Time `json:"-"`
}

// ListExamsForStudent: danh sách của sinh viên — chỉ SCHEDULED / OPEN / CLOSED / PUBLISHED (effective), trường giới hạn.
func (s *Service) ListExamsForStudent(ctx context.Context, courseID, studentID uuid.UUID, status string, cur *Cursor, fetch int) ([]StudentListRow, error) {
	rows, err := s.examListRows(ctx, courseID, ExamListFilter{Status: status, StudentID: &studentID}, cur, fetch)
	if err != nil {
		return nil, err
	}
	out := make([]StudentListRow, len(rows))
	for i, r := range rows {
		out[i] = StudentListRow{ExamStudentView: studentView(r), CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// GetExamForStudent: chi tiết của sinh viên; DRAFT hoặc bài lớp khác → 404 (không lộ tồn tại).
func (s *Service) GetExamForStudent(ctx context.Context, courseID, studentID, examID uuid.UUID) (ExamStudentView, error) {
	rows, err := s.examListRows(ctx, courseID, ExamListFilter{StudentID: &studentID, ExamID: &examID}, nil, 1)
	if err != nil {
		return ExamStudentView{}, err
	}
	if len(rows) == 0 {
		return ExamStudentView{}, notFound()
	}
	return studentView(rows[0]), nil
}

// ---- Xem trước (SRS 4.2.3) -----------------------------------------------------------------------------------------------

// viewSource là dữ liệu thô để dựng bài như sinh viên thấy; dùng chung cho xem trước và (về sau) lượt làm thật.
type viewSource struct {
	Items   []store.ExamPreviewItemsRow
	Options map[uuid.UUID][]store.ExamPreviewOptionsRow
	Samples map[uuid.UUID][]SampleView
}

// Seed là cặp hạt giống của bộ xáo trộn (SRS 4.3.2): `h1`, `h2` là hai nửa đầu của SHA-256(attempt_id), đọc big-endian.
type Seed struct{ H1, H2 uint64 }

// AttemptSeed là hạt giống tất định của một lượt làm: cùng lượt luôn cùng thứ tự (tải lại, đổi thiết bị), không lưu thứ tự.
func AttemptSeed(attemptID uuid.UUID) Seed {
	sum := sha256.Sum256(attemptID[:])
	return Seed{H1: binary.BigEndian.Uint64(sum[0:8]), H2: binary.BigEndian.Uint64(sum[8:16])}
}

// permute xáo `n` phần tử bằng Fisher–Yates ĐÚNG như đặc tả (`for i := n-1; i > 0; i-- { j := r.IntN(i+1); swap }`), không dùng rand.Shuffle: thứ tự gọi bộ sinh là
// một phần của hợp đồng (luồng PCG được đặc tả ổn định) nên đổi thư viện cũng không đổi đề của sinh viên đã làm dở.
func permute[T any](r *rand.Rand, xs []T) {
	for i := len(xs) - 1; i > 0; i-- {
		j := r.IntN(i + 1)
		xs[i], xs[j] = xs[j], xs[i]
	}
}

const goldenGamma = 0x9E3779B97F4A7C15

// BuildStudentView dựng danh sách mục theo DTO sinh viên: thứ tự câu (nếu `shuffleQ`) và đáp án (nếu `shuffleO`; đáp án ghim đứng cuối, giữ thứ tự gốc) là hàm
// THUẦN của (nguồn, hạt giống). Thứ tự đáp án của câu thứ k (theo vị trí gốc, từ 0) dùng luồng PCG(h1 ^ (k+1)·γ, h2) riêng nên không phụ thuộc thứ tự câu. `saved` là
// câu trả lời đã lưu của CHÍNH sinh viên theo item_id (nil ở xem trước). Nhãn A, B, C… gán ở frontend theo thứ tự hiển thị.
func BuildStudentView(src viewSource, shuffleQ, shuffleO bool, seed Seed, saved map[uuid.UUID]json.RawMessage) ([]ItemView, error) {
	order := slices.Clone(src.Items)
	origin := make(map[uuid.UUID]int, len(order))
	for k, it := range order {
		origin[it.ItemID] = k
	}
	if shuffleQ {
		permute(rand.New(rand.NewPCG(seed.H1, seed.H2)), order)
	}
	out := make([]ItemView, len(order))
	for i, it := range order {
		v := ItemView{ItemID: it.ItemID, Position: i + 1, Type: string(it.Type), Points: it.Points.StringFixed(2), Stem: it.Stem, Options: []OptionView{}, Answer: saved[it.ItemID]}
		if it.Type == store.QuestionTypeCODE {
			c := CodeItemView{Languages: it.Languages, StarterCode: map[string]string{}, Samples: src.Samples[it.QuestionID]}
			if it.TimeLimitMs != nil {
				c.TimeLimitMS = int(*it.TimeLimitMs)
			}
			if it.MemoryLimitMb != nil {
				c.MemoryLimitMB = int(*it.MemoryLimitMb)
			}
			if len(it.StarterCode) > 0 {
				if err := json.Unmarshal(it.StarterCode, &c.StarterCode); err != nil {
					return nil, fmt.Errorf("exam: mã khởi tạo: %w", err)
				}
			}
			if c.Samples == nil {
				c.Samples = []SampleView{}
			}
			v.Code = &c
		} else {
			opts := slices.Clone(src.Options[it.QuestionID])
			if shuffleO {
				var free, pinned []store.ExamPreviewOptionsRow
				for _, o := range opts {
					if o.PinnedLast {
						pinned = append(pinned, o)
					} else {
						free = append(free, o)
					}
				}
				k := uint64(origin[it.ItemID]) //nolint:gosec // chỉ số ≥ 0
				permute(rand.New(rand.NewPCG(seed.H1^((k+1)*goldenGamma), seed.H2)), free)
				opts = append(free, pinned...)
			}
			for _, o := range opts {
				v.Options = append(v.Options, OptionView{ID: o.ID, Body: o.Body})
			}
		}
		out[i] = v
	}
	return out, nil
}

func (s *Service) loadViewSource(ctx context.Context, q *store.Queries, courseID, examID uuid.UUID) (viewSource, error) {
	items, err := q.ExamPreviewItems(ctx, store.ExamPreviewItemsParams{CourseID: courseID, ExamID: examID})
	if err != nil {
		return viewSource{}, fmt.Errorf("exam: mục xem trước: %w", err)
	}
	opts, err := q.ExamPreviewOptions(ctx, store.ExamPreviewOptionsParams{CourseID: courseID, ExamID: examID})
	if err != nil {
		return viewSource{}, fmt.Errorf("exam: đáp án xem trước: %w", err)
	}
	samples, err := q.ExamPreviewSamples(ctx, store.ExamPreviewSamplesParams{CourseID: courseID, ExamID: examID})
	if err != nil {
		return viewSource{}, fmt.Errorf("exam: test mẫu xem trước: %w", err)
	}
	src := viewSource{Items: items, Options: map[uuid.UUID][]store.ExamPreviewOptionsRow{}, Samples: map[uuid.UUID][]SampleView{}}
	for _, o := range opts {
		src.Options[o.QuestionID] = append(src.Options[o.QuestionID], o)
	}
	for _, t := range samples {
		in, exp := "", ""
		if t.Input != nil {
			in = *t.Input
		}
		if t.Expected != nil {
			exp = *t.Expected
		}
		src.Samples[t.ProblemID] = append(src.Samples[t.ProblemID], SampleView{Name: t.Name, Input: in, Expected: exp})
	}
	return src, nil
}

// PreviewExam dựng bài như sinh viên sẽ thấy, với hạt giống ngẫu nhiên của lần xem. KHÔNG ghi DB, không tạo lượt làm, không đặt khoá, không ghi sự kiện.
func (s *Service) PreviewExam(ctx context.Context, courseID, examID uuid.UUID) (PreviewView, error) {
	q := store.New(s.Pool)
	e, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: examID})
	if errors.Is(err, pgx.ErrNoRows) {
		return PreviewView{}, notFound()
	}
	if err != nil {
		return PreviewView{}, fmt.Errorf("exam: đọc bài thi: %w", err)
	}
	src, err := s.loadViewSource(ctx, q, courseID, examID)
	if err != nil {
		return PreviewView{}, err
	}
	items, err := BuildStudentView(src, e.ShuffleQuestions, e.ShuffleOptions, Seed{H1: rand.Uint64(), H2: rand.Uint64()}, nil) //nolint:gosec // hạt giống xem trước: ngẫu nhiên mỗi lần xem
	if err != nil {
		return PreviewView{}, err
	}
	return PreviewView{Preview: true, Items: items, Exam: ExamStudentView{ID: e.ID, Title: e.Title, Instructions: e.Instructions, Kind: string(e.Kind), OpensAt: e.OpensAt, ClosesAt: e.ClosesAt,
		DurationMinutes: durationOf(e.DurationMinutes), MaxScore: e.MaxScore.StringFixed(2), Status: EffectiveStatus(string(e.Status), e.OpensAt, e.ClosesAt, s.now())}}, nil
}
