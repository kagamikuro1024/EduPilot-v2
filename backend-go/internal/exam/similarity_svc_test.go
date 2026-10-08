package exam_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// simExam dựng một bài code ĐÃ ĐÓNG (một câu code) rồi trả bài + id câu + id mục.
func (r *rig) simExam() (e exam.ExamDetail, qid, item uuid.UUID) {
	r.t.Helper()
	qid, _, _ = r.codeQuestion("{c11,cpp17}")
	e = r.openExam("so độ giống", false, qid)
	r.exec(`update exams set status='CLOSED', opens_at = now() - interval '4 hours', closes_at = now() - interval '1 minute' where id=$1`, e.ID)
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select id from exam_items where exam_id=$1`, e.ID).Scan(&item))
	return e, qid, item
}

// seedSubs tạo mỗi nguồn một sinh viên + một lượt (GRADING) + một bản SUBMIT cuối; trả id các bản nộp.
func (r *rig) seedSubs(e exam.ExamDetail, qid, item uuid.UUID, sources ...string) []uuid.UUID {
	r.t.Helper()
	ctx := r.t.Context()
	tx, err := r.pool.Begin(ctx)
	require.NoError(r.t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	var ids []uuid.UUID
	for i, src := range sources {
		var uid, att, sub uuid.UUID
		require.NoError(r.t, tx.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, $2, 'STUDENT') returning id`, fmt.Sprintf("sim.%s.%d@example.test", uuid.NewString()[:8], i), fmt.Sprintf("SV %d", i)).Scan(&uid))
		require.NoError(r.t, tx.QueryRow(ctx, `insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at, status, submitted_at, submit_reason) values ($1, $2, $3, now() - interval '3 hours', now() - interval '2 hours', 'GRADING', now() - interval '2 hours', 'MANUAL') returning id`, r.course, e.ID, uid).Scan(&att))
		require.NoError(r.t, tx.QueryRow(ctx, `insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256, status, verdict, compile_ok, judged_at)
			values ($1, $2, $3, $4, $5, $6, 'SUBMIT', 'cpp17', $7, repeat('a', 64), 'DONE', 'AC', true, now()) returning id`, r.course, e.ID, att, item, qid, uid, src).Scan(&sub))
		ids = append(ids, sub)
	}
	require.NoError(r.t, tx.Commit(ctx))
	return ids
}

// prog dựng một chương trình C++ từ "khuôn" (cấu trúc) và tên biến: cùng khuôn → giống nhau dù đổi tên; khác khuôn → khác.
func prog(shape int, v string) string {
	switch shape % 6 {
	case 0:
		return fmt.Sprintf(`#include <cstdio>
int main(){ int %[1]s, %[1]sq; scanf("%%d %%d", &%[1]s, &%[1]sq); long long sum = 0;
 for (int i = 0; i < %[1]s; i++) { int x; scanf("%%d", &x); sum += x * (long long)%[1]sq; if (sum > 1000000007LL) sum %%= 1000000007LL; }
 printf("%%lld\n", sum); for (int j = 0; j < 3; j++) printf("%%d\n", j); return 0; }`, v)
	case 1:
		return fmt.Sprintf(`#include <cstdio>
int gcdOf(int a, int b){ while (b != 0) { int t = a %% b; a = b; b = t; } return a; }
int main(){ int %[1]s; scanf("%%d", &%[1]s); int g = 0; while (%[1]s--) { int x; scanf("%%d", &x); g = gcdOf(g, x); }
 printf("%%d\n", g); if (g > 1) printf("YES\n"); else printf("NO\n"); return 0; }`, v)
	case 2:
		return fmt.Sprintf(`#include <cstdio>
#include <algorithm>
int arr[100005];
int main(){ int %[1]s; scanf("%%d", &%[1]s); for (int i = 0; i < %[1]s; i++) scanf("%%d", &arr[i]); std::sort(arr, arr + %[1]s);
 int best = 0, run = 1; for (int i = 1; i < %[1]s; i++) { if (arr[i] == arr[i-1] + 1) run++; else if (arr[i] != arr[i-1]) run = 1; best = std::max(best, run); }
 printf("%%d\n", best); return 0; }`, v)
	case 3:
		return fmt.Sprintf(`#include <cstdio>
#include <cstring>
char s[1005];
int main(){ scanf("%%s", s); int %[1]s = strlen(s); int cnt[26]; memset(cnt, 0, sizeof cnt);
 for (int i = 0; i < %[1]s; i++) cnt[s[i] - 'a']++; int odd = 0; for (int c = 0; c < 26; c++) if (cnt[c] %% 2) odd++;
 puts(odd <= 1 ? "PALIN" : "NO"); return 0; }`, v)
	case 4:
		return fmt.Sprintf(`#include <cstdio>
int fib(int n){ if (n < 2) return n; return fib(n - 1) + fib(n - 2); }
int main(){ int %[1]s; scanf("%%d", &%[1]s); for (int k = 0; k <= %[1]s; k++) { printf("%%d ", fib(k)); if (k %% 10 == 9) printf("\n"); } printf("\n"); return 0; }`, v)
	default:
		return fmt.Sprintf(`#include <cstdio>
int main(){ double %[1]s; scanf("%%lf", &%[1]s); double lo = 0, hi = %[1]s > 1 ? %[1]s : 1;
 for (int it = 0; it < 100; it++) { double mid = (lo + hi) / 2; if (mid * mid < %[1]s) lo = mid; else hi = mid; }
 printf("%%.6f\n", lo); if (lo > 100) printf("BIG\n"); return 0; }`, v)
	}
}

