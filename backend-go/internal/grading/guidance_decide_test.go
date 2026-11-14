package grading_test

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
)

func (g *gradeEnv) draftItem(crit, text string) uuid.UUID {
	g.t.Helper()
	var id uuid.UUID
	g.one(&id, `insert into grading_guidance_items (course_id, assignment_id, criterion_id, text, direction, support) values ($1, $2, $3, $4, 'STRICTER', 3) returning id`, g.course, g.asg, crit, text)
	return id
}

func (g *gradeEnv) itemVersion(id uuid.UUID) (v int) {
	g.t.Helper()
	g.one(&v, `select version from grading_guidance_items where id = $1`, id)
	return v
}

// TestApplyVersion — SRS 4.2.9–4.2.10: 2 APPROVE + 1 RETIRE → +1 đúng một lần; chỉ REJECT → NOTHING_TO_APPLY; expected_version cũ → 409; mục chờ nghỉ chỉ rời tập ở bản mới.
func TestApplyVersion(t *testing.T) {
	g := newGradeEnv(t, nil)
	svc, _ := newGuidanceSvc(t, g, true)
	a := grading.GuidanceActor{UserID: g.teacher}
	i1, i2, i3 := g.draftItem("c1", "Chấm chặt hơn khi chỉ nêu tên."), g.draftItem("c2", "Cần ví dụ thực tế."), g.draftItem("c3", "Rõ ràng hơn về bề mặt.")

	_, err := svc.Apply(t.Context(), a, g.course, g.asg, 0)
	_, code := apiCode(t, err)
	require.Equal(t, "NOTHING_TO_APPLY", code, "chưa duyệt mục nào")
	v, err := svc.Resolve(t.Context(), a, g.course, g.asg, i3, "REJECT", nil, 1)
	require.NoError(t, err)
	require.Equal(t, "REJECTED", v.Status)
	_, err = svc.Apply(t.Context(), a, g.course, g.asg, 0)
	_, code = apiCode(t, err)
	require.Equal(t, "NOTHING_TO_APPLY", code, "chỉ REJECT")

	_, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "APPROVE", nil, 99)
	_, code = apiCode(t, err)
	require.Equal(t, "VERSION_CONFLICT", code)
	v, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "APPROVE", nil, 1)
	require.NoError(t, err)
	require.Equal(t, "APPROVED", v.Status)
	require.Nil(t, v.ApprovedInVersion, "duyệt chưa là áp dụng")
	_, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "APPROVE", nil, g.itemVersion(i1))
	_, code = apiCode(t, err)
	require.Equal(t, "ITEM_NOT_DRAFT", code)
	_, err = svc.Resolve(t.Context(), a, g.course, g.asg, i2, "APPROVE", nil, 1)
	require.NoError(t, err)
	var cur int
	g.one(&cur, `select guidance_version from assignments where id = $1`, g.asg)
	require.Zero(t, cur, "duyệt không đổi phiên bản")

	_, err = svc.Apply(t.Context(), a, g.course, g.asg, 5)
	_, code = apiCode(t, err)
	require.Equal(t, "VERSION_CONFLICT", code)
	out, err := svc.Apply(t.Context(), a, g.course, g.asg, 0)
	require.NoError(t, err)
	require.Equal(t, 1, out.Version)
	var items, versions int
	g.one(&items, `select count(*) from grading_guidance_items where assignment_id = $1 and approved_in_version = 1`, g.asg)
	g.one(&versions, `select item_count from grading_guidance_versions where assignment_id = $1 and version = 1`, g.asg)
	require.Equal(t, 2, items)
	require.Equal(t, 2, versions)
	_, err = svc.Apply(t.Context(), a, g.course, g.asg, 1)
	_, code = apiCode(t, err)
	require.Equal(t, "NOTHING_TO_APPLY", code)

	// Cho nghỉ: chờ nghỉ vẫn có hiệu lực ở v1; sang v2 mới rời tập
	v, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "RETIRE", nil, g.itemVersion(i1))
	require.NoError(t, err)
	require.True(t, v.PendingRetire)
	var n int
	g.one(&n, `select count(*) from grading_guidance_items where assignment_id = $1 and status = 'APPROVED'`, g.asg)
	require.Equal(t, 2, n)
	out, err = svc.Apply(t.Context(), a, g.course, g.asg, 1)
	require.NoError(t, err)
	require.Equal(t, 2, out.Version)
	var st string
	var retired int
	g.one(&st, `select status::text from grading_guidance_items where id = $1`, i1)
	g.one(&retired, `select retired_in_version from grading_guidance_items where id = $1`, i1)
	require.Equal(t, "RETIRED", st)
	require.Equal(t, 2, retired)
	set1, err := queriesSetAt(t, g, 1), error(nil)
	require.NoError(t, err)
	require.Len(t, set1, 2, "bản 1 dựng lại được: hàm thuần theo (bài tập, v)")
	require.Len(t, queriesSetAt(t, g, 2), 1)
	g.one(&n, `select count(*) from audit_log where action like 'grading_guidance.%' and course_id = $1`, g.course)
	require.GreaterOrEqual(t, n, 6)
}

