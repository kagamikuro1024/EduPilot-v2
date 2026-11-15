package seedtool

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedCheckinDemo (lệnh dev `gateway admin seed-checkin-demo`, FEAT-attendance-selfcheck 11.6) bật điểm danh tự phục vụ cho lớp và dựng MỘT cửa sổ đã đóng
// (vòng 1) ở buổi đầu chưa có cửa sổ, cho nửa số sinh viên ACTIVE điểm danh, hai người trong số đó mang cờ DEVICE_SHARED để mục «Hôm nay» CHECKIN_REVIEW
// xuất hiện. Ghi PRESENT cho người đã điểm danh (như lúc đóng cửa sổ); giảng viên vẫn phải «Đánh vắng người chưa điểm danh» rồi hoàn tất như thật. Trả số lượt đã ghi.
func SeedCheckinDemo(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("seedtool: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `update courses set settings = settings || jsonb_build_object('attendance', coalesce(settings->'attendance', '{}'::jsonb) || '{"self_checkin": true}'::jsonb) where id = $1`, courseID); err != nil {
		return 0, fmt.Errorf("seedtool: bật điểm danh tự phục vụ: %w", err)
	}
	var sid uuid.UUID
	var teacher uuid.UUID
	err = tx.QueryRow(ctx, `select s.id, e.user_id from class_sessions s
		join enrollments e on e.course_id = s.course_id and e.role_in_course = 'TEACHER' and e.status = 'ACTIVE'
		where s.course_id = $1 and not exists (select 1 from checkin_windows w where w.course_id = s.course_id and w.session_id = s.id)
		order by s.session_no limit 1`, courseID).Scan(&sid, &teacher)
	if err != nil {
		return 0, fmt.Errorf("seedtool: không có buổi trống cửa sổ hoặc giảng viên: %w", err)
	}
	var wid uuid.UUID
	if err := tx.QueryRow(ctx, `insert into checkin_windows (course_id, session_id, round, rotate_seconds, opened_by, opened_at, closes_at, closed_at, closed_by)
		values ($1, $2, 1, 15, $3, now() - interval '70 minutes', now() - interval '60 minutes', now() - interval '60 minutes', $3) returning id`, courseID, sid, teacher).Scan(&wid); err != nil {
		return 0, fmt.Errorf("seedtool: tạo cửa sổ: %w", err)
	}
	var devs [2]uuid.UUID
	for i := range devs {
		if err := tx.QueryRow(ctx, `insert into user_devices (key_sha256, public_key, kind) values (sha256(gen_random_uuid()::text::bytea), '\x00', 'WEB_KEY') returning id`).Scan(&devs[i]); err != nil {
			return 0, fmt.Errorf("seedtool: tạo thiết bị: %w", err)
		}
	}
	rows, err := tx.Query(ctx, `select user_id from enrollments where course_id = $1 and role_in_course = 'STUDENT' and status = 'ACTIVE' order by user_id`, courseID)
	if err != nil {
		return 0, fmt.Errorf("seedtool: đọc sinh viên: %w", err)
	}
	var students []uuid.UUID
	for rows.Next() {
		var u uuid.UUID
		if err := rows.Scan(&u); err != nil {
			rows.Close()
			return 0, fmt.Errorf("seedtool: đọc sinh viên: %w", err)
		}
		students = append(students, u)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("seedtool: đọc sinh viên: %w", err)
	}
	var n int64
	for i, u := range students[:(len(students)+1)/2] {
		flags, dev := []string{}, (*uuid.UUID)(nil)
		if i < 2 { // hai người dùng chung một máy
			flags, dev = []string{"DEVICE_SHARED"}, &devs[0]
		}
		if _, err := tx.Exec(ctx, `insert into attendance_checkins (course_id, session_id, window_id, student_id, device_id, channel, assurance, token_age_ms, flags, created_at)
			values ($1, $2, $3, $4, $5, 'WEB', $6, 1500, $7, now() - interval '65 minutes' + ($8 * interval '5 seconds'))`, courseID, sid, wid, u, dev, map[bool]int{true: 2, false: 1}[dev != nil], flags, i); err != nil {
			return 0, fmt.Errorf("seedtool: ghi lượt điểm danh: %w", err)
		}
		if _, err := tx.Exec(ctx, `insert into attendance_records (course_id, session_id, student_id, status, marked_by) values ($1, $2, $3, 'PRESENT', $4)`, courseID, sid, u, teacher); err != nil {
			return 0, fmt.Errorf("seedtool: ghi điểm danh: %w", err)
		}
		n++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("seedtool: commit: %w", err)
	}
	return n, nil
}
