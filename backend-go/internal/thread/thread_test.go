package thread_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/thread"
)

const clean = "Điều 5 quy chế học vụ nói gì về việc thi lại?"

func pg(n int) httpx.PageParams { return httpx.PageParams{Limit: n} }

// ---- precheck -------------------------------------------------------------------------------------------------------------------

// TestPrecheckNoWrites / Reasons — AC2: không ghi forum_* / pii_events; kiểm cả tiêu đề lẫn nội dung; sạch → allowed.
func TestPrecheckReasons(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	ok, err := r.svc.Precheck(t.Context(), r.sv, r.course, "Hỏi về Điều 5", clean)
	require.NoError(t, err)
	require.True(t, ok.Allowed)
	require.Empty(t, ok.Reasons)
	out, err := r.svc.Precheck(t.Context(), r.sv, r.course, "Em "+svName+" hỏi", "MSSV "+svCode+", mail "+svEmail+", số 0912345678, cccd 001203004567")
	require.NoError(t, err)
	require.False(t, out.Allowed)
	got := map[string]int{}
	for _, x := range out.Reasons {
		got[x.Type] = x.Count
	}
	require.Equal(t, map[string]int{"NAME": 1, "MSSV": 1, "EMAIL": 1, "PHONE": 1, "CCCD": 1}, got)
	require.NotContains(t, out.RedactedText+out.RedactedTitle, svCode)
	require.NotContains(t, out.RedactedTitle, svName)
	require.Contains(t, out.RedactedText, thread.Redacted)
}