// uniq dựng một chuỗi token ngẫu nhiên có hạt giống (không cần biên dịch được: so độ giống chỉ nhìn token): hai hạt giống khác nhau gần như không có 5-gram chung, cùng hạt giống thì y hệt.
func uniq(seed int) string {
	words := []string{"if", "(", ")", "{", "}", ";", "x", "7", "+", "-", "*", "/", "<", ">", "==", "&&", "||", "for", "while", "return", "int", "=", "[", "]", "++", "--", "else", ",", "%", "!"}
	x := uint64(seed)*6364136223846793005 + 1442695040888963407
	var b strings.Builder
	for i := range 160 {
		x = x*6364136223846793005 + 1442695040888963407
		b.WriteString(words[int(x>>33)%len(words)])
		if i%12 == 11 {
			b.WriteString("\n")
		} else {
			b.WriteString(" ")
		}
	}
	return b.String()
}

func (r *rig) runSim(e exam.ExamDetail) exam.SimilarityResult {
	r.t.Helper()
	run := newRunner(r.t, r, &exam.Worker{})
	id, err := r.svc.EnqueueSimilarity(r.t.Context(), r.teacher, r.course, e.ID)
	require.NoError(r.t, err)
	j := runJob(r.t, r, run, id)
	require.Equal(r.t, "SUCCEEDED", j.Status)
	var res exam.SimilarityResult
	require.NoError(r.t, json.Unmarshal(j.Result, &res))
	return res
}

// TestSimilarityJobTopPairs — AC8: so mọi cặp khác sinh viên; giữ cặp ≥ 0,40 và ≥ 10 dấu vân tay chung; cặp giống (đổi tên biến) có điểm cao, cặp khác khuôn không có; lưu cùng một run_id; `attempt_a < attempt_b`.
func TestSimilarityJobTopPairs(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e, qid, item := r.simExam()
	srcs := []string{uniq(1000), uniq(1000)} // một cặp chép y hệt (đổi tên không có: cùng nguồn)
	for i := range 38 {
		srcs = append(srcs, uniq(i))
	}
	subs := r.seedSubs(e, qid, item, srcs...)
	res := r.runSim(e)
	require.Equal(t, 40, res.Docs)
	require.Equal(t, 1, res.Pairs, "một cặp trùng hẳn; 38 bài còn lại có cấu trúc riêng nên không đủ giống")
	rows, err := r.pool.Query(t.Context(), `select submission_a::text, submission_b::text, score::text, flagged, shared_fingerprints, run_id::text, (attempt_a < attempt_b) from similarity_reports where exam_id=$1 order by score desc, id`, e.ID)
	require.NoError(t, err)
	defer rows.Close()
	runs := map[string]bool{}
	in := map[string]bool{subs[0].String(): true, subs[1].String(): true}
	n := 0
	for rows.Next() {
		var a, b, score, run string
		var flagged, ordered bool
		var shared int
		require.NoError(t, rows.Scan(&a, &b, &score, &flagged, &shared, &run, &ordered))
		require.True(t, in[a] && in[b], "chỉ cặp trùng hẳn")
		require.Equal(t, "1.000", score)
		require.True(t, flagged, "≥ 0,60 và vượt mức nền của lớp")
		require.GreaterOrEqual(t, shared, 10)
		require.True(t, ordered)
		runs[run] = true
		n++
	}
	require.Equal(t, 1, n)
	require.Len(t, runs, 1)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.similarity_done' and payload->>'exam_id'=$1`, e.ID.String()))
}

