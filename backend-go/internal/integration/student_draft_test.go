package integration_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/grade"
)

// TestStudentNeverSeesDraftGrade — US-P7-09 AC7: bài trải qua (a) chưa chấm (b) QUEUED / RUNNING (c) DRAFT chưa cờ (d) DRAFT có cờ (e) REVIEWED chưa công bố (f) FAILED, canary ở nhận xét nháp và điểm nháp:
// MỌI bề mặt dành cho sinh viên (…/mine, …/assignments, /me/grades, tool tóm tắt điểm, /notifications, "Hôm nay", lịch, thư) không chứa canary, điểm nháp, flags, confidence, run_state, needs_review;
// trạng thái hiện cho sinh viên là chữ "đang chấm" (GRADING) cho (b)–(f).
func TestStudentNeverSeesDraftGrade(t *testing.T) {
	w := newGBWorld(t)
	r := w.r
	ctx := t.Context()
	student := w.sv["sv.gioi"]
	stuID := w.userOf["sv.gioi"]
	var gv uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `select created_by from courses where id = $1`, w.c1).Scan(&gv))
	var asg uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into assignments (course_id, title, max_score, due_at, submission_code, created_by, published_at, rubric)
		values ($1, 'Bài tập 03', 10, now() + interval '2 days', 'BT03', $2, now(), '[{"id":"c1","name":"Khái niệm","max_score":"4","guidance":"HDAN-RIENG"},{"id":"c2","name":"Ví dụ","max_score":"6","guidance":""}]'::jsonb) returning id`, w.c1, gv).Scan(&asg))
	var sub uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at, extracted_text)
		values ($1, $2, $3, 'IN_APP', 1, 'k', 'bai.txt', 1, 'text/plain', 'x', 'MATCHED', 'IN_APP', now(), 'Bài của em.') returning id`, w.c1, asg, stuID).Scan(&sub))

	const canaryText = "CANARY-DRAFT-7K3"
	const canaryScore = "6.37"
	crit := `[{"id":"c1","score":"3.37","comment":"` + canaryText + `","evidence":[{"quote":"` + canaryText + `","start":0,"end":3}]},{"id":"c2","score":"3.00","comment":"` + canaryText + `","evidence":[]}]`
	states := []struct {
		name, run, status, flags string
		wantState                string
	}{
		{"a chưa chấm", "", "", "", "SUBMITTED"},
		{"b QUEUED", "QUEUED", "DRAFT", "{}", "GRADING"},
		{"b RUNNING", "RUNNING", "DRAFT", "{}", "GRADING"},
		{"c DRAFT chưa cờ", "DONE", "DRAFT", "{}", "GRADING"},
		{"d DRAFT NEEDS_REVIEW", "DONE", "DRAFT", "{NEEDS_REVIEW}", "GRADING"},
		{"d DRAFT INJECTION_SUSPECTED", "DONE", "DRAFT", "{INJECTION_SUSPECTED}", "GRADING"},
		{"e REVIEWED chưa công bố", "DONE", "REVIEWED", "{NEEDS_REVIEW}", "GRADING"},
		{"f FAILED", "FAILED", "DRAFT", "{}", "GRADING"},
	}
	cid := w.c1.String()
	surfaces := []string{
		"/courses/" + cid + "/assignments/" + asg.String() + "/mine",
		"/courses/" + cid + "/assignments",
		"/courses/" + cid + "/me/grades",
		"/notifications",
		"/me/today",
		"/courses/" + cid + "/today",
		"/courses/" + cid + "/calendar?from=" + time.Now().UTC().Add(-24*time.Hour).Format(time.RFC3339) + "&to=" + time.Now().UTC().Add(30*24*time.Hour).Format(time.RFC3339),
	}
	bad := []string{canaryText, "HDAN-RIENG", `"flags"`, `"confidence"`, `"run_state"`, `"needs_review"`, `"ai_scores"`, "NEEDS_REVIEW", "INJECTION_SUSPECTED", "pass_detail"}
	for _, st := range states {
		t.Run(st.name, func(t *testing.T) {
			mustExec := func(sql string, args ...any) {
				_, err := r.pool.Exec(ctx, sql, args...)
				require.NoError(t, err, sql)
			}
			mustExec(`delete from grading_results where course_id = $1`, w.c1)
			if st.run != "" {
				total := "null"
				if st.run == "DONE" {
					total = canaryScore
				}
				mustExec(`insert into grading_results (course_id, submission_id, run_state, status, criteria, overall_feedback, total, ai_total_before_late, ai_scores, flags, confidence, pass_detail)
					values ($1, $2, $3, $4, $5::jsonb, $6, `+total+`, `+total+`, '{"c1":"3.37","c2":"3.00"}'::jsonb, $7::text[], 0.31, '{"x":"`+canaryText+`"}'::jsonb)`, w.c1, sub, st.run, st.status, crit, canaryText, st.flags)
			}
			for _, path := range surfaces {
				res := w.get(student, path)
				require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, res.code, path+": "+string(res.body))
				for _, b := range bad {
					require.NotContains(t, string(res.body), b, "%s (%s): lộ %q", path, st.name, b)
				}
				require.False(t, leaksNumber(res.body, canaryScore), "%s (%s): lộ điểm nháp %s", path, st.name, canaryScore)
				if strings.HasSuffix(path, "/mine") && res.code == http.StatusOK {
					j := res.json()
					require.Equal(t, st.wantState, j["state"], st.name)
					require.Nil(t, j["result"], "chưa công bố: không có kết quả")
				}
			}
			// tool `get_my_grade_summary` / `what_if_final_grade` đi cùng grade.MeGrades
			facts, _, err := agent.GradeAdapter{Grade: &grade.Service{Pool: r.pool}}.Summary(ctx, agent.TrustedContext{UserID: stuID, CourseID: w.c1})
			require.NoError(t, err)
			raw, _ := json.Marshal(facts)
			for _, b := range bad {
				require.NotContains(t, string(raw), b, "tool tóm tắt điểm lộ %q", b)
			}
			require.False(t, leaksNumber(raw, canaryScore), "tool tóm tắt điểm lộ điểm nháp %s", canaryScore)
			var n int
			require.NoError(t, r.pool.QueryRow(ctx, `select count(*)::int from mail_outbox where template = 'assignment_feedback'`).Scan(&n))
			require.Zero(t, n, "chưa công bố: không thư nhận xét")
			require.NoError(t, r.pool.QueryRow(ctx, `select count(*)::int from notifications where user_id = $1 and type = 'GRADE_PUBLISHED'`, stuID).Scan(&n))
			require.Zero(t, n)
		})
	}
}

