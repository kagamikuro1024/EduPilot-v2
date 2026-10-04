package today

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func item(id string, tier int) Item {
	return Item{ID: id, Kind: KindJoinRequest, Tier: tier, Course: &CourseRef{ClassCode: "C1"}}
}

func idsOf(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}

// Bảng bậc đúng như SRS 4.7 (mỗi vai một dãy tăng dần).
func TestRankTierOrder(t *testing.T) {
	t.Parallel()
	for name, tiers := range map[string][]int{
		"giảng viên": {TierEmailMismatch, TierJoinRequest, TierCourseSetup},
		"admin":      {TierLLMProviderError, TierLLMBudgetOut, TierLLMBudgetWarn, TierCourseNoTeacher, TierInviteExpired},
		"sinh viên":  {TierVerifyEmail, TierJoinCode, TierJoinPending},
	} {
		var items []Item
		for i := len(tiers) - 1; i >= 0; i-- { // đưa vào ngược
			items = append(items, item(fmt.Sprint(tiers[i]), tiers[i]))
		}
		Rank(items)
		var got []string
		for _, tr := range tiers {
			got = append(got, fmt.Sprint(tr))
		}
		require.Equal(t, got, idsOf(items), name)
		require.IsIncreasing(t, tiers)
	}
}

func TestRankAllTierPairs(t *testing.T) {
	t.Parallel()
	tiers := []int{10, 20, 30, 40, 45, 50, 60, 70, 80, 90, 95}
	for i, a := range tiers {
		for _, b := range tiers[i+1:] {
			items := []Item{item("b", b), item("a", a)}
			Rank(items)
			require.Equal(t, []string{"a", "b"}, idsOf(items), "bậc %d phải đứng trước %d", a, b)
		}
	}
}

func TestRankOverdueFirst(t *testing.T) {
	t.Parallel()
	late := item("late", TierCourseSetup)
	late.Overdue = true
	items := []Item{item("mismatch", TierEmailMismatch), item("req", TierJoinRequest), late}
	Rank(items)
	require.Equal(t, []string{"late", "mismatch", "req"}, idsOf(items), "quá hạn nổi lên trước mọi bậc")
}

func TestRankStableTiebreak(t *testing.T) {
	t.Parallel()
	mk := func(id string, age int, code string) Item {
		return Item{ID: id, Tier: TierJoinRequest, AgeMinutes: age, Course: &CourseRef{ClassCode: code}}
	}
	items := []Item{mk("z", 10, "B"), mk("a", 10, "B"), mk("m", 99, "Z"), mk("k", 10, "A")}
	Rank(items)
	require.Equal(t, []string{"m", "k", "a", "z"}, idsOf(items), "tuổi giảm dần, rồi mã lớp tăng dần, rồi id tăng dần")
}

func TestRankingDeterministic(t *testing.T) {
	t.Parallel()
	base := []Item{}
	for i := range 40 {
		base = append(base, Item{ID: fmt.Sprintf("i%02d", i), Tier: []int{10, 45, 80}[i%3], AgeMinutes: i % 5, Overdue: i%11 == 0, Course: &CourseRef{ClassCode: fmt.Sprintf("C%d", i%4)}})
	}
	want := append([]Item(nil), base...)
	Rank(want)
	rng := rand.New(rand.NewSource(1))
	for range 1000 {
		got := append([]Item(nil), base...)
		rng.Shuffle(len(got), func(i, j int) { got[i], got[j] = got[j], got[i] })
		Rank(got)
		require.Equal(t, idsOf(want), idsOf(got))
	}
}

type fakeProvider struct {
	name  string
	items []Item
	err   error
	panic bool
	slow  bool
}