func queriesSetAt(t *testing.T, g *gradeEnv, v int16) []string {
	t.Helper()
	rows, err := g.pool.Query(t.Context(), `select id::text from grading_guidance_items where course_id = $1 and assignment_id = $2 and status in ('APPROVED','RETIRED')
		and approved_in_version is not null and approved_in_version <= $3 and (retired_in_version is null or retired_in_version > $3)`, g.course, g.asg, v)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		out = append(out, s)
	}
	return out
}

// TestGuidanceDecideText — sửa chữ khi duyệt chạy lại từ khoá + PII (422 GUIDANCE_TEXT_REJECTED), lưu text_original; RETIRE không kèm chữ; cờ tắt → 409.
func TestGuidanceDecideText(t *testing.T) {
	g := newGradeEnv(t, nil)
	svc, _ := newGuidanceSvc(t, g, true)
	a := grading.GuidanceActor{UserID: g.teacher}
	id := g.draftItem("c1", "Bản gốc của AI.")
	for _, bad := range []string{"Trừ điểm nếu nộp muộn", "", strings.Repeat("a", 401)} {
		_, err := svc.Resolve(t.Context(), a, g.course, g.asg, id, "APPROVE", &bad, 1)
		st, code := apiCode(t, err)
		require.Equal(t, http.StatusUnprocessableEntity, st, bad)
		require.Equal(t, "GUIDANCE_TEXT_REJECTED", code)
	}
	txt := "Bản đã sửa bởi giảng viên."
	v, err := svc.Resolve(t.Context(), a, g.course, g.asg, id, "APPROVE", &txt, 1)
	require.NoError(t, err)
	require.Equal(t, txt, v.Text)
	require.NotNil(t, v.TextOriginal)
	require.Equal(t, "Bản gốc của AI.", *v.TextOriginal)
	_, err = svc.Resolve(t.Context(), a, g.course, g.asg, id, "RETIRE", &txt, v.Version)
	st, _ := apiCode(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, st)
	off, _ := newGuidanceSvc(t, g, false)
	_, err = off.Resolve(t.Context(), a, g.course, g.asg, id, "APPROVE", nil, 1)
	_, code := apiCode(t, err)
	require.Equal(t, "GUIDANCE_DISABLED", code)
}

// TestGuidanceVersionOnlyViaApply — chỉ truy vấn GuidanceSetVersion (guidance.sql) được đổi assignments.guidance_version; mã Go khác không có câu UPDATE nào như vậy.
func TestGuidanceVersionOnlyViaApply(t *testing.T) {
	re := regexp.MustCompile("(?i)update\\s+assignments\\s+set\\s+[^;`]*guidance_version\\s*=")
	var hits []string
	require.NoError(t, filepath.WalkDir("../../internal", func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || (!strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, ".sql")) || strings.HasSuffix(p, "_test.go") {
			return err
		}
		b, _ := os.ReadFile(p)
		if re.Match(b) {
			hits = append(hits, filepath.ToSlash(p))
		}
		return nil
	}))
	require.ElementsMatch(t, []string{"../../internal/store/guidance.sql.go", "../../internal/store/queries/guidance.sql"}, hits)
}