// TestPrecheckNoWrites — AC2: kiểm tra không ghi hàng nào (forum_*, pii_events), kể cả khi phát hiện PII.
func TestPrecheckNoWrites(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, err := r.svc.Precheck(t.Context(), r.sv, r.course, "Em "+svName, "MSSV "+svCode+" mail "+svEmail)
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*) from forum_threads`)+r.count(`select count(*) from forum_posts`)+r.count(`select count(*) from pii_events`))
}

func TestPrecheckRateLimit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	var last error
	for range thread.PrecheckPerMin + 1 {
		_, last = r.svc.Precheck(t.Context(), r.sv, r.course, "", clean)
	}
	require.Equal(t, 429, status(last))
}

// ---- tường lửa khi đăng ---------------------------------------------------------------------------------------------------------------

// TestPostWithPIIBlocked / TestBlockedWritesPIIEventNoText — AC4.
func TestPostWithPIIBlocked(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, err := r.create(r.sv, "Hỏi", "MSSV "+svCode+" được mấy điểm?", false)
	require.Equal(t, 422, status(err))
	require.Equal(t, apierr.PIIDetected, code(err))
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	d := ae.Details.(map[string]any)
	require.Contains(t, d, "redacted_text")
	require.NotContains(t, fmt.Sprint(d["redacted_text"]), svCode)
	require.Zero(t, r.count(`select count(*) from forum_threads`)+r.count(`select count(*) from forum_posts`))
	require.Equal(t, 1, r.count(`select count(*) from pii_events where user_id=$1 and action='BLOCKED' and pii_type='MSSV' and count=1 and channel='PUBLIC'`, r.sv.UserID))
}

// TestBlockedWritesPIIEventNoText — AC4: sự kiện BLOCKED chỉ có loại + số đếm; không chuỗi nào đã gõ lọt vào hàng pii_events.
func TestBlockedWritesPIIEventNoText(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, err := r.create(r.sv, "Hỏi "+svName, "MSSV "+svCode+" mail "+svEmail, false)
	require.Equal(t, 422, status(err))
	require.Positive(t, r.count(`select count(*) from pii_events where action='BLOCKED'`))
	for _, raw := range []string{svCode, svEmail, svName} {
		require.Zero(t, r.count(`select count(*) from pii_events e where e::text like '%'||$1||'%'`, raw), raw)
	}
}

// TestPersonalQuestionBlockedNoRedactPath — AC8: câu hỏi riêng tư không định danh bị chặn và KHÔNG có lối "Ẩn rồi đăng" (redact:true vẫn 422).
func TestPersonalQuestionBlockedNoRedactPath(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	for _, redact := range []bool{false, true} {
		_, err := r.create(r.sv, "Điểm", "Em được mấy điểm giữa kỳ?", redact)
		require.Equal(t, apierr.PIIDetected, code(err), "redact=%v", redact)
		var ae *apierr.Error
		require.True(t, errors.As(err, &ae))
		require.Equal(t, true, ae.Details.(map[string]any)["personal_question"])
	}
	require.Zero(t, r.count(`select count(*) from forum_threads`))
	require.Equal(t, 2, r.count(`select count(*) from pii_events where user_id=$1 and pii_type='PERSONAL_QUESTION' and action='BLOCKED'`, r.sv.UserID))
}

// TestRawPIINeverStored — AC6: redact:true chỉ lưu bản "[đã ẩn]"; quét MỌI cột chữ của mọi bảng không thấy chuỗi đã gõ.
func TestRawPIINeverStored(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	phone, cccd := "0912345678", "001203004567"
	title := "Hỏi " + svName
	body := "MSSV " + svCode + ", mail " + svEmail + ", sđt " + phone + ", cccd " + cccd + ". Thi lại thế nào?"
	v, err := r.create(r.sv, title, body, true)
	require.NoError(t, err)
	require.Contains(t, v.Thread.Body, thread.Redacted)
	require.Positive(t, r.count(`select count(*) from pii_events where user_id=$1 and action='REDACTED'`, r.sv.UserID))
	_, err = r.svc.Comment(t.Context(), r.sv, r.course, v.Thread.ID, thread.CommentIn{Body: "Em là " + svName + " " + svCode, Redact: true})
	require.NoError(t, err)
	rows, err := r.pool.Query(t.Context(), `select table_name, column_name from information_schema.columns where table_schema='public' and data_type in ('text','character varying','jsonb','json','ARRAY') and table_name not in ('users','enrollments')`)
	require.NoError(t, err)
	type col struct{ t, c string }
	var cols []col
	for rows.Next() {
		var c col
		require.NoError(t, rows.Scan(&c.t, &c.c))
		cols = append(cols, c)
	}
	rows.Close()
	for _, c := range cols {
		for _, secret := range []string{svName, svCode, svEmail, phone, cccd} {
			require.Zero(t, r.count(fmt.Sprintf(`select count(*) from %q where %q::text ilike $1`, c.t, c.c), "%"+secret+"%"), "%q còn ở %s.%s", secret, c.t, c.c)
		}
	}
}

// TestServerEnforcesFirewallOnTitleBodyComment — AC7: tiêu đề, nội dung và bình luận đều bị chặn khi gọi thẳng API không redact.
func TestServerEnforcesFirewallOnTitleBodyComment(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	_, err := r.create(r.sv, "Em "+svName+" hỏi", clean, false) // PII chỉ ở TIÊU ĐỀ
	require.Equal(t, apierr.PIIDetected, code(err))
	_, err = r.create(r.sv, "Hỏi", "Email "+svEmail, false) // chỉ ở NỘI DUNG
	require.Equal(t, apierr.PIIDetected, code(err))
	id := r.mustCreate(r.sv, "Hỏi", clean)
	_, err = r.svc.Comment(t.Context(), r.other, r.course, id, thread.CommentIn{Body: "Bạn " + svName + " giải thích hộ"}) // BÌNH LUẬN
	require.Equal(t, apierr.PIIDetected, code(err))
	require.Zero(t, r.count(`select count(*) from forum_posts where kind='HUMAN'`))
}

// TestClientRedactedTextNotTrusted — AC7: server không nhận `redacted_text` của máy khách; tự ẩn; bản đã ẩn còn PII (tên viết lệch dấu cách ở thân lẫn tiêu đề) thì không lưu.
func TestClientRedactedTextNotTrusted(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	v, err := r.create(r.sv, "Hỏi", "Tên tôi "+strings.ToUpper(svName)+" và MSSV "+svCode, true)
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(v.Thread.Body), strings.ToLower(svName))
	require.NotContains(t, v.Thread.Body, svCode)
}

// TestAllWritePathsUseFirewall — AC19: mọi hàm ghi bài công khai (ThreadInsert / InsertForumPost của người) chỉ chạy sau `gate` → `CheckPost`.
func TestAllWritePathsUseFirewall(t *testing.T) {
	t.Parallel()
	for _, file := range []string{"service.go"} {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		require.NoError(t, err)
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			writes, gated := false, false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if c, ok := n.(*ast.CallExpr); ok {
					if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
						switch sel.Sel.Name {
						case "ThreadInsert", "InsertForumPost":
							writes = true
						case "gate", "CheckPost":
							gated = true
						}
					}
				}
				return true
			})
			if writes {
				require.True(t, gated, "%s.%s ghi bài công khai mà không qua tường lửa", file, fn.Name.Name)
			}
		}
	}
	// bài AI (answer.go) là ngoại lệ có tên: nội dung do máy sinh từ tài liệu của lớp, không phải chữ người dùng
	src, err := os.ReadFile("answer.go")
	require.NoError(t, err)
	require.NotContains(t, string(src), "ThreadInsert(")
}

// ---- đăng, khoá giờ thi, việc AI -------------------------------------------------------------------------------------------------

// TestThreadCreateLockedDuringExam / TestThreadLockCheckBeforeFirewall / TestThreadLockerErrorDenies / TestThreadReadAllowedWhileLocked — AC20.
func (r *rig) lockedFor(d time.Duration) (uuid.UUID, time.Time) {
	att := uuid.New()
	until := time.Now().Add(d).UTC()
	r.lock.lock = &exam.Lock{AttemptID: att, Until: until}
	return att, until
}

func TestThreadCreateLockedDuringExam(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	att, until := r.lockedFor(20 * time.Minute)
	_, err := r.create(r.sv, "Hỏi", clean, false)
	require.Equal(t, apierr.ExamInProgress, code(err))
	require.Equal(t, 409, status(err))
	var ae *apierr.Error
	require.True(t, errors.As(err, &ae))
	require.Equal(t, until.Format(time.RFC3339), ae.Details.(map[string]any)["until"])
	require.Zero(t, r.count(`select count(*) from forum_threads`))
	require.Equal(t, []uuid.UUID{att}, r.lock.blocked, "ghi exam_events CHAT_BLOCKED qua exam.RecordChatBlocked")
}

// TestThreadLockCheckBeforeFirewall — AC20: tin có PII trong giờ thi vẫn nhận 409 (khoá thắng tường lửa), không ghi pii_events.
func TestThreadLockCheckBeforeFirewall(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.lockedFor(20 * time.Minute)
	_, err := r.create(r.sv, "Hỏi", "MSSV "+svCode, false)
	require.Equal(t, apierr.ExamInProgress, code(err))
	require.Zero(t, r.count(`select count(*) from pii_events`)+r.count(`select count(*) from forum_threads`))
}

// TestThreadLockerErrorDenies — AC20: không đọc được khoá → từ chối (503), không "mở" nhầm.
func TestThreadLockerErrorDenies(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.lock.err = errors.New("db chết")
	_, err := r.create(r.sv, "Hỏi", clean, false)
	require.Equal(t, apierr.ChatUnavailable, code(err))
	require.Zero(t, r.count(`select count(*) from forum_threads`))
}

// TestThreadReadAllowedWhileLocked — AC20: trong giờ thi vẫn ĐỌC được danh sách và chi tiết; bình luận và precheck không bị khoá (chủ dự án chỉ nêu thread mới).
func TestThreadReadAllowedWhileLocked(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	r.lockedFor(20 * time.Minute)
	_, err := r.svc.List(t.Context(), r.sv, r.course, thread.Filter{}, pg(10))
	require.NoError(t, err)
	_, err = r.svc.Get(t.Context(), r.sv, r.course, id, pg(10))
	require.NoError(t, err)
	_, err = r.svc.Precheck(t.Context(), r.sv, r.course, "", clean)
	require.NoError(t, err)
	_, err = r.svc.Comment(t.Context(), r.sv, r.course, id, thread.CommentIn{Body: "bổ sung"})
	require.NoError(t, err)
}

// TestThreadUnlockedAfterSubmit — AC20: hết khoá (nộp bài) → đăng được ngay.
func TestThreadUnlockedAfterSubmit(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.lockedFor(20 * time.Minute)
	_, err := r.create(r.sv, "Hỏi", clean, false)
	require.Equal(t, apierr.ExamInProgress, code(err))
	r.lock.lock = nil
	r.mustCreate(r.sv, "Hỏi", clean)
}

// TestThreadCreateEnqueuesAnswerJob — AC9: đăng thread xếp ĐÚNG MỘT việc `thread.answer` trong cùng giao dịch (+ outbox thread.created); bình luận không xếp việc AI.
func TestThreadCreateEnqueuesAnswerJob(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	require.Equal(t, 1, r.count(`select count(*) from jobs where kind=$1 and owner_id=$2`, thread.KindAnswer, r.sv.UserID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic=$1`, thread.TopicCreated))
	_, err := r.svc.Comment(t.Context(), r.other, r.course, id, thread.CommentIn{Body: "Mình cũng thắc mắc"})
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from jobs where kind=$1`, thread.KindAnswer), "AI không trả lời bình luận")
}

func (r *rig) answer(id uuid.UUID) error {
	return r.svc.Answer(r.t.Context(), uuid.Nil, r.sv.UserID, id)
}

// TestAIAnswerOncePerThread — AC9: một lời gọi Chat, đúng một bài AI PENDING có ≥ 1 nguồn, ai_state=ANSWERED.
func TestAIAnswerOncePerThread(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.rag.hits = []rag.Hit{hit(0.8), hit(0.6)}
	id := r.mustCreate(r.sv, "Thi lại", clean)
	require.NoError(t, r.answer(id))
	require.Equal(t, int32(1), r.llm.calls.Load(), "một lời gọi Chat")
	require.Equal(t, 1, r.count(`select count(*) from forum_posts where thread_id=$1 and kind='AI' and verification_state='PENDING' and jsonb_array_length(citations) >= 1`, id))
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id=$1 and ai_state='ANSWERED'`, id))
}

