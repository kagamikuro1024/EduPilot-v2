package practice_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/practice"
	"github.com/edupilot/backend-go/internal/testutil"
)

// fakeStructured trả JSON dựng sẵn và ghi lại mọi prompt đã gửi (để kiểm không lộ đáp án).
type fakeStructured struct {
	mu      sync.Mutex
	prompts []string
	reply   func(system, user string) string
}

func (f *fakeStructured) Structured(_ context.Context, r llm.Request, _ json.RawMessage) (json.RawMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sys, usr := r.Messages[0].Content, r.Messages[len(r.Messages)-1].Content
	f.prompts = append(f.prompts, sys+"\n"+usr)
	return json.RawMessage(f.reply(sys, usr)), nil
}

// staff chuẩn bị Service có Jobs + Exam cho các đường của giảng viên.
func (r *rig) staff() { r.svc.Jobs, r.svc.Exam = jobs.NewService(r.pool), r.ex }

// doc tạo tài liệu READY của lớp + các đoạn (mỗi phần tử `pages` là một đoạn; số trang = vị trí + 1).
func (r *rig) doc(typ, title string, pages ...string) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into documents (course_id, title, type, status, visible_to_students) values ($1, $2, $3::document_type, 'READY', $4) returning id`,
		r.course, title, typ, typ != "ANSWER_KEY" && typ != "EXAM_PAPER").Scan(&id))
	for i, p := range pages {
		aud := "ALL"
		if typ == "ANSWER_KEY" {
			aud = "GRADING"
		}
		r.exec(`insert into content_chunks (document_id, course_ids, ord, page_no, text, audience) values ($1, array[$2::uuid], $3, $4, $5, $6::chunk_audience)`, id, r.course, i, i+1, p, aud)
	}
	return id
}

// runJob xếp việc rồi chạy ĐÚNG như worker (outbox `job.enqueue` → Runner); trả kết quả JSON của việc.
func (r *rig) runJob(w *practice.Worker, owner uuid.UUID, kind string, payload any) (status string, result json.RawMessage, errCode string) {
	r.t.Helper()
	ctx := r.t.Context()
	runner := jobs.NewRunner(r.pool, nil, clock.Real{}, slog.New(slog.DiscardHandler))
	w.Register(runner)
	j, err := jobs.NewService(r.pool).Enqueue(ctx, owner, kind, payload)
	require.NoError(r.t, err)
	var msg []byte
	require.NoError(r.t, r.pool.QueryRow(ctx, `select payload from outbox where topic = 'job.enqueue' and payload ->> 'job_id' = $1`, j.ID.String()).Scan(&msg))
	require.NoError(r.t, runner.HandleMessage(ctx, outboxMsg(msg)))
	var res []byte
	var ec *string
	require.NoError(r.t, r.pool.QueryRow(ctx, `select status::text, result, error ->> 'code' from jobs where id = $1`, j.ID).Scan(&status, &res, &ec))
	if ec != nil {
		errCode = *ec
	}
	return status, res, errCode
}

var questionLine = regexp.MustCompile(`(?m)^Câu (\d+)\. (.+)$`)

// paper dựng đề 24 câu (12 câu mỗi trang) + bảng đáp án "N. B" cho các câu trong `keyed`.
func paper(n int) []string {
	var pages []string
	for pg := 0; pg < 2; pg++ {
		var b strings.Builder
		for i := pg*n/2 + 1; i <= (pg+1)*n/2; i++ {
			fmt.Fprintf(&b, "Câu %d. Nội dung câu hỏi số %d là gì?\nA. Phương án một %d\nB. Phương án hai %d\nC. Phương án ba %d\nD. Phương án bốn %d\n\n", i, i, i, i, i, i)
		}
		pages = append(pages, b.String())
	}
	return pages
}

func paperReply(_, user string) string {
	type opt struct {
		Body string `json:"body"`
	}
	type q struct {
		Number       int     `json:"number"`
		Stem         string  `json:"stem"`
		Page         int     `json:"page"`
		CorrectLabel *string `json:"correct_label"`
		Options      []opt   `json:"options"`
	}
	var out []q
	page := 1
	for _, line := range strings.Split(user, "\n") {
		if m := regexp.MustCompile(`^\[Trang (\d+)\]`).FindStringSubmatch(line); m != nil {
			_, _ = fmt.Sscan(m[1], &page)
		}
		if m := questionLine.FindStringSubmatch(line); m != nil {
			var n int
			_, _ = fmt.Sscan(m[1], &n)
			out = append(out, q{Number: n, Stem: strings.TrimSpace(m[2]), Page: page, Options: []opt{{"Phương án một"}, {"Phương án hai"}, {"Phương án ba"}, {"Phương án bốn"}}})
		}
	}
	b, _ := json.Marshal(map[string]any{"questions": out})
	return string(b)
}

// US-P9-02 AC2–AC5: trích 24 câu từ đề; đáp án khớp theo số câu từ bảng đáp án; câu không khớp / trùng → "bỏ qua" có lý do; đáp án KHÔNG vào prompt; chạy lại không tạo trùng.
func TestExtractQuestions(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.staff()
	ctx := t.Context()
	llmStub := &fakeStructured{reply: paperReply}
	w := &practice.Worker{Pool: r.pool, Exam: r.ex, LLM: llmStub}

	exist, err := r.ex.Create(ctx, r.teacher, r.course, exam.QuestionIn{Type: "TRUE_FALSE", Title: "đã có", Topic: "X", Stem: "Nội dung câu hỏi số 5 là gì?", Value: new(true)})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, exist.ID)

	paperID := r.doc("EXAM_PAPER", "Đề cũ", paper(24)...)
	var key strings.Builder
	key.WriteString("ZZKEYMARK Đáp án đề cũ\n")
	for i := 1; i <= 24; i++ {
		if i == 22 || i == 24 {
			continue // không có trong bảng đáp án
		}
		fmt.Fprintf(&key, "%d. %c\n", i, 'A'+i%4)
	}
	keyID := r.doc("ANSWER_KEY", "Đáp án", key.String())

	_, err = r.svc.EnqueueExtract(ctx, r.teacher, r.course, practice.ExtractIn{DocumentID: keyID})
	require.Equal(t, "WRONG_DOCUMENT_TYPE", code(t, err), "tài liệu không phải đề")
	_, err = r.svc.EnqueueExtract(ctx, r.teacher, r.course, practice.ExtractIn{DocumentID: uuid.New()})
	require.Equal(t, "NOT_FOUND", code(t, err))
	other := &rig{t: t, pool: r.pool}
	otherCourse := r.newCourse("OTHER-" + r.sfx)
	_ = other
	var foreign uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into documents (course_id, title, type, status) values ($1, 'Đề lớp khác', 'EXAM_PAPER', 'READY') returning id`, otherCourse).Scan(&foreign))
	_, err = r.svc.EnqueueExtract(ctx, r.teacher, r.course, practice.ExtractIn{DocumentID: foreign})
	require.Equal(t, "NOT_FOUND", code(t, err), "đề của lớp khác: không lộ tồn tại")

	jid, err := r.svc.EnqueueExtract(ctx, r.teacher, r.course, practice.ExtractIn{DocumentID: paperID, AnswerKeyDocumentID: &keyID})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, jid)
	_ = jid
	status, raw, ec := r.runJob(w, r.teacher, practice.KindExtract, practice.ExtractPayload{CourseID: r.course, ExtractIn: practice.ExtractIn{DocumentID: paperID, AnswerKeyDocumentID: &keyID}})
	require.Equal(t, "SUCCEEDED", status, ec)
	var res practice.ExtractResult
	require.NoError(t, json.Unmarshal(raw, &res))
	require.Len(t, res.Created, 21, "24 − 2 không khớp đáp án − 1 trùng")
	reasons := map[string][]int{}
	for _, s := range res.Skipped {
		reasons[s.Reason] = append(reasons[s.Reason], s.Number)
	}
	require.ElementsMatch(t, []int{22, 24}, reasons["NO_ANSWER_MATCH"])
	require.Equal(t, []int{5}, reasons["DUPLICATE"])

	var bad int
	require.NoError(t, r.pool.QueryRow(ctx, `select count(*) from question_bank q where course_id = $1 and origin = 'EXTRACTED' and (review_status <> 'PENDING' or answer_key is null or jsonb_array_length(citations) <> 1
		or (select count(*) from question_options o where o.question_id = q.id) < 2 or citations -> 0 ->> 'document_id' <> $2::text)`, r.course, paperID.String()).Scan(&bad))
	require.Zero(t, bad, "mọi câu: PENDING, có đáp án, ≥ 2 phương án, trích dẫn tới đề")
	// Câu 1: bảng đáp án ghi "1. B" ('A'+1%4) → phương án thứ hai.
	var correctPos int
	require.NoError(t, r.pool.QueryRow(ctx, `select o.position from question_bank q join question_options o on o.question_id = q.id
		where q.course_id = $1 and q.stem = 'Nội dung câu hỏi số 1 là gì?' and q.answer_key -> 'option_ids' ->> 0 = o.id::text`, r.course).Scan(&correctPos))
	require.Equal(t, 2, correctPos)
	for _, p := range llmStub.prompts {
		require.NotContains(t, p, "ZZKEYMARK", "nội dung ANSWER_KEY không vào prompt")
	}

	status, raw, _ = r.runJob(w, r.teacher, practice.KindExtract, practice.ExtractPayload{CourseID: r.course, ExtractIn: practice.ExtractIn{DocumentID: paperID, AnswerKeyDocumentID: &keyID}})
	require.Equal(t, "SUCCEEDED", status)
	require.NoError(t, json.Unmarshal(raw, &res))
	require.Empty(t, res.Created, "chạy lại: câu trùng bị bỏ qua")

	// Không có bảng đáp án và LLM không thấy đáp án → không câu nào có đáp án thì không câu nào được lưu.
	paper2 := r.doc("EXAM_PAPER", "Đề 2", "Câu 1. Câu mới hoàn toàn thứ nhất?\nA. a\nB. b\n")
	status, raw, _ = r.runJob(w, r.teacher, practice.KindExtract, practice.ExtractPayload{CourseID: r.course, ExtractIn: practice.ExtractIn{DocumentID: paper2}})
	require.Equal(t, "SUCCEEDED", status)
	require.NoError(t, json.Unmarshal(raw, &res))
	require.Empty(t, res.Created)
	require.Equal(t, "NO_ANSWER", res.Skipped[0].Reason)
}

