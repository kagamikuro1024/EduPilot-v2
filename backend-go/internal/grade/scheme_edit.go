package grade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Đầu vào PUT / impact (SRS 4.4.2–4.4.3). Số là chuỗi decimal.
type (
	PutItem struct {
		Key      string     `json:"key"`
		Name     string     `json:"name"`
		MaxScore string     `json:"max_score"`
		Source   string     `json:"source"`
		ExamID   *uuid.UUID `json:"exam_id"`
	}
	PutComponent struct {
		Key    string    `json:"key"`
		Name   string    `json:"name"`
		Grp    string    `json:"grp"`
		Weight string    `json:"weight"`
		Items  []PutItem `json:"items"`
	}
	PutRule struct {
		Kind       string          `json:"kind"`
		Applicable *bool           `json:"applicable"`
		Params     json.RawMessage `json:"params"`
	}
	PutInput struct {
		Version    int            `json:"version"`
		Components []PutComponent `json:"components"`
		Rules      []PutRule      `json:"rules"`
	}
)

type (
	parsedItem struct {
		key, name string
		max       decimal.Decimal
		source    ItemSource
		exam      *uuid.UUID
	}
	parsedComp struct {
		key, name string
		grp       Group
		weight    decimal.Decimal
		items     []parsedItem
	}
	parsedRule struct {
		kind       RuleKind
		applicable bool
		params     json.RawMessage
	}
	parsed struct {
		comps []parsedComp
		rules []parsedRule
	}
)

var keyRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

const (
	maxComponents = 20
	maxItems      = 20
	maxName       = 80
)

var ruleKinds = map[RuleKind]bool{KindBonus: true, KindPenalty: true, KindRounding: true, KindFail: true, KindLetterScale: true, KindMissingScore: true} //nolint:gochecknoglobals // bảng tra bất biến

// decimal2 đọc số ≤ 2 chữ số thập phân (`numeric(5,2)`); chấp nhận dấu phẩy là lỗi (API nhận dấu chấm).
func decimal2(s string) (decimal.Decimal, bool) {
	d, err := decimal.NewFromString(s)
	if err != nil || d.Exponent() < -2 && !d.Equal(d.Round(2)) {
		return decimal.Zero, false
	}
	return d, true
}

func invalid(fields map[string]string) error {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fe := make([]apierr.FieldError, len(keys))
	for i, k := range keys {
		fe[i] = apierr.FieldError{Field: k, Code: "invalid", Message: fields[k]}
	}
	return apierr.Validation(fe...)
}

// parseInput kiểm hình dạng đầu vào (422 `VALIDATION_FAILED` kèm lỗi theo ô). Chưa kiểm bài thi (cần DB) — xem checkExams.
func parseInput(comps []PutComponent, rules []PutRule) (parsed, error) {
	bad := map[string]string{}
	var p parsed
	if len(comps) > maxComponents {
		bad["components"] = "Tối đa 20 thành phần."
	}
	seenComp, seenItem := map[string]bool{}, map[string]bool{}
	for i, c := range comps {
		at := fmt.Sprintf("components[%d]", i)
		pc := parsedComp{key: c.Key, name: c.Name, grp: Group(c.Grp)}
		if !keyRE.MatchString(c.Key) || seenComp[c.Key] {
			bad[at+".key"] = "Khoá không hợp lệ hoặc trùng."
		}
		seenComp[c.Key] = true
		if n := utf8.RuneCountInString(c.Name); n < 1 || n > maxName {
			bad[at+".name"] = "Tên 1–80 ký tự."
		}
		if pc.grp != GroupQT && pc.grp != GroupCK {
			bad[at+".grp"] = "Nhóm phải là QT hoặc CK."
		}
		w, ok := decimal2(c.Weight)
		if !ok || w.Sign() < 0 || w.GreaterThan(decimal.NewFromInt(100)) {
			bad[at+".weight"] = "Trọng số là số từ 0 đến 100, tối đa 2 chữ số thập phân."
		}
		pc.weight = w
		items := c.Items
		if len(items) == 0 { // thành phần mới: một cột điểm nhập tay cùng tên, thang 10
			items = []PutItem{{Key: c.Key, Name: c.Name, MaxScore: "10", Source: string(SourceManual)}}
		}
		if len(items) > maxItems {
			bad[at+".items"] = "Tối đa 20 cột điểm."
		}
		for j, it := range items {
			ia := fmt.Sprintf("%s.items[%d]", at, j)
			pi := parsedItem{key: it.Key, name: it.Name, source: ItemSource(it.Source), exam: it.ExamID}
			if !keyRE.MatchString(it.Key) || seenItem[it.Key] {
				bad[ia+".key"] = "Khoá không hợp lệ hoặc trùng."
			}
			seenItem[it.Key] = true
			if n := utf8.RuneCountInString(it.Name); n < 1 || n > maxName {
				bad[ia+".name"] = "Tên 1–80 ký tự."
			}
			m, ok := decimal2(it.MaxScore)
			if !ok || !m.IsPositive() || m.GreaterThan(decimal.NewFromInt(100)) {
				bad[ia+".max_score"] = "Thang điểm là số lớn hơn 0 đến 100."
			}
			pi.max = m
			switch pi.source {
			case SourceManual:
				if it.ExamID != nil {
					bad[ia+".exam_id"] = "Cột nhập tay không gắn bài thi."
				}
			case SourceExam:
				if it.ExamID == nil {
					bad[ia+".exam_id"] = "Chọn bài thi."
				}
			default:
				bad[ia+".source"] = "Nguồn là MANUAL hoặc EXAM."
			}
			pc.items = append(pc.items, pi)
		}
		p.comps = append(p.comps, pc)
	}
	seenKind := map[RuleKind]bool{}
	for i, r := range rules {
		at := fmt.Sprintf("rules[%d]", i)
		k := RuleKind(r.Kind)
		if !ruleKinds[k] {
			bad[at+".kind"] = "Loại luật không hợp lệ."
			continue
		}
		if k != KindFail && seenKind[k] {
			bad[at+".kind"] = "Luật bị khai hai lần."
		}
		seenKind[k] = true
		params := r.Params
		if len(params) == 0 || string(params) == "null" {
			params = json.RawMessage(`{}`)
		}
		var obj map[string]any
		if err := json.Unmarshal(params, &obj); err != nil || obj == nil {
			bad[at+".params"] = "Tham số phải là đối tượng."
			continue
		}
		app := true
		if r.Applicable != nil {
			app = *r.Applicable
		}
		p.rules = append(p.rules, parsedRule{kind: k, applicable: app, params: params})
	}
	if len(bad) > 0 {
		return p, invalid(bad)
	}
	return p, nil
}

