package integration_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	mockimap "github.com/edupilot/backend-go/cmd/mock-imap/server"
	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/emersion/go-imap/v2"
)

// Dữ liệu sprint 9 của lớp 1 dựng trực tiếp trong CSDL từ đúng các tệp của BA (seed-plan.csv, expected_grading.json, extraction-lop1-v2.json) — cùng dữ liệu `scripts/seed.mjs` đưa vào qua API thật;
// phần HTTP (khớp tay, bỏ qua, chốt điểm, phân quyền) đi qua router thật. Đường nộp thật (presign, ZIP, IMAP, Teams) đã có test riêng (internal/submission, imap_test, teams_test).

const demoLLMFile = "../../../seed/demo/fake_llm.json"

type seedAsg struct {
	*gbWorld
	asg      uuid.UUID
	plan     []map[string]string
	code     map[string]string    // account → MSSV
	sub      map[string]uuid.UUID // account → bài nộp mới nhất
	unmatch  map[string]uuid.UUID // "zip30" | "mail29" | "foreign" → bài UNMATCHED
	rubric   string
	refText  string
	llmCalls int
}

func asgFixtureText(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(asgFixDir, rel))
	require.NoError(t, err)
	return string(b)
}

func newSeedAsg(t *testing.T) *seedAsg {
	t.Helper()
	w := newGBWorld(t)
	s := &seedAsg{gbWorld: w, plan: readCSV(t, asgFixDir+"seed-plan.csv"), code: map[string]string{}, sub: map[string]uuid.UUID{}, unmatch: map[string]uuid.UUID{}}
	for _, e := range readCSV(t, gbSeed+"expected_final_grades_v2.csv") {
		s.code[e["account"]] = e["student_code"]
	}
	var rb struct {
		Criteria json.RawMessage `json:"criteria"`
	}
	require.NoError(t, json.Unmarshal([]byte(asgFixtureText(t, "rubric-bt03.json")), &rb))
	s.rubric, s.refText = string(rb.Criteria), asgFixtureText(t, "dap-an-chuan-bt03.md")
	var gv uuid.UUID
	require.NoError(t, w.r.pool.QueryRow(t.Context(), `select created_by from courses where id = $1`, w.c1).Scan(&gv))
	// hạn đã qua > 2 ngày (cửa nộp muộn cũng đóng), như sau `gateway admin seed-age --assignment BT03`
	require.NoError(t, w.r.pool.QueryRow(t.Context(), `insert into assignments (course_id, title, description, max_score, due_at, allow_late, late_penalty_per_day, late_max_days, appeal_days, channels, submission_code, reference_answer, rubric, created_by, published_at)
		values ($1, 'Bài tập 03 — Phân tích một vụ tấn công thực tế', $2, 10, now() - interval '4 days', true, 0.5, 2, 7, ARRAY['IN_APP','UPLOAD','IMAP'], 'BT03', $3, $4::jsonb, $5, now() - interval '5 days') returning id`,
		w.c1, asgFixtureText(t, "de-bai-bt03.md"), s.refText, s.rubric, gv).Scan(&s.asg))
	return s
}

func (s *seedAsg) sampleFile(id string) string {
	var all []struct{ ID, File string }
	require.NoError(s.r.t, json.Unmarshal([]byte(asgFixtureText(s.r.t, "expected_grading.json")), &all))
	for _, e := range all {
		if e.ID == id {
			return e.File
		}
	}
	s.r.t.Fatalf("không có mẫu %s", id)
	return ""
}

