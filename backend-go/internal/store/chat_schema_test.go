package store_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

var chatTables = []string{"chat_sessions", "chat_messages", "forum_threads", "forum_posts", "pii_events"}

// chatFx: hai lớp, một giáo viên, hai sinh viên, một phiên + tin ở lớp 1, một thread ở lớp 1, một phiên ở lớp 2.
type chatFx struct {
	c1, c2, teacher, s1, s2 uuid.UUID
	sess1, sessS2, sessC2   uuid.UUID
	msgUser, thread, doc    uuid.UUID
	msgOtherCourse          uuid.UUID
}

func newChatFx(t *testing.T, ctx context.Context, conn *pgx.Conn) chatFx {
	t.Helper()
	var f chatFx
	one := func(dst *uuid.UUID, sql string, args ...any) {
		require.NoError(t, conn.QueryRow(ctx, sql, args...).Scan(dst), sql)
	}
	id := uuid.New()
	sfx := strings.ToUpper(id.String()[:6])
	lsfx := strings.ToLower(sfx)
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := func(seed byte) string {
		b := make([]byte, 7)
		for i := range b {
			b[i] = alpha[(int(id[i])+int(seed)*7)%len(alpha)]
		}
		return string(b)
	}
	one(&f.teacher, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, "t."+lsfx+"@example.test")
	one(&f.s1, `insert into users (email, full_name, role) values ($1, 'SV1', 'STUDENT') returning id`, "s1."+lsfx+"@example.test")
	one(&f.s2, `insert into users (email, full_name, role) values ($1, 'SV2', 'STUDENT') returning id`, "s2."+lsfx+"@example.test")
	one(&f.c1, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp 1', '2026-2027-HK1', $2, $3) returning id`, "CH1-"+sfx, jc(1), f.teacher)
	one(&f.c2, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp 2', '2026-2027-HK1', $2, $3) returning id`, "CH2-"+sfx, jc(2), f.teacher)
	one(&f.doc, `insert into documents (course_id, title) values ($1, 'Giáo trình') returning id`, f.c1)
	sess := `insert into chat_sessions (course_id, user_id, document_id) values ($1, $2, $3) returning id`
	one(&f.sess1, sess, f.c1, f.s1, f.doc)
	one(&f.sessS2, sess, f.c1, f.s2, nil)
	one(&f.sessC2, sess, f.c2, f.s1, nil)
	msg := `insert into chat_messages (course_id, session_id, user_id, role, content) values ($1, $2, $3, 'USER', 'xin chào') returning id`
	one(&f.msgUser, msg, f.c1, f.sess1, f.s1)
	one(&f.msgOtherCourse, msg, f.c2, f.sessC2, f.s1)
	one(&f.thread, `insert into forum_threads (course_id, author_id, title, body) values ($1, $2, 'Hỏi về bài 1', 'Nội dung') returning id`, f.c1, f.s1)
	return f
}

// expect chạy từng ca ghi sai và đòi đúng SQLSTATE.
func expect(t *testing.T, ctx context.Context, conn *pgx.Conn, cases map[string]struct{ code, sql string }) {
	t.Helper()
	for name, c := range cases {
		_, err := conn.Exec(ctx, c.sql)
		require.Equal(t, c.code, sqlState(err), name)
	}
}

// TestSchemaChat — US-P3-01 AC2, AC7: cột của sáu bảng mới, CHECK từng ca → 23514, UNIQUE client_msg_id → 23505.
func TestSchemaChat(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)

	want := map[string][]string{
		"chat_sessions": {"id", "course_id", "user_id", "channel", "title", "document_id", "last_message_at", "deleted_at", "created_at", "updated_at"},
		"chat_messages": {"id", "course_id", "session_id", "user_id", "role", "content", "partial_content", "stream_status", "client_msg_id", "reply_to", "attempt", "intent",
			"citations", "blocks", "confidence", "low_confidence", "no_context", "degraded", "masked_count", "feedback", "error_code", "trace_id", "completed_at", "created_at", "updated_at"},
		"forum_threads": {"id", "course_id", "author_id", "title", "body", "tags", "week_no", "state", "ai_state", "ai_skip_reason", "similar_of", "pinned_at", "reply_count",
			"last_activity_at", "embedding", "deleted_at", "version", "created_at", "updated_at"},
		"forum_posts": {"id", "course_id", "thread_id", "author_id", "kind", "body", "verification_state", "citations", "confidence", "ai_body", "verified_by", "verified_at",
			"hidden_at", "hidden_reason", "hidden_by", "deleted_at", "embedding", "version", "created_at", "updated_at"},
		"pii_events": {"id", "course_id", "session_id", "user_id", "channel", "pii_type", "count", "action", "created_at"},
	}
	for table, cols := range want {
		rows, err := conn.Query(ctx, `select column_name from information_schema.columns where table_schema='public' and table_name=$1 order by ordinal_position`, table)
		require.NoError(t, err)
		var got []string
		for rows.Next() {
			var c string
			require.NoError(t, rows.Scan(&c))
			got = append(got, c)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, cols, got, table)
	}

	const check, unique = "23514", "23505"
	sid, c1, s1 := f.sess1.String(), f.c1.String(), f.s1.String()
	m := func(cols, vals string) string {
		return fmt.Sprintf(`insert into chat_messages (course_id, session_id, user_id, %s) values ('%s', '%s', '%s', %s)`, cols, c1, sid, s1, vals)
	}
	idem := uuid.NewString()
	_, err := conn.Exec(ctx, m("role, client_msg_id", fmt.Sprintf("'USER', '%s'", idem)))
	require.NoError(t, err)
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"USER không STREAMING":         {check, m("role, stream_status", "'USER', 'STREAMING'")},
		"USER không có confidence":     {check, m("role, confidence", "'USER', 0.5")},
		"DONE không có partial":        {check, m("role, stream_status, partial_content", "'ASSISTANT', 'DONE', 'x'")},
		"STREAMING không completed_at": {check, m("role, stream_status, completed_at", "'ASSISTANT', 'STREAMING', now()")},
		"content ≤ 20000":              {check, m("role, content", "'USER', repeat('x', 20001)")},
		"attempt ≥ 1":                  {check, m("role, attempt", "'ASSISTANT', 0")},
		"intent đúng mẫu":              {check, m("role, intent", "'ASSISTANT', 'chữ thường'")},
		"citations là mảng":            {check, m("role, citations", "'ASSISTANT', '{}'")},
		"blocks là mảng":               {check, m("role, blocks", "'ASSISTANT', '{}'")},
		"confidence ≤ 1":               {check, m("role, confidence", "'ASSISTANT', 1.5")},
		"masked_count ≥ 0":             {check, m("role, masked_count", "'ASSISTANT', -1")},
		"client_msg_id duy nhất":       {unique, m("role, client_msg_id", fmt.Sprintf("'USER', '%s'", idem))},
		"title ≤ 120":                  {check, fmt.Sprintf(`insert into chat_sessions (course_id, user_id, title) values ('%s', '%s', repeat('x', 121))`, c1, s1)},
	})
	// ca hợp lệ: ASSISTANT đang STREAMING có partial; DONE rồi thì partial phải NULL.
	_, err = conn.Exec(ctx, m("role, stream_status, partial_content", "'ASSISTANT', 'STREAMING', 'đang viết'"))
	require.NoError(t, err)
}

