package seedtool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// GuidanceDemoResults là số kết quả ĐÃ DUYỆT mà seed «học từ lần sửa» tạo cho bài tập (≥ GUIDANCE_MIN_RESULTS để `Rút lưu ý` dùng được).
const GuidanceDemoResults = 8

// SeedGuidance (10.6 L6, SEED_GUIDANCE_DEMO=1): bật cờ GRADING_GUIDANCE cho lớp và đưa GuidanceDemoResults kết quả ĐÃ DUYỆT vào bài tập `code`:
// ở tiêu chí ĐẦU TIÊN giảng viên luôn chấm thấp hơn AI 1,0 điểm («sở thích gắt»), các tiêu chí khác giữ nguyên điểm AI. Ghi thẳng DB vì demo không được phụ thuộc LLM thật.
// Chỉ gọi từ `gateway admin seed-guidance` (dev | test). Idempotent: đủ GuidanceDemoResults kết quả đã duyệt thì không thêm. Trả số kết quả mới.
func SeedGuidance(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, code string) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("seed lưu ý: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var asg, teacher uuid.UUID
	var rubricRaw []byte
	err = tx.QueryRow(ctx, `select id, rubric, created_by from assignments where course_id = $1 and submission_code = $2 and type = 'ESSAY'`, courseID, code).Scan(&asg, &rubricRaw, &teacher)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("không có bài tập tự luận %q", code)
	}
	if err != nil {
		return 0, fmt.Errorf("seed lưu ý: đọc bài tập: %w", err)
	}
	var rubric []struct {
		ID       string `json:"id"`
		MaxScore string `json:"max_score"`
	}
	if json.Unmarshal(rubricRaw, &rubric) != nil || len(rubric) == 0 {
		return 0, errors.New("bài tập chưa có rubric")
	}
	if _, err := tx.Exec(ctx, `insert into course_guidance_flags (course_id, feature, enabled, updated_by) values ($1, 'GRADING_GUIDANCE', true, $2)
		on conflict (course_id, feature) do update set enabled = true`, courseID, teacher); err != nil {
		return 0, fmt.Errorf("seed lưu ý: bật cờ: %w", err)
	}
	var have int
	if err := tx.QueryRow(ctx, `select count(*)::int from grading_results r join submissions s on s.course_id = r.course_id and s.id = r.submission_id
		where r.course_id = $1 and s.assignment_id = $2 and r.status = 'REVIEWED'`, courseID, asg).Scan(&have); err != nil {
		return 0, fmt.Errorf("seed lưu ý: đếm: %w", err)
	}
	need := GuidanceDemoResults - have
	if need <= 0 {
		return 0, tx.Commit(ctx)
	}
	rows, err := tx.Query(ctx, `select s.id from submissions s
		where s.course_id = $1 and s.assignment_id = $2 and s.match_status = 'MATCHED'
		  and not exists (select 1 from grading_results r where r.course_id = s.course_id and r.submission_id = s.id)
		  and not exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version)
		order by s.id limit $3`, courseID, asg, need)
	if err != nil {
		return 0, fmt.Errorf("seed lưu ý: bài nộp: %w", err)
	}
	var subs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("seed lưu ý: bài nộp: %w", err)
		}
		subs = append(subs, id)
	}
	rows.Close()
	type crit struct {
		ID       string   `json:"id"`
		Score    string   `json:"score"`
		Comment  string   `json:"comment"`
		Edited   bool     `json:"edited"`
		Evidence []string `json:"evidence"`
	}
	var crits []crit
	ai := map[string]string{}
	comments := map[string]string{}
	var aiSum, tSum decimal.Decimal
	quarter := decimal.NewFromFloat(0.25)
	for i, r := range rubric {
		mx, err := decimal.NewFromString(r.MaxScore)
		if err != nil {
			return 0, fmt.Errorf("rubric hỏng: %w", err)
		}
		a := mx.Mul(decimal.NewFromFloat(0.8)).Div(quarter).Round(0).Mul(quarter) // 80 % điểm tối đa, làm tròn bước 0,25
		t := a
		if i == 0 {
			t = decimal.Max(a.Sub(decimal.NewFromInt(1)), decimal.Zero)
		}
		ai[r.ID], comments[r.ID] = a.StringFixed(2), "Nhận xét mẫu của AI."
		crits = append(crits, crit{ID: r.ID, Score: t.StringFixed(2), Edited: !t.Equal(a), Evidence: []string{}})
		aiSum, tSum = aiSum.Add(a), tSum.Add(t)
	}
	cj, _ := json.Marshal(crits)
	aj, _ := json.Marshal(ai)
	cm, _ := json.Marshal(map[string]any{"criteria": comments, "overall": "Nhận xét tổng quát mẫu."})
	for _, sub := range subs {
		if _, err := tx.Exec(ctx, `insert into grading_results (course_id, submission_id, run_state, status, criteria, ai_scores, ai_comments, ai_total_before_late, total, reviewed_by, reviewed_at, first_reviewed_by, first_reviewed_at)
			values ($1, $2, 'DONE', 'REVIEWED', $3::jsonb, $4::jsonb, $5::jsonb, $6, $7, $8, now(), $8, now())`, courseID, sub, cj, aj, cm, aiSum, tSum, teacher); err != nil {
			return 0, fmt.Errorf("seed lưu ý: ghi kết quả: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("seed lưu ý: %w", err)
	}
	return int64(len(subs)), nil
}
