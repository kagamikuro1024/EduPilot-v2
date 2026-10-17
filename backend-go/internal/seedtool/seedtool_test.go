package seedtool_test

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/crm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/seedtool"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

var ict = time.FixedZone("Asia/Ho_Chi_Minh", 7*3600) //nolint:gochecknoglobals // hằng

// today: thứ Năm 29/10/2026 — buổi 10 là hôm nay, buổi 9 (22/10) nằm trong tuần trọn gần nhất.
var today = time.Date(2026, 10, 29, 10, 0, 0, 0, ict) //nolint:gochecknoglobals // hằng

type plan struct {
	Attendance map[string][]string `json:"attendance"`
	Speak      map[string]int      `json:"speak"`
	Totals     struct {
		Absent int `json:"absent"`
	} `json:"totals"`
}

type world struct {
	pool    *pgxpool.Pool
	course  uuid.UUID
	teacher uuid.UUID
	ids     map[string]uuid.UUID // khoá seed (sv.gioi, sv12…) → user id
	ses     []uuid.UUID          // buổi 1..15
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, v))
}

// newWorld dựng lớp 761987 với 30 sinh viên đúng khoá / email của `scripts/seed.mjs`, 15 buổi, buổi n cách hôm nay 7·(10−n) ngày.
func newWorld(t *testing.T, p plan) world {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	w := world{pool: pool, ids: map[string]uuid.UUID{}}
	q := store.New(pool)
	hash, err := auth.HashPassword("Mat-khau-thu-2026", 4)
	require.NoError(t, err)
	mk := func(email, name string, role store.UserRole) uuid.UUID {
		u, err := q.InsertUser(t.Context(), store.InsertUserParams{Email: email, FullName: name, Role: role, Status: store.UserStatusACTIVE, PasswordHash: &hash})
		require.NoError(t, err)
		return u.ID
	}
	w.teacher = mk("teacher@edupilot.local", "Giảng viên", store.UserRoleTEACHER)
	require.NoError(t, pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', '761987', 'An ninh mạng', '2026-2027-HK1', 'AN7K2MQ', $1) returning id`, w.teacher).Scan(&w.course))
	_, err = pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, w.course, w.teacher)
	require.NoError(t, err)
	for k := range p.Attendance {
		w.ids[k] = mk(k+"@edupilot.local", k, store.UserRoleSTUDENT)
		_, err := pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, w.course, w.ids[k])
		require.NoError(t, err)
	}
	for n := 1; n <= 15; n++ {
		start := time.Date(today.Year(), today.Month(), today.Day(), 9, 0, 0, 0, ict).AddDate(0, 0, -7*(10-n))
		var id uuid.UUID
		require.NoError(t, pool.QueryRow(t.Context(), `insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, $2, $3, $4) returning id`, w.course, n, start, start.Add(150*time.Minute)).Scan(&id))
		w.ses = append(w.ses, id)
	}
	for k, row := range p.Attendance {
		for i, st := range row {
			_, err := pool.Exec(t.Context(), `insert into attendance_records (course_id, session_id, student_id, status, marked_by) values ($1, $2, $3, $4::attendance_status, $5)`, w.course, w.ses[i], w.ids[k], st, w.teacher)
			require.NoError(t, err)
		}
		for i := range p.Speak[k] {
			_, err := pool.Exec(t.Context(), `insert into participation_events (course_id, student_id, session_id, type, points, client_event_id, created_by) values ($1, $2, $3, 'SPEAK', 0.25, $4, $5)`, w.course, w.ids[k], w.ses[i%9], uuid.New(), w.teacher)
			require.NoError(t, err)
		}
	}
	return w
}

func levels(t *testing.T, w world) map[string]string {
	t.Helper()
	rs, err := w.pool.Query(t.Context(), `select u.email, r.level::text from student_risk r join users u on u.id = r.student_id where r.course_id = $1`, w.course)
	require.NoError(t, err)
	defer rs.Close()
	out := map[string]string{}
	for rs.Next() {
		var e, l string
		require.NoError(t, rs.Scan(&e, &l))
		out[e] = l
	}
	return out
}

