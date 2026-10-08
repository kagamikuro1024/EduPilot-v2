package exam_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

func newLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

// ---- helper -----------------------------------------------------------------------------------------------------------------

func lockSvc(t *testing.T, r *rig) *exam.Service {
	t.Helper()
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	svc := *r.svc
	svc.Redis = rdb
	return &svc
}

func (r *rig) startVia(svc *exam.Service, e exam.ExamDetail, sv uuid.UUID) exam.AttemptStartView {
	r.t.Helper()
	v, _, err := svc.StartAttempt(r.t.Context(), sv, r.course, e.ID, uuid.New())
	require.NoError(r.t, err)
	return v
}

func lockValue(t *testing.T, svc *exam.Service, sv uuid.UUID) (string, time.Duration) {
	t.Helper()
	v, err := svc.Redis.Get(t.Context(), exam.LockKey(sv)).Result()
	if err != nil {
		return "", 0
	}
	ttl, err := svc.Redis.PTTL(t.Context(), exam.LockKey(sv)).Result()
	require.NoError(t, err)
	return v, ttl
}

// ---- AC1: khoá chat ---------------------------------------------------------------------------------------------------------

// TestLockSetOnStart — AC1: bắt đầu lượt → `ep:exam_lock:<user>` = id lượt; làm tiếp đặt lại.
func TestLockSetOnStart(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	sv := r.student("ACTIVE")
	e := r.openExam("khoá", true, r.mcqSet(2)...)
	v := r.startVia(svc, e, sv)
	got, ttl := lockValue(t, svc, sv)
	require.Equal(t, v.Attempt.ID.String(), got)
	require.Positive(t, ttl, "TTL không bao giờ −1")
	require.NoError(t, svc.Redis.Del(t.Context(), exam.LockKey(sv)).Err())
	r.startVia(svc, e, sv) // làm tiếp: SET lại
	got, _ = lockValue(t, svc, sv)
	require.Equal(t, v.Attempt.ID.String(), got)
}

// TestLockTTLEqualsRemaining — AC1: TTL = hạn − now + grace (sai số vài giây).
func TestLockTTLEqualsRemaining(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	svc.Attempt = exam.AttemptConfig{Grace: 10 * time.Second}
	sv := r.student("ACTIVE")
	e := r.openExam("ttl", true, r.mcqSet(2)...)
	v := r.startVia(svc, e, sv)
	_, ttl := lockValue(t, svc, sv)
	want := time.Until(v.Attempt.DeadlineAt) + 10*time.Second
	require.InDelta(t, want.Seconds(), ttl.Seconds(), 3)
}

// TestLockClearedOnSubmit — AC1: nộp xong, không còn lượt IN_PROGRESS → khoá bị gỡ.
func TestLockClearedOnSubmit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	sv := r.student("ACTIVE")
	e := r.openExam("nộp", true, r.mcqSet(2)...)
	tab := uuid.New()
	v, _, err := svc.StartAttempt(t.Context(), sv, r.course, e.ID, tab)
	require.NoError(t, err)
	_, err = svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	got, _ := lockValue(t, svc, sv)
	require.Empty(t, got)
}

