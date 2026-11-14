package grading_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/testutil"
)

// reviewed tạo một bài đã duyệt: điểm AI (bất biến) và điểm giảng viên chốt; canary nằm trong văn bản bài và trong nhận xét sửa tay.
func (g *gradeEnv) reviewed(name string, ai, fin map[string]string, canary string) uuid.UUID {
	g.t.Helper()
	res := g.submit(name, "bai.md", "Nội dung bài làm "+canary, nil)
	var crit []map[string]any
	for _, r := range g.rubric {
		crit = append(crit, map[string]any{"id": r.ID, "score": fin[r.ID], "comment": "SỬA TAY " + canary, "edited": fin[r.ID] != ai[r.ID], "evidence": []any{}})
	}
	cj, _ := json.Marshal(crit)
	aj, _ := json.Marshal(ai)
	cm, _ := json.Marshal(map[string]any{"criteria": map[string]string{"c1": "AI nhận xét c1", "c2": "AI nhận xét c2", "c3": "x", "c4": "y"}, "overall": "tổng"})
	g.exec(`update grading_results set run_state = 'DONE', criteria = $2, ai_scores = $3, ai_comments = $4, ai_total_before_late = 5 where id = $1`, res, string(cj), string(aj), string(cm))
	g.exec(`update grading_results set status = 'REVIEWED', reviewed_by = $2, reviewed_at = now(), first_reviewed_by = $2, first_reviewed_at = now() where id = $1`, res, g.teacher)
	g.exec(`update submissions set extracted_text = $2, extract_status = 'OK' where id = (select submission_id from grading_results where id = $1)`, res, "BÀI LÀM "+canary)
	return res
}

func (g *gradeEnv) tenReviewed(canary string) {
	for i := range 6 {
		// c1: AI chấm cao hơn giảng viên 1,0 ở mọi bài (STRICTER); c2: AI thấp hơn 0,5 ở ba bài đầu (LENIENT)
		fin := map[string]string{"c1": "1.00", "c2": "1.50", "c3": "1.00", "c4": "1.00"}
		if i < 3 {
			fin["c2"] = "2.00"
		}
		g.reviewed("SV"+string(rune('A'+i)), map[string]string{"c1": "2.00", "c2": "1.50", "c3": "1.00", "c4": "1.00"}, fin, canary)
	}
}

func guidanceFlagsOn(g *gradeEnv, on bool) *grading.GuidanceFlags {
	f := grading.NewGuidanceFlags(g.pool, nil)
	if on {
		f.Set(map[uuid.UUID][]string{g.course: {grading.FlagGradingGuidance}})
	}
	return f
}

func caseIDsFor(t *testing.T, r llm.Request, crit string) []string {
	t.Helper()
	user := r.Messages[1].Content
	start, end := strings.Index(user, "\n")+1, strings.LastIndex(user, "\n")
	var in struct {
		Cases []struct {
			CaseID      string `json:"case_id"`
			CriterionID string `json:"criterion_id"`
		} `json:"cases"`
	}
	require.NoError(t, json.Unmarshal([]byte(user[start:end]), &in))
	var ids []string
	for _, c := range in.Cases {
		if c.CriterionID == crit {
			ids = append(ids, c.CaseID)
		}
	}
	return ids
}