// US-P9-03: câu sinh phải dẫn nguồn thật trong các đoạn được cấp; ANSWER_KEY không làm nguồn; bulk APPROVE chặn câu thiếu trích dẫn; gen-stats đếm nguyên / sửa / loại.
func TestGenerateCitations(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.staff()
	ctx := t.Context()
	good := func(stem, src string) map[string]any {
		return map[string]any{"type": "MCQ_SINGLE", "stem": stem, "value": nil, "explanation": "Theo đoạn nguồn.", "sources": []string{src},
			"options": []map[string]any{{"body": "Đúng", "correct": true}, {"body": "Sai một", "correct": false}, {"body": "Sai hai", "correct": false}, {"body": "Sai ba", "correct": false}}}
	}
	stub := &fakeStructured{reply: func(_, _ string) string {
		b, _ := json.Marshal(map[string]any{"questions": []any{good("Câu sinh có nguồn một?", "C1"), good("Câu sinh nguồn bịa?", "C9"), good("Câu sinh có nguồn hai?", "[C2]"), good("Câu sinh thứ tư?", "C1")}})
		return string(b)
	}}
	w := &practice.Worker{Pool: r.pool, Exam: r.ex, LLM: stub}
	lecture := r.doc("LECTURE", "Bài giảng", "Đoạn một về mạng máy tính.", "Đoạn hai về giao thức.")
	keyDoc := r.doc("ANSWER_KEY", "Đáp án", "1. A")

	in := practice.GenerateIn{DocumentIDs: []uuid.UUID{lecture}, Topic: "Mạng", Difficulty: "MEDIUM", Count: 3}
	_, err := r.svc.EnqueueGenerate(ctx, r.teacher, r.course, practice.GenerateIn{DocumentIDs: []uuid.UUID{keyDoc}, Topic: "Mạng", Difficulty: "EASY", Count: 2})
	require.Equal(t, "WRONG_DOCUMENT_TYPE", code(t, err), "ANSWER_KEY bị từ chối làm nguồn")
	_, err = r.svc.EnqueueGenerate(ctx, r.teacher, r.course, practice.GenerateIn{DocumentIDs: []uuid.UUID{lecture}, Topic: "Mạng", Difficulty: "EASY", Count: 11})
	require.Equal(t, "VALIDATION_FAILED", code(t, err), "count ≤ 10")
	_, err = r.svc.EnqueueGenerate(ctx, r.teacher, r.course, practice.GenerateIn{Topic: "Mạng", Difficulty: "EASY", Count: 2})
	require.Equal(t, "VALIDATION_FAILED", code(t, err), "thiếu nguồn")
	_, err = r.svc.EnqueueGenerate(ctx, r.teacher, r.course, in)
	require.NoError(t, err)

	status, raw, ec := r.runJob(w, r.teacher, practice.KindGenerate, practice.GeneratePayload{CourseID: r.course, GenerateIn: in})
	require.Equal(t, "SUCCEEDED", status, ec)
	var res practice.ExtractResult
	require.NoError(t, json.Unmarshal(raw, &res))
	require.Len(t, res.Created, 2, "%v: câu nguồn bịa không được lưu; câu thứ 4 vượt count", res.Skipped)
	got := map[string]bool{}
	for _, s := range res.Skipped {
		got[s.Reason] = true
	}
	require.True(t, got["CITATION_REQUIRED"] && got["EXTRA"], "%v", res.Skipped)
	var withCites int
	require.NoError(t, r.pool.QueryRow(ctx, `select count(*) from question_bank where course_id = $1 and origin = 'GENERATED' and review_status = 'PENDING' and citations -> 0 ->> 'document_id' = $2::text and citations -> 0 ->> 'chunk_id' is not null`, r.course, lecture.String()).Scan(&withCites))
	require.Equal(t, 2, withCites)

	// Bulk: APPROVE một câu (nguyên), sửa nhẹ rồi APPROVE câu kia qua SET_TAGS → APPROVE.
	a, b := res.Created[0], res.Created[1]
	out, err := r.svc.Bulk(ctx, r.teacher, r.course, practice.BulkIn{Action: "SET_TAGS", IDs: []uuid.UUID{b, uuid.New()}, Topic: new("Giao thức")})
	require.NoError(t, err)
	require.True(t, out[0].OK)
	require.Equal(t, "NOT_FOUND", out[1].Code, "một id lỗi không huỷ cả lô")
	out, err = r.svc.Bulk(ctx, r.teacher, r.course, practice.BulkIn{Action: "APPROVE", IDs: []uuid.UUID{a, b}})
	require.NoError(t, err)
	require.True(t, out[0].OK && out[1].OK, "%+v", out)
	// Câu GENERATED bị xoá trích dẫn → APPROVE bị chặn.
	r.exec(`update question_bank set citations = '[]' where id = $1`, res.Created[0])
	extra := r.approvedPending(t, "GENERATED")
	_, err = r.ex.Review(ctx, r.teacher, r.course, extra, exam.DecisionApprove, 1)
	require.Equal(t, "VALIDATION_FAILED", code(t, err), "duyệt tay câu sinh không có trích dẫn cũng bị chặn (details: CITATION_REQUIRED)")
	out, err = r.svc.Bulk(ctx, r.teacher, r.course, practice.BulkIn{Action: "APPROVE", IDs: []uuid.UUID{extra}})
	require.NoError(t, err)
	require.Equal(t, "CITATION_REQUIRED", out[0].Code)
	out, err = r.svc.Bulk(ctx, r.teacher, r.course, practice.BulkIn{Action: "REJECT", IDs: []uuid.UUID{extra}})
	require.NoError(t, err)
	require.True(t, out[0].OK)
	out, err = r.svc.Bulk(ctx, r.teacher, r.course, practice.BulkIn{Action: "SET_TAGS", IDs: []uuid.UUID{a}, Topic: new("X")})
	require.NoError(t, err)
	require.Equal(t, "CONFLICT", out[0].Code, "đã duyệt: không gắn nhãn hàng loạt")
	_, err = r.svc.Bulk(ctx, r.teacher, r.course, practice.BulkIn{Action: "APPROVE", IDs: make([]uuid.UUID, 51)})
	require.Equal(t, "VALIDATION_FAILED", code(t, err))
	_, err = r.svc.Bulk(ctx, r.teacher, r.course, practice.BulkIn{Action: "DELETE", IDs: []uuid.UUID{a}})
	require.Equal(t, "VALIDATION_FAILED", code(t, err))

	var jobID uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `select ai_job_id from question_bank where id = $1`, a).Scan(&jobID))
	st, err := r.svc.GenStatsOf(ctx, r.course, jobID)
	require.NoError(t, err)
	require.Equal(t, practice.GenStats{Unchanged: 1, Edited: 1}, st, "a: nguyên; b: có sửa nhãn trước khi duyệt")
	_, err = r.svc.GenStatsOf(ctx, r.course, uuid.New())
	require.Equal(t, "NOT_FOUND", code(t, err))
}

