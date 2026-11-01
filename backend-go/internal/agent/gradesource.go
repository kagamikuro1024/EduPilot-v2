package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// GradeAdapter nối `grade.Service` vào GradeSource + GradeSchemeSource. Danh tính chỉ từ TrustedContext (luật 2).
type GradeAdapter struct{ Grade *grade.Service }

var (
	_ GradeSource       = GradeAdapter{}
	_ GradeSchemeSource = GradeAdapter{}
)

// Scheme trả công thức ĐÃ XÁC NHẬN của lớp dạng khối cấu trúc `{components[], bonus, penalty, rounding, letter_scale, source}` (số do code, LLM chỉ diễn đạt); chưa có
// (kể cả chỉ có bản nháp) → ok=false (bản nháp không bao giờ lộ cho sinh viên). `source` là "Quy chế học phần, tr. N" của thành phần đầu tiên có căn cứ.
func (a GradeAdapter) Scheme(ctx context.Context, tc TrustedContext) (Facts, bool, error) {
	l, ok, err := grade.LoadConfirmed(ctx, store.New(a.Grade.Pool), tc.CourseID)
	if err != nil || !ok {
		return nil, false, err
	}
	comps := make([]map[string]any, 0, len(l.Scheme.Components))
	page := 0
	for _, c := range l.Scheme.Components {
		comps = append(comps, map[string]any{"name": c.Name, "weight_percent": c.Weight.String(), "group": string(c.Group)})
		if page == 0 && len(c.Citations) > 0 {
			page = c.Citations[0].Page
		}
	}
	r := l.Scheme.Rules
	f := Facts{"components": comps, "scheme_no": l.No, "bonus": nil, "penalty": nil, "rounding": nil, "letter_scale": nil}
	if b := r.Bonus; b != nil && b.Applicable {
		f["bonus"] = map[string]any{"per_speak": b.PerSpeak.String(), "cap": b.Cap.String()}
		if page == 0 && b.Page > 0 {
			page = b.Page
		}
	}
	if p := r.Penalty; p != nil && p.Applicable {
		f["penalty"] = map[string]any{"free_absences": p.Free, "per_absence": p.Per.String()}
	}
	if r.Rounding != nil {
		f["rounding"] = map[string]any{"step": r.Rounding.Step.String(), "mode": "HALF_UP"}
	}
	if r.Letters != nil {
		bands := make([]map[string]any, 0, len(r.Letters.Bands))
		for _, b := range r.Letters.Bands {
			bands = append(bands, map[string]any{"min": b.Min.String(), "letter": b.Letter})
		}
		f["letter_scale"] = bands
	}
	if page > 0 {
		f["source"] = fmt.Sprintf("Quy chế học phần, tr. %d", page)
	}
	return f, true, nil
}

// factsOf đổi DTO sinh viên (đã lọc: không điểm nháp) thành Facts bằng JSON — chỉ khoá đã được allowlist ở `grade.MeGrades`.
func factsOf(v any) (Facts, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("agent: mã hoá dữ liệu điểm: %w", err)
	}
	var f Facts
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("agent: giải mã dữ liệu điểm: %w", err)
	}
	return f, nil
}

// Summary là `get_my_grade_summary`: đúng DTO của `GET /me/grades` (số trong câu trả lời bằng số ở màn hình); `user_id`, `course_id` chỉ từ TrustedContext.
func (a GradeAdapter) Summary(ctx context.Context, tc TrustedContext) (Facts, bool, error) {
	m, err := a.Grade.MeGrades(ctx, tc.CourseID, tc.UserID)
	if err != nil {
		return nil, false, fmt.Errorf("agent: điểm của sinh viên: %w", err)
	}
	if m.Status == grade.MeNoScheme {
		return nil, false, ErrNoScheme
	}
	f, err := factsOf(m)
	return f, err == nil, err
}

// WhatIf là `what_if_final_grade`: LLM chỉ đưa giả định `final`; điểm do `grade.MeWhatIf` (engine) tính. Chưa đủ điểm quá trình / không hợp lệ → "chưa có dữ liệu"
// (không bịa); đã chốt → trả điểm đã chốt thay cho tính thử.
func (a GradeAdapter) WhatIf(ctx context.Context, tc TrustedContext, assumed map[string]decimal.Decimal) (Facts, bool, error) {
	ck, ok := assumed["final"]
	if !ok {
		return a.needScheme(ctx, tc)
	}
	s := ck.String()
	return a.whatIf(ctx, tc, grade.WhatIfInput{FinalExamScore: &s})
}

// MinFinal là `what_if_final_grade {target_letter}`: điểm cuối kỳ tối thiểu để đạt điểm chữ mục tiêu.
func (a GradeAdapter) MinFinal(ctx context.Context, tc TrustedContext, letter string) (Facts, bool, error) {
	return a.whatIf(ctx, tc, grade.WhatIfInput{TargetLetter: &letter})
}

func (a GradeAdapter) whatIf(ctx context.Context, tc TrustedContext, in grade.WhatIfInput) (Facts, bool, error) {
	out, err := a.Grade.MeWhatIf(ctx, tc.CourseID, tc.UserID, in)
	var ae *apierr.Error
	if errors.As(err, &ae) {
		switch ae.Code {
		case apierr.SchemeNotConfirmed:
			return nil, false, ErrNoScheme
		case apierr.FinalAlreadyLocked:
			return a.Summary(ctx, tc)
		case apierr.NotEnoughGrades, apierr.ScoreOutOfRange, apierr.ScorePrecision, apierr.ValidationFailed:
			return nil, false, nil
		}
	}
	if err != nil {
		return nil, false, fmt.Errorf("agent: tính thử điểm: %w", err)
	}
	f, err := factsOf(out)
	return f, err == nil, err
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