// TestAIAnswerUsesNearRealtimeLane — AC9: làn NEAR_REALTIME, không chạm INTERACTIVE của chat riêng.
func TestAIAnswerUsesNearRealtimeLane(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.rag.hits = []rag.Hit{hit(0.8)}
	require.NoError(t, r.answer(r.mustCreate(r.sv, "Thi lại", clean)))
	require.Equal(t, []llm.Lane{llm.LaneNearRealtime}, r.llm.lanes)
}

// TestAIAnswerRedeliveryIdempotent — AC9: giao lại việc không tạo bài AI thứ hai, không gọi LLM lần nữa, không thêm thông báo.
func TestAIAnswerRedeliveryIdempotent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.rag.hits = []rag.Hit{hit(0.8)}
	id := r.mustCreate(r.sv, "Thi lại", clean)
	for range 3 {
		require.NoError(t, r.answer(id))
	}
	require.Equal(t, int32(1), r.llm.calls.Load())
	require.Equal(t, 1, r.count(`select count(*) from forum_posts where thread_id=$1 and kind='AI'`, id))
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='THREAD_ANSWERED'`, r.sv.UserID))
}

// TestAIDoesNotAnswerComments — AC9: bình luận không xếp việc AI.
func TestAIDoesNotAnswerComments(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	_, err := r.svc.Comment(t.Context(), r.other, r.course, id, thread.CommentIn{Body: "Mình cũng thắc mắc"})
	require.NoError(t, err)
	require.Equal(t, 1, r.count(`select count(*) from jobs where kind=$1`, thread.KindAnswer))
	require.Zero(t, r.llm.calls.Load())
}

// TestThreadCreatedHandlerOnlyEnqueues — AC9 / TLR-9: handler `job.enqueue` của `thread.answer` chỉ XADD vào ep:ingest rồi trả ngay: không gọi LLM, việc không bị đánh dấu xong.
func TestThreadCreatedHandlerOnlyEnqueues(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	run := jobs.NewRunner(r.pool, nil, clock.Real{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	thread.RegisterKind(run, r.rdb)
	var msg outbox.Message
	require.NoError(t, r.pool.QueryRow(t.Context(), `select id, topic, payload from outbox where topic='job.enqueue' and payload::text like '%'||$1||'%'`, id.String()).Scan(&msg.ID, &msg.Topic, &msg.Payload))
	before, err := r.rdb.XLen(t.Context(), ingest.StreamName).Result()
	require.NoError(t, err)
	require.NoError(t, run.HandleMessage(t.Context(), msg))
	after, err := r.rdb.XLen(t.Context(), ingest.StreamName).Result()
	require.NoError(t, err)
	require.Equal(t, before+1, after, "đúng một tin vào ep:ingest")
	require.Zero(t, r.llm.calls.Load(), "handler outbox không gọi LLM")
	require.Zero(t, r.count(`select count(*) from jobs where kind=$1 and status in ('SUCCEEDED','FAILED')`, thread.KindAnswer))
}

// TestAISkippedNoContext / TestAISkippedLowScore — AC10: không đủ tin cậy → không bài AI, không gọi LLM, ai_state=SKIPPED + lý do.
func (r *rig) skipped(hits []rag.Hit, reason string) {
	r.t.Helper()
	r.rag.hits = hits
	id := r.mustCreate(r.sv, "Hỏi", clean)
	require.NoError(r.t, r.answer(id))
	require.Equal(r.t, 1, r.count(`select count(*) from forum_threads where id=$1 and ai_state='SKIPPED' and ai_skip_reason=$2`, id, reason))
	require.Zero(r.t, r.count(`select count(*) from forum_posts where thread_id=$1`, id))
	require.Zero(r.t, r.llm.calls.Load())
}

func TestAISkippedNoContext(t *testing.T) { t.Parallel(); newRig(t).skipped(nil, thread.SkipNoContext) }
func TestAISkippedLowScore(t *testing.T) {
	t.Parallel()
	newRig(t).skipped([]rag.Hit{hit(0.1)}, thread.SkipLowScore)
}

// TestAISkippedOnLastAttempt — AC10: lỗi truy xuất ở lần thử cuối cũng tự ghi SKIPPED (thread không kẹt PENDING).
func TestAISkippedOnLastAttempt(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.rag.err = errors.New("pgvector chết")
	id := r.mustCreate(r.sv, "Hỏi", clean)
	require.Error(t, r.answer(id))
	require.Error(t, r.answer(id))
	require.NoError(t, r.answer(id))
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id=$1 and ai_state='SKIPPED'`, id))
}