// TestReflectRun — job rút lưu ý đầu cuối: lời gọi LLM không chứa văn bản bài / nhận xét sửa tay (canary), support do code đếm, nháp ghi DRAFT (không tự áp), điểm AI không đổi.
func TestReflectRun(t *testing.T) {
	const canary = "CANARY_BAI_9f3a"
	var seen llm.Request
	g := newGradeEnv(t, nil)
	g.tenReviewed(canary)
	cfg := grading.DefaultGuidanceConfig()
	r := &grading.Reflector{Pool: g.pool, Flags: guidanceFlagsOn(g, true), Cfg: cfg,
		LLM: llmFunc(func(_ context.Context, req llm.Request, _ json.RawMessage) (json.RawMessage, error) {
			seen = req
			out := map[string]any{"items": []map[string]any{
				{"criterion_id": "c1", "text": "Chấm chặt hơn khi chỉ nêu tên tác nhân mà chưa phân tích.", "direction": "STRICTER", "case_ids": caseIDsFor(t, req, "c1")},
				{"criterion_id": "c2", "text": "Chấm rộng hơn khi có ví dụ thực tế.", "direction": "LENIENT", "case_ids": caseIDsFor(t, req, "c2")[:1]}, // 1 ca: support < 2
			}}
			b, _ := json.Marshal(out)
			return b, nil
		})}
	var before string
	g.one(&before, `select string_agg(ai_scores::text, '|' order by id) from grading_results where course_id = $1`, g.course)
	res, err := r.Run(t.Context(), uuid.New(), g.teacher, grading.ReflectPayload{CourseID: g.course, AssignmentID: g.asg})
	require.NoError(t, err)
	require.Equal(t, 1, res.Drafts)
	require.Equal(t, 1, res.Dropped.Support)
	for _, m := range seen.Messages {
		require.NotContains(t, m.Content, canary, "không văn bản bài / nhận xét sửa tay trong lời gọi")
	}
	require.Equal(t, llm.LaneBatch, *seen.Lane)
	require.Equal(t, llm.TaskGrading, seen.Task)
	var status, crit string
	var support int
	var ver int
	g.one(&status, `select status::text from grading_guidance_items where assignment_id = $1`, g.asg)
	g.one(&crit, `select criterion_id from grading_guidance_items where assignment_id = $1`, g.asg)
	g.one(&support, `select support from grading_guidance_items where assignment_id = $1`, g.asg)
	g.one(&ver, `select guidance_version from assignments where id = $1`, g.asg)
	require.Equal(t, "DRAFT", status)
	require.Equal(t, "c1", crit)
	require.Equal(t, 6, support)
	require.Zero(t, ver, "không lưu ý nào tự áp")
	var after string
	g.one(&after, `select string_agg(ai_scores::text, '|' order by id) from grading_results where course_id = $1`, g.course)
	require.Equal(t, before, after)

	// cờ tắt / thiếu dữ liệu → dừng có mã
	off := &grading.Reflector{Pool: g.pool, Flags: guidanceFlagsOn(g, false), Cfg: cfg, LLM: r.LLM}
	_, err = off.Run(t.Context(), uuid.New(), g.teacher, grading.ReflectPayload{CourseID: g.course, AssignmentID: g.asg})
	var ab *grading.ErrReflectAbort
	require.ErrorAs(t, err, &ab)
	require.Equal(t, "GUIDANCE_DISABLED", ab.Code)
	g2 := newGradeEnv(t, nil)
	g2.reviewed("Một", map[string]string{"c1": "2.00"}, map[string]string{"c1": "1.00"}, canary)
	few := &grading.Reflector{Pool: g2.pool, Flags: guidanceFlagsOn(g2, true), Cfg: cfg, LLM: r.LLM}
	_, err = few.Run(t.Context(), uuid.New(), g2.teacher, grading.ReflectPayload{CourseID: g2.course, AssignmentID: g2.asg})
	require.ErrorAs(t, err, &ab)
	require.Equal(t, "NOT_ENOUGH_DATA", ab.Code)
}

func newGuidanceSvc(t *testing.T, g *gradeEnv, on bool) (*grading.GuidanceService, *goredis.Client) {
	t.Helper()
	opt, err := goredis.ParseURL(testutil.RedisURL(t))
	require.NoError(t, err)
	rdb := goredis.NewClient(opt)
	t.Cleanup(func() { _ = rdb.Close() })
	return &grading.GuidanceService{Pool: g.pool, Redis: rdb, Jobs: jobs.NewService(g.pool), Flags: guidanceFlagsOn(g, on), Cfg: grading.DefaultGuidanceConfig()}, rdb
}