// checkExams: item EXAM gắn bài thi CỦA LỚP, không còn là bản nháp chưa lên lịch, một bài chỉ ở một item.
func checkExams(ctx context.Context, q *store.Queries, courseID uuid.UUID, p parsed) error {
	var ids []string
	seen := map[uuid.UUID]bool{}
	bad := map[string]string{}
	for i, c := range p.comps {
		for j, it := range c.items {
			if it.exam == nil {
				continue
			}
			at := fmt.Sprintf("components[%d].items[%d].exam_id", i, j)
			if seen[*it.exam] {
				bad[at] = "Một bài thi chỉ ở một cột điểm."
			}
			seen[*it.exam] = true
			ids = append(ids, it.exam.String())
		}
	}
	if len(ids) > 0 {
		rows, err := q.GradeExamsOfCourse(ctx, store.GradeExamsOfCourseParams{CourseID: courseID, ExamIds: ids})
		if err != nil {
			return fmt.Errorf("grade: bài thi của lớp: %w", err)
		}
		status := map[uuid.UUID]store.ExamStatus{}
		for _, r := range rows {
			status[r.ID] = r.Status
		}
		for i, c := range p.comps {
			for j, it := range c.items {
				if it.exam == nil {
					continue
				}
				at := fmt.Sprintf("components[%d].items[%d].exam_id", i, j)
				switch st, ok := status[*it.exam]; {
				case !ok:
					bad[at] = "Bài thi không thuộc lớp."
				case st == store.ExamStatusDRAFT:
					bad[at] = "Bài thi chưa lên lịch."
				}
			}
		}
	}
	if len(bad) > 0 {
		return invalid(bad)
	}
	return nil
}

// versionConflict dựng 409 kèm bản hiện tại để giao diện hoàn lại.
func versionConflict(current SchemeDoc) error {
	return apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current_version": current.Version, "current": current})
}

func sameItems(old []Item, nu []parsedItem) bool {
	if len(old) != len(nu) {
		return false
	}
	for i, o := range old {
		n := nu[i]
		if o.Key != n.key || o.Name != n.name || !o.Max.Equal(n.max) || o.Source != n.source || (o.ExamID == nil) != (n.exam == nil) || (o.ExamID != nil && *o.ExamID != *n.exam) {
			return false
		}
	}
	return true
}