// addSubmissions nạp 26 bản ghi của seed-plan.csv: 22 bài khớp (sv12 hai phiên bản), sv30 (ZIP) + thư sv29 + thư lạ là UNMATCHED. Trả blob (khoá → nội dung) cho Grader.
func (s *seedAsg) addSubmissions() blobMem {
	t := s.r.t
	ctx := t.Context()
	blobs := blobMem{}
	ins := func(account, sample, channel string, version int, hours float64) uuid.UUID {
		key := fmt.Sprintf("courses/%s/seed/%s-v%d.md", s.c1, account, version)
		text := asgFixtureText(t, s.sampleFile(sample))
		blobs[key] = []byte(text)
		var id uuid.UUID
		submitted := "now() - interval '4 days' + make_interval(secs => $7::float8 * 3600)"
		if channel == "UPLOAD" {
			submitted = "null"
		}
		require.NoError(t, s.r.pool.QueryRow(ctx, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at, received_at)
			values ($1, $2, $3, $4, $5, $6, 'bai.md', $8, 'text/markdown', $9, 'MATCHED', case when $4 = 'IN_APP' then 'IN_APP' else 'EMAIL' end, `+submitted+`, now() - interval '4 days' + make_interval(secs => $7::float8 * 3600)) returning id`,
			s.c1, s.asg, s.userOf[account], channel, version, key, hours, len(text), uuid.NewString()).Scan(&id))
		return id
	}
	for _, p := range s.plan {
		a := p["account"]
		if p["sample_v1"] == "" || p["initial_match"] != "MATCHED" {
			continue
		}
		h := 0.0
		if p["hours_vs_due"] != "" {
			_, err := fmt.Sscanf(p["hours_vs_due"], "%g", &h)
			require.NoError(t, err)
		}
		if p["sample_v2"] != "" {
			ins(a, p["sample_v1"], p["channel"], 1, h)
			s.sub[a] = ins(a, p["sample_v2"], p["channel"], 2, h)
			continue
		}
		s.sub[a] = ins(a, p["sample_v1"], p["channel"], 1, h)
	}
	un := func(name, source string, withAsg bool, sample string) {
		key := fmt.Sprintf("courses/%s/seed/%s.md", s.c1, name)
		blobs[key] = []byte(asgFixtureText(t, s.sampleFile(sample)))
		var asg any
		if withAsg {
			asg = s.asg
		}
		var id uuid.UUID
		require.NoError(t, s.r.pool.QueryRow(ctx, `insert into submissions (course_id, assignment_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, unmatched_reason, received_at)
			values ($1, $2, $3, 1, $4, 'bai.md', 10, 'text/markdown', $5, 'UNMATCHED', 'NO_STUDENT_SIGNAL', now() - interval '4 days') returning id`, s.c1, asg, source, key, uuid.NewString()).Scan(&id))
		s.unmatch[name] = id
	}
	un("zip30", "UPLOAD", true, "S2")
	un("mail29", "IMAP", false, "S2")
	un("foreign", "IMAP", true, "S3")
	return blobs
}

// demoGrader: Grader thật + provider giả đọc ĐÚNG seed/demo/fake_llm.json (đường Chat có schema của gateway); đếm lời gọi.
func (s *seedAsg) demoGrader(blobs blobMem) *grading.Grader {
	p := fake.NewNamed(fake.NewController(fake.Settings{DemoFile: demoLLMFile}), "demo", "")
	return &grading.Grader{Pool: s.r.pool, Blob: blobs, Extract: noExtract{}, Cfg: grading.DefaultEssayConfig(),
		LLM: structFn(func(ctx context.Context, req llm.Request, schema json.RawMessage) (json.RawMessage, error) {
			s.llmCalls++
			msgs := make([]provider.Message, len(req.Messages))
			for i, m := range req.Messages {
				msgs[i] = provider.Message{Role: m.Role, Content: m.Content}
			}
			res, err := p.Chat(ctx, provider.ChatOpts{Messages: msgs, Schema: schema})
			return json.RawMessage(res.Text), err
		})}
}

func lateDaysOf(hours float64) int {
	if hours <= 0 {
		return 0
	}
	d := int(hours / 24)
	if float64(d)*24 < hours {
		d++
	}
	return min(d, 2)
}

// TestSeedGradeAllMatchesExpected — US-S9-SEED AC2: 22 bài khớp, provider giả (seed/demo/fake_llm.json) cho ĐÚNG expected_grading.json theo mẫu của từng sinh viên (điểm tiêu chí, tổng trước / sau muộn, cờ);
// sv24 (S4, bài trống) không gọi LLM; sv12 chỉ chấm phiên bản 2; tổng 21 × 2 lượt = 42 lời gọi. `grade-all` chạy chính Grader này.
func TestSeedGradeAllMatchesExpected(t *testing.T) {
	s := newSeedAsg(t)
	blobs := s.addSubmissions()
	g := s.demoGrader(blobs)
	var want []struct {
		ID       string `json:"id"`
		Expected struct {
			Scores      map[string]string `json:"criteria_scores"`
			BeforeLate  string            `json:"total_before_late"`
			Total       string            `json:"total"`
			Flags       []string          `json:"flags"`
			NeedsReview bool              `json:"needs_review"`
		} `json:"expected"`
	}
	require.NoError(t, json.Unmarshal([]byte(asgFixtureText(t, "expected_grading.json")), &want))
	exp := map[string]int{}
	for i, w := range want {
		exp[w.ID] = i
	}
	require.Len(t, s.sub, 22)
	n := 0
	for _, p := range s.plan {
		sub, ok := s.sub[p["account"]]
		if !ok {
			continue
		}
		var res uuid.UUID
		require.NoError(t, s.r.pool.QueryRow(t.Context(), `insert into grading_results (course_id, submission_id) values ($1, $2) returning id`, s.c1, sub).Scan(&res))
		_, err := g.Run(t.Context(), s.c1, res)
		require.NoError(t, err, p["account"])
		sample := p["sample_v1"]
		if p["sample_v2"] != "" {
			sample = p["sample_v2"]
		}
		e := want[exp[sample]].Expected
		h := 0.0
		if p["hours_vs_due"] != "" {
			_, _ = fmt.Sscanf(p["hours_vs_due"], "%g", &h)
		}
		late := lateDaysOf(h)
		if p["channel"] == "UPLOAD" {
			late = 0
		}
		var total, before, state, errCode string
		var flags []string
		var needs bool
		var raw []byte
		require.NoError(t, s.r.pool.QueryRow(t.Context(), `select coalesce(total::text, ''), coalesce(ai_total_before_late::text, ''), run_state, coalesce(error_code, ''), flags, needs_review, criteria from grading_results where id = $1`, res).Scan(&total, &before, &state, &errCode, &flags, &needs, &raw))
		require.Equal(t, "DONE", state, "%s (%s): %s", p["account"], sample, errCode)
		var crit []struct {
			ID    string  `json:"id"`
			Score *string `json:"score"`
		}
		require.NoError(t, json.Unmarshal(raw, &crit))
		for _, c := range crit {
			require.NotNil(t, c.Score, p["account"])
			require.True(t, decimal.RequireFromString(*c.Score).Equal(decimal.RequireFromString(e.Scores[c.ID])), "%s %s: %s ≠ %s", p["account"], c.ID, *c.Score, e.Scores[c.ID])
		}
		wantTotal := decimal.RequireFromString(e.BeforeLate).Sub(decimal.NewFromFloat(0.5).Mul(decimal.NewFromInt(int64(late)))) // sv08 trễ 2 ngày: −1,0; mẫu S7 đã gồm muộn của B
		if wantTotal.Sign() < 0 {
			wantTotal = decimal.Zero
		}
		require.True(t, decimal.RequireFromString(before).Equal(decimal.RequireFromString(e.BeforeLate)), "%s tổng trước muộn %s ≠ %s", p["account"], before, e.BeforeLate)
		require.True(t, decimal.RequireFromString(total).Equal(wantTotal), "%s tổng %s ≠ %s", p["account"], total, wantTotal)
		require.ElementsMatch(t, e.Flags, append([]string{}, flags...), p["account"])
		require.Equal(t, e.NeedsReview, needs, p["account"])
		n++
	}
	require.Equal(t, 22, n)
	require.Equal(t, 42, s.llmCalls, "21 bài × 2 lượt; sv24 (bài trống S4) không gọi LLM")
	var drafts int
	require.NoError(t, s.r.pool.QueryRow(t.Context(), `select count(*)::int from grading_results where course_id = $1 and status = 'DRAFT'`, s.c1).Scan(&drafts))
	require.Equal(t, 22, drafts, "chấm xong chỉ là nháp, không công bố")
}

// ---- AC4: công thức bản 2 + bài chưa khớp ----

func (s *seedAsg) publishScores(onlyMatched bool, skip ...string) {
	t := s.r.t
	for _, p := range s.plan {
		a := p["account"]
		id, ok := s.sub[a]
		if !ok || p["published_final_score"] == "" || strings.Contains(strings.Join(skip, ","), a) {
			continue
		}
		s.r.must(`insert into grading_results (course_id, submission_id, run_state, status, criteria, total, ai_total_before_late, published_at, published_by, reviewed_at, reviewed_by)
			values ($1, $2, 'DONE', 'PUBLISHED', '[]'::jsonb, $3::numeric, $3::numeric, now() - interval '1 day', $4, now() - interval '1 day', $4)`, s.c1, id, p["published_final_score"], s.gvID(t))
	}
	_ = onlyMatched
}

func (s *seedAsg) gvID(t *testing.T) (id uuid.UUID) {
	require.NoError(t, s.r.pool.QueryRow(t.Context(), `select created_by from courses where id = $1`, s.c1).Scan(&id))
	return id
}

// confirmV2 đưa bản 2 (extraction-lop1-v2.json, gắn item Bài tập vào BT03 + item kiểm tra tuần vào bài thi) lên đã xác nhận.
func (s *seedAsg) confirmV2() {
	t := s.r.t
	var ex struct {
		Components []struct {
			Key, Name, Group, Weight string
			KindHint                 string `json:"kind_hint"`
		} `json:"components"`
		Rules []struct {
			Kind       string
			Applicable bool
			Params     json.RawMessage
		} `json:"rules"`
	}
	require.NoError(t, json.Unmarshal([]byte(asgFixtureText(t, "extraction-lop1-v2.json")), &ex))
	in := grade.PutInput{Version: 1}
	for _, c := range ex.Components {
		pc := grade.PutComponent{Key: c.Key, Name: c.Name, Grp: c.Group, Weight: c.Weight}
		switch c.KindHint {
		case "EXAM":
			pc.Items = []grade.PutItem{{Key: c.Key, Name: c.Name, MaxScore: "10", Source: "EXAM", ExamID: &s.exam}}
		case "ASSIGNMENT":
			asg := s.asg
			pc.Items = []grade.PutItem{{Key: c.Key, Name: c.Name, MaxScore: "10", Source: "ASSIGNMENT", AssignmentID: &asg}}
		}
		in.Components = append(in.Components, pc)
	}
	for _, ru := range ex.Rules {
		ap := ru.Applicable
		in.Rules = append(in.Rules, grade.PutRule{Kind: ru.Kind, Applicable: &ap, Params: ru.Params})
	}
	svc := &grade.Service{Pool: s.r.pool}
	gv := s.gvID(t)
	s.r.must(`insert into grade_schemes (course_id, scheme_no, created_by) values ($1, 2, $2)`, s.c1, gv) // bản nháp mới, như `grade-scheme/extract` tạo
	doc, err := svc.PutDraft(t.Context(), gv, s.c1, in)
	require.NoError(t, err)
	_, err = svc.ConfirmScheme(t.Context(), gv, s.c1, doc.Version)
	require.NoError(t, err)
}

func (s *seedAsg) preview() map[string]map[string]any {
	out := map[string]map[string]any{}
	cursor := ""
	for {
		res := s.get(s.gv, "/courses/"+s.c1.String()+"/gradebook/preview?limit=100"+cursor)
		require.Equal(s.r.t, http.StatusOK, res.code, string(res.body))
		j := res.json()
		for _, x := range j["items"].([]any) {
			m := x.(map[string]any)
			out[m["student_code"].(string)] = m
		}
		next, _ := j["next_cursor"].(string)
		if next == "" {
			return out
		}
		cursor = "&cursor=" + next
	}
}

func (s *seedAsg) nonSubmitters() []string {
	var out []string
	for _, p := range s.plan {
		if p["sample_v1"] == "" && p["account"] != "(ngoài lớp)" {
			out = append(out, p["account"])
		}
	}
	return out
}

// TestSeedUnmatchedBlocksZero — AC4: bản 2 xác nhận, còn bài UNMATCHED của Bài tập 03 → sinh viên không nộp INCOMPLETE ("Còn bài chưa khớp"), KHÔNG phải 0; `Chốt điểm` → 409 GRADES_INCOMPLETE.
func TestSeedUnmatchedBlocksZero(t *testing.T) {
	s := newSeedAsg(t)
	s.addSubmissions()
	s.publishScores(true)
	s.confirmV2()
	rows := s.preview()
	non := s.nonSubmitters()
	require.Len(t, non, 6)
	for _, a := range non {
		r := rows[s.code[a]]
		require.Equal(t, "INCOMPLETE", r["status"], a)
		require.Contains(t, r["verdict"], "INCOMPLETE", a)
	}
	for _, a := range []string{"sv.gioi", "sv.kha", "sv04"} {
		require.Equal(t, "COMPLETE", rows[s.code[a]]["status"], "%s: %v", a, rows[s.code[a]]["verdict"])
	}
	res := s.r.do(req{path: "/courses/" + s.c1.String() + "/gradebook/finalize", bearer: s.gv.access, body: map[string]any{}, hdr: map[string]string{"Idempotency-Key": "chk-" + uuid.NewString()}})
	require.Equal(t, http.StatusConflict, res.code, string(res.body))
	require.Equal(t, "GRADES_INCOMPLETE", res.json()["code"])
	var locked int
	require.NoError(t, s.r.pool.QueryRow(t.Context(), `select count(*)::int from final_grade_snapshots where course_id = $1`, s.c1).Scan(&locked))
	require.Zero(t, locked, "không ghi gì khi bị chặn")
}

// TestSeedV2AfterResolvingUnmatched — AC4: giảng viên gán tay sv29 (thư) và sv30 (ZIP), bỏ qua thư email lạ → công bố điểm → xem trước cả lớp ra 30/30 đúng seed/expected_final_grades_v2.csv
// (sinh viên không nộp = 0 theo MISSING_SCORE); bản 1 (seed/expected_final_grades.csv) không đổi băm.
func TestSeedV2AfterResolvingUnmatched(t *testing.T) {
	s := newSeedAsg(t)
	s.addSubmissions()
	s.publishScores(true)
	s.confirmV2()
	c := "/courses/" + s.c1.String()
	for acc, name := range map[string]string{"sv29": "mail29", "sv30": "zip30"} {
		id := s.unmatch[name]
		var ver int64
		require.NoError(t, s.r.pool.QueryRow(t.Context(), `select (extract(epoch from updated_at) * 1000000)::bigint from submissions where id = $1`, id).Scan(&ver))
		res := s.r.do(req{method: http.MethodPut, path: c + "/submissions/" + id.String() + "/match", bearer: s.gv.access, body: map[string]any{"assignment_id": s.asg, "student_id": s.userOf[acc], "version": ver}})
		require.Equal(t, http.StatusOK, res.code, acc+": "+string(res.body))
		s.sub[acc] = id
	}
	// kết quả đã công bố của hai bài vừa gán (grade-all + duyệt + công bố ở bước thật; ở đây nạp thẳng điểm công bố của seed-plan.csv)
	for _, p := range s.plan {
		if a := p["account"]; a == "sv29" || a == "sv30" {
			s.r.must(`insert into grading_results (course_id, submission_id, run_state, status, criteria, total, ai_total_before_late, published_at, published_by, reviewed_at, reviewed_by)
				values ($1, $2, 'DONE', 'PUBLISHED', '[]'::jsonb, $3::numeric, $3::numeric, now(), $4, now(), $4)`, s.c1, s.sub[a], p["published_final_score"], s.gvID(t))
		}
	}
	var ver int64
	foreign := s.unmatch["foreign"]
	require.NoError(t, s.r.pool.QueryRow(t.Context(), `select (extract(epoch from updated_at) * 1000000)::bigint from submissions where id = $1`, foreign).Scan(&ver))
	res := s.r.do(req{path: c + "/submissions/" + foreign.String() + "/dismiss", bearer: s.gv.access, body: map[string]any{"reason": "Thư từ email lạ, MSSV ngoài lớp", "version": ver}})
	require.Contains(t, []int{http.StatusOK, http.StatusNoContent}, res.code, string(res.body))
	rows := s.preview()
	require.Len(t, rows, 30)
	for _, e := range readCSV(t, gbSeed+"expected_final_grades_v2.csv") {
		r := rows[e["student_code"]]
		require.NotNil(t, r, e["account"])
		require.Equal(t, "COMPLETE", r["status"], "%s: %v", e["account"], r["verdict"])
		require.True(t, decimal.RequireFromString(r["hp"].(string)).Equal(decimal.RequireFromString(e["hp"])), "%s: HP %v ≠ %s", e["account"], r["hp"], e["hp"])
		require.Equal(t, e["letter"], r["letter"], e["account"])
	}
}

// TestSeedAssignmentsIsolation — AC8: A, B, C chỉ đọc bài nộp / điểm của chính mình; D (chưa vào lớp) không thấy bài tập; TA thấy hàng chờ lớp 1 (không công bố), không thấy lớp 2.
func TestSeedAssignmentsIsolation(t *testing.T) {
	s := newSeedAsg(t)
	s.addSubmissions()
	c, a := "/courses/"+s.c1.String(), s.asg.String()
	b := s.sub["sv.kha"]
	s.r.must(`insert into grading_results (course_id, submission_id, run_state, status, criteria, overall_feedback, total, ai_total_before_late, flags) values ($1, $2, 'DONE', 'DRAFT', '[]'::jsonb, 'CANARY-DRAFT-7K3', 6.5, 7, ARRAY['NEEDS_REVIEW'])`, s.c1, b)
	for _, acc := range []string{"sv.gioi", "sv.kha", "sv.nguyco"} {
		res := s.get(s.sv[acc], c+"/assignments/"+a+"/mine")
		require.Equal(t, http.StatusOK, res.code, acc)
		j := res.json()
		if acc == "sv.nguyco" {
			require.Equal(t, "NOT_SUBMITTED", j["state"], "C không nộp")
		} else {
			require.Equal(t, s.fileOf(acc), j["submission"].(map[string]any)["file_name"], acc)
		}
		require.Nil(t, j["result"], acc)
		other := s.userOf["sv.gioi"]
		if acc == "sv.gioi" {
			other = s.userOf["sv.kha"]
		}
		res2 := s.get(s.sv[acc], c+"/assignments/"+a+"/mine?student_id="+other.String())
		require.Equal(t, res.body, res2.body, acc+": tham số tự khai không đổi người xem")
		require.Equal(t, http.StatusForbidden, s.get(s.sv[acc], c+"/submissions").code, acc)
		require.Equal(t, http.StatusForbidden, s.get(s.sv[acc], c+"/grading-results/"+uuid.NewString()).code, acc)
		require.Equal(t, http.StatusForbidden, s.get(s.sv[acc], c+"/mail-inbound").code, acc)
		require.Equal(t, http.StatusForbidden, s.get(s.sv[acc], "/courses/"+s.c2.String()+"/assignments").code, acc+" lớp 2")
	}
	// D chưa vào lớp nào
	require.Equal(t, http.StatusForbidden, s.get(s.dd, c+"/assignments").code)
	require.Equal(t, http.StatusForbidden, s.get(s.dd, c+"/assignments/"+a+"/mine").code)
	// TA: hàng chờ lớp 1 đọc được; công bố thì không; lớp 2 không thấy
	require.Equal(t, http.StatusOK, s.get(s.ta, c+"/submissions").code)
	pub := s.r.do(req{path: c + "/grading-results/publish", bearer: s.ta.access, body: map[string]any{"assignment_id": s.asg, "scope": "ALL_REVIEWED"}, hdr: map[string]string{"Idempotency-Key": "chk-" + uuid.NewString()}})
	require.Equal(t, http.StatusForbidden, pub.code)
	require.Equal(t, http.StatusForbidden, s.get(s.ta, "/courses/"+s.c2.String()+"/submissions").code)
	// lớp 2 không có bài tập nào
	require.Empty(t, s.get(s.gv, "/courses/"+s.c2.String()+"/assignments").json()["items"])
}

func (s *seedAsg) fileOf(account string) (f string) {
	require := require.New(s.r.t)
	require.NoError(s.r.pool.QueryRow(s.r.t.Context(), `select file_name from submissions where id = $1`, s.sub[account]).Scan(&f))
	return f
}

// TestSeedAssignmentsAdminForbidden — AC8: ADMIN 403 ở mọi route bài tập / chấm / tích hợp của lớp.
func TestSeedAssignmentsAdminForbidden(t *testing.T) {
	s := newSeedAsg(t)
	s.addSubmissions()
	c, a := "/courses/"+s.c1.String(), s.asg.String()
	sid := s.sub["sv.kha"].String()
	for _, p := range []string{"/assignments", "/assignments/" + a, "/assignments/" + a + "/stats", "/assignments/" + a + "/versions?student_id=" + s.userOf["sv.kha"].String(), "/submissions", "/submissions/" + sid + "/file", "/mail-inbound"} {
		require.Equal(t, http.StatusForbidden, s.get(s.admin, c+p).code, p)
	}
	for _, p := range []string{"/assignments/" + a + "/grade-all", "/assignments/" + a + "/publish", "/assignments/" + a + "/teams/sync", "/grading-results/publish", "/integrations/imap/sync-now"} {
		res := s.r.do(req{path: c + p, bearer: s.admin.access, body: map[string]any{}, hdr: map[string]string{"Idempotency-Key": "chk-" + uuid.NewString()}})
		require.Equal(t, http.StatusForbidden, res.code, p)
	}
}

// TestSeedCanaryNotToStudents — AC8: canary đáp án chuẩn (CANARY-BT03-ANSWER) và nhận xét nháp (CANARY-DRAFT-7K3) không lọt vào bất kỳ phản hồi nào dành cho sinh viên, ở mọi trạng thái chưa công bố.
func TestSeedCanaryNotToStudents(t *testing.T) {
	s := newSeedAsg(t)
	s.addSubmissions()
	require.Contains(t, s.refText, "CANARY-BT03-ANSWER")
	c, a := "/courses/"+s.c1.String(), s.asg.String()
	b := s.sub["sv.kha"]
	for _, st := range []struct{ run, status string }{{"", ""}, {"QUEUED", "DRAFT"}, {"DONE", "DRAFT"}, {"DONE", "REVIEWED"}} {
		s.r.must(`delete from grading_results where course_id = $1`, s.c1)
		if st.run != "" {
			s.r.must(`insert into grading_results (course_id, submission_id, run_state, status, criteria, overall_feedback, total, ai_total_before_late, flags, reviewed_at) values ($1, $2, $3, $4, '[{"id":"c1","score":"2.00","comment":"CANARY-DRAFT-7K3","evidence":[]}]'::jsonb, 'CANARY-DRAFT-7K3', 6.37, 6.37, ARRAY['NEEDS_REVIEW'], case when $4 = 'REVIEWED' then now() end)`, s.c1, b, st.run, st.status)
		}
		for _, acc := range []string{"sv.gioi", "sv.kha", "sv.nguyco"} {
			for _, p := range []string{c + "/assignments", c + "/assignments/" + a, c + "/assignments/" + a + "/mine", c + "/me/grades", "/notifications", "/me/today", c + "/today"} {
				res := s.get(s.sv[acc], p)
				require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, res.code, p)
				for _, bad := range []string{"CANARY-BT03-ANSWER", "CANARY-DRAFT-7K3", "6.37", "6,37", "reference_answer"} {
					require.NotContains(t, string(res.body), bad, "%s (%s/%s, %s): lộ %q", p, st.run, st.status, acc, bad)
				}
			}
		}
	}
}

// TestSeedImapAuthHeaders — AC1: thư seed do cmd/mock-imap dựng mang `Authentication-Results: mx.edupilot.test; dmarc=pass header.from={domain}` TRÊN CÙNG; Syncer (IMAP_TRUSTED_AUTHSERV_ID = mx.edupilot.test)
// khớp thư đúng mẫu theo email đã xác minh; thư mang authserv-id khác / không có header (thư lạ) KHÔNG được tin.
func TestSeedImapAuthHeaders(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.sy.TrustedAuthservID = "mx.edupilot.test"
	srv, err := mockimap.New(mockimap.Options{Username: "x", Password: "y"})
	require.NoError(t, err)
	att := []mockimap.Attachment{{Name: "bai.md", Base64: encode(s2()), Type: "text/markdown"}}
	deliver := func(m mockimap.Message) {
		raw, err := srv.Raw(m)
		require.NoError(t, err)
		_, err = r.box.Append("INBOX", literal{strings.NewReader(raw), int64(len(raw))}, &imap.AppendOptions{Time: m.InternalDate})
		require.NoError(t, err)
	}
	at := r.now.Add(-time.Hour)
	deliver(mockimap.Message{From: r.email(r.kha), Subject: "[BT03] 20229001", MessageID: "ok-" + uuid.NewString(), InternalDate: at, Attachments: att})
	res := r.mustSync()
	require.Equal(t, 1, res.Matched, "máy chủ thư của seed được tin")
	other := "mx.khac.test; dmarc=pass header.from=truong.edu"
	deliver(mockimap.Message{From: r.email(r.b), Subject: "[BT03] 20229002", MessageID: "other-" + uuid.NewString(), InternalDate: at, AuthResults: &other, Attachments: att})
	none := ""
	deliver(mockimap.Message{From: "ban.la@example.org", Subject: "[BT03] 20250001", MessageID: "foreign-" + uuid.NewString(), InternalDate: at, AuthResults: &none, Attachments: att})
	res = r.mustSync()
	require.Zero(t, res.Matched, "authserv-id lạ / không có header → không khớp")
	require.Equal(t, 2, res.Unmatched)
}

// TestSeedRosterJoinedVia — AC1: MSSV chỉ đáng tin khi sinh viên vào bằng danh sách lớp (`joined_via = ROSTER`): thư đúng mẫu `[BT03] MSSV` khớp sinh viên ROSTER, KHÔNG khớp sinh viên tự vào bằng mã lớp dù MSSV tự khai trùng.
func TestSeedRosterJoinedVia(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.sy.TrustedAuthservID = "mx.edupilot.test"
	srv, err := mockimap.New(mockimap.Options{Username: "x", Password: "y"})
	require.NoError(t, err)
	r.exec(`update enrollments set joined_via = 'CODE' where course_id = $1 and user_id = $2`, r.course, r.b) // b: MSSV 20229002 tự khai
	att := []mockimap.Attachment{{Name: "bai.md", Base64: encode(s2()), Type: "text/markdown"}}
	for i, m := range []mockimap.Message{
		{From: "nguoi.la@truong.edu", Subject: "[BT03] 20229001", MessageID: "roster-" + uuid.NewString()}, // MSSV của sinh viên ROSTER
		{From: "nguoi.la2@truong.edu", Subject: "[BT03] 20229002", MessageID: "code-" + uuid.NewString()},  // MSSV tự khai của sinh viên vào bằng mã lớp
	} {
		m.InternalDate, m.Attachments = r.now.Add(-time.Hour), att
		raw, err := srv.Raw(m)
		require.NoError(t, err, i)
		_, err = r.box.Append("INBOX", literal{strings.NewReader(raw), int64(len(raw))}, &imap.AppendOptions{Time: m.InternalDate})
		require.NoError(t, err)
	}
	res := r.mustSync()
	require.Equal(t, 1, res.Matched)
	require.Equal(t, 1, res.Unmatched)
	var student uuid.UUID
	r.one(&student, `select student_id from submissions where course_id = $1 and match_status = 'MATCHED'`, r.course)
	require.Equal(t, r.kha, student, "khớp theo MSSV ROSTER")
}

// TestSeedTeamsHappyCounts — AC3: kịch bản happy của mock-graph (sinh từ teams-plan.csv) với lớp có đủ 30 sinh viên seed → 24 bài khớp + 1 UNMATCHED; email / MSSV của teams-plan khớp tài khoản seed.
func TestSeedTeamsHappyCounts(t *testing.T) {
	plan := readCSV(t, asgFixDir+"teams-plan.csv")
	code := map[string]string{}
	for _, e := range readCSV(t, gbSeed+"expected_final_grades_v2.csv") {
		code[e["account"]] = e["student_code"]
	}
	ext := 0
	for _, p := range plan {
		if _, ok := code[p["account"]]; !ok {
			ext++
			require.False(t, strings.HasSuffix(p["mail"], "@edupilot.local"), "người ngoài lớp không dùng email của lớp: %s", p["mail"])
			continue
		}
		require.Equal(t, p["account"]+"@edupilot.local", p["mail"], "email khớp tài khoản seed")
	}
	require.Equal(t, 1, ext, "đúng một email ngoài lớp")
	r := newTM(t)
	res := r.mustRun()
	require.Equal(t, 25, res.Created)
	require.Equal(t, 1, res.Unmatched)
	require.Equal(t, 24, r.count(`select count(*)::int from submissions where course_id = $1 and source = 'TEAMS' and match_status = 'MATCHED'`, r.w.Course))
	var linked string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select coalesce(teams_assignment_id, '') from assignments where course_id = $1 and teams_assignment_id is not null limit 1`, r.w.Course).Scan(&linked))
	require.Equal(t, r.meta.AssignmentID, linked, "bài tập trỏ đúng bài trong mock-graph")
}

func encode(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
