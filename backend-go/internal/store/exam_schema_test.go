package store_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/db"
)

func migrateCmd(ctx context.Context, url, cmd string) error {
	return db.Migrate(ctx, url, cmd, io.Discard)
}

var examTables = []string{"question_bank", "question_options", "code_problems", "code_testcases", "exams", "exam_items", "exam_attempts", "exam_answers",
	"code_drafts", "code_submissions", "exam_events", "similarity_reports", "exam_appeals"}

// TestExamSchema — US-PE-01 AC1: 13 bảng, 16 enum, đủ cột / kiểu của SRS FEAT-weekly-exam 5.2–5.14, mọi bảng có course_id.
func TestExamSchema(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from information_schema.tables where table_schema='public' and table_name = any($1)`, examTables).Scan(&n))
	require.Equal(t, 13, n)

	wantEnums := map[string][]string{
		"question_type":           {"MCQ_SINGLE", "MCQ_MULTI", "TRUE_FALSE", "CODE", "SHORT", "ESSAY"},
		"question_difficulty":     {"EASY", "MEDIUM", "HARD"},
		"question_origin":         {"MANUAL", "AI_DRAFT", "EXTRACTED", "GENERATED"},
		"question_review_status":  {"DRAFT", "PENDING", "APPROVED", "REJECTED"},
		"checker_kind":            {"EXACT", "TOKENS", "FLOAT_EPS"},
		"exam_kind":               {"MCQ", "CODE", "MIXED"},
		"exam_status":             {"DRAFT", "SCHEDULED", "OPEN", "CLOSED", "PUBLISHED"},
		"multi_scoring":           {"ALL_OR_NOTHING", "PARTIAL"},
		"attempt_status":          {"IN_PROGRESS", "GRADING", "GRADED"},
		"attempt_submit_reason":   {"MANUAL", "TIMEOUT", "CLOSED"},
		"submission_kind":         {"RUN", "SUBMIT"},
		"submission_status":       {"QUEUED", "RUNNING", "DONE", "SUPERSEDED", "ERROR"},
		"judge_verdict":           {"AC", "WA", "TLE", "MLE", "RE", "CE", "OLE", "IE"},
		"exam_event_type":         {"TAB_HIDDEN", "TAB_VISIBLE", "PASTE", "OFFLINE", "ONLINE", "TAB_TAKEOVER", "CHAT_BLOCKED"},
		"similarity_review_state": {"NEW", "CLEARED", "FOLLOW_UP"},
		"appeal_status":           {"OPEN", "UPHELD", "ADJUSTED"},
	}
	for enum, want := range wantEnums {
		rows, err := conn.Query(ctx, `select e.enumlabel from pg_enum e join pg_type t on t.oid = e.enumtypid where t.typname = $1 order by e.enumsortorder`, enum)
		require.NoError(t, err)
		var got []string
		for rows.Next() {
			var l string
			require.NoError(t, rows.Scan(&l))
			got = append(got, l)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, want, got, enum)
	}

	wantCols := map[string][]string{
		"question_bank":      {"id", "course_id", "type", "title", "topic", "difficulty", "stem", "answer_key", "explanation", "citations", "origin", "review_status", "created_by", "reviewed_by", "reviewed_at", "ai_job_id", "archived_at", "version", "created_at", "updated_at"},
		"question_options":   {"id", "course_id", "question_id", "position", "body", "pinned_last"},
		"code_problems":      {"question_id", "course_id", "languages", "time_limit_ms", "memory_limit_mb", "output_limit_kb", "checker", "float_eps", "starter_code", "reference_language", "reference_source", "reference_verified_version", "reference_verified_at", "tests_version", "created_at", "updated_at"},
		"code_testcases":     {"id", "course_id", "problem_id", "position", "name", "is_sample", "weight", "input", "input_blob_key", "expected", "expected_blob_key", "input_bytes", "expected_bytes", "source", "approved", "created_at", "updated_at"},
		"exams":              {"id", "course_id", "title", "instructions", "kind", "status", "opens_at", "closes_at", "duration_minutes", "shuffle_questions", "shuffle_options", "max_score", "rounding_step", "multi_scoring", "reveal_answers", "appeal_days", "publish_hold", "regrading", "published_at", "created_by", "version", "created_at", "updated_at"},
		"exam_items":         {"id", "course_id", "exam_id", "question_id", "position", "points", "override", "created_at", "updated_at"},
		"exam_attempts":      {"id", "course_id", "exam_id", "student_id", "status", "started_at", "deadline_at", "submitted_at", "submit_reason", "writer_tab", "writer_seen_at", "auto_score", "adjusted_score", "adjusted_reason", "adjusted_by", "adjusted_at", "breakdown", "graded_at", "version", "created_at", "updated_at"},
		"exam_answers":       {"attempt_id", "item_id", "course_id", "answer", "saved_at"},
		"code_drafts":        {"attempt_id", "item_id", "course_id", "language", "source", "rev", "updated_at"},
		"code_submissions":   {"id", "course_id", "exam_id", "attempt_id", "item_id", "problem_id", "student_id", "kind", "language", "source", "source_sha256", "auto", "status", "verdict", "tests_version", "compile_ok", "compile_log", "results", "passed_weight", "total_weight", "time_ms_max", "memory_kb_max", "attempts", "lease_until", "next_attempt_at", "fail_count", "enqueued_at", "judged_at", "created_at", "updated_at"},
		"exam_events":        {"id", "course_id", "exam_id", "attempt_id", "student_id", "type", "occurred_at", "client_at", "meta"},
		"similarity_reports": {"id", "course_id", "exam_id", "problem_id", "run_id", "submission_a", "submission_b", "attempt_a", "attempt_b", "score", "shared_fingerprints", "flagged", "algorithm", "review_state", "reviewed_by", "reviewed_at", "note", "created_at"},
		"exam_appeals":       {"id", "course_id", "exam_id", "attempt_id", "student_id", "reason", "status", "response", "responded_by", "responded_at", "score_before", "score_after", "version", "created_at", "updated_at"},
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
		require.Contains(t, got, "course_id", table+": bảng thuộc lớp phải có course_id")
	}

	typ := func(table, col string) string {
		var s string
		require.NoError(t, conn.QueryRow(ctx, `select format_type(a.atttypid, a.atttypmod) || case when a.attnotnull then ' NOT NULL' else '' end from pg_attribute a where a.attrelid = ('public.'||$1)::regclass and a.attname = $2`, table, col).Scan(&s))
		return s
	}
	require.Equal(t, "numeric(5,2) NOT NULL", typ("exams", "max_score"))
	require.Equal(t, "numeric(3,2) NOT NULL", typ("exams", "rounding_step"))
	require.Equal(t, "numeric(5,2) NOT NULL", typ("exam_items", "points"))
	require.Equal(t, "numeric(5,2)", typ("exam_attempts", "auto_score"))
	require.Equal(t, "numeric(5,2)", typ("exam_attempts", "adjusted_score"))
	require.Equal(t, "numeric(4,3) NOT NULL", typ("similarity_reports", "score"))
	require.Equal(t, "numeric(12,10)", typ("code_problems", "float_eps"))
	require.Equal(t, "text[] NOT NULL", typ("code_problems", "languages"))
	require.Equal(t, "character(64) NOT NULL", typ("code_submissions", "source_sha256"))
	require.Equal(t, "jsonb", typ("question_bank", "answer_key"))
	require.Equal(t, "smallint NOT NULL", typ("code_submissions", "fail_count"))
	require.Equal(t, "timestamp with time zone", typ("code_submissions", "lease_until"))
	require.Equal(t, "timestamp with time zone", typ("code_submissions", "enqueued_at"))
	require.Equal(t, "timestamp with time zone NOT NULL", typ("code_submissions", "next_attempt_at"))
	require.Equal(t, "smallint", typ("exams", "duration_minutes"))
	require.Equal(t, "integer NOT NULL", typ("code_testcases", "input_bytes"))
}

// TestExamIndexesCourseFirst — chỉ mục không-UNIQUE của 13 bảng bắt đầu bằng course_id, trừ ĐÚNG 9 ngoại lệ quét toàn cục (AC1, góp ý #7).
func TestExamIndexesCourseFirst(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	rows, err := conn.Query(ctx, `
		select c.relname, i.relname, a.attname
		  from pg_index x
		  join pg_class c on c.oid = x.indrelid
		  join pg_class i on i.oid = x.indexrelid
		  join pg_attribute a on a.attrelid = c.oid and a.attnum = x.indkey[0]
		 where c.relname = any($1) and not x.indisunique and not x.indisprimary`, examTables)
	require.NoError(t, err)
	defer rows.Close()
	var notFirst []string
	total := 0
	for rows.Next() {
		var table, idx, first string
		require.NoError(t, rows.Scan(&table, &idx, &first))
		total++
		if first != "course_id" {
			notFirst = append(notFirst, idx)
		}
	}
	require.NoError(t, rows.Err())
	sort.Strings(notFirst)
	require.Equal(t, []string{
		"code_submissions_lease_idx", "code_submissions_queue_idx", "code_submissions_runs_idx", "exam_attempts_due_idx", "exam_attempts_running_idx",
		"exam_items_question_idx", "exams_due_close_idx", "exams_due_open_idx", "similarity_reports_run_idx",
	}, notFirst, "đúng 9 ngoại lệ, mỗi cái có lý do ở US-PE-01 AC1")
	require.Greater(t, total, 20)
}

// TestExamForeignKeys — 5.16: FK phức hợp (course_id, …) tới bảng thuộc lớp, hành động xoá cha; cột người trỏ users(id).
func TestExamForeignKeys(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	type fk struct{ table, cols, parent, parentCols, onDelete string }
	want := []fk{
		{"question_options", "course_id,question_id", "question_bank", "course_id,id", "CASCADE"},
		{"code_problems", "course_id,question_id", "question_bank", "course_id,id", "CASCADE"},
		{"code_testcases", "course_id,problem_id", "code_problems", "course_id,question_id", "CASCADE"},
		{"exam_items", "course_id,exam_id", "exams", "course_id,id", "CASCADE"},
		{"exam_items", "course_id,question_id", "question_bank", "course_id,id", "NO ACTION"},
		{"exam_attempts", "course_id,exam_id", "exams", "course_id,id", "NO ACTION"},
		{"exam_answers", "course_id,attempt_id", "exam_attempts", "course_id,id", "CASCADE"},
		{"exam_answers", "course_id,item_id", "exam_items", "course_id,id", "CASCADE"},
		{"code_drafts", "course_id,attempt_id", "exam_attempts", "course_id,id", "CASCADE"},
		{"code_drafts", "course_id,item_id", "exam_items", "course_id,id", "CASCADE"},
		{"code_submissions", "course_id,exam_id", "exams", "course_id,id", "NO ACTION"},
		{"code_submissions", "course_id,attempt_id", "exam_attempts", "course_id,id", "NO ACTION"},
		{"code_submissions", "course_id,item_id", "exam_items", "course_id,id", "NO ACTION"},
		{"code_submissions", "course_id,problem_id", "code_problems", "course_id,question_id", "NO ACTION"},
		{"exam_events", "course_id,exam_id", "exams", "course_id,id", "NO ACTION"},
		{"exam_events", "course_id,attempt_id", "exam_attempts", "course_id,id", "CASCADE"},
		{"similarity_reports", "course_id,exam_id", "exams", "course_id,id", "NO ACTION"},
		{"similarity_reports", "course_id,problem_id", "code_problems", "course_id,question_id", "NO ACTION"},
		{"similarity_reports", "course_id,submission_a", "code_submissions", "course_id,id", "CASCADE"},
		{"similarity_reports", "course_id,submission_b", "code_submissions", "course_id,id", "CASCADE"},
		{"similarity_reports", "course_id,attempt_a", "exam_attempts", "course_id,id", "CASCADE"},
		{"similarity_reports", "course_id,attempt_b", "exam_attempts", "course_id,id", "CASCADE"},
		{"exam_appeals", "course_id,exam_id", "exams", "course_id,id", "NO ACTION"},
		{"exam_appeals", "course_id,attempt_id", "exam_attempts", "course_id,id", "CASCADE"},
	}
	rows, err := conn.Query(ctx, `
		select c.conrelid::regclass::text,
		       (select string_agg(a.attname, ',' order by k.ord) from unnest(c.conkey) with ordinality k(attnum, ord) join pg_attribute a on a.attrelid = c.conrelid and a.attnum = k.attnum),
		       c.confrelid::regclass::text,
		       (select string_agg(a.attname, ',' order by k.ord) from unnest(c.confkey) with ordinality k(attnum, ord) join pg_attribute a on a.attrelid = c.confrelid and a.attnum = k.attnum),
		       case c.confdeltype when 'c' then 'CASCADE' when 'a' then 'NO ACTION' when 'r' then 'RESTRICT' else c.confdeltype::text end
		  from pg_constraint c
		 where c.contype = 'f' and c.conrelid::regclass::text = any($1) and array_length(c.conkey, 1) = 2`, examTables)
	require.NoError(t, err)
	defer rows.Close()
	var got []fk
	for rows.Next() {
		var f fk
		require.NoError(t, rows.Scan(&f.table, &f.cols, &f.parent, &f.parentCols, &f.onDelete))
		got = append(got, f)
	}
	require.NoError(t, rows.Err())
	key := func(f fk) string {
		return strings.Join([]string{f.table, f.cols, f.parent, f.parentCols, f.onDelete}, "|")
	}
	var gk, wk []string
	for _, f := range got {
		gk = append(gk, key(f))
	}
	for _, f := range want {
		wk = append(wk, key(f))
	}
	require.ElementsMatch(t, wk, gk)

	// đích của FK phức hợp phải có UNIQUE (course_id, …)
	for _, idx := range []string{"question_bank_course_id_key", "exams_course_id_key", "exam_items_course_id_key", "exam_attempts_course_id_key", "code_submissions_course_id_key", "code_problems_course_question_key"} {
		var uniq bool
		require.NoError(t, conn.QueryRow(ctx, `select x.indisunique from pg_index x join pg_class i on i.oid = x.indexrelid where i.relname = $1`, idx).Scan(&uniq), idx)
		require.True(t, uniq, idx)
	}

	// cột người và course_id (đơn) → users / courses
	var nUsers int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_constraint c where c.contype = 'f' and c.confrelid = 'users'::regclass and c.conrelid::regclass::text = any($1)`, examTables).Scan(&nUsers))
	require.Equal(t, 10, nUsers) // created_by×2, reviewed_by×2, student_id×4 (attempts, submissions, events, appeals), adjusted_by, responded_by
}