// TestSeedCRMExpected (US-S7-SEED AC1–AC3): kế hoạch `seed/crm_plan.json` + hồ sơ `seed/study_profile.json` cho đúng số liệu mục tiêu của SRS 4.10.1 và kết quả `crm.risk`.
func TestSeedCRMExpected(t *testing.T) {
	var p plan
	readJSON(t, "../../../seed/crm_plan.json", &p)
	require.Len(t, p.Attendance, 30)
	w := newWorld(t, p)
	prof, err := os.ReadFile("../../../seed/study_profile.json")
	require.NoError(t, err)
	profile, err := seedtool.ParseProfile(prof)
	require.NoError(t, err)

	// AC1: 270 dòng, buổi 10 chưa có, ABSENT cả lớp 18–26, khớp expected_crm.csv
	require.Equal(t, 270, count(t, w, `select count(*) from attendance_records where course_id = $1`))
	require.Zero(t, count(t, w, `select count(*) from attendance_records ar join class_sessions s on s.id = ar.session_id where s.course_id = $1 and s.session_no >= 10`))
	require.GreaterOrEqual(t, p.Totals.Absent, 18)
	require.LessOrEqual(t, p.Totals.Absent, 26)
	require.Equal(t, p.Totals.Absent, count(t, w, `select count(*) from attendance_records where course_id = $1 and status = 'ABSENT'`))
	f, err := os.Open("../../../seed/expected_crm.csv")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	rows, err := csv.NewReader(f).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 4)
	for _, r := range rows[1:] {
		var got [7]int
		var pts string
		require.NoError(t, w.pool.QueryRow(t.Context(), `select a.recorded, a.present, a.late, a.excused, a.absent,
			(select count(*) filter (where type = 'SPEAK') from participation_events e where e.course_id = a.course_id and e.student_id = a.student_id)::int,
			(select coalesce(sum(points), 0)::numeric(8,2)::text from participation_events e where e.course_id = a.course_id and e.student_id = a.student_id)
			from v_attendance_summary a where a.course_id = $1 and a.student_id = $2`, w.course, w.ids[r[0]]).Scan(&got[0], &got[1], &got[2], &got[3], &got[4], &got[5], &pts), r[0])
		for i := range 6 {
			want, err := strconv.Atoi(r[i+1])
			require.NoError(t, err)
			require.Equal(t, want, got[i], fmt.Sprintf("%s cột %d", r[0], i+1))
		}
		require.Equal(t, r[7], pts, r[0])
	}
	// B đúng hai sự kiện SPEAK +0,50 (SRS 4.10.1)
	require.Equal(t, 2, p.Speak["sv.kha"])

	// AC2: thời gian học 10 tuần, ≥ 60 ngày khác nhau, chạy lại không đổi dòng nào
	n, err := seedtool.SeedStudyTime(t.Context(), w.pool, w.course, 10, profile, today)
	require.NoError(t, err)
	require.Positive(t, n)
	require.GreaterOrEqual(t, count(t, w, `select count(distinct date) from study_time_daily where course_id = $1`), 60)
	again, err := seedtool.SeedStudyTime(t.Context(), w.pool, w.course, 10, profile, today)
	require.NoError(t, err)
	require.Zero(t, again, "chạy hai lần không nhân đôi / đổi dòng")
	weekMin := func(key string, weekNo int) int { // tuần weekNo (1..10; 10 = hiện tại)
		monday := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -7*(10-weekNo))
		return count(t, w, `select (coalesce(sum(active_seconds), 0) / 60)::int from study_time_daily where course_id = $1 and student_id = $2 and date >= $3 and date < $4`, w.course, w.ids[key], monday, monday.AddDate(0, 0, 7))
	}
	require.Equal(t, 300, weekMin("sv.gioi", 5))
	require.Equal(t, 150, weekMin("sv.kha", 5))
	for _, wk := range []int{7, 8, 9} {
		require.InDelta(t, 30, weekMin("sv.nguyco", wk), 10, "C tuần %d ∈ [20, 40]", wk)
		require.GreaterOrEqual(t, weekMin("sv.nguyco", wk), 20)
		require.LessOrEqual(t, weekMin("sv.nguyco", wk), 40)
	}
	require.Equal(t, 120, weekMin("sv.nguyco", 3))
	require.Less(t, weekMin("sv12", 9), 60)
	require.Less(t, weekMin("sv19", 9), 60)

	// AC3: crm.risk → đúng một AT_RISK (C), đúng hai WATCH (sv12, sv19), A và B NONE
	svc := &crm.Service{Pool: w.pool, Clock: clock.NewFake(today)}
	require.NoError(t, svc.RunCourse(t.Context(), w.course, today))
	got := levels(t, w)
	require.Zero(t, count(t, w, `select count(*) from student_risk r, jsonb_array_elements(r.weeks) wk where r.course_id = $1 and wk->'study' is null`), "mọi sinh viên có tín hiệu STUDY ở cả 4 tuần (đã xác nhận thông báo từ trước tuần 1)")
	var atRisk, watch []string
	for e, l := range got {
		switch l {
		case "AT_RISK":
			atRisk = append(atRisk, e)
		case "WATCH":
			watch = append(watch, e)
		}
	}
	require.ElementsMatch(t, []string{"sv.nguyco@edupilot.local"}, atRisk)
	require.ElementsMatch(t, []string{"sv12@edupilot.local", "sv19@edupilot.local"}, watch)
	require.Equal(t, "NONE", got["sv.gioi@edupilot.local"])
	require.Equal(t, "NONE", got["sv.kha@edupilot.local"])
}

