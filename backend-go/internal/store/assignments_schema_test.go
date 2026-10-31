package store_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// asgFx: gradeFx + một bài tập ở mỗi lớp, một bài nộp MATCHED của s1 ở lớp 1 và một kết quả chấm của bài nộp đó.
type asgFx struct {
	gradeFx
	a1, a2, sub1, res1 uuid.UUID
}

func newAsgFx(t *testing.T, ctx context.Context, conn *pgx.Conn) asgFx {
	t.Helper()
	f := asgFx{gradeFx: newGradeFx(t, ctx, conn)}
	one := func(dst *uuid.UUID, sql string, args ...any) {
		require.NoError(t, conn.QueryRow(ctx, sql, args...).Scan(dst), sql)
	}
	a := `insert into assignments (course_id, title, due_at, submission_code, created_by) values ($1, 'BT', now() + interval '7 days', 'BT03', $2) returning id`
	one(&f.a1, a, f.c1, f.teacher)
	one(&f.a2, a, f.c2, f.teacher)
	one(&f.sub1, f.q(`insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method)
		values ('{c1}', $1, '{s1}', 'IN_APP', 1, 'k/1', 'a.pdf', 10, 'application/pdf', 'abc', 'MATCHED', 'IN_APP') returning id`), f.a1)
	one(&f.res1, `insert into grading_results (course_id, submission_id) values ($1, $2) returning id`, f.c1, f.sub1)
	return f
}

func (f asgFx) q(sql string) string {
	return strings.NewReplacer("{a1}", f.a1.String(), "{a2}", f.a2.String(), "{sub1}", f.sub1.String(), "{res1}", f.res1.String()).Replace(f.gradeFx.q(sql))
}

var asgTables = []string{"assignments", "submissions", "grading_results", "grade_appeals", "mail_inbound", "integration_configs"}

// TestSchemaAssignments — US-P7-01 AC1/AC2: đủ cột, đúng hai ALTER, không ADD COLUMN, mặc định id là uuidv7().
func TestSchemaAssignments(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	want := map[string][]string{
		"assignments":         {"id", "course_id", "title", "description", "type", "max_score", "due_at", "allow_late", "late_penalty_per_day", "late_max_days", "appeal_days", "channels", "submission_code", "reference_answer", "rubric", "teams_class_id", "teams_assignment_id", "published_at", "created_by", "version", "created_at", "updated_at"},
		"submissions":         {"id", "course_id", "assignment_id", "student_id", "candidate_student_id", "source", "version", "blob_key", "file_name", "file_size", "file_mime", "sha256", "extracted_text", "extracted_at", "extract_status", "external_ref", "external_ts", "from_email", "subject", "sender_verified", "match_status", "match_method", "unmatched_reason", "dismissed_at", "dismissed_by", "dismissed_reason", "submitted_at", "received_at", "created_at", "updated_at"},
		"grading_results":     {"id", "course_id", "submission_id", "round", "run_state", "status", "attempts", "not_before", "lease_owner", "lease_until", "error_code", "criteria", "overall_feedback", "total", "ai_scores", "ai_total_before_late", "late_days", "late_penalty", "late_penalty_manual", "llm_reported_total", "flags", "needs_review", "confidence", "pass_detail", "deviation", "teacher_changed_criteria", "reviewed_by", "reviewed_at", "published_by", "published_at", "version", "created_at", "updated_at"},
		"grade_appeals":       {"id", "course_id", "assignment_id", "student_id", "submission_id", "result_id", "criteria_ids", "reason", "status", "score_before", "score_after", "response", "decided_by", "decided_at", "version", "created_at", "updated_at"},
		"mail_inbound":        {"id", "course_id", "uidvalidity", "message_uid", "message_id", "from_email", "subject", "received_at", "status", "reason", "submission_id", "created_at"},
		"integration_configs": {"id", "course_id", "kind", "config_public", "config_enc", "enabled", "state", "last_ok_at", "last_error", "version", "created_at", "updated_at"},
	}
	for table, cols := range want {
		require.Equal(t, cols, tableCols(t, ctx, conn, table), table)
	}
	b, err := os.ReadFile("../../db/migrations/00017_assignments.sql")
	require.NoError(t, err)
	var alters []string
	for _, ln := range strings.Split(string(b), "\n") {
		require.NotRegexp(t, `(?i)ALTER TABLE .* ADD COLUMN`, ln)
		if strings.HasPrefix(ln, "ALTER ") {
			alters = append(alters, ln[:strings.Index(ln, " ADD ")])
		}
	}
	require.Equal(t, []string{"ALTER TABLE grade_items", "ALTER TYPE reminder_source"}, alters)
	require.Contains(t, enumLabels(t, ctx, conn, "reminder_source"), "ASSIGNMENT")
}