// examFx là dữ liệu nền tối thiểu của TestExamConstraints / TestExamIndexes.
type examFx struct {
	course1, course2, teacher, student                                  uuid.UUID
	q1, q2, qOther                                                      uuid.UUID // MCQ lớp 1; câu CODE lớp 1; MCQ lớp 2
	exam, examOther, item1, item2, attempt, attempt2, sub, sub2, appeal uuid.UUID
}

func newExamFx(t *testing.T, ctx context.Context, conn *pgx.Conn) examFx {
	t.Helper()
	var f examFx
	one := func(dst *uuid.UUID, sql string, args ...any) {
		require.NoError(t, conn.QueryRow(ctx, sql, args...).Scan(dst), sql)
	}
	id := uuid.New()
	sfx := strings.ToUpper(id.String()[:6])
	lsfx := strings.ToLower(sfx)
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789" // mã tham gia không có 0 O 1 I L
	jc := func(seed byte) string {
		b := make([]byte, 7)
		for i := range b {
			b[i] = alpha[(int(id[i])+int(seed)*7)%len(alpha)]
		}
		return string(b)
	}
	one(&f.teacher, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, "t."+lsfx+"@example.test")
	one(&f.student, `insert into users (email, full_name, role) values ($1, 'SV', 'STUDENT') returning id`, "s."+lsfx+"@example.test")
	one(&f.course1, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp 1', '2026-2027-HK1', $2, $3) returning id`, "EX1-"+sfx, jc(1), f.teacher)
	one(&f.course2, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp 2', '2026-2027-HK1', $2, $3) returning id`, "EX2-"+sfx, jc(2), f.teacher)
	mcq := `insert into question_bank (course_id, type, title, topic, stem, answer_key, created_by) values ($1, 'MCQ_SINGLE', 'Câu 1', 'Chủ đề', 'Nội dung', '{"option_ids":["x"]}', $2) returning id`
	one(&f.q1, mcq, f.course1, f.teacher)
	one(&f.qOther, mcq, f.course2, f.teacher)
	one(&f.q2, `insert into question_bank (course_id, type, title, topic, stem, created_by) values ($1, 'CODE', 'Bài code', 'Chủ đề', 'Đề', $2) returning id`, f.course1, f.teacher)
	exec := func(sql string, args ...any) {
		_, err := conn.Exec(ctx, sql, args...)
		require.NoError(t, err, sql)
	}
	exec(`insert into code_problems (question_id, course_id) values ($1, $2)`, f.q2, f.course1)
	one(&f.exam, `insert into exams (course_id, title, created_by) values ($1, 'Bài thi', $2) returning id`, f.course1, f.teacher)
	one(&f.examOther, `insert into exams (course_id, title, created_by) values ($1, 'Bài thi lớp 2', $2) returning id`, f.course2, f.teacher)
	one(&f.item1, `insert into exam_items (course_id, exam_id, question_id, position) values ($1, $2, $3, 1) returning id`, f.course1, f.exam, f.q1)
	one(&f.item2, `insert into exam_items (course_id, exam_id, question_id, position) values ($1, $2, $3, 2) returning id`, f.course1, f.exam, f.q2)
	one(&f.attempt, `insert into exam_attempts (course_id, exam_id, student_id, deadline_at) values ($1, $2, $3, now() + interval '1 hour') returning id`, f.course1, f.exam, f.student)
	var s2 uuid.UUID
	one(&s2, `insert into users (email, full_name, role) values ($1, 'SV2', 'STUDENT') returning id`, "s2."+lsfx+"@example.test")
	one(&f.attempt2, `insert into exam_attempts (course_id, exam_id, student_id, deadline_at) values ($1, $2, $3, now() + interval '1 hour') returning id`, f.course1, f.exam, s2)
	sub := `insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256) values ($1, $2, $3, $4, $5, $6, 'SUBMIT', 'cpp17', 'int main(){}', repeat('a', 64)) returning id`
	one(&f.sub, sub, f.course1, f.exam, f.attempt, f.item2, f.q2, f.student)
	one(&f.sub2, sub, f.course1, f.exam, f.attempt2, f.item2, f.q2, s2)
	one(&f.appeal, `insert into exam_appeals (course_id, exam_id, attempt_id, student_id, reason) values ($1, $2, $3, $4, 'Xin xem lại') returning id`, f.course1, f.exam, f.attempt2, s2)
	exec(`insert into code_testcases (course_id, problem_id, position, name, input, expected, input_bytes, expected_bytes) values ($1, $2, 1, 'sample1', '1', '1', 1, 1)`, f.course1, f.q2)
	return f
}

// TestExamConstraints — US-PE-01 AC2: ≥ 42 ca ghi sai bị DB từ chối đúng SQLSTATE (23514 CHECK · 23505 UNIQUE · 23503 FK).
func TestExamConstraints(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newExamFx(t, ctx, conn)
	const check, unique, fk = "23514", "23505", "23503"
	rep := func(n int, s string) string { return strings.Repeat(s, n) }
	q := func(sql string) string { // thay {tên} bằng id thật
		r := strings.NewReplacer("{c1}", f.course1.String(), "{c2}", f.course2.String(), "{t}", f.teacher.String(), "{s}", f.student.String(),
			"{q1}", f.q1.String(), "{q2}", f.q2.String(), "{qo}", f.qOther.String(), "{ex}", f.exam.String(), "{exo}", f.examOther.String(),
			"{i1}", f.item1.String(), "{i2}", f.item2.String(), "{a1}", f.attempt.String(), "{a2}", f.attempt2.String(), "{sub1}", f.sub.String(), "{sub2}", f.sub2.String())
		return r.Replace(sql)
	}
	qb := `insert into question_bank (course_id, type, title, topic, stem, answer_key, created_by%s) values ('{c1}', '%s', '%s', '%s', '%s', %s, '{t}'%s)`
	bank := func(typ, title, topic, stem, key string) string {
		return q(fmt.Sprintf(qb, "", typ, title, topic, stem, key, ""))
	}
	const okKey = `'{"option_ids":["x"]}'`
	// code_problems mới cần một câu CODE chưa có problem: tạo trong ca bằng CTE.
	newProb := func(cols, vals string) string {
		return q(fmt.Sprintf(`with qq as (insert into question_bank (course_id, type, title, topic, stem, created_by) values ('{c1}', 'CODE', 'x', 't', 's', '{t}') returning id)
			insert into code_problems (question_id, course_id%s) select id, '{c1}'%s from qq`, cols, vals))
	}
	tc := func(cols, vals string) string {
		return q(fmt.Sprintf(`insert into code_testcases (course_id, problem_id, position, name, input_bytes, expected_bytes%s) values ('{c1}', '{q2}', 50, 'tc50', 1, 1%s)`, cols, vals))
	}
	ex := func(cols, vals string) string {
		return q(fmt.Sprintf(`insert into exams (course_id, title, created_by%s) values ('{c1}', 'B', '{t}'%s)`, cols, vals))
	}
	att := func(cols, vals string) string {
		return q(fmt.Sprintf(`insert into exam_attempts (course_id, exam_id, student_id, deadline_at%s) values ('{c1}', '{ex}', (select id from users where email like 's2.%%' limit 1), now() + interval '2 hours'%s)`, cols, vals))
	}
	subIns := func(cols, vals string) string {
		return q(fmt.Sprintf(`insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256%s) values ('{c1}', '{ex}', '{a1}', '{i2}', '{q2}', '{s}', 'SUBMIT', 'cpp17', 'x', repeat('b', 64)%s)`, cols, vals))
	}

	cases := []struct{ name, sql, state string }{
		// question_bank
		{"title rỗng", bank("MCQ_SINGLE", "", "t", "s", okKey), check},
		{"title 121 ký tự", bank("MCQ_SINGLE", rep(121, "a"), "t", "s", okKey), check},
		{"topic rỗng", bank("MCQ_SINGLE", "t", "", "s", okKey), check},
		{"stem rỗng", bank("MCQ_SINGLE", "t", "t", "", okKey), check},
		{"MCQ thiếu answer_key", bank("MCQ_SINGLE", "t", "t", "s", "NULL"), check},
		{"TRUE_FALSE thiếu answer_key", bank("TRUE_FALSE", "t", "t", "s", "NULL"), check},
		{"CODE có answer_key", bank("CODE", "t", "t", "s", okKey), check},
		{"APPROVED thiếu reviewed_by", q(fmt.Sprintf(qb, ", review_status", "MCQ_SINGLE", "t", "t", "s", okKey, ", 'APPROVED'")), check},
		{"DRAFT có reviewed_by", q(fmt.Sprintf(qb, ", reviewed_by", "MCQ_SINGLE", "t", "t", "s", okKey, ", '{t}'")), check},
		{"explanation > 4000", q(fmt.Sprintf(qb, ", explanation", "MCQ_SINGLE", "t", "t", "s", okKey, ", '"+rep(4001, "e")+"'")), check},
		// question_options
		{"option position 0", q(`insert into question_options (course_id, question_id, position, body) values ('{c1}', '{q1}', 0, 'a')`), check},
		{"option position 9", q(`insert into question_options (course_id, question_id, position, body) values ('{c1}', '{q1}', 9, 'a')`), check},
		{"option body rỗng", q(`insert into question_options (course_id, question_id, position, body) values ('{c1}', '{q1}', 1, '')`), check},
		{"option body > 1000", q(`insert into question_options (course_id, question_id, position, body) values ('{c1}', '{q1}', 1, '` + rep(1001, "b") + `')`), check},
		{"option lệch lớp với câu hỏi", q(`insert into question_options (course_id, question_id, position, body) values ('{c2}', '{q1}', 1, 'a')`), fk},
		// code_problems
		{"languages rỗng", newProb(", languages", ", '{}'"), check},
		{"languages ngoài c11/cpp17", newProb(", languages", ", '{java}'"), check},
		{"time_limit 99", newProb(", time_limit_ms", ", 99"), check},
		{"time_limit 10001", newProb(", time_limit_ms", ", 10001"), check},
		{"memory 15", newProb(", memory_limit_mb", ", 15"), check},
		{"memory 513 (trần 512)", newProb(", memory_limit_mb", ", 513"), check},
		{"output_limit 0", newProb(", output_limit_kb", ", 0"), check},
		{"FLOAT_EPS thiếu float_eps", newProb(", checker", ", 'FLOAT_EPS'"), check},
		{"float_eps với EXACT", newProb(", float_eps", ", 0.001"), check},
		{"float_eps 0.5 vượt 0.1", newProb(", checker, float_eps", ", 'FLOAT_EPS', 0.5"), check},
		{"reference_language lạ", newProb(", reference_language, reference_source", ", 'java', 'x'"), check},
		{"reference_language không có source", newProb(", reference_language", ", 'cpp17'"), check},
		{"problem lệch lớp với câu hỏi", q(`with qq as (insert into question_bank (course_id, type, title, topic, stem, created_by) values ('{c1}', 'CODE', 'x', 't', 's', '{t}') returning id)
			insert into code_problems (question_id, course_id) select id, '{c2}' from qq`), fk},
		// code_testcases
		{"test vừa input vừa blob", tc(", input, input_blob_key, expected", ", 'a', 'k', 'b'"), check},
		{"test không input", tc(", expected", ", 'b'"), check},
		{"test tên sai", q(`insert into code_testcases (course_id, problem_id, position, name, input, expected, input_bytes, expected_bytes) values ('{c1}', '{q2}', 51, 'a b', 'a', 'b', 1, 1)`), check},
		{"test weight 1001", tc(", input, expected, weight", ", 'a', 'b', 1001"), check},
		{"test weight âm", tc(", input, expected, weight", ", 'a', 'b', -1"), check},
		{"test position 0", q(`insert into code_testcases (course_id, problem_id, position, name, input, expected, input_bytes, expected_bytes) values ('{c1}', '{q2}', 0, 'p0', 'a', 'b', 1, 1)`), check},
		{"test input_bytes > 1 MiB", q(`insert into code_testcases (course_id, problem_id, position, name, input, expected, input_bytes, expected_bytes) values ('{c1}', '{q2}', 52, 'big', 'a', 'b', 1048577, 1)`), check},
		{"test source lạ", tc(", input, expected, source", ", 'a', 'b', 'FOO'"), check},
		{"test trùng position", q(`insert into code_testcases (course_id, problem_id, position, name, input, expected, input_bytes, expected_bytes) values ('{c1}', '{q2}', 1, 'dup', 'a', 'b', 1, 1)`), unique}, // hoãn tới COMMIT
		{"test lệch lớp với bài code", q(`insert into code_testcases (course_id, problem_id, position, name, input, expected, input_bytes, expected_bytes) values ('{c2}', '{q2}', 60, 'x', 'a', 'b', 1, 1)`), fk},
		// exams
		{"exam title rỗng", q(`insert into exams (course_id, title, created_by) values ('{c1}', '', '{t}')`), check},
		{"exam SCHEDULED thiếu mốc", ex(", status", ", 'SCHEDULED'"), check},
		{"exam closes <= opens", ex(", opens_at, closes_at", ", now() + interval '2 hours', now() + interval '1 hour'"), check},
		{"exam duration 0", ex(", duration_minutes", ", 0"), check},
		{"exam duration 301", ex(", duration_minutes", ", 301"), check},
		{"exam duration vượt khung giờ", ex(", opens_at, closes_at, duration_minutes", ", now() + interval '1 hour', now() + interval '1 hour 10 minutes', 15"), check},
		{"exam max_score 0", ex(", max_score", ", 0"), check},
		{"exam max_score 101", ex(", max_score", ", 101"), check},
		{"exam rounding_step 0.05", ex(", rounding_step", ", 0.05"), check},
		{"exam appeal_days 31", ex(", appeal_days", ", 31"), check},
		{"exam PUBLISHED thiếu published_at", ex(", status, opens_at, closes_at, duration_minutes", ", 'PUBLISHED', now() - interval '3 hours', now() - interval '1 hour', 30"), check},
		{"exam instructions > 4000", ex(", instructions", ", '"+rep(4001, "i")+"'"), check},
		// exam_items
		{"item points 0", q(`insert into exam_items (course_id, exam_id, question_id, position, points) values ('{c1}', '{ex}', '{q1}', 9, 0)`), check},
		{"item points 101", q(`insert into exam_items (course_id, exam_id, question_id, position, points) values ('{c1}', '{ex}', '{q1}', 9, 101)`), check},
		{"item position 0", q(`insert into exam_items (course_id, exam_id, question_id, position) values ('{c1}', '{ex}', '{q1}', 0)`), check},
		{"item trùng câu trong bài", q(`insert into exam_items (course_id, exam_id, question_id, position) values ('{c1}', '{ex}', '{q1}', 7)`), unique},
		{"item trộn câu của lớp khác", q(`insert into exam_items (course_id, exam_id, question_id, position) values ('{c1}', '{ex}', '{qo}', 8)`), fk},
		{"item bài thi lớp khác", q(`insert into exam_items (course_id, exam_id, question_id, position) values ('{c1}', '{exo}', '{q1}', 1)`), fk},
		// exam_attempts
		{"attempt deadline <= started", q(`insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at) values ('{c1}', '{ex}', '{t}', now(), now() - interval '1 minute')`), check},
		{"attempt GRADED thiếu điểm", att(", status, submitted_at, submit_reason", ", 'GRADED', now(), 'MANUAL'"), check},
		{"attempt submitted_at khi IN_PROGRESS", att(", submitted_at, submit_reason", ", now(), 'MANUAL'"), check},
		{"attempt submit_reason thiếu submitted_at", att(", submit_reason", ", 'TIMEOUT'"), check},
		{"attempt sửa điểm thiếu lý do", att(", adjusted_score", ", 5"), check},
		{"attempt auto_score 101", att(", auto_score", ", 101"), check},
		{"attempt trùng (exam, student)", q(`insert into exam_attempts (course_id, exam_id, student_id, deadline_at) values ('{c1}', '{ex}', '{s}', now() + interval '1 hour')`), unique},
		{"attempt bài thi lớp khác", q(`insert into exam_attempts (course_id, exam_id, student_id, deadline_at) values ('{c1}', '{exo}', '{s}', now() + interval '1 hour')`), fk},
		// exam_answers / code_drafts
		{"answer > 4096 byte", q(`insert into exam_answers (attempt_id, item_id, course_id, answer) values ('{a1}', '{i1}', '{c1}', jsonb_build_object('x', '` + rep(4100, "z") + `'))`), check},
		{"answer item lớp khác", q(`insert into exam_answers (attempt_id, item_id, course_id, answer) values ('{a1}', '{i1}', '{c2}', '{}')`), fk},
		{"draft language lạ", q(`insert into code_drafts (attempt_id, item_id, course_id, language, source) values ('{a1}', '{i2}', '{c1}', 'rust', 'x')`), check},
		{"draft source > 64 KiB", q(`insert into code_drafts (attempt_id, item_id, course_id, language, source) values ('{a1}', '{i2}', '{c1}', 'cpp17', '` + rep(65537, "d") + `')`), check},
		// code_submissions
		{"submission source rỗng", q(`insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256) values ('{c1}', '{ex}', '{a1}', '{i2}', '{q2}', '{s}', 'SUBMIT', 'cpp17', '', repeat('b', 64))`), check},
		{"submission auto với RUN", q(`insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256, auto) values ('{c1}', '{ex}', '{a1}', '{i2}', '{q2}', '{s}', 'RUN', 'cpp17', 'x', repeat('b', 64), true)`), check},
		{"submission DONE thiếu verdict", subIns(", status", ", 'DONE'"), check},
		{"submission verdict khi QUEUED", subIns(", verdict", ", 'AC'"), check},
		{"submission RUNNING thiếu lease_until", subIns(", status", ", 'RUNNING'"), check},
		{"submission lease_until khi QUEUED", subIns(", lease_until", ", now()"), check},
		{"submission fail_count 5", subIns(", fail_count", ", 5"), check},
		{"submission passed_weight > total", subIns(", passed_weight, total_weight", ", 5, 4"), check},
		{"submission compile_log > 8192", subIns(", compile_log", ", '"+rep(8193, "l")+"'"), check},
		{"submission language lạ", q(`insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256) values ('{c1}', '{ex}', '{a1}', '{i2}', '{q2}', '{s}', 'SUBMIT', 'java', 'x', repeat('b', 64))`), check},
		{"submission trộn attempt lớp khác", q(`insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256) values ('{c2}', '{exo}', '{a1}', '{i2}', '{q2}', '{s}', 'SUBMIT', 'cpp17', 'x', repeat('b', 64))`), fk},
		// exam_events
		{"event meta > 300 byte", q(`insert into exam_events (course_id, exam_id, attempt_id, student_id, type, meta) values ('{c1}', '{ex}', '{a1}', '{s}', 'PASTE', jsonb_build_object('x', '` + rep(310, "m") + `'))`), check},
		{"event attempt lớp khác", q(`insert into exam_events (course_id, exam_id, attempt_id, student_id, type) values ('{c2}', '{exo}', '{a1}', '{s}', 'PASTE')`), fk},
		// similarity_reports
		{"similarity attempt_a >= attempt_b", q(`insert into similarity_reports (course_id, exam_id, problem_id, run_id, submission_a, submission_b, attempt_a, attempt_b, score, shared_fingerprints) values ('{c1}', '{ex}', '{q2}', gen_random_uuid(), '{sub1}', '{sub2}', '{a2}', '{a2}', 0.5, 3)`), check},
		{"similarity score > 1", q(`insert into similarity_reports (course_id, exam_id, problem_id, run_id, submission_a, submission_b, attempt_a, attempt_b, score, shared_fingerprints) values ('{c1}', '{ex}', '{q2}', gen_random_uuid(), '{sub1}', '{sub2}', least('{a1}'::uuid, '{a2}'::uuid), greatest('{a1}'::uuid, '{a2}'::uuid), 1.5, 3)`), check},
		{"similarity CLEARED thiếu reviewed_by", q(`insert into similarity_reports (course_id, exam_id, problem_id, run_id, submission_a, submission_b, attempt_a, attempt_b, score, shared_fingerprints, review_state) values ('{c1}', '{ex}', '{q2}', gen_random_uuid(), '{sub1}', '{sub2}', least('{a1}'::uuid, '{a2}'::uuid), greatest('{a1}'::uuid, '{a2}'::uuid), 0.5, 3, 'CLEARED')`), check},
		{"similarity bài nộp lớp khác", q(`insert into similarity_reports (course_id, exam_id, problem_id, run_id, submission_a, submission_b, attempt_a, attempt_b, score, shared_fingerprints) values ('{c2}', '{exo}', '{q2}', gen_random_uuid(), '{sub1}', '{sub2}', least('{a1}'::uuid, '{a2}'::uuid), greatest('{a1}'::uuid, '{a2}'::uuid), 0.5, 3)`), fk},
		// exam_appeals
		{"appeal reason rỗng", q(`insert into exam_appeals (course_id, exam_id, attempt_id, student_id, reason) values ('{c1}', '{ex}', '{a1}', '{s}', '')`), check},
		{"appeal ADJUSTED thiếu score_after", q(`insert into exam_appeals (course_id, exam_id, attempt_id, student_id, reason, status, responded_at) values ('{c1}', '{ex}', '{a1}', '{s}', 'r', 'ADJUSTED', now())`), check},
		{"appeal UPHELD thiếu responded_at", q(`insert into exam_appeals (course_id, exam_id, attempt_id, student_id, reason, status) values ('{c1}', '{ex}', '{a1}', '{s}', 'r', 'UPHELD')`), check},
		{"appeal trùng attempt", q(`insert into exam_appeals (course_id, exam_id, attempt_id, student_id, reason) values ('{c1}', '{ex}', '{a2}', '{s}', 'lần hai')`), unique},
		{"appeal attempt lớp khác", q(`insert into exam_appeals (course_id, exam_id, attempt_id, student_id, reason) values ('{c2}', '{exo}', '{a1}', '{s}', 'r')`), fk},
	}
	require.GreaterOrEqual(t, len(cases), 42)
	for _, c := range cases {
		tx, err := conn.Begin(ctx)
		require.NoError(t, err, c.name)
		_, err = tx.Exec(ctx, c.sql)
		if err == nil { // ràng buộc hoãn (UNIQUE DEFERRABLE) chỉ nổ lúc COMMIT
			err = tx.Commit(ctx)
		} else {
			_ = tx.Rollback(ctx)
		}
		var pg *pgconn.PgError
		require.ErrorAsf(t, err, &pg, "%s: phải bị DB từ chối, nhận %v", c.name, err)
		require.Equalf(t, c.state, pg.Code, "%s: %s", c.name, pg.Message)
	}
}

// TestExamIndexes — US-PE-01 AC3: EXPLAIN các truy vấn thật của PE trên 20.000 câu / 2.000 lượt làm / 20.000 bài nộp: dùng chỉ mục, không Seq Scan ở bảng > 1.000 dòng.
func TestExamIndexes(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newExamFx(t, ctx, conn)
	exec := func(sql string, args ...any) {
		_, err := conn.Exec(ctx, sql, args...)
		require.NoError(t, err, sql)
	}
	// 20.000 câu hỏi nhiều chủ đề / trạng thái
	exec(`insert into question_bank (course_id, type, title, topic, stem, answer_key, created_by, review_status, reviewed_by, difficulty)
	      select $1::uuid, 'MCQ_SINGLE', 'Câu ' || g, 'Chủ đề ' || (g % 40), 's', '{"option_ids":["x"]}', $2::uuid,
	             (array['DRAFT','APPROVED','APPROVED','APPROVED'])[1 + g % 4]::question_review_status,
	             case when g % 4 <> 0 then $2::uuid end, (array['EASY','MEDIUM','HARD'])[1 + g % 3]::question_difficulty
	        from generate_series(1, 20000) g`, f.course1, f.teacher)
	// 40 bài thi, 2.000 sinh viên, 2.000 lượt làm (1.900 đã nộp)
	exec(`insert into exams (course_id, title, created_by, status, opens_at, closes_at, duration_minutes, published_at)
	      select $1::uuid, 'B' || g, $2::uuid, (array['DRAFT','SCHEDULED','OPEN','CLOSED','PUBLISHED'])[1 + g % 5]::exam_status,
	             case when g % 5 <> 0 then now() + (g || ' hours')::interval end, case when g % 5 <> 0 then now() + (g + 3 || ' hours')::interval end,
	             case when g % 5 <> 0 then 30 end, case when g % 5 = 4 then now() end
	        from generate_series(1, 40) g`, f.course1, f.teacher)
	exec(`insert into users (email, full_name, role) select 'bulk' || g || '.' || $1 || '@example.test', 'SV ' || g, 'STUDENT' from generate_series(1, 2000) g`, uuid.NewString()[:8])
	exec(`insert into exam_attempts (course_id, exam_id, student_id, status, deadline_at, submitted_at, submit_reason, auto_score, graded_at)
	      select $1::uuid, (select id from exams where course_id = $1 and title = 'B' || (1 + u.g % 40)), u.id,
	             case when u.g % 20 = 0 then 'IN_PROGRESS' else 'GRADED' end::attempt_status, now() + interval '1 hour',
	             case when u.g % 20 <> 0 then now() end, case when u.g % 20 <> 0 then 'MANUAL' end::attempt_submit_reason,
	             case when u.g % 20 <> 0 then 5 end, case when u.g % 20 <> 0 then now() end
	        from (select id, row_number() over (order by email) g from users where email like 'bulk%@example.test') u`, f.course1)
	exec(`create temp table att as select row_number() over (order by id) - 1 n, id, exam_id, student_id from exam_attempts where course_id = $1`, f.course1)
	// 20.000 bài nộp trên các lượt đã tạo
	exec(`insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256, status, verdict, tests_version, created_at)
	      select $1::uuid, a.exam_id, a.id, $2::uuid, $3::uuid, a.student_id, case when g % 4 = 0 then 'RUN' else 'SUBMIT' end::submission_kind, 'cpp17', 'x', repeat('c', 64),
	             case when g % 50 = 0 then 'QUEUED' else 'DONE' end::submission_status, case when g % 50 <> 0 then 'AC' end::judge_verdict, 1, now() - (g || ' seconds')::interval
	        from generate_series(1, 20000) g
	        join att a on a.n = g % (select count(*) from att)`, f.course1, f.item2, f.q2)
	exec(`analyze`)

	var tests int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from exam_attempts`).Scan(&tests))
	require.Greater(t, tests, 1900)

	type plan struct{ name, sql string }
	plans := []plan{
		{"câu hỏi theo (course, review_status)", fmt.Sprintf(`select id from question_bank where course_id = '%s' and review_status = 'APPROVED' and archived_at is null order by created_at desc, id desc limit 30`, f.course1)},
		{"câu hỏi theo (course, topic, difficulty, type)", fmt.Sprintf(`select id from question_bank where course_id = '%s' and topic = 'Chủ đề 7' and difficulty = 'EASY' and type = 'MCQ_SINGLE' and archived_at is null`, f.course1)},
		{"lượt làm theo (exam, student)", fmt.Sprintf(`select id from exam_attempts where exam_id = '%s' and student_id = '%s'`, f.exam, f.student)},
		{"khoá chat: lượt đang làm theo student", fmt.Sprintf(`select id from exam_attempts where student_id = '%s' and status = 'IN_PROGRESS' and deadline_at > now()`, f.student)},
		{"tự nộp: (status, deadline_at)", `select id from exam_attempts where status = 'IN_PROGRESS' and deadline_at <= now() + interval '2 hours' limit 100`},
		{"bài nộp lần cuối theo (attempt, item)", fmt.Sprintf(`select id from code_submissions where course_id = '%s' and attempt_id = '%s' and item_id = '%s' and kind = 'SUBMIT' order by created_at desc, id desc limit 1`, f.course1, f.attempt, f.item2)},
		{"hàng chờ chấm", `select id from code_submissions where status = 'QUEUED' and enqueued_at is null and next_attempt_at <= now() order by next_attempt_at, id limit 50`},
		{"bảng điểm theo (course, exam, status)", fmt.Sprintf(`select id, student_id from exam_attempts where course_id = '%s' and exam_id = '%s' and status = 'GRADED' order by student_id limit 30`, f.course1, f.exam)},
		{"bài thi đến hạn mở", `select id from exams where status = 'SCHEDULED' and opens_at <= now() + interval '1 day'`},
		{"bài thi đến hạn đóng", `select id from exams where status in ('SCHEDULED', 'OPEN') and closes_at <= now() + interval '1 day'`},
		{"bài thi theo (course, status)", fmt.Sprintf(`select id from exams where course_id = '%s' and status = 'OPEN' order by opens_at desc, id desc limit 30`, f.course1)},
		{"tra câu đang dùng ở bài nào", fmt.Sprintf(`select exam_id from exam_items where question_id = '%s'`, f.q1)},
		{"so độ giống: bài nộp theo (course, exam, problem)", fmt.Sprintf(`select id from code_submissions where course_id = '%s' and exam_id = '%s' and problem_id = '%s' and kind = 'SUBMIT'`, f.course1, f.exam, f.q2)},
	}
	big := map[string]bool{"question_bank": true, "exam_attempts": true, "code_submissions": true}
	for _, p := range plans {
		var raw []byte
		require.NoError(t, conn.QueryRow(ctx, `explain (format json) `+p.sql).Scan(&raw), p.name)
		var doc []map[string]any
		require.NoError(t, json.Unmarshal(raw, &doc), p.name)
		var seq []string
		var walk func(n map[string]any)
		walk = func(n map[string]any) {
			if n["Node Type"] == "Seq Scan" {
				rel, _ := n["Relation Name"].(string)
				if big[rel] {
					seq = append(seq, rel)
				}
			}
			if kids, ok := n["Plans"].([]any); ok {
				for _, k := range kids {
					walk(k.(map[string]any))
				}
			}
		}
		walk(doc[0]["Plan"].(map[string]any))
		require.Emptyf(t, seq, "%s: Seq Scan trên bảng lớn %v", p.name, seq)
	}
}

// TestExamMigrationDownUp — `down` (về 0) gỡ sạch 13 bảng + 16 enum của 00006 TRƯỚC khi gỡ `courses` / `users` (thứ tự Down đúng) và `up` lại được.
func TestExamMigrationDownUp(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	url := conn.Config().ConnString()
	count := func(q string, args ...any) int {
		var n int
		require.NoError(t, conn.QueryRow(ctx, q, args...).Scan(&n))
		return n
	}
	tables := func() int {
		return count(`select count(*) from information_schema.tables where table_schema='public' and table_name = any($1)`, examTables)
	}
	enums := func() int {
		return count(`select count(*) from pg_type where typtype = 'e' and typname in ('question_type','question_difficulty','question_origin','question_review_status','checker_kind','exam_kind','exam_status','multi_scoring','attempt_status','attempt_submit_reason','submission_kind','submission_status','judge_verdict','exam_event_type','similarity_review_state','appeal_status')`)
	}
	require.Equal(t, 13, tables())
	require.Equal(t, 16, enums())
	require.NoError(t, migrateCmd(ctx, url, "down"))
	require.Equal(t, 0, tables())
	require.Equal(t, 0, enums())
	require.Equal(t, 0, count(`select count(*) from information_schema.tables where table_schema='public' and table_name in ('courses','users')`))
	require.NoError(t, migrateCmd(ctx, url, "up"))
	require.Equal(t, 13, tables())
}