// TestLockClearedOnClose — AC1: hết giờ / bài đóng → tick tự nộp → khoá bị gỡ.
func TestLockClearedOnClose(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	clk := clock.NewFake(time.Now().UTC())
	svc.Clock = clk
	sv := r.student("ACTIVE")
	e := r.openExam("đóng", true, r.mcqSet(2)...)
	r.startVia(svc, e, sv)
	got, _ := lockValue(t, svc, sv)
	require.NotEmpty(t, got)
	clk.Advance(46*time.Minute + 11*time.Second)
	n, err := svc.AutoSubmitDue(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	got, _ = lockValue(t, svc, sv)
	require.Empty(t, got)
}

// TestLockMultipleAttemptsLatestDeadline — AC1: nhiều lượt IN_PROGRESS (hai bài cùng giờ) → khoá lấy lượt có hạn MUỘN nhất; nộp lượt đó thì khoá chuyển sang lượt còn lại; hết cả hai thì gỡ.
func TestLockMultipleAttemptsLatestDeadline(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	sv := r.student("ACTIVE")
	short, long := r.openExam("ngắn", true, r.mcqSet(1)...), r.openExam("dài", true, r.mcqSet(1)...)
	r.exec(`update exams set duration_minutes = 30 where id=$1`, short.ID)
	r.exec(`update exams set duration_minutes = 90, closes_at = now() + interval '5 hours' where id=$1`, long.ID)
	tabA, tabB := uuid.New(), uuid.New()
	va, _, err := svc.StartAttempt(t.Context(), sv, r.course, short.ID, tabA)
	require.NoError(t, err)
	vb, _, err := svc.StartAttempt(t.Context(), sv, r.course, long.ID, tabB)
	require.NoError(t, err)
	got, _ := lockValue(t, svc, sv)
	require.Equal(t, vb.Attempt.ID.String(), got, "lượt hạn muộn nhất")
	_, err = svc.SubmitAttempt(t.Context(), sv, r.course, long.ID, vb.Attempt.ID, tabB)
	require.NoError(t, err)
	got, _ = lockValue(t, svc, sv)
	require.Equal(t, va.Attempt.ID.String(), got, "còn lượt kia → khoá chuyển sang nó")
	_, err = svc.SubmitAttempt(t.Context(), sv, r.course, short.ID, va.Attempt.ID, tabA)
	require.NoError(t, err)
	got, _ = lockValue(t, svc, sv)
	require.Empty(t, got)
}

// TestLockFollowsExtend — AC11: gia hạn bài đang mở → TTL của khoá dài ra theo hạn mới.
func TestLockFollowsExtend(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	sv := r.student("ACTIVE")
	e := r.openExam("gia hạn", true, r.mcqSet(1)...)
	r.exec(`update exams set duration_minutes = 120, opens_at = now() - interval '3 hours', closes_at = now() + interval '1 hour' where id=$1`, e.ID) // hạn của lượt bị giờ đóng chặn
	r.startVia(svc, e, sv)
	_, before := lockValue(t, svc, sv)
	_, err := svc.ExtendExam(t.Context(), r.teacher, r.course, e.ID, time.Now().Add(2*time.Hour))
	require.NoError(t, err)
	_, after := lockValue(t, svc, sv)
	require.Greater(t, after, before+30*time.Minute, "TTL dài ra theo hạn mới")
}

// ---- AC2: Locker -------------------------------------------------------------------------------------------------------------

func (r *rig) locker(svc *exam.Service) *exam.Locker {
	return &exam.Locker{Pool: r.pool, Redis: svc.Redis, Grace: 10 * time.Second}
}

// TestExamLockContract — AC2: đang làm bài → khoá; nộp xong → hết khoá; người khác không bị ảnh hưởng.
func TestExamLockContract(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	lk := r.locker(svc)
	a, b := r.student("ACTIVE"), r.student("ACTIVE")
	e := r.openExam("hợp đồng", true, r.mcqSet(1)...)
	_, locked, err := lk.IsLocked(t.Context(), a)
	require.NoError(t, err)
	require.False(t, locked)
	tab := uuid.New()
	v, _, err := svc.StartAttempt(t.Context(), a, r.course, e.ID, tab)
	require.NoError(t, err)
	got, locked, err := lk.IsLocked(t.Context(), a)
	require.NoError(t, err)
	require.True(t, locked)
	require.Equal(t, v.Attempt.ID, got.AttemptID)
	require.True(t, got.Until.After(time.Now()))
	_, locked, err = lk.IsLocked(t.Context(), b)
	require.NoError(t, err)
	require.False(t, locked, "người khác không bị khoá")
	_, err = svc.SubmitAttempt(t.Context(), a, r.course, e.ID, v.Attempt.ID, tab)
	require.NoError(t, err)
	_, locked, err = lk.IsLocked(t.Context(), a)
	require.NoError(t, err)
	require.False(t, locked)
}

// TestLockerRedisMissFallsBackDB — AC2: trượt Redis → tra DB (true) rồi nạp lại Redis.
func TestLockerRedisMissFallsBackDB(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	sv := r.student("ACTIVE")
	e := r.openExam("trượt", true, r.mcqSet(1)...)
	v := r.startVia(svc, e, sv)
	require.NoError(t, svc.Redis.Del(t.Context(), exam.LockKey(sv)).Err())
	got, locked, err := r.locker(svc).IsLocked(t.Context(), sv)
	require.NoError(t, err)
	require.True(t, locked)
	require.Equal(t, v.Attempt.ExamID, got.ExamID, "tra từ DB có ExamID")
	val, ttl := lockValue(t, svc, sv)
	require.Equal(t, v.Attempt.ID.String(), val, "đã nạp lại Redis")
	require.Positive(t, ttl)
}

// TestLockerRedisDownUsesDB — AC2 / AC11: Redis lỗi (đóng kết nối) vẫn trả đúng nhờ DB — cả khi đang khoá lẫn khi không.
func TestLockerRedisDownUsesDB(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	a, b := r.student("ACTIVE"), r.student("ACTIVE")
	e := r.openExam("redis sập", true, r.mcqSet(1)...)
	r.startVia(svc, e, a)
	dead, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	require.NoError(t, dead.Close()) // mọi lệnh sau đó lỗi
	lk := &exam.Locker{Pool: r.pool, Redis: dead, Grace: 10 * time.Second}
	_, locked, err := lk.IsLocked(t.Context(), a)
	require.NoError(t, err)
	require.True(t, locked, "chat vẫn bị chặn khi Redis sập")
	_, locked, err = lk.IsLocked(t.Context(), b)
	require.NoError(t, err)
	require.False(t, locked)
}

// TestLockerBothDownDeniesChat — AC2: Redis VÀ DB đều lỗi → trả lỗi (người gọi phải từ chối chat — an toàn khi nghi ngờ).
func TestLockerBothDownDeniesChat(t *testing.T) {
	t.Parallel()
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	pool.Close() // mọi truy vấn sau đó lỗi
	dead, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	require.NoError(t, dead.Close())
	_, locked, err := (&exam.Locker{Pool: pool, Redis: dead}).IsLocked(t.Context(), uuid.New())
	require.Error(t, err)
	require.False(t, locked)
}

// TestLockSurvivesRedisRestart — AC11: Redis mất khoá (khởi động lại) giữa giờ thi → IsLocked vẫn true nhờ DB, và nạp lại ở yêu cầu kế tiếp.
func TestLockSurvivesRedisRestart(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	sv := r.student("ACTIVE")
	e := r.openExam("khởi động lại", true, r.mcqSet(1)...)
	r.startVia(svc, e, sv)
	require.NoError(t, svc.Redis.Del(t.Context(), exam.LockKey(sv)).Err()) // như Redis vừa khởi động lại, rỗng
	_, locked, err := r.locker(svc).IsLocked(t.Context(), sv)
	require.NoError(t, err)
	require.True(t, locked)
	got, _ := lockValue(t, svc, sv)
	require.NotEmpty(t, got, "nạp lại khi có yêu cầu kế tiếp")
}

// TestLockerHitLatencyP95 — AC2: trúng cache p95 ≤ 5 ms (bản `-race`: ≤ 40 ms). Không chạy song song với test khác của gói để số đo không bị nhiễu.
func TestLockerHitLatencyP95(t *testing.T) {
	r := newRig(t)
	svc := lockSvc(t, r)
	sv := r.student("ACTIVE")
	e := r.openExam("độ trễ", true, r.mcqSet(1)...)
	r.startVia(svc, e, sv)
	lk := r.locker(svc)
	var d []time.Duration
	for range 200 {
		t0 := time.Now()
		_, locked, err := lk.IsLocked(t.Context(), sv)
		require.NoError(t, err)
		require.True(t, locked)
		d = append(d, time.Since(t0))
	}
	slices.Sort(d)
	limit := 5 * time.Millisecond
	if raceBuild {
		limit = 40 * time.Millisecond
	}
	require.LessOrEqual(t, d[len(d)*95/100], limit, "p95 = %s", d[len(d)*95/100])
}

// TestLockSweepRemovesOrphans — AC1 (≤ 10 s): khoá của lượt đã kết thúc mà `DEL` không tới được Redis bị tick quét gỡ; khoá của lượt đang làm giữ nguyên.
func TestLockSweepRemovesOrphans(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svc := lockSvc(t, r)
	live, gone := r.student("ACTIVE"), r.student("ACTIVE")
	e := r.openExam("mồ côi", true, r.mcqSet(1)...)
	r.startVia(svc, e, live)
	tab := uuid.New()
	vg, _, err := svc.StartAttempt(t.Context(), gone, r.course, e.ID, tab)
	require.NoError(t, err)
	r.exec(`update exam_attempts set status='GRADING', submitted_at=now(), submit_reason='MANUAL' where id=$1`, vg.Attempt.ID) // nộp mà không qua Service: khoá còn
	got, _ := lockValue(t, svc, gone)
	require.NotEmpty(t, got)
	n, err := r.locker(svc).Sweep(t.Context(), 100000)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)
	got, _ = lockValue(t, svc, gone)
	require.Empty(t, got)
	got, _ = lockValue(t, svc, live)
	require.NotEmpty(t, got)
}

