package scheduler_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

var techWords = regexp.MustCompile(`(?i)provider|fallback|trace|RAG|PII`)

func downRig(t *testing.T) *rig {
	t.Helper()
	r := newRig(t, baseCfg(), fakeSettings(), nil)
	r.ctl.Set(fake.Settings{ErrorRate: 1, ErrorKind: provider.KindAuth})
	return r
}

func TestDegradedExtractive(t *testing.T) {
	t.Parallel()
	r := downRig(t)
	long := strings.Repeat("Hạn nộp bài là hết tuần 5. ", 40)
	ps := []llm.Passage{
		{Text: "đoạn thấp điểm", Source: "Đề cương", Page: 1, Score: 0.2},
		{Text: long, Source: "Quy chế", Page: 12, Score: 0.9},
		{Text: "Sinh viên nộp qua hệ thống.", Source: "Hướng dẫn", Page: 3, Score: 0.8},
		{Text: "đoạn thứ tư", Source: "Phụ lục", Page: 9, Score: 0.1},
		{Text: "Điểm danh theo buổi.", Source: "Đề cương", Page: 2, Score: 0.5},
	}
	resp, err := r.g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: msg("hạn nộp bài?"), Passages: ps})
	if err != nil {
		t.Fatalf("INTERACTIVE khi chuỗi chết phải trả 200 suy giảm, có lỗi: %v", err)
	}
	if !resp.Degraded || !strings.HasPrefix(resp.Text, "AI tạm thời không khả dụng. Dưới đây là các đoạn tài liệu liên quan nhất:") {
		t.Fatalf("resp = %+v", resp)
	}
	if n := strings.Count(resp.Text, "«"); n != 3 {
		t.Errorf("số đoạn = %d, muốn 3 (điểm cao nhất)", n)
	}
	for _, want := range []string{"— Quy chế, tr. 12", "— Hướng dẫn, tr. 3", "— Đề cương, tr. 2"} {
		if !strings.Contains(resp.Text, want) {
			t.Errorf("thiếu %q trong %q", want, resp.Text)
		}
	}
	if strings.Contains(resp.Text, "đoạn thấp điểm") || strings.Contains(resp.Text, "đoạn thứ tư") {
		t.Error("đoạn điểm thấp không được xuất hiện")
	}
	for _, block := range strings.Split(resp.Text, "\n\n")[1:] {
		inner := block[strings.Index(block, "«")+len("«") : strings.Index(block, "»")]
		if n := len([]rune(inner)); n > 600 {
			t.Errorf("đoạn dài %d ký tự, muốn ≤ 600", n)
		}
	}
	if techWords.MatchString(resp.Text) {
		t.Errorf("văn bản suy giảm chứa từ kỹ thuật: %q", resp.Text)
	}
	rows := r.cap.all(r.g)
	if len(rows) != 1 || !rows[0].Degraded || rows[0].Status != "degraded" {
		t.Errorf("audit = %+v", rows)
	}
}

