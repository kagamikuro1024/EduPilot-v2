package grade_test

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
)

// sheet: lớp có công thức CONFIRMED (weekly 40 QT + final 60 CK, làm tròn 0,5), một sinh viên chưa có điểm.
func (d *tdb) sheet() (course, gv, student uuid.UUID, weekly, final uuid.UUID) {
	d.t.Helper()
	gv = d.user("TEACHER", "GV")
	course = d.course(gv, "L1")
	doc := d.newDraft(course, gv, 1, nil)
	d.confirmDoc(course, gv, doc)
	student = d.student(course, "An", "")
	items := d.itemIDs(course, doc.ID)
	return course, gv, student, items["weekly"], items["final"]
}

func put(d *tdb, gv, course, student, item uuid.UUID, score *string, version *int) (grade.PutEntryOut, error) {
	return d.svc().PutEntry(d.ctx, gv, course, grade.PutEntryInput{ItemID: item, StudentID: student, Score: score, Version: version})
}

func s(v string) *string { return &v }
func n(v int) *int       { return &v }

func TestPutEntryOK(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, _ := d.sheet()
	out, err := put(d, gv, c, st, weekly, s("8,5"), nil)
	require.NoError(t, err)
	require.Equal(t, "8.5", out.Entry.Score)
	require.Equal(t, 1, out.Entry.Version)
	out, err = put(d, gv, c, st, weekly, s("9"), n(1))
	require.NoError(t, err)
	require.Equal(t, "9", out.Entry.Score)
	require.Equal(t, 2, out.Entry.Version)
	out, err = put(d, gv, c, st, weekly, s(" 7.25 "), n(2))
	require.NoError(t, err)
	require.Equal(t, "7.25", out.Entry.Score)
}

func TestPutEntryOutOfRange(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, _ := d.sheet()
	_, err := put(d, gv, c, st, weekly, s("11"), nil)
	status, code := apiCode(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, status)
	require.Equal(t, apierr.ScoreOutOfRange, code)
	require.Zero(t, d.countRows(`select count(*) from grade_entries`))
}

// TLR-7: quá 2 chữ số thập phân là lỗi, KHÔNG tự làm tròn.
func TestPutEntryPrecision(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, _ := d.sheet()
	for _, bad := range []string{"8,333", "8.999", "7.123456"} {
		_, err := put(d, gv, c, st, weekly, s(bad), nil)
		status, code := apiCode(t, err)
		require.Equal(t, http.StatusUnprocessableEntity, status, bad)
		require.Equal(t, apierr.ScorePrecision, code, bad)
	}
	for _, bad := range []string{"abc", "-1", "1e1", "8,5,1"} {
		_, err := put(d, gv, c, st, weekly, s(bad), nil)
		status, code := apiCode(t, err)
		require.Equal(t, http.StatusUnprocessableEntity, status, bad)
		require.Equal(t, apierr.ValidationFailed, code, bad)
	}
	require.Zero(t, d.countRows(`select count(*) from grade_entries`))
}

func TestPutEntryClear(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, _ := d.sheet()
	_, err := put(d, gv, c, st, weekly, s("8"), nil)
	require.NoError(t, err)
	out, err := put(d, gv, c, st, weekly, nil, n(1))
	require.NoError(t, err)
	require.Nil(t, out.Entry)
	require.Zero(t, d.countRows(`select count(*) from grade_entries`))
	_, err = put(d, gv, c, st, weekly, s("8"), nil) // ô trống nhập lại
	require.NoError(t, err)
	_, err = put(d, gv, c, st, weekly, nil, n(5)) // xoá cũng kiểm version
	_, code := apiCode(t, err)
	require.Equal(t, apierr.VersionConflict, code)
}

func TestPutEntryExamItemRejected(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, final := d.sheet()
	ex := d.exam(c, gv, "PUBLISHED", "10")
	d.exec(`update grade_items set source = 'EXAM', exam_id = $2 where id = $1`, final, ex)
	_, err := put(d, gv, c, st, final, s("7"), nil)
	status, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, apierr.ItemNotEditable, code)
	_, err = put(d, gv, c, st, weekly, s("7"), nil)
	require.NoError(t, err, "cột nhập tay vẫn sửa được")
}

