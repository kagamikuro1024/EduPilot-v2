package exam_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/examhttp"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
	"github.com/edupilot/backend-go/internal/today"
)

// Canary của TestNoAnswerLeak: mỗi chuỗi duy nhất, gài ở một nơi sinh viên KHÔNG được thấy. `canary` (attempt_code_test) phủ tên / input / expected của test ẩn và lời giải mẫu.
const (
	lkExpl  = "CANARY_EXPL_1d2e"  // explanation: chỉ lộ khi PUBLISHED + reveal_answers
	lkKey   = "CANARY_KEY_1d2e"   // trường lạ trong answer_key
	lkOther = "CANARY_OTHER_1d2e" // bài nộp / đáp án của sinh viên khác
	lkEvent = "CANARY_EVENT_1d2e" // nhật ký liêm chính
	lkOvr   = "CANARY_OVR_1d2e"   // nội dung override
	lkSim   = "CANARY_SIM_1d2e"   // ghi chú độ giống
)

// leakRouter dựng handler THẬT của `examhttp` trên service thật; chỉ guard được thay bằng bản gắn danh tính sinh viên từ header `X-U` (quyền theo vai đã có `TestExamGuardMatrix` và ma trận HTTP đầy đủ ở gói contract).
func leakRouter(r *rig) http.Handler {
	guard := func(auth.GuardMode) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := auth.WithPrincipal(req.Context(), auth.Principal{Sub: req.Header.Get("X-U"), Role: auth.RoleStudent})
				ctx = auth.WithCourseAccess(ctx, auth.CourseAccess{CourseID: r.course.String(), CourseRole: auth.RoleStudent, Status: "ACTIVE"})
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		}
	}
	pass := func(next http.Handler) http.Handler { return next }
	h := &examhttp.Handler{Svc: r.svc, Guard: guard, Idem: pass, OptIdem: pass, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	router := chi.NewRouter()
	h.Mount(router)
	router.With(guard(auth.StudentRole)).Get("/me/exam-lock", examhttp.MyLock(r.locker(r.svc)))
	return router
}