// PutDraft thay TOÀN BỘ bản nháp (SRS 4.4.2): so version → ghi lại thành phần / item / luật → chạy bộ kiểm → ghi `unclear_count` (#16), cùng một giao dịch.
// Mục không đổi giá trị giữ trích dẫn và cờ cũ; mục đổi / mới → `filled_by_user = true`, bỏ trích dẫn.
func (s *Service) PutDraft(ctx context.Context, actor, courseID uuid.UUID, in PutInput) (SchemeDoc, error) {
	p, err := parseInput(in.Components, in.Rules)
	if err != nil {
		return SchemeDoc{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return SchemeDoc{}, fmt.Errorf("grade: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := store.New(tx)
	if err := checkExams(ctx, q, courseID, p); err != nil {
		return SchemeDoc{}, err
	}
	cur, err := q.GradeDraftLock(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, e := q.GradeSchemeIDByStatus(ctx, store.GradeSchemeIDByStatusParams{CourseID: courseID, Status: store.GradeSchemeStatusCONFIRMED}); e == nil {
			return SchemeDoc{}, apierr.New(http.StatusConflict, apierr.SchemeNotDraft)
		}
		return SchemeDoc{}, apierr.New(http.StatusConflict, apierr.NoDraft)
	}
	if err != nil {
		return SchemeDoc{}, fmt.Errorf("grade: khoá bản nháp: %w", err)
	}
	if int(cur.Version) != in.Version {
		doc, err := schemeDocByID(ctx, q, courseID, cur.ID)
		if err != nil {
			return SchemeDoc{}, err
		}
		return SchemeDoc{}, versionConflict(doc)
	}
	old, err := LoadScheme(ctx, q, courseID, cur.ID)
	if err != nil {
		return SchemeDoc{}, err
	}
	oldComp := map[string]Component{}
	for _, c := range old.Scheme.Components {
		oldComp[c.Key] = c
	}
	oldRule := map[RuleKind][]DraftRule{}
	for _, r := range old.Draft.Rules {
		oldRule[r.Kind] = append(oldRule[r.Kind], r)
	}
	if err := q.GradeComponentsDelete(ctx, store.GradeComponentsDeleteParams{CourseID: courseID, SchemeID: cur.ID}); err != nil {
		return SchemeDoc{}, fmt.Errorf("grade: xoá thành phần: %w", err)
	}
	if err := q.GradeRulesDelete(ctx, store.GradeRulesDeleteParams{CourseID: courseID, SchemeID: cur.ID}); err != nil {
		return SchemeDoc{}, fmt.Errorf("grade: xoá luật: %w", err)
	}
	pos := int32(0)
	for i, c := range p.comps {
		keep := false
		var cites []Citation
		filled := true
		if o, ok := oldComp[c.key]; ok && o.Name == c.name && o.Group == c.grp && o.Weight.Equal(c.weight) && sameItems(o.Items, c.items) {
			keep, cites, filled = true, o.Citations, o.FilledByUser
		}
		if !keep {
			cites = nil
		}
		craw, _ := json.Marshal(citesOrEmpty(cites))
		cid, err := q.GradeComponentInsert(ctx, store.GradeComponentInsertParams{CourseID: courseID, SchemeID: cur.ID, Key: c.key, Name: c.name, Grp: store.GradeGroup(c.grp), Weight: c.weight, Position: int32(i), Citations: craw, FilledByUser: filled}) //nolint:gosec // i ≤ 20
		if err != nil {
			return SchemeDoc{}, fmt.Errorf("grade: lưu thành phần: %w", err)
		}
		for _, it := range c.items {
			if _, err := q.GradeItemInsert(ctx, store.GradeItemInsertParams{CourseID: courseID, SchemeID: cur.ID, ComponentID: cid, Key: it.key, Name: it.name, MaxScore: it.max, Source: store.GradeItemSource(it.source), ExamID: it.exam, Position: pos}); err != nil {
				return SchemeDoc{}, fmt.Errorf("grade: lưu cột điểm: %w", err)
			}
			pos++
		}
	}
	for _, r := range p.rules {
		var quote *string
		var page *int32
		filled := true
		for _, o := range oldRule[r.kind] {
			if o.Applicable == r.applicable && canonJSON(o.Params) == canonJSON(r.params) {
				filled = o.FilledByUser
				if o.Quote != "" {
					q, pg := o.Quote, int32(o.Page) //nolint:gosec // số trang nhỏ
					quote, page = &q, &pg
				}
				break
			}
		}
		if err := q.GradeRuleInsert(ctx, store.GradeRuleInsertParams{CourseID: courseID, SchemeID: cur.ID, Kind: store.GradeRuleKind(r.kind), Applicable: r.applicable, Params: r.params, SourceQuote: quote, SourcePage: page, FilledByUser: filled}); err != nil {
			return SchemeDoc{}, fmt.Errorf("grade: lưu luật: %w", err)
		}
	}
	ver, err := q.GradeSchemeBump(ctx, store.GradeSchemeBumpParams{CourseID: courseID, ID: cur.ID, Version: cur.Version})
	if err != nil {
		return SchemeDoc{}, fmt.Errorf("grade: tăng version: %w", err)
	}
	doc, err := schemeDocByID(ctx, q, courseID, cur.ID)
	if err != nil {
		return SchemeDoc{}, err
	}
	if _, err := q.GradeSetUnclear(ctx, store.GradeSetUnclearParams{CourseID: courseID, ID: cur.ID, Version: ver, UnclearCount: int32(len(doc.UnclearItems))}); err != nil { //nolint:gosec // ≤ vài chục
		return SchemeDoc{}, fmt.Errorf("grade: ghi số mục chưa rõ: %w", err)
	}
	after, _ := json.Marshal(map[string]any{"version": ver, "unclear_codes": CodesOf(doc.UnclearItems)})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "grade_schemes", EntityID: cur.ID.String(), Action: "edit", After: after}); err != nil {
		return SchemeDoc{}, fmt.Errorf("grade: audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return SchemeDoc{}, fmt.Errorf("grade: lưu bản nháp: %w", err)
	}
	return doc, nil
}
