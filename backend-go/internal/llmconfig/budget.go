package llmconfig

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/edupilot/backend-go/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// Phạm vi ngân sách.
const (
	ScopeSystem = "system"
	ScopeCourse = "course"
)

// Budget là hạn mức chi phí (VND) của hệ thống hoặc một lớp; nil = không giới hạn. Version 0 = chưa từng đặt.
type Budget struct {
	Scope    string
	CourseID *uuid.UUID
	Daily    *decimal.Decimal
	Monthly  *decimal.Decimal
	Version  int
}

func readBudget(ctx context.Context, q *store.Queries, scope string, courseID *uuid.UUID) (Budget, error) {
	b := Budget{Scope: scope, CourseID: courseID}
	row, err := q.GetLLMBudget(ctx, store.GetLLMBudgetParams{Scope: scope, CourseID: courseID})
	if errors.Is(err, pgx.ErrNoRows) {
		return b, nil
	}
	if err != nil {
		return Budget{}, fmt.Errorf("llmconfig: đọc ngân sách: %w", err)
	}
	b.Version = int(row.Version)
	if row.DailyLimit.Valid {
		d := row.DailyLimit.Decimal
		b.Daily = &d
	}
	if row.MonthlyLimit.Valid {
		m := row.MonthlyLimit.Decimal
		b.Monthly = &m
	}
	return b, nil
}

// GetBudget đọc ngân sách (ADMIN, TEACHER). Ngân sách lớp chỉ ADMIN (ở tầng handler theo SRS 6.2) — tầng này chỉ kiểm vai đọc.
func (s *Service) GetBudget(ctx context.Context, scope string, courseID *uuid.UUID) (Budget, error) {
	if _, err := actorFrom(ctx, false); err != nil {
		return Budget{}, err
	}
	if err := checkScope(scope, courseID); err != nil {
		return Budget{}, err
	}
	return readBudget(ctx, s.q, scope, courseID)
}

func checkScope(scope string, courseID *uuid.UUID) error {
	switch {
	case scope == ScopeSystem && courseID == nil:
		return nil
	case scope == ScopeCourse && courseID != nil:
		return nil
	}
	return invalid("scope", "INVALID_SCOPE", "phạm vi ngân sách không hợp lệ")
}

// SetBudget đặt hạn mức (ADMIN). version: 0 khi chưa có dòng, ngược lại phải khớp.
func (s *Service) SetBudget(ctx context.Context, scope string, courseID *uuid.UUID, daily, monthly *decimal.Decimal, version int) (Budget, error) {
	a, err := actorFrom(ctx, true)
	if err != nil {
		return Budget{}, err
	}
	if err := checkScope(scope, courseID); err != nil {
		return Budget{}, err
	}
	for field, v := range map[string]*decimal.Decimal{"daily_limit": daily, "monthly_limit": monthly} {
		if v != nil && v.IsNegative() {
			return Budget{}, invalid(field, "OUT_OF_RANGE", "hạn mức không âm")
		}
	}
	if daily != nil && monthly != nil && daily.GreaterThan(*monthly) {
		return Budget{}, invalid("daily_limit", "OUT_OF_RANGE", "hạn mức ngày không được lớn hơn hạn mức tháng")
	}
	var out Budget
	err = s.inTx(ctx, func(q *store.Queries) error {
		cur, err := readBudget(ctx, q, scope, courseID)
		if err != nil {
			return err
		}
		if cur.Version != version {
			return &ErrVersionConflict{Current: cur.Version}
		}
		dl, ml := toNull(daily), toNull(monthly)
		if cur.Version == 0 {
			if _, err := q.InsertLLMBudget(ctx, store.InsertLLMBudgetParams{Scope: scope, CourseID: courseID, DailyLimit: dl, MonthlyLimit: ml}); err != nil {
				if isUnique(err, "") { // hai Admin tạo cùng lúc
					return &ErrVersionConflict{Current: 1}
				}
				return fmt.Errorf("llmconfig: chèn ngân sách: %w", err)
			}
		} else {
			row, err := q.GetLLMBudget(ctx, store.GetLLMBudgetParams{Scope: scope, CourseID: courseID})
			if err != nil {
				return fmt.Errorf("llmconfig: đọc ngân sách: %w", err)
			}
			if _, err := q.UpdateLLMBudget(ctx, store.UpdateLLMBudgetParams{ID: row.ID, Version: int32(version), DailyLimit: dl, MonthlyLimit: ml}); err != nil { //nolint:gosec // version nhỏ
				return fmt.Errorf("llmconfig: sửa ngân sách: %w", err)
			}
		}
		if out, err = readBudget(ctx, q, scope, courseID); err != nil {
			return err
		}
		entityID := scope
		if courseID != nil {
			entityID = courseID.String()
		}
		return s.writeAudit(ctx, q, a, "llm_budget", entityID, "set", budgetSnap(cur), budgetSnap(out))
	})
	if err != nil {
		return Budget{}, err
	}
	s.changed(ctx)
	return out, nil
}

func budgetSnap(b Budget) map[string]any {
	m := map[string]any{"version": b.Version}
	if b.Daily != nil {
		m["daily_limit"] = b.Daily.StringFixed(2)
	}
	if b.Monthly != nil {
		m["monthly_limit"] = b.Monthly.StringFixed(2)
	}
	return m
}

func toNull(d *decimal.Decimal) decimal.NullDecimal {
	if d == nil {
		return decimal.NullDecimal{}
	}
	return decimal.NullDecimal{Decimal: *d, Valid: true}
}

// UsageRow là một dòng GET usage; tiền là decimal.
type UsageRow struct {
	Key                        string
	Calls, TokensIn, TokensOut int64
	CostEst                    decimal.Decimal
	LatencyP50Ms, LatencyP95Ms int64
	Errors, Degraded           int64
}

// Usage tổng hợp llm_audit theo tác vụ ("task") hoặc ngày ("day"); courseID nil = cả hệ thống (ADMIN, TEACHER).
func (s *Service) Usage(ctx context.Context, group string, from, to time.Time, courseID *uuid.UUID) ([]UsageRow, error) {
	if _, err := actorFrom(ctx, false); err != nil {
		return nil, err
	}
	if !to.After(from) {
		return nil, invalid("to", "OUT_OF_RANGE", "khoảng thời gian không hợp lệ")
	}
	if to.Sub(from) > 92*24*time.Hour {
		return nil, invalid("from", "OUT_OF_RANGE", "khoảng thời gian tối đa 92 ngày")
	}
	switch group {
	case "task":
		rows, err := s.q.LLMUsageByTask(ctx, store.LLMUsageByTaskParams{FromTs: from, ToTs: to, CourseID: courseID})
		if err != nil {
			return nil, fmt.Errorf("llmconfig: mức dùng theo tác vụ: %w", err)
		}
		out := make([]UsageRow, 0, len(rows))
		for _, r := range rows {
			out = append(out, UsageRow(r))
		}
		return out, nil
	case "day":
		rows, err := s.q.LLMUsageByDay(ctx, store.LLMUsageByDayParams{FromTs: from, ToTs: to, CourseID: courseID})
		if err != nil {
			return nil, fmt.Errorf("llmconfig: mức dùng theo ngày: %w", err)
		}
		out := make([]UsageRow, 0, len(rows))
		for _, r := range rows {
			out = append(out, UsageRow(r))
		}
		return out, nil
	}
	return nil, invalid("group", "INVALID_GROUP", "group là task hoặc day")
}
