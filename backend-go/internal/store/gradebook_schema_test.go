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

// gradeFx: hai lớp (crmFx), công thức DRAFT + thành phần + item ở lớp 1, một công thức + thành phần ở lớp 2, hai bài thi.
type gradeFx struct {
	crmFx
	scheme1, scheme2, comp1, comp2, item1, exam1, exam2 uuid.UUID
}

func newGradeFx(t *testing.T, ctx context.Context, conn *pgx.Conn) gradeFx {
	t.Helper()
	f := gradeFx{crmFx: newCrmFx(t, ctx, conn)}
	one := func(dst *uuid.UUID, sql string, args ...any) {
		require.NoError(t, conn.QueryRow(ctx, sql, args...).Scan(dst), sql)
	}
	sch := `insert into grade_schemes (course_id, scheme_no, created_by) values ($1, 1, $2) returning id`
	one(&f.scheme1, sch, f.c1, f.teacher)
	one(&f.scheme2, sch, f.c2, f.teacher)
	comp := `insert into grade_components (course_id, scheme_id, key, name, grp, weight) values ($1, $2, 'midterm', 'Giữa kỳ', 'QT', 20) returning id`
	one(&f.comp1, comp, f.c1, f.scheme1)
	one(&f.comp2, comp, f.c2, f.scheme2)
	one(&f.item1, `insert into grade_items (course_id, scheme_id, component_id, key, name) values ($1, $2, $3, 'midterm', 'Giữa kỳ') returning id`, f.c1, f.scheme1, f.comp1)
	one(&f.exam1, `insert into exams (course_id, title, created_by) values ($1, 'Bài thi 1', $2) returning id`, f.c1, f.teacher)
	one(&f.exam2, `insert into exams (course_id, title, created_by) values ($1, 'Bài thi 2', $2) returning id`, f.c2, f.teacher)
	return f
}

func (f gradeFx) q(sql string) string {
	return strings.NewReplacer("{c1}", f.c1.String(), "{c2}", f.c2.String(), "{t}", f.teacher.String(), "{s1}", f.s1.String(), "{s2}", f.s2.String(),
		"{sc1}", f.scheme1.String(), "{sc2}", f.scheme2.String(), "{k1}", f.comp1.String(), "{k2}", f.comp2.String(), "{i1}", f.item1.String(),
		"{e1}", f.exam1.String(), "{e2}", f.exam2.String()).Replace(sql)
}

// TestSchemaGradebook — US-P6-01 AC1/AC2: cột, enum, không câu ALTER nào trong 00016.
func TestSchemaGradebook(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	want := map[string][]string{
		"grade_schemes":         {"id", "course_id", "scheme_no", "status", "source_document_id", "source_title", "extraction", "extract_job_id", "confirmed_by", "confirmed_at", "created_by", "version", "unclear_count", "created_at", "updated_at"},
		"grade_components":      {"id", "course_id", "scheme_id", "key", "name", "grp", "weight", "position", "citations", "filled_by_user", "created_at", "updated_at"},
		"grade_rules":           {"id", "course_id", "scheme_id", "kind", "applicable", "params", "source_quote", "source_page", "filled_by_user", "created_at", "updated_at"},
		"grade_items":           {"id", "course_id", "scheme_id", "component_id", "key", "name", "max_score", "source", "exam_id", "assignment_id", "published_at", "position", "created_at", "updated_at"},
		"grade_entries":         {"id", "course_id", "item_id", "student_id", "score", "source", "version", "entered_by", "created_at", "updated_at"},
		"gradebooks":            {"course_id", "status", "snapshot_version", "unlock_reason", "unlocked_by", "unlocked_at", "version", "created_at", "updated_at"},
		"final_grade_snapshots": {"id", "course_id", "student_id", "version", "scheme_id", "breakdown", "numeric_grade", "letter", "failed_reason", "locked_at", "locked_by", "unlock_reason", "created_at"},
	}
	for table, cols := range want {
		require.Equal(t, cols, tableCols(t, ctx, conn, table), table)
	}
	require.Equal(t, []string{"DRAFT", "CONFIRMED", "SUPERSEDED"}, enumLabels(t, ctx, conn, "grade_scheme_status"))
	require.Equal(t, []string{"QT", "CK"}, enumLabels(t, ctx, conn, "grade_group"))
	require.Equal(t, []string{"BONUS", "PENALTY", "ROUNDING", "FAIL_CONDITION", "LETTER_SCALE", "MISSING_SCORE"}, enumLabels(t, ctx, conn, "grade_rule_kind"))
	require.Equal(t, []string{"MANUAL", "EXAM", "ASSIGNMENT"}, enumLabels(t, ctx, conn, "grade_item_source"))
	require.Equal(t, []string{"MANUAL", "AI_GRADED", "QUIZ", "IMPORT"}, enumLabels(t, ctx, conn, "grade_entry_source"))
	require.Equal(t, []string{"OPEN", "LOCKED"}, enumLabels(t, ctx, conn, "gradebook_status"))

	b, err := os.ReadFile("../../db/migrations/00016_gradebook.sql")
	require.NoError(t, err)
	for _, ln := range strings.Split(string(b), "\n") {
		require.False(t, strings.HasPrefix(strings.ToUpper(strings.TrimSpace(ln)), "ALTER TABLE"), "00016 chỉ CREATE (D45): %s", ln)
	}
}