// staleRes tạo một kết quả đã chấm xong (DONE) ở phiên bản ghim `pin`; `fin` là điểm cuối (khác điểm AI → đã sửa tay).
func (g *gradeEnv) staleRes(name, status string, pin int, round int, fin string, first bool) uuid.UUID {
	g.t.Helper()
	res := g.submit(name, "bai.md", "Nội dung "+name, nil)
	crit := `[{"id":"c1","score":"` + fin + `","comment":"","edited":false,"evidence":[]}]`
	g.exec(`update grading_results set run_state = 'DONE', criteria = $2, ai_scores = '{"c1":"2.00"}', ai_total_before_late = 2, total = 2, guidance_version = $3, round = $4 where id = $1`, res, crit, pin, round)
	if first {
		g.exec(`update grading_results set first_reviewed_by = $2, first_reviewed_at = now() where id = $1`, res, g.teacher)
	}
	if status != "DRAFT" {
		g.exec(`update grading_results set status = $2, reviewed_by = $3, reviewed_at = now(), first_reviewed_by = $3, first_reviewed_at = coalesce(first_reviewed_at, now()) where id = $1`, res, "REVIEWED", g.teacher)
	}
	return res
}

// TestRegradeStale + NeverOverwrites — SRS 4.2.11: chỉ DRAFT bản cũ, mọi tiêu chí chưa sửa, chưa từng duyệt → round + 1. KHÔNG BAO GIỜ chọn REVIEWED, DRAFT có tiêu chí sửa, DRAFT từng duyệt, round 20, bản đã mới.
func TestRegradeStale(t *testing.T) {
	g := newGradeEnv(t, nil)
	svc, _ := newGuidanceSvc(t, g, true)
	a := grading.GuidanceActor{UserID: g.teacher}
	g.approveAt("c1", "Lưu ý v1.", 1)
	fresh := g.staleRes("A", "DRAFT", 0, 1, "2.00", false)
	edited := g.staleRes("B", "DRAFT", 0, 1, "1.00", false)
	reopened := g.staleRes("C", "DRAFT", 0, 1, "2.00", true)
	reviewed := g.staleRes("D", "REVIEWED", 0, 1, "1.00", true)
	maxRound := g.staleRes("E", "DRAFT", 0, 20, "2.00", false)
	current := g.staleRes("F", "DRAFT", 1, 1, "2.00", false)
	snap := func() string {
		var s string
		g.one(&s, `select string_agg(id::text || ':' || coalesce(total::text,'') || ':' || status || ':' || round::text || ':' || criteria::text, '|' order by id) from grading_results where course_id = $1 and id = any($2)`,
			g.course, []string{reviewed.String(), edited.String(), reopened.String(), maxRound.String(), current.String()})
		return s
	}
	before := snap()

	out, err := svc.RegradeStale(t.Context(), a, g.course, g.asg)
	require.NoError(t, err)
	require.Equal(t, 1, out.Queued)
	require.NotNil(t, out.JobID)
	require.Equal(t, grading.RegradeSkipped{Reviewed: 1, Edited: 2, MaxRound: 1}, out.Skipped)
	var sub uuid.UUID
	g.one(&sub, `select submission_id from grading_results where id = $1`, fresh)
	var n int
	g.one(&n, `select count(*) from grading_results where submission_id = $1 and round = 2 and run_state = 'QUEUED'`, sub)
	require.Equal(t, 1, n, "round + 1 cho bài nháp chưa sửa")
	g.one(&n, `select count(*) from grading_results where course_id = $1 and round = 2`, g.course)
	require.Equal(t, 1, n, "không bài nào khác bị xếp chấm lại")
	require.Equal(t, before, snap(), "điểm / trạng thái của bài đã duyệt, đã sửa, đã đủ vòng không đổi")

	again, err := svc.RegradeStale(t.Context(), a, g.course, g.asg)
	require.NoError(t, err)
	require.Zero(t, again.Queued, "chạy lại: bài đã xếp QUEUED không bị xếp đôi")
	require.Nil(t, again.JobID)
	off, _ := newGuidanceSvc(t, g, false)
	_, err = off.RegradeStale(t.Context(), a, g.course, g.asg)
	_, code := apiCode(t, err)
	require.Equal(t, "GUIDANCE_DISABLED", code)
}