// AC4: phản hồi `PUT` có `Compute` của hàng; thiếu điểm → INCOMPLETE, đủ → COMPLETE đúng số.
func TestPutEntryReturnsRowResult(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, final := d.sheet()
	out, err := put(d, gv, c, st, weekly, s("8"), nil)
	require.NoError(t, err)
	require.Equal(t, "INCOMPLETE", out.RowResult.Status)
	require.Equal(t, []string{"final"}, out.RowResult.Missing)
	out, err = put(d, gv, c, st, final, s("6"), nil)
	require.NoError(t, err)
	require.Equal(t, "COMPLETE", out.RowResult.Status)
	require.Equal(t, "7", out.RowResult.HP, "0,4×8 + 0,6×6 = 6,8 → làm tròn 0,5 = 7")
	require.Equal(t, "A", out.RowResult.Letter)
}

func TestPutEntryUnder300ms(t *testing.T) {
	testutil.SkipTiming(t)
	d := newTDB(t)
	c, gv, st, weekly, _ := d.sheet()
	var ds []time.Duration
	var ver *int
	for i := range 20 {
		t0 := time.Now()
		out, err := put(d, gv, c, st, weekly, s([]string{"5", "6"}[i%2]), ver)
		require.NoError(t, err)
		ds = append(ds, time.Since(t0))
		ver = &out.Entry.Version
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
	require.Less(t, ds[18], 300*time.Millisecond, "p95 ≤ 300 ms")
}

// AC5: thân 409 có giá trị hiện tại, version và tên người sửa.
func TestPutEntryConflictBody(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, _ := d.sheet()
	other := d.user("TEACHER", "Thầy Nam")
	_, err := put(d, gv, c, st, weekly, s("7,5"), nil)
	require.NoError(t, err)
	_, err = put(d, other, c, st, weekly, s("6"), n(1))
	require.NoError(t, err)
	_, err = put(d, gv, c, st, weekly, s("9"), n(1)) // version cũ
	var ae *apierr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, http.StatusConflict, ae.Status)
	require.Equal(t, apierr.VersionConflict, ae.Code)
	det := ae.Details.(map[string]any)
	cur := det["current"].(map[string]any)
	require.Equal(t, "6", cur["score"])
	require.EqualValues(t, 2, cur["version"])
	require.Equal(t, map[string]any{"name": "Thầy Nam"}, cur["updated_by"])
	require.NotNil(t, cur["updated_at"])
}

func TestPutEntryLocked(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, _ := d.sheet()
	d.exec(`update gradebooks set status = 'LOCKED', snapshot_version = 1 where course_id = $1`, c)
	_, err := put(d, gv, c, st, weekly, s("8"), nil)
	status, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, apierr.GradebookLocked, code)
	require.Zero(t, d.countRows(`select count(*) from grade_entries`))
}

func TestPutEntryAuditRow(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, _ := d.sheet()
	_, err := put(d, gv, c, st, weekly, s("8"), nil)
	require.NoError(t, err)
	_, err = put(d, gv, c, st, weekly, s("9"), n(1))
	require.NoError(t, err)
	var before, after string
	var actor uuid.UUID
	d.one(&before, `select before::text from audit_log where entity = 'grade_entries' and action = 'set' order by created_at desc, id desc limit 1`)
	d.one(&after, `select after::text from audit_log where entity = 'grade_entries' and action = 'set' order by created_at desc, id desc limit 1`)
	d.one(&actor, `select actor_id from audit_log where entity = 'grade_entries' order by created_at desc, id desc limit 1`)
	require.Equal(t, gv, actor)
	require.Contains(t, before, `"score": "8"`)
	require.Contains(t, after, `"score": "9"`)
	require.Equal(t, 2, d.countRows(`select count(*) from audit_log where entity = 'grade_entries' and action = 'set'`))
}

// AC11: audit lỗi → không ghi điểm (fail-closed).
func TestPutEntryAuditFailClosed(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, _ := d.sheet()
	d.exec(`create function fail_audit() returns trigger language plpgsql as $$ begin raise exception 'audit down'; end $$`)
	d.exec(`create trigger fail_audit_grade before insert on audit_log for each row when (new.entity = 'grade_entries') execute function fail_audit()`)
	_, err := put(d, gv, c, st, weekly, s("8"), nil)
	require.Error(t, err)
	require.Zero(t, d.countRows(`select count(*) from grade_entries`), "không có audit thì không có điểm")
}

