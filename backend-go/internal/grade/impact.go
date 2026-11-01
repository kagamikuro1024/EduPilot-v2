package grade

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// Kết quả `POST …/grade-scheme/impact` (SRS 4.4.3).
type (
	Dropped struct {
		ItemKey string `json:"item_key"`
		Count   int    `json:"count"`
		Reason  string `json:"reason"`
	}
	GradeValue struct {
		HP     string `json:"hp"`
		Letter string `json:"letter"`
	}
	SampleStudent struct {
		ID       uuid.UUID `json:"id"`
		FullName string    `json:"full_name"`
	}
	Sample struct {
		Student SampleStudent `json:"student"`
		Before  *GradeValue   `json:"before"`
		After   GradeValue    `json:"after"`
	}
	Impact struct {
		StudentsTotal      int       `json:"students_total"`
		StudentsComputable int       `json:"students_computable"`
		ChangedVsConfirmed int       `json:"changed_vs_confirmed"`
		EntriesCarried     int       `json:"entries_carried"`
		EntriesDropped     []Dropped `json:"entries_dropped"`
		Samples            []Sample  `json:"samples"`
	}
)

// Lý do không mang điểm nhập tay sang công thức mới (TLR-8).
const (
	DropItemRemoved   = "ITEM_REMOVED"
	DropMaxChanged    = "MAX_CHANGED"
	DropSourceChanged = "SOURCE_CHANGED"
)

type carryPair struct{ old, nu Item }

func parsedFromLoaded(l Loaded) parsed {
	var p parsed
	for _, c := range l.Scheme.Components {
		pc := parsedComp{key: c.Key, name: c.Name, grp: c.Group, weight: c.Weight}
		for _, it := range c.Items {
			pc.items = append(pc.items, parsedItem{key: it.Key, name: it.Name, max: it.Max, source: it.Source, exam: it.ExamID, assignment: it.AssignmentID})
		}
		p.comps = append(p.comps, pc)
	}
	for _, r := range l.Draft.Rules {
		p.rules = append(p.rules, parsedRule{kind: r.Kind, applicable: r.Applicable, params: r.Params})
	}
	return p
}

// Impact chạy `Compute` cho mọi sinh viên ACTIVE theo công thức đang xét (chưa lưu) và so với công thức đã xác nhận. Chỉ đọc.
func (s *Service) Impact(ctx context.Context, courseID uuid.UUID, comps []PutComponent, rules []PutRule) (Impact, error) {
	p, err := parseInput(comps, rules)
	if err != nil {
		return Impact{}, err
	}
	q := store.New(s.Pool)
	if err := checkExams(ctx, q, courseID, p); err != nil {
		return Impact{}, err
	}
	if err := checkAssignments(ctx, q, courseID, p); err != nil {
		return Impact{}, err
	}
	return impactFor(ctx, q, courseID, p)
}

