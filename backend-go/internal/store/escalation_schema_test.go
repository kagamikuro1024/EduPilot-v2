package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib" // driver "pgx" cho goose provider của TestMigration00012Defaults
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/db"
	"github.com/edupilot/backend-go/internal/testutil"
)

// joinCode dựng mã vào lớp hợp lệ (7 ký tự, bảng chữ không có 0 / 1 / I / L / O) từ một uuid.
func joinCode(id uuid.UUID) string {
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, 7)
	for i := range b {
		b[i] = alpha[int(id[i])%len(alpha)]
	}
	return string(b)
}

func tableCols(t *testing.T, ctx context.Context, conn *pgx.Conn, table string) []string {
	t.Helper()
	rows, err := conn.Query(ctx, `select column_name from information_schema.columns where table_schema='public' and table_name=$1 order by ordinal_position`, table)
	require.NoError(t, err)
	defer rows.Close()
	var got []string
	for rows.Next() {
		var c string
		require.NoError(t, rows.Scan(&c))
		got = append(got, c)
	}
	require.NoError(t, rows.Err())
	return got
}

// ticketFx: lớp 1 + lớp 2 (chatFx) kèm tin trợ lý của sinh viên 1 ở lớp 1 và lớp 2, và một giảng viên khác.
type ticketFx struct {
	chatFx
	asst1, asstC2 uuid.UUID
}

func newTicketFx(t *testing.T, ctx context.Context, conn *pgx.Conn) ticketFx {
	t.Helper()
	f := ticketFx{chatFx: newChatFx(t, ctx, conn)}
	msg := `insert into chat_messages (course_id, session_id, user_id, role, content, intent) values ($1, $2, $3, 'ASSISTANT', 'trả lời', 'COURSE_QA') returning id`
	require.NoError(t, conn.QueryRow(ctx, msg, f.c1, f.sess1, f.s1).Scan(&f.asst1))
	require.NoError(t, conn.QueryRow(ctx, msg, f.c2, f.sessC2, f.s1).Scan(&f.asstC2))
	return f
}

// ins là lệnh chèn ticket: cols/vals là phần thêm vào bộ cột tối thiểu (course, origin, source_type/id, student).
func (f ticketFx) ins(cols, vals string) string {
	return fmt.Sprintf(`insert into escalation_tickets (course_id, origin, source_type, source_id, student_id, %s) values ('%s', 'STUDENT_REQUEST', 'CHAT_MESSAGE', '%s', '%s', %s)`, cols, f.c1, uuid.New(), f.s1, vals)
}