// TestGradebookOneDraftOneConfirmed — một DRAFT và một CONFIRMED mỗi lớp; SUPERSEDED không giới hạn; scheme_no duy nhất mỗi lớp.
func TestGradebookOneDraftOneConfirmed(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newGradeFx(t, ctx, conn)
	const unique = "23505"
	ins := func(no int, status string) error {
		extra := ""
		if status != "DRAFT" {
			extra = fmt.Sprintf(", confirmed_by, confirmed_at) values ('%s', %d, '%s', '%s', '%s', now())", f.c1, no, status, f.teacher, f.teacher)
			_, err := conn.Exec(ctx, fmt.Sprintf(`insert into grade_schemes (course_id, scheme_no, status, created_by%s`, extra))
			return err
		}
		_, err := conn.Exec(ctx, `insert into grade_schemes (course_id, scheme_no, status, created_by) values ($1, $2, 'DRAFT', $3)`, f.c1, no, f.teacher)
		return err
	}
	require.Equal(t, unique, sqlState(ins(2, "DRAFT")), "bản DRAFT thứ hai (scheme1 đã là DRAFT)")
	require.NoError(t, ins(2, "CONFIRMED"))
	require.Equal(t, unique, sqlState(ins(3, "CONFIRMED")), "bản CONFIRMED thứ hai")
	require.NoError(t, ins(3, "SUPERSEDED"))
	require.NoError(t, ins(4, "SUPERSEDED"))
	require.Equal(t, unique, sqlState(ins(4, "SUPERSEDED")), "scheme_no trùng")
	// CHECK: DRAFT không có confirmed_at; CONFIRMED / SUPERSEDED bắt buộc có.
	_, err := conn.Exec(ctx, `update grade_schemes set confirmed_at = now(), confirmed_by = $2 where id = $1`, f.scheme1, f.teacher)
	require.Equal(t, "23514", sqlState(err))
}

// TestGradebookChecks — CHECK / UNIQUE từng cột (US-P6-01 AC2).
func TestGradebookChecks(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newGradeFx(t, ctx, conn)
	const check, unique = "23514", "23505"
	comp := func(key string, w int) string {
		return f.q(fmt.Sprintf(`insert into grade_components (course_id, scheme_id, key, name, grp, weight) values ('{c1}', '{sc1}', '%s', 'X', 'QT', %d)`, key, w))
	}
	item := func(cols, vals string) string {
		return f.q(fmt.Sprintf(`insert into grade_items (course_id, scheme_id, component_id, key, name%s) values ('{c1}', '{sc1}', '{k1}', %s)`, cols, vals))
	}
	entry := func(vals string) string {
		return f.q(fmt.Sprintf(`insert into grade_entries (course_id, item_id, student_id, score, entered_by) values ('{c1}', '{i1}', %s)`, vals))
	}
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"trọng số > 100":           {check, comp("w101", 101)},
		"trọng số < 0":             {check, comp("wneg", -1)},
		"khoá thành phần trùng":    {unique, comp("midterm", 10)},
		"khoá thành phần sai dạng": {check, comp("Bad Key", 10)},
		"item max_score = 0":       {check, item(", max_score", `'i0', 'X', 0`)},
		"item khoá trùng":          {unique, item("", `'midterm', 'Giữa kỳ 2'`)},
		"EXAM thiếu exam_id":       {check, item(", source", `'ex0', 'X', 'EXAM'`)},
		"exam_id mà source MANUAL": {check, item(", exam_id", `'ex1', 'X', '{e1}'`)},
		"ASSIGNMENT thiếu id":      {check, item(", source", `'as0', 'X', 'ASSIGNMENT'`)},
		"điểm âm":                  {check, entry(`'{s1}', -0.5, '{t}'`)},
		"sinh viên ngoài lớp":      {"23503", entry(fmt.Sprintf(`'%s', 5, '{t}'`, uuid.New()))},
		"người nhập lạ":            {"23503", entry(fmt.Sprintf(`'{s1}', 5, '%s'`, uuid.New()))},
	})
	_, err := conn.Exec(ctx, entry(`'{s1}', 8.50, '{t}'`))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, entry(`'{s1}', 9, '{t}'`))
	require.Equal(t, unique, sqlState(err), "(item, sinh viên) duy nhất")
	// bài thi chỉ gắn một item mỗi công thức; hai lần cùng exam_id → UNIQUE (scheme_id, exam_id)
	ex := func(key string) string {
		return f.q(fmt.Sprintf(`insert into grade_items (course_id, scheme_id, component_id, key, name, source, exam_id) values ('{c1}', '{sc1}', '{k1}', '%s', 'Thi', 'EXAM', '{e1}')`, key))
	}
	_, err = conn.Exec(ctx, ex("exam_a"))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, ex("exam_b"))
	require.Equal(t, unique, sqlState(err))
}

