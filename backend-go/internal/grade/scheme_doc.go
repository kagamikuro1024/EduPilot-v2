package grade

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// Dạng JSON của công thức ở `GET/PUT …/grade-scheme` (SRS 4.4.1). Số là chuỗi decimal (không float).
type (
	ItemDoc struct {
		Key          string     `json:"key"`
		Name         string     `json:"name"`
		MaxScore     string     `json:"max_score"`
		Source       string     `json:"source"`
		ExamID       *uuid.UUID `json:"exam_id,omitempty"`
		AssignmentID *uuid.UUID `json:"assignment_id,omitempty"`
	}
	ComponentDoc struct {
		Key          string     `json:"key"`
		Name         string     `json:"name"`
		Grp          string     `json:"grp"`
		Weight       string     `json:"weight"`
		Items        []ItemDoc  `json:"items"`
		Citations    []Citation `json:"citations"`
		FilledByUser bool       `json:"filled_by_user"`
	}
	RuleDoc struct {
		Kind         string          `json:"kind"`
		Applicable   bool            `json:"applicable"`
		Params       json.RawMessage `json:"params"`
		Citations    []Citation      `json:"citations"`
		FilledByUser bool            `json:"filled_by_user"`
	}
	SchemeDoc struct {
		ID               uuid.UUID      `json:"id"`
		SchemeNo         int            `json:"scheme_no"`
		Status           string         `json:"status"`
		Version          int            `json:"version"`
		SourceTitle      string         `json:"source_title,omitempty"`
		SourceDocumentID *uuid.UUID     `json:"source_document_id,omitempty"`
		ConfirmedAt      *time.Time     `json:"confirmed_at,omitempty"`
		Components       []ComponentDoc `json:"components"`
		Rules            []RuleDoc      `json:"rules"`
		UnclearItems     []Unclear      `json:"unclear_items"`
		Warnings         []string       `json:"warnings"`
	}
	// SchemesView là thân `GET …/grade-scheme`.
	SchemesView struct {
		Confirmed *SchemeDoc `json:"confirmed"`
		Draft     *SchemeDoc `json:"draft"`
		Diff      *Diff      `json:"diff,omitempty"`
	}
)

// Diff: khác biệt bản nháp so với bản đã xác nhận, do CODE tính (SRS 4.4.1).
type Diff struct {
	ComponentsAdded   []string       `json:"components_added"`
	ComponentsRemoved []string       `json:"components_removed"`
	WeightChanged     []WeightChange `json:"weight_changed"`
	RulesChanged      []RuleChange   `json:"rules_changed"`
	WeightSumFrom     string         `json:"weight_sum_from"`
	WeightSumTo       string         `json:"weight_sum_to"`
}

type WeightChange struct {
	Key  string `json:"key"`
	From string `json:"from"`
	To   string `json:"to"`
}

type RuleChange struct {
	Kind string          `json:"kind"`
	From json.RawMessage `json:"from"` // null = chưa có
	To   json.RawMessage `json:"to"`
}

func citesOrEmpty(c []Citation) []Citation {
	if c == nil {
		return []Citation{}
	}
	return c
}

// docOf dựng SchemeDoc từ một công thức đã đọc (kèm kết quả bộ kiểm nếu có).
func docOf(l Loaded, insp *Inspection) SchemeDoc {
	d := SchemeDoc{ID: l.ID, SchemeNo: l.No, Status: l.Status, Version: l.Version, SourceTitle: l.SourceTitle, SourceDocumentID: l.SourceDocID, ConfirmedAt: l.ConfirmedAt,
		Components: []ComponentDoc{}, Rules: []RuleDoc{}, UnclearItems: []Unclear{}, Warnings: []string{}}
	for _, c := range l.Scheme.Components {
		cd := ComponentDoc{Key: c.Key, Name: c.Name, Grp: string(c.Group), Weight: c.Weight.String(), Items: []ItemDoc{}, Citations: citesOrEmpty(c.Citations), FilledByUser: c.FilledByUser}
		for _, it := range c.Items {
			cd.Items = append(cd.Items, ItemDoc{Key: it.Key, Name: it.Name, MaxScore: it.Max.String(), Source: string(it.Source), ExamID: it.ExamID, AssignmentID: it.AssignmentID})
		}
		d.Components = append(d.Components, cd)
	}
	for _, r := range l.Draft.Rules {
		rd := RuleDoc{Kind: string(r.Kind), Applicable: r.Applicable, Params: r.Params, Citations: []Citation{}, FilledByUser: r.FilledByUser}
		if len(rd.Params) == 0 {
			rd.Params = json.RawMessage(`{}`)
		}
		if r.Quote != "" {
			rd.Citations = []Citation{{Page: r.Page, Quote: r.Quote}}
		}
		d.Rules = append(d.Rules, rd)
	}
	if insp != nil {
		if insp.Unclear != nil {
			d.UnclearItems = insp.Unclear
		}
		if insp.Warnings != nil {
			d.Warnings = insp.Warnings
		}
	}
	return d
}

