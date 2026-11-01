package grade

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// RiskQT trả điểm quá trình HIỆN TẠI (thang 10, sau điểm cộng / trừ) của từng sinh viên ACTIVE mà mọi thành phần quá trình đã đủ điểm (SRS FEAT-gradebook 4.8.4 — tín hiệu
// `GRADE` của at-risk). Lớp chưa có công thức CONFIRMED → map rỗng. Giao diện chỉ đọc cho `crm` (không import ngược): đầu vào CẢ LỚP đọc theo lô `ANY($ids)`, rồi `Compute`
// trong bộ nhớ (không truy vấn riêng từng sinh viên — TLR-16). Không đọc thời gian học (D17).
func RiskQT(ctx context.Context, q *store.Queries, courseID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	conf, ok, err := LoadConfirmed(ctx, q, courseID)
	if err != nil || !ok {
		return map[uuid.UUID]decimal.Decimal{}, err
	}
	studs, err := ActiveStudents(ctx, q, courseID)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]decimal.Decimal, len(studs))
	for lo := 0; lo < len(studs); lo += finalizeChunk {
		part := studs[lo:min(lo+finalizeChunk, len(studs))]
		ids := make([]uuid.UUID, len(part))
		for i, st := range part {
			ids[i] = st.ID
		}
		in, err := LoadInputs(ctx, q, courseID, conf.ID, ids)
		if err != nil {
			return nil, fmt.Errorf("grade: đầu vào cho tín hiệu GRADE: %w", err)
		}
		for _, id := range ids {
			if r := WhatIf(conf.Scheme, in[id], decimal.Zero); r.HasQT { // cuối kỳ giả định chỉ để engine không dừng ở thành phần cuối kỳ còn thiếu
				out[id] = r.QT
			}
		}
	}
	return out, nil
}