// TestReflectIdempotent — hai POST cùng dữ liệu → cùng job_id; trần 3 / giờ → 429; thiếu dữ liệu → 409 NOT_ENOUGH_DATA {results, edits}; cờ tắt → 409 GUIDANCE_DISABLED.
func TestReflectIdempotent(t *testing.T) {
	g := newGradeEnv(t, nil)
	svc, rdb := newGuidanceSvc(t, g, true)
	a := grading.GuidanceActor{UserID: g.teacher}

	_, err := svc.Reflect(t.Context(), a, g.course, g.asg)
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, st)
	require.Equal(t, "NOT_ENOUGH_DATA", code)
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, map[string]int{"results": 0, "edits": 0}, ae.Details)

	g.tenReviewed("x")
	j1, err := svc.Reflect(t.Context(), a, g.course, g.asg)
	require.NoError(t, err)
	j2, err := svc.Reflect(t.Context(), a, g.course, g.asg)
	require.NoError(t, err)
	require.Equal(t, j1.JobID, j2.JobID, "cùng khoá → cùng job")
	var n int
	g.one(&n, `select count(*) from jobs where kind = 'grading.reflect' and owner_id = $1`, g.teacher)
	require.Equal(t, 1, n)

	// mỗi lần duyệt thêm một bài là dữ liệu mới → việc mới; lần thứ 4 trong giờ bị trần
	for i := range 2 {
		g.reviewed("Thêm"+string(rune('A'+i)), map[string]string{"c1": "2.00"}, map[string]string{"c1": "1.00"}, "x")
		_, err = svc.Reflect(t.Context(), a, g.course, g.asg)
		require.NoError(t, err)
	}
	g.reviewed("ThêmC", map[string]string{"c1": "2.00"}, map[string]string{"c1": "1.00"}, "x")
	_, err = svc.Reflect(t.Context(), a, g.course, g.asg)
	st, code = apiCode(t, err)
	require.Equal(t, http.StatusTooManyRequests, st)
	require.Equal(t, "RATE_LIMITED", code)
	rdb.Del(t.Context(), "ep:rl:reflect:"+g.asg.String())

	off, _ := newGuidanceSvc(t, g, false)
	_, err = off.Reflect(t.Context(), a, g.course, g.asg)
	st, code = apiCode(t, err)
	require.Equal(t, http.StatusConflict, st)
	require.Equal(t, "GUIDANCE_DISABLED", code)
}

// TestGuidanceFlags — GET đủ bốn cờ; REGRESSION_CAPTURE chỉ ADMIN; version cũ → 409; hàng chưa có là version 1; công tắc cứng tắt → 409.
func TestGuidanceFlags(t *testing.T) {
	g := newGradeEnv(t, nil)
	svc, _ := newGuidanceSvc(t, g, false)
	teacher := grading.GuidanceActor{UserID: g.teacher}
	admin := grading.GuidanceActor{UserID: g.teacher, Admin: true}
	fl, err := svc.ListFlags(t.Context(), teacher, g.course)
	require.NoError(t, err)
	require.Len(t, fl, 4)
	for _, f := range fl {
		require.False(t, f.Enabled)
		require.Equal(t, 1, f.Version)
		require.Equal(t, f.Feature != grading.FlagRegressionCapture, f.CanEdit)
	}
	v, err := svc.SetFlag(t.Context(), teacher, g.course, grading.FlagGradingGuidance, true, 1)
	require.NoError(t, err)
	require.True(t, v.Enabled)
	require.Equal(t, 2, v.Version)
	require.True(t, svc.Flags.On(g.course, grading.FlagGradingGuidance), "ảnh chụp nạp lại ngay")
	_, err = svc.SetFlag(t.Context(), teacher, g.course, grading.FlagGradingGuidance, false, 1)
	st, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, st)
	require.Equal(t, "VERSION_CONFLICT", code)
	_, err = svc.SetFlag(t.Context(), teacher, g.course, grading.FlagRegressionCapture, true, 1)
	st, _ = apiCode(t, err)
	require.Equal(t, http.StatusForbidden, st)
	_, err = svc.SetFlag(t.Context(), admin, g.course, grading.FlagRegressionCapture, true, 1)
	require.NoError(t, err)
	_, err = svc.SetFlag(t.Context(), teacher, g.course, "LA", true, 1)
	st, _ = apiCode(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, st)
	var n int
	g.one(&n, `select count(*) from audit_log where action = 'guidance_flag.set' and course_id = $1`, g.course)
	require.Equal(t, 2, n)
	hard := *svc
	hard.Cfg.Enabled = false
	_, err = hard.SetFlag(t.Context(), teacher, g.course, grading.FlagThreadSuggest, true, 1)
	_, code = apiCode(t, err)
	require.Equal(t, "GUIDANCE_DISABLED", code)
}