// TestSchemaForum — US-P3-01 AC3: CHECK của thread / post và đúng một bài AI mỗi thread.
func TestSchemaForum(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	const check, unique = "23514", "23505"
	c1, th, s1 := f.c1.String(), f.thread.String(), f.s1.String()
	thr := func(cols, vals string) string {
		return fmt.Sprintf(`insert into forum_threads (course_id, author_id, title, body, %s) values ('%s', '%s', 't', 'b', %s)`, cols, c1, s1, vals)
	}
	post := func(cols, vals string) string {
		return fmt.Sprintf(`insert into forum_posts (course_id, thread_id, %s) values ('%s', '%s', %s)`, cols, c1, th, vals)
	}
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"title rỗng":                     {check, fmt.Sprintf(`insert into forum_threads (course_id, author_id, title, body) values ('%s', '%s', '', 'b')`, c1, s1)},
		"body > 8000":                    {check, fmt.Sprintf(`insert into forum_threads (course_id, author_id, title, body) values ('%s', '%s', 't', repeat('x', 8001))`, c1, s1)},
		"tối đa 5 thẻ":                   {check, thr("tags", `'{a,b,c,d,e,f}'`)},
		"thẻ > 30 ký tự":                 {check, thr("tags", fmt.Sprintf(`'{%s}'`, strings.Repeat("x", 31)))},
		"tuần 1–20":                      {check, thr("week_no", "21")},
		"SKIPPED cần lý do":              {check, thr("ai_state", `'SKIPPED'`)},
		"lý do chỉ khi SKIPPED":          {check, thr("ai_skip_reason", `'NO_CONTEXT'`)},
		"lý do trong danh sách":          {check, thr("ai_state, ai_skip_reason", `'SKIPPED', 'KHAC'`)},
		"reply_count ≥ 0":                {check, thr("reply_count", "-1")},
		"bài AI không có author":         {check, post("kind, author_id, body, verification_state", fmt.Sprintf(`'AI', '%s', 'x', 'PENDING'`, s1))},
		"bài HUMAN phải có author":       {check, post("kind, body", `'HUMAN', 'x'`)},
		"HUMAN có verification NONE":     {check, post("kind, author_id, body, verification_state", fmt.Sprintf(`'HUMAN', '%s', 'x', 'PENDING'`, s1))},
		"AI không NONE":                  {check, post("kind, body", `'AI', 'x'`)},
		"confidence chỉ bài AI":          {check, post("kind, author_id, body, confidence", fmt.Sprintf(`'HUMAN', '%s', 'x', 0.5`, s1))},
		"verified_by đi với verified_at": {check, post("kind, body, verification_state, verified_by", fmt.Sprintf(`'AI', 'x', 'VERIFIED', '%s'`, f.teacher))},
		"hidden_at đi với hidden_reason": {check, post("kind, author_id, body, hidden_at", fmt.Sprintf(`'HUMAN', '%s', 'x', now()`, s1))},
		"hidden_reason ≤ 200":            {check, post("kind, author_id, body, hidden_at, hidden_reason", fmt.Sprintf(`'HUMAN', '%s', 'x', now(), repeat('x', 201)`, s1))},
	})
	ai := post("kind, body, verification_state", `'AI', 'trả lời', 'PENDING'`)
	_, err := conn.Exec(ctx, ai)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, ai)
	require.Equal(t, unique, sqlState(err), "đúng một bài AI mỗi thread")
	_, err = conn.Exec(ctx, post("kind, author_id, body", fmt.Sprintf(`'HUMAN', '%s', 'đã hiểu'`, s1)))
	require.NoError(t, err)
}

