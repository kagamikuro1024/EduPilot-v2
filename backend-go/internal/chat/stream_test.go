package chat_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/chat"
	"github.com/edupilot/backend-go/internal/llm"
)

// TestFirstEventBeforeProvider — AC1: sự kiện đầu tới ≤ 300 ms trong khi provider treo.
func TestFirstEventBeforeProvider(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	gate := make(chan struct{})
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: r.ag.stream(ctx, gate, time.Millisecond, llm.Response{}, "ok")}, nil
	}
	t0 := time.Now()
	m, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "hỏi gì đó")
	require.NoError(t, err)
	c := newCol()
	go func() { _ = r.svc.Tail(t.Context(), c, m, "") }()
	select {
	case <-c.ch:
	case <-time.After(5 * time.Second):
		t.Fatal("không có sự kiện nào")
	}
	if os.Getenv("EP_SKIP_TIMING") != "1" { // ngưỡng 300 ms chạy tuần tự; khi chạy song song chỉ kiểm thứ tự (sự kiện tới trước khi provider nhả)
		require.Less(t, time.Since(t0), 300*time.Millisecond)
	}
	require.Equal(t, chat.EvStatus, c.all()[0].Ev)
	require.Equal(t, "received", c.all()[0].D["stage"])
	close(gate)
	r.svc.Wait()
}