func TestDegradedNoPassages(t *testing.T) {
	t.Parallel()
	r := downRig(t)
	resp, err := r.g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: msg("hạn nộp bài?")})
	if err != nil || !resp.Degraded || resp.Text != "AI tạm thời không khả dụng. Câu hỏi của bạn đã được ghi lại, giảng viên sẽ xem." {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
	// Stream cũng suy giảm (chưa phát byte nào)
	ch, err := r.g.Stream(t.Context(), llm.Request{Task: llm.TaskChat, Messages: msg("x"), Passages: []llm.Passage{{Text: "a", Source: "S", Page: 1, Score: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	var last llm.Chunk
	var text strings.Builder
	for c := range ch {
		text.WriteString(c.Text)
		last = c
	}
	if !last.Done || !last.Response.Degraded || !strings.Contains(text.String(), "«a» — S, tr. 1") {
		t.Errorf("stream suy giảm: %q %+v", text.String(), last)
	}
}

func TestBatchNoDegrade(t *testing.T) {
	t.Parallel()
	r := downRig(t)
	for _, task := range []llm.Task{llm.TaskGrading, llm.TaskInsight, llm.TaskQuestionGen, llm.TaskClassify, llm.TaskUtility} {
		_, err := r.g.Chat(t.Context(), llm.Request{Task: task, Messages: msg("x"), Passages: []llm.Passage{{Text: "a", Score: 1}}})
		var un *llm.ErrUnavailable
		if !errors.As(err, &un) || un.Reason != llm.ReasonAllFailed {
			t.Errorf("%s: err = %v, muốn LLM_UNAVAILABLE all_providers_failed", task, err)
		}
	}
}

func TestEveryRejectionHasCode(t *testing.T) {
	t.Parallel()
	// mỗi tình huống từ chối trả lỗi có kiểu để handler ánh xạ ra mã + message (không timeout trơn)
	cfg := baseCfg()
	cfg.MaxConcurrency, cfg.QueueMax = 1, 1
	r := newRig(t, cfg, fake.Settings{LatencyMin: 400 * 1e6, LatencyMax: 400 * 1e6}, nil)
	lane := llm.LaneBatch
	go func() {
		_, _ = r.g.Chat(t.Context(), llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: msg("giữ")})
	}()
	for i := 0; i < 200 && r.s.Stats(t.Context()).ProviderInflight < 1; i++ { // lời gọi đầu đã giữ chỗ
		sleepMs(5)
	}
	go func() {
		_, _ = r.g.Chat(t.Context(), llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: msg("hàng")})
	}()
	for i := 0; i < 200 && r.s.Stats(t.Context()).QueueDepth["BATCH"] < 1; i++ { // lời gọi hai chờ trong hàng (đầy: LLM_QUEUE_MAX=1)
		sleepMs(5)
	}
	_, err := r.g.Chat(t.Context(), llm.Request{Task: llm.TaskInsight, Lane: &lane, Messages: msg("đầy")})
	var ov *llm.ErrOverloaded
	if !errors.As(err, &ov) || ov.RetryAfter < 1e9 {
		t.Errorf("hàng đầy: %v", err)
	}
	rows := r.cap.all(r.g)
	found := false
	for _, row := range rows {
		found = found || row.Status == "overloaded"
	}
	if !found {
		t.Errorf("llm_audit phải ghi status=overloaded: %+v", rows)
	}
	// chưa cấu hình
	empty := llm.NewStaticRegistry(map[llm.Task]llm.Route{})
	g := llm.New(llm.Options{Registry: empty, Log: discard()})
	if _, err := g.Chat(t.Context(), llm.Request{Task: llm.TaskChat, Messages: msg("x")}); !errors.Is(err, llm.ErrNotConfigured) {
		t.Errorf("chưa cấu hình: %v", err)
	}
}

// Nhà cung cấp chết GIỮA luồng: đã phát token thì KHÔNG chuyển dự phòng; luồng kết thúc bằng lỗi có chủ đích (không im lặng).
func TestStreamMidFailure(t *testing.T) {
	t.Parallel()
	a := &stubProv{name: "A", stream: func(context.Context) (<-chan provider.Delta, error) {
		ch := make(chan provider.Delta, 3)
		ch <- provider.Delta{Text: "một nửa "}
		ch <- provider.Delta{Err: &provider.Error{Kind: provider.KindServer, Status: 503}}
		close(ch)
		return ch, nil
	}}
	b := &stubProv{name: "B"}
	r := stubRig(t, baseCfg(), a, b)
	ch, err := r.g.Stream(t.Context(), llm.Request{Task: llm.TaskChat, Messages: msg("x")})
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	var last llm.Chunk
	for c := range ch {
		text.WriteString(c.Text)
		last = c
	}
	if !errors.Is(last.Err, llm.ErrStream) || text.String() != "một nửa " || last.Done {
		t.Fatalf("text=%q last=%+v", text.String(), last)
	}
	if b.calls.Load() != 0 {
		t.Error("đã phát token thì không được sang nhà dự phòng")
	}
	if st := r.s.Stats(t.Context()); st.ProviderInflight != 0 {
		t.Errorf("chỗ chưa được trả: %+v", st)
	}
	rows := r.cap.all(r.g)
	if len(rows) != 1 || rows[0].Status != "error" {
		t.Errorf("audit = %+v", rows)
	}
}
