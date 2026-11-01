package grade_test

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
)

const (
	fixDir   = "../../../docs/specs/FEAT-gradebook/fixtures/"
	seedDir  = "../../../seed/"
	oneItem  = "weekly" // key của thành phần "Kiểm tra hằng tuần"
	midKey   = "midterm"
	finalKey = "final"
)

type extraction struct {
	Components []struct {
		Key      string   `json:"key"`
		Name     string   `json:"name"`
		Group    string   `json:"group"`
		Weight   string   `json:"weight"`
		KindHint string   `json:"kind_hint"`
		CitIDs   []string `json:"citation_ids"`
	} `json:"components"`
	Rules []struct {
		Kind       string          `json:"kind"`
		Applicable bool            `json:"applicable"`
		Params     json.RawMessage `json:"params"`
		CitIDs     []string        `json:"citation_ids"`
	} `json:"rules"`
	Citations []struct {
		ID    string `json:"id"`
		Page  int    `json:"page"`
		Quote string `json:"quote"`
	} `json:"citations"`
}

func loadExtraction(t *testing.T, name string) extraction {
	t.Helper()
	b, err := os.ReadFile(fixDir + name)
	require.NoError(t, err)
	var e extraction
	require.NoError(t, json.Unmarshal(b, &e))
	return e
}

// tweak cho phép đổi công thức theo ca (trọng số, thang item giữa kỳ, nguồn item tuần, luật).
type tweak struct {
	weights     map[string]string
	midMax      string
	weeklyExam  bool // item tuần là EXAM (mặc định); false = MANUAL
	weeklyClose bool // nguồn đã chốt (bài thi PUBLISHED / cột đã công bố)
	rules       func(rows []grade.RuleRow) []grade.RuleRow
}

func defaultTweak() tweak { return tweak{weeklyExam: true, weeklyClose: true} }

// buildScheme dựng Scheme từ bản trích mong đợi (fixtures/extraction-*.json) và trả id item theo key thành phần.
func buildScheme(t *testing.T, e extraction, tw tweak) (grade.Scheme, map[string]uuid.UUID) {
	t.Helper()
	cit := map[string]grade.Citation{}
	for _, c := range e.Citations {
		cit[c.ID] = grade.Citation{Page: c.Page, Quote: c.Quote}
	}
	s := grade.Scheme{ID: uuid.New(), No: 1, Version: 1}
	ids := map[string]uuid.UUID{}
	for _, c := range e.Components {
		w := c.Weight
		if v, ok := tw.weights[c.Key]; ok {
			w = v
		}
		comp := grade.Component{ID: uuid.New(), Key: c.Key, Name: c.Name, Group: grade.Group(c.Group), Weight: dec(w)}
		for _, id := range c.CitIDs[:1] {
			comp.Citations = append(comp.Citations, cit[id])
		}
		it := grade.Item{ID: uuid.New(), Key: c.Key, Name: c.Name, Max: grade.MaxScale, Source: grade.SourceManual, Closed: true, Published: true, ComponentID: comp.ID}
		if c.Key == midKey && tw.midMax != "" {
			it.Max = dec(tw.midMax)
		}
		if c.Key == oneItem {
			if tw.weeklyExam {
				it.Source = grade.SourceExam
			}
			it.Closed = tw.weeklyClose
			it.Published = tw.weeklyClose
		}
		ids[c.Key] = it.ID
		comp.Items = []grade.Item{it}
		s.Components = append(s.Components, comp)
	}
	var rows []grade.RuleRow
	for _, r := range e.Rules {
		row := grade.RuleRow{Kind: grade.RuleKind(r.Kind), Applicable: r.Applicable, Params: r.Params}
		if len(r.CitIDs) > 0 {
			row.SourceQuote, row.SourcePage = cit[r.CitIDs[0]].Quote, cit[r.CitIDs[0]].Page
		}
		rows = append(rows, row)
	}
	if tw.rules != nil {
		rows = tw.rules(rows)
	}
	rules, err := grade.BuildRules(rows)
	require.NoError(t, err)
	s.Rules = rules
	return s, ids
}