func TestEntryHistoryFromAudit(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, final := d.sheet()
	_, err := put(d, gv, c, st, weekly, s("8"), nil)
	require.NoError(t, err)
	_, err = put(d, gv, c, st, weekly, s("9"), n(1))
	require.NoError(t, err)
	_, err = put(d, gv, c, st, final, s("6"), nil)
	require.NoError(t, err)
	_, err = d.svc().PublishItem(d.ctx, gv, c, weekly, true)
	require.NoError(t, err)
	other := d.student(c, "Binh", "")
	_, err = put(d, gv, c, other, weekly, s("5"), nil)
	require.NoError(t, err)

	page, err := d.svc().History(d.ctx, c, st, nil, nil, 50)
	require.NoError(t, err)
	require.Len(t, page.Items, 3, "chỉ ô của sinh viên này (việc công bố cột không gắn sinh viên)")
	first := page.Items[0] // mới nhất trước
	require.Equal(t, "6", *first.To)
	require.Nil(t, first.From)
	require.Equal(t, "final", first.Item)
	require.Equal(t, "GV", first.By)
	require.Equal(t, "8", *page.Items[2].To)
	require.Equal(t, "8", *page.Items[1].From)

	small, err := d.svc().History(d.ctx, c, st, nil, nil, 2)
	require.NoError(t, err)
	require.Len(t, small.Items, 2)
	require.NotNil(t, small.NextCursor)
	at, id, err := grade.ParseHistoryCursor(*small.NextCursor)
	require.NoError(t, err)
	rest, err := d.svc().History(d.ctx, c, st, &at, &id, 2)
	require.NoError(t, err)
	require.Len(t, rest.Items, 1)
	require.Nil(t, rest.NextCursor)
}

func TestPublishItemToggle(t *testing.T) {
	d := newTDB(t)
	c, gv, _, weekly, _ := d.sheet()
	for _, want := range []bool{true, true, false} {
		got, err := d.svc().PublishItem(d.ctx, gv, c, weekly, want)
		require.NoError(t, err)
		require.Equal(t, want, got)
		var has bool
		d.one(&has, `select published_at is not null from grade_items where id = $1`, weekly)
		require.Equal(t, want, has)
	}
}

func TestPublishItemAudit(t *testing.T) {
	d := newTDB(t)
	c, gv, _, weekly, _ := d.sheet()
	for _, want := range []bool{true, true, false} { // lần lặp không phát sinh audit
		_, err := d.svc().PublishItem(d.ctx, gv, c, weekly, want)
		require.NoError(t, err)
	}
	require.Equal(t, 2, d.countRows(`select count(*) from audit_log where entity = 'grade_items' and action = 'publish'`))
}

func TestPublishExamItemNotToggleable(t *testing.T) {
	d := newTDB(t)
	c, gv, _, _, final := d.sheet()
	ex := d.exam(c, gv, "PUBLISHED", "10")
	d.exec(`update grade_items set source = 'EXAM', exam_id = $2 where id = $1`, final, ex)
	_, err := d.svc().PublishItem(d.ctx, gv, c, final, true)
	status, code := apiCode(t, err)
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, apierr.ItemNotPublishable, code)
}

func TestPublishLockedRejected(t *testing.T) {
	d := newTDB(t)
	c, gv, _, weekly, _ := d.sheet()
	d.exec(`update gradebooks set status = 'LOCKED', snapshot_version = 1 where course_id = $1`, c)
	_, err := d.svc().PublishItem(d.ctx, gv, c, weekly, true)
	_, code := apiCode(t, err)
	require.Equal(t, apierr.GradebookLocked, code)
}

// Cột đã công bố: điểm nhập mới vào cột vẫn thuộc cột công bố (published_at ở cột, không ở từng điểm).
func TestPublishedColumnShownInGradebook(t *testing.T) {
	d := newTDB(t)
	c, gv, st, weekly, _ := d.sheet()
	_, err := d.svc().PublishItem(d.ctx, gv, c, weekly, true)
	require.NoError(t, err)
	_, err = put(d, gv, c, st, weekly, s("8"), nil)
	require.NoError(t, err)
	page, err := d.svc().Gradebook(d.ctx, c, "", 30, true)
	require.NoError(t, err)
	var pub int
	for _, col := range page.Columns {
		if col.Published {
			pub++
			require.Equal(t, weekly, col.ItemID)
		}
	}
	require.Equal(t, 1, pub)
}

