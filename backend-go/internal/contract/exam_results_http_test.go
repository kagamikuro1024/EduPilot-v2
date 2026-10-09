package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

// Canary của TestNoAnswerLeak (US-PE-08 AC18): mỗi chuỗi là duy nhất, gài ở một nơi sinh viên KHÔNG được thấy.
const (
	cvExpl  = "CANARY_EXPL_7c1f"    // explanation: chỉ lộ khi PUBLISHED và reveal_answers
	cvKey   = "CANARY_KEY_7c1f"     // trường lạ trong answer_key
	cvHName = "CANARY_HNAME_7c1f"   // tên test ẩn
	cvHIn   = "CANARY_HIN_7c1f"     // input test ẩn
	cvHOut  = "CANARY_HOUT_7c1f"    // expected test ẩn
	cvRef   = "CANARY_REF_7c1f"     // lời giải mẫu
	cvOther = "CANARY_OTHER_7c1f"   // bài nộp / đáp án của sinh viên khác
	cvEvent = "CANARY_EVENT_7c1f"   // nhật ký liêm chính
	cvOvr   = "CANARY_OVERRIDE_7c1" // nội dung override
	cvSim   = "CANARY_SIM_7c1f"     // ghi chú độ giống
)

// leakWorld: một lớp, Giảng viên, TA, Admin, sinh viên A (người gọi), sinh viên B (người khác) và một bài MIXED (MCQ + code) đầy canary.
type leakWorld struct {
	t                       *testing.T
	r                       *runner
	course, exam            string
	gv, ta, admin, svA, svB string
	aID, bID                uuid.UUID
	mcqItem, codeItem       string
	bAttempt, bSub          string
	sample, hidden          string
	tab                     string
}

func (w *leakWorld) exec(sql string, args ...any) {
	w.t.Helper()
	if _, err := w.r.rig.deps.DB.Exec(context.Background(), sql, args...); err != nil {
		w.t.Fatalf("%v: %s", err, sql)
	}
}

func (w *leakWorld) scan(sql string, dest any, args ...any) {
	w.t.Helper()
	if err := w.r.rig.deps.DB.QueryRow(context.Background(), sql, args...).Scan(dest); err != nil {
		w.t.Fatalf("%v: %s", err, sql)
	}
}

