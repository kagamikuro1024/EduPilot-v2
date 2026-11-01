package grading_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/llm"
)

var wordRe = regexp.MustCompile(`[\p{L}\p{N}_]+`)

// firstWords: n từ đầu của đáp án chuẩn (nguyên văn, kể cả dấu câu giữa các từ).
func firstWords(ref string, n int) string {
	loc := wordRe.FindAllStringIndex(ref, n)
	return ref[loc[0][0]:loc[len(loc)-1][1]]
}

// withComment trả bản ghi của mẫu S1 với nhận xét của c1 thay bằng `c` ở CẢ HAI lượt.
func withComment(t *testing.T, raw json.RawMessage, c string) json.RawMessage {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	m["criteria"].([]any)[0].(map[string]any)["comment"] = c //nolint:forcetypeassert // cấu trúc của fixture
	out, err := json.Marshal(m)
	require.NoError(t, err)
	return out
}

// TestAnswerEchoInRun — QC BUG-P7-06-1: `Grader.Run` gắn cờ ANSWER_ECHO (và needs_review) khi nhận xét chép đúng 12 từ liên tiếp của đáp án chuẩn; 11 từ thì không.
func TestAnswerEchoInRun(t *testing.T) {
	ss := samples(t)
	body := string(mustRead(t, asgFix+ss[0].File))
	for _, c := range []struct {
		words int
		want  bool
	}{{12, true}, {11, false}} {
		var g *gradeEnv
		g = newGradeEnv(t, func(_ *gradeEnv, r llm.Request) (json.RawMessage, error) {
			raw, err := byPrompt(t, ss[:1], map[string]string{"S1": body})(g, r)
			return withComment(t, raw, firstWords(g.ref, c.words)), err
		})
		res := g.submit("A", "bai.md", body, nil)
		_, err := g.grader.Run(t.Context(), g.course, res)
		require.NoError(t, err)
		_, _, flags, _, _, _, _ := g.row(res)
		if c.want {
			require.Contains(t, flags, "ANSWER_ECHO", "%d từ", c.words)
			var nr bool
			g.one(&nr, `select needs_review from grading_results where id = $1`, res)
			require.True(t, nr)
		} else {
			require.NotContains(t, flags, "ANSWER_ECHO", "%d từ", c.words)
		}
	}
}

// docxBytes dựng một DOCX tối thiểu (zip có [Content_Types].xml và word/document.xml).
func docxBytes(t *testing.T, text string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for name, body := range map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": "<w:document><w:t>" + text + "</w:t></w:document>"} {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return b.Bytes()
}

// TestExtractDocx — DOCX đi qua docling (giả) như PDF: một lần gọi, văn bản trích lưu vào bài nộp, chấm tiếp được.
func TestExtractDocx(t *testing.T) {
	ss := samples(t)
	b := mustRead(t, asgFix+ss[0].File)
	g := newGradeEnv(t, byPrompt(t, ss[:1], map[string]string{"S1": string(b)}))
	g.ext.text = string(b)
	res := g.submit("A", "bai.docx", string(docxBytes(t, "x")), nil)
	out, err := g.grader.Run(t.Context(), g.course, res)
	require.NoError(t, err)
	require.Equal(t, 1, g.ext.n, "DOCX qua docling")
	require.Empty(t, out.Code)
	var st, text string
	g.one(&st, `select extract_status from submissions where id = (select submission_id from grading_results where id = $1)`, res)
	g.one(&text, `select extracted_text from submissions where id = (select submission_id from grading_results where id = $1)`, res)
	require.Equal(t, "OK", st)
	require.Equal(t, string(b), text)
}