// TestGradebookCrossCourseFK — item / thành phần / bài thi / sinh viên của lớp khác không gắn được (23503).
func TestGradebookCrossCourseFK(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newGradeFx(t, ctx, conn)
	const fk = "23503"
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"thành phần của lớp 2 trong công thức lớp 1": {fk, f.q(`insert into grade_components (course_id, scheme_id, key, name, grp, weight) values ('{c2}', '{sc1}', 'x', 'X', 'QT', 10)`)},
		"item lớp 1 gắn thành phần lớp 2":            {fk, f.q(`insert into grade_items (course_id, scheme_id, component_id, key, name) values ('{c1}', '{sc1}', '{k2}', 'x', 'X')`)},
		"item lớp 1 gắn công thức lớp 2":             {fk, f.q(`insert into grade_items (course_id, scheme_id, component_id, key, name) values ('{c1}', '{sc2}', '{k1}', 'x', 'X')`)},
		"item lớp 1 nối bài thi lớp 2":               {fk, f.q(`insert into grade_items (course_id, scheme_id, component_id, key, name, source, exam_id) values ('{c1}', '{sc1}', '{k1}', 'x', 'X', 'EXAM', '{e2}')`)},
		"điểm lớp 2 vào item lớp 1":                  {fk, f.q(`insert into grade_entries (course_id, item_id, student_id, score, entered_by) values ('{c2}', '{i1}', '{s1}', 5, '{t}')`)},
		"điểm của sinh viên chưa vào lớp 2":          {fk, f.q(`insert into grade_entries (course_id, item_id, student_id, score, entered_by) values ('{c1}', '{i1}', '` + f.s3.String() + `', 5, '{t}')`)},
		"quy tắc lớp 1 gắn công thức lớp 2":          {fk, f.q(`insert into grade_rules (course_id, scheme_id, kind) values ('{c1}', '{sc2}', 'ROUNDING')`)},
	})
	// luật duy nhất theo kind trừ FAIL_CONDITION
	rule := func(kind string) string {
		return f.q(fmt.Sprintf(`insert into grade_rules (course_id, scheme_id, kind) values ('{c1}', '{sc1}', '%s')`, kind))
	}
	for range 2 {
		_, err := conn.Exec(ctx, rule("FAIL_CONDITION"))
		require.NoError(t, err, "nhiều FAIL_CONDITION")
	}
	_, err := conn.Exec(ctx, rule("ROUNDING"))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, rule("ROUNDING"))
	require.Equal(t, "23505", sqlState(err))
}

// TestIndexesStartWithCourseID_Gradebook — luật 13 cho bảng của 00016 (chỉ mục phức hợp bắt đầu bằng course_id; UNIQUE theo scheme_id / item_id là ràng buộc, không phải chỉ mục truy vấn).
func TestIndexesStartWithCourseID_Gradebook(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	rows, err := conn.Query(ctx, `
		select c.relname, i.relname, a.attname
		  from pg_index x
		  join pg_class c on c.oid = x.indrelid
		  join pg_class i on i.oid = x.indexrelid
		  join pg_attribute a on a.attrelid = c.oid and a.attnum = x.indkey[0]
		 where c.relnamespace = 'public'::regnamespace
		   and c.relname in ('grade_schemes', 'grade_components', 'grade_rules', 'grade_items', 'grade_entries', 'gradebooks', 'final_grade_snapshots')
		   and x.indnatts > 1 and not x.indisunique`)
	require.NoError(t, err)
	defer rows.Close()
	seen := 0
	for rows.Next() {
		var table, index, first string
		require.NoError(t, rows.Scan(&table, &index, &first))
		require.Equal(t, "course_id", first, "%s.%s", table, index)
		seen++
	}
	require.NoError(t, rows.Err())
	require.GreaterOrEqual(t, seen, 8)
}