func TestAISkippedLLMDownOnLastAttempt(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.rag.hits = []rag.Hit{hit(0.8)}
	r.llm.err = &llm.ErrOverloaded{RetryAfter: time.Second}
	id := r.mustCreate(r.sv, "Hỏi", clean)
	require.Error(t, r.answer(id), "lần 1: lỗi tạm thời → giữ tin để giao lại")
	require.Error(t, r.answer(id), "lần 2")
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id=$1 and ai_state='PENDING'`, id), "chưa bỏ cuộc")
	require.NoError(t, r.answer(id), "lần 3 (cuối): tự ghi SKIPPED, thread không kẹt PENDING")
	require.Equal(t, 1, r.count(`select count(*) from forum_threads where id=$1 and ai_state='SKIPPED' and ai_skip_reason='LLM_UNAVAILABLE'`, id))
	require.Zero(t, r.count(`select count(*) from forum_posts where thread_id=$1`, id))
}

// ---- projection, quyết định, đọc ----------------------------------------------------------------------------------------------------

func (r *rig) answered() uuid.UUID {
	r.t.Helper()
	r.rag.hits = []rag.Hit{hit(0.8)}
	id := r.mustCreate(r.sv, "Thi lại", clean)
	require.NoError(r.t, r.answer(id))
	return id
}

func (r *rig) aiPost(thread uuid.UUID) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select id from forum_posts where thread_id=$1 and kind='AI'`, thread).Scan(&id))
	return id
}