func buildLeakWorld(t *testing.T) *leakWorld {
	t.Helper()
	r := &runner{t: t, rig: getRig(t), cache: map[string]string{}}
	r.prod, r.test = mustLoad(t)
	r.freshIP()
	w := &leakWorld{t: t, r: r, tab: uuid.NewString()}
	db := r.rig.deps.DB
	mk := func(role string) uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(context.Background(), `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Người Thử', $2::user_role, 'ACTIVE', 'x', now()) returning id`,
			"ct-leak-"+uuid.NewString()[:8]+"@example.test", role).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	adminID, gvID, taID := mk("ADMIN"), mk("TEACHER"), mk("TA")
	w.aID, w.bID = mk("STUDENT"), mk("STUDENT")
	w.admin, w.gv = r.rig.token(t, adminID.String(), auth.RoleAdmin), r.rig.token(t, gvID.String(), auth.RoleTeacher)
	w.ta, w.svA, w.svB = r.rig.token(t, taID.String(), auth.RoleTA), r.rig.token(t, w.aID.String(), auth.RoleStudent), r.rig.token(t, w.bID.String(), auth.RoleStudent)
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "lk-" + uuid.NewString()} }
	_, b := r.must(call{method: "POST", path: "/api/v1/admin/courses", token: w.admin, headers: idem(), body: `{"subject_code":"INT1006","class_code":"LK-` + uuid.NewString()[:8] + `","name":"Lớp thử","semester":"2026-2027-HK1","teacher_id":"` + gvID.String() + `","ta_ids":["` + taID.String() + `"]}`}, 201)
	var c struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(b, &c)
	w.course = c.ID
	for _, u := range []uuid.UUID{w.aID, w.bID} {
		w.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, w.course, u)
	}
	qb := "/api/v1/courses/" + w.course + "/questions"
	type created struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	mkq := func(body string) created {
		_, b := r.must(call{method: "POST", path: qb, token: w.gv, body: body}, 201)
		var q created
		_ = json.Unmarshal(b, &q)
		return q
	}
	approve := func(q created) {
		_, b := r.must(call{method: "GET", path: qb + "/" + q.ID, token: w.gv}, 200)
		var cur created
		_ = json.Unmarshal(b, &cur)
		r.must(call{method: "PUT", path: qb + "/" + q.ID + "/review", token: w.gv, body: `{"decision":"APPROVE","version":` + itoa(cur.Version) + `}`}, 200)
	}
	mcq := mkq(`{"type":"MCQ_SINGLE","title":"2+2","topic":"Số học","difficulty":"EASY","stem":"2+2=?","explanation":"` + cvExpl + `","options":[{"body":"3"},{"body":"4"},{"body":"Tất cả đáp án trên","pinned_last":true}],"correct":[1]}`)
	approve(mcq)
	w.exec(`update question_bank set answer_key = answer_key || jsonb_build_object('canary', $2::text) where id=$1`, mcq.ID, cvKey)
	code := mkq(`{"type":"CODE","title":"a+b","topic":"Cơ bản","stem":"Đọc a b, in a+b."}`)
	r.must(call{method: "PUT", path: qb + "/" + code.ID + "/code", token: w.gv, body: `{"languages":["cpp17"],"starter_code":{"cpp17":"int main(){}"},"reference":{"language":"cpp17","source":"int main(){/*` + cvRef + `*/}"},"version":` + itoa(code.Version) + `}`}, 200)
	tcs := qb + "/" + code.ID + "/testcases"
	var s, h struct {
		ID string `json:"id"`
	}
	_, b = r.must(call{method: "POST", path: tcs, token: w.gv, body: `{"name":"sample1","input":"1 2","expected":"3","is_sample":true}`}, 201)
	_ = json.Unmarshal(b, &s)
	_, b = r.must(call{method: "POST", path: tcs, token: w.gv, body: `{"name":"` + cvHName + `","input":"` + cvHIn + `","expected":"` + cvHOut + `","weight":2}`}, 201)
	_ = json.Unmarshal(b, &h)
	w.sample, w.hidden = s.ID, h.ID
	w.exec(`update code_testcases set approved=true where problem_id=$1`, code.ID)
	w.exec(`update code_problems set reference_verified_version=tests_version, reference_verified_at=now() where question_id=$1`, code.ID)
	approve(code)
	eb := "/api/v1/courses/" + w.course + "/exams"
	now := time.Now().UTC()
	_, b = r.must(call{method: "POST", path: eb, token: w.gv, headers: idem(), body: `{"title":"Giữa kỳ","opens_at":"` + now.Add(2*time.Hour).Format(time.RFC3339) + `","closes_at":"` + now.Add(4*time.Hour).Format(time.RFC3339) + `","duration_minutes":45}`}, 201)
	var e created
	_ = json.Unmarshal(b, &e)
	w.exam = e.ID
	r.must(call{method: "PUT", path: eb + "/" + e.ID + "/items", token: w.gv, body: `{"items":[{"question_id":"` + mcq.ID + `","points":"1"},{"question_id":"` + code.ID + `","points":"1"}],"version":` + itoa(e.Version) + `}`}, 200)
	r.must(call{method: "POST", path: eb + "/" + e.ID + "/schedule", token: w.gv}, 200)
	w.scan(`select id::text from exam_items where exam_id=$1 and position=1`, &w.mcqItem, w.exam)
	w.scan(`select id::text from exam_items where exam_id=$1 and position=2`, &w.codeItem, w.exam)
	w.exec(`update exam_items set override = jsonb_build_object('note', $2::text) where id=$1`, w.mcqItem, cvOvr)
	// sinh viên B: lượt đã chấm, đáp án và bài nộp mang canary, sự kiện liêm chính mang canary
	w.scan(`insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at, status, submitted_at, submit_reason, auto_score, graded_at)
		values ($1, $2, $3, now() - interval '3 hours', now() - interval '2 hours', 'GRADED', now() - interval '2 hours', 'MANUAL', 5, now()) returning id::text`, &w.bAttempt, w.course, w.exam, w.bID)
	w.exec(`insert into exam_answers (course_id, attempt_id, item_id, answer) values ($1, $2, $3, jsonb_build_object('option_ids', jsonb_build_array($4::text), 'note', $4::text))`, w.course, w.bAttempt, w.mcqItem, cvOther)
	w.scan(`insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256, status, verdict, compile_ok, judged_at)
		values ($1, $2, $3, $4, $5, $6, 'SUBMIT', 'cpp17', $7, repeat('b', 64), 'DONE', 'AC', true, now()) returning id::text`, &w.bSub, w.course, w.exam, w.bAttempt, w.codeItem, code.ID, w.bID, "int main(){/*"+cvOther+"*/}")
	w.exec(`insert into exam_events (course_id, exam_id, attempt_id, student_id, type, meta) values ($1, $2, $3, $4, 'PASTE', jsonb_build_object('chars', 5, 'note', $5::text))`, w.course, w.exam, w.bAttempt, w.bID, cvEvent)
	return w
}