// snapshotFx chèn một snapshot hợp lệ cho (s1, version).
func (f gradeFx) snapshot(t *testing.T, ctx context.Context, conn *pgx.Conn, version int) error {
	t.Helper()
	_, err := conn.Exec(ctx, f.q(`insert into final_grade_snapshots (course_id, student_id, version, scheme_id, breakdown, numeric_grade, letter, locked_by)
		values ('{c1}', '{s1}', $1, '{sc1}', '{}', 9.5, 'A+', '{t}')`), version)
	return err
}

// TestSnapshotImmutable / TruncateBlocked / UniqueVersion / VersionIndex — US-P6-01 AC3.
func TestSnapshotImmutable(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newGradeFx(t, ctx, conn)
	require.NoError(t, f.snapshot(t, ctx, conn, 1))
	_, err := conn.Exec(ctx, `update final_grade_snapshots set letter = 'F'`)
	require.Equal(t, "P0001", sqlState(err))
	require.ErrorContains(t, err, "snapshot is immutable")
	_, err = conn.Exec(ctx, `delete from final_grade_snapshots`)
	require.Equal(t, "P0001", sqlState(err))
	require.NoError(t, f.snapshot(t, ctx, conn, 2), "INSERT hợp lệ vẫn chạy")
}

func TestSnapshotTruncateBlocked(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newGradeFx(t, ctx, conn)
	require.NoError(t, f.snapshot(t, ctx, conn, 1))
	_, err := conn.Exec(ctx, `truncate final_grade_snapshots`)
	require.Equal(t, "P0001", sqlState(err))
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from final_grade_snapshots`).Scan(&n))
	require.Equal(t, 1, n)
}

func TestSnapshotUniqueVersion(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newGradeFx(t, ctx, conn)
	require.NoError(t, f.snapshot(t, ctx, conn, 1))
	require.Equal(t, "23505", sqlState(f.snapshot(t, ctx, conn, 1)))
}

func TestSnapshotVersionIndex(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newGradeFx(t, ctx, conn)
	require.NoError(t, f.snapshot(t, ctx, conn, 1))
	_, err := conn.Exec(ctx, `set enable_seqscan = off`)
	require.NoError(t, err)
	plan := explain(t, ctx, conn, `explain (costs off) select student_id from final_grade_snapshots where course_id = $1 and version = 1 order by student_id`, f.c1)
	require.Contains(t, plan, "final_grade_snapshots_version_idx")
	require.NotContains(t, plan, "Seq Scan")
}

// TestMigration00016UpDownUp — AC1: up → down sạch (bảng, enum, hàm) → up lại.
func TestMigration00016UpDownUp(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	url := conn.Config().ConnString()
	count := func(q string) int {
		var n int
		require.NoError(t, conn.QueryRow(ctx, q).Scan(&n))
		return n
	}
	tables := func() int {
		return count(`select count(*) from information_schema.tables where table_schema='public' and table_name in ('grade_schemes','grade_components','grade_rules','grade_items','grade_entries','gradebooks','final_grade_snapshots')`)
	}
	enums := func() int {
		return count(`select count(*) from pg_type where typtype = 'e' and typname in ('grade_scheme_status','grade_group','grade_rule_kind','grade_item_source','grade_entry_source','gradebook_status')`)
	}
	require.Equal(t, 7, tables())
	require.Equal(t, 6, enums())
	require.NoError(t, migrateCmd(ctx, url, "down")) // 00017 (P7) đứng trên 00016: gỡ nó trước (proposals sprint 9 #7)
	require.NoError(t, migrateCmd(ctx, url, "down"))
	require.Equal(t, 0, tables())
	require.Equal(t, 0, enums())
	require.Equal(t, 0, count(`select count(*) from pg_proc where proname = 'final_grade_snapshots_immutable'`))
	require.NoError(t, migrateCmd(ctx, url, "up"))
	require.NoError(t, migrateCmd(ctx, url, "up"))
	require.Equal(t, 7, tables())
}