// TestStudentProjectionNoConfidence — AC11: JSON của sinh viên không có khoá confidence / ai_body / định danh nội bộ; bài AI không tác giả, nhãn PENDING.
func TestStudentProjectionNoConfidence(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	sv, err := r.svc.Get(t.Context(), r.other, r.course, id, pg(30))
	require.NoError(t, err)
	raw, _ := json.Marshal(sv)
	for _, k := range []string{"confidence", "ai_body", "retrieval_score", "groundedness", "rejected", "hidden_reason", "verified_by", "user_id", "email", "author_id", "ai_state"} {
		require.NotContains(t, string(raw), `"`+k+`"`, "projection sinh viên lộ %s", k)
	}
	require.Len(t, sv.Posts, 1)
	require.Nil(t, sv.Posts[0].Author)
	require.Equal(t, "PENDING", *sv.Posts[0].Verification)
}

// TestStaffProjectionHasConfidence — AC11, AC16: Staff thấy confidence (3 chữ số thập phân) và ai_state.
func TestStaffProjectionHasConfidence(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	st, err := r.svc.Get(t.Context(), r.ta, r.course, id, pg(30))
	require.NoError(t, err)
	raw, _ := json.Marshal(st)
	require.Contains(t, string(raw), `"confidence":"1.000"`)
	require.Contains(t, string(raw), `"ai_state"`)
}

// TestAuthorNameVisibleToClass — AC16 (Q3): tên người đăng công khai với cả lớp; `is_me` đúng theo người xem.
func TestAuthorNameVisibleToClass(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	other, err := r.svc.Get(t.Context(), r.other, r.course, id, pg(30))
	require.NoError(t, err)
	require.Equal(t, svName, other.Thread.Author.FullName)
	require.False(t, other.Thread.Author.IsMe)
	mine, err := r.svc.Get(t.Context(), r.sv, r.course, id, pg(30))
	require.NoError(t, err)
	require.True(t, mine.Thread.Author.IsMe)
}

// TestAuthorNoSensitiveFields — AC16: tác giả chỉ có tên + vai + is_me; không MSSV / email / id người dùng ở bất kỳ vai nào.
func TestAuthorNoSensitiveFields(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	_, err := r.svc.Comment(t.Context(), r.sv, r.course, id, thread.CommentIn{Body: "bổ sung"})
	require.NoError(t, err)
	for _, a := range []thread.Actor{r.other, r.ta, r.teacher} {
		v, err := r.svc.Get(t.Context(), a, r.course, id, pg(30))
		require.NoError(t, err)
		l, err := r.svc.List(t.Context(), a, r.course, thread.Filter{}, pg(30))
		require.NoError(t, err)
		raw, _ := json.Marshal([]any{v, l})
		for _, bad := range []string{svCode, svEmail, r.sv.UserID.String(), `"email"`, `"student_code"`, `"user_id"`} {
			require.NotContains(t, string(raw), bad)
		}
	}
}

// TestVerifyCorrectReject — AC12: Xác nhận → VERIFIED; Chỉnh sửa (kèm version) → CORRECTED, bản AI gốc giữ ở ai_body; Loại → REJECTED.
func TestVerifyCorrectReject(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	pid := r.aiPost(r.answered())
	v, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.NoError(t, err)
	require.Equal(t, "VERIFIED", *v.Verification)
	c, err := r.svc.Decide(t.Context(), r.teacher, r.course, pid, thread.Correct, "Bản đã sửa bởi giảng viên.", v.Version)
	require.NoError(t, err)
	require.Equal(t, "CORRECTED", *c.Verification)
	require.Equal(t, "Bản đã sửa bởi giảng viên.", c.Body)
	require.Contains(t, *c.AIBody, "Theo quy chế")
	rj, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Reject, "", 0)
	require.NoError(t, err)
	require.Equal(t, "REJECTED", *rj.Verification)
}

