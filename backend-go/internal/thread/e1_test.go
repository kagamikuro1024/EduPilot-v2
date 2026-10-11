package thread_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/thread"
)

type e1Item struct {
	ID, Split, Stratum, Text string
	ExpectBlock              bool `json:"expect_block"`
}

type rosterRow struct {
	N     int
	Name  string
	Code  string
	Email string
}

func fold(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'đ':
			b.WriteRune('d')
		case 'Đ':
			b.WriteRune('D')
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	for _, p := range [][2]string{{"àáạảãâầấậẩẫăằắặẳẵ", "a"}, {"èéẹẻẽêềếệểễ", "e"}, {"ìíịỉĩ", "i"}, {"òóọỏõôồốộổỗơờớợởỡ", "o"}, {"ùúụủũưừứựửữ", "u"}, {"ỳýỵỷỹ", "y"}} {
		for _, c := range p[0] {
			out = strings.ReplaceAll(out, string(c), p[1])
			out = strings.ReplaceAll(out, strings.ToUpper(string(c)), strings.ToUpper(p[1]))
		}
	}
	return out
}

func fill(text string, roster map[int]rosterRow) string {
	for n, r := range roster {
		w := strings.Fields(r.Name)
		rev := make([]string, len(w))
		for i := range w {
			rev[len(w)-1-i] = w[i]
		}
		p := fmt.Sprintf("{{SV%02d.", n)
		for k, v := range map[string]string{"name}}": r.Name, "name.noaccent}}": fold(r.Name), "name.reversed}}": strings.Join(rev, " "), "mssv}}": r.Code,
			"mssv.spaced}}": r.Code[:4] + " " + r.Code[4:], "mssv.dotted}}": r.Code[:4] + "." + r.Code[4:], "email}}": r.Email} {
			text = strings.ReplaceAll(text, p+k, v)
		}
	}
	return text
}

// TestE1DatasetMeetsGate — US-P3-07 AC8 ở tầng dịch vụ (không cần stack compose): 200 mẫu `benchmarks/pii/e1_dataset.jsonl` qua tường lửa thật với roster seed lớp 1;
// recall ≥ 0,95 và chặn nhầm ≤ 0,05 trên toàn bộ. Khi đỏ chỉ in các mẫu thuộc tập `dev` (tập `test` không được dùng để chỉnh luật).
func TestE1DatasetMeetsGate(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	var roster []rosterRow
	raw, err := os.ReadFile("../../../benchmarks/pii/roster_seed.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &roster))
	by := map[int]rosterRow{}
	for _, x := range roster {
		by[x.N] = x
		if x.N > 3 { // 1–3 có tên / MSSV cố định (người dùng của rig đã có sv, other); các dòng còn lại thêm vào lớp
			u := r.user(x.Name, "STUDENT", x.Email)
			r.enroll(u, "STUDENT", x.Code)
		}
	}
	f, err := os.Open("../../../benchmarks/pii/e1_dataset.jsonl")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	type cnt struct{ tp, fn, fp, tn int }
	tot, perSplit, perStratum := cnt{}, map[string]*cnt{"dev": {}, "test": {}}, map[string]*cnt{}
	var devMiss []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var it e1Item
		require.NoError(t, json.Unmarshal(sc.Bytes(), &it))
		v, err := r.svc.FW.CheckPost(t.Context(), r.course, "", fill(it.Text, by))
		require.NoError(t, err)
		blocked := !v.Allowed
		if perStratum[it.Stratum] == nil {
			perStratum[it.Stratum] = &cnt{}
		}
		for _, c := range []*cnt{&tot, perSplit[it.Split], perStratum[it.Stratum]} {
			switch {
			case it.ExpectBlock && blocked:
				c.tp++
			case it.ExpectBlock:
				c.fn++
			case blocked:
				c.fp++
			default:
				c.tn++
			}
		}
		if blocked != it.ExpectBlock && it.Split == "dev" {
			devMiss = append(devMiss, fmt.Sprintf("%s expect=%v: %s", it.ID, it.ExpectBlock, fill(it.Text, by)))
		}
	}
	require.NoError(t, sc.Err())
	keys := make([]string, 0, len(perStratum))
	for k := range perStratum {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("%s %+v", k, *perStratum[k])
	}
	t.Logf("tổng %+v | dev %+v | test %+v", tot, *perSplit["dev"], *perSplit["test"])
	recall, fb := float64(tot.tp)/100, float64(tot.fp)/100
	require.Equal(t, 200, tot.tp+tot.fn+tot.fp+tot.tn)
	if recall < 0.95 || fb > 0.05 {
		t.Fatalf("E1 chưa đạt: recall=%.2f false_block=%.2f; sai ở dev:\n%s", recall, fb, strings.Join(devMiss, "\n"))
	}
	_ = auth.RoleStudent
	_ = uuid.Nil
	_ = thread.Redacted
}
