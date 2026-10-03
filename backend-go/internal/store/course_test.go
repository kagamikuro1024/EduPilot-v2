package store_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

// TestCourseSchema — US-P2-07 AC1: 8 bảng của 00003 đủ cột / kiểu / chỉ mục, bảng thuộc lớp có chỉ mục phức hợp bắt đầu bằng course_id.
func TestCourseSchema(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from information_schema.tables where table_schema='public' and table_name in
		('courses','enrollments','class_sessions','notifications','user_settings','documents','content_chunks','document_courses')`).Scan(&n))
	require.Equal(t, 8, n)

	wantCols := map[string][]string{
		"courses":          {"id", "subject_code", "class_code", "name", "semester", "status", "escalation_threshold", "settings", "join_code", "join_enabled", "join_expires_at", "join_require_approval", "allowed_email_domain", "capacity", "created_by", "archived_at", "version", "created_at", "updated_at"},
		"enrollments":      {"id", "course_id", "user_id", "role_in_course", "status", "joined_via", "student_code_snapshot", "warning", "previous_status", "status_changed_at", "status_changed_by", "removed_at", "version", "created_at", "updated_at"},
		"class_sessions":   {"id", "course_id", "session_no", "starts_at", "ends_at", "room", "topic", "version", "created_at", "updated_at"},
		"notifications":    {"id", "user_id", "course_id", "type", "title", "body", "link", "dedupe_key", "read_at", "created_at", "updated_at"},
		"user_settings":    {"user_id", "notify_ticket_by_mail", "notify_answer_by_mail", "remind_deadline_by_mail", "preferences", "version", "created_at", "updated_at"},
		"documents":        {"id", "course_id", "title", "type", "filename", "mime_type", "size_bytes", "sha256", "blob_key", "status", "error", "page_count", "visible_to_students", "use_for_rag", "category", "week_no", "download_count", "uploaded_by", "version", "created_at", "updated_at"},
		"content_chunks":   {"id", "document_id", "course_ids", "audience", "ord", "page_no", "heading", "text", "token_count", "embedding", "created_at", "updated_at"},
		"document_courses": {"document_id", "course_id", "shared_by", "created_at"},
	}
	for table, want := range wantCols {
		rows, err := conn.Query(ctx, `select column_name from information_schema.columns where table_schema='public' and table_name=$1 order by ordinal_position`, table)
		require.NoError(t, err)
		var got []string
		for rows.Next() {
			var c string
			require.NoError(t, rows.Scan(&c))
			got = append(got, c)
		}
		require.NoError(t, rows.Err())
		require.ElementsMatch(t, want, got, table)
	}

	typ := func(table, col string) string {
		var s string
		require.NoError(t, conn.QueryRow(ctx, `select format_type(a.atttypid, a.atttypmod) from pg_attribute a where a.attrelid = ('public.'||$1)::regclass and a.attname = $2`, table, col).Scan(&s))
		return s
	}
	require.Equal(t, "uuid[]", typ("content_chunks", "course_ids"))
	require.Equal(t, "vector(1536)", typ("content_chunks", "embedding"))
	require.Equal(t, "chunk_audience", typ("content_chunks", "audience"))
	require.Equal(t, "character(7)", typ("courses", "join_code"))
	require.Equal(t, "text", typ("documents", "blob_key"))
	require.Equal(t, "boolean", typ("documents", "visible_to_students"))
	require.Equal(t, "boolean", typ("documents", "use_for_rag"))
	require.Equal(t, "smallint", typ("documents", "week_no"))
	require.Equal(t, "integer", typ("documents", "download_count"))

	var gin int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_indexes where tablename='content_chunks' and indexdef ilike '%gin%course_ids%'`).Scan(&gin))
	require.Equal(t, 1, gin)

	// bảng thuộc lớp: mọi chỉ mục phức hợp (trừ khoá chính / duy nhất theo tài liệu) bắt đầu bằng course_id
	rows, err := conn.Query(ctx, `select tablename, indexname, indexdef from pg_indexes where schemaname='public'
		and tablename in ('enrollments','class_sessions','documents','document_courses') and indexname not like '%_pkey'`)
	require.NoError(t, err)
	for rows.Next() {
		var table, name, def string
		require.NoError(t, rows.Scan(&table, &name, &def))
		inside := def[strings.Index(def, "(")+1 : strings.Index(def, ")")]
		if strings.Contains(inside, ",") && name != "enrollments_user_idx" {
			require.True(t, strings.HasPrefix(inside, "course_id"), "%s: chỉ mục phức hợp phải bắt đầu bằng course_id: %s", name, def)
		}
	}
	require.NoError(t, rows.Err())
}

