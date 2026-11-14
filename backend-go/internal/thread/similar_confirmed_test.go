package thread_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/thread"
)

type suggestGate bool

func (g suggestGate) On(uuid.UUID, string) bool { return bool(g) }

func unitVec(i int) []float32 {
	v := make([]float32, 1536)
	v[i] = 1
	return v
}

// verifiedWithVector: thread đã có câu trả lời AI được xác nhận + mục kho tri thức có vectơ `unitVec(i)`; trả id và tiêu đề.
func (r *rig) verifiedWithVector(i int) (uuid.UUID, string) {
	r.t.Helper()
	id := r.answered()
	_, err := r.svc.Decide(r.t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
	require.NoError(r.t, err)
	r.syncKnowledge()
	_, err = r.pool.Exec(r.t.Context(), `update content_chunks set embedding = $2::vector where id in (select k.chunk_id from knowledge_items k join forum_posts p on p.id = k.source_id where k.source_type = 'POST' and p.thread_id = $1)`, id, vecLit(unitVec(i)))
	require.NoError(r.t, err)
	var title string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select title from forum_threads where id = $1`, id).Scan(&title))
	return id, title
}

// TestSimilarConfirmed — M4 (FEAT-grading-guidance 4.4): chỉ thread «Đã xác nhận» cùng lớp, cosine ≥ min; cờ tắt / < 3 từ / có PII / nhúng lỗi → []; 0 lời gọi sinh chữ; quá hạn mức → 429.
func TestSimilarConfirmed(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ctx := t.Context()
	var embeds atomic.Int32
	cur := unitVec(0)
	var embedErr error
	r.svc.FW.Embed = func(context.Context, string) ([]float32, error) { embeds.Add(1); return cur, embedErr }
	r.svc.Suggest = thread.SuggestConfig{Gate: suggestGate(true), Min: 0.8, RPM: 6}
	id, title := r.verifiedWithVector(0)
	r.answered() // có bài AI nhưng CHƯA được xác nhận: không vào kho
	q := "Thi lại môn này được mấy lần"
	llmBase := r.llm.calls.Load() // các câu trả lời AI dựng sẵn ở trên đã dùng LLM

	got, err := r.svc.SimilarConfirmed(ctx, r.sv, r.course, q)
	require.NoError(t, err)
	require.Equal(t, []thread.SimilarConfirmedRow{{ThreadID: id, Title: title}}, got)
	require.Equal(t, llmBase, r.llm.calls.Load(), "0 lời gọi sinh chữ")

	cur = unitVec(1) // trực giao: cosine 0 < 0,8
	got, err = r.svc.SimilarConfirmed(ctx, r.sv, r.course, q)
	require.NoError(t, err)
	require.Empty(t, got)
	cur = unitVec(0)

	before := embeds.Load()
	got, err = r.svc.SimilarConfirmed(ctx, r.sv, r.course, "ngắn quá")
	require.NoError(t, err)
	require.Empty(t, got, "< 3 từ")
	got, err = r.svc.SimilarConfirmed(ctx, r.sv, r.course, "Em là "+svName+" muốn hỏi về thi lại")
	require.NoError(t, err)
	require.Empty(t, got, "có tên sinh viên")
	got, err = r.svc.SimilarConfirmed(ctx, r.sv, r.course, "MSSV "+svCode+" hỏi về thi lại môn này")
	require.NoError(t, err)
	require.Empty(t, got, "có MSSV")
	require.Equal(t, before, embeds.Load(), "bản nháp có PII / quá ngắn không được nhúng")

	got, err = r.svc.SimilarConfirmed(ctx, r.sv, uuid.New(), q)
	require.NoError(t, err)
	require.Empty(t, got, "lớp khác: không thấy thread của lớp này")

	embedErr = errors.New("provider chết")
	got, err = r.svc.SimilarConfirmed(ctx, r.other, r.course, q)
	require.NoError(t, err, "nhúng lỗi → [] im lặng")
	require.Empty(t, got)
	embedErr = nil

	_, err = r.pool.Exec(ctx, `update forum_posts set hidden_at = now(), hidden_reason = 'MODERATION' where thread_id = $1 and kind = 'AI'`, id)
	require.NoError(t, err)
	got, err = r.svc.SimilarConfirmed(ctx, r.teacher, r.course, q)
	require.NoError(t, err)
	require.Empty(t, got, "bài bị ẩn không gợi ý")

	off := *r.svc
	off.Suggest.Gate = suggestGate(false)
	got, err = off.SimilarConfirmed(ctx, r.sv, r.course, q)
	require.NoError(t, err)
	require.Empty(t, got, "cờ tắt")

	long := make([]rune, 4001)
	for i := range long {
		long[i] = 'a'
	}
	_, err = r.svc.SimilarConfirmed(ctx, r.sv, r.course, string(long))
	require.Equal(t, 422, status(err))
	for range 6 { // hạn mức tính theo người: r.other đã dùng 1 lượt (nhúng lỗi); các lượt còn lại cạn dần
		_, _ = r.svc.SimilarConfirmed(ctx, r.other, r.course, q)
	}
	_, err = r.svc.SimilarConfirmed(ctx, r.other, r.course, q)
	require.Equal(t, 429, status(err), "quá THREAD_SUGGEST_RPM")
}

// TestSimilarNoBodyLog — chữ đang soạn (chưa qua tường lửa) không vào log ở mọi nhánh và không ghi DB (thread / pii_events / outbox).
func TestSimilarNoBodyLog(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ctx := t.Context()
	var buf bytes.Buffer
	r.svc.Log = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	var embedErr error
	r.svc.FW.Embed = func(context.Context, string) ([]float32, error) { return unitVec(0), embedErr }
	r.svc.Suggest = thread.SuggestConfig{Gate: suggestGate(true), Min: 0.8, RPM: 100}
	r.verifiedWithVector(0)
	counts := func() [3]int {
		return [3]int{r.count(`select count(*) from forum_threads`), r.count(`select count(*) from pii_events`), r.count(`select count(*) from outbox`)}
	}
	before := counts()
	const canary = "CANARY_SOAN_7731"
	for _, q := range []string{"Thi lại " + canary + " mấy lần", canary + " " + svName + " hỏi thi lại", canary + " ngắn"} {
		_, err := r.svc.SimilarConfirmed(ctx, r.sv, r.course, q)
		require.NoError(t, err)
	}
	embedErr = errors.New("lỗi " + canary) // lỗi của nhà cung cấp cũng không được ghi lại kèm chữ
	_, err := r.svc.SimilarConfirmed(ctx, r.sv, r.course, "Thi lại "+canary+" mấy lần")
	require.NoError(t, err)
	require.NotContains(t, buf.String(), canary)
	require.NotContains(t, buf.String(), svName)
	require.Equal(t, before, counts(), "chỉ đọc: không ghi forum_threads / pii_events / outbox")
}