// TestCrossCourseFK + TestChatOwnerFK + TestReplyToFK — US-P3-01 AC4: DB chặn trộn lớp, trộn chủ phiên.
func TestCrossCourseFK(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	const fk = "23503"
	expect(t, ctx, conn, map[string]struct{ code, sql string }{
		"post course_id ≠ lớp của thread": {fk, fmt.Sprintf(`insert into forum_posts (course_id, thread_id, kind, body, verification_state) values ('%s', '%s', 'AI', 'x', 'PENDING')`, f.c2, f.thread)},
		"tin course_id ≠ lớp của phiên":   {fk, fmt.Sprintf(`insert into chat_messages (course_id, session_id, user_id, role) values ('%s', '%s', '%s', 'USER')`, f.c2, f.sess1, f.s1)},
		"similar_of ở lớp khác":           {fk, fmt.Sprintf(`insert into forum_threads (course_id, author_id, title, body, similar_of) values ('%s', '%s', 't', 'b', '%s')`, f.c2, f.s1, f.thread)},
		"pii_events session ở lớp khác":   {fk, fmt.Sprintf(`insert into pii_events (course_id, session_id, user_id, channel, pii_type, count, action) values ('%s', '%s', '%s', 'PRIVATE', 'MSSV', 1, 'MASKED')`, f.c2, f.sess1, f.s1)},
	})
}