// approvedPending tạo một câu PENDING không có trích dẫn với origin cho trước.
func (r *rig) approvedPending(t *testing.T, origin string) uuid.UUID {
	t.Helper()
	d, err := r.ex.Create(t.Context(), r.teacher, r.course, exam.QuestionIn{Type: "TRUE_FALSE", Title: "chờ", Topic: "Chờ", Stem: "Nhận định chờ duyệt " + uuid.NewString()[:6], Value: new(true)})
	require.NoError(t, err)
	r.exec(`update question_bank set origin = $2::question_origin, review_status = 'PENDING' where id = $1`, d.ID, origin)
	return d.ID
}

// P-10-8 F3 / Q-P9-12: ≤ 5 việc extract và ≤ 5 việc generate mỗi lớp mỗi giờ; việc thứ 6 → 429 RATE_LIMITED + Retry-After (không xếp việc); lớp khác không bị ảnh hưởng.
func TestJobLimitsPerCourse(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.staff()
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	r.svc.Redis = rdb
	ctx := t.Context()
	paperID := r.doc("EXAM_PAPER", "Đề", "Câu 1. a?\nA. x\nB. y\n")
	lecture := r.doc("LECTURE", "Bài giảng", "Nội dung bài giảng.")
	for i := range practice.MaxJobsPerHr {
		_, err := r.svc.EnqueueExtract(ctx, r.teacher, r.course, practice.ExtractIn{DocumentID: paperID})
		require.NoError(t, err, "việc %d", i+1)
		_, err = r.svc.EnqueueGenerate(ctx, r.teacher, r.course, practice.GenerateIn{DocumentIDs: []uuid.UUID{lecture}, Topic: "Mạng", Difficulty: "EASY", Count: 2})
		require.NoError(t, err, "việc %d", i+1)
	}
	var ae *apierr.Error
	_, err = r.svc.EnqueueExtract(ctx, r.teacher, r.course, practice.ExtractIn{DocumentID: paperID})
	require.ErrorAs(t, err, &ae)
	require.Equal(t, 429, ae.Status)
	require.Equal(t, "RATE_LIMITED", ae.Code)
	require.Positive(t, ae.RetryAfter)
	require.LessOrEqual(t, ae.RetryAfter, 3600)
	_, err = r.svc.EnqueueGenerate(ctx, r.teacher, r.course, practice.GenerateIn{DocumentIDs: []uuid.UUID{lecture}, Topic: "Mạng", Difficulty: "EASY", Count: 2})
	require.Equal(t, 429, status(t, err))
	var n int
	require.NoError(t, r.pool.QueryRow(ctx, `select count(*) from jobs where kind in ('question.extract','question.generate')`).Scan(&n))
	require.Equal(t, 2*practice.MaxJobsPerHr, n, "việc thứ 6 không được xếp")
}