func count(t *testing.T, w world, sql string, args ...any) int {
	t.Helper()
	if len(args) == 0 {
		args = []any{w.course}
	}
	var n int
	require.NoError(t, w.pool.QueryRow(t.Context(), sql, args...).Scan(&n), sql)
	return n
}

// TestSeedChatNoTickets (BUG-S1): lịch sử chat ghi thẳng KHÔNG tạo ticket nào (126 ticket tự mở khi đi qua luồng chat thật với provider giả), chạy lại không thêm gì,
// tin hỏi không bị gán độ tin cậy, câu ngoài tài liệu có no_context, mọi mốc ≤ hôm nay.
func TestSeedChatNoTickets(t *testing.T) {
	var p plan
	readJSON(t, "../../../seed/crm_plan.json", &p)
	w := newWorld(t, p)
	cp := seedtool.ChatPlan{Sessions: []seedtool.ChatSession{
		{Email: "sv.gioi@edupilot.local", Key: "s-gioi-0", Turns: []seedtool.ChatTurn{{Key: "t1", Question: "Phishing là gì?", Answer: "Mẫu."}, {Key: "t2", Question: "Hôm nay thời tiết thế nào?", Answer: "Chưa tìm thấy.", Outside: true}}},
		{Email: "sv.kha@edupilot.local", Key: "s-kha-0", Turns: []seedtool.ChatTurn{{Key: "t3", Question: "Điều kiện dự thi?", Answer: "Mẫu."}}},
	}}
	n, err := seedtool.SeedChat(t.Context(), w.pool, w.course, 10, cp, today)
	require.NoError(t, err)
	require.EqualValues(t, 6, n)
	again, err := seedtool.SeedChat(t.Context(), w.pool, w.course, 10, cp, today)
	require.NoError(t, err)
	require.Zero(t, again)
	require.Zero(t, count(t, w, `select count(*) from escalation_tickets where course_id = $1`))
	require.Equal(t, 2, count(t, w, `select count(*) from chat_sessions where course_id = $1`))
	require.Equal(t, 3, count(t, w, `select count(*) from chat_messages where course_id = $1 and role = 'USER' and confidence is null`))
	require.Equal(t, 1, count(t, w, `select count(*) from chat_messages where course_id = $1 and role = 'ASSISTANT' and no_context and low_confidence`))
	require.Zero(t, count(t, w, `select count(*) from chat_messages where course_id = $1 and (created_at > $2 or intent is distinct from case role when 'ASSISTANT' then 'COURSE_QA' end)`, w.course, today))
	require.Error(t, mustErr(seedtool.SeedChat(t.Context(), w.pool, w.course, 10, seedtool.ChatPlan{Sessions: []seedtool.ChatSession{{Email: "khong.co@edupilot.local", Key: "x", Turns: []seedtool.ChatTurn{{Key: "y", Question: "q", Answer: "a"}}}}}, today)))
}

func mustErr(_ int64, err error) error { return err }