// TestSchemaEscalationTickets — US-P4-01 AC2: đúng cột / kiểu của SRS 5.2, enum, UNIQUE message_id, chỉ mục.
func TestSchemaEscalationTickets(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	require.Equal(t, []string{"id", "course_id", "kind", "origin", "status", "source_type", "source_id", "message_id", "session_id", "student_id", "confidence", "appeal_payload",
		"claimed_by", "claimed_at", "answer", "answered_by", "answered_at", "closed_at", "closed_reason", "dismiss_reason", "reopened_count", "sla_stage", "sla_stage_at",
		"waiting_since", "save_as_knowledge", "version", "created_at", "updated_at"}, tableCols(t, ctx, conn, "escalation_tickets"))
	for enum, want := range map[string][]string{
		"ticket_status": {"OPEN", "CLAIMED", "ANSWERED", "CLOSED"},
		"ticket_kind":   {"QUESTION", "GRADE_APPEAL", "TEACHER_OUTREACH"},
		"ticket_origin": {"AUTO_LOW_CONFIDENCE", "AUTO_NO_FORMULA", "STUDENT_REQUEST", "STAFF_OUTREACH", "STUDENT_APPEAL"},
	} {
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
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_indexes where tablename='escalation_tickets' and indexname in
		('escalation_tickets_course_status_idx','escalation_tickets_student_idx','escalation_tickets_sla_idx','escalation_tickets_autoclose_idx','escalation_tickets_claimed_idx','escalation_tickets_message_key')`).Scan(&n))
	require.Equal(t, 6, n)
	f := newTicketFx(t, ctx, conn)
	_, err := conn.Exec(ctx, f.ins("message_id, session_id", fmt.Sprintf("'%s', '%s'", f.asst1, f.sess1)))
	require.NoError(t, err)
	_, err = conn.Exec(ctx, f.ins("message_id", fmt.Sprintf("'%s'", f.asst1)))
	require.Equal(t, "23505", sqlState(err), "message_id duy nhất")
}

// TestTicketChecks — AC2: từng CHECK → 23514.
func TestTicketChecks(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newTicketFx(t, ctx, conn)
	m := fmt.Sprintf("'%s'", f.asst1)
	const check = "23514"
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"OPEN có claimed_by":             {check, f.ins("message_id, claimed_by", m+fmt.Sprintf(", '%s'", f.teacher))},
		"CLAIMED thiếu claimed_by":       {check, f.ins("message_id, status", m+", 'CLAIMED'")},
		"ANSWERED thiếu answered_by":     {check, f.ins("message_id, status", m+", 'ANSWERED'")},
		"CLOSED thiếu closed_at":         {check, f.ins("message_id, status", m+", 'CLOSED'")},
		"QUESTION thiếu message_id":      {check, f.ins("kind", "'QUESTION'")},
		"GRADE_APPEAL thiếu payload":     {check, f.ins("kind", "'GRADE_APPEAL'")},
		"payload ở kind khác GRADE":      {check, f.ins("message_id, appeal_payload", m+", '{}'")},
		"payload không phải object":      {check, f.ins("kind, appeal_payload", "'GRADE_APPEAL', '[]'")},
		"confidence ≤ 1":                 {check, f.ins("message_id, confidence", m+", 1.5")},
		"sla_stage ≤ 2":                  {check, f.ins("message_id, sla_stage", m+", 3")},
		"version ≥ 1":                    {check, f.ins("message_id, version", m+", 0")},
		"answer ≤ 4000":                  {check, f.ins("message_id, answer", m+", repeat('x', 4001)")},
		"closed_reason ngoài danh sách":  {check, f.ins("message_id, closed_reason", m+", 'X'")},
		"dismiss_reason ngoài danh sách": {check, f.ins("message_id, dismiss_reason", m+", 'X'")},
		"source_type sai mẫu":            {check, strings.Replace(f.ins("message_id", m), "'CHAT_MESSAGE'", "'chat'", 1)},
	})
	// ca hợp lệ của từng nhánh
	for _, q := range []string{
		f.ins("kind, appeal_payload", `'GRADE_APPEAL', '{"item":1}'`),
		f.ins("kind, status, claimed_by, answered_by", fmt.Sprintf("'TEACHER_OUTREACH', 'ANSWERED', '%[1]s', '%[1]s'", f.teacher)),
		f.ins("message_id, status, closed_at, closed_reason", m+", 'CLOSED', now(), 'AUTO_7D'"),
	} {
		_, err := conn.Exec(ctx, q)
		require.NoError(t, err, q)
	}
}

// TestTicketCrossCourseFK — AC2: tin / phiên của lớp khác → 23503.
func TestTicketCrossCourseFK(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newTicketFx(t, ctx, conn)
	_, err := conn.Exec(ctx, f.ins("message_id", fmt.Sprintf("'%s'", f.asstC2)))
	require.Equal(t, "23503", sqlState(err), "message_id của lớp khác")
	_, err = conn.Exec(ctx, f.ins("message_id, session_id", fmt.Sprintf("'%s', '%s'", f.asst1, f.sessC2)))
	require.Equal(t, "23503", sqlState(err), "session_id của lớp khác")
}

// TestSchemaPostReports — AC3: đúng một trong thread_id / post_id; UNIQUE (reporter, target); lý do thuộc enum; note chỉ ở OTHER.
func TestSchemaPostReports(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	require.Equal(t, []string{"id", "course_id", "thread_id", "post_id", "target_id", "reporter_id", "reason", "note", "status", "handled_by", "handled_at", "created_at", "updated_at"}, tableCols(t, ctx, conn, "post_reports"))
	var post uuid.UUID
	require.NoError(t, conn.QueryRow(ctx, `insert into forum_posts (course_id, thread_id, kind, author_id, body) values ($1, $2, 'HUMAN', $3, 'bình luận') returning id`, f.c1, f.thread, f.s2).Scan(&post))
	rep := func(cols, vals string) string {
		return fmt.Sprintf(`insert into post_reports (course_id, reporter_id, reason, %s) values ('%s', '%s', 'SPAM', %s)`, cols, f.c1, f.s2, vals)
	}
	th := fmt.Sprintf("'%s'", f.thread)
	_, err := conn.Exec(ctx, rep("thread_id", th))
	require.NoError(t, err)
	var target uuid.UUID
	require.NoError(t, conn.QueryRow(ctx, `select target_id from post_reports where thread_id = $1`, f.thread).Scan(&target))
	require.Equal(t, f.thread, target, "target_id = thread_id")
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"cùng người báo cùng bài": {"23505", rep("thread_id", th)},
		"không có đích":           {"23514", rep("note", "null")},
		"cả thread lẫn post":      {"23514", rep("thread_id, post_id", th+fmt.Sprintf(", '%s'", post))},
		"note ở lý do khác OTHER": {"23514", rep("post_id, note", fmt.Sprintf("'%s', 'x'", post))},
		"note ≤ 200":              {"23514", strings.Replace(rep("post_id, note", fmt.Sprintf("'%s', repeat('x', 201)", post)), "'SPAM'", "'OTHER'", 1)},
		"handled lệch cặp":        {"23514", rep("post_id, handled_by", fmt.Sprintf("'%s', '%s'", post, f.teacher))},
		"lý do ngoài enum":        {"22P02", strings.Replace(rep("post_id", fmt.Sprintf("'%s'", post)), "'SPAM'", "'KHAC'", 1)},
		"thread của lớp khác":     {"23503", strings.Replace(strings.Replace(rep("thread_id", th), f.c1.String(), f.c2.String(), 1), f.s2.String(), f.s1.String(), 1)},
	})
}

// TestSchemaKnowledgeItems — AC3: UNIQUE (source_type, source_id); chunk bị xoá thì dòng đi theo.
func TestSchemaKnowledgeItems(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	var chunk uuid.UUID
	require.NoError(t, conn.QueryRow(ctx, `insert into content_chunks (document_id, ord, text, course_ids) values ($1, 0, 'Hỏi: a? Đáp: b', array[$2::uuid]) returning id`, f.doc, f.c1).Scan(&chunk))
	src := uuid.New()
	ins := func(typ string) error {
		_, err := conn.Exec(ctx, `insert into knowledge_items (course_id, source_type, source_id, chunk_id, created_by) values ($1, $2, $3, $4, $5)`, f.c1, typ, src, chunk, f.teacher)
		return err
	}
	require.NoError(t, ins("TICKET"))
	require.Equal(t, "23505", sqlState(ins("TICKET")), "UNIQUE (source_type, source_id)")
	require.NoError(t, ins("POST"), "cùng source_id khác loại là hợp lệ")
	require.Equal(t, "23514", sqlState(ins("OTHER")))
	_, err := conn.Exec(ctx, `delete from content_chunks where id = $1`, chunk)
	require.NoError(t, err)
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from knowledge_items`).Scan(&n))
	require.Zero(t, n, "ON DELETE CASCADE")
	var idx int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from pg_indexes where indexname in ('documents_knowledge_key','mail_outbox_dead_idx')`).Scan(&idx))
	require.Equal(t, 2, idx)
}

// TestMigration00012Defaults — AC3 (TLR-15): mặc định mới và UPDATE dữ liệu cũ. Dựng DB ở 00011 rồi lên 00012.
func TestMigration00012Defaults(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	url := testutil.PostgresURL(t)
	sdb, err := sql.Open("pgx", url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sdb.Close() })
	p, err := goose.NewProvider(goose.DialectPostgres, sdb, db.FS())
	require.NoError(t, err)
	_, err = p.UpTo(ctx, 11)
	require.NoError(t, err)

	conn, err := pgx.Connect(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(ctx)) })
	sfx := strings.ToLower(uuid.NewString()[:6])
	var teacher, u1, u2 uuid.UUID
	for dst, email := range map[*uuid.UUID]string{&teacher: "t", &u1: "a", &u2: "b"} {
		require.NoError(t, conn.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'X', 'TEACHER') returning id`, email+"."+sfx+"@example.test").Scan(dst))
	}
	mkCourse := func(code string, thr string, version int) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, conn.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by, escalation_threshold, version) values ('INT1006', $1, 'L', '2026-2027-HK1', $2, $3, $4::numeric, $5) returning id`,
			code, joinCode(uuid.New()), teacher, thr, version).Scan(&id))
		return id
	}
	c60, c70 := mkCourse("LOP60A1", "0.60", 3), mkCourse("LOP70B2", "0.70", 1)
	for _, u := range []uuid.UUID{u1, u2} {
		_, err := conn.Exec(ctx, `insert into user_settings (user_id, notify_ticket_by_mail, version) values ($1, true, 2)`, u)
		require.NoError(t, err)
	}
	_, err = p.Up(ctx)
	require.NoError(t, err)

	thr := func(id uuid.UUID) string {
		var s string
		require.NoError(t, conn.QueryRow(ctx, `select escalation_threshold::text from courses where id = $1`, id).Scan(&s))
		return s
	}
	require.Equal(t, "0.80", thr(c60), "lớp version=3 ngưỡng 0.60 → 0.80")
	require.Equal(t, "0.70", thr(c70), "giá trị khác 0.60 giữ nguyên")
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from user_settings where notify_ticket_by_mail`).Scan(&n))
	require.Zero(t, n, "mọi user_settings về false (kể cả version=2)")
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from user_settings where version = 2`).Scan(&n))
	require.Equal(t, 2, n, "UPDATE không đổi version")
	var def string
	require.NoError(t, conn.QueryRow(ctx, `select column_default from information_schema.columns where table_name='user_settings' and column_name='notify_ticket_by_mail'`).Scan(&def))
	require.Equal(t, "false", def)
	require.NoError(t, conn.QueryRow(ctx, `select column_default from information_schema.columns where table_name='courses' and column_name='escalation_threshold'`).Scan(&def))
	require.Equal(t, "0.80", def)
}

// TestEscalationIndexesUsed — AC14: 10.000 ticket / 50 lớp; trang đầu theo lớp + trạng thái, theo sinh viên, và truy vấn của job SLA không Seq Scan.
func TestEscalationIndexesUsed(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	var courses []uuid.UUID
	for i := range 50 {
		var id uuid.UUID
		require.NoError(t, conn.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'L', '2026-2027-HK1', $2, $3) returning id`,
			fmt.Sprintf("IDX%02d-%s", i, strings.ToUpper(uuid.NewString()[:4])), joinCode(uuid.New()), f.teacher).Scan(&id))
		courses = append(courses, id)
	}
	// TEACHER_OUTREACH không cần tin chat; trạng thái xoay vòng, thời điểm ngẫu nhiên.
	_, err := conn.Exec(ctx, `insert into escalation_tickets (course_id, kind, origin, status, source_type, source_id, student_id, claimed_by, answered_by, answered_at, closed_at, waiting_since, created_at)
		select (select c from unnest($1::uuid[]) with ordinality as t(c, n) where n = 1 + g % 50), 'TEACHER_OUTREACH', 'STAFF_OUTREACH',
		       (array['OPEN','CLAIMED','ANSWERED','CLOSED'])[1 + g % 4]::ticket_status, 'STUDENT', $2, $2,
		       case when g % 4 in (1) then $3::uuid end, case when g % 4 in (2) then $3::uuid end,
		       case when g % 4 = 2 then now() - (g || ' minutes')::interval end, case when g % 4 = 3 then now() end,
		       now() - (random() * 1000 || ' hours')::interval, now() - (g || ' seconds')::interval
		from generate_series(1, 10000) g`, courses, f.s1, f.teacher)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `analyze escalation_tickets`)
	require.NoError(t, err)

	c := courses[7]
	for name, q := range map[string]string{
		"theo lớp + trạng thái":    fmt.Sprintf(`select * from escalation_tickets where course_id='%s' and status='OPEN' order by created_at, id limit 30`, c),
		"theo lớp, mọi trạng thái": fmt.Sprintf(`select * from escalation_tickets where course_id='%s' order by created_at, id limit 30`, c),
		"theo sinh viên":           fmt.Sprintf(`select * from escalation_tickets where course_id='%s' and student_id='%s' and status in ('OPEN','CLAIMED') order by created_at desc, id desc limit 30`, c, f.s1),
		"đếm trần":                 fmt.Sprintf(`select count(*) from escalation_tickets where course_id='%s' and student_id='%s' and status in ('OPEN','CLAIMED')`, c, f.s1),
		"job SLA":                  `select * from escalation_tickets where status='OPEN' and sla_stage=0 and waiting_since < now() - interval '24 hours' order by waiting_since, id limit 200`,
		"tự đóng":                  `select id from escalation_tickets where status='ANSWERED' and answered_at < now() - interval '7 days' limit 200`,
	} {
		rows, err := conn.Query(ctx, `explain `+q)
		require.NoError(t, err, name)
		var plan strings.Builder
		for rows.Next() {
			var line string
			require.NoError(t, rows.Scan(&line))
			plan.WriteString(line + "\n")
		}
		require.NoError(t, rows.Err())
		require.NotContains(t, plan.String(), "Seq Scan", name+"\n"+plan.String())
	}
}