// TestExtractFailedNotZero — trích hỏng → FAILED, mọi điểm NULL (không ghi 0), không điểm nháp, giảng viên chấm tay.
func TestExtractFailedNotZero(t *testing.T) {
	g := newGradeEnv(t, func(*gradeEnv, llm.Request) (json.RawMessage, error) { t.Fatal("không gọi LLM"); return nil, nil })
	g.ext.err = &ingest.ExtractFailure{Message: "hỏng"}
	res := g.submit("A", "bai.docx", "x", nil)
	_, err := g.grader.Run(t.Context(), g.course, res)
	require.NoError(t, err)
	state, status, _, total, before, code, crit := g.row(res)
	require.Equal(t, "FAILED", state)
	require.Equal(t, "DRAFT", status)
	require.Nil(t, total)
	require.Nil(t, before)
	require.Equal(t, grading.ErrExtractFailed, *code)
	var cs []struct {
		Score *string `json:"score"`
	}
	require.NoError(t, json.Unmarshal(crit, &cs))
	for _, c := range cs {
		require.Nil(t, c.Score, "tiêu chí không có điểm 0 giả")
	}
}

// TestGradingInvalidOutputFailed — đầu ra LLM không qua kiểm (thiếu tiêu chí) → FAILED, không điểm; điểm là chuỗi, đầu ra có số JSON bị từ chối.
func TestGradingInvalidOutputFailed(t *testing.T) {
	for name, raw := range map[string]string{
		"thiếu tiêu chí":  `{"criteria":[],"overall_feedback":"x","confidence":0.9,"flags":[],"total":"0"}`,
		"không phải JSON": `đây không phải JSON`,
	} {
		g := newGradeEnv(t, func(*gradeEnv, llm.Request) (json.RawMessage, error) { return json.RawMessage(raw), nil })
		res := g.submit("A", "bai.md", strings.Repeat("Nội dung bài làm đủ dài để chấm. ", 5), nil)
		out, err := g.grader.Run(t.Context(), g.course, res)
		require.NoError(t, err, name)
		require.Equal(t, grading.ErrOutputInvalid, out.Code, name)
		state, _, _, total, _, _, _ := g.row(res)
		require.Equal(t, "FAILED", state, name)
		require.Nil(t, total, name)
	}
}

// TestInjectionFlagAndHighlight — câu giả mệnh lệnh trong bài chỉ gắn cờ + trả vị trí để tô; không đổi điểm, câu vẫn nằm nguyên trong prompt.
func TestInjectionFlagAndHighlight(t *testing.T) {
	ss := samples(t)
	var s6 sample
	for _, s := range ss {
		if s.ID == "S6" {
			s6 = s
		}
	}
	b := string(mustRead(t, asgFix+s6.File))
	g := newGradeEnv(t, byPrompt(t, []sample{s6}, map[string]string{"S6": b}))
	res := g.submit("A", "bai.md", b, nil)
	_, err := g.grader.Run(t.Context(), g.course, res)
	require.NoError(t, err)
	_, _, flags, total, _, _, _ := g.row(res)
	require.Contains(t, flags, "INJECTION_SUSPECTED")
	require.NotNil(t, total)
	spans := grading.ScanInjection(b)
	require.NotEmpty(t, spans)
	for _, sp := range spans {
		require.Contains(t, g.prompts[0], strings.TrimSpace(sp.Sentence), "câu nghi ngờ vẫn nguyên trong prompt")
	}
}

// TestAuditHasNoSubmissionText — nhật ký và bảng ghi chung (audit_log) không chứa nội dung bài nộp hay tên sinh viên.
func TestAuditHasNoSubmissionText(t *testing.T) {
	ss := samples(t)
	b := string(mustRead(t, asgFix+ss[0].File))
	g := newGradeEnv(t, byPrompt(t, ss[:1], map[string]string{"S1": b}))
	var buf bytes.Buffer
	g.grader.Log = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	res := g.submit("Nguyễn Văn Ẩn Danh", "bai.md", b, nil)
	_, err := g.grader.Run(t.Context(), g.course, res)
	require.NoError(t, err)
	var n int
	g.one(&n, `select count(*) from audit_log where after::text like '%SolarWinds%' or before::text like '%SolarWinds%' or after::text like '%Ẩn Danh%'`)
	require.Zero(t, n)
	require.NotContains(t, buf.String(), "SolarWinds")
	require.NotContains(t, buf.String(), "Ẩn Danh")
}