// TestSimilarityRerunNewRunID — AC8: chạy lại tạo run_id MỚI và giữ bản cũ; danh sách mặc định là bản mới nhất, `run=` đọc bản cũ.
func TestSimilarityRerunNewRunID(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e, qid, item := r.simExam()
	r.seedSubs(e, qid, item, uniq(5), uniq(5))
	first := r.runSim(e)
	time.Sleep(20 * time.Millisecond)
	second := r.runSim(e)
	require.NotEqual(t, first.RunID, second.RunID)
	require.Equal(t, 2, r.count(`select count(distinct run_id) from similarity_reports where exam_id=$1`, e.ID))
	latest, err := r.svc.ListSimilarity(t.Context(), r.course, e.ID, nil, false, nil, 51)
	require.NoError(t, err)
	require.Len(t, latest, 1)
	require.Equal(t, second.RunID, latest[0].RunID)
	old, err := r.svc.ListSimilarity(t.Context(), r.course, e.ID, &first.RunID, false, nil, 51)
	require.NoError(t, err)
	require.Len(t, old, 1)
	require.Equal(t, first.RunID, old[0].RunID)
}

// TestSimilarityScale1000 — AC8: 1.000 bài nộp ≤ 30 s (chỉ mục ngược); tối đa 200 cặp / bài; không O(n²) thô.
func TestSimilarityScale1000(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e, qid, item := r.simExam()
	srcs := make([]string, 1000)
	for i := range srcs {
		srcs[i] = prog(i, fmt.Sprintf("v%d", i))
	}
	r.seedSubs(e, qid, item, srcs...)
	t0 := time.Now()
	res := r.runSim(e)
	took := time.Since(t0)
	t.Logf("1.000 bài: %s, %d cặp, %d gắn cờ", took, res.Pairs, res.Flagged)
	require.LessOrEqual(t, took, 30*time.Second)
	require.Equal(t, 1000, res.Docs)
	require.Equal(t, 200, res.Pairs, "giữ tối đa 200 cặp điểm cao nhất")
}

