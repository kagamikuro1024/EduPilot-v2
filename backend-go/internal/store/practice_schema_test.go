package store_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

// TestSchemaPractice — US-P9-01 AC1: cột đúng SRS 5; chèn dòng con khác course_id với cha bị DB từ chối (23503); ràng buộc CHECK / UNIQUE từng phần.
func TestSchemaPractice(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	f := newCrmFx(t, ctx, conn)
	want := map[string][]string{
		"practice_attempts":    {"id", "course_id", "student_id", "mode", "topic", "status", "started_at", "deadline_at", "submitted_at", "item_count", "current_index", "correct_count", "score", "matrix", "notes", "tab_id", "version", "created_at", "updated_at"},
		"practice_items":       {"id", "course_id", "attempt_id", "question_id", "position", "option_order", "answer_key", "question", "question_version"},
		"practice_answers":     {"id", "course_id", "attempt_id", "item_id", "answer", "is_correct", "answered_at", "locked", "version"},
		"forms_import_batches": {"id", "course_id", "item_id", "file_hash", "blob_key", "mapping", "matched", "unmatched", "created_by", "created_at"},
		"forms_import_rows":    {"course_id", "batch_id", "row_no", "student_id", "identifier_masked", "correct", "total", "score"},
	}
	for table, cols := range want {
		require.Equal(t, cols, tableCols(t, ctx, conn, table), table)
	}

	one := func(sql string, args ...any) uuid.UUID {
		var id uuid.UUID
		require.NoError(t, conn.QueryRow(ctx, sql, args...).Scan(&id), sql)
		return id
	}
	code := func(err error) string {
		var pe *pgconn.PgError
		require.True(t, errors.As(err, &pe), "%v", err)
		return pe.Code
	}
	q1 := one(`insert into question_bank (course_id, type, title, topic, stem, answer_key, created_by) values ($1, 'SHORT', 't', 'x', 's', '{"accepted":["a"]}', $2) returning id`, f.c1, f.teacher)
	q2 := one(`insert into question_bank (course_id, type, title, topic, stem, answer_key, created_by) values ($1, 'SHORT', 't', 'x', 's', '{"accepted":["a"]}', $2) returning id`, f.c2, f.teacher)
	att := one(`insert into practice_attempts (course_id, student_id, mode, topic, item_count) values ($1, $2, 'TOPIC', 'x', 2) returning id`, f.c1, f.s1)
	item := one(`insert into practice_items (course_id, attempt_id, question_id, position, answer_key, question, question_version) values ($1, $2, $3, 0, '{}', '{}', 1) returning id`, f.c1, att, q1)

	// FK phức hợp: cha ở lớp 1, con khai lớp 2 → 23503 (ở mỗi hướng)
	_, err := conn.Exec(ctx, `insert into practice_items (course_id, attempt_id, question_id, position, answer_key, question, question_version) values ($1, $2, $3, 1, '{}', '{}', 1)`, f.c2, att, q2)
	require.Equal(t, "23503", code(err), "attempt khác lớp")
	_, err = conn.Exec(ctx, `insert into practice_items (course_id, attempt_id, question_id, position, answer_key, question, question_version) values ($1, $2, $3, 1, '{}', '{}', 1)`, f.c1, att, q2)
	require.Equal(t, "23503", code(err), "câu hỏi lớp khác")
	_, err = conn.Exec(ctx, `insert into practice_answers (course_id, attempt_id, item_id, answer) values ($1, $2, $3, '{}')`, f.c2, att, item)
	require.Equal(t, "23503", code(err), "answer khác lớp")
	_, err = conn.Exec(ctx, `insert into practice_attempts (course_id, student_id, mode, topic, item_count) values ($1, $2, 'TOPIC', 'y', 1)`, f.c2, f.s2)
	require.Equal(t, "23503", code(err), "sinh viên không ghi danh lớp 2")

	// CHECK: MOCK cần deadline, TOPIC cần chủ đề, current_index < item_count, trạng thái đóng khớp submitted_at
	for name, sql := range map[string]string{
		"MOCK thiếu deadline":   `insert into practice_attempts (course_id, student_id, mode, item_count) values ($1, $2, 'MOCK', 5)`,
		"TOPIC có deadline":     `insert into practice_attempts (course_id, student_id, mode, topic, item_count, deadline_at) values ($1, $2, 'TOPIC', 'z', 5, now())`,
		"TOPIC thiếu chủ đề":    `insert into practice_attempts (course_id, student_id, mode, item_count) values ($1, $2, 'TOPIC', 5)`,
		"item_count 0":          `insert into practice_attempts (course_id, student_id, mode, topic, item_count) values ($1, $2, 'TOPIC', 'z', 0)`,
		"current_index quá":     `insert into practice_attempts (course_id, student_id, mode, topic, item_count, current_index) values ($1, $2, 'TOPIC', 'z', 3, 3)`,
		"đóng mà thiếu nộp lúc": `insert into practice_attempts (course_id, student_id, mode, topic, item_count, status) values ($1, $2, 'TOPIC', 'z', 3, 'SUBMITTED')`,
		"điểm ngoài [0, 10]":    `insert into practice_attempts (course_id, student_id, mode, topic, item_count, status, submitted_at, correct_count, score) values ($1, $2, 'TOPIC', 'z', 3, 'SUBMITTED', now(), 1, 10.5)`,
	} {
		_, err := conn.Exec(ctx, sql, f.c1, f.s1)
		require.Equal(t, "23514", code(err), name)
	}

	// UNIQUE từng phần: một lượt dở mỗi (sinh viên, lớp, chế độ, chủ đề); lượt đã đóng thì không tính
	_, err = conn.Exec(ctx, `insert into practice_attempts (course_id, student_id, mode, topic, item_count) values ($1, $2, 'TOPIC', 'x', 2)`, f.c1, f.s1)
	require.Equal(t, "23505", code(err), "hai lượt TOPIC dở cùng chủ đề")
	_, err = conn.Exec(ctx, `insert into practice_attempts (course_id, student_id, mode, topic, item_count) values ($1, $2, 'TOPIC', 'khác', 2)`, f.c1, f.s1)
	require.NoError(t, err, "chủ đề khác được")
	mock := `insert into practice_attempts (course_id, student_id, mode, item_count, deadline_at) values ($1, $2, 'MOCK', 5, now() + interval '30 minutes')`
	_, err = conn.Exec(ctx, mock, f.c1, f.s1)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, mock, f.c1, f.s1)
	require.Equal(t, "23505", code(err), "hai lượt MOCK dở")
	_, err = conn.Exec(ctx, `update practice_attempts set status = 'EXPIRED', submitted_at = now(), correct_count = 0 where course_id = $1 and student_id = $2 and mode = 'MOCK'`, f.c1, f.s1)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, mock, f.c1, f.s1)
	require.NoError(t, err, "lượt MOCK đã đóng không chặn lượt mới")

	// UNIQUE (attempt_id, position), UNIQUE (item_id)
	_, err = conn.Exec(ctx, `insert into practice_items (course_id, attempt_id, question_id, position, answer_key, question, question_version) values ($1, $2, $3, 0, '{}', '{}', 1)`, f.c1, att, q1)
	require.Equal(t, "23505", code(err))
	_, err = conn.Exec(ctx, `insert into practice_answers (course_id, attempt_id, item_id, answer) values ($1, $2, $3, '{}')`, f.c1, att, item)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `insert into practice_answers (course_id, attempt_id, item_id, answer) values ($1, $2, $3, '{}')`, f.c1, att, item)
	require.Equal(t, "23505", code(err))
}

// TestMigration00019UpDownUp — AC1: `down` (về 0) gỡ sạch 5 bảng + 2 enum của 00019 và up lại được.
func TestMigration00019UpDownUp(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)
	url := conn.Config().ConnString()
	count := func(q string) int {
		var n int
		require.NoError(t, conn.QueryRow(ctx, q).Scan(&n))
		return n
	}
	tables := func() int {
		return count(`select count(*) from information_schema.tables where table_schema='public' and table_name in ('practice_attempts','practice_items','practice_answers','forms_import_batches','forms_import_rows')`)
	}
	enums := func() int {
		return count(`select count(*) from pg_type where typtype = 'e' and typname in ('practice_mode','practice_status')`)
	}
	require.Equal(t, 5, tables())
	require.Equal(t, 2, enums())
	require.NoError(t, migrateCmd(ctx, url, "down"))
	require.Equal(t, 0, tables())
	require.Equal(t, 0, enums())
	require.NoError(t, migrateCmd(ctx, url, "up"))
	require.Equal(t, 5, tables())
	require.Equal(t, 2, enums())
}