// approveAt gắn một lưu ý đang có hiệu lực từ phiên bản v (như sau «Áp dụng»): trực tiếp bằng SQL, không qua API.
func (g *gradeEnv) approveAt(crit, text string, v int) {
	g.t.Helper()
	g.exec(`insert into grading_guidance_items (course_id, assignment_id, criterion_id, text, direction, support, status, decided_by, approved_in_version)
		values ($1, $2, $3, $4, 'STRICTER', 3, 'APPROVED', $5, $6)`, g.course, g.asg, crit, text, g.teacher, v)
	g.exec(`update assignments set guidance_version = $2 where id = $1`, g.asg, v)
}

func s1Env(t *testing.T) (*gradeEnv, string) {
	t.Helper()
	ss := samples(t)
	b := string(mustRead(t, asgFix+ss[0].File))
	return newGradeEnv(t, byPrompt(t, ss[:1], map[string]string{"S1": b})), b
}

// TestGuidancePinAndComments — cờ bật + bản v1: prompt cả hai lượt có khối «bản v1», kết quả ghim v1, ai_comments là nhận xét gốc lượt 1; cờ tắt → v0, không khối (như trước 10.6).
func TestGuidancePinAndComments(t *testing.T) {
	g, text := s1Env(t)
	g.approveAt("c2", "Chỉ cho điểm tối đa khi có ví dụ thực tế.", 1)
	g.grader.Flags = guidanceFlagsOn(g, true)
	res := g.submit("A", "bai.md", text, nil)
	_, err := g.grader.Run(t.Context(), g.course, res)
	require.NoError(t, err)
	for _, p := range g.prompts {
		require.Contains(t, p, "Lưu ý của giảng viên (bản v1)")
		require.Contains(t, p, "[c2] Chỉ cho điểm tối đa khi có ví dụ thực tế.")
	}
	var v int
	var comments string
	g.one(&v, `select guidance_version from grading_results where id = $1`, res)
	g.one(&comments, `select coalesce(ai_comments::text, '') from grading_results where id = $1`, res)
	require.Equal(t, 1, v)
	require.Contains(t, comments, `"criteria"`)
	require.Contains(t, comments, `"overall"`)
	// nhận xét gốc bất biến ở mức DB: sau DONE không đổi được
	_, err = g.pool.Exec(t.Context(), `update grading_results set ai_comments = '{}' where id = $1`, res)
	require.Error(t, err)

	// cờ tắt: v0, prompt không có khối
	g2, text2 := s1Env(t)
	g2.approveAt("c2", "Lưu ý.", 1)
	res2 := g2.submit("B", "bai.md", text2, nil)
	_, err = g2.grader.Run(t.Context(), g2.course, res2)
	require.NoError(t, err)
	for _, p := range g2.prompts {
		require.NotContains(t, p, "Lưu ý của giảng viên")
	}
	g2.one(&v, `select guidance_version from grading_results where id = $1`, res2)
	require.Zero(t, v)
}

// TestGuidancePinIsVersionUsed — TL-1 3.2: giảng viên «Áp dụng» v2 GIỮA lúc chấm: kết quả phải ghim v1 (bản đã nằm trong prompt), không phải v2.
func TestGuidancePinIsVersionUsed(t *testing.T) {
	ss := samples(t)
	text := string(mustRead(t, asgFix+ss[0].File))
	hold, release := make(chan struct{}), make(chan struct{})
	var first sync.Once
	g := newGradeEnv(t, nil)
	g.grader.LLM = llmFunc(func(ctx context.Context, r llm.Request, _ json.RawMessage) (json.RawMessage, error) {
		first.Do(func() { close(hold); <-release }) // lượt đầu chờ tới khi v2 đã áp dụng
		g.mu.Lock()
		g.prompts = append(g.prompts, r.Messages[0].Content)
		g.mu.Unlock()
		return byPrompt(t, ss[:1], map[string]string{"S1": text})(g, r)
	})
	g.approveAt("c2", "Lưu ý v1.", 1)
	g.grader.Flags = guidanceFlagsOn(g, true)
	res := g.submit("A", "bai.md", text, nil)
	done := make(chan error, 1)
	go func() { _, err := g.grader.Run(t.Context(), g.course, res); done <- err }()
	<-hold
	g.approveAt("c3", "Lưu ý v2.", 2)
	close(release)
	require.NoError(t, <-done)
	var v int
	g.one(&v, `select guidance_version from grading_results where id = $1`, res)
	require.Equal(t, 1, v)
	for _, p := range g.prompts {
		require.Contains(t, p, "(bản v1)")
		require.NotContains(t, p, "Lưu ý v2.")
	}
}
