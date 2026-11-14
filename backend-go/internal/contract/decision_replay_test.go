package contract

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/llm/fake"
)

const replayFile = "../llm/testdata/decision-replay.json"

type chatRow struct {
	Text    string `json:"text"`
	Intent  string `json:"intent"`
	Stratum string `json:"stratum"`
}

type threadRow struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	Stratum string `json:"stratum"`
}

func jsonl[T any](t *testing.T, path string) []T {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	var out []T
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var r T
		require.NoError(t, json.Unmarshal(sc.Bytes(), &r))
		out = append(out, r)
	}
	return out
}

// TestDecisionReplayFile — US-DM-08 AC4 / US-DM-04 AC5: bản phát lại của provider decision giả (`llm/testdata/decision-replay.json`) được DỰNG từ bộ nhãn
// `benchmarks/decision/data` + 6 biến thể chèn lệnh 9.5 + 7 fixture bài tập, và khớp từng byte với tệp đã commit. Chạy với `UPDATE_DECISION_REPLAY=1` để dựng lại.
// Đồng thời kiểm tiền đề của bộ dữ liệu: luật chưa quyết đúng các câu «paraphrase» (nên decision có việc để làm) và khớp đúng ý định ở nhóm «rules».
func TestDecisionReplayFile(t *testing.T) {
	root := "../../.."
	want := map[string]fake.DecisionRecord{}
	num := func(v float64) fake.DecisionRecord { return fake.DecisionRecord{Noul: &v} }

	// CHAT_ROUTING: câu «paraphrase» và «course_qa» (luật rơi về COURSE_QA) → decision chọn đúng nhãn với 0,97.
	intents := []string{"WHAT_IF_GRADE", "GRADE_FORMULA", "PERSONAL_ATTENDANCE", "PERSONAL_PARTICIPATION", "PERSONAL_GRADE", "EXAM_SCHEDULE", "UPCOMING_EVENTS", "LIBRARY_SEARCH", "COURSE_QA", "OTHER_PERSON", "SMALLTALK"}
	for _, r := range jsonl[chatRow](t, filepath.Join(root, "benchmarks/decision/data/chat.jsonl")) {
		got := string(agent.DetectIntent(r.Text))
		switch r.Stratum {
		case "rules":
			require.Equal(t, r.Intent, got, "luật phải khớp: %s", r.Text)
			continue
		default:
			require.Equal(t, "COURSE_QA", got, "luật phải rơi về COURSE_QA: %s", r.Text)
		}
		probs := map[string]float64{}
		for _, k := range intents {
			probs[k] = 0.03 / float64(len(intents)-1)
		}
		probs[r.Intent] = 0.97
		want[fake.DecisionKey(r.Text, "intent")] = fake.DecisionRecord{Choice: r.Intent, Probabilities: probs}
	}

	// THREADS_PRECHECK: bài riêng tư tinh vi → có; bài sạch → không. Nhóm «rules» do luật bắt, không tới decision.
	for _, r := range jsonl[threadRow](t, filepath.Join(root, "benchmarks/decision/data/threads.jsonl")) {
		switch r.Stratum {
		case "private":
			want[fake.DecisionKey(r.Title+"\n"+r.Body, "private_or_named")] = num(0.92)
		case "clean":
			want[fake.DecisionKey(r.Title+"\n"+r.Body, "private_or_named")] = num(0.03)
		}
	}

	// INJECTION_FLAG: 5 biến thể 9.5 + s6 → có; 5 đối chứng còn lại (s4 trống không gọi decision) → không.
	files, err := filepath.Glob(filepath.Join(root, "benchmarks/real-llm/data/injection/*.md"))
	require.NoError(t, err)
	fx, err := filepath.Glob(filepath.Join(root, "docs/specs/FEAT-assignments/fixtures/submissions/*.md"))
	require.NoError(t, err)
	require.Len(t, files, 5)
	require.Len(t, fx, 7)
	for _, p := range append(files, fx...) {
		raw, err := os.ReadFile(p)
		require.NoError(t, err)
		cleaned := grading.StripIdentityLines(string(raw))
		if len(strings.Fields(cleaned)) < 20 {
			continue // bài ngắn: không qua tầng decision
		}
		yes := strings.Contains(p, "/injection/") || strings.HasSuffix(p, "s6-injection.md")
		v := 0.04
		if yes {
			v = 0.93
		}
		want[fake.DecisionKey(cleaned, "grader_instruction")] = num(v)
	}

	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	buf.WriteString("{\n")
	for i, k := range keys {
		b, err := json.Marshal(want[k])
		require.NoError(t, err)
		buf.WriteString("  \"" + k + "\": " + string(b))
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("}\n")
	if os.Getenv("UPDATE_DECISION_REPLAY") == "1" {
		require.NoError(t, os.WriteFile(replayFile, buf.Bytes(), 0o644)) //nolint:gosec // tệp dữ liệu test
		return
	}
	have, err := os.ReadFile(replayFile)
	require.NoError(t, err)
	require.Equal(t, buf.String(), string(have), "bản phát lại lệch bộ dữ liệu: chạy lại với UPDATE_DECISION_REPLAY=1")
}