// state đặt trạng thái bài thi trong DB.
func (w *leakWorld) state(status string, reveal bool) {
	w.t.Helper()
	switch status {
	case "SCHEDULED":
		w.exec(`update exams set status='SCHEDULED', opens_at=now() + interval '2 hours', closes_at=now() + interval '4 hours', reveal_answers=$2 where id=$1`, w.exam, reveal)
	case "OPEN":
		w.exec(`update exams set status='OPEN', opens_at=now() - interval '1 minute', closes_at=now() + interval '3 hours', reveal_answers=$2 where id=$1`, w.exam, reveal)
	case "CLOSED":
		w.exec(`update exams set status='CLOSED', opens_at=now() - interval '4 hours', closes_at=now() - interval '1 minute', publish_hold=true, reveal_answers=$2 where id=$1`, w.exam, reveal)
	case "PUBLISHED":
		w.exec(`update exams set status='PUBLISHED', opens_at=now() - interval '4 hours', closes_at=now() - interval '1 minute', publish_hold=false, published_at=now(), reveal_answers=$2 where id=$1`, w.exam, reveal)
	}
}

// grade chấm xong lượt của A bằng SQL (máy chấm giả): breakdown đủ MCQ + code, bản nộp DONE với kết quả từng test.
func (w *leakWorld) grade(aAttempt string) {
	w.t.Helper()
	res, _ := json.Marshal([]map[string]any{
		{"test_id": w.sample, "position": 1, "is_sample": true, "verdict": "WA", "time_ms": 12, "memory_kb": 900},
		{"test_id": w.hidden, "position": 2, "is_sample": false, "verdict": "AC", "time_ms": 7, "memory_kb": 800},
	})
	w.exec(`update code_submissions set status='DONE', verdict='WA', compile_ok=true, results=$2::jsonb, passed_weight=2, total_weight=3, tests_version=1, judged_at=now(), lease_until=null where attempt_id=$1 and kind='SUBMIT'`, aAttempt, string(res))
	var sub string
	w.scan(`select id::text from code_submissions where attempt_id=$1 and kind='SUBMIT' limit 1`, &sub, aAttempt)
	bd, _ := json.Marshal([]map[string]any{
		{"item_id": w.mcqItem, "points": "1.00", "max": "1.00", "earned": "1"},
		{"item_id": w.codeItem, "points": "1.00", "max": "1.00", "earned": "0.6666666666666667", "code": map[string]any{"compile_ok": true, "hidden": map[string]int{"passed": 1, "total": 1}, "samples": []map[string]any{{"test_id": w.sample, "verdict": "WA", "time_ms": 12, "memory_kb": 900}}, "submission_id": sub}},
	})
	w.exec(`update exam_attempts set status='GRADED', auto_score=8.33, graded_at=now(), breakdown=$2::jsonb where id=$1`, aAttempt, string(bd))
}

type leakEndpoint struct {
	name, method, path, body string
	own                      bool // dùng được với id lượt của người khác (đường lỗi 404)
}