func (f fakeProvider) Name() string { return f.name }
func (f fakeProvider) Items(ctx context.Context, _ Viewer, _ Scope) ([]Item, error) {
	if f.panic {
		panic("hỏng")
	}
	if f.slow {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.items, f.err
}

func TestProviderRegistry(t *testing.T) {
	t.Parallel()
	a := &Aggregator{}
	a.Register(fakeProvider{name: "một"})
	a.Register(fakeProvider{name: "hai"})
	require.Equal(t, []string{"một", "hai"}, a.Names())
}

func TestAggregatorSkipsFailingProvider(t *testing.T) {
	t.Parallel()
	logs := &bytes.Buffer{}
	a := &Aggregator{Log: slog.New(slog.NewTextHandler(logs, nil)), Timeout: 40 * time.Millisecond}
	a.Register(fakeProvider{name: "tốt", items: []Item{{ID: "ok", Kind: KindJoinCode, Tier: TierJoinCode}}})
	a.Register(fakeProvider{name: "lỗi", err: errors.New("db sập")})
	a.Register(fakeProvider{name: "panic", panic: true})
	a.Register(fakeProvider{name: "chậm", slow: true})
	start := time.Now()
	items, total, err := a.Collect(t.Context(), Viewer{Role: RoleStudent}, Scope{})
	require.NoError(t, err)
	require.Less(t, time.Since(start), time.Second, "Provider chậm bị cắt ở hạn riêng")
	require.Equal(t, []string{"ok"}, idsOf(items))
	require.Equal(t, 1, total)
	for _, p := range []string{"lỗi", "panic", "chậm"} {
		require.Contains(t, logs.String(), "provider="+p, "log warn cho Provider hỏng")
	}
	require.Contains(t, logs.String(), "level=WARN")
}

// Bộ gộp lọc theo vai: việc dành cho staff không bao giờ ra cho sinh viên dù Provider trả nhầm.
func TestAggregatorFiltersByRole(t *testing.T) {
	t.Parallel()
	a := &Aggregator{}
	a.Register(fakeProvider{name: "lẫn", items: []Item{
		{ID: "req", Kind: KindJoinRequest, Tier: TierJoinRequest}, {ID: "mm", Kind: KindEmailMismatch, Tier: TierEmailMismatch},
		{ID: "set", Kind: KindCourseSetup, Tier: TierCourseSetup}, {ID: "vf", Kind: KindVerifyEmail, Tier: TierVerifyEmail},
		{ID: "adm", Kind: KindCourseNoTeacher, Tier: TierCourseNoTeacher}, {ID: "lạ", Kind: "KIND_LẠ", Tier: 1},
	}})
	for role, want := range map[Role][]string{
		RoleStudent: {"vf"}, RoleTeacher: {"mm", "req", "set"}, RoleTA: {"req"}, RoleAdmin: {"adm"},
	} {
		items, _, err := a.Collect(t.Context(), Viewer{Role: role}, Scope{})
		require.NoError(t, err)
		require.Equal(t, want, idsOf(items), role)
	}
}

func TestAggregatorCapsAt50(t *testing.T) {
	t.Parallel()
	var many []Item
	for i := range 60 {
		many = append(many, Item{ID: fmt.Sprintf("i%02d", i), Kind: KindJoinRequest, Tier: TierJoinRequest})
	}
	a := &Aggregator{}
	a.Register(fakeProvider{name: "nhiều", items: many})
	items, total, err := a.Collect(t.Context(), Viewer{Role: RoleTeacher}, Scope{})
	require.NoError(t, err)
	require.Len(t, items, 50)
	require.Equal(t, 60, total, "count là tổng, danh sách bị cắt 50")
}

func TestAggregatorParentDeadline(t *testing.T) {
	t.Parallel()
	a := &Aggregator{}
	a.Register(fakeProvider{name: "chậm", slow: true})
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	_, _, err := a.Collect(ctx, Viewer{Role: RoleTeacher}, Scope{})
	require.ErrorIs(t, err, context.DeadlineExceeded, "yêu cầu hết hạn ⇒ lỗi (504), không trả trang rỗng")
}

func TestAgeFormat(t *testing.T) {
	t.Parallel()
	for d, want := range map[time.Duration]string{
		0: "0 phút", 59 * time.Minute: "59 phút", time.Hour: "1 giờ", 47*time.Hour + 59*time.Minute: "47 giờ",
		48 * time.Hour: "2 ngày", 73 * time.Hour: "3 ngày", -time.Minute: "0 phút",
	} {
		require.Equal(t, want, Age(d))
	}
}

func TestMaskEmail(t *testing.T) {
	t.Parallel()
	require.Equal(t, "a***@x.com", MaskEmail("abc@x.com"))
	require.Equal(t, "***", MaskEmail("khong-co-at"))
}

// AC1: gói này không import internal/llm (xếp hạng bằng luật cứng, không LLM).
func TestNoLLMImport(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		af, err := parser.ParseFile(token.NewFileSet(), f, raw, parser.ImportsOnly)
		require.NoError(t, err)
		for _, im := range af.Imports {
			require.NotContains(t, im.Path.Value, "internal/llm", f)
		}
	}
	_ = uuid.Nil
}