// ---- AC3, AC10: sự kiện --------------------------------------------------------------------------------------------------------

func (r *rig) events(svc *exam.Service, sv uuid.UUID, v exam.AttemptStartView, evs ...exam.EventIn) error {
	r.t.Helper()
	return svc.RecordEvents(r.t.Context(), sv, r.course, v.Attempt.ExamID, v.Attempt.ID, exam.EventsIn{Events: evs})
}

func ev(typ string, meta string) exam.EventIn {
	e := exam.EventIn{Type: typ}
	if meta != "" {
		_ = json.Unmarshal([]byte(meta), &e.Meta)
	}
	return e
}

func (r *rig) eventMetas(attempt uuid.UUID) []string {
	r.t.Helper()
	rows, err := r.pool.Query(r.t.Context(), `select meta::text from exam_events where attempt_id=$1 order by occurred_at, id`, attempt)
	require.NoError(r.t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		require.NoError(r.t, rows.Scan(&m))
		out = append(out, m)
	}
	return out
}

func eventAttempt(t *testing.T, r *rig) (exam.AttemptStartView, uuid.UUID) {
	t.Helper()
	sv := r.student("ACTIVE")
	e := r.openExam("sự kiện", true, r.mcqSet(1)...)
	v, _ := r.start(e, sv, uuid.New())
	return v, sv
}