// TestStreamEventOrder + TestMessagesPersistedBeforeFirstEvent — AC2.
func TestStreamEventOrder(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, in agent.Input) (agent.Outcome, error) {
		in.OnStage("searching")
		return agent.Outcome{Plan: agent.IntentCourseQA, Hits: []hitT{hit("Quy chế", "Nội dung quy chế.")},
			Blocks: []agent.Block{{Kind: "upcoming_events", Data: map[string]any{"items": []string{}}}},
			Stream: r.ag.stream(ctx, nil, 5*time.Millisecond, llm.Response{MaskedCurrent: 2}, "Theo ", "quy chế [1] ", "và [7].")}, nil
	}
	m, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "Quy chế thi nói gì?")
	require.NoError(t, err)
	// trước sự kiện đầu: cả hai hàng đã có, ASSISTANT đang STREAMING, partial NULL
	require.Equal(t, 2, r.count(`select count(*) from chat_messages where session_id=$1`, r.sess))
	st, _, partial, _ := r.row(m.ID)
	require.Equal(t, "STREAMING", st)
	require.Nil(t, partial)
	c := newCol()
	require.NoError(t, r.svc.Tail(t.Context(), c, m, ""))
	ev := c.events()
	require.Equal(t, chat.EvStatus, ev[0])
	require.Equal(t, "received", c.all()[0].D["stage"])
	require.Equal(t, "searching", c.all()[1].D["stage"])
	idx := func(name string) int {
		for i, e := range ev {
			if e == name {
				return i
			}
		}
		return -1
	}
	require.Less(t, idx(chat.EvBlock), idx(chat.EvToken))
	require.Less(t, idx(chat.EvToken), idx(chat.EvNotice))
	require.Equal(t, chat.EvDone, ev[len(ev)-1])
	require.Equal(t, 2, int(c.all()[idx(chat.EvNotice)].D["masked"].(float64)))
	st, content, partial, _ := r.row(m.ID)
	require.Equal(t, "DONE", st)
	require.Nil(t, partial)
	require.Equal(t, "Theo quy chế [1] và.", content, "[7] không có trong danh sách bị bỏ")
	var mc int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select masked_count from chat_messages where id=$1`, m.ID).Scan(&mc))
	require.Equal(t, 2, mc)
}

// TestCitationsFromMarkers / TestDanglingMarkerStripped — AC15.
func TestCitationsFromMarkers(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	hs := []hitT{hit("A", "Đoạn A."), hit("B", "Đoạn B."), hit("C", "Đoạn C.")}
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Hits: hs, Stream: r.ag.stream(ctx, nil, time.Millisecond, llm.Response{}, "Có [2] rồi [2] và [9].")}, nil
	}
	c, mid, err := r.send(r.student, r.sess, "hỏi")
	require.NoError(t, err)
	done := c.last()
	cites := done.D["citations"].([]any)
	require.Len(t, cites, 1, "chỉ [2] được trích; trùng không lặp")
	require.Equal(t, "B", cites[0].(map[string]any)["title"])
	require.Equal(t, "Có [2] rồi [2] và.", done.D["content"])
	_, content, _, _ := r.row(mid)
	require.Equal(t, "Có [2] rồi [2] và.", content)
}

// TestPartialContentFlushCadence / TestFinalizeAtomic — AC3.
func TestPartialContentFlushCadence(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	pieces := make([]string, 30)
	for i := range pieces {
		pieces[i] = "ab"
	}
	r.ag.say(30*time.Millisecond, pieces...)
	m, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "dài")
	require.NoError(t, err)
	var seen []int
	deadline := time.After(10 * time.Second)
loop:
	for {
		select {
		case <-deadline:
			t.Fatal("quá hạn")
		case <-time.After(20 * time.Millisecond):
		}
		st, content, partial, _ := r.row(m.ID)
		if st == "DONE" {
			require.Nil(t, partial, "ghi DONE và xoá partial cùng giao dịch")
			require.Equal(t, strings.Repeat("ab", 30), content)
			break loop
		}
		if partial != nil {
			if n := len([]rune(*partial)); len(seen) == 0 || seen[len(seen)-1] != n {
				seen = append(seen, n)
			}
		}
	}
	require.GreaterOrEqual(t, len(seen), 3, "partial_content cập nhật nhiều lần trong lúc sinh")
	for i := 1; i < len(seen); i++ {
		require.Greater(t, seen[i], seen[i-1], "chỉ ghi khi có thay đổi")
	}
}

// TestResumeNoLossNoDup / TestResumeOtherInstance / TestResumeNoGapBetweenFlushes — AC4.
func TestResumeNoLossNoDup(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	pieces := make([]string, 25)
	for i := range pieces {
		pieces[i] = string(rune('a'+i%26)) + "ế "
	}
	r.ag.say(20*time.Millisecond, pieces...)
	m, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "dài")
	require.NoError(t, err)

	ctx1, cancel1 := context.WithCancel(t.Context())
	c1 := newCol()
	go func() { _ = r.svc.Tail(ctx1, c1, m, "") }()
	for len(c1.all()) < 8 { // ngắt giữa chừng
		<-c1.ch
	}
	cancel1()
	time.Sleep(50 * time.Millisecond)
	got := c1.all()
	lastID := got[len(got)-1].ID
	require.NotEmpty(t, lastID)
	require.True(t, strings.HasPrefix(lastID, "1:"))

	// nối lại từ bản gateway KHÁC (cùng Redis / DB), Last-Event-ID = khung cuối đã nhận
	other := r.service(chat.Config{StreamMax: 20 * time.Second})
	c2 := newCol()
	require.NoError(t, other.Tail(t.Context(), c2, m, lastID))
	all := append(got, c2.all()...)
	var b strings.Builder
	off := 0
	for _, f := range all {
		if f.Ev != chat.EvToken {
			continue
		}
		require.EqualValues(t, off, f.D["off"], "off liên tục: không khe, không lặp")
		s := f.D["t"].(string)
		b.WriteString(s)
		off += len([]rune(s))
	}
	r.svc.Wait()
	_, content, _, _ := r.row(m.ID)
	require.Equal(t, content, b.String(), "văn bản ghép bằng đúng lượt không bị ngắt")
	require.Equal(t, chat.EvDone, all[len(all)-1].Ev)
}

// TestReloadAfterDone — AC4: sinh xong khi đang vắng → mở lại thấy trạng thái cuối (có bộ đệm và không có bộ đệm).
func TestReloadAfterDone(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	c, mid, err := r.send(r.student, r.sess, "chào")
	require.NoError(t, err)
	require.Equal(t, "Xin chào bạn.", c.text())
	m, err := r.svc.Message(t.Context(), r.student, mid)
	require.NoError(t, err)

	again := newCol()
	require.NoError(t, r.svc.Tail(t.Context(), again, m, ""))
	require.Equal(t, "Xin chào bạn.", again.text())
	require.Equal(t, chat.EvDone, again.last().Ev)

	require.NoError(t, r.rdb.Del(t.Context(), "ep:chat:buf:"+mid.String()+":1").Err())
	snap := newCol()
	require.NoError(t, r.svc.Tail(t.Context(), snap, m, ""))
	require.Equal(t, chat.EvSnapshot, snap.all()[0].Ev)
	require.Equal(t, "Xin chào bạn.", snap.all()[0].D["t"])
	require.Equal(t, chat.EvDone, snap.last().Ev)
}

// TestResumeAfterRetryUsesAttempt — AC4: Last-Event-ID của lượt trước không bị hiểu nhầm sau retry.
func TestResumeAfterRetryUsesAttempt(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.ag.fn = func(context.Context, agent.TrustedContext, agent.Input) (agent.Outcome, error) {
		return agent.Outcome{}, errors.New("provider chết")
	}
	c, mid, err := r.send(r.student, r.sess, "hỏi")
	require.NoError(t, err)
	oldID := c.last().ID
	require.True(t, strings.HasPrefix(oldID, "1:"))
	r.ag.say(time.Millisecond, "Lần ", "hai.")
	m, err := r.svc.Retry(t.Context(), r.student, mid)
	require.NoError(t, err)
	require.EqualValues(t, 2, m.Attempt)
	c2 := newCol()
	require.NoError(t, r.svc.Tail(t.Context(), c2, m, oldID))
	require.Equal(t, "Lần hai.", c2.text(), "id lượt 1 không cắt mất đầu lượt 2")
	require.True(t, strings.HasPrefix(c2.all()[0].ID, "2:"))
}

// TestDisconnectDoesNotCancel / TestStreamMaxDuration120s — AC5.
func TestDisconnectDoesNotCancel(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.ag.say(30*time.Millisecond, "một ", "hai ", "ba ", "bốn.")
	ctx, cancel := context.WithCancel(t.Context())
	m, err := r.svc.Send(ctx, r.student, r.sess, uuid.New(), "chào")
	require.NoError(t, err)
	cancel() // request ngắt ngay
	r.svc.Wait()
	st, content, _, _ := r.row(m.ID)
	require.Equal(t, "DONE", st)
	require.Equal(t, "một hai ba bốn.", content)
	require.False(t, r.ag.provAbort.Load())
}

func TestStreamMaxDuration(t *testing.T) {
	t.Parallel()
	r := newRig(t, func(c *chat.Config) { c.StreamMax = 300 * time.Millisecond })
	gate := make(chan struct{})
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: r.ag.stream(ctx, gate, time.Millisecond, llm.Response{}, "x")}, nil
	}
	t0 := time.Now()
	c, mid, err := r.send(r.student, r.sess, "treo")
	require.NoError(t, err)
	require.Less(t, time.Since(t0), 3*time.Second)
	st, _, _, code := r.row(mid)
	require.Equal(t, "FAILED", st)
	require.Equal(t, "PROVIDER_ERROR", *code)
	require.Equal(t, chat.EvError, c.last().Ev)
	require.True(t, r.ag.provAbort.Load(), "ctx provider bị huỷ khi hết hạn")
}

// TestCancelReachesProvider / TestCancelIdempotent / TestCancelOtherUser404 / TestFinalWriteNeverOverwritesCancelled — AC6.
func TestCancelReachesProvider(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	gate := make(chan struct{})
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: r.ag.stream(ctx, gate, 10*time.Millisecond, llm.Response{}, "đã ", "có ", "chữ")}, nil
	}
	m, err := r.svc.Send(t.Context(), r.student, r.sess, uuid.New(), "chào")
	require.NoError(t, err)
	c := newCol()
	go func() { _ = r.svc.Tail(t.Context(), c, m, "") }()
	close(gate)
	for !strings.Contains(c.text(), "đã") {
		<-c.ch
	}
	t0 := time.Now()
	require.NoError(t, r.svc.Cancel(t.Context(), r.student, m.ID))
	r.svc.Wait()
	if os.Getenv("EP_SKIP_TIMING") != "1" { // AC6 ≤ 1 s chạy tuần tự; song song chỉ kiểm provider thật sự bị huỷ
		require.Less(t, time.Since(t0), time.Second)
	}
	require.True(t, r.ag.provAbort.Load(), "ctx của provider bị huỷ")
	st, _, partial, _ := r.row(m.ID)
	require.Equal(t, "CANCELLED", st)
	require.NotNil(t, partial)
	require.Contains(t, *partial, "đã")
	// lần hai: 204, không đổi
	require.NoError(t, r.svc.Cancel(t.Context(), r.student, m.ID))
	require.NoError(t, r.svc.Cancel(t.Context(), r.student, m.ID))
	st2, _, _, _ := r.row(m.ID)
	require.Equal(t, "CANCELLED", st2)
	// khoá CHAT_BUSY đã nhả: gửi tiếp được
	r.ag.say(time.Millisecond, "ok")
	_, _, err = r.send(r.student, r.sess, "tiếp")
	require.NoError(t, err)
}

func TestCancelOtherUser404(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, mid, err := r.send(r.student, r.sess, "chào")
	require.NoError(t, err)
	require.Equal(t, 404, httpStatus(r.svc.Cancel(t.Context(), r.other, mid)))
}

// TestCancelBeforeSubscribeNoOverwrite: không có G sống (không ai nghe) mà DB còn STREAMING → ghi CANCELLED thẳng, có điều kiện.
func TestCancelWithoutLiveGenerator(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, mid := r.orphan(r.student)
	require.NoError(t, r.svc.Cancel(t.Context(), r.student, mid))
	st, _, _, _ := r.row(mid)
	require.Equal(t, "CANCELLED", st)
}

// TestOverloadedEvent / TestRetryReusesAssistantRow / TestRetryResetsAllColumns — AC7, AC19.
func TestOverloadedEvent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.ag.fn = func(context.Context, agent.TrustedContext, agent.Input) (agent.Outcome, error) {
		return agent.Outcome{}, &llm.ErrOverloaded{RetryAfter: 20 * time.Second}
	}
	c, mid, err := r.send(r.student, r.sess, "hỏi")
	require.NoError(t, err)
	e := c.last()
	require.Equal(t, chat.EvError, e.Ev)
	require.Equal(t, "OVERLOADED", e.D["code"])
	require.EqualValues(t, 20, e.D["retry_after"])
	st, _, _, code := r.row(mid)
	require.Equal(t, "FAILED", st)
	require.Equal(t, "OVERLOADED", *code)

	users := r.count(`select count(*) from chat_messages where session_id=$1 and role='USER'`, r.sess)
	r.ag.say(time.Millisecond, "Xong.")
	m, err := r.svc.Retry(t.Context(), r.student, mid)
	require.NoError(t, err)
	require.Equal(t, mid, m.ID, "cùng hàng ASSISTANT")
	c2 := newCol()
	require.NoError(t, r.svc.Tail(t.Context(), c2, m, ""))
	require.Equal(t, users, r.count(`select count(*) from chat_messages where session_id=$1 and role='USER'`, r.sess))
	st, content, _, code := r.row(mid)
	require.Equal(t, "DONE", st)
	require.Equal(t, "Xong.", content)
	require.Nil(t, code)
}

func TestRetryResetsAllColumns(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, mid := r.orphan(r.student)
	_, err := r.pool.Exec(t.Context(), `update chat_messages set stream_status='FAILED', error_code='INTERRUPTED', completed_at=now(), partial_content='dở', citations='[{"n":1}]', blocks='[{"kind":"x"}]', confidence=0.9, low_confidence=true, no_context=true, degraded=true, masked_count=3 where id=$1`, mid)
	require.NoError(t, err)
	gate := make(chan struct{})
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentCourseQA, Stream: r.ag.stream(ctx, gate, time.Millisecond, llm.Response{}, "x")}, nil
	}
	_, err = r.svc.Retry(t.Context(), r.student, mid)
	require.NoError(t, err)
	var cnt int
	require.NoError(t, r.pool.QueryRow(t.Context(), `select count(*) from chat_messages where id=$1 and stream_status='STREAMING' and content='' and partial_content is null and completed_at is null and error_code is null and citations='[]' and blocks='[]' and confidence is null and not low_confidence and not no_context and not degraded and masked_count=0 and attempt=2`, mid).Scan(&cnt))
	require.Equal(t, 1, cnt)
	close(gate)
}

// TestRetryNotRetryable: chỉ FAILED / CANCELLED của mình.
func TestRetryNotRetryable(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, mid, err := r.send(r.student, r.sess, "chào")
	require.NoError(t, err)
	_, err = r.svc.Retry(t.Context(), r.student, mid)
	require.Equal(t, 409, httpStatus(err))
	_, err = r.svc.Retry(t.Context(), r.other, mid)
	require.Equal(t, 404, httpStatus(err))
}

// TestAllProvidersDownExtractive / NoContext / DegradedDropsLLMText / NoInstructorPromiseTextToStudent — AC8.
func TestAllProvidersDownExtractive(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	long := strings.Repeat("Câu số một dài lắm. ", 40)
	hs := []hitT{hit("Quy chế", long), hit("Giáo trình", "Đoạn hai."), hit("Slide", "Đoạn ba."), hit("Khác", "Đoạn bốn.")}
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		ch := make(chan llm.Chunk, 2)
		ch <- llm.Chunk{Text: llm.DegradedNoPassages, Degraded: true}
		ch <- llm.Chunk{Done: true, Response: &llm.Response{Degraded: true, Text: llm.DegradedNoPassages}}
		close(ch)
		return agent.Outcome{Plan: agent.IntentCourseQA, Hits: hs, Stream: ch}, nil
	}
	c, mid, err := r.send(r.student, r.sess, "Quy chế thi?")
	require.NoError(t, err)
	_, content, _, _ := r.row(mid)
	require.True(t, strings.HasPrefix(content, chat.DegradedNotice))
	require.NotContains(t, content, "giảng viên sẽ xem")
	require.NotContains(t, c.text(), "giảng viên sẽ xem")
	require.NotContains(t, content, "Đoạn bốn", "tối đa 3 đoạn")
	require.Contains(t, content, "[3]")
	for _, seg := range strings.Split(content, "«")[1:] {
		body, _, _ := strings.Cut(seg, "»")
		require.LessOrEqual(t, len([]rune(body)), 400)
	}
	require.Equal(t, true, c.last().D["degraded"])
	require.Len(t, c.last().D["citations"], 3)
	require.Equal(t, 1, r.count(`select count(*) from chat_messages where id=$1 and degraded`, mid))
}

func TestAllProvidersDownNoContext(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.ag.fn = func(ctx context.Context, _ agent.TrustedContext, _ agent.Input) (agent.Outcome, error) {
		ch := make(chan llm.Chunk, 2)
		ch <- llm.Chunk{Text: llm.DegradedNoPassages, Degraded: true}
		ch <- llm.Chunk{Done: true, Response: &llm.Response{Degraded: true}}
		close(ch)
		return agent.Outcome{Plan: agent.IntentCourseQA, NoContext: true, Stream: ch}, nil
	}
	c, mid, err := r.send(r.student, r.sess, "Giá vàng?")
	require.NoError(t, err)
	_, content, _, _ := r.row(mid)
	require.Equal(t, chat.DegradedNoContxt, content)
	require.NotContains(t, c.text(), "giảng viên")
}

// TestCannedRepliesSameShape — AC21: nhánh 0-LLM cùng khuôn status → một token đủ câu → done.
func TestCannedRepliesSameShape(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.ag.fn = func(context.Context, agent.TrustedContext, agent.Input) (agent.Outcome, error) {
		return agent.Outcome{Plan: agent.IntentOtherPerson, Canned: "Mình chỉ xem được dữ liệu của chính bạn."}, nil
	}
	c, mid, err := r.send(r.student, r.sess, "điểm của bạn khác")
	require.NoError(t, err)
	require.Equal(t, []string{chat.EvStatus, chat.EvToken, chat.EvDone}, c.events())
	_, content, _, _ := r.row(mid)
	require.Equal(t, "Mình chỉ xem được dữ liệu của chính bạn.", content)
	var intent string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select intent from chat_messages where id=$1`, mid).Scan(&intent))
	require.Equal(t, "OTHER_PERSON", intent)
	require.Zero(t, r.count(`select count(*) from pii_events where user_id=$1`, r.student.UserID), "intent không vào pii_events")
}