// TLR-17: luật không áp dụng mà vẫn có sự kiện → cảnh báo (kèm sinh viên liên quan); không có sự kiện → không cảnh báo.
func TestGradebookWarningsPenaltyNotInScheme(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	d.confirmDoc(c, gv, d.newDraft(c, gv, 1, nil)) // PENALTY / BONUS applicable=false
	st := d.student(c, "An", "")
	page, err := d.svc().Gradebook(d.ctx, c, "", 30, true)
	require.NoError(t, err)
	require.Empty(t, page.Warnings)

	d.part(c, st, gv, "PENALTY", "-0.5")
	d.part(c, st, gv, "SPEAK", "0.25")
	page, err = d.svc().Gradebook(d.ctx, c, "", 30, true)
	require.NoError(t, err)
	require.Equal(t, []string{grade.WarnPenaltyNotInScheme, grade.WarnBonusNotInScheme}, page.Warnings)
	require.Equal(t, st, page.WarningStudents[grade.WarnPenaltyNotInScheme][0].ID)
	next, err := d.svc().Gradebook(d.ctx, c, "x", 30, false)
	_ = next
	require.Error(t, err, "cursor hỏng → 422")
	second, err := d.svc().Gradebook(d.ctx, c, "", 30, false)
	require.NoError(t, err)
	require.Empty(t, second.Warnings, "cảnh báo chỉ ở trang đầu")
}

func TestGradebookNoSchemeHasNoColumns(t *testing.T) {
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	c := d.course(gv, "L1")
	d.student(c, "An", "")
	page, err := d.svc().Gradebook(d.ctx, c, "", 30, true)
	require.NoError(t, err)
	require.Nil(t, page.Scheme)
	require.Empty(t, page.Columns)
	require.Len(t, page.Rows, 1)
	require.Nil(t, page.Rows[0].Result)
}

func (d *tdb) bulk(course uuid.UUID, count int) {
	d.t.Helper()
	d.exec(`with u as (insert into users (email, full_name, role) select 'bulk' || g || '.' || $2 || '@example.test', 'Sv ' || lpad(g::text, 4, '0'), 'STUDENT' from generate_series(1, $3::int) g returning id)
		insert into enrollments (course_id, user_id, role_in_course, status, joined_via) select $1, id, 'STUDENT', 'ACTIVE', 'ADMIN' from u`, course, uuid.NewString()[:8], count)
}

type counter struct{ n atomic.Int64 }