func TestChatOwnerFK(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	_, err := conn.Exec(ctx, `insert into chat_messages (course_id, session_id, user_id, role) values ($1, $2, $3, 'USER')`, f.c1, f.sess1, f.s2)
	require.Equal(t, "23503", sqlState(err), "user_id ≠ chủ phiên")
}

func TestReplyToFK(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	_, err := conn.Exec(ctx, `insert into chat_messages (course_id, session_id, user_id, role, reply_to) values ($1, $2, $3, 'ASSISTANT', $4)`, f.c1, f.sess1, f.s1, f.msgOtherCourse)
	require.Equal(t, "23503", sqlState(err), "reply_to trỏ tin lớp khác")
	_, err = conn.Exec(ctx, `insert into chat_messages (course_id, session_id, user_id, role, reply_to) values ($1, $2, $3, 'ASSISTANT', $4)`, f.c1, f.sess1, f.s1, f.msgUser)
	require.NoError(t, err)
}

// TestDocumentDeleteSetsNull — xoá tài liệu không vướng FK; phiên thành "cả lớp".
func TestDocumentDeleteSetsNull(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	_, err := conn.Exec(ctx, `delete from documents where id = $1`, f.doc)
	require.NoError(t, err)
	var doc *uuid.UUID
	require.NoError(t, conn.QueryRow(ctx, `select document_id from chat_sessions where id = $1`, f.sess1).Scan(&doc))
	require.Nil(t, doc)
}

// TestUpdatedAtTriggers — bốn bảng có trigger set_updated_at (reaper dựa vào chat_messages.updated_at).
func TestUpdatedAtTriggers(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	for _, c := range []struct{ table, set string }{
		{"chat_sessions", "title = 'đổi'"}, {"chat_messages", "content = 'đổi'"}, {"forum_threads", "title = 'đổi'"},
	} {
		var advanced bool
		require.NoError(t, conn.QueryRow(ctx, fmt.Sprintf(`with u as (update %s set %s where course_id = $1 returning created_at, updated_at) select bool_and(updated_at > created_at) from u`, c.table, c.set), f.c1).Scan(&advanced), c.table)
		require.True(t, advanced, c.table)
	}
	_, err := conn.Exec(ctx, `insert into forum_posts (course_id, thread_id, kind, body, verification_state) values ($1, $2, 'AI', 'x', 'PENDING')`, f.c1, f.thread)
	require.NoError(t, err)
	var advanced bool
	require.NoError(t, conn.QueryRow(ctx, `with u as (update forum_posts set body = 'đổi' where course_id = $1 returning created_at, updated_at) select bool_and(updated_at > created_at) from u`, f.c1).Scan(&advanced))
	require.True(t, advanced, "forum_posts")
}

// TestPIIEventsAppendOnly / NoFreeText — US-P3-01 AC5.
func TestPIIEventsAppendOnly(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	_, err := conn.Exec(ctx, `insert into pii_events (course_id, session_id, user_id, channel, pii_type, count, action) values ($1, $2, $3, 'PRIVATE', 'MSSV', 2, 'MASKED')`, f.c1, f.sess1, f.s1)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `insert into pii_events (course_id, user_id, channel, pii_type, count, action) values ($1, $2, 'PUBLIC', 'PERSONAL_QUESTION', 1, 'BLOCKED')`, f.c1, f.s1)
	require.NoError(t, err, "session_id NULL được")
	_, err = conn.Exec(ctx, `insert into pii_events (course_id, user_id, channel, pii_type, count, action) values ($1, $2, 'PUBLIC', 'EMAIL', 0, 'REDACTED')`, f.c1, f.s1)
	require.Equal(t, "23514", sqlState(err), "count ≥ 1")
	for _, sql := range []string{`update pii_events set count = 2`, `delete from pii_events`, `truncate pii_events`} {
		_, err = conn.Exec(ctx, sql)
		require.Equal(t, "42501", sqlState(err), sql)
		require.ErrorContains(t, err, "append-only")
	}
}

