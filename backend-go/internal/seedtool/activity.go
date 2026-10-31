package seedtool

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ActivityWant là yêu cầu hoạt động của một sinh viên (khớp theo MSSV của dòng ghi danh ACTIVE) ở hai tuần trọn gần nhất:
// "0" = không hoạt động nào, ">=1" = ít nhất một. Nguồn: cột `activity_week9` / `activity_week8` của seed/expected_risk.csv (US-P6-09 AC8).
type ActivityWant struct {
	StudentCode string `json:"student_code"`
	Week9       string `json:"week9"` // tuần trọn gần nhất (W1 của at-risk)
	Week8       string `json:"week8"` // tuần trọn liền trước (W2)
}

// ParseActivityPlan đọc `[]ActivityWant`; giá trị lạ → lỗi (không đoán).
func ParseActivityPlan(raw []byte) ([]ActivityWant, error) {
	var p []ActivityWant
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("kế hoạch hoạt động không phải JSON: %w", err)
	}
	for _, w := range p {
		for _, v := range []string{w.Week9, w.Week8} {
			if v != "0" && v != ">=1" && v != "" {
				return nil, fmt.Errorf("sinh viên %s: giá trị %q (chỉ \"0\", \">=1\" hoặc rỗng)", w.StudentCode, v)
			}
		}
	}
	return p, nil
}

// PlaceActivity đặt hoạt động đúng yêu cầu ở tuần 9 / tuần 8 bằng cách DỜI `created_at` của tin chat USER / thread / bình luận đã có (không thêm nội dung). Chạy sau `AgeActivity`,
// trước `crm-risk-run`. "0": dời mọi hoạt động của tuần đó về 4 tuần trước; ">=1" mà tuần trống: dời một tin chat USER nằm ngoài hai tuần đó vào giữa tuần. Chạy lại không đổi gì thêm.
func PlaceActivity(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, plan []ActivityWant, now time.Time) (int64, error) {
	cur := monday(now)
	cw := time.Date(cur.Year(), cur.Month(), cur.Day(), 0, 0, 0, 0, ict)
	w9, w8 := cw.AddDate(0, 0, -7), cw.AddDate(0, 0, -14)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var moved int64
	count := func(uid uuid.UUID, from, to time.Time) (n int, err error) {
		err = tx.QueryRow(ctx, `select
			(select count(*) from chat_messages m join chat_sessions cs on cs.id = m.session_id where m.course_id = $1 and m.user_id = $2 and m.role = 'USER' and cs.deleted_at is null and m.created_at >= $3 and m.created_at < $4)
		  + (select count(*) from forum_threads t where t.course_id = $1 and t.author_id = $2 and t.deleted_at is null and t.created_at >= $3 and t.created_at < $4)
		  + (select count(*) from forum_posts p where p.course_id = $1 and p.author_id = $2 and p.kind = 'HUMAN' and p.deleted_at is null and p.created_at >= $3 and p.created_at < $4)`, courseID, uid, from, to).Scan(&n)
		return n, err
	}
	type job struct {
		uid      uuid.UUID
		code     string
		from, to time.Time
		zero     bool
	}
	var zeros, fills []job
	for _, w := range plan {
		var uid uuid.UUID
		if err := tx.QueryRow(ctx, `select user_id from enrollments where course_id = $1 and role_in_course = 'STUDENT' and status = 'ACTIVE' and student_code_snapshot = $2`, courseID, w.StudentCode).Scan(&uid); err != nil {
			return 0, fmt.Errorf("MSSV %s không ACTIVE ở lớp: %w", w.StudentCode, err)
		}
		for _, x := range []struct {
			want     string
			from, to time.Time
		}{{w.Week8, w8, w9}, {w.Week9, w9, cw}} { // W2 trước, W1 sau: hai lần đặt không giành cùng một tin
			switch x.want {
			case "0":
				zeros = append(zeros, job{uid, w.StudentCode, x.from, x.to, true})
			case ">=1":
				fills = append(fills, job{uid, w.StudentCode, x.from, x.to, false})
			}
		}
	}
	for _, j := range zeros {
		for _, q := range []string{
			`update chat_messages set created_at = created_at - interval '28 days' where course_id = $1 and user_id = $2 and role = 'USER' and created_at >= $3 and created_at < $4`,
			`update forum_threads set created_at = created_at - interval '28 days' where course_id = $1 and author_id = $2 and created_at >= $3 and created_at < $4`,
			`update forum_posts set created_at = created_at - interval '28 days' where course_id = $1 and author_id = $2 and kind = 'HUMAN' and created_at >= $3 and created_at < $4`,
		} {
			tag, err := tx.Exec(ctx, q, courseID, j.uid, j.from, j.to)
			if err != nil {
				return 0, fmt.Errorf("dời hoạt động của %s: %w", j.code, err)
			}
			moved += tag.RowsAffected()
		}
	}
	for _, j := range fills {
		n, err := count(j.uid, j.from, j.to)
		if err != nil {
			return 0, err
		}
		if n > 0 {
			continue
		}
		tag, err := tx.Exec(ctx, `update chat_messages set created_at = $5 where id = (
			select id from chat_messages where course_id = $1 and user_id = $2 and role = 'USER' and (created_at < $3 or created_at >= $4) order by id limit 1)`,
			courseID, j.uid, w8, cw, j.from.Add(58*time.Hour))
		if err != nil {
			return 0, fmt.Errorf("đặt hoạt động của %s: %w", j.code, err)
		}
		if tag.RowsAffected() == 0 { // không còn tin nào để dời (mọi tin đã nằm trong hai tuần này): thêm một câu hỏi, khoá cố định theo MSSV + tuần
			at := j.from.Add(58 * time.Hour)
			sid, mid := detID("place/"+j.code+"/"+j.from.Format(time.DateOnly)+"/s"), detID("place/"+j.code+"/"+j.from.Format(time.DateOnly)+"/q")
			if _, err := tx.Exec(ctx, `insert into chat_sessions (id, course_id, user_id, title, created_at, updated_at, last_message_at) values ($1, $2, $3, 'Câu hỏi ôn tập', $4, $4, $4) on conflict (id) do nothing`, sid, courseID, j.uid, at); err != nil {
				return 0, fmt.Errorf("thêm phiên chat của %s: %w", j.code, err)
			}
			if _, err := tx.Exec(ctx, `insert into chat_messages (id, course_id, session_id, user_id, role, content, client_msg_id, completed_at, created_at, updated_at) values ($1, $2, $3, $4, 'USER', 'Em xin nhắc lại nội dung buổi học.', $1, $5, $5, $5) on conflict (id) do nothing`, mid, courseID, sid, j.uid, at); err != nil {
				return 0, fmt.Errorf("thêm câu hỏi của %s: %w", j.code, err)
			}
			moved++
			continue
		}
		moved += tag.RowsAffected()
	}
	return moved, tx.Commit(ctx)
}