// TestGuidanceStats — decimal: MAE tiêu chí / tổng, lệch trung bình có dấu (dương = AI cao hơn giảng viên), theo phiên bản ghim; n < 5 → null.
func TestGuidanceStats(t *testing.T) {
	g := newGradeEnv(t, nil)
	svc, _ := newGuidanceSvc(t, g, true)
	for i := range 4 {
		g.staleRes("S"+string(rune('A'+i)), "REVIEWED", 0, 1, "1.00", true) // AI 2,00 → chốt 1,00: lệch +1,00
	}
	v, err := svc.Stats(t.Context(), g.course, g.asg)
	require.NoError(t, err)
	require.Len(t, v.Items, 1)
	require.Equal(t, 4, v.Items[0].N)
	require.Nil(t, v.Items[0].MaeTotal, "n < 5 → null")
	g.staleRes("S5", "REVIEWED", 0, 1, "1.50", true) // lệch +0,50
	v, err = svc.Stats(t.Context(), g.course, g.asg)
	require.NoError(t, err)
	it := v.Items[0]
	require.Equal(t, 5, it.N)
	require.Equal(t, "0.90", *it.MaeCriterion) // (4×1 + 0,5) / 5
	require.Equal(t, "0.90", *it.MaeTotal)
	require.Equal(t, "0.90", *it.BiasTotal)
	require.Equal(t, 5, it.EditedCriteria)
	require.Equal(t, "1.00", *it.EditedRate)
}

// round2Of trả id kết quả round + 1 của bài nộp có kết quả `res`.
func (g *gradeEnv) round2Of(res uuid.UUID) (r2 uuid.UUID) {
	g.t.Helper()
	g.one(&r2, `select x.id from grading_results x join grading_results r on r.course_id = x.course_id and r.submission_id = x.submission_id where r.id = $1 and x.round = r.round + 1`, res)
	return r2
}