// TestCourseConstraints — US-P2-07 AC2: dữ liệu sai bị DB từ chối đúng SQLSTATE (≥ 20 ca).
func TestCourseConstraints(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	mustExec := func(sql string, args ...any) {
		t.Helper()
		_, err := conn.Exec(ctx, sql, args...)
		require.NoError(t, err, sql)
	}
	var admin, t1, t2, sv, ta, c1 uuid.UUID //nolint:wsl
	user := func(email, role string) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, conn.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'T', $2::user_role) returning id`, email+"."+uuid.NewString()[:6]+"@example.test", role).Scan(&id))
		return id
	}
	admin, t1, t2, sv, ta = user("ad", "ADMIN"), user("t1", "TEACHER"), user("t2", "TEACHER"), user("sv", "STUDENT"), user("ta", "TA")
	suffix := uuid.NewString()[:8]
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := func(seed int) string {
		b := make([]byte, 7)
		x := seed
		for i := range b {
			b[i] = alpha[x%len(alpha)]
			x = x/len(alpha) + 7*i + seed
		}
		return string(b)
	}
	newCourse := func(cls string, join string) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, conn.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`, cls, join, admin).Scan(&id))
		return id
	}
	base := int(uuid.New().ID() % 100000)
	c1 = newCourse("LOP-"+suffix+"A", jc(base+1))
	mustExec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, c1, t1)
	mustExec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'CODE')`, c1, sv)
	mustExec(`insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, 1, '2026-10-05 08:00+07', '2026-10-05 10:00+07')`, c1)

	const (
		check  = "23514"
		unique = "23505"
	)
	ins := func(cls, join, extraCols, extraVals string) string {
		return fmt.Sprintf(`insert into courses (subject_code, class_code, name, semester, join_code, created_by%s) values ('INT1006', '%s', 'Lớp', '2026-2027-HK1', '%s', '%s'%s)`, extraCols, cls, join, admin, extraVals)
	}
	okJoin := jc(base + 2)
	cases := []struct {
		name, sql string
		args      []any
		state     string
	}{
		{"join_code có chữ 0", ins("X-"+suffix+"1", "ABCDEF0", "", ""), nil, check},
		{"join_code có chữ O", ins("X-"+suffix+"2", "ABCDEFO", "", ""), nil, check},
		{"join_code có chữ 1", ins("X-"+suffix+"3", "ABCDEF1", "", ""), nil, check},
		{"join_code có chữ I", ins("X-"+suffix+"4", "ABCDEFI", "", ""), nil, check},
		{"join_code có chữ L", ins("X-"+suffix+"5", "ABCDEFL", "", ""), nil, check},
		{"join_code chữ thường", ins("X-"+suffix+"6", "abcdefg", "", ""), nil, check},
		{"join_code 6 ký tự", ins("X-"+suffix+"7", "ABCDEF", "", ""), nil, check},
		{"trùng class_code", ins("LOP-"+suffix+"A", okJoin, "", ""), nil, unique},
		{"trùng join_code", ins("X-"+suffix+"8", jc(base+1), "", ""), nil, unique},
		{"semester sai dạng", strings.Replace(ins("X-"+suffix+"9", okJoin, "", ""), "2026-2027-HK1", "2026-HK1", 1), nil, check},
		{"semester HK4", strings.Replace(ins("X-"+suffix+"10", okJoin, "", ""), "HK1", "HK4", 1), nil, check},
		{"ARCHIVED thiếu archived_at", ins("X-"+suffix+"11", okJoin, ", status, join_enabled", ", 'ARCHIVED', false"), nil, check},
		{"ACTIVE mà có archived_at", ins("X-"+suffix+"12", okJoin, ", archived_at", ", now()"), nil, check},
		{"ARCHIVED còn join_enabled", ins("X-"+suffix+"13", okJoin, ", status, archived_at, join_enabled", ", 'ARCHIVED', now(), true"), nil, check},
		{"capacity 0", ins("X-"+suffix+"14", okJoin, ", capacity", ", 0"), nil, check},
		{"capacity 1001", ins("X-"+suffix+"15", okJoin, ", capacity", ", 1001"), nil, check},
		{"allowed_email_domain sai dạng", ins("X-"+suffix+"16", okJoin, ", allowed_email_domain", ", '@ptit'"), nil, check},
		{"allowed_email_domain chữ hoa", ins("X-"+suffix+"17", okJoin, ", allowed_email_domain", ", 'PTIT.edu.vn'"), nil, check},
		{"enrollments trùng (course, user)", `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'PENDING', 'CODE')`, []any{c1, sv}, unique},
		{"hai giảng viên ACTIVE một lớp", `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, []any{c1, t2}, unique},
		{"student_code_snapshot cho TA", `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'TA', 'ACTIVE', 'ADMIN', 'B20DCCN001')`, []any{c1, ta}, check},
		{"student_code_snapshot sai dạng", `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'CODE', 'ab')`, []any{c1, ta}, check},
		{"REMOVED thiếu removed_at", `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'REMOVED', 'CODE')`, []any{c1, ta}, check},
		{"warning lạ", `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, warning) values ($1, $2, 'STUDENT', 'PENDING', 'CODE', 'LAC')`, []any{c1, ta}, check},
		{"warning cho vai không phải STUDENT", `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, warning) values ($1, $2, 'TA', 'PENDING', 'ADMIN', 'EMAIL_MISMATCH')`, []any{c1, ta}, check},
		{"buổi học ends_at ≤ starts_at", `insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, 2, '2026-10-12 10:00+07', '2026-10-12 08:00+07')`, []any{c1}, check},
		{"buổi học trùng session_no", `insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, 1, '2026-10-19 08:00+07', '2026-10-19 10:00+07')`, []any{c1}, unique},
		{"buổi học trùng starts_at", `insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, 3, '2026-10-05 08:00+07', '2026-10-05 10:00+07')`, []any{c1}, unique},
		{"notification link ngoài site", `insert into notifications (user_id, type, title, link) values ($1, 'TICKET', 'x', '//evil.example')`, []any{sv}, check},
		{"notification type chữ thường", `insert into notifications (user_id, type, title) values ($1, 'ticket', 'x')`, []any{sv}, check},
		{"documents ANSWER_KEY hiện cho SV", `insert into documents (course_id, title, type, visible_to_students) values ($1, 'đáp án', 'ANSWER_KEY', true)`, []any{c1}, check},
		{"documents sha256 sai", `insert into documents (course_id, title, sha256) values ($1, 'x', 'zz')`, []any{c1}, check},
		{"documents week_no 21", `insert into documents (course_id, title, week_no) values ($1, 'x', 21)`, []any{c1}, check},
		{"chunk course_ids rỗng", `insert into content_chunks (document_id, course_ids, ord, text) select (select id from documents limit 1), '{}', 0, 't'`, nil, check},
	}
	mustExec(`insert into documents (course_id, title) values ($1, 'tài liệu mồi')`, c1)
	require.GreaterOrEqual(t, len(cases), 20)
	for _, tc := range cases {
		_, err := conn.Exec(ctx, tc.sql, tc.args...)
		require.Error(t, err, tc.name)
		require.Equal(t, tc.state, sqlState(err), "%s: %v", tc.name, err)
		// sau mỗi lỗi vẫn dùng được kết nối (không nằm trong giao dịch)
	}
	// hợp lệ vẫn chèn được: kiểm chứng rằng các ca trên bị từ chối vì đúng ràng buộc, không vì dữ liệu nền sai
	mustExec(ins("OK-"+suffix, okJoin, ", capacity, allowed_email_domain", ", 1000, 'ptit.edu.vn'"))
	mustExec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'PENDING', 'CODE', 'B20DCCN002')`, c1, ta)
}

// TestCourseIndexes — US-P2-07 AC3: truy vấn thật của P2 dùng chỉ mục (không Seq Scan) trên bảng > 1.000 dòng.
func TestCourseIndexes(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	var admin, hot uuid.UUID
	require.NoError(t, conn.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'A', 'ADMIN') returning id`, "ix."+uuid.NewString()[:8]+"@example.test").Scan(&admin))
	require.NoError(t, conn.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'H', 'STUDENT') returning id`, "hot."+uuid.NewString()[:8]+"@example.test").Scan(&hot))
	_, err := conn.Exec(ctx, `
		insert into courses (subject_code, class_code, name, semester, join_code, created_by)
		select 'INT1006', 'IX-'||i, 'Lớp '||i, '2026-2027-HK1', translate(upper(substr(md5(i::text), 1, 7)), '01', 'XY'), $1
		  from generate_series(1, 3000) i`, admin)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `
		insert into users (email, full_name, role)
		select 'bulk'||i||'.'||substr(md5(random()::text),1,6)||'@example.test', 'SV '||i, 'STUDENT' from generate_series(1, 20000) i`)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `
		insert into enrollments (course_id, user_id, role_in_course, status, joined_via, removed_at)
		select c.id, u.id, 'STUDENT', (array['ACTIVE','PENDING','REMOVED'])[1 + (u.rn % 3)]::enrollment_status, 'CODE',
		       case when u.rn % 3 = 2 then now() end
		from (select id, row_number() over (order by id) rn from users where email like 'bulk%') u
		join (select id, row_number() over (order by id) rn from courses where class_code like 'IX-%') c on c.rn = 1 + (u.rn % 30)`) // 20.000 ghi danh dồn vào 30 lớp đầu
	require.NoError(t, err)
	var cid uuid.UUID
	require.NoError(t, conn.QueryRow(ctx, `select id from courses where class_code = 'IX-7'`).Scan(&cid))
	var someJoin string
	require.NoError(t, conn.QueryRow(ctx, `select join_code from courses where class_code = 'IX-1234'`).Scan(&someJoin))
	_, err = conn.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'CODE')`, cid, hot)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `
		insert into notifications (user_id, type, title, created_at)
		select (array[$1::uuid])[1], 'TICKET', 'n'||i, now() - (i||' seconds')::interval from generate_series(1, 3000) i`, hot)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `insert into documents (course_id, title) values ($1, 'tl')`, cid)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `
		insert into content_chunks (document_id, course_ids, ord, text)
		select (select id from documents where course_id = $1 limit 1),
		       case when i % 500 = 0 then array[$1::uuid] else array[gen_random_uuid()] end, i, 't'||i from generate_series(1, 20000) i`, cid)
	require.NoError(t, err)
	for _, tbl := range []string{"enrollments", "notifications", "content_chunks", "courses"} {
		_, err = conn.Exec(ctx, "analyze "+tbl)
		require.NoError(t, err)
	}

	plan := func(sql string, args ...any) string {
		t.Helper()
		var raw []byte
		require.NoError(t, conn.QueryRow(ctx, "explain (format json) "+sql, args...).Scan(&raw))
		var doc []map[string]any
		require.NoError(t, json.Unmarshal(raw, &doc))
		return string(raw)
	}
	cases := []struct {
		name, sql string
		args      []any
		index     []string // chấp nhận chỉ mục nào trong danh sách (planner chọn theo thống kê)
	}{
		{"enrollments theo (course_id, user_id)", `select role_in_course, status from enrollments where course_id = $1 and user_id = $2`, []any{cid, hot}, []string{"enrollments_course_user_key", "enrollments_user_idx"}},
		{"enrollments theo (user_id, status)", `select course_id from enrollments where user_id = $1 and status = 'ACTIVE'`, []any{hot}, []string{"enrollments_user_idx"}},
		{"thành viên theo (course_id, status, …)", `select user_id from enrollments where course_id = $1 and status = 'ACTIVE' and role_in_course = 'STUDENT' order by status_changed_at desc, user_id limit 30`, []any{cid}, []string{"enrollments_course_status_idx"}},
		{"tra join_code", `select id from courses where join_code = $1`, []any{someJoin}, []string{"courses_join_code_key"}},
		{"chunk theo course_ids (GIN)", `select id from content_chunks where course_ids @> array[$1]::uuid[]`, []any{cid}, []string{"content_chunks_course_ids_gin"}},
		{"notifications theo (user_id, created_at DESC, id DESC)", `select id from notifications where user_id = $1 order by created_at desc, id desc limit 30`, []any{hot}, []string{"notifications_user_created_idx"}},
	}
	for _, tc := range cases {
		p := plan(tc.sql, tc.args...)
		hit := false
		for _, ix := range tc.index {
			hit = hit || strings.Contains(p, ix)
		}
		require.True(t, hit, "%s: plan không dùng %v: %s", tc.name, tc.index, p)
		require.NotContains(t, p, `"Seq Scan"`, tc.name)
	}
	require.Contains(t, plan(`select id from content_chunks where course_ids @> array[$1]::uuid[]`, cid), "Bitmap Index Scan")
	// `ListMyCourses` (truy vấn thật của handler) cũng không quét tuần tự bảng ghi danh
	require.NotContains(t, plan(`select e.id from enrollments e join courses c on c.id = e.course_id where e.user_id = $1 and e.status = 'ACTIVE' order by e.created_at desc, e.id desc limit 31`, hot), `"Seq Scan","Parallel Aware":false,"Relation Name":"enrollments"`)
}