// TestDecisionAudited — AC12: mỗi quyết định thật có đúng một dòng audit_log; quyết định lặp lại không thêm.
func TestDecisionAudited(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	pid := r.aiPost(r.answered())
	v, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.NoError(t, err)
	_, err = r.svc.Decide(t.Context(), r.teacher, r.course, pid, thread.Correct, "sửa", v.Version)
	require.NoError(t, err)
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Reject, "", 0)
	require.NoError(t, err)
	require.Equal(t, 3, r.count(`select count(*) from audit_log where entity='forum_post' and entity_id=$1`, pid.String()))
}

// TestRejectedNeverVisibleToStudent — AC12: quét 5 đường đọc của sinh viên (danh sách, lọc pending, lọc verified, chi tiết, thread tương tự): bài AI bị loại không lộ trạng thái / chữ.
func TestRejectedNeverVisibleToStudent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	pid := r.aiPost(id)
	c, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Correct, "Chữ-riêng-của-bản-sửa", 1)
	require.NoError(t, err)
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Reject, "", c.Version)
	require.NoError(t, err)
	var reads []any
	for _, f := range []thread.Filter{{}, {State: "pending"}, {State: "verified"}} {
		l, err := r.svc.List(t.Context(), r.other, r.course, f, pg(10))
		require.NoError(t, err)
		reads = append(reads, l)
		if f.State == "" {
			require.Len(t, l.Items, 1, "thread vẫn hiện")
			require.Nil(t, l.Items[0].AnswerState)
		} else {
			require.Empty(t, l.Items, f.State)
		}
	}
	view, err := r.svc.Get(t.Context(), r.other, r.course, id, pg(30))
	require.NoError(t, err)
	require.Empty(t, view.Posts)
	sim, err := r.svc.Similar(t.Context(), r.course, r.mustCreate(r.sv, "Khác", clean))
	require.NoError(t, err)
	reads = append(reads, view, sim)
	raw, _ := json.Marshal(reads)
	require.NotContains(t, string(raw), "Chữ-riêng-của-bản-sửa")
	require.NotContains(t, string(raw), "REJECTED")
	st, err := r.svc.Get(t.Context(), r.ta, r.course, id, pg(30))
	require.NoError(t, err)
	require.True(t, *st.Posts[0].Rejected, "Staff vẫn thấy dòng Đã loại")
}

// TestListHidesRejectedFromStudent — AC1: bài AI REJECTED không hiện trong thread, thread vẫn hiện trong danh sách.
func TestListHidesRejectedFromStudent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Reject, "", 0)
	require.NoError(t, err)
	l, err := r.svc.List(t.Context(), r.other, r.course, thread.Filter{}, pg(10))
	require.NoError(t, err)
	require.Len(t, l.Items, 1)
	require.Nil(t, l.Items[0].AnswerState)
}

// TestDecisionIdempotent — AC13: bấm đúp cùng quyết định → không đổi, không thêm thông báo, không thêm audit.
func TestDecisionIdempotent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	pid := r.aiPost(r.answered())
	a, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.NoError(t, err)
	b, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.NoError(t, err)
	require.Equal(t, a.Version, b.Version)
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='THREAD_VERIFIED'`, r.sv.UserID))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity_id=$1`, pid.String()))
}

// TestDecisionConflict409 — AC13: xác nhận bài đã REJECTED, quyết định trên bình luận người → 409 POST_STATE_CONFLICT; bài lạ → 404.
func TestDecisionConflict409(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	pid := r.aiPost(id)
	_, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Reject, "", 0)
	require.NoError(t, err)
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.Equal(t, apierr.PostStateConflict, code(err))
	require.Equal(t, 409, status(err))
	h, _ := r.svc.Comment(t.Context(), r.other, r.course, id, thread.CommentIn{Body: "ok"})
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, h.ID, thread.Verify, "", 0)
	require.Equal(t, apierr.PostStateConflict, code(err))
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, uuid.New(), thread.Verify, "", 0)
	require.Equal(t, 404, status(err))
}

// TestCorrectVersionConflict — AC13: sửa với version cũ → 409 VERSION_CONFLICT; version đúng → được.
func TestCorrectVersionConflict(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	pid := r.aiPost(r.answered())
	a, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Verify, "", 0)
	require.NoError(t, err)
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Correct, "sửa", a.Version-1)
	require.Equal(t, apierr.VersionConflict, code(err))
	c, err := r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Correct, "sửa", a.Version)
	require.NoError(t, err)
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, pid, thread.Correct, "sửa nữa", c.Version+1)
	require.Equal(t, 409, status(err))
}