func impactFor(ctx context.Context, q *store.Queries, courseID uuid.UUID, p parsed) (Impact, error) {
	out := Impact{EntriesDropped: []Dropped{}, Samples: []Sample{}}
	students, err := ActiveStudents(ctx, q, courseID)
	if err != nil {
		return out, err
	}
	out.StudentsTotal = len(students)
	ids := make([]uuid.UUID, len(students))
	for i, st := range students {
		ids[i] = st.ID
	}
	conf, hasConf, err := LoadConfirmed(ctx, q, courseID)
	if err != nil {
		return out, err
	}
	confID := uuid.Nil
	oldManual := map[string]Item{}
	var oldItems []string
	if hasConf {
		confID = conf.ID
		for _, c := range conf.Scheme.Components {
			for _, it := range c.Items {
				if it.Source == SourceManual {
					oldManual[it.Key] = it
					oldItems = append(oldItems, it.ID.String())
				}
			}
		}
	}
	base, err := LoadInputs(ctx, q, courseID, confID, ids)
	if err != nil {
		return out, err
	}
	counts := map[uuid.UUID]int{}
	if len(oldItems) > 0 {
		rows, err := q.GradeEntryCountsByItem(ctx, store.GradeEntryCountsByItemParams{CourseID: courseID, ItemIds: oldItems})
		if err != nil {
			return out, fmt.Errorf("grade: số điểm đã nhập: %w", err)
		}
		for _, r := range rows {
			counts[r.ItemID] = int(r.N)
		}
	}

	// Công thức mới dạng engine (ID tạm) + cặp mang điểm.
	nu := Scheme{ID: uuid.New()}
	var pairs []carryPair
	newByKey := map[string]parsedItem{}
	examItems := map[uuid.UUID]Item{} // bài thi → item mới
	var examIDs []string
	examPublished := map[uuid.UUID]bool{}
	for _, c := range p.comps {
		for _, it := range c.items {
			newByKey[it.key] = it
			if it.exam != nil {
				examIDs = append(examIDs, it.exam.String())
			}
		}
	}
	if len(examIDs) > 0 {
		rows, err := q.GradeExamsOfCourse(ctx, store.GradeExamsOfCourseParams{CourseID: courseID, ExamIds: examIDs})
		if err != nil {
			return out, fmt.Errorf("grade: bài thi của lớp: %w", err)
		}
		for _, r := range rows {
			examPublished[r.ID] = r.Status == store.ExamStatusPUBLISHED
		}
	}
	for i, c := range p.comps {
		comp := Component{ID: uuid.New(), Key: c.key, Name: c.name, Group: c.grp, Weight: c.weight}
		for j, it := range c.items {
			item := Item{ID: uuid.New(), Key: it.key, Name: it.name, Max: it.max, Source: it.source, ExamID: it.exam, AssignmentID: it.assignment, ComponentID: comp.ID, Position: i*maxItems + j}
			switch it.source {
			case SourceExam:
				item.Closed = examPublished[*it.exam]
				examItems[*it.exam] = item
			case SourceManual:
				if o, ok := oldManual[it.key]; ok && o.Max.Equal(it.max) {
					item.Published, item.Closed = o.Published, o.Published
					pairs = append(pairs, carryPair{old: o, nu: item})
				}
			}
			comp.Items = append(comp.Items, item)
		}
		nu.Components = append(nu.Components, comp)
	}
	carried := map[uuid.UUID]bool{}
	for _, pr := range pairs {
		carried[pr.old.ID] = true
		out.EntriesCarried += counts[pr.old.ID]
	}
	keys := make([]string, 0, len(oldManual))
	for k := range oldManual {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		o := oldManual[k]
		if carried[o.ID] || counts[o.ID] == 0 {
			continue
		}
		reason := DropItemRemoved
		if n, ok := newByKey[k]; ok {
			reason = DropMaxChanged
			if n.source != SourceManual {
				reason = DropSourceChanged
			}
		}
		out.EntriesDropped = append(out.EntriesDropped, Dropped{ItemKey: k, Count: counts[o.ID], Reason: reason})
	}

	rr := make([]RuleRow, len(p.rules))
	for i, r := range p.rules {
		rr[i] = RuleRow{Kind: r.kind, Applicable: r.applicable, Params: r.params}
	}
	rules, rerr := BuildRules(rr)
	if rerr != nil {
		return out, nil // luật chưa dùng được (tham số hỏng / thiếu): chưa tính thử được
	}
	nu.Rules = rules

	exam := map[uuid.UUID]map[uuid.UUID]decimal.Decimal{} // student → item mới → điểm quy thang
	if len(examIDs) > 0 {
		rows, err := q.GradeInputExamsByExam(ctx, store.GradeInputExamsByExamParams{CourseID: courseID, ExamIds: examIDs, StudentIds: idStrings(ids)})
		if err != nil {
			return out, fmt.Errorf("grade: điểm bài thi: %w", err)
		}
		for _, r := range rows {
			item, ok := examItems[r.ExamID]
			if !ok || r.ExamMax.Sign() <= 0 {
				continue
			}
			if exam[r.StudentID] == nil {
				exam[r.StudentID] = map[uuid.UUID]decimal.Decimal{}
			}
			exam[r.StudentID][item.ID] = r.Score.Mul(item.Max).DivRound(r.ExamMax, DivPrecision)
		}
	}

	type scored struct {
		st     Student
		before *Result
		after  Result
	}
	var done []scored
	for _, st := range students {
		b := base[st.ID]
		in := Input{Scores: map[uuid.UUID]decimal.Decimal{}, BonusPoints: b.BonusPoints, PenaltyEventPoints: b.PenaltyEventPoints, Absent: b.Absent}
		for _, pr := range pairs {
			if sc, ok := b.Scores[pr.old.ID]; ok {
				in.Scores[pr.nu.ID] = sc
			}
		}
		for id, sc := range exam[st.ID] {
			in.Scores[id] = sc
		}
		after := Compute(nu, in)
		if after.Status != StatusComplete {
			continue
		}
		sc := scored{st: st, after: after}
		if hasConf {
			bf := Compute(conf.Scheme, b)
			if bf.Status == StatusComplete {
				sc.before = &bf
				if !bf.HP.Equal(after.HP) || bf.Letter != after.Letter {
					out.ChangedVsConfirmed++
				}
			}
		}
		done = append(done, sc)
	}
	out.StudentsComputable = len(done)
	if len(done) == 0 {
		return out, nil
	}
	asc := append([]scored(nil), done...)
	sort.Slice(asc, func(i, j int) bool {
		if c := asc[i].after.HP.Cmp(asc[j].after.HP); c != 0 {
			return c < 0
		}
		return asc[i].st.ID.String() < asc[j].st.ID.String()
	})
	hi := asc[len(asc)-1]
	for _, c := range asc { // cao nhất; hoà → student_id nhỏ nhất
		if c.after.HP.Equal(hi.after.HP) {
			hi = c
			break
		}
	}
	picks := []scored{hi, asc[(len(asc)-1)/2], asc[0]}
	seen := map[uuid.UUID]bool{}
	for _, c := range picks {
		if seen[c.st.ID] {
			continue
		}
		seen[c.st.ID] = true
		sm := Sample{Student: SampleStudent{ID: c.st.ID, FullName: c.st.FullName}, After: GradeValue{HP: c.after.HP.String(), Letter: c.after.Letter}}
		if c.before != nil {
			sm.Before = &GradeValue{HP: c.before.HP.String(), Letter: c.before.Letter}
		}
		out.Samples = append(out.Samples, sm)
	}
	return out, nil
}
