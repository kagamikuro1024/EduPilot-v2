package crm_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/crm"
)

// bigData dựng dữ liệu giả lập của SRS 9.4: 1 lớp 1.000 sinh viên + 19 lớp 100 sinh viên, 15 buổi mỗi lớp, ≥ 280.000 dòng study_time_daily, ≥ 5.000 participation_events; ANALYZE.
// Trả lớp lớn và một lớp nhỏ.
func (r *rig) bigData(t *testing.T) (big, small uuid.UUID) {
	t.Helper()
	adm := r.addUser(uniq("adm"), "ADMIN", "ACTIVE")
	tag := uuid.NewString()[:6]
	var courses []uuid.UUID
	for i := range 20 {
		courses = append(courses, r.course(adm.ID, "P"+tag+string(rune('A'+i))))
	}
	r.exec(`insert into users (email, full_name, role, status)
		select 'perf' || g || '.' || $1::text || '@example.test', 'Sv ' || lpad(g::text, 4, '0'), 'STUDENT', 'INVITED' from generate_series(1, 2900) g`, tag)
	r.exec(`create temp table perf_u as select id, row_number() over (order by email) n from users where email like 'perf%.' || $1::text || '@example.test'`, tag)
	// lớp 0: 1.000 người đầu; lớp k (1..19): 100 người kế tiếp
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot)
		select (array(select unnest($1::uuid[])))[case when n <= 1000 then 1 else 2 + (n - 1001) / 100 end], id, 'STUDENT', 'ACTIVE', 'ADMIN', 'B21' || lpad(n::text, 6, '0') from perf_u`, courses)
	r.exec(`insert into class_sessions (course_id, session_no, starts_at, ends_at)
		select c, s, now() - make_interval(days => s * 2), now() - make_interval(days => s * 2) + interval '2 hours' from unnest($1::uuid[]) c, generate_series(1, 15) s`, courses)
	r.exec(`insert into attendance_records (course_id, session_id, student_id, status, marked_by)
		select e.course_id, s.id, e.user_id, (array['PRESENT','PRESENT','PRESENT','LATE','EXCUSED','ABSENT'])[1 + (abs(hashtext(e.user_id::text || s.session_no::text)) % 6)]::attendance_status, $1
		from enrollments e join class_sessions s on s.course_id = e.course_id where e.role_in_course = 'STUDENT' and e.course_id = any($2::uuid[])`, adm.ID, courses)
	r.exec(`insert into study_time_daily (course_id, student_id, date, area, active_seconds)
		select e.course_id, e.user_id, current_date - d, 'CHAT', 600 + (abs(hashtext(e.user_id::text)) % 3000) from enrollments e, generate_series(0, 99) d where e.role_in_course = 'STUDENT' and e.course_id = any($1::uuid[])`, courses)
	r.exec(`insert into participation_events (course_id, student_id, type, points, client_event_id, created_by)
		select e.course_id, e.user_id, 'SPEAK', 0.25, gen_random_uuid(), $1 from enrollments e, generate_series(1, 2) g where e.role_in_course = 'STUDENT' and e.course_id = any($2::uuid[])`, adm.ID, courses)
	r.exec(`insert into student_risk (course_id, student_id, level, entered_at, computed_at)
		select course_id, user_id, case when abs(hashtext(user_id::text)) % 10 = 0 then 'AT_RISK' else 'WATCH' end::risk_level, case when abs(hashtext(user_id::text)) % 10 = 0 then now() end, now()
		from enrollments where role_in_course = 'STUDENT' and course_id = any($1::uuid[]) and abs(hashtext(user_id::text)) % 4 = 0`, courses)
	for _, tb := range []string{"enrollments", "attendance_records", "study_time_daily", "participation_events", "student_risk", "class_sessions", "users"} {
		r.exec(`analyze ` + tb)
	}
	require.GreaterOrEqual(t, r.count(`select count(*) from study_time_daily where course_id = any($1::uuid[])`, courses), 280000)
	require.GreaterOrEqual(t, r.count(`select count(*) from participation_events where course_id = any($1::uuid[])`, courses), 5000)
	return courses[0], courses[1]
}

type planNode struct {
	NodeType string     `json:"Node Type"`
	Relation string     `json:"Relation Name"`
	Plans    []planNode `json:"Plans"`
}

// scans gom (loại nút, bảng) của mọi nút quét trong kế hoạch EXPLAIN.
func scans(n planNode, out *[][2]string) {
	if n.Relation != "" {
		*out = append(*out, [2]string{n.NodeType, n.Relation})
	}
	for _, c := range n.Plans {
		scans(c, out)
	}
}

// explain chạy EXPLAIN (FORMAT JSON) cho đúng câu SQL + tham số vừa được tracer ghi lại.
func (r *rig) explain(name string) [][2]string {
	r.t.Helper()
	r.qc.mu.Lock()
	c, ok := r.qc.last[name]
	r.qc.mu.Unlock()
	require.True(r.t, ok, "chưa có câu %s", name)
	var raw []byte
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), "EXPLAIN (FORMAT JSON) "+c.sql, c.args...).Scan(&raw))
	var p []struct {
		Plan planNode `json:"Plan"`
	}
	require.NoError(r.t, json.Unmarshal(raw, &p))
	require.NotEmpty(r.t, p)
	var out [][2]string
	scans(p[0].Plan, &out)
	require.NotContains(r.t, string(raw), "chat_messages", "danh sách không chạm chat_messages")
	return out
}

var bigTables = []string{"attendance_records", "study_time_daily", "participation_events", "student_risk", "student_observations"} //nolint:gochecknoglobals // tra cứu bất biến

// TestStudents360ExplainNoSeqScan — AC3(a): lớp 100 sinh viên (≤ 5 % mỗi bảng lớn): trang đầu, trang giữa, filter=attention, q= không có Seq Scan trên năm bảng lớn.
func TestStudents360ExplainNoSeqScan(t *testing.T) {
	r := newRig(t)
	_, small := r.bigData(t)
	svc := r.svc()
	first, err := svc.ListStudents(t.Context(), small, crm.StudentsQuery{Limit: 20})
	require.NoError(t, err)
	require.NotNil(t, first.NextCursor)
	cur, err := crm.ParseStudentCursor(*first.NextCursor)
	require.NoError(t, err)
	for name, q := range map[string]crm.StudentsQuery{
		"đầu":       {Limit: 20},
		"giữa":      {Limit: 20, Cursor: &cur},
		"attention": {Limit: 20, Attention: true},
		"q":         {Limit: 20, Q: "sv 00"},
	} {
		_, err := svc.ListStudents(t.Context(), small, q)
		require.NoError(t, err, name)
		touched := 0
		for _, sn := range r.explain("CrmStudentsList") {
			if slices.Contains(bigTables, sn[1]) {
				touched++
				if sn[0] == "Seq Scan" {
					t.Errorf("%s: Seq Scan trên %s", name, sn[1])
				}
			}
		}
		require.Positive(t, touched, "%s: kế hoạch phải chạm các bảng lớn (kiểm EXPLAIN đúng câu)", name)
	}
}

// TestStudents360LargeCoursePerf + TestStudentsListP95Under300ms — AC3(b): lớp 1.000 sinh viên: trang đầu p95 ≤ 300 ms (SLO đọc).
func TestStudentsListP95Under300ms(t *testing.T) {
	r := newRig(t)
	big, _ := r.bigData(t)
	svc := r.svc()
	var d []time.Duration
	for range 30 {
		t0 := time.Now()
		page, err := svc.ListStudents(t.Context(), big, crm.StudentsQuery{Limit: 50})
		require.NoError(t, err)
		require.Equal(t, 1000, page.Total)
		require.Len(t, page.Items, 50)
		d = append(d, time.Since(t0))
	}
	slices.Sort(d)
	p95 := d[len(d)*95/100]
	t.Logf("trang đầu lớp 1.000 SV: p50 %v, p95 %v", d[len(d)/2], p95)
	require.LessOrEqual(t, p95, 300*time.Millisecond)
}

// TestStudents360LargeCoursePerf — AC3(b): lớp 1.000 sinh viên: truy vấn 360 của MỘT sinh viên (độ chọn lọc 1/1.000) đi bằng chỉ mục, không Seq Scan trên bảng lớn, và nhanh (≪ 3 s).
// Hồ sơ đầy đủ (route + audit) thuộc US-P5-05 (`TestProfileUnder3s`); ở đây kiểm đúng truy vấn đọc `v_student_360` mà hồ sơ dùng.
func TestStudents360LargeCoursePerf(t *testing.T) {
	r := newRig(t)
	big, _ := r.bigData(t)
	var student uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `select student_id from v_student_360 where course_id = $1 order by student_id limit 1 offset 500`, big).Scan(&student))
	q := `select * from v_student_360 where course_id = $1 and student_id = $2`
	var raw []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), "EXPLAIN (FORMAT JSON) "+q, big, student).Scan(&raw))
	var p []struct {
		Plan planNode `json:"Plan"`
	}
	require.NoError(t, json.Unmarshal(raw, &p))
	var sc [][2]string
	scans(p[0].Plan, &sc)
	touched := 0
	for _, sn := range sc {
		if slices.Contains(bigTables, sn[1]) {
			touched++
			require.NotEqual(t, "Seq Scan", sn[0], "%s: truy vấn một sinh viên phải dùng chỉ mục", sn[1])
		}
	}
	require.Positive(t, touched)
	t0 := time.Now()
	var n int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from (`+q+`) x`, big, student).Scan(&n))
	require.Equal(t, 1, n)
	require.Less(t, time.Since(t0), 3*time.Second)
}