// TestPosterNotified — AC14: người đăng nhận chuông khi có bình luận của người khác, khi AI trả lời, khi bài được xác nhận / sửa; link về thread.
func TestPosterNotified(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	_, err := r.svc.Comment(t.Context(), r.other, r.course, id, thread.CommentIn{Body: "Mình trả lời nhé"})
	require.NoError(t, err)
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
	require.NoError(t, err)
	for _, typ := range []string{"THREAD_ANSWERED", "THREAD_REPLY", "THREAD_VERIFIED"} {
		require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type=$2 and link=$3`, r.sv.UserID, typ, "/threads/"+id.String()), typ)
	}
}

// TestNoSelfNotification — AC14: tự bình luận thread của mình không tự báo mình.
func TestNoSelfNotification(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.mustCreate(r.sv, "Hỏi", clean)
	_, err := r.svc.Comment(t.Context(), r.sv, r.course, id, thread.CommentIn{Body: "Em bổ sung thêm"})
	require.NoError(t, err)
	require.Zero(t, r.count(`select count(*) from notifications where user_id=$1`, r.sv.UserID))
}

// TestNotificationDedupe — AC14: giao lại việc AI và bấm đúp Xác nhận không tạo thông báo thứ hai (dedupe_key theo bài / quyết định).
func TestNotificationDedupe(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	require.NoError(t, r.answer(id))
	for range 2 {
		_, err := r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(id), thread.Verify, "", 0)
		require.NoError(t, err)
	}
	require.Equal(t, 2, r.count(`select count(*) from notifications where user_id=$1`, r.sv.UserID), "một ANSWERED + một VERIFIED")
}

// TestListFilters / TestListCursor — AC1.
func TestListFilters(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	w3 := 3
	mk := func(title string, week *int, tags []string) uuid.UUID {
		v, err := r.svc.Create(t.Context(), r.sv, r.course, thread.CreateIn{Title: title, Body: clean, Tags: tags, WeekNo: week})
		require.NoError(t, err)
		return v.Thread.ID
	}
	a := mk("Cảnh báo học vụ", &w3, []string{"quy-che"})
	mk("Giao thức TCP", nil, []string{"mang"})
	r.rag.hits = []rag.Hit{hit(0.9)}
	require.NoError(t, r.answer(a))
	ids := func(f thread.Filter) []uuid.UUID {
		p, err := r.svc.List(t.Context(), r.other, r.course, f, pg(30))
		require.NoError(t, err)
		var out []uuid.UUID
		for _, x := range p.Items {
			out = append(out, x.ID)
		}
		return out
	}
	require.Len(t, ids(thread.Filter{}), 2)
	require.Equal(t, []uuid.UUID{a}, ids(thread.Filter{Week: &w3}))
	require.Equal(t, []uuid.UUID{a}, ids(thread.Filter{Tag: "quy-che"}))
	require.Equal(t, []uuid.UUID{a}, ids(thread.Filter{State: "pending"}))
	require.Len(t, ids(thread.Filter{State: "none"}), 1)
	require.Equal(t, []uuid.UUID{a}, ids(thread.Filter{Q: "canh bao hoc vu"}), "tìm không dấu")
	require.Empty(t, ids(thread.Filter{Q: "100%"}), "ký tự đặc biệt không thành mẫu")
}

func TestListCursor(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	for i := range 5 {
		r.mustCreate(r.sv, fmt.Sprintf("Câu hỏi %d", i), clean)
		time.Sleep(5 * time.Millisecond)
	}
	p1, err := r.svc.List(t.Context(), r.other, r.course, thread.Filter{}, pg(2))
	require.NoError(t, err)
	require.Len(t, p1.Items, 2)
	require.NotNil(t, p1.NextCursor)
	seen := map[uuid.UUID]bool{}
	for _, x := range p1.Items {
		seen[x.ID] = true
	}
	cur, err := httpx.ParseCursor(*p1.NextCursor)
	require.NoError(t, err)
	page := pg(10)
	page.Cursor = &cur
	p2, err := r.svc.List(t.Context(), r.other, r.course, thread.Filter{}, page)
	require.NoError(t, err)
	require.Len(t, p2.Items, 3)
	for _, x := range p2.Items {
		require.False(t, seen[x.ID])
	}
}

// TestSimilarThreads / TestSimilarExcludesHidden — AC15.
func TestSimilarThreads(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	vec := func(i int) string {
		parts := make([]string, 1536)
		for k := range parts {
			parts[k] = "0"
		}
		parts[i] = "1"
		return "[" + strings.Join(parts, ",") + "]"
	}
	a := r.mustCreate(r.sv, "A", clean)
	b := r.mustCreate(r.sv, "B", clean)
	c := r.mustCreate(r.sv, "C", clean)
	d := r.mustCreate(r.sv, "D", clean)
	for id, v := range map[uuid.UUID]string{a: vec(0), b: vec(0), c: vec(1), d: vec(0)} {
		_, err := r.pool.Exec(t.Context(), `update forum_threads set embedding=$2::vector where id=$1`, id, v)
		require.NoError(t, err)
	}
	got, err := r.svc.Similar(t.Context(), r.course, a)
	require.NoError(t, err)
	ids := map[uuid.UUID]bool{}
	for _, x := range got {
		ids[x.ID] = true
	}
	require.Equal(t, map[uuid.UUID]bool{b: true, d: true}, ids, "không gồm chính nó và thread khác hướng")
	// thread có bài AI bị loại không xuất hiện
	r.rag.hits = []rag.Hit{hit(0.9)}
	require.NoError(t, r.answer(b))
	_, err = r.svc.Decide(t.Context(), r.ta, r.course, r.aiPost(b), thread.Reject, "", 0)
	require.NoError(t, err)
	got, _ = r.svc.Similar(t.Context(), r.course, a)
	require.Len(t, got, 1)
	require.Equal(t, d, got[0].ID)
	// thread lớp khác → 404
	_, err = r.svc.Similar(t.Context(), uuid.New(), a)
	require.Equal(t, 404, status(err))
}

// TestSimilarExcludesHidden — AC15: thread đã xoá mềm và thread có bài AI bị ẩn (hidden_at) không xuất hiện trong "tương tự".
func TestSimilarExcludesHidden(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	a := r.mustCreate(r.sv, "A", clean)
	b := r.mustCreate(r.sv, "B", clean)
	c := r.mustCreate(r.sv, "C", clean)
	for _, id := range []uuid.UUID{a, b, c} {
		_, err := r.pool.Exec(t.Context(), `update forum_threads set embedding=array_fill(0.5::real, array[1536])::vector where id=$1`, id)
		require.NoError(t, err)
	}
	got, err := r.svc.Similar(t.Context(), r.course, a)
	require.NoError(t, err)
	require.Len(t, got, 2)
	_, err = r.pool.Exec(t.Context(), `update forum_threads set deleted_at=now() where id=$1`, b)
	require.NoError(t, err)
	r.rag.hits = []rag.Hit{hit(0.9)}
	require.NoError(t, r.answer(c))
	_, err = r.pool.Exec(t.Context(), `update forum_posts set hidden_at=now(), hidden_reason='MODERATION' where thread_id=$1 and kind='AI'`, c)
	require.NoError(t, err)
	got, err = r.svc.Similar(t.Context(), r.course, a)
	require.NoError(t, err)
	require.Empty(t, got)
}

var (
	_ = context.Background
	_ = auth.RoleStudent
)

// TestSwitchedEvent — US-P3-06 AC5: `from-draft` ghi SWITCHED theo từng loại (số Finding tính lại phía máy chủ), không ghi nội dung.
func TestSwitchedEvent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.svc.RecordSwitched(t.Context(), r.sv.UserID, r.course, "Hỏi", "MSSV "+svCode+" và mail "+svEmail)
	require.Equal(t, 1, r.count(`select count(*) from pii_events where user_id=$1 and action='SWITCHED' and pii_type='MSSV' and channel='PUBLIC'`, r.sv.UserID))
	require.Equal(t, 1, r.count(`select count(*) from pii_events where user_id=$1 and action='SWITCHED' and pii_type='EMAIL'`, r.sv.UserID))
}

// TestStudentNeverSeesConfidence — US-P3-07 AC3: mọi đường đọc / ghi của Threads cho sinh viên (danh sách, chi tiết, tương tự, bình luận) không có khoá confidence / retrieval_score / groundedness.
func TestStudentNeverSeesConfidence(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	id := r.answered()
	l, err := r.svc.List(t.Context(), r.other, r.course, thread.Filter{}, pg(10))
	require.NoError(t, err)
	v, err := r.svc.Get(t.Context(), r.other, r.course, id, pg(30))
	require.NoError(t, err)
	sim, err := r.svc.Similar(t.Context(), r.course, id)
	require.NoError(t, err)
	c, err := r.svc.Comment(t.Context(), r.other, r.course, id, thread.CommentIn{Body: "Mình cũng hỏi"})
	require.NoError(t, err)
	raw, _ := json.Marshal([]any{l, v, sim, c})
	for _, k := range []string{"confidence", "retrieval_score", "groundedness"} {
		require.NotContains(t, string(raw), `"`+k+`"`)
	}
}

// TestStaffConfidenceThreadsOnly — US-P3-07 AC5: TA và TEACHER thấy "Độ tin cậy" của bài AI ở Threads, tính bằng công thức SRS 4.8 (ở đây retr 1,000 và mọi nhận định có căn cứ → 1.000;
// khi truy xuất chỉ vừa trên sàn thì thấp hơn); sinh viên không thấy.
func TestStaffConfidenceThreadsOnly(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.rag.hits = []rag.Hit{hit(0.45)} // retr 0,5; câu trả lời có căn cứ → 0,6·0,5 + 0,4·1 = 0,700
	id := r.mustCreate(r.sv, "Thi lại", clean)
	require.NoError(t, r.answer(id))
	for _, a := range []thread.Actor{r.ta, r.teacher} {
		v, err := r.svc.Get(t.Context(), a, r.course, id, pg(30))
		require.NoError(t, err)
		require.Equal(t, "0.700", *v.Posts[0].Confidence)
	}
	v, err := r.svc.Get(t.Context(), r.sv, r.course, id, pg(30))
	require.NoError(t, err)
	require.Nil(t, v.Posts[0].Confidence)
}
