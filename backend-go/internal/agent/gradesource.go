package agent

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/store"
)

// GradeAdapter nối `grade.Service` vào GradeSource + GradeSchemeSource. Danh tính chỉ từ TrustedContext (luật 2).
type GradeAdapter struct{ Grade *grade.Service }

var (
	_ GradeSource       = GradeAdapter{}
	_ GradeSchemeSource = GradeAdapter{}
)

// Scheme trả công thức ĐÃ XÁC NHẬN của lớp; chưa có (kể cả chỉ có bản nháp) → ok=false (bản nháp không bao giờ lộ cho sinh viên).
func (a GradeAdapter) Scheme(ctx context.Context, tc TrustedContext) (Facts, bool, error) {
	l, ok, err := grade.LoadConfirmed(ctx, store.New(a.Grade.Pool), tc.CourseID)
	if err != nil || !ok {
		return nil, false, err
	}
	comps := make([]map[string]any, 0, len(l.Scheme.Components))
	for _, c := range l.Scheme.Components {
		comps = append(comps, map[string]any{"name": c.Name, "weight_percent": c.Weight.String(), "group": string(c.Group)})
	}
	return Facts{"components": comps, "rules": l.Scheme.Rules, "scheme_no": l.No}, true, nil
}

// Summary / WhatIf: chưa có công thức xác nhận → ErrNoScheme. Có công thức thì điểm của sinh viên đến ở US-P6-09 (cần bảng điểm đã chốt); tới lúc đó trả "chưa có dữ liệu".
func (a GradeAdapter) Summary(ctx context.Context, tc TrustedContext) (Facts, bool, error) {
	return a.needScheme(ctx, tc)
}

func (a GradeAdapter) WhatIf(ctx context.Context, tc TrustedContext, _ map[string]decimal.Decimal) (Facts, bool, error) {
	return a.needScheme(ctx, tc)
}

func (a GradeAdapter) needScheme(ctx context.Context, tc TrustedContext) (Facts, bool, error) {
	_, ok, err := grade.LoadConfirmed(ctx, store.New(a.Grade.Pool), tc.CourseID)
	if err != nil {
		return nil, false, fmt.Errorf("agent: công thức điểm: %w", err)
	}
	if !ok {
		return nil, false, ErrNoScheme
	}
	return nil, false, nil
}