func (c *counter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.n.Add(1)
	return ctx
}
func (c *counter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// countingDB: cùng CSDL nhưng pool có bộ đếm câu SQL (kiểm không N+1).
func countingDB(t *testing.T) (*tdb, *counter) {
	t.Helper()
	testutil.RequireContainers(t)
	url := testutil.MigratedPostgresURL(t)
	cfg, err := pgxpool.ParseConfig(url)
	require.NoError(t, err)
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	c := &counter{}
	cfg.ConnConfig.Tracer = c
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return &tdb{t: t, ctx: t.Context(), pool: pool, q: store.New(pool)}, c
}

// AC8: số truy vấn của một trang không phụ thuộc số sinh viên của trang.
func TestGradebookNoNPlusOne(t *testing.T) {
	d, c := countingDB(t)
	gv := d.user("TEACHER", "GV")
	course := d.course(gv, "L1")
	d.confirmDoc(course, gv, d.newDraft(course, gv, 1, nil))
	d.bulk(course, 120)
	count := func(limit int) int64 {
		before := c.n.Load()
		page, err := d.svc().Gradebook(d.ctx, course, "", limit, false)
		require.NoError(t, err)
		require.Len(t, page.Rows, limit)
		return c.n.Load() - before
	}
	small, big := count(5), count(100)
	require.Equal(t, small, big, "cùng số truy vấn cho 5 và 100 dòng")
	require.LessOrEqual(t, big, int64(12))
}

func TestGradebookPageP95(t *testing.T) {
	testutil.SkipTiming(t)
	d := newTDB(t)
	gv := d.user("TEACHER", "GV")
	course := d.course(gv, "L1")
	d.confirmDoc(course, gv, d.newDraft(course, gv, 1, nil))
	d.bulk(course, 1000)
	var ds []time.Duration
	cursor := ""
	rows := 0
	for range 10 {
		t0 := time.Now()
		page, err := d.svc().Gradebook(d.ctx, course, cursor, 100, cursor == "")
		require.NoError(t, err)
		ds = append(ds, time.Since(t0))
		rows += len(page.Rows)
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	require.Equal(t, 1000, rows, "đi hết 1.000 sinh viên, không trùng, không sót")
	sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
	require.Less(t, ds[len(ds)-1], 500*time.Millisecond, "p95 ≤ 500 ms")
}

// AC8: truy vấn của trang có chỉ mục (tắt quét tuần tự → kế hoạch không còn Seq Scan trên các bảng nguồn).
func TestGradebookExplainNoSeqScan(t *testing.T) {
	d := newTDB(t)
	c, _, _, _, _ := d.sheet()
	d.bulk(c, 50)
	ctx := d.ctx
	tx, err := d.pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `set local enable_seqscan = off`)
	require.NoError(t, err)
	ids := "(select coalesce(array_agg(user_id), '{}') from enrollments where course_id = $1)"
	for name, sql := range map[string]string{
		"grade_entries":        `explain select e.item_id from grade_entries e join grade_items i on i.course_id = e.course_id and i.id = e.item_id where e.course_id = $1 and i.scheme_id = (select id from grade_schemes where course_id = $1 and status = 'CONFIRMED') and e.student_id = any(` + ids + `)`,
		"participation_events": `explain select sum(points) from participation_events where course_id = $1 and student_id = any(` + ids + `) group by student_id`,
		"attendance_records":   `explain select count(*) from attendance_records where course_id = $1 and student_id = any(` + ids + `) and status = 'ABSENT' group by student_id`,
	} {
		rows, err := tx.Query(ctx, sql, c)
		require.NoError(t, err, name)
		var plan []string
		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			plan = append(plan, line)
		}
		rows.Close()
		for _, l := range plan {
			require.False(t, strings.Contains(l, "Seq Scan on "+name), "%s: %s", name, l)
		}
	}
}

// TestPreviewUsesCompute — xem trước = `Compute` (cùng engine với bảng điểm / giải trình); `summary` đếm cả lớp.
func TestPreviewUsesCompute(t *testing.T) {
	d := newTDB(t)
	c, _, first, items := func() (uuid.UUID, uuid.UUID, uuid.UUID, map[string]uuid.UUID) {
		course, gv, confirmed, students := d.gradedClass()
		return course, gv, students[0], d.itemIDs(course, confirmed.ID)
	}()
	_ = items
	pv, err := d.svc().Preview(d.ctx, c, "", 30)
	require.NoError(t, err)
	require.NotNil(t, pv.Summary)
	require.Equal(t, 3, pv.Summary.Students)
	require.Equal(t, 3, pv.Summary.Complete)
	require.Equal(t, 3, pv.Summary.Distribution["A"]+pv.Summary.Distribution["F"]+pv.Summary.Distribution["A+"], "mọi sinh viên có điểm chữ")
	ex, err := d.svc().Explain(d.ctx, c, first)
	require.NoError(t, err)
	var item *grade.PreviewItem
	for i := range pv.Items {
		if pv.Items[i].StudentID == first {
			item = &pv.Items[i]
		}
	}
	require.NotNil(t, item)
	require.Equal(t, ex.Result.HP, item.HP)
	require.Equal(t, ex.Result.Letter, item.Letter)
	next, err := d.svc().Preview(d.ctx, c, "", 2)
	require.NoError(t, err)
	require.NotNil(t, next.NextCursor)
	rest, err := d.svc().Preview(d.ctx, c, *next.NextCursor, 2)
	require.NoError(t, err)
	require.Nil(t, rest.Summary, "summary chỉ ở trang đầu")
	require.Len(t, rest.Items, 1)
}
