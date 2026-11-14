package seedtool

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// llmAuditMarker mở đầu trace_id của mọi dòng do seed ghi: dấu để chạy lại không nhân đôi (idempotent theo lớp).
const llmAuditMarker = "seed-obs-"

// SeedLLMAudit rải yêu cầu LLM mô phỏng 7 ngày của MỘT lớp (≥ 300 dòng, quan sát AI — US-P10-07): đa số thành công, vài lỗi / bị giới hạn / chuyển dự phòng,
// ≈ 10 % có che thông tin; mọi dòng chat có bản prompt ĐÃ CHE (chỉ nhãn [STUDENT_n], không tên / MSSV thật). Trả số dòng ghi (0 nếu lớp đã có).
// Tất định theo `now` (cùng ngày chạy lại cho cùng kết quả).
func SeedLLMAudit(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, now time.Time) (int, error) {
	var have int
	if err := pool.QueryRow(ctx, `select count(*) from llm_audit where course_id = $1 and trace_id like $2`, courseID, llmAuditMarker+"%").Scan(&have); err != nil {
		return 0, fmt.Errorf("seed-llm-audit: kiểm lớp: %w", err)
	}
	if have > 0 {
		return 0, nil
	}
	tasks := []string{"CHAT", "CHAT", "CHAT", "CLASSIFY", "UTILITY", "GRADING", "EMBEDDING", "INSIGHT"}
	questions := []string{
		"Em là [STUDENT_1], MSSV [STUDENT_1_CODE], cho em hỏi hạn nộp bài tập 2 ạ?",
		"Điều kiện dự thi cuối kỳ là gì?",
		"Phishing khác spear phishing ở điểm nào?",
		"Bạn [STUDENT_2] nói hôm trước nghỉ có phép, vậy điểm danh tính thế nào?",
		"Ransomware lây lan qua đường nào?",
		"Email [STUDENT_3_EMAIL] chưa nhận được thư mời vào lớp.",
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("seed-llm-audit: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	n := 0
	for day := 0; day < 7; day++ {
		for i := 0; i < 48; i++ {
			k := day*48 + i
			h := hash32(fmt.Sprintf("llm-audit/%s/%d", courseID, k))
			at := now.Add(-time.Duration(day)*24*time.Hour - time.Duration(h%86000)*time.Second)
			task := tasks[h%uint32(len(tasks))] //nolint:gosec // len cố định nhỏ
			status, kind, fb := "ok", "", 0
			switch {
			case h%97 == 0:
				status, kind = "error", "SERVER"
			case h%53 == 0:
				status, kind = "rate_limited", "RATE_LIMIT"
			case h%41 == 0:
				status, fb = "ok", 1
			}
			pii := 0
			if task == "CHAT" && h%10 == 0 {
				pii = 1 + int(h%3)
			}
			latency := 300 + int(h%3200)
			if task == "EMBEDDING" {
				latency = 40 + int(h%200)
			}
			id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("edupilot-seed-llm-audit/%s/%d", courseID, k)))
			var kindArg any
			if kind != "" {
				kindArg = kind
			}
			if _, err := tx.Exec(ctx, `insert into llm_audit (id, task, lane, provider, model, tokens_in, tokens_out, latency_ms, queue_wait_ms, attempts, fallback_index, cost_est, status, error_kind, degraded, pii_masked_count, course_id, trace_id, created_at)
				values ($1, $2, 'INTERACTIVE', 'fake', 'fake-chat', $3, $4, $5, 0, 1, $6, $7, $8, $9, false, $10, $11, $12, $13) on conflict (id) do nothing`,
				id, task, 200+int(h%400), 60+int(h%200), latency, fb, 12+int(h%20), status, kindArg, pii, courseID, fmt.Sprintf("%s%s-%d", llmAuditMarker, courseID.String()[:8], k), at); err != nil {
				return 0, fmt.Errorf("seed-llm-audit: ghi llm_audit: %w", err)
			}
			if task == "CHAT" && status == "ok" {
				prompt := "[system]\nBạn là trợ giảng của lớp. Chỉ trả lời từ tài liệu.\n\n[user]\n" + questions[h%uint32(len(questions))]
				if _, err := tx.Exec(ctx, `insert into llm_request_payloads (audit_id, course_id, masked_prompt, tools, created_at) values ($1, $2, $3, $4, $5) on conflict do nothing`,
					id, courseID, prompt, `[]`, at); err != nil {
					return 0, fmt.Errorf("seed-llm-audit: ghi payload: %w", err)
				}
			}
			n++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("seed-llm-audit: commit: %w", err)
	}
	return n, nil
}
