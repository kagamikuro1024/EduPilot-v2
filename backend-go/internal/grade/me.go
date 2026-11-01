package grade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Dạng JSON của điểm phía SINH VIÊN (SRS FEAT-gradebook 4.8.1). Đây là struct RIÊNG với allowlist khoá (`TestStudentDTOAllowlist`):
// không `hp_raw`, không `breakdown` nội bộ, không `unclear_items`, không tên người sửa, không `drift`, không `unlock_reason`, không điểm chưa công bố.
type (
	MeSchemeComponent struct {
		Key    string `json:"key"`
		Name   string `json:"name"`
		Grp    string `json:"grp"`
		Weight string `json:"weight"`
	}
	MeScheme struct {
		SchemeNo    int                 `json:"scheme_no"`
		ConfirmedAt *time.Time          `json:"confirmed_at"`
		Components  []MeSchemeComponent `json:"components"`
	}
	// MeComponent: `score` CHỈ có khi `status = PUBLISHED` (không có khoá `score` khi PENDING).
	MeComponent struct {
		Key    string `json:"key"`
		Name   string `json:"name"`
		Weight string `json:"weight"`
		Status string `json:"status"`
		Score  string `json:"score,omitempty"`
	}
	MeBonus struct {
		Applied string `json:"applied"`
		Raw     string `json:"raw"`
		Cap     string `json:"cap"`
	}
	MePenalty struct {
		Value  string `json:"value"`
		Absent int    `json:"absent"`
		Free   int    `json:"free"`
		Per    string `json:"per"`
	}
	MeFinal struct {
		NumericGrade string    `json:"numeric_grade"`
		Letter       string    `json:"letter"`
		FailedReason *string   `json:"failed_reason,omitempty"`
		Version      int       `json:"version"`
		LockedAt     time.Time `json:"locked_at"`
	}
	MeLine struct {
		Label       string `json:"label"`
		Expr        string `json:"expr"`
		Value       string `json:"value"`
		SourceQuote string `json:"source_quote,omitempty"`
		Page        int    `json:"page,omitempty"`
	}
	MeGrades struct {
		Status       string        `json:"status"`
		Scheme       *MeScheme     `json:"scheme,omitempty"`
		Components   []MeComponent `json:"components,omitempty"`
		Bonus        *MeBonus      `json:"bonus,omitempty"`
		Penalty      *MePenalty    `json:"penalty,omitempty"`
		QT           string        `json:"qt,omitempty"`
		Final        *MeFinal      `json:"final,omitempty"`
		Calculation  []MeLine      `json:"calculation,omitempty"`
		OfficialNote string        `json:"official_note,omitempty"`
	}
)

// Trạng thái điểm của sinh viên.
const (
	MeNoScheme    = "NO_SCHEME"
	MeInProgress  = "IN_PROGRESS"
	MeStatusFinal = "FINAL"
)

// publishedInput bỏ điểm của item chưa công bố: tính trên đầu vào đã lọc là cách duy nhất để không lộ điểm nháp (không có đường tính thứ hai).
func publishedInput(conf Loaded, in Input) Input {
	out := Input{Scores: map[uuid.UUID]decimal.Decimal{}, BonusPoints: in.BonusPoints, PenaltyEventPoints: in.PenaltyEventPoints, Absent: in.Absent}
	for _, c := range conf.Scheme.Components {
		for _, it := range c.Items {
			if v, ok := in.Scores[it.ID]; ok && itemPublished(it) {
				out.Scores[it.ID] = v
			}
		}
	}
	return out
}

func itemPublished(it Item) bool {
	switch it.Source {
	case SourceExam:
		return it.Closed
	case SourceManual:
		return it.Published
	}
	return false
}

func componentPublished(c Component) bool {
	if len(c.Items) == 0 {
		return false
	}
	for _, it := range c.Items {
		if !itemPublished(it) {
			return false
		}
	}
	return true
}

// lietText đổi mã lý do liệt (`LIET:final<3.0`) sang câu cho sinh viên.
func lietText(v string) string {
	v = strings.TrimPrefix(v, "LIET:")
	switch {
	case strings.HasPrefix(v, "final<"):
		return "Điểm thi cuối kỳ dưới " + strings.Replace(strings.TrimPrefix(v, "final<"), ".", ",", 1)
	case strings.HasPrefix(v, "absent>="):
		return "Vắng từ " + strings.TrimPrefix(v, "absent>=") + " buổi trở lên"
	}
	return "Không đạt điều kiện của học phần"
}

func meScheme(conf Loaded) *MeScheme {
	m := &MeScheme{SchemeNo: conf.No, ConfirmedAt: conf.ConfirmedAt}
	for _, c := range conf.Scheme.Components {
		m.Components = append(m.Components, MeSchemeComponent{Key: c.Key, Name: c.Name, Grp: string(c.Group), Weight: c.Weight.String()})
	}
	return m
}