// P-10-8 F3: đề > 60 trang chỉ được trích 60 trang đầu; phần cắt được báo bằng PAGE_LIMIT.
func TestExtractPageCap(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.staff()
	pages := make([]string, 62)
	for i := range pages {
		pages[i] = fmt.Sprintf("Câu %d. Nội dung câu số %d?\nA. một\nB. hai\n", i+1, i+1)
	}
	paperID := r.doc("EXAM_PAPER", "Đề dài", pages...)
	keyID := r.doc("ANSWER_KEY", "Đáp án", func() string {
		var b strings.Builder
		for i := 1; i <= 62; i++ {
			fmt.Fprintf(&b, "%d. A\n", i)
		}
		return b.String()
	}())
	w := &practice.Worker{Pool: r.pool, Exam: r.ex, LLM: &fakeStructured{reply: paperReply}}
	status, raw, ec := r.runJob(w, r.teacher, practice.KindExtract, practice.ExtractPayload{CourseID: r.course, ExtractIn: practice.ExtractIn{DocumentID: paperID, AnswerKeyDocumentID: &keyID}})
	require.Equal(t, "SUCCEEDED", status, ec)
	var res practice.ExtractResult
	require.NoError(t, json.Unmarshal(raw, &res))
	require.Len(t, res.Created, practice.MaxExtractPages)
	require.Equal(t, []int{practice.MaxExtractPages + 1}, func() (p []int) {
		for _, s := range res.Skipped {
			if s.Reason == "PAGE_LIMIT" {
				p = append(p, s.Page)
			}
		}
		return
	}())
}