func TestPIIEventsNoFreeText(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	var n int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from information_schema.columns where table_name = 'pii_events' and data_type in ('text', 'character varying', 'json', 'jsonb')`).Scan(&n))
	require.Zero(t, n)
}

// TestEnumRejectsUnknown / TestMigrateIdempotentUp — US-P3-01 AC8.
func TestEnumRejectsUnknown(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	_, err := conn.Exec(ctx, `insert into chat_messages (course_id, session_id, user_id, role, stream_status) values ($1, $2, $3, 'ASSISTANT', 'X')`, f.c1, f.sess1, f.s1)
	require.Equal(t, "22P02", sqlState(err))
}

func TestMigrateIdempotentUp(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	require.NoError(t, migrateCmd(ctx, conn.Config().ConnString(), "up"), "up lần hai không làm gì")
}

// TestChatMigrationDownUp — down gỡ sạch 5 bảng + 10 enum + hàm của 00007/00008 trước khi gỡ courses/users; up lại được.
func TestChatMigrationDownUp(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	url := conn.Config().ConnString()
	count := func(q string, args ...any) int {
		var n int
		require.NoError(t, conn.QueryRow(ctx, q, args...).Scan(&n))
		return n
	}
	tables := func() int {
		return count(`select count(*) from information_schema.tables where table_schema='public' and table_name = any($1)`, chatTables)
	}
	enums := func() int {
		return count(`select count(*) from pg_type where typtype = 'e' and typname in ('chat_channel','chat_role','chat_stream_status','chat_feedback','thread_state','thread_ai_state','post_kind','post_verification','pii_kind','pii_action')`)
	}
	require.Equal(t, 5, tables())
	require.Equal(t, 10, enums())
	require.NoError(t, migrateCmd(ctx, url, "down"))
	require.Equal(t, 0, tables())
	require.Equal(t, 0, enums())
	require.Equal(t, 0, count(`select count(*) from pg_proc where proname = 'forum_tags_valid'`))
	require.NoError(t, migrateCmd(ctx, url, "up"))
	require.Equal(t, 5, tables())
}

// TestChatForumIndexesUsed — US-P3-01 AC6: trang đầu của bốn danh sách trên 10.000 dòng không Seq Scan.
func TestChatForumIndexesUsed(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newChatFx(t, ctx, conn)
	for _, q := range []string{
		`insert into chat_sessions (course_id, user_id, last_message_at) select '%[1]s', '%[2]s', now() - (g || ' seconds')::interval from generate_series(1, 10000) g`,
		`insert into chat_messages (course_id, session_id, user_id, role, created_at) select '%[1]s', '%[3]s', '%[2]s', 'USER', now() - (g || ' seconds')::interval from generate_series(1, 10000) g`,
		`insert into forum_threads (course_id, author_id, title, body, last_activity_at) select '%[1]s', '%[2]s', 't' || g, 'b', now() - (g || ' seconds')::interval from generate_series(1, 10000) g`,
	} {
		_, err := conn.Exec(ctx, fmt.Sprintf(q, f.c1, f.s1, f.sess1))
		require.NoError(t, err)
	}
	_, err := conn.Exec(ctx, `insert into forum_posts (course_id, thread_id, kind, author_id, body, created_at)
		select $1, $2, 'HUMAN', $3, 'p', now() + (g || ' seconds')::interval from generate_series(1, 10000) g`, f.c1, f.thread, f.s1)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `analyze chat_sessions, chat_messages, forum_threads, forum_posts`)
	require.NoError(t, err)

	for name, q := range map[string]string{
		"chat_sessions": fmt.Sprintf(`select * from chat_sessions where course_id='%s' and user_id='%s' and deleted_at is null order by last_message_at desc, id desc limit 30`, f.c1, f.s1),
		"chat_messages": fmt.Sprintf(`select * from chat_messages where session_id='%s' order by created_at desc, id desc limit 30`, f.sess1),
		"forum_threads": fmt.Sprintf(`select * from forum_threads where course_id='%s' and deleted_at is null order by last_activity_at desc, id desc limit 30`, f.c1),
		"forum_posts":   fmt.Sprintf(`select * from forum_posts where thread_id='%s' order by created_at, id limit 30`, f.thread),
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