// leaksNumber: `score` ("6.37") xuất hiện như MỘT SỐ đứng riêng (cũng thử dấu phẩy "6,37"). Khác so chuỗi con thô: số không được dính chữ số hoặc dấu chấm ở hai đầu,
// nên "…07:44:46.379254Z" (dấu thời gian) hay "16.37", "6.375" không bị tính là lộ (P-105-2: dương tính giả phụ thuộc giờ chạy); "6.37", 6.37, "điểm 6.37" vẫn bị bắt.
func leaksNumber(body []byte, score string) bool {
	re := regexp.MustCompile(`(?:^|[^0-9.,])` + regexp.QuoteMeta(score) + `(?:$|[^0-9])|(?:^|[^0-9.,])` + regexp.QuoteMeta(strings.Replace(score, ".", ",", 1)) + `(?:$|[^0-9])`)
	return re.Match(body)
}

// TestLeaksNumber chứng minh phép kiểm đủ chặt (bắt lộ thật) mà không báo nhầm dấu thời gian chứa chuỗi "6.37" (P-105-2).
func TestLeaksNumber(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{
		{`{"confirmed_at":"2026-11-14T07:44:46.379254Z"}`, false}, // đúng ca CI main 83ec3630
		{`{"at":"2026-11-14T07:44:46.37Z"}`, false},
		{`{"x":16.37}`, false},
		{`{"x":6.375}`, false},
		{`{"x":"1.6.37"}`, false},
		{`{"total":6.37}`, true},
		{`{"total":"6.37"}`, true},
		{`{"note":"được 6.37 điểm"}`, true},
		{`{"total":"6,37"}`, true},
		{`6.37`, true},
		{`{"a":"2026-11-14T07:44:46.379254Z","total":6.37}`, true}, // dấu thời gian vô hại không che lộ thật
	} {
		require.Equal(t, tc.want, leaksNumber([]byte(tc.body), "6.37"), tc.body)
	}
}