// TestNoAnswerLeak — US-PE-08 AC18: sinh viên A gọi MỌI endpoint dành cho sinh viên ở 6 tình huống (PUBLISHED chạy với reveal_answers bật và tắt); quét TỪNG byte của thân + header
// (kể cả lỗi và SSE) tìm canary. Ngoại lệ khai báo cụ thể: `explanation` của MCQ khi PUBLISHED + reveal_answers. Ba canary phải LUÔN vắng: trường lạ trong answer_key, test ẩn, lời giải mẫu,
// bài / đáp án của người khác, sự kiện, override, độ giống. Đối chứng dương: trong tình huống cho phép, `explanation` PHẢI có mặt (nên bộ quét không "mù").
func TestNoAnswerLeak(t *testing.T) {
	w := buildLeakWorld(t)
	r := w.r
	if err := r.rig.deps.Redis.Set(t.Context(), "ep:judge:up", "1", 2*time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	defer r.rig.deps.Redis.Del(context.Background(), "ep:judge:up")
	never := []string{cvKey, cvHName, cvHIn, cvHOut, cvRef, cvOther, cvEvent, cvOvr, cvSim}
	c, e := w.course, w.exam
	one := "/api/v1/courses/" + c + "/exams/" + e
	calls := 0
	check := func(scenario string, allowExpl bool, ep leakEndpoint, st int, h http.Header, body []byte) {
		t.Helper()
		calls++
		var all bytes.Buffer
		all.Write(body)
		for k, vs := range h {
			all.WriteString(k + ": " + strings.Join(vs, ",") + "\n")
		}
		for _, cn := range never {
			if bytes.Contains(all.Bytes(), []byte(cn)) {
				t.Errorf("[%s] %s %s → %d lộ %s: %.300s", scenario, ep.method, ep.path, st, cn, body)
			}
		}
		if !allowExpl && bytes.Contains(all.Bytes(), []byte(cvExpl)) {
			t.Errorf("[%s] %s %s → %d lộ giải thích khi chưa được phép: %.300s", scenario, ep.method, ep.path, st, body)
		}
	}
	runAll := func(scenario string, allowExpl bool, aAttempt string, skip ...string) {
		t.Helper()
		aid := aAttempt
		if aid == "" {
			aid = uuid.NewString()
		}
		at := func(id string) string { return one + "/attempts/" + id }
		ans := `{"items":[{"item_id":"` + w.mcqItem + `","answer":{"option_ids":["` + uuid.NewString() + `"]}}]}`
		evb := `{"events":[{"type":"PASTE","meta":{"chars":3}}]}`
		src := `{"language":"cpp17","source":"int main(){return 0;}"}`
		eps := []leakEndpoint{
			{"exams", "GET", "/api/v1/courses/" + c + "/exams", "", false},
			{"exam", "GET", one, "", false},
			{"start", "POST", one + "/attempts", "", false},
			{"mine", "GET", one + "/attempts/mine", "", false},
			{"answers", "PUT", at(aid) + "/answers", ans, true},
			{"takeover", "POST", at(aid) + "/takeover", `{"reload":false}`, true},
			{"events", "POST", at(aid) + "/events", evb, true},
			{"draft", "PUT", at(aid) + "/code/" + w.codeItem + "/draft", `{"language":"cpp17","source":"x","base_rev":0}`, true},
			{"run", "POST", at(aid) + "/code/" + w.codeItem + "/run", src, true},
			{"runs", "GET", at(aid) + "/runs/" + w.bSub, "", true},
			{"submitcode", "POST", at(aid) + "/code/" + w.codeItem + "/submit", src, true},
			{"submissions", "GET", at(aid) + "/code/" + w.codeItem + "/submissions", "", true},
			{"submission", "GET", at(aid) + "/submissions/" + w.bSub, "", true},
			{"submit", "POST", at(aid) + "/submit", "", true},
			{"result", "GET", at(aid) + "/result", "", true},
			{"appeal", "POST", at(aid) + "/appeal", `{"reason":"thử"}`, true},
			{"lock", "GET", "/api/v1/me/exam-lock", "", false},
			{"today", "GET", "/api/v1/me/today", "", false},
			{"coursetoday", "GET", "/api/v1/courses/" + c + "/today", "", false},
		}
		for _, ep := range eps {
			if slices.Contains(skip, ep.name) {
				continue
			}
			variants := []string{ep.path}
			if ep.own { // đường lỗi: cùng route nhưng id lượt của NGƯỜI KHÁC
				variants = append(variants, strings.Replace(ep.path, "/attempts/"+aid, "/attempts/"+w.bAttempt, 1))
			}
			for _, p := range variants {
				hd := map[string]string{"X-Exam-Tab": w.tab, "Idempotency-Key": "lk-" + uuid.NewString()}
				req := call{method: ep.method, path: p, token: w.svA, headers: hd, body: ep.body}
				st, h, b := r.do(req)
				check(scenario, allowExpl, leakEndpoint{ep.name, ep.method, p, "", false}, st, h, b)
				if st >= 500 && st != 503 {
					t.Errorf("[%s] %s %s → %d (lỗi máy chủ): %s", scenario, ep.method, p, st, b)
				}
			}
		}
		// SSE: mở luồng, đọc ~1 giây rồi quét.
		sctx, cancel := context.WithTimeout(t.Context(), 1500*time.Millisecond)
		defer cancel()
		sreq, _ := http.NewRequestWithContext(sctx, http.MethodGet, r.rig.srv.URL+"/api/v1/events", nil)
		sreq.Header.Set("Authorization", "Bearer "+w.svA)
		if resp, err := http.DefaultClient.Do(sreq); err == nil {
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			check(scenario, allowExpl, leakEndpoint{"sse", "GET", "/api/v1/events", "", false}, resp.StatusCode, resp.Header, b)
		}
	}

	w.state("SCHEDULED", true)
	runAll("SCHEDULED", false, "")
	w.state("OPEN", true)
	runAll("OPEN chưa bắt đầu", false, "", "start") // `start` có tác dụng phụ (bắt đầu lượt): để tình huống sau
	_, b := r.must(call{method: "POST", path: one + "/attempts", token: w.svA, headers: map[string]string{"X-Exam-Tab": w.tab, "Idempotency-Key": "lk-" + uuid.NewString()}}, 201)
	var st struct {
		Attempt struct {
			ID string `json:"id"`
		} `json:"attempt"`
	}
	_ = json.Unmarshal(b, &st)
	aAttempt := st.Attempt.ID
	runAll("OPEN đang làm", false, aAttempt, "submit") // `submit` kết thúc lượt: để tình huống sau
	hdr := func() map[string]string {
		return map[string]string{"X-Exam-Tab": w.tab, "Idempotency-Key": "lk-" + uuid.NewString()}
	}
	r.must(call{method: "POST", path: one + "/attempts/" + aAttempt + "/submit", token: w.svA, headers: hdr()}, 200)
	var aSub string
	w.scan(`select id::text from code_submissions where attempt_id=$1 and kind='SUBMIT' limit 1`, &aSub, aAttempt)
	lo, hi, sa, sb := aAttempt, w.bAttempt, aSub, w.bSub
	if lo > hi {
		lo, hi, sa, sb = hi, lo, sb, sa
	}
	var problem string
	w.scan(`select problem_id::text from code_submissions where id=$1`, &problem, aSub)
	w.exec(`insert into similarity_reports (course_id, exam_id, problem_id, run_id, submission_a, submission_b, attempt_a, attempt_b, score, shared_fingerprints, flagged, note)
		values ($1, $2, $3, $4, $5, $6, $7, $8, 0.9, 30, true, $9)`, c, e, problem, uuid.NewString(), sa, sb, lo, hi, cvSim)
	runAll("OPEN đã nộp", false, aAttempt)
	w.state("CLOSED", true)
	runAll("CLOSED chưa công bố", false, aAttempt)
	w.grade(aAttempt)
	w.state("PUBLISHED", true)
	runAll("PUBLISHED reveal_answers=true", true, aAttempt)
	_, rb := r.must(call{method: "GET", path: one + "/attempts/" + aAttempt + "/result", token: w.svA}, 200)
	if !bytes.Contains(rb, []byte(cvExpl)) {
		t.Fatalf("đối chứng dương: giải thích phải có khi PUBLISHED + reveal_answers: %s", rb)
	}
	w.state("PUBLISHED", false)
	runAll("PUBLISHED reveal_answers=false", false, aAttempt)
	_, rb = r.must(call{method: "GET", path: one + "/attempts/" + aAttempt + "/result", token: w.svA}, 200)
	if bytes.Contains(rb, []byte(cvExpl)) || bytes.Contains(rb, []byte(`"answer":{`)) {
		t.Fatalf("reveal_answers=false: không đáp án / giải thích: %s", rb)
	}
	if t.Failed() {
		return
	}
	fmt.Printf("leak_matrix 20x6 clean (%d lời gọi; PUBLISHED chạy hai lần reveal_answers)\n", calls)
	t.Logf("leak_matrix 20x6 clean (%d lời gọi)", calls)
}

// TestResultsPermissionMatrix — US-PE-08 AC19: ADMIN → 403 mọi route; sinh viên chỉ route của mình (lượt người khác → 404); TA đọc nhưng không sửa / không xem sự kiện, độ giống;
// Giảng viên làm được tất cả; người ngoài lớp → 403.
func TestResultsPermissionMatrix(t *testing.T) {
	w := buildLeakWorld(t)
	r := w.r
	one := "/api/v1/courses/" + w.course + "/exams/" + w.exam
	w.state("CLOSED", true)
	outsider := r.rig.token(t, uuid.NewString(), auth.RoleStudent)
	outsiderStaff := r.rig.token(t, uuid.NewString(), auth.RoleTeacher)
	bid := w.bAttempt
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "pm-" + uuid.NewString()} }
	type route struct {
		method, path, body string
		teacher, ta        bool // ai được phép (sinh viên không có route nào ở bảng này)
		idem               bool
	}
	routes := []route{
		{"PUT", one + "/publish-hold", `{"hold":true,"version":999}`, true, false, false},
		{"GET", one + "/results", "", true, true, false},
		{"GET", one + "/results.csv", "", true, true, false},
		{"GET", one + "/results/" + bid, "", true, true, false},
		{"PUT", one + "/results/" + bid + "/score", `{"score":null,"reason":"x","version":999}`, true, false, false},
		{"PUT", one + "/items/" + w.mcqItem + "/override", `{"reason":"x"}`, true, false, false},
		{"POST", one + "/regrade", `{"scope":"all","reason":"x"}`, true, false, true},
		{"GET", one + "/stats", "", true, true, false},
		{"GET", one + "/appeals", "", true, true, false},
		{"POST", one + "/appeals/" + uuid.NewString() + "/answer", `{"decision":"UPHELD","response":"x","version":1}`, true, false, false},
		{"GET", one + "/events?attempt=" + bid, "", true, false, false},
		{"GET", one + "/similarity", "", true, false, false},
	}
	for _, rt := range routes {
		call1 := func(tok string) int {
			h := map[string]string{}
			if rt.idem {
				h = idem()
			}
			st, _, _ := r.do(call{method: rt.method, path: rt.path, token: tok, headers: h, body: rt.body})
			return st
		}
		if st := call1(w.admin); st != http.StatusForbidden {
			t.Errorf("%s %s: ADMIN → %d (cần 403)", rt.method, rt.path, st)
		}
		if st := call1(w.svA); st != http.StatusForbidden {
			t.Errorf("%s %s: sinh viên → %d (cần 403)", rt.method, rt.path, st)
		}
		if st := call1(outsider); st != http.StatusForbidden {
			t.Errorf("%s %s: sinh viên ngoài lớp → %d (cần 403)", rt.method, rt.path, st)
		}
		if st := call1(outsiderStaff); st != http.StatusForbidden {
			t.Errorf("%s %s: giảng viên lớp khác → %d (cần 403)", rt.method, rt.path, st)
		}
		if st := call1(w.ta); (st == http.StatusForbidden) == rt.ta {
			t.Errorf("%s %s: TA → %d (được phép = %v)", rt.method, rt.path, st, rt.ta)
		}
		if st := call1(w.gv); st == http.StatusForbidden || st == http.StatusUnauthorized {
			t.Errorf("%s %s: Giảng viên → %d (phải được phép)", rt.method, rt.path, st)
		}
	}
	// sinh viên chỉ đọc kết quả của CHÍNH MÌNH: lượt người khác → 404 (dù bài đã công bố)
	w.state("PUBLISHED", true)
	r.must(call{method: "GET", path: one + "/attempts/" + bid + "/result", token: w.svA}, 404)
	r.must(call{method: "POST", path: one + "/attempts/" + bid + "/appeal", token: w.svA, headers: idem(), body: `{"reason":"x"}`}, 404)
	// TA không có khoá `flags`; Giảng viên có
	_, tb := r.must(call{method: "GET", path: one + "/results", token: w.ta}, 200)
	if bytes.Contains(tb, []byte(`"flags"`)) {
		t.Errorf("TA thấy `flags`: %s", tb)
	}
	_, gb := r.must(call{method: "GET", path: one + "/results", token: w.gv}, 200)
	if !bytes.Contains(gb, []byte(`"flags"`)) {
		t.Errorf("Giảng viên không thấy `flags`: %s", gb)
	}
	_, db := r.must(call{method: "GET", path: one + "/results/" + bid, token: w.ta}, 200)
	if bytes.Contains(db, []byte(`"integrity"`)) {
		t.Errorf("TA thấy `integrity` ở chi tiết: %s", db)
	}
}