func readCSV(t *testing.T, path string) []map[string]string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	rows, err := csv.NewReader(f).ReadAll()
	require.NoError(t, err)
	var out []map[string]string
	for _, r := range rows[1:] {
		m := map[string]string{}
		for i, h := range rows[0] {
			m[h] = r[i]
		}
		out = append(out, m)
	}
	return out
}

func optDec(s string) (decimal.Decimal, bool) {
	if s == "" {
		return decimal.Zero, false
	}
	return dec(s), true
}

// inputOf dựng Input từ các cột CSV (giữa kỳ, tuần, cuối kỳ rỗng = thiếu điểm).
func inputOf(row map[string]string, ids map[string]uuid.UUID) grade.Input {
	in := grade.Input{Scores: map[uuid.UUID]decimal.Decimal{}, BonusPoints: dec(row["speak_bonus_points"]), PenaltyEventPoints: dec(orZero(row["penalty_event_points"]))}
	in.Absent = atoiMust(row["absent"])
	for col, key := range map[string]string{"midterm": midKey, "weekly_exam_t9": oneItem, "final_exam": finalKey} {
		if v, ok := optDec(row[col]); ok {
			in.Scores[ids[key]] = v
		}
	}
	return in
}

func orZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func atoiMust(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// ---- bộ đánh giá biểu thức của dòng giải trình: + − × ÷ ( ) min(a; b) max(a; b) round(x); số có dấu phẩy hoặc chấm ----

type evalr struct {
	s   []rune
	pos int
}

func evalExpr(t *testing.T, expr string) decimal.Decimal {
	t.Helper()
	e := &evalr{s: []rune(strings.NewReplacer("/", "÷", "-", "−").Replace(expr))}
	v := e.sum(t)
	e.skip()
	require.Equal(t, len(e.s), e.pos, "biểu thức còn dư: %q", expr)
	return v
}

func (e *evalr) skip() {
	for e.pos < len(e.s) && e.s[e.pos] == ' ' {
		e.pos++
	}
}

func (e *evalr) peek() rune {
	e.skip()
	if e.pos >= len(e.s) {
		return 0
	}
	return e.s[e.pos]
}

func (e *evalr) sum(t *testing.T) decimal.Decimal {
	v := e.prod(t)
	for {
		switch e.peek() {
		case '+':
			e.pos++
			v = v.Add(e.prod(t))
		case '−':
			e.pos++
			v = v.Sub(e.prod(t))
		default:
			return v
		}
	}
}

func (e *evalr) prod(t *testing.T) decimal.Decimal {
	v := e.atom(t)
	for {
		switch e.peek() {
		case '×', 'x':
			e.pos++
			v = v.Mul(e.atom(t))
		case '÷':
			e.pos++
			v = v.DivRound(e.atom(t), grade.DivPrecision)
		default:
			return v
		}
	}
}

func (e *evalr) atom(t *testing.T) decimal.Decimal {
	c := e.peek()
	switch {
	case c == '(':
		e.pos++
		v := e.sum(t)
		require.Equal(t, ')', e.peek())
		e.pos++
		return v
	case c >= 'a' && c <= 'z':
		start := e.pos
		for e.pos < len(e.s) && e.s[e.pos] >= 'a' && e.s[e.pos] <= 'z' {
			e.pos++
		}
		name := string(e.s[start:e.pos])
		require.Equal(t, '(', e.peek())
		e.pos++
		args := []decimal.Decimal{e.sum(t)}
		for e.peek() == ';' {
			e.pos++
			args = append(args, e.sum(t))
		}
		require.Equal(t, ')', e.peek())
		e.pos++
		switch name {
		case "min":
			return decimal.Min(args[0], args[1])
		case "max":
			return decimal.Max(args[0], args[1])
		case "round":
			return args[0].Round(0)
		}
		require.Failf(t, "hàm lạ", name)
	}
	start := e.pos
	for e.pos < len(e.s) && (e.s[e.pos] >= '0' && e.s[e.pos] <= '9' || e.s[e.pos] == ',' || e.s[e.pos] == '.') {
		e.pos++
	}
	require.NotEqual(t, start, e.pos, "mong số tại %d trong %q", start, string(e.s))
	return dec(strings.Replace(string(e.s[start:e.pos]), ",", ".", 1))
}
