package practice_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/practice"
)

// demoStructured: provider giả với bản ghi DEMO_MODE thật (seed/demo/fake_llm.json) và kiểm schema như `llm.Gateway.Structured`.
type demoStructured struct{ p *fake.Provider }

func newDemo() *demoStructured {
	return &demoStructured{fake.NewNamed(fake.NewController(fake.Settings{DemoFile: "../../../seed/demo/fake_llm.json"}), "demo", "")}
}

func (d *demoStructured) Structured(ctx context.Context, r llm.Request, schema json.RawMessage) (json.RawMessage, error) {
	msgs := make([]provider.Message, len(r.Messages))
	for i, m := range r.Messages {
		msgs[i] = provider.Message{Role: m.Role, Content: m.Content}
	}
	res, err := d.p.Chat(ctx, provider.ChatOpts{Messages: msgs, Schema: schema})
	if err != nil {
		return nil, err
	}
	if llm.ValidateJSON(schema, json.RawMessage(res.Text)) != nil {
		return nil, &llm.ErrUnavailable{Reason: llm.ReasonAllFailed}
	}
	return json.RawMessage(res.Text), nil
}

type deMau struct {
	Title     string `json:"title"`
	PerPage   int    `json:"per_page"`
	Questions []struct {
		Number  int      `json:"number"`
		Page    int      `json:"page"`
		Stem    string   `json:"stem"`
		Options []string `json:"options"`
		Correct int      `json:"correct"`
	} `json:"questions"`
}

func loadDeMau(t *testing.T) deMau {
	t.Helper()
	raw, err := os.ReadFile("../../../seed/practice/de-mau.json")
	require.NoError(t, err)
	var d deMau
	require.NoError(t, json.Unmarshal(raw, &d))
	require.Len(t, d.Questions, 24)
	return d
}

// pages dựng văn bản từng trang như PDF `de-mau-trac-nghiem.pdf` (tiêu đề ở trang 1).
func (d deMau) pages(title string, n int) []string {
	var out []string
	var b strings.Builder
	for i, q := range d.Questions[:n] {
		if i%d.PerPage == 0 {
			if b.Len() > 0 {
				out = append(out, b.String())
				b.Reset()
			}
			if i == 0 {
				b.WriteString(title + "\n")
			}
		}
		fmt.Fprintf(&b, "Câu %d. %s\n", q.Number, q.Stem)
		for k, o := range q.Options {
			fmt.Fprintf(&b, "%c. %s\n", 'A'+k, o)
		}
	}
	return append(out, b.String())
}

func (d deMau) key(skip ...int) string {
	var b strings.Builder
	for _, q := range d.Questions {
		if len(skip) > 0 && (q.Number == skip[0] || q.Number == skip[1]) {
			continue
		}
		fmt.Fprintf(&b, "%d. %c\n", q.Number, 'A'+q.Correct)
	}
	return b.String()
}

func numbers(res practice.ExtractResult, reason string) []int {
	var n []int
	for _, s := range res.Skipped {
		if s.Reason == reason {
			n = append(n, s.Number)
		}
	}
	return n
}