func ruleParams(conf Loaded) (capV string, free int, per string) {
	if b := conf.Scheme.Rules.Bonus; b != nil && b.Applicable {
		capV = b.Cap.String()
	}
	if p := conf.Scheme.Rules.Penalty; p != nil && p.Applicable {
		free, per = p.Free, p.Per.String()
	}
	return capV, free, per
}

// MeGrades trả điểm của CHÍNH sinh viên `user` (SRS 4.8.1). `NO_SCHEME` chỉ có `status`; `FINAL` lấy từ bản chốt mới nhất của sinh viên; còn lại `IN_PROGRESS`
// với chỉ những gì đã công bố. `final` chỉ sau chốt; `qt` chỉ khi mọi thành phần quá trình đã công bố.
func (s *Service) MeGrades(ctx context.Context, courseID, user uuid.UUID) (MeGrades, error) {
	q := store.New(s.Pool)
	conf, ok, err := LoadConfirmed(ctx, q, courseID)
	if err != nil {
		return MeGrades{}, err
	}
	if !ok {
		return MeGrades{Status: MeNoScheme}, nil
	}
	out := MeGrades{Status: MeInProgress, Scheme: meScheme(conf), OfficialNote: OfficialNote}
	capV, free, per := ruleParams(conf)

	if gb, err := q.GradebookState(ctx, courseID); err == nil && gb.Status == store.GradebookStatusLOCKED {
		sn, err := q.GradeSnapshotsForStudents(ctx, store.GradeSnapshotsForStudentsParams{CourseID: courseID, Version: gb.SnapshotVersion, StudentIds: idStrings([]uuid.UUID{user})})
		if err != nil {
			return MeGrades{}, fmt.Errorf("grade: bản chốt của sinh viên: %w", err)
		}
		if len(sn) == 1 {
			return meFinal(out, sn[0], gb.SnapshotVersion, capV, free, per)
		}
	}

	inputs, err := LoadInputs(ctx, q, courseID, conf.ID, []uuid.UUID{user})
	if err != nil {
		return MeGrades{}, err
	}
	in := publishedInput(conf, inputs[user])
	rc := Compute(conf.Scheme, in)
	r := WhatIf(conf.Scheme, in, decimal.Zero) // điểm quá trình hiện tại: cuối kỳ giả định chỉ để engine không dừng ở thành phần cuối kỳ còn thiếu, không hiển thị
	byKey := map[string]ComponentResult{}
	for _, c := range rc.Components {
		byKey[c.Key] = c
	}
	qtPublished := true
	for _, c := range conf.Scheme.Components {
		mc := MeComponent{Key: c.Key, Name: c.Name, Weight: c.Weight.String(), Status: "PENDING"}
		if componentPublished(c) {
			mc.Status = "PUBLISHED"
			if cr, ok := byKey[c.Key]; ok && cr.Computed {
				mc.Score = cr.Score.String()
			}
		}
		if c.Group == GroupQT && mc.Status != "PUBLISHED" {
			qtPublished = false
		}
		out.Components = append(out.Components, mc)
	}
	out.Bonus = &MeBonus{Applied: r.BonusApplied.String(), Raw: r.Bonus.String(), Cap: capV}
	out.Penalty = &MePenalty{Value: r.Penalty.String(), Absent: in.Absent, Free: free, Per: per}
	showQT := r.HasQT && qtPublished
	if showQT {
		out.QT = r.QT.String()
	}
	for _, l := range r.Lines {
		if l.Expr == "" {
			continue // dòng "thiếu điểm" nội bộ
		}
		if !showQT && !strings.HasSuffix(l.Label, "%)") {
			continue
		}
		out.Calculation = append(out.Calculation, meLine(l))
		if showQT && l.Label == "Điểm quá trình" {
			break // `final` chỉ sau chốt: không có dòng cuối kỳ / điểm học phần trước đó
		}
	}
	return out, nil
}

func meLine(l Line) MeLine {
	m := MeLine{Label: l.Label, Expr: l.Expr, Value: l.Value, Page: l.Page}
	if l.Quote != nil {
		m.SourceQuote = *l.Quote
	}
	return m
}