// TestUuidv7Defaults — mọi bảng của 00017 có `id` mặc định uuidv7() (TLR-11).
func TestUuidv7Defaults(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	for _, tb := range asgTables {
		var def string
		require.NoError(t, conn.QueryRow(ctx, `select column_default from information_schema.columns where table_schema='public' and table_name=$1 and column_name='id'`, tb).Scan(&def), tb)
		require.Equal(t, "uuidv7()", def, tb)
	}
}

// TestMigration00017Transactional — tệp không có NO TRANSACTION và up → down → up sạch (giá trị enum còn lại, bảng đi hết).
func TestMigration00017Transactional(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../../db/migrations/00017_assignments.sql")
	require.NoError(t, err)
	require.NotContains(t, strings.ToUpper(string(b)), "NO TRANSACTION")
	ctx, conn := connect(t)
	url := conn.Config().ConnString()
	tables := func() (n int) {
		require.NoError(t, conn.QueryRow(ctx, `select count(*) from information_schema.tables where table_schema='public' and table_name = any($1)`, asgTables).Scan(&n))
		return n
	}
	require.Equal(t, 6, tables())
	require.NoError(t, migrateCmd(ctx, url, "down"))
	require.Zero(t, tables())
	var fk int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_constraint where conname = 'grade_items_assignment_fkey'`).Scan(&fk))
	require.Zero(t, fk)
	require.NoError(t, migrateCmd(ctx, url, "up"))
	require.Equal(t, 6, tables())
}

// TestAssignmentsChecks — US-P7-01 AC2/AC5: vi phạm từng ràng buộc của assignments cho đúng mã.
func TestAssignmentsChecks(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	const check, unique = "23514", "23505"
	ins := func(set string) string {
		return f.q(fmt.Sprintf(`insert into assignments (course_id, title, due_at, submission_code, created_by%s)`, set))
	}
	row := func(code string, extraCols, extraVals string) string {
		return ins(extraCols) + f.q(fmt.Sprintf(` values ('{c1}', 'X', now(), '%s', '{t}'%s)`, code, extraVals))
	}
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"mã bài trùng trong lớp":            {unique, row("BT03", "", "")},
		"mã bài sai dạng":                   {check, row("bt-1", "", "")},
		"max_score = 0":                     {check, row("A01", ", max_score", ", 0")},
		"max_score > 100":                   {check, row("A02", ", max_score", ", 100.25")},
		"max_score không bội 0,25 (TLR-16)": {check, row("A03", ", max_score", ", 7.3")},
		"trừ muộn âm":                       {check, row("A04", ", late_penalty_per_day", ", -1")},
		"late_max_days > 14":                {check, row("A05", ", late_max_days", ", 15")},
		"cho nộp muộn mà late_max_days = 0": {check, row("A06", ", allow_late, late_max_days", ", true, 0")},
		"appeal_days > 30":                  {check, row("A07", ", appeal_days", ", 31")},
		"kênh rỗng":                         {check, row("A08", ", channels", ", '{}'")},
		"kênh lạ":                           {check, row("A09", ", channels", ", '{IN_APP,FAX}'")},
		"TEAMS thiếu lớp Teams":             {check, row("A10", ", channels", ", '{TEAMS}'")},
		"có lớp Teams mà không có TEAMS":    {check, row("A11", ", teams_class_id, teams_assignment_id", ", 'c', 'a'")},
		"thiếu teams_assignment_id":         {check, row("A12", ", channels, teams_class_id", ", '{TEAMS}', 'c'")},
		"loại bài lạ":                       {check, row("A13", ", type", ", 'ORAL'")},
		"tiêu đề rỗng":                      {check, f.q(`insert into assignments (course_id, title, due_at, submission_code, created_by) values ('{c1}', '  ', now(), 'A14', '{t}')`)},
	})
	for _, ok := range []string{row("B01", ", max_score, allow_late, late_max_days, late_penalty_per_day", ", 7.25, true, 3, 0.5"), row("B02", ", channels, teams_class_id, teams_assignment_id", ", '{IN_APP,TEAMS}', 'c', 'a'")} {
		_, err := conn.Exec(ctx, ok)
		require.NoError(t, err)
	}
	// cùng mã ở lớp khác là hợp lệ
	_, err := conn.Exec(ctx, f.q(`insert into assignments (course_id, title, due_at, submission_code, created_by) values ('{c2}', 'X', now(), 'BT03', '{t}')`))
	require.Error(t, err, "unique(course_id, code): a2 đã dùng BT03 ở lớp 2")
}

func (f asgFx) sub(cols, vals string) string {
	return f.q(fmt.Sprintf(`insert into submissions (course_id, source, blob_key, file_name, file_size, file_mime, sha256, match_status%s) values ('{c1}', 'IN_APP', 'k/x', 'a.pdf', 10, 'application/pdf', 'x', %s)`, cols, vals))
}

// TestSubmissionsMatchedChecks — MATCHED ⇒ đủ bốn cột; UNMATCHED ⇒ có lý do; phiên bản duy nhất; ngoại tệp bất biến.
func TestSubmissionsMatchedChecks(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	const check, unique, fk = "23514", "23505", "23503"
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"MATCHED thiếu student_id":              {check, f.sub(", assignment_id, match_method, version", `'MATCHED', '{a1}', 'IN_APP', 1`)},
		"MATCHED thiếu match_method":            {check, f.sub(", assignment_id, student_id, version", `'MATCHED', '{a1}', '{s1}', 1`)},
		"MATCHED thiếu version":                 {check, f.sub(", assignment_id, student_id, match_method", `'MATCHED', '{a1}', '{s1}', 'IN_APP'`)},
		"hai phiên bản trùng":                   {unique, f.sub(", assignment_id, student_id, match_method, version", `'MATCHED', '{a1}', '{s1}', 'IN_APP', 1`)},
		"UNMATCHED thiếu lý do":                 {check, f.sub("", `'UNMATCHED'`)},
		"UNMATCHED lý do lạ":                    {check, f.sub(", unmatched_reason", `'UNMATCHED', 'LY_DO_LA'`)},
		"MATCHED mà có lý do":                   {check, f.sub(", assignment_id, student_id, match_method, version, unmatched_reason", `'MATCHED', '{a1}', '{s2}', 'IN_APP', 1, 'NO_STUDENT_SIGNAL'`)},
		"bỏ qua bài đã khớp":                    {check, f.sub(", assignment_id, student_id, match_method, version, dismissed_at", `'MATCHED', '{a1}', '{s2}', 'IN_APP', 1, now()`)},
		"nguồn lạ":                              {check, strings.Replace(f.sub(", unmatched_reason", `'UNMATCHED', 'NO_STUDENT_SIGNAL'`), "'IN_APP'", "'FAX'", 1)},
		"tệp 0 byte":                            {check, strings.Replace(f.sub(", unmatched_reason", `'UNMATCHED', 'NO_STUDENT_SIGNAL'`), "'k/x', 'a.pdf', 10", "'k/x', 'a.pdf', 0", 1)},
		"tệp > 20 MiB":                          {check, strings.Replace(f.sub(", unmatched_reason", `'UNMATCHED', 'NO_STUDENT_SIGNAL'`), "'k/x', 'a.pdf', 10", "'k/x', 'a.pdf', 20971521", 1)},
		"bài tập của lớp 2 trong bài nộp lớp 1": {fk, f.sub(", assignment_id, student_id, match_method, version", `'MATCHED', '{a2}', '{s2}', 'IN_APP', 1`)},
		"sinh viên chưa vào lớp 1":              {fk, f.sub(", assignment_id, student_id, match_method, version", `'MATCHED', '{a1}', '`+f.s3.String()+`', 'IN_APP', 1`)},
	})
	_, err := conn.Exec(ctx, f.sub(", unmatched_reason", `'UNMATCHED', 'NO_STUDENT_SIGNAL'`))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, f.sub(", assignment_id, student_id, match_method, version", `'MATCHED', '{a1}', '{s1}', 'IN_APP', 2`))
	require.NoError(t, err)
	// khoá ngoài ref: (nguồn, external_ref, external_ts) duy nhất
	ext := f.sub(", unmatched_reason, external_ref, external_ts", `'UNMATCHED', 'NO_STUDENT_SIGNAL', 'g1', '2026-10-30T00:00:00Z'`)
	_, err = conn.Exec(ctx, ext)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, ext)
	require.Equal(t, unique, sqlState(err))
	// bất biến: blob_key, sha256, extracted_text (chỉ NULL → giá trị một lần)
	_, err = conn.Exec(ctx, f.q(`update submissions set extracted_text = 'abc' where id = '{sub1}'`))
	require.NoError(t, err)
	for _, u := range []string{`blob_key = 'k/2'`, `sha256 = 'zzz'`, `extracted_text = 'khác'`, `extracted_text = NULL`} {
		_, err = conn.Exec(ctx, f.q(`update submissions set `+u+` where id = '{sub1}'`))
		require.Equal(t, "P0001", sqlState(err), u)
	}
	_, err = conn.Exec(ctx, f.q(`update submissions set extract_status = 'OK', extracted_at = now() where id = '{sub1}'`))
	require.NoError(t, err)
}

// TestGradingResultsChecks — status ≠ DRAFT ⇒ run_state DONE; PUBLISHED ⇔ published_at; cờ ngoài danh mục; cột sinh needs_review.
func TestGradingResultsChecks(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	const check, unique = "23514", "23505"
	upd := func(set string) string { return f.q(`update grading_results set ` + set + ` where id = '{res1}'`) }
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"REVIEWED khi QUEUED":            {check, upd(`status = 'REVIEWED'`)},
		"PUBLISHED thiếu published_at":   {check, upd(`status = 'PUBLISHED', run_state = 'DONE'`)},
		"published_at mà chưa PUBLISHED": {check, upd(`published_at = now()`)},
		"cờ lạ":                          {check, upd(`flags = ARRAY['KHONG_CO']`)},
		"run_state lạ":                   {check, upd(`run_state = 'PAUSED'`)},
		"vòng chấm trùng":                {unique, f.q(`insert into grading_results (course_id, submission_id, round) values ('{c1}', '{sub1}', 1)`)},
	})
	var nr bool
	require.NoError(t, conn.QueryRow(ctx, f.q(`select needs_review from grading_results where id = '{res1}'`)).Scan(&nr))
	require.False(t, nr)
	_, err := conn.Exec(ctx, upd(`flags = ARRAY['SCORE_ADJUSTED']`))
	require.NoError(t, err)
	require.NoError(t, conn.QueryRow(ctx, f.q(`select needs_review from grading_results where id = '{res1}'`)).Scan(&nr))
	require.False(t, nr, "SCORE_ADJUSTED chỉ hiển thị")
	_, err = conn.Exec(ctx, upd(`flags = ARRAY['INJECTION_SUSPECTED']`))
	require.NoError(t, err)
	require.NoError(t, conn.QueryRow(ctx, f.q(`select needs_review from grading_results where id = '{res1}'`)).Scan(&nr))
	require.True(t, nr)
	_, err = conn.Exec(ctx, upd(`run_state = 'DONE', status = 'REVIEWED'`))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, upd(`status = 'PUBLISHED', published_at = now()`))
	require.NoError(t, err)
}

// TestAiScoresImmutable / FirstWriteAllowed — US-P7-01 AC3.
func TestAiScoresImmutable(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	_, err := conn.Exec(ctx, f.q(`update grading_results set ai_scores = '{"c1":"2.00"}', ai_total_before_late = 2 where id = '{res1}'`))
	require.NoError(t, err)
	for _, u := range []string{`ai_scores = '{"c1":"3.00"}'`, `ai_scores = NULL`, `ai_total_before_late = 3`, `ai_total_before_late = NULL`} {
		_, err = conn.Exec(ctx, f.q(`update grading_results set `+u+` where id = '{res1}'`))
		require.Equal(t, "P0001", sqlState(err), u)
		require.Contains(t, err.Error(), "ai draft is immutable")
	}
	_, err = conn.Exec(ctx, f.q(`update grading_results set criteria = '[{"id":"c1","score":"3.00"}]', total = 3, run_state = 'DONE', status = 'REVIEWED' where id = '{res1}'`))
	require.NoError(t, err, "criteria / total / status sửa được")
}

func TestAiScoresFirstWriteAllowed(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	_, err := conn.Exec(ctx, f.q(`update grading_results set ai_scores = '{"c1":"2.00"}', ai_total_before_late = 2 where id = '{res1}'`))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, f.q(`update grading_results set ai_scores = '{"c1":"2.00"}' where id = '{res1}'`))
	require.NoError(t, err, "ghi lại cùng giá trị không phải thay đổi")
}

// TestAppealsUnique — một lần mỗi bài (course, assignment, student); OPEN ⇔ chưa quyết.
func TestAppealsUnique(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	ap := func(status, decided string) string {
		return f.q(fmt.Sprintf(`insert into grade_appeals (course_id, assignment_id, student_id, submission_id, result_id, criteria_ids, reason, status, score_before%s) values ('{c1}', '{a1}', '{s1}', '{sub1}', '{res1}', ARRAY['c1'], 'lý do', '%s', 5%s)`,
			map[bool]string{true: ", decided_at", false: ""}[decided != ""], status, map[bool]string{true: ", now()", false: ""}[decided != ""]))
	}
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"OPEN mà đã có decided_at":  {"23514", ap("OPEN", "x")},
		"UPHELD mà chưa decided_at": {"23514", ap("UPHELD", "")},
		"trạng thái lạ":             {"23514", ap("DONE", "")},
	})
	_, err := conn.Exec(ctx, ap("OPEN", ""))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, ap("OPEN", ""))
	require.Equal(t, "23505", sqlState(err), "một lần mỗi bài")
}

// TestAssignmentsCrossCourseFK — bài nộp / kết quả / phúc khảo / thư vào của lớp này không gắn được vào lớp khác (23503).
func TestAssignmentsCrossCourseFK(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	const fk = "23503"
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"kết quả lớp 2 cho bài nộp lớp 1":   {fk, f.q(`insert into grading_results (course_id, submission_id) values ('{c2}', '{sub1}')`)},
		"phúc khảo lớp 2 cho bài tập lớp 1": {fk, f.q(`insert into grade_appeals (course_id, assignment_id, student_id, submission_id, result_id, criteria_ids, reason, score_before) values ('{c2}', '{a1}', '{s1}', '{sub1}', '{res1}', ARRAY['c1'], 'x', 1)`)},
		"thư vào lớp 2 trỏ bài nộp lớp 1":   {fk, f.q(`insert into mail_inbound (course_id, uidvalidity, message_uid, from_email, received_at, status, submission_id) values ('{c2}', 1, 1, 'a@b.c', now(), 'ACCEPTED', '{sub1}')`)},
		"ứng viên chưa vào lớp":             {fk, f.sub(", candidate_student_id, unmatched_reason", `'UNMATCHED', '`+f.s3.String()+`', 'NO_STUDENT_SIGNAL'`)},
	})
	_, err := conn.Exec(ctx, f.q(`insert into mail_inbound (course_id, uidvalidity, message_uid, from_email, received_at, status) values ('{c1}', 1, 1, 'a@b.c', now(), 'IGNORED')`))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, f.q(`insert into mail_inbound (course_id, uidvalidity, message_uid, from_email, received_at, status) values ('{c1}', 1, 1, 'a@b.c', now(), 'IGNORED')`))
	require.Equal(t, "23505", sqlState(err))
	_, err = conn.Exec(ctx, f.q(`insert into integration_configs (course_id, kind) values ('{c1}', 'IMAP')`))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, f.q(`insert into integration_configs (course_id, kind) values ('{c1}', 'IMAP')`))
	require.Equal(t, "23505", sqlState(err))
	_, err = conn.Exec(ctx, f.q(`insert into integration_configs (course_id, kind) values ('{c1}', 'FAX')`))
	require.Equal(t, "23514", sqlState(err))
}

// TestGradeItemAssignmentFK / DeleteRestricted — US-P7-01 AC4.
func TestGradeItemAssignmentFK(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	item := func(a string) string {
		return f.q(`insert into grade_items (course_id, scheme_id, component_id, key, name, source, assignment_id) values ('{c1}', '{sc1}', '{k1}', 'bt', 'BT', 'ASSIGNMENT', '` + a + `')`)
	}
	_, err := conn.Exec(ctx, item(f.a2.String()))
	require.Equal(t, "23503", sqlState(err), "item lớp 1 không trỏ bài tập lớp 2")
	_, err = conn.Exec(ctx, item(f.a1.String()))
	require.NoError(t, err)
}

func TestAssignmentDeleteRestrictedByGradeItem(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newAsgFx(t, ctx, conn)
	// Bài KHÔNG có bài nộp: nếu lỗi 23503 đến từ đâu đó khác `grade_items` thì test không bắt được đột biến ON DELETE CASCADE (QC BUG-P7-01-1).
	var a3 uuid.UUID
	require.NoError(t, conn.QueryRow(ctx, `insert into assignments (course_id, title, due_at, submission_code, created_by) values ($1, 'BT9', now() + interval '7 days', 'BT09', $2) returning id`, f.c1, f.teacher).Scan(&a3))
	_, err := conn.Exec(ctx, f.q(`insert into grade_items (course_id, scheme_id, component_id, key, name, source, assignment_id) values ('{c1}', '{sc1}', '{k1}', 'bt', 'BT', 'ASSIGNMENT', '`+a3.String()+`')`))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `delete from assignments where id = $1`, a3)
	require.Equal(t, "23503", sqlState(err), "item sổ điểm chặn xoá")
	_, err = conn.Exec(ctx, f.q(`delete from assignments where id = '{a1}'`))
	require.Equal(t, "23503", sqlState(err), "bài nộp chặn xoá")
	_, err = conn.Exec(ctx, f.q(`delete from assignments where id = '{a2}'`))
	require.NoError(t, err, "bài tập không ai trỏ tới xoá được")
}

// TestIndexesStartWithCourseID_Assignments — luật 13: mọi chỉ mục nhiều cột không-duy-nhất bắt đầu bằng course_id; ngoại lệ có tên duy nhất.
func TestIndexesStartWithCourseID_Assignments(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	rows, err := conn.Query(ctx, `
		select c.relname, i.relname, a.attname, x.indisunique
		  from pg_index x
		  join pg_class c on c.oid = x.indrelid
		  join pg_class i on i.oid = x.indexrelid
		  join pg_attribute a on a.attrelid = c.oid and a.attnum = x.indkey[0]
		 where c.relnamespace = 'public'::regnamespace and c.relname = any($1) and not x.indisprimary`, asgTables)
	require.NoError(t, err)
	defer rows.Close()
	seen := 0
	for rows.Next() {
		var table, index, first string
		var uniq bool
		require.NoError(t, rows.Scan(&table, &index, &first, &uniq))
		if index == "grading_results_lease_idx" {
			require.Equal(t, "run_state", first)
			continue
		}
		require.Equal(t, "course_id", first, "%s.%s", table, index)
		seen++
	}
	require.NoError(t, rows.Err())
	require.GreaterOrEqual(t, seen, 15)
}

// TestMigration00017Shape — một giao dịch, hai ALTER, mọi FK phức hợp có course_id (trừ users / courses).
func TestMigration00017Shape(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	rows, err := conn.Query(ctx, `
		select conrelid::regclass::text, conname, pg_get_constraintdef(oid)
		  from pg_constraint
		 where contype = 'f' and conrelid::regclass::text = any($1)`, asgTables)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var table, name, def string
		require.NoError(t, rows.Scan(&table, &name, &def))
		if strings.Contains(def, "REFERENCES users(") || strings.Contains(def, "REFERENCES courses(") {
			continue
		}
		require.True(t, strings.HasPrefix(def, "FOREIGN KEY (course_id, "), "%s.%s: %s", table, name, def)
		require.NotContains(t, def, "CASCADE", name)
	}
	require.NoError(t, rows.Err())
}
