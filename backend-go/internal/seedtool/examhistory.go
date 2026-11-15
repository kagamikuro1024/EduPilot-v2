package seedtool

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// examHistoryTitles: bốn bài thi PE đã công bố của lớp (mở cách hôm nay 28 / 21 / 14 / 7 ngày); khoá tự nhiên = tiêu đề, dùng cho idempotency.
var examHistoryTitles = [4]string{"Kiểm tra tuần 2 — Ôn tập", "Kiểm tra tuần 4 — Ôn tập", "Kiểm tra tuần 6 — Ôn tập", "Kiểm tra tuần 8 — Ôn tập"} //nolint:gochecknoglobals // hằng

// historyCorrect: số câu đúng (trên 4 câu, mỗi câu 2,5 điểm) của sinh viên thứ i ở bài thi e. -1 = vắng.
// Mặc định KHÔNG bao giờ giảm (2–3 câu rồi +1 từ bài thứ 3) để chỉ các ca dưới đây vào «Điểm thi đang giảm» (US-P10-06).
func historyCorrect(i, e int) int {
	switch i {
	case 0: // giảm nghiêm ngặt 4 → 3 → 2 → 1: bộ ba mới nhất (bài 2–4) vào danh sách
		return [4]int{4, 3, 2, 1}[e]
	case 1: // giảm 3 liên tiếp ở bài 1–3 rồi hồi phục: bộ ba mới nhất vẫn là bài 1–3 → vào danh sách
		return [4]int{4, 3, 2, 3}[e]
	case 2: // vắng bài 3 làm đứt chuỗi → không vào
		return [4]int{4, 3, -1, 1}[e]
	case 3: // chỉ giảm 2 bài rồi đi ngang → không vào
		return [4]int{4, 3, 3, 3}[e]
	case 4: // không giảm nghiêm ngặt → không vào
		return [4]int{3, 3, 3, 2}[e]
	}
	base := 2 + i%2
	if e >= 2 {
		base++
	}
	return base
}

// SeedExamHistory ghi bốn bài thi PE `PUBLISHED` quá khứ (opens_at 28 / 21 / 14 / 7 ngày trước) kèm lượt GRADED của tối đa 20 sinh viên ACTIVE đầu tiên (theo email),
// để màn Điểm thi có phân bố, xu hướng, câu sai nhiều và «điểm đang giảm» (US-P10-07 AC2). Ghi thẳng DB vì bài thi thật mở theo giờ; idempotent theo tiêu đề bài.
// Trả số lượt ghi (0 nếu lớp đã có đủ bài). Cần ≥ 4 câu MCQ_SINGLE APPROVED trong ngân hàng của lớp.
func SeedExamHistory(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, now time.Time) (int, error) {
	var have int
	if err := pool.QueryRow(ctx, `select count(*) from exams where course_id = $1 and title = any ($2)`, courseID, examHistoryTitles[:]).Scan(&have); err != nil {
		return 0, fmt.Errorf("seed-exam-history: kiểm lớp: %w", err)
	}
	if have == len(examHistoryTitles) {
		return 0, nil
	}
	var teacher uuid.UUID
	if err := pool.QueryRow(ctx, `select user_id from enrollments where course_id = $1 and role_in_course = 'TEACHER' and status = 'ACTIVE' order by created_at limit 1`, courseID).Scan(&teacher); err != nil {
		return 0, fmt.Errorf("seed-exam-history: giảng viên của lớp: %w", err)
	}
	rows, err := pool.Query(ctx, `select e.user_id from enrollments e join users u on u.id = e.user_id where e.course_id = $1 and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE' order by u.email limit 20`, courseID)
	if err != nil {
		return 0, fmt.Errorf("seed-exam-history: sinh viên: %w", err)
	}
	var students []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("seed-exam-history: đọc sinh viên: %w", err)
		}
		students = append(students, id)
	}
	rows.Close()
	qrows, err := pool.Query(ctx, `select id from question_bank where course_id = $1 and type = 'MCQ_SINGLE' and review_status = 'APPROVED' order by title, id limit 4`, courseID)
	if err != nil {
		return 0, fmt.Errorf("seed-exam-history: câu hỏi: %w", err)
	}
	var questions []uuid.UUID
	for qrows.Next() {
		var id uuid.UUID
		if err := qrows.Scan(&id); err != nil {
			return 0, fmt.Errorf("seed-exam-history: đọc câu hỏi: %w", err)
		}
		questions = append(questions, id)
	}
	qrows.Close()
	if len(questions) < 4 || len(students) == 0 {
		return 0, fmt.Errorf("seed-exam-history: lớp cần ≥ 4 câu MCQ_SINGLE APPROVED (có %d) và ≥ 1 sinh viên (có %d)", len(questions), len(students))
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("seed-exam-history: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	n := 0
	for e, title := range examHistoryTitles {
		var exists bool
		if err := tx.QueryRow(ctx, `select exists (select 1 from exams where course_id = $1 and title = $2)`, courseID, title).Scan(&exists); err != nil {
			return 0, fmt.Errorf("seed-exam-history: kiểm bài: %w", err)
		}
		if exists {
			continue
		}
		opens := now.AddDate(0, 0, -7*(4-e)).Truncate(time.Hour)
		var eid uuid.UUID
		if err := tx.QueryRow(ctx, `insert into exams (course_id, title, status, opens_at, closes_at, duration_minutes, max_score, published_at, created_by)
			values ($1, $2, 'PUBLISHED', $3, $4, 30, 10, $4, $5) returning id`, courseID, title, opens, opens.Add(2*time.Hour), teacher).Scan(&eid); err != nil {
			return 0, fmt.Errorf("seed-exam-history: ghi bài: %w", err)
		}
		items := make([]uuid.UUID, len(questions))
		for k, q := range questions {
			if err := tx.QueryRow(ctx, `insert into exam_items (course_id, exam_id, question_id, position, points) values ($1, $2, $3, $4, 2.5) returning id`, courseID, eid, q, k+1).Scan(&items[k]); err != nil {
				return 0, fmt.Errorf("seed-exam-history: ghi câu của bài: %w", err)
			}
		}
		for i, s := range students {
			c := historyCorrect(i, e)
			if c < 0 {
				continue
			}
			type row struct {
				ItemID uuid.UUID `json:"item_id"`
				Points string    `json:"points"`
				Max    string    `json:"max"`
				Earned string    `json:"earned"`
			}
			bd := make([]row, len(items))
			for k, it := range items {
				earned := "0"
				if k < c {
					earned = "2.5"
				}
				bd[k] = row{it, "2.5", "2.5", earned}
			}
			raw, _ := json.Marshal(bd) //nolint:errchkjson // kiểu cố định
			start := opens.Add(10 * time.Minute)
			if _, err := tx.Exec(ctx, `insert into exam_attempts (course_id, exam_id, student_id, status, started_at, deadline_at, submitted_at, submit_reason, auto_score, breakdown, graded_at)
				values ($1, $2, $3, 'GRADED', $4, $5, $6, 'MANUAL', $7, $8::jsonb, $6)`, courseID, eid, s, start, start.Add(30*time.Minute), start.Add(25*time.Minute), float64(c)*2.5, string(raw)); err != nil {
				return 0, fmt.Errorf("seed-exam-history: ghi lượt: %w", err)
			}
			n++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("seed-exam-history: chốt: %w", err)
	}
	return n, nil
}