// Bàn giao QC (P-10-7 Q4, Q6): bản ghi `practice-*` của seed/demo/fake_llm.json cho kết quả cố định đúng như handoff mô tả.
func TestDemoFixturesExtract(t *testing.T) {
	t.Parallel()
	d := loadDeMau(t)
	run := func(t *testing.T, paperTitle string, n int, key *string) (string, practice.ExtractResult, string, *rig) {
		r := newRig(t)
		r.staff()
		w := &practice.Worker{Pool: r.pool, Exam: r.ex, LLM: newDemo()}
		in := practice.ExtractIn{DocumentID: r.doc("EXAM_PAPER", "Đề mẫu", d.pages(paperTitle, n)...)}
		if key != nil {
			k := r.doc("ANSWER_KEY", "Đáp án", *key)
			in.AnswerKeyDocumentID = &k
		}
		status, raw, ec := r.runJob(w, r.teacher, practice.KindExtract, practice.ExtractPayload{CourseID: r.course, ExtractIn: in})
		var res practice.ExtractResult
		if status == "SUCCEEDED" {
			require.NoError(t, json.Unmarshal(raw, &res))
		}
		return status, res, ec, r
	}

	t.Run("đủ 24 câu, đáp án khớp bảng", func(t *testing.T) {
		t.Parallel()
		key := d.key()
		status, res, ec, r := run(t, d.Title, 24, &key)
		require.Equal(t, "SUCCEEDED", status, ec)
		require.Len(t, res.Created, 24)
		require.Empty(t, res.Skipped)
		for _, q := range d.Questions {
			var pos int
			require.NoError(t, r.pool.QueryRow(t.Context(), `select o.position from question_bank b join question_options o on o.question_id = b.id
				where b.course_id = $1 and b.stem = $2 and b.answer_key -> 'option_ids' ->> 0 = o.id::text`, r.course, q.Stem).Scan(&pos))
			require.Equal(t, q.Correct+1, pos, "câu %d", q.Number)
		}
	})
	t.Run("bảng đáp án thiếu câu 22 và 24", func(t *testing.T) {
		t.Parallel()
		key := d.key(22, 24)
		status, res, ec, _ := run(t, d.Title, 24, &key)
		require.Equal(t, "SUCCEEDED", status, ec)
		require.Len(t, res.Created, 22)
		require.ElementsMatch(t, []int{22, 24}, numbers(res, "NO_ANSWER_MATCH"))
	})
	t.Run("không có bảng đáp án: bỏ qua cả 24", func(t *testing.T) {
		t.Parallel()
		status, res, ec, _ := run(t, d.Title, 24, nil)
		require.Equal(t, "SUCCEEDED", status, ec)
		require.Empty(t, res.Created)
		require.Len(t, numbers(res, "NO_ANSWER"), 24)
	})
	t.Run("lỗi mức câu", func(t *testing.T) {
		t.Parallel()
		status, res, ec, _ := run(t, "ĐỀ MẪU LỖI CÂU — MẠNG MÁY TÍNH", 5, nil)
		require.Equal(t, "SUCCEEDED", status, ec)
		require.Len(t, res.Created, 1)
		require.ElementsMatch(t, []int{2, 3, 4}, numbers(res, "INVALID"))
		require.Equal(t, []int{5}, numbers(res, "NO_ANSWER"))
	})
	t.Run("sai schema: việc thất bại, không lưu câu nào", func(t *testing.T) {
		t.Parallel()
		status, _, ec, r := run(t, "ĐỀ MẪU LỖI SCHEMA — MẠNG MÁY TÍNH", 2, nil)
		require.Equal(t, "FAILED", status)
		require.Equal(t, "LLM_UNAVAILABLE", ec)
		var n int
		require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from question_bank where course_id = $1`, r.course).Scan(&n))
		require.Zero(t, n)
	})
}

func TestDemoFixturesGenerate(t *testing.T) {
	t.Parallel()
	gen := func(t *testing.T, topic string) (string, practice.ExtractResult, string) {
		r := newRig(t)
		r.staff()
		w := &practice.Worker{Pool: r.pool, Exam: r.ex, LLM: newDemo()}
		lecture := r.doc("LECTURE", "Bài giảng mạng", "Giao thức TCP thiết lập kết nối bằng bắt tay ba bước.")
		in := practice.GenerateIn{DocumentIDs: []uuid.UUID{lecture}, Topic: topic, Difficulty: "MEDIUM", Count: 5}
		status, raw, ec := r.runJob(w, r.teacher, practice.KindGenerate, practice.GeneratePayload{CourseID: r.course, GenerateIn: in})
		var res practice.ExtractResult
		if status == "SUCCEEDED" {
			require.NoError(t, json.Unmarshal(raw, &res))
		}
		return status, res, ec
	}
	t.Run("mẫu: 5 câu có nguồn", func(t *testing.T) {
		t.Parallel()
		status, res, ec := gen(t, "Mẫu sinh câu")
		require.Equal(t, "SUCCEEDED", status, ec)
		require.Len(t, res.Created, 5)
		require.Empty(t, res.Skipped)
	})
	t.Run("lỗi mức câu", func(t *testing.T) {
		t.Parallel()
		status, res, ec := gen(t, "Mẫu sinh lỗi")
		require.Equal(t, "SUCCEEDED", status, ec)
		require.Len(t, res.Created, 1)
		got := map[string]int{}
		for _, s := range res.Skipped {
			got[s.Reason] = s.Number
		}
		require.Equal(t, map[string]int{"CITATION_REQUIRED": 2, "TYPE_NOT_ALLOWED": 3, "INVALID": 4, "DUPLICATE": 5, "EXTRA": 6}, got)
	})
	t.Run("sai schema", func(t *testing.T) {
		t.Parallel()
		status, _, ec := gen(t, "Mẫu sinh sai schema")
		require.Equal(t, "FAILED", status)
		require.Equal(t, "LLM_UNAVAILABLE", ec)
	})
}

func TestDemoFixturesFeedback(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ stem, wantFirst string }{
		{"Trình bày cách một cuộc tấn công SQL injection diễn ra và hai biện pháp phòng chống.", "Cơ chế chèn mã"},
		{"So sánh chế độ ECB và CBC của mật mã khối.", "Chế độ ECB"},
	} {
		res, err := grading.Feedback(t.Context(), newDemo(), grading.FeedbackInput{Stem: c.stem, ModelAnswer: "x", Rubric: "y", Text: "Bài viết thử của sinh viên."})
		require.NoError(t, err)
		require.Len(t, res.Criteria, 3)
		require.Equal(t, c.wantFirst, res.Criteria[0].Name)
		require.False(t, res.Injected)
	}
}

func TestDemoFixturesValidSchemas(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("../../../seed/demo/fake_llm.json")
	require.NoError(t, err)
	var f struct {
		Recordings []struct {
			Name     string
			Response json.RawMessage
		}
	}
	require.NoError(t, json.Unmarshal(raw, &f))
	bad := map[string]bool{"practice-extract-loi-schema": true, "practice-generate-loi-schema": true}
	n := 0
	for _, r := range f.Recordings {
		var schema string
		switch {
		case strings.HasPrefix(r.Name, "practice-extract"):
			schema = practice.ExtractSchema
		case strings.HasPrefix(r.Name, "practice-generate"):
			schema = practice.GenerateSchema
		default:
			continue
		}
		n++
		err := llm.ValidateJSON(json.RawMessage(schema), r.Response)
		if bad[r.Name] {
			require.Error(t, err, r.Name)
		} else {
			require.NoError(t, err, r.Name)
		}
	}
	require.Equal(t, 6, n)
}

// P-10-8 Q29: bản ghi `practice-explain` áp cho luồng của provider giả (DEMO_MODE) và khớp lời nhắc giải thích thật.
func TestDemoFixturesExplainStream(t *testing.T) {
	t.Parallel()
	sys, err := os.ReadFile("prompts/explain.txt")
	require.NoError(t, err)
	p := fake.NewNamed(fake.NewController(fake.Settings{DemoFile: "../../../seed/demo/fake_llm.json"}), "demo", "")
	ch, err := p.Stream(t.Context(), provider.ChatOpts{Messages: []provider.Message{{Role: "system", Content: string(sys)}, {Role: "user", Content: "ĐỀ: x\nĐÁP ÁN ĐÚNG: B."}}})
	require.NoError(t, err)
	var sb strings.Builder
	for c := range ch {
		require.NoError(t, c.Err)
		sb.WriteString(c.Text)
	}
	require.Contains(t, sb.String(), "Phương án đúng khớp với nội dung đã học")
	require.NotContains(t, sb.String(), "giả lập")
}