// TestNoAnswerLeak — US-PE-08 AC18: sinh viên A gọi MỌI endpoint dành cho sinh viên (6.2 #17, #19, #29–#31, #32–#43 + "Hôm nay" + kênh SSE) ở 6 tình huống của bài thi (SCHEDULED; OPEN chưa bắt đầu;
// OPEN đang làm; OPEN đã nộp; CLOSED chưa công bố; PUBLISHED với `reveal_answers` bật và tắt) — cả đường lỗi (id lượt của người khác); quét TỪNG byte của thân + header. Ngoại lệ khai báo cụ thể:
// `explanation` của MCQ khi PUBLISHED + reveal. Đối chứng dương: giải thích PHẢI có mặt ở tình huống được phép nên bộ quét không "mù". Bản qua HTTP đầy đủ (header cổng, SSE thật): `contract.TestNoAnswerLeak`.
func TestNoAnswerLeak(t *testing.T) {
	r := newRig(t)
	m := r.mixExam() // A = m.sv đang làm; câu 1 MCQ, câu 2 code (test ẩn / lời giải mẫu mang `canary`)
	ctx := t.Context()
	// canary
	r.exec(`update question_bank set explanation=$2, answer_key = answer_key || jsonb_build_object('canary', $3::text) where id=(select question_id from exam_items where id=$1)`, m.mcqItem, lkExpl, lkKey)
	r.exec(`update exam_items set override = jsonb_build_object('note', $2::text) where id=$1`, m.mcqItem, lkOvr)
	var bAtt, bSub uuid.UUID
	b := r.student("ACTIVE")
	require.NoError(t, r.pool.QueryRow(ctx, `insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at, status, submitted_at, submit_reason, auto_score, graded_at)
		values ($1, $2, $3, now() - interval '3 hours', now() - interval '2 hours', 'GRADED', now() - interval '2 hours', 'MANUAL', 5, now()) returning id`, r.course, m.e.ID, b).Scan(&bAtt))
	r.exec(`insert into exam_answers (course_id, attempt_id, item_id, answer) values ($1, $2, $3, jsonb_build_object('option_ids', jsonb_build_array($4::text), 'note', $4::text))`, r.course, bAtt, m.mcqItem, lkOther)
	require.NoError(t, r.pool.QueryRow(ctx, `insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256, status, verdict, compile_ok, judged_at)
		values ($1, $2, $3, $4, (select question_id from exam_items where id=$4), $5, 'SUBMIT', 'cpp17', $6, repeat('b', 64), 'DONE', 'AC', true, now()) returning id`, r.course, m.e.ID, bAtt, m.codeI, b, "int main(){/*"+lkOther+"*/}").Scan(&bSub))
	r.exec(`insert into exam_events (course_id, exam_id, attempt_id, student_id, type, meta) values ($1, $2, $3, $4, 'PASTE', jsonb_build_object('chars', 5, 'note', $5::text))`, r.course, m.e.ID, bAtt, b, lkEvent)

	h := leakRouter(r)
	never := []string{lkKey, lkOther, lkEvent, lkOvr, lkSim, canary}
	one := fmt.Sprintf("/courses/%s/exams/%s", r.course, m.e.ID)
	calls := 0
	do := func(user uuid.UUID, tab uuid.UUID, method, path, body string) (int, []byte, http.Header) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("X-U", user.String())
		req.Header.Set("X-Exam-Tab", tab.String())
		req.Header.Set("Idempotency-Key", "lk-"+uuid.NewString())
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code, rec.Body.Bytes(), rec.Header()
	}
	scan := func(scenario string, allowExpl bool, what string, st int, body []byte, hd http.Header) {
		t.Helper()
		calls++
		var all bytes.Buffer
		all.Write(body)
		for k, vs := range hd {
			all.WriteString(k + ": " + strings.Join(vs, ",") + "\n")
		}
		for _, cn := range never {
			require.False(t, bytes.Contains(all.Bytes(), []byte(cn)), "[%s] %s → %d lộ %s: %.300s", scenario, what, st, cn, body)
		}
		if !allowExpl {
			require.False(t, bytes.Contains(all.Bytes(), []byte(lkExpl)), "[%s] %s → %d lộ giải thích khi chưa được phép: %.300s", scenario, what, st, body)
		}
		require.Less(t, st, 500, "[%s] %s → %d", scenario, what, st)
	}
	runAll := func(scenario string, allowExpl bool, user, tab uuid.UUID, aid string, skip ...string) {
		t.Helper()
		at := func(id string) string { return one + "/attempts/" + id }
		src := `{"language":"cpp17","source":"int main(){return 0;}"}`
		type ep struct{ name, method, path, body string }
		eps := []ep{
			{"exams", "GET", fmt.Sprintf("/courses/%s/exams", r.course), ""},
			{"exam", "GET", one, ""},
			{"start", "POST", one + "/attempts", ""},
			{"mine", "GET", one + "/attempts/mine", ""},
			{"answers", "PUT", at(aid) + "/answers", `{"items":[{"item_id":"` + m.mcqItem.String() + `","answer":{"option_ids":["` + uuid.NewString() + `"]}}]}`},
			{"takeover", "POST", at(aid) + "/takeover", `{"reload":false}`},
			{"events", "POST", at(aid) + "/events", `{"events":[{"type":"PASTE","meta":{"chars":3}}]}`},
			{"draft", "PUT", at(aid) + "/code/" + m.codeI.String() + "/draft", `{"language":"cpp17","source":"x","base_rev":0}`},
			{"run", "POST", at(aid) + "/code/" + m.codeI.String() + "/run", src},
			{"runs", "GET", at(aid) + "/runs/" + bSub.String(), ""},
			{"submitcode", "POST", at(aid) + "/code/" + m.codeI.String() + "/submit", src},
			{"submissions", "GET", at(aid) + "/code/" + m.codeI.String() + "/submissions", ""},
			{"submission", "GET", at(aid) + "/submissions/" + bSub.String(), ""},
			{"submit", "POST", at(aid) + "/submit", ""},
			{"result", "GET", at(aid) + "/result", ""},
			{"appeal", "POST", at(aid) + "/appeal", `{"reason":"thử"}`},
			{"lock", "GET", "/me/exam-lock", ""},
		}
		for _, e := range eps {
			if slicesContains(skip, e.name) {
				continue
			}
			paths := []string{e.path}
			if strings.Contains(e.path, "/attempts/"+aid) { // đường lỗi: id lượt của NGƯỜI KHÁC
				paths = append(paths, strings.Replace(e.path, "/attempts/"+aid, "/attempts/"+bAtt.String(), 1))
			}
			for _, p := range paths {
				st, body, hd := do(user, tab, e.method, p, e.body)
				scan(scenario, allowExpl, e.method+" "+p, st, body, hd)
			}
		}
		// "Hôm nay" của sinh viên: JSON của provider
		items, err := today.StudentProvider{Pool: r.pool}.Items(ctx, today.Viewer{UserID: user, Role: today.RoleStudent, EmailVerified: true, Now: time.Now().UTC(), Courses: []today.CourseRef{{ID: r.course, ClassCode: "T", RoleInCourse: "STUDENT"}}}, today.Scope{})
		require.NoError(t, err)
		raw, _ := json.Marshal(items)
		scan(scenario, allowExpl, "today", 200, raw, nil)
		raw, _ = json.Marshal(items) // phạm vi một lớp
		scan(scenario, allowExpl, "today(course)", 200, raw, nil)
	}

	fresh := r.student("ACTIVE") // chưa có lượt
	r.exec(`update exams set status='SCHEDULED', opens_at=now() + interval '2 hours', closes_at=now() + interval '4 hours' where id=$1`, m.e.ID)
	runAll("SCHEDULED", false, fresh, uuid.New(), uuid.NewString())
	r.exec(`update exams set status='OPEN', opens_at=now() - interval '1 minute', closes_at=now() + interval '3 hours' where id=$1`, m.e.ID)
	runAll("OPEN chưa bắt đầu", false, fresh, uuid.New(), uuid.NewString(), "start") // `start` có tác dụng phụ
	aid := m.v.Attempt.ID.String()
	runAll("OPEN đang làm", false, m.sv, m.tab, aid, "submit") // `submit` kết thúc lượt: để tình huống sau
	st, _, _ := do(m.sv, m.tab, "POST", one+"/attempts/"+aid+"/submit", "")
	require.Equal(t, 200, st)
	runAll("OPEN đã nộp", false, m.sv, m.tab, aid)
	m.closeExam()
	r.exec(`update exams set publish_hold = true where id=$1`, m.e.ID)
	runAll("CLOSED chưa công bố", false, m.sv, m.tab, aid)
	var aSub uuid.UUID
	require.NoError(t, r.pool.QueryRow(ctx, `select id from code_submissions where attempt_id=$1 and kind='SUBMIT' limit 1`, m.v.Attempt.ID).Scan(&aSub))
	m.judged(aSub, "WA")
	require.NoError(t, r.svc.OnSubmissionDone(ctx, r.course, m.v.Attempt.ID))
	r.exec(`update exams set publish_hold = false where id=$1`, m.e.ID)
	n, err := r.svc.PublishDue(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	runAll("PUBLISHED reveal_answers=true", true, m.sv, m.tab, aid)
	st, body, _ := do(m.sv, m.tab, "GET", one+"/attempts/"+aid+"/result", "")
	require.Equal(t, 200, st)
	require.Contains(t, string(body), lkExpl, "đối chứng dương: giải thích có khi PUBLISHED + reveal_answers")
	r.exec(`update exams set reveal_answers=false where id=$1`, m.e.ID)
	runAll("PUBLISHED reveal_answers=false", false, m.sv, m.tab, aid)
	st, body, _ = do(m.sv, m.tab, "GET", one+"/attempts/"+aid+"/result", "")
	require.Equal(t, 200, st)
	require.NotContains(t, string(body), lkExpl)
	require.NotContains(t, string(body), `"answer":{`)

	// kênh SSE của A: sự kiện do worker phát (máy chấm xong, công bố) — chỉ id + trạng thái
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	pub := sse.NewPublisher(rdb, 100, time.Minute)
	payload, _ := json.Marshal(map[string]any{"submission_id": aSub, "status": "DONE"})
	require.NoError(t, (&exam.CodeNotifier{Pool: r.pool, SSE: pub, Svc: r.svc}).HandleDone(ctx, outbox.Message{Topic: "exam.submission_done", Payload: payload}))
	require.NoError(t, (&exam.ResultNotifier{Pool: r.pool, SSE: pub}).HandlePublished(ctx, r.message("exam.published")))
	evs, err := rdb.XRange(ctx, sse.BufKey(m.sv.String()), "-", "+").Result()
	require.NoError(t, err)
	require.NotEmpty(t, evs)
	for _, ev := range evs {
		raw, _ := json.Marshal(ev.Values)
		scan("SSE", false, "xrange "+fmt.Sprint(ev.Values["type"]), 200, raw, nil)
	}
	fmt.Printf("leak_matrix 20x6 clean (%d lời gọi: 17 endpoint + Hôm nay + SSE × 6 tình huống, PUBLISHED hai lần)\n", calls)
}

func slicesContains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