// TestRegradeKeepsLatePolicy — GD-BUG-2: chấm lại (round + 1) tính trừ muộn đúng như lần đầu: cùng số ngày muộn, cùng khoản trừ, tổng = tổng AI − khoản trừ; bài nộp đúng hạn không bị trừ.
func TestRegradeKeepsLatePolicy(t *testing.T) {
	g, text := s1Env(t)
	g.grader.Flags = guidanceFlagsOn(g, true)
	svc, _ := newGuidanceSvc(t, g, true)
	late := g.due.Add(30 * time.Hour) // trễ 2 ngày (làm tròn lên) × 0,5
	onTime := g.due.Add(-time.Hour)
	lateRes, okRes := g.submit("Muon", "bai.md", text, &late), g.submit("DungHan", "bai.md", text, &onTime)
	first := map[uuid.UUID][2]string{}
	for _, r := range []uuid.UUID{lateRes, okRes} {
		_, err := g.grader.Run(t.Context(), g.course, r)
		require.NoError(t, err)
		var pen, total string
		g.one(&pen, `select late_penalty::text from grading_results where id = $1`, r)
		g.one(&total, `select total::text from grading_results where id = $1`, r)
		first[r] = [2]string{pen, total}
	}
	require.False(t, decimalZero(first[lateRes][0]), "bài muộn có khoản trừ")
	require.True(t, decimalZero(first[okRes][0]), "đúng hạn không trừ")

	g.approveAt("c2", "Chỉ cho điểm tối đa khi có ví dụ thực tế.", 1) // bản mới → hai bài thành «bản cũ»
	out, err := svc.RegradeStale(t.Context(), grading.GuidanceActor{UserID: g.teacher}, g.course, g.asg)
	require.NoError(t, err)
	require.Equal(t, 2, out.Queued)
	for _, r := range []uuid.UUID{lateRes, okRes} {
		r2 := g.round2Of(r)
		_, err := g.grader.Run(t.Context(), g.course, r2)
		require.NoError(t, err)
		var pen, total, ai string
		var ver int
		g.one(&pen, `select late_penalty::text from grading_results where id = $1`, r2)
		g.one(&total, `select total::text from grading_results where id = $1`, r2)
		g.one(&ai, `select ai_total_before_late::text from grading_results where id = $1`, r2)
		g.one(&ver, `select guidance_version from grading_results where id = $1`, r2)
		require.Equal(t, first[r][0], pen, "khoản trừ muộn giữ nguyên")
		require.Equal(t, 1, ver)
		require.True(t, decimalEq(total, subDec(ai, pen)), "tổng = tổng AI − khoản trừ: %s − %s ≠ %s", ai, pen, total)
	}
}

// TestRegradeNeverOverwritesTeacherScores — GD-BUG-2: sau «Chấm lại» chạy xong, điểm / nhận xét giảng viên đã sửa (bài DRAFT có tiêu chí sửa, bài từng duyệt, bài đã duyệt) không đổi và không có round mới.
func TestRegradeNeverOverwritesTeacherScores(t *testing.T) {
	g, text := s1Env(t)
	g.grader.Flags = guidanceFlagsOn(g, true)
	svc, _ := newGuidanceSvc(t, g, true)
	fresh := g.submit("Moi", "bai.md", text, nil)
	_, err := g.grader.Run(t.Context(), g.course, fresh)
	require.NoError(t, err)
	// ba bài còn lại: dựng trạng thái người đã chạm vào
	edited := g.staleRes("Sua", "DRAFT", 0, 1, "1.00", false)
	reopened := g.staleRes("MoLai", "DRAFT", 0, 1, "2.00", true)
	reviewed := g.staleRes("Duyet", "REVIEWED", 0, 1, "1.00", true)
	snap := func() string {
		var s string
		g.one(&s, `select string_agg(id::text || ':' || status::text || ':' || round::text || ':' || coalesce(total::text, '') || ':' || criteria::text || ':' || coalesce(overall_feedback, ''), '|' order by id) from grading_results where course_id = $1 and id = any($2)`, g.course, []string{edited.String(), reopened.String(), reviewed.String()})
		return s
	}
	before := snap()
	g.approveAt("c2", "Chỉ cho điểm tối đa khi có ví dụ thực tế.", 1)
	out, err := svc.RegradeStale(t.Context(), grading.GuidanceActor{UserID: g.teacher}, g.course, g.asg)
	require.NoError(t, err)
	require.Equal(t, 1, out.Queued, "chỉ bài nháp chưa ai chạm")
	_, err = g.grader.Run(t.Context(), g.course, g.round2Of(fresh))
	require.NoError(t, err)
	require.Equal(t, before, snap())
	var n int
	g.one(&n, `select count(*) from grading_results where course_id = $1 and round > 1 and id = any($2)`, g.course, []string{edited.String(), reopened.String(), reviewed.String()})
	require.Zero(t, n)
}

func decimalEq(a, b string) bool {
	return decimal.RequireFromString(a).Equal(decimal.RequireFromString(b))
}
func decimalZero(a string) bool { return decimal.RequireFromString(a).IsZero() }
func subDec(a, b string) string {
	return decimal.RequireFromString(a).Sub(decimal.RequireFromString(b)).String()
}