// TestEventsWhitelistMeta — AC3: `meta` chỉ giữ duration_ms / chars / item_id; khoá lạ (clipboard…) và giá trị sai bị bỏ; kiểu sự kiện lạ bị bỏ; không lỗi.
func TestEventsWhitelistMeta(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	v, sv := eventAttempt(t, r)
	item := v.Items[0].ItemID
	require.NoError(t, r.events(r.svc, sv, v,
		ev("PASTE", fmt.Sprintf(`{"chars":812,"clipboard":"SECRET","item_id":%q,"duration_ms":-5}`, item)),
		ev("TAB_HIDDEN", `{"duration_ms":4000,"chars":1.5,"user_agent":"x"}`),
		ev("TAB_TAKEOVER", `{"chars":1}`), // loại do máy chủ ghi: máy khách không được gửi
		ev("KEYLOG", `{"chars":1}`),
		ev("OFFLINE", ""),
	))
	got := r.eventMetas(v.Attempt.ID)
	require.Len(t, got, 3, "chỉ PASTE, TAB_HIDDEN, OFFLINE được ghi")
	require.JSONEq(t, fmt.Sprintf(`{"chars":812,"item_id":%q}`, item), got[0])
	require.JSONEq(t, `{"duration_ms":4000}`, got[1])
	require.JSONEq(t, `{}`, got[2])
	for _, m := range got {
		require.NotContains(t, m, "SECRET")
		require.NotContains(t, m, "clipboard")
		require.LessOrEqual(t, len(m), 300)
	}
}

