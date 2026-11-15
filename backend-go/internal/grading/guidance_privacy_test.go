package grading_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/store"
)

type fixedRoster []privacy.Member

func (f fixedRoster) Members(context.Context, uuid.UUID) ([]privacy.Member, error) { return f, nil }

// TestReflectInputNoIdentity — SRS 4.2.6 / AC4: lời gọi rút lưu ý không chứa họ tên, email, user_id, id bài nộp / kết quả của sinh viên.
func TestReflectInputNoIdentity(t *testing.T) {
	g := newGradeEnv(t, nil)
	g.tenReviewed("CANARY_X")
	g.exec(`update users set full_name = 'Phạm Canary ' || id::text, email = 'canary-' || id::text || '@sv.example.test' where id in (select user_id from enrollments where course_id = $1 and role_in_course = 'STUDENT')`, g.course)
	var secrets []string
	for _, q := range []string{
		`select full_name from users where id in (select user_id from enrollments where course_id = $1 and role_in_course = 'STUDENT')`,
		`select email from users where id in (select user_id from enrollments where course_id = $1 and role_in_course = 'STUDENT')`,
		`select id::text from users where id in (select user_id from enrollments where course_id = $1 and role_in_course = 'STUDENT')`,
		`select id::text from submissions where course_id = $1`, `select id::text from grading_results where course_id = $1`,
	} {
		var rows []string
		if strings.Contains(q, "$1") {
			rows = g.col(q, g.course)
		} else {
			rows = g.col(q)
		}
		secrets = append(secrets, rows...)
	}
	require.GreaterOrEqual(t, len(secrets), 6*5)
	var seen llm.Request
	r := &grading.Reflector{Pool: g.pool, Flags: guidanceFlagsOn(g, true), Cfg: grading.DefaultGuidanceConfig(),
		LLM: llmFunc(func(_ context.Context, req llm.Request, _ json.RawMessage) (json.RawMessage, error) {
			seen = req
			return json.RawMessage(`{"items":[]}`), nil
		})}
	_, err := r.Run(t.Context(), uuid.New(), g.teacher, grading.ReflectPayload{CourseID: g.course, AssignmentID: g.asg})
	require.NoError(t, err)
	require.NotEmpty(t, seen.Messages)
	for _, m := range seen.Messages {
		for _, s := range secrets {
			require.NotContains(t, m.Content, s, "danh tính không ra khỏi máy chủ")
		}
	}
}

func (g *gradeEnv) col(sql string, args ...any) []string {
	g.t.Helper()
	rows, err := g.pool.Query(g.t.Context(), sql, args...)
	require.NoError(g.t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		require.NoError(g.t, rows.Scan(&s))
		out = append(out, s)
	}
	return out
}

// TestReflectOutputPII — SRS 4.2.8 (vi) / AC12: nhận xét gốc của AI có tên / MSSV; LLM chép lại vào lưu ý → mục đó bị bỏ (đếm dropped.pii), không bao giờ thành DRAFT.
func TestReflectOutputPII(t *testing.T) {
	const name, code = "Lê Quang Khải", "B21DCAT123"
	g := newGradeEnv(t, nil)
	g.aiC1 = "AI nhận xét: " + name + " (" + code + ") nêu tên mà chưa phân tích."
	g.tenReviewed("CANARY_P")
	r := &grading.Reflector{Pool: g.pool, Flags: guidanceFlagsOn(g, true), Cfg: grading.DefaultGuidanceConfig(),
		Det: &privacy.Detector{Roster: &privacy.Roster{Src: fixedRoster{{Name: name, Code: code}}}},
		LLM: llmFunc(func(_ context.Context, req llm.Request, _ json.RawMessage) (json.RawMessage, error) {
			c1 := caseIDsFor(t, req, "c1")
			c2 := caseIDsFor(t, req, "c2")
			b, _ := json.Marshal(map[string]any{"items": []map[string]any{
				{"criterion_id": "c1", "text": "Chấm chặt hơn khi chỉ nêu tên tác nhân mà chưa phân tích.", "direction": "STRICTER", "case_ids": c1},
				{"criterion_id": "c1", "text": "Giống bài của " + name + " thì chấm chặt hơn.", "direction": "STRICTER", "case_ids": c1},
				{"criterion_id": "c2", "text": "Mã " + code + " cho thấy cần ví dụ thực tế hơn.", "direction": "LENIENT", "case_ids": c2},
				{"criterion_id": "c2", "text": "Liên hệ nguyenvana@example.test để hỏi thêm ví dụ.", "direction": "LENIENT", "case_ids": c2},
			}})
			return b, nil
		})}
	res, err := r.Run(t.Context(), uuid.New(), g.teacher, grading.ReflectPayload{CourseID: g.course, AssignmentID: g.asg})
	require.NoError(t, err)
	require.Equal(t, 1, res.Drafts)
	require.Equal(t, 3, res.Dropped.Count("pii"))
	for _, text := range g.col(`select text from grading_guidance_items where assignment_id = $1`, g.asg) {
		require.NotContains(t, text, name)
		require.NotContains(t, text, code)
		require.NotContains(t, text, "@")
	}
}