// TestChunksForCourseIsolation — US-P2-07 AC6: chỉ trả chunk của đúng lớp; chunk chia sẻ hiện ở cả hai lớp; không lộ chunk lớp khác.
func TestChunksForCourseIsolation(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	var admin uuid.UUID
	require.NoError(t, conn.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'A', 'ADMIN') returning id`, "ch."+uuid.NewString()[:8]+"@example.test").Scan(&admin))
	mk := func(cls, join string) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, conn.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'L', '2026-2027-HK1', $2, $3) returning id`, cls, join, admin).Scan(&id))
		return id
	}
	base := int(uuid.New().ID() % 1000000)
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	j := func(off int) string {
		b := make([]byte, 7)
		x := base + off*977
		for i := range b {
			b[i] = alpha[(x+i*i*31+off)%len(alpha)]
			x /= 3
			x += off + i
		}
		return string(b)
	}
	tag := uuid.NewString()[:6]
	c1, c2, c3 := mk("CH1-"+tag, j(1)), mk("CH2-"+tag, j(2)), mk("CH3-"+tag, j(3))
	var doc uuid.UUID
	require.NoError(t, conn.QueryRow(ctx, `insert into documents (course_id, title) values ($1, 'tl') returning id`, c1).Scan(&doc))
	ins := func(ord int, ids ...uuid.UUID) {
		_, err := conn.Exec(ctx, `insert into content_chunks (document_id, course_ids, ord, text) values ($1, $2, $3, $4)`, doc, ids, ord, fmt.Sprintf("chunk-%d", ord))
		require.NoError(t, err)
	}
	ins(1, c1)
	ins(2, c1, c2)
	ins(3, c3)

	q := store.New(conn)
	texts := func(c uuid.UUID) []string {
		rows, err := q.ChunksForCourse(ctx, store.ChunksForCourseParams{CourseID: c, Lim: 100})
		require.NoError(t, err)
		var out []string
		for _, r := range rows {
			out = append(out, r.Text)
		}
		return out
	}
	require.ElementsMatch(t, []string{"chunk-1", "chunk-2"}, texts(c1))
	require.Equal(t, []string{"chunk-2"}, texts(c2), "lớp 2 chỉ thấy chunk chia sẻ")
	require.Equal(t, []string{"chunk-3"}, texts(c3), "lớp 3 không thấy chunk của lớp 1")
	require.Empty(t, texts(uuid.New()))
}