// TestGuidanceItemReset — P-106-8 / US-GD-04 AC3a: Duyệt / Bỏ / Cho nghỉ đảo ngược được TRƯỚC khi Áp dụng (chờ áp dụng / đã bỏ → DRAFT, GIỮ chữ đã sửa; chờ nghỉ → đang dùng); DRAFT idempotent;
// mục đã vào phiên bản đã áp dụng → 409 ITEM_NOT_RESETTABLE; đua với Áp dụng: một bên thắng.
func TestGuidanceItemReset(t *testing.T) {
	g := newGradeEnv(t, nil)
	svc, _ := newGuidanceSvc(t, g, true)
	a := grading.GuidanceActor{UserID: g.teacher}
	i1, i2 := g.draftItem("c1", "Bản gốc của AI."), g.draftItem("c2", "Cần ví dụ thực tế.")
	txt := "Bản giảng viên sửa."
	v, err := svc.Resolve(t.Context(), a, g.course, g.asg, i1, "APPROVE", &txt, 1)
	require.NoError(t, err)
	v, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "RESET", nil, v.Version)
	require.NoError(t, err)
	require.Equal(t, "DRAFT", v.Status)
	require.Equal(t, txt, v.Text, "chữ đã sửa được giữ")
	require.NotNil(t, v.TextOriginal)
	require.Equal(t, "Bản gốc của AI.", *v.TextOriginal)
	var by *uuid.UUID
	g.one(&by, `select decided_by from grading_guidance_items where id = $1`, i1)
	require.Nil(t, by, "xoá người quyết")
	again, err := svc.Resolve(t.Context(), a, g.course, g.asg, i1, "RESET", nil, 1)
	require.NoError(t, err, "đang là nháp: idempotent, kể cả version cũ")
	require.Equal(t, v.Version, again.Version)
	r, err := svc.Resolve(t.Context(), a, g.course, g.asg, i2, "REJECT", nil, 1)
	require.NoError(t, err)
	r, err = svc.Resolve(t.Context(), a, g.course, g.asg, i2, "RESET", nil, r.Version)
	require.NoError(t, err)
	require.Equal(t, "DRAFT", r.Status)

	// đã áp dụng: không hoàn tác duyệt; «Cho nghỉ» thì hoàn tác được
	v, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "APPROVE", nil, v.Version)
	require.NoError(t, err)
	_, err = svc.Apply(t.Context(), a, g.course, g.asg, 0)
	require.NoError(t, err)
	_, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "RESET", nil, g.itemVersion(i1)+0)
	st, code := apiCode(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "ITEM_NOT_RESETTABLE", code, "đã vào phiên bản 1")
	v, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "RETIRE", nil, g.itemVersion(i1))
	require.NoError(t, err)
	require.True(t, v.PendingRetire)
	v, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "RESET", nil, v.Version)
	require.NoError(t, err)
	require.False(t, v.PendingRetire)
	require.Equal(t, "APPROVED", v.Status)
	_, err = svc.Apply(t.Context(), a, g.course, g.asg, 1)
	_, code = apiCode(t, err)
	require.Equal(t, "NOTHING_TO_APPLY", code, "sau hoàn tác không còn gì chờ áp dụng")

	// chờ nghỉ rồi Áp dụng → đã nghỉ: không đặt lại được
	v, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "RETIRE", nil, v.Version)
	require.NoError(t, err)
	_, err = svc.Apply(t.Context(), a, g.course, g.asg, 1)
	require.NoError(t, err)
	_, err = svc.Resolve(t.Context(), a, g.course, g.asg, i1, "RESET", nil, g.itemVersion(i1))
	_, code = apiCode(t, err)
	require.Equal(t, "ITEM_NOT_RESETTABLE", code)
}