// TestGuidanceSetAtVersion — SRS 4.2.3 / US AC5: tập lưu ý tại phiên bản v là hàm thuần, tất định theo (bài tập, v); DRAFT / REJECTED không bao giờ vào; mục chờ nghỉ vẫn hiệu lực tới khi áp dụng; dựng lại bản cũ không đổi sau các lần áp dụng sau.
func TestGuidanceSetAtVersion(t *testing.T) {
	g := newGradeEnv(t, nil)
	add := func(text, status string, approved, retired *int, pending bool) {
		g.exec(`insert into grading_guidance_items (course_id, assignment_id, criterion_id, text, direction, support, status, decided_by, approved_in_version, retired_in_version, pending_retire)
			values ($1, $2, 'c1', $3, 'STRICTER', 3, $4::guidance_item_status, case when $4 = 'DRAFT' then null else $5::uuid end, $6, $7, $8)`, g.course, g.asg, text, status, g.teacher, approved, retired, pending)
	}
	i := func(n int) *int { return &n }
	add("A", "APPROVED", i(1), nil, false)
	add("B", "RETIRED", i(1), i(3), false)
	add("C", "APPROVED", i(2), nil, false)
	add("D", "DRAFT", nil, nil, false)
	add("E", "REJECTED", nil, nil, false)
	add("F", "APPROVED", i(2), nil, true) // chờ nghỉ: còn hiệu lực
	setAt := func(v int16) string {
		rows, err := store.New(g.pool).GuidanceSetAt(t.Context(), store.GuidanceSetAtParams{CourseID: g.course, AssignmentID: g.asg, V: v})
		require.NoError(t, err)
		var s []string
		for _, r := range rows {
			s = append(s, r.Text)
		}
		return strings.Join(s, ",")
	}
	for v, want := range map[int16]string{0: "", 1: "A,B", 2: "A,B,C,F", 3: "A,C,F", 4: "A,C,F"} {
		require.Equal(t, want, setAt(v), "v%d", v)
		require.Equal(t, want, setAt(v), "tất định, v%d", v)
	}
}

// TestGuidanceIsolation — AGENTS 2/4 + SRS FR-1: lưu ý khoá theo (course_id, assignment_id): bài A không vào tập / prompt của bài B, lớp này không vào lớp kia.
func TestGuidanceIsolation(t *testing.T) {
	g := newGradeEnv(t, nil)
	g.approveAt("c1", "LUU_Y_CUA_BAI_A", 1)
	// bài tập B cùng lớp + lớp khác có cùng id bài tập A
	var other uuid.UUID
	g.one(&other, `insert into assignments (course_id, title, description, type, max_score, due_at, rubric, created_by, submission_code)
		select course_id, 'BT khác', description, type, max_score, due_at, rubric, created_by, 'ISO2' from assignments where id = $1 returning id`, g.asg)
	g2 := newGradeEnv(t, nil)
	q := store.New(g.pool)
	for name, p := range map[string]store.GuidanceSetAtParams{
		"bài khác cùng lớp":    {CourseID: g.course, AssignmentID: other, V: 1},
		"lớp khác, id bài A":   {CourseID: g2.course, AssignmentID: g.asg, V: 1},
		"lớp khác, bài của nó": {CourseID: g2.course, AssignmentID: g2.asg, V: 1},
	} {
		rows, err := q.GuidanceSetAt(t.Context(), p)
		require.NoError(t, err)
		require.Empty(t, rows, name)
	}
	rows, err := q.GuidanceSetAt(t.Context(), store.GuidanceSetAtParams{CourseID: g.course, AssignmentID: g.asg, V: 1})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	// prompt: bài của lớp khác chấm với cờ bật không thấy lưu ý của bài A
	g2.grader.Flags = guidanceFlagsOn(g2, true)
	ss := samples(t)
	text := string(mustRead(t, asgFix+ss[0].File))
	g2.grader.LLM = llmFunc(func(_ context.Context, r llm.Request, _ json.RawMessage) (json.RawMessage, error) {
		g2.mu.Lock()
		g2.prompts = append(g2.prompts, r.Messages[0].Content)
		g2.mu.Unlock()
		return byPrompt(t, ss[:1], map[string]string{"S1": text})(g2, r)
	})
	res := g2.submit("A", "bai.md", text, nil)
	_, err = g2.grader.Run(t.Context(), g2.course, res)
	require.NoError(t, err)
	require.NotEmpty(t, g2.prompts)
	for _, p := range g2.prompts {
		require.NotContains(t, p, "LUU_Y_CUA_BAI_A")
		require.NotContains(t, p, "Lưu ý của giảng viên")
	}
}