func meFinal(out MeGrades, sn store.GradeSnapshotsForStudentsRow, version int32, capV string, free int, per string) (MeGrades, error) {
	var bd struct {
		SnapBreakdown
		Absent     int `json:"absent"`
		Components []struct {
			Key, Name, Weight, Score string
		} `json:"components"`
		Lines []struct {
			Label       string `json:"label"`
			Expr        string `json:"expr"`
			Value       string `json:"value"`
			SourceQuote string `json:"source_quote"`
			Page        int    `json:"page"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(sn.Breakdown, &bd); err != nil {
		return MeGrades{}, fmt.Errorf("grade: đọc bản chốt: %w", err)
	}
	out.Status = MeStatusFinal
	for _, c := range bd.Components {
		out.Components = append(out.Components, MeComponent{Key: c.Key, Name: c.Name, Weight: c.Weight, Status: "PUBLISHED", Score: c.Score})
	}
	out.Bonus = &MeBonus{Applied: bd.Bonus, Raw: bd.Bonus, Cap: capV}
	out.Penalty = &MePenalty{Value: bd.Penalty, Absent: bd.Absent, Free: free, Per: per}
	out.QT = bd.QT
	f := &MeFinal{NumericGrade: sn.NumericGrade.String(), Letter: sn.Letter, Version: int(version)}
	if sn.FailedReason != nil {
		t := lietText(*sn.FailedReason)
		f.FailedReason = &t
	}
	for _, l := range bd.Lines {
		out.Calculation = append(out.Calculation, MeLine{Label: l.Label, Expr: l.Expr, Value: l.Value, SourceQuote: l.SourceQuote, Page: l.Page})
	}
	out.Final = f
	return out, nil
}

// WhatIfInput là thân `POST …/me/grades/what-if`: đúng MỘT trong hai trường.
type WhatIfInput struct {
	FinalExamScore *string `json:"final_exam_score"`
	TargetLetter   *string `json:"target_letter"`
}

// WhatIfOut: `hp` / `letter` / `failed` / `calculation` cho điểm cuối kỳ giả định, hoặc `min_final_exam_score` (null = không đạt được) cho điểm chữ mục tiêu.
type WhatIfOut struct {
	HP           string   `json:"hp,omitempty"`
	Letter       string   `json:"letter,omitempty"`
	Failed       *string  `json:"failed,omitempty"`
	Calculation  []MeLine `json:"calculation,omitempty"`
	MinFinalExam *string  `json:"min_final_exam_score,omitempty"`
	Reachable    *bool    `json:"reachable,omitempty"`
}

// MeWhatIf tính "nếu cuối kỳ được X" bằng `WhatIf` / `MinFinalFor` của engine (SRS 4.8.2): chỉ đọc, chỉ khi `IN_PROGRESS` và mọi thành phần quá trình đã công bố.
func (s *Service) MeWhatIf(ctx context.Context, courseID, user uuid.UUID, in WhatIfInput) (WhatIfOut, error) {
	if (in.FinalExamScore == nil) == (in.TargetLetter == nil) {
		return WhatIfOut{}, apierr.Validation(apierr.FieldError{Field: "final_exam_score", Code: "one_of", Message: "Nhập điểm cuối kỳ hoặc điểm chữ mong muốn."})
	}
	var ck decimal.Decimal
	if in.FinalExamScore != nil {
		d, err := ParseScore(*in.FinalExamScore)
		if err != nil {
			return WhatIfOut{}, err
		}
		if d.GreaterThan(MaxScale) {
			return WhatIfOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.ScoreOutOfRange)
		}
		ck = d
	}
	q := store.New(s.Pool)
	conf, ok, err := LoadConfirmed(ctx, q, courseID)
	if err != nil {
		return WhatIfOut{}, err
	}
	if !ok {
		return WhatIfOut{}, apierr.New(http.StatusConflict, apierr.SchemeNotConfirmed)
	}
	if gb, err := q.GradebookState(ctx, courseID); err == nil && gb.Status == store.GradebookStatusLOCKED {
		if sn, err := q.GradeSnapshotsForStudents(ctx, store.GradeSnapshotsForStudentsParams{CourseID: courseID, Version: gb.SnapshotVersion, StudentIds: idStrings([]uuid.UUID{user})}); err == nil && len(sn) == 1 {
			return WhatIfOut{}, apierr.New(http.StatusConflict, apierr.FinalAlreadyLocked)
		}
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return WhatIfOut{}, fmt.Errorf("grade: trạng thái sổ điểm: %w", err)
	}
	inputs, err := LoadInputs(ctx, q, courseID, conf.ID, []uuid.UUID{user})
	if err != nil {
		return WhatIfOut{}, err
	}
	pin := publishedInput(conf, inputs[user])
	qtOK := WhatIf(conf.Scheme, pin, decimal.Zero).HasQT
	for _, c := range conf.Scheme.Components {
		if c.Group == GroupQT && !componentPublished(c) {
			qtOK = false
		}
	}
	if !qtOK {
		return WhatIfOut{}, apierr.New(http.StatusConflict, apierr.NotEnoughGrades)
	}
	if in.TargetLetter != nil {
		v, reach := MinFinalFor(conf.Scheme, pin, strings.TrimSpace(*in.TargetLetter))
		out := WhatIfOut{Reachable: &reach}
		if reach {
			m := v.String()
			out.MinFinalExam = &m
		}
		return out, nil
	}
	r := WhatIf(conf.Scheme, pin, ck)
	if r.Status != StatusComplete {
		return WhatIfOut{}, apierr.New(http.StatusConflict, apierr.NotEnoughGrades)
	}
	out := WhatIfOut{HP: r.HP.String(), Letter: r.Letter}
	if strings.HasPrefix(r.Verdict, "LIET") {
		t := lietText(r.Verdict)
		out.Failed = &t
	}
	for _, l := range r.Lines {
		if l.Expr != "" {
			out.Calculation = append(out.Calculation, meLine(l))
		}
	}
	return out, nil
}