// TestEventsNoContentStored — AC3 / AC10: bảng `exam_events` không có cột nào chứa nội dung / IP / user-agent; giờ lưu là giờ MÁY CHỦ, `client_at` chỉ tham khảo.
func TestEventsNoContentStored(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	v, sv := eventAttempt(t, r)
	client := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, r.events(r.svc, sv, v, exam.EventIn{Type: "PASTE", ClientAt: &client, Meta: map[string]json.RawMessage{"chars": json.RawMessage(`9`)}}))
	var occurred, clientAt time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select occurred_at, client_at from exam_events where attempt_id=$1`, v.Attempt.ID).Scan(&occurred, &clientAt))
	require.WithinDuration(t, time.Now(), occurred, time.Minute, "giờ máy chủ")
	require.Equal(t, client, clientAt.UTC())
}

// TestIntegrityDataMinimal — AC10: `exam_events` chỉ có các cột tối thiểu; `similarity_reports` chỉ có id lượt / bản nộp và số liệu.
func TestIntegrityDataMinimal(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	cols := func(table string) []string {
		rows, err := r.pool.Query(t.Context(), `select column_name from information_schema.columns where table_name=$1 order by column_name`, table)
		require.NoError(t, err)
		defer rows.Close()
		var out []string
		for rows.Next() {
			var c string
			require.NoError(t, rows.Scan(&c))
			out = append(out, c)
		}
		return out
	}
	require.Equal(t, []string{"attempt_id", "client_at", "course_id", "exam_id", "id", "meta", "occurred_at", "student_id", "type"}, cols("exam_events"))
	require.Equal(t, []string{"algorithm", "attempt_a", "attempt_b", "course_id", "created_at", "exam_id", "flagged", "id", "note", "problem_id", "review_state", "reviewed_at", "reviewed_by", "run_id",
		"score", "shared_fingerprints", "submission_a", "submission_b"}, cols("similarity_reports"))
}

// TestEventsCapPerAttempt — AC3: tối đa EXAM_EVENTS_MAX / lượt; vượt → nhận phần còn chỗ, bỏ phần dư, không lỗi.
func TestEventsCapPerAttempt(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	v, sv := eventAttempt(t, r)
	svc := *r.svc
	svc.Integrity = exam.IntegrityConfig{EventsMax: 5}
	for range 3 {
		require.NoError(t, r.events(&svc, sv, v, ev("TAB_HIDDEN", `{"duration_ms":1}`), ev("TAB_VISIBLE", "")))
	}
	require.Equal(t, 5, r.count(`select count(*) from exam_events where attempt_id=$1`, v.Attempt.ID))
}

// TestEventsBatchLimit / TestEventsOver50Truncated — AC3: một yêu cầu nhận 50 sự kiện ĐẦU; phần dư bị bỏ, vẫn thành công; ghi `warn` một dòng không có nội dung.
func TestEventsBatchLimit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	v, sv := eventAttempt(t, r)
	var buf bytes.Buffer
	svc := *r.svc
	svc.Log = newLogger(&buf)
	var batch []exam.EventIn
	for i := range 80 {
		batch = append(batch, ev("PASTE", fmt.Sprintf(`{"chars":%d}`, i)))
	}
	require.NoError(t, r.events(&svc, sv, v, batch...))
	require.Equal(t, 50, r.count(`select count(*) from exam_events where attempt_id=$1`, v.Attempt.ID))
	metas := r.eventMetas(v.Attempt.ID)
	require.JSONEq(t, `{"chars":0}`, metas[0])
	require.JSONEq(t, `{"chars":49}`, metas[49], "50 sự kiện đầu")
	require.Equal(t, 1, strings.Count(buf.String(), "\n"), "đúng MỘT dòng nhật ký")
	require.Contains(t, buf.String(), v.Attempt.ID.String())
	require.Contains(t, buf.String(), `"dropped":30`)
	require.NotContains(t, buf.String(), "chars", "không nội dung sự kiện")
}

// TestEventsNeverBreakSave — AC3: lượt không IN_PROGRESS → bỏ qua (nil); lượt người khác → 404; sự kiện hỏng không làm thao tác lưu bài thất bại.
func TestEventsNeverBreakSave(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	v, sv := eventAttempt(t, r)
	other := r.student("ACTIVE")
	err := r.events(r.svc, other, v, ev("PASTE", ""))
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
	require.NoError(t, r.events(r.svc, sv, v, ev("PASTE", `{"chars":"không phải số","item_id":"x"}`), ev("???", "")))
	_, err = r.save(v, sv, uuid.New()) // không ảnh hưởng việc lưu bài (tab lạ → lỗi nghiệp vụ của việc lưu, không phải của sự kiện)
	require.Error(t, err)
	require.NoError(t, r.events(r.svc, sv, v), "lô rỗng không lỗi")
	r.exec(`update exam_attempts set status='GRADING', submitted_at=now(), submit_reason='MANUAL' where id=$1`, v.Attempt.ID)
	before := r.count(`select count(*) from exam_events where attempt_id=$1`, v.Attempt.ID)
	require.NoError(t, r.events(r.svc, sv, v, ev("PASTE", `{"chars":1}`)))
	require.Equal(t, before, r.count(`select count(*) from exam_events where attempt_id=$1`, v.Attempt.ID), "sau khi nộp: bỏ qua")
}

// TestEventsCascadeDelete — AC10: xoá lượt xoá luôn sự kiện.
func TestEventsCascadeDelete(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	v, sv := eventAttempt(t, r)
	require.NoError(t, r.events(r.svc, sv, v, ev("PASTE", `{"chars":3}`)))
	require.Equal(t, 1, r.count(`select count(*) from exam_events where attempt_id=$1`, v.Attempt.ID))
	r.exec(`delete from exam_attempts where id=$1`, v.Attempt.ID)
	require.Zero(t, r.count(`select count(*) from exam_events where attempt_id=$1`, v.Attempt.ID))
}

// TestEventsSummaryAndList — AC4: Giảng viên đọc tóm tắt (số lần rời tab, tổng thời gian, dán, ký tự, offline, takeover) và danh sách mới nhất trước; lượt không thuộc bài → 404.
func TestEventsSummaryAndList(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	v, sv := eventAttempt(t, r)
	require.NoError(t, r.events(r.svc, sv, v,
		ev("TAB_HIDDEN", `{"duration_ms":3000}`), ev("TAB_VISIBLE", ""), ev("TAB_HIDDEN", `{"duration_ms":2000}`),
		ev("PASTE", `{"chars":100}`), ev("PASTE", `{"chars":50}`), ev("OFFLINE", "")))
	_, err := r.svc.Takeover(t.Context(), sv, r.course, v.Attempt.ExamID, v.Attempt.ID, uuid.New(), false)
	require.NoError(t, err)
	page, err := r.svc.ListEvents(t.Context(), r.course, v.Attempt.ExamID, v.Attempt.ID, nil, 51)
	require.NoError(t, err)
	require.Equal(t, exam.IntegritySummary{TabHiddenCount: 2, TabHiddenMS: 5000, PasteCount: 2, PasteChars: 150, OfflineCount: 1, TakeoverCount: 1}, page.Summary)
	require.Len(t, page.Items, 7)
	for i := 1; i < len(page.Items); i++ {
		require.False(t, page.Items[i].OccurredAt.After(page.Items[i-1].OccurredAt), "mới nhất trước")
	}
	_, err = r.svc.ListEvents(t.Context(), r.course, uuid.New(), v.Attempt.ID, nil, 51)
	st, _ := apiStatus(t, err)
	require.Equal(t, 404, st)
}

// ---- AC5: không tự trừ điểm ----------------------------------------------------------------------------------------------------

// TestScoreIgnoresIntegritySignals — AC5: `score*.go` không import / không nhắc tới hai bảng liêm chính (go/parser); lượt có 100 sự kiện và cặp nghi giống nhận CÙNG điểm như lượt sạch.
func TestScoreIgnoresIntegritySignals(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("score*.go")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, im := range af.Imports {
			require.NotContains(t, im.Path.Value, "similarity", f)
		}
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, bad := range []string{"exam_events", "similarity_reports", "ExamEvent", "Similarity"} {
			require.NotContains(t, string(raw), bad, f)
		}
	}
	r := newRig(t)
	qs := r.mcqSet(2)
	e := r.openExam("điểm", false, qs...)
	clean, noisy := r.student("ACTIVE"), r.student("ACTIVE")
	score := func(sv uuid.UUID, noise bool) string {
		tab := uuid.New()
		v, _ := r.start(e, sv, tab)
		if noise {
			var batch []exam.EventIn
			for range 50 {
				batch = append(batch, ev("TAB_HIDDEN", `{"duration_ms":9000}`), ev("PASTE", `{"chars":900}`))
			}
			require.NoError(t, r.events(r.svc, sv, v, batch...))
			require.NoError(t, r.events(r.svc, sv, v, batch...))
		}
		_, err := r.save(v, sv, tab, answerCorrect(v.Items[0]), pick(v.Items[1], 0))
		require.NoError(t, err)
		_, err = r.svc.SubmitAttempt(t.Context(), sv, r.course, e.ID, v.Attempt.ID, tab)
		require.NoError(t, err)
		_, _, sc, _ := r.attemptRow(v.Attempt.ID)
		return *sc
	}
	a := score(clean, false)
	b := score(noisy, true)
	require.Equal(t, a, b, "100 sự kiện rời tab / dán không đổi điểm")
	require.Equal(t, 100, r.count(`select count(*) from exam_events where exam_id=$1`, e.ID))
}

// TestNoPenaltyColumns — AC5: không bảng nào của bài thi có cột "điểm trừ" / cờ "gian lận".
func TestNoPenaltyColumns(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	n := r.count(`select count(*) from information_schema.columns where table_schema='public'
		and table_name in ('exams','exam_items','exam_attempts','exam_answers','exam_events','code_submissions','code_drafts','similarity_reports','exam_appeals')
		and column_name ~* '(penalt|deduct|cheat|fraud|plagiar|violation|trừ)'`)
	require.Zero(t, n)
}

// TestNoAccusatoryCopy — AC5: giao diện không dùng "gian lận", "vi phạm", "nghi gian lận" (grep `frontend/src/features/exam` và `i18n/vi.ts`).
func TestNoAccusatoryCopy(t *testing.T) {
	t.Parallel()
	re := regexp.MustCompile(`(?i)gian\s+lận|vi\s+phạm`)
	roots := []string{"../../../frontend/src/features/exam", "../../../frontend/src/shared/i18n/vi.ts"}
	for _, root := range roots {
		require.NoError(t, filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			require.NoError(t, err)
			if d.IsDir() || (!strings.HasSuffix(p, ".ts") && !strings.HasSuffix(p, ".tsx")) {
				return nil
			}
			raw, rerr := os.ReadFile(p)
			require.NoError(t, rerr)
			require.False(t, re.Match(raw), "%s dùng từ buộc tội", p)
			return nil
		}))
	}
}