// TestGuidanceDeleteStudent — US-GD-07 AC7 / US-GD-11: xoá sinh viên (kéo theo bài nộp + kết quả) thì ca hồi quy tham chiếu kết quả đó bị xoá (trigger gc), lưu ý đã rút không chứa danh tính nên còn nguyên.
func TestGuidanceDeleteStudent(t *testing.T) {
	g := newGradeEnv(t, nil)
	canary := "CANARY_XOA_SV"
	res := g.reviewed("Sinh Vien Bi Xoa", map[string]string{"c1": "2.00"}, map[string]string{"c1": "1.00"}, canary)
	keep := g.reviewed("Sinh Vien Con Lai", map[string]string{"c1": "2.00"}, map[string]string{"c1": "1.00"}, canary)
	for _, r := range []uuid.UUID{res, keep} {
		g.exec(`insert into grading_regression_cases (course_id, kind, source_id, criterion_id, expected) values ($1, 'GRADING_EDIT', $2, 'c1', '{"ai_score":"2.00","teacher_score":"1.00","direction":"STRICTER"}'::jsonb)`, g.course, r)
	}
	item := g.draftItem("c1", "Chấm chặt hơn khi chỉ nêu tên.")
	g.exec(`update grading_guidance_items set source_result_ids = array[$2::uuid] where id = $1`, item, res)
	var stu uuid.UUID
	g.one(&stu, `select s.student_id from grading_results r join submissions s on s.id = r.submission_id where r.id = $1`, res)
	g.exec(`delete from grading_results where submission_id in (select id from submissions where student_id = $1)`, stu) // thứ tự xoá của job xoá dữ liệu sinh viên
	g.exec(`delete from submissions where student_id = $1`, stu)
	g.exec(`delete from enrollments where user_id = $1`, stu)
	g.exec(`delete from users where id = $1`, stu)
	var n int
	g.one(&n, `select count(*) from grading_regression_cases where course_id = $1 and source_id = $2`, g.course, res)
	require.Zero(t, n, "ca của kết quả đã xoá không còn")
	g.one(&n, `select count(*) from grading_regression_cases where course_id = $1 and source_id = $2`, g.course, keep)
	require.Equal(t, 1, n, "ca của sinh viên khác giữ nguyên")
	var items string
	g.one(&items, `select string_agg(i::text, '|') from grading_guidance_items i where assignment_id = $1`, g.asg)
	require.NotContains(t, items, stu.String(), "lưu ý không chứa id sinh viên")
	require.NotContains(t, items, "Sinh Vien Bi Xoa")
}

// TestGuidanceStatsRegressionCases — P-106-10 / US-GD-07 AC6: `regression_cases` đếm ca chấm (GRADING_EDIT + GRADING_APPEAL) của ĐÚNG bài tập; không đếm ca Threads hay ca của kết quả khác; không có thì 0.
func TestGuidanceStatsRegressionCases(t *testing.T) {
	g := newGradeEnv(t, nil)
	svc, _ := newGuidanceSvc(t, g, true)
	v, err := svc.Stats(t.Context(), g.course, g.asg)
	require.NoError(t, err)
	require.Zero(t, v.RegressionCases)
	a, b := g.staleRes("RA", "REVIEWED", 0, 1, "1.00", true), g.staleRes("RB", "REVIEWED", 0, 1, "1.00", true)
	ins := func(kind string, src uuid.UUID) {
		g.exec(`insert into grading_regression_cases (course_id, kind, source_id, criterion_id, expected) values ($1, $2, $3, 'c1', '{"direction":"STRICTER"}'::jsonb)`, g.course, kind, src)
	}
	ins("GRADING_EDIT", a)
	ins("GRADING_EDIT", b)
	ins("GRADING_APPEAL", a)
	ins("THREAD_CORRECTED", uuid.New()) // ca của lớp, không thuộc bài tập
	ins("GRADING_EDIT", uuid.New())     // tham chiếu không phải kết quả của bài tập
	v, err = svc.Stats(t.Context(), g.course, g.asg)
	require.NoError(t, err)
	require.Equal(t, 3, v.RegressionCases)
}
