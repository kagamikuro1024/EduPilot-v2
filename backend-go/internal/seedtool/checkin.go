package seedtool

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedCheckinDemo (lệnh dev `gateway admin seed-checkin-demo`, FEAT-attendance-selfcheck 11.6, US-AS-14 AC1) bật điểm danh tự phục vụ cho lớp và dựng hai cửa sổ ĐÃ ĐÓNG ở buổi
// đầu tiên của lớp:
//   - vòng 1: 22 lượt (hoặc ít hơn nếu lớp nhỏ) đủ các ca — không cờ, một cặp DEVICE_SHARED, DEVICE_NEW, NO_DEVICE_KEY, TOKEN_LATE, một sinh viên LATE;
//   - vòng 2: mọi người trừ người cuối ⇒ người cuối mang RECHECK_MISS ở lượt vòng 1; headcount thấp hơn số lượt ⇒ over_headcount.
//
// Sinh viên `sv.gioi` không có lượt nào (dùng để thử điểm danh tay khi demo). Ghi PRESENT / LATE cho người đã điểm danh như lúc đóng cửa sổ; giảng viên vẫn phải «Đánh vắng
// người chưa điểm danh» rồi hoàn tất như thật. Chạy lại là không đổi: IDEMPOTENT: buổi đầu đã có cửa sổ thì không làm gì (trả 0). Chỉ gọi từ lệnh dev | test.
func SeedCheckinDemo(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("seedtool: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `update courses set settings = settings || jsonb_build_object('attendance', coalesce(settings->'attendance', '{}'::jsonb) || '{"self_checkin": true}'::jsonb) where id = $1`, courseID); err != nil {
		return 0, fmt.Errorf("seedtool: bật điểm danh tự phục vụ: %w", err)
	}
	var sid, teacher uuid.UUID
	var taken bool
	err = tx.QueryRow(ctx, `select s.id, e.user_id, exists (select 1 from checkin_windows w where w.course_id = s.course_id and w.session_id = s.id)
		from class_sessions s join enrollments e on e.course_id = s.course_id and e.role_in_course = 'TEACHER' and e.status = 'ACTIVE'
		where s.course_id = $1 order by s.session_no limit 1`, courseID).Scan(&sid, &teacher, &taken)
	if err != nil {
		return 0, fmt.Errorf("seedtool: không có buổi hoặc giảng viên: %w", err)
	}
	if taken {
		return 0, tx.Commit(ctx) // chạy lại: giữ nguyên, số hàng không đổi
	}
	rows, err := tx.Query(ctx, `select e.user_id from enrollments e join users u on u.id = e.user_id
		where e.course_id = $1 and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE' and u.email <> 'sv.gioi@edupilot.local' order by e.user_id`, courseID)
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
	if len(students) > 22 {
		students = students[:22]
	}
	newDevice := func() (uuid.UUID, error) {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `insert into user_devices (key_sha256, public_key, kind) values (sha256(gen_random_uuid()::text::bytea), '\x00', 'WEB_KEY') returning id`).Scan(&id)
		return id, err
	}
	window := func(round int, openedAgo, closedAgo string, headcount any) (uuid.UUID, error) {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `insert into checkin_windows (course_id, session_id, round, rotate_seconds, headcount, opened_by, opened_at, closes_at, closed_at, closed_by)
			values ($1, $2, $3, 15, $4, $5, now() - $6::interval, now() - $7::interval, now() - $7::interval, $5) returning id`, courseID, sid, round, headcount, teacher, openedAgo, closedAgo).Scan(&id)
		return id, err
	}
	w1, err := window(1, "95 minutes", "65 minutes", nil)
	if err != nil {
		return 0, fmt.Errorf("seedtool: tạo cửa sổ vòng 1: %w", err)
	}
	var n int64
	devs := make([]*uuid.UUID, len(students))
	for i, u := range students {
		var dev *uuid.UUID
		flags, assurance, channel, age := []string{}, 2, "WEB", 1500
		switch {
		case i < 2: // một cặp dùng chung một máy
			flags = []string{"DEVICE_SHARED"}
			if i == 0 {
				d, err := newDevice()
				if err != nil {
					return 0, fmt.Errorf("seedtool: tạo thiết bị: %w", err)
				}
				dev = &d
			} else {
				dev = devs[0]
			}
		case i == 2:
			flags = []string{"DEVICE_NEW"}
		case i == 3:
			flags, assurance = []string{"NO_DEVICE_KEY"}, 1
		case i == 4:
			flags, age = []string{"TOKEN_LATE"}, 24000
		}
		if dev == nil && assurance == 2 {
			d, err := newDevice()
			if err != nil {
				return 0, fmt.Errorf("seedtool: tạo thiết bị: %w", err)
			}
			dev = &d
		}
		devs[i] = dev
		offset := fmt.Sprintf("%d seconds", 30+i*45) // dồn trong ~17 phút đầu
		status := "PRESENT"
		if i == 5 { // LATE: sau mốc «tính muộn» 15 phút
			offset, status = "1200 seconds", "LATE"
		}
		if _, err := tx.Exec(ctx, `insert into attendance_checkins (course_id, session_id, window_id, student_id, device_id, channel, assurance, token_age_ms, flags, created_at)
			values ($1, $2, $3, $4, $5, $6::checkin_channel, $7, $8, $9, now() - interval '95 minutes' + $10::interval)`, courseID, sid, w1, u, dev, channel, assurance, age, flags, offset); err != nil {
			return 0, fmt.Errorf("seedtool: ghi lượt điểm danh: %w", err)
		}
		if _, err := tx.Exec(ctx, `insert into attendance_records (course_id, session_id, student_id, status, marked_by) values ($1, $2, $3, $4::attendance_status, $5)`, courseID, sid, u, status, teacher); err != nil {
			return 0, fmt.Errorf("seedtool: ghi điểm danh: %w", err)
		}
		n++
	}
	if len(students) >= 2 {
		// vòng 2: mọi người trừ người cuối; headcount = số lượt − 4 ⇒ over_headcount
		w2, err := window(2, "60 minutes", "55 minutes", len(students)-5)
		if err != nil {
			return 0, fmt.Errorf("seedtool: tạo cửa sổ vòng 2: %w", err)
		}
		for i, u := range students[:len(students)-1] {
			if _, err := tx.Exec(ctx, `insert into attendance_checkins (course_id, session_id, window_id, student_id, device_id, channel, assurance, token_age_ms, flags, created_at)
				values ($1, $2, $3, $4, $5, 'WEB', $6, 1500, '{}', now() - interval '60 minutes' + $7::interval)`, courseID, sid, w2, u, devs[i], map[bool]int{true: 2, false: 1}[devs[i] != nil], fmt.Sprintf("%d seconds", 20+i*10)); err != nil {
				return 0, fmt.Errorf("seedtool: ghi lượt vòng 2: %w", err)
			}
			n++
		}
		if _, err := tx.Exec(ctx, `update attendance_checkins set flags = array_append(flags, 'RECHECK_MISS') where window_id = $1 and student_id = $2`, w1, students[len(students)-1]); err != nil {
			return 0, fmt.Errorf("seedtool: cờ RECHECK_MISS: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("seedtool: commit: %w", err)
	}
	return n, nil
}