// TestSimilarityReviewStates — AC9: `Đã xem` (CLEARED) / `Cần trao đổi` (FOLLOW_UP) kèm ghi chú ≤ 500 ký tự; trạng thái lạ và ghi chú quá dài → 422; có audit và outbox; không có đường nào đổi điểm.
func TestSimilarityReviewStates(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	e, qid, item := r.simExam()
	r.seedSubs(e, qid, item, uniq(5), uniq(5))
	r.runSim(e)
	list, err := r.svc.ListSimilarity(t.Context(), r.course, e.ID, nil, true, nil, 51)
	require.NoError(t, err)
	require.Len(t, list, 1)
	id := list[0].ID
	require.Equal(t, "NEW", list[0].ReviewState)
	note := "Cùng cách làm tự nhiên"
	v, err := r.svc.ReviewSimilarity(t.Context(), r.teacher, r.course, e.ID, id, exam.SimilarityReviewIn{State: "CLEARED", Note: &note})
	require.NoError(t, err)
	require.Equal(t, "CLEARED", v.ReviewState)
	require.Equal(t, note, *v.Note)
	require.NotNil(t, v.ReviewedAt)
	v, err = r.svc.ReviewSimilarity(t.Context(), r.teacher, r.course, e.ID, id, exam.SimilarityReviewIn{State: "FOLLOW_UP"})
	require.NoError(t, err)
	require.Equal(t, "FOLLOW_UP", v.ReviewState)
	require.Nil(t, v.Note)
	_, err = r.svc.ReviewSimilarity(t.Context(), r.teacher, r.course, e.ID, id, exam.SimilarityReviewIn{State: "NEW"})
	require.Equal(t, []string{"enum"}, fieldCodes(t, err))
	long := strings.Repeat("a", 501)
	_, err = r.svc.ReviewSimilarity(t.Context(), r.teacher, r.course, e.ID, id, exam.SimilarityReviewIn{State: "CLEARED", Note: &long})
	require.Equal(t, []string{"max"}, fieldCodes(t, err))
	_, err = r.svc.ReviewSimilarity(t.Context(), r.teacher, r.course, e.ID, uuid.New(), exam.SimilarityReviewIn{State: "CLEARED"})
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
	require.Equal(t, 2, r.count(`select count(*) from audit_log where action='exam.similarity.review' and entity_id=$1`, id.String()))
	require.Equal(t, 2, r.count(`select count(*) from outbox where topic='exam.similarity_reviewed'`)-r.count(`select count(*) from outbox where topic='exam.similarity_reviewed' and payload->>'exam_id' <> $1`, e.ID.String()))
	d, err := r.svc.GetSimilarity(t.Context(), r.course, e.ID, id)
	require.NoError(t, err)
	require.NotEmpty(t, d.A.MatchLines)
	require.Contains(t, d.A.Source, "while")
}

// TestSimilarityTeacherOnly — AC4 / AC9: mọi thao tác liêm chính của Giảng viên (#51–#54 và chi tiết) chỉ cho TEACHER; sinh viên gửi sự kiện (#38) chỉ cho STUDENT. (Ma trận vai × route thật chạy ở kịch bản hợp đồng: TA / SV / Admin → 403.)
func TestSimilarityTeacherOnly(t *testing.T) {
	t.Parallel()
	for _, rt := range exam.Routes() {
		switch rt.No {
		case 51, 52, 53, 54, 57:
			require.Equal(t, auth.TeacherRole, rt.Mode, "#%d %s", rt.No, rt.Path)
		case 38:
			require.Equal(t, auth.StudentRole, rt.Mode)
		}
	}
	_ = apierr.Forbidden
}

// TestEnqueueSimilarity — bài chưa đóng → 409; bài không có câu code → 422; bộ lập lịch xếp việc MỘT lần cho bài code đã đóng và đã nộp hết.
func TestEnqueueSimilarity(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	open := r.openExam("đang mở", false, r.approvedCode("c"))
	_, err := r.svc.EnqueueSimilarity(t.Context(), r.teacher, r.course, open.ID)
	st, _ := apiStatus(t, err)
	require.Equal(t, 409, st)
	mcq := r.openExam("trắc nghiệm", false, r.mcqSet(1)...)
	_, err = r.svc.EnqueueSimilarity(t.Context(), r.teacher, r.course, mcq.ID)
	require.Equal(t, []string{"NO_CODE_ITEMS"}, fieldCodes(t, err))
	e, _, _ := r.simExam()
	r.exec(`update exams set created_by=$2 where id=$1`, e.ID, r.teacher)
	before := r.count(`select count(*) from jobs where kind='exam.similarity'`)
	n, err := r.svc.EnqueueDueSimilarity(t.Context())
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)
	n2, err := r.svc.EnqueueDueSimilarity(t.Context())
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*) from audit_log where action='exam.similarity.queued' and entity_id=$1`, e.ID.String())-1)
	_ = n2
	require.Greater(t, r.count(`select count(*) from jobs where kind='exam.similarity'`), before)
	// bài còn lượt IN_PROGRESS thì chưa xếp
	e2, _, _ := r.simExam()
	r.exec(`insert into exam_attempts (course_id, exam_id, student_id, deadline_at) values ($1, $2, $3, now() + interval '1 hour')`, r.course, e2.ID, r.student("ACTIVE"))
	_, err = r.svc.EnqueueDueSimilarity(t.Context())
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*) from audit_log where action='exam.similarity.queued' and entity_id=$1`, e2.ID.String()))
}