// schemeDocByID đọc một công thức của lớp và chạy bộ kiểm (bản nháp: mục chưa rõ + cảnh báo; bản khác: cảnh báo).
func schemeDocByID(ctx context.Context, q *store.Queries, courseID, id uuid.UUID) (SchemeDoc, error) {
	l, err := LoadScheme(ctx, q, courseID, id)
	if err != nil {
		return SchemeDoc{}, err
	}
	var insp Inspection
	if l.Status == "DRAFT" {
		m, err := InspectDrafts(ctx, q, []draftRef{{ID: id, CourseID: courseID}})
		if err != nil {
			return SchemeDoc{}, err
		}
		insp = m[id]
	} else {
		_, insp.Warnings = Check(l.Draft, nil)
	}
	return docOf(l, &insp), nil
}

func optionalDoc(ctx context.Context, q *store.Queries, courseID uuid.UUID, status store.GradeSchemeStatus) (*SchemeDoc, error) {
	id, err := q.GradeSchemeIDByStatus(ctx, store.GradeSchemeIDByStatusParams{CourseID: courseID, Status: status})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("grade: công thức %s: %w", status, err)
	}
	d, err := schemeDocByID(ctx, q, courseID, id)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// Schemes trả `{confirmed, draft, diff?}` của lớp (SRS 4.4.1). `diff` chỉ khi đòi và có đủ hai bản.
func (s *Service) Schemes(ctx context.Context, courseID uuid.UUID, withDiff bool) (SchemesView, error) {
	q := store.New(s.Pool)
	var v SchemesView
	var err error
	if v.Confirmed, err = optionalDoc(ctx, q, courseID, store.GradeSchemeStatusCONFIRMED); err != nil {
		return v, err
	}
	if v.Draft, err = optionalDoc(ctx, q, courseID, store.GradeSchemeStatusDRAFT); err != nil {
		return v, err
	}
	if withDiff && v.Confirmed != nil && v.Draft != nil {
		d := DiffDocs(*v.Confirmed, *v.Draft)
		v.Diff = &d
	}
	return v, nil
}

func weightSum(cs []ComponentDoc) string {
	var sum decimal.Decimal
	for _, c := range cs {
		if w, err := decimal.NewFromString(c.Weight); err == nil {
			sum = sum.Add(w)
		}
	}
	return sum.String()
}

// canonJSON chuẩn hoá JSON (khoá sắp xếp) để so sánh giá trị tham số, không phụ thuộc thứ tự khoá / khoảng trắng.
func canonJSON(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(bytes.TrimSpace(raw))
	}
	out, _ := json.Marshal(v)
	return string(out)
}

func ruleValue(rs []RuleDoc) json.RawMessage {
	if len(rs) == 0 {
		return json.RawMessage("null")
	}
	vals := make([]map[string]any, len(rs))
	for i, r := range rs {
		var p any
		_ = json.Unmarshal(r.Params, &p)
		vals[i] = map[string]any{"applicable": r.Applicable, "params": p}
	}
	out, _ := json.Marshal(vals)
	if len(vals) == 1 {
		out, _ = json.Marshal(vals[0])
	}
	return out
}

// DiffDocs so hai công thức: thành phần thêm / bớt / đổi trọng số, luật đổi tham số (cũ → mới), tổng trọng số. Thuần, không I/O.
func DiffDocs(old, cur SchemeDoc) Diff {
	d := Diff{ComponentsAdded: []string{}, ComponentsRemoved: []string{}, WeightChanged: []WeightChange{}, RulesChanged: []RuleChange{}, WeightSumFrom: weightSum(old.Components), WeightSumTo: weightSum(cur.Components)}
	oldBy, curBy := map[string]ComponentDoc{}, map[string]ComponentDoc{}
	for _, c := range old.Components {
		oldBy[c.Key] = c
	}
	for _, c := range cur.Components {
		curBy[c.Key] = c
		o, ok := oldBy[c.Key]
		switch {
		case !ok:
			d.ComponentsAdded = append(d.ComponentsAdded, c.Key)
		case !weightEqual(o.Weight, c.Weight):
			d.WeightChanged = append(d.WeightChanged, WeightChange{Key: c.Key, From: o.Weight, To: c.Weight})
		}
	}
	for _, c := range old.Components {
		if _, ok := curBy[c.Key]; !ok {
			d.ComponentsRemoved = append(d.ComponentsRemoved, c.Key)
		}
	}
	group := func(rs []RuleDoc) map[string][]RuleDoc {
		m := map[string][]RuleDoc{}
		for _, r := range rs {
			m[r.Kind] = append(m[r.Kind], r)
		}
		for _, v := range m {
			sort.Slice(v, func(i, j int) bool { return canonJSON(v[i].Params) < canonJSON(v[j].Params) })
		}
		return m
	}
	og, cg := group(old.Rules), group(cur.Rules)
	kinds := map[string]bool{}
	for k := range og {
		kinds[k] = true
	}
	for k := range cg {
		kinds[k] = true
	}
	sorted := make([]string, 0, len(kinds))
	for k := range kinds {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		from, to := ruleValue(og[k]), ruleValue(cg[k])
		if canonJSON(from) != canonJSON(to) {
			d.RulesChanged = append(d.RulesChanged, RuleChange{Kind: k, From: from, To: to})
		}
	}
	return d
}

func weightEqual(a, b string) bool {
	x, e1 := decimal.NewFromString(a)
	y, e2 := decimal.NewFromString(b)
	return e1 == nil && e2 == nil && x.Equal(y)
}
