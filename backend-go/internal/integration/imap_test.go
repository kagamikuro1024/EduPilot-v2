package integration_test

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/httpapi"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/mailin"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

func code(err error) string {
	if ae, ok := err.(*apierr.Error); ok { //nolint:errorlint // apierr.Error trả thẳng
		return ae.Code
	}
	return ""
}

// ---- AC1: cấu hình ----

// TestImapConfigSecretWriteOnly — AC1: không đường nào trả mật khẩu hay đuôi mật khẩu (GET, PUT, last_error, audit).
func TestImapConfigSecretWriteOnly(t *testing.T) {
	r := newImap(t)
	v := r.config(r.course, "CANARY-IMAP-PASS-9917", 0)
	got, err := r.svc.Get(t.Context(), r.course, mailin.KindIMAP)
	require.NoError(t, err)
	for _, x := range []mailin.View{v, got} {
		raw, _ := json.Marshal(x)
		require.NotContains(t, string(raw), "CANARY")
		require.NotContains(t, string(raw), "9917")
		require.True(t, x.Configured)
	}
	var pub, state, lastErr, audit string
	r.one(&pub, `select config_public::text from integration_configs where course_id = $1`, r.course)
	r.one(&state, `select state::text from integration_configs where course_id = $1`, r.course)
	r.one(&audit, `select coalesce(string_agg(before::text || after::text, ''), '') from audit_log where course_id = $1`, r.course)
	r.one(&lastErr, `select coalesce(last_error, '') from integration_configs where course_id = $1`, r.course)
	for _, s := range []string{pub, state, lastErr, audit} {
		require.NotContains(t, s, "CANARY")
	}
}

// TestImapConfigEncrypted — AC1: mật khẩu ở config_enc là bản mã AES-GCM (không có chữ rõ), giải mã lại đúng; để trống khi lưu lại = giữ cũ.
func TestImapConfigEncrypted(t *testing.T) {
	r := newImap(t)
	v := r.config(r.course, "CANARY-IMAP-PASS-9917", 0)
	var enc []byte
	r.one(&enc, `select config_enc from integration_configs where course_id = $1`, r.course)
	require.NotEmpty(t, enc)
	require.NotContains(t, string(enc), "CANARY")
	plain, err := r.cipher.Decrypt(enc, []byte(r.course.String()+":IMAP"))
	require.NoError(t, err)
	require.Contains(t, string(plain), "CANARY-IMAP-PASS-9917")
	_, err = r.cipher.Decrypt(enc, []byte(uuid.NewString()+":IMAP"))
	require.Error(t, err, "AAD gắn với lớp: bản mã không dùng được ở lớp khác")
	r.config(r.course, "", v.Version) // trống = giữ cũ
	var enc2 []byte
	r.one(&enc2, `select config_enc from integration_configs where course_id = $1`, r.course)
	require.Equal(t, enc, enc2)
}

// TestImapConfigVersionConflict — AC1: sai version → 409 VERSION_CONFLICT; NONE bị từ chối ở production; lần đầu thiếu mật khẩu → 422.
func TestImapConfigVersionConflict(t *testing.T) {
	r := newImap(t)
	v := r.config(r.course, imapPass, 0)
	_, err := r.svc.Put(t.Context(), r.admin, r.course, mailin.PutIn{Host: "h", Port: 993, Security: mailin.SecTLS, Username: "u", Version: v.Version + 5})
	require.Equal(t, apierr.VersionConflict, code(err))
	prod := *r.svc
	prod.Production = true
	_, err = prod.Put(t.Context(), r.admin, r.course, mailin.PutIn{Host: "h", Port: 993, Security: mailin.SecNone, Username: "u", Version: v.Version})
	require.Equal(t, apierr.ValidationFailed, code(err))
	c2, _ := r.newCourse("BT09")
	_, err = r.svc.Put(t.Context(), r.admin, c2, mailin.PutIn{Host: "h", Port: 993, Security: mailin.SecTLS, Username: "u"})
	require.Equal(t, apierr.ValidationFailed, code(err))
}

// TestImapConfigAuditNoSecret — AC1: audit_log ghi lần lưu (phần không bí mật) và không chứa mật khẩu.
func TestImapConfigAuditNoSecret(t *testing.T) {
	r := newImap(t)
	r.config(r.course, "CANARY-AUDIT-5521", 0)
	var n int
	r.one(&n, `select count(*)::int from audit_log where entity = 'integration_configs' and course_id = $1 and action = 'imap.put'`, r.course)
	require.Equal(t, 1, n)
	var blob string
	r.one(&blob, `select (to_jsonb(a))::text from audit_log a where entity = 'integration_configs' and course_id = $1`, r.course)
	require.NotContains(t, blob, "CANARY")
	require.Contains(t, blob, `"password_changed": true`)
}

// ---- AC2: Test kết nối ----

func (r *imapRig) test() mailin.TestResult {
	r.t.Helper()
	out, err := r.svc.Test(r.t.Context(), r.course)
	require.NoError(r.t, err)
	return out
}

func TestImapTestOK(t *testing.T) {
	r := newImap(t)
	r.ready()
	require.True(t, r.test().OK)
	var ok bool
	r.one(&ok, `select last_ok_at is not null and last_error is null from integration_configs where course_id = $1`, r.course)
	require.True(t, ok)
}

func TestImapTestAuthFail(t *testing.T) {
	r := newImap(t)
	r.config(r.course, "sai-mat-khau", 0)
	out := r.test()
	require.False(t, out.OK)
	require.Equal(t, mailin.KindAuth, *out.Kind)
	var e string
	r.one(&e, `select last_error from integration_configs where course_id = $1`, r.course)
	require.Equal(t, "AUTH", e)
}

func TestImapTestUnreachable(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.stop()
	out := r.test()
	require.False(t, out.OK)
	require.Equal(t, mailin.KindUnreachable, *out.Kind)
}

func TestImapTestMailboxMissing(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.exec(`update integration_configs set config_public = jsonb_set(config_public, '{mailbox}', '"KHONG-CO"') where course_id = $1`, r.course)
	out := r.test()
	require.False(t, out.OK)
	require.Equal(t, mailin.KindMailbox, *out.Kind)
}

// TestImapTestTimeout — AC2: máy chủ nhận kết nối nhưng không bao giờ chào → TIMEOUT trong hạn, không treo.
func TestImapTestTimeout(t *testing.T) {
	r := newImap(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { _ = c.Close() })
		}
	}()
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	p, _ := strconv.Atoi(port)
	_, err = r.svc.Put(t.Context(), r.admin, r.course, mailin.PutIn{Host: host, Port: p, Security: mailin.SecNone, Username: imapUser, Password: imapPass, Enabled: true})
	require.NoError(t, err)
	start := time.Now()
	out := r.test()
	require.False(t, out.OK)
	require.Equal(t, mailin.KindTimeout, *out.Kind)
	require.Less(t, time.Since(start), 5*time.Second)
}

// ---- AC3: poll ----

// TestImapPollNewOnly — AC3: chỉ thư UID > last_uid; lượt sau không đọc lại thư cũ.
func TestImapPollNewOnly(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.deliver(r.khaMail(), r.now.Add(-time.Hour))
	res := r.mustSync()
	require.Equal(t, mailin.Result{Received: 1, Created: 1, Matched: 1}, res)
	require.Zero(t, r.mustSync().Received, "không có thư mới")
	r.deliver(mailSpec{subject: "chào", atts: nil}, r.now.Add(-time.Minute))
	res = r.mustSync()
	require.Equal(t, 1, res.Received)
	require.Equal(t, 1, res.Ignored)
	var uid int
	r.one(&uid, `select (state->>'last_uid')::int from integration_configs where course_id = $1`, r.course)
	require.Equal(t, 2, uid)
}

// TestImapPollIdempotentRestart — AC3: đọc cùng thư hai lần (khởi động lại làm mất last_uid) → một bài nộp, một dòng mail_inbound.
func TestImapPollIdempotentRestart(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.deliver(r.khaMail(), r.now.Add(-time.Hour))
	r.mustSync()
	r.exec(`update integration_configs set state = state || '{"last_uid":0}'::jsonb where course_id = $1`, r.course) // "chết" trước khi ghi vị trí
	res := r.mustSync()
	require.Zero(t, res.Created)
	require.Equal(t, 1, r.subs())
	var n int
	r.one(&n, `select count(*)::int from mail_inbound where course_id = $1`, r.course)
	require.Equal(t, 1, n)
	var blobs int
	infos, err := r.blob.List(t.Context(), "courses/"+r.course.String()+"/submissions/")
	require.NoError(t, err)
	blobs = len(infos)
	require.Equal(t, 1, blobs, "không để lại tệp thừa của lần đọc lại")
}

// TestImapUidValidityChange — AC3: đổi UIDVALIDITY (hộp thư tạo lại) → quét lại hộp thư nhưng không nhập lại thư đã có.
func TestImapUidValidityChange(t *testing.T) {
	r := newImap(t)
	r.ready()
	m := r.khaMail()
	m.msgID = "co-dinh-1@truong.edu"
	r.deliver(m, r.now.Add(-time.Hour))
	r.mustSync()
	require.NoError(t, r.box.Delete("INBOX"))
	require.NoError(t, r.box.Create("INBOX", nil))
	r.deliver(m, r.now.Add(-time.Hour)) // cùng thư (cùng INTERNALDATE) nhưng UIDVALIDITY mới
	res := r.mustSync()
	require.Equal(t, 1, res.Received, "quét lại từ đầu")
	require.Equal(t, 0, res.Created, "thư đã nhận (cùng Message-ID) → không nhập lại")
	require.Equal(t, 1, r.subs())
	var n int
	r.one(&n, `select count(*)::int from mail_inbound where course_id = $1`, r.course)
	require.Equal(t, 1, n)
	var uv int
	r.one(&uv, `select (state->>'uidvalidity')::int from integration_configs where course_id = $1`, r.course)
	require.Equal(t, 2, uv, "đã ghi nhận UIDVALIDITY mới")
}

// TestImapAutoReplyIgnored — AC3: Auto-Submitted ≠ no, hoặc thư từ chính địa chỉ gửi của hệ thống → IGNORED AUTO_REPLY.
func TestImapAutoReplyIgnored(t *testing.T) {
	r := newImap(t)
	r.ready()
	a := r.khaMail()
	a.autoSubmitted = "auto-replied"
	r.deliver(a, r.now.Add(-3*time.Minute))
	b := r.khaMail()
	b.from = systemFrom
	r.deliver(b, r.now.Add(-2*time.Minute))
	c := r.khaMail()
	c.autoSubmitted = "no"
	r.deliver(c, r.now.Add(-time.Minute))
	r.mustSync()
	require.Equal(t, [][2]string{{"IGNORED", "AUTO_REPLY"}, {"IGNORED", "AUTO_REPLY"}, {"ACCEPTED", ""}}, r.inbound())
	require.Equal(t, 1, r.subs())
}

// TestImapSinglePoller — AC3: lớp đang có poller khác giữ khoá → lượt này bỏ (ErrBusy), không nhập đôi; khoá nhả thì chạy.
func TestImapSinglePoller(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.deliver(r.khaMail(), r.now.Add(-time.Hour))
	key := "ep:lock:imap:" + r.course.String()
	require.NoError(t, r.rdb.Set(t.Context(), key, "poller-khac", time.Minute).Err())
	_, err := r.sync()
	require.ErrorIs(t, err, mailin.ErrBusy)
	require.Zero(t, r.subs())
	require.NoError(t, r.rdb.Del(t.Context(), key).Err())
	require.Equal(t, 1, r.mustSync().Created)
	exists, err := r.rdb.Exists(t.Context(), key).Result()
	require.NoError(t, err)
	require.Zero(t, exists, "khoá nhả sau khi xong")
}

// ---- AC4: đính kèm và khớp ----

func TestImapOneAttachment(t *testing.T) {
	r := newImap(t)
	r.ready()
	when := r.now.Add(-26 * time.Hour).Truncate(time.Second)
	r.deliver(r.khaMail(), when)
	r.mustSync()
	var src, ref, status string
	var sub, ts time.Time
	var size int
	r.one(&src, `select source from submissions where course_id = $1`, r.course)
	r.one(&ref, `select external_ref from submissions where course_id = $1`, r.course)
	r.one(&status, `select match_status from submissions where course_id = $1`, r.course)
	r.one(&sub, `select submitted_at from submissions where course_id = $1`, r.course)
	r.one(&ts, `select external_ts from submissions where course_id = $1`, r.course)
	r.one(&size, `select file_size::int from submissions where course_id = $1`, r.course)
	require.Equal(t, "IMAP", src)
	require.Equal(t, "MATCHED", status)
	require.Regexp(t, `^\d+:1$`, ref)
	require.True(t, when.Equal(sub) && when.Equal(ts), "submitted_at = INTERNALDATE của máy chủ thư")
	require.Equal(t, len(s2()), size)
	var key string
	r.one(&key, `select blob_key from submissions where course_id = $1`, r.course)
	rc, err := r.blob.Get(t.Context(), key)
	require.NoError(t, err)
	_ = rc.Close()
}

func TestImapNoAttachment(t *testing.T) {
	r := newImap(t)
	r.ready()
	m := r.khaMail()
	m.atts = nil
	r.deliver(m, r.now)
	r.mustSync()
	require.Equal(t, [][2]string{{"IGNORED", "NO_ATTACHMENT"}}, r.inbound())
	require.Zero(t, r.subs())
}

func TestImapTooManyAttachments(t *testing.T) {
	r := newImap(t)
	r.ready()
	m := r.khaMail()
	m.atts = append(m.atts, att{name: "bai2.md", data: s2()})
	r.deliver(m, r.now)
	r.mustSync()
	require.Equal(t, [][2]string{{"IGNORED", "TOO_MANY_ATTACHMENTS"}}, r.inbound())
	require.Zero(t, r.subs(), "không đoán")
}

func TestImapAttachmentRejected(t *testing.T) {
	r := newImap(t)
	r.ready()
	bad := r.khaMail()
	bad.atts = []att{{name: "virus.exe", data: []byte("MZ....")}}
	r.deliver(bad, r.now.Add(-3*time.Minute))
	fake := r.khaMail()
	fake.atts = []att{{name: "gia.pdf", data: []byte("không phải PDF")}} // sai nội dung so với đuôi
	r.deliver(fake, r.now.Add(-2*time.Minute))
	r.sy.MaxBytes = 100
	big := r.khaMail()
	big.atts = []att{{name: "to.md", data: []byte(strings.Repeat("x", 200))}}
	r.deliver(big, r.now.Add(-time.Minute))
	r.mustSync()
	require.Equal(t, [][2]string{{"IGNORED", "ATTACHMENT_REJECTED"}, {"IGNORED", "ATTACHMENT_REJECTED"}, {"IGNORED", "ATTACHMENT_REJECTED"}}, r.inbound())
	require.Zero(t, r.subs())
}

// TestImapMatchBySubject — AC4: khớp bằng email đã xác minh hoặc MSSV trong tiêu đề `[BT03] 20229002` khi người gửi được máy chủ thư xác thực.
func TestImapMatchBySubject(t *testing.T) {
	r := newImap(t)
	r.ready()
	byCode := goodMail()
	byCode.from = "nguoi.la@truong.edu"
	byCode.subject = "[BT03] 20229002"
	byCode.auth = trustedID + "; dmarc=pass header.from=truong.edu"
	r.deliver(byCode, r.now.Add(-2*time.Minute))
	byMail := r.khaMail()
	byMail.subject = "[BT03] bài làm"
	r.deliver(byMail, r.now.Add(-time.Minute))
	res := r.mustSync()
	require.Equal(t, 2, res.Matched)
	var gotB, gotKha uuid.UUID
	r.one(&gotB, `select student_id from submissions where course_id = $1 and match_method = 'CODE'`, r.course)
	r.one(&gotKha, `select student_id from submissions where course_id = $1 and match_method = 'EMAIL'`, r.course)
	require.Equal(t, r.b, gotB)
	require.Equal(t, r.kha, gotKha)
}

// TestImapUnmatchedGoesToTab — AC4: thư không khớp được sinh viên / bài → bài nộp UNMATCHED (tab `Chưa khớp`); thư IGNORED có trong danh sách thư không xử lý được.
func TestImapUnmatchedGoesToTab(t *testing.T) {
	r := newImap(t)
	r.ready()
	m := r.khaMail()
	m.from, m.subject = "la@truong.edu", "nộp bài"
	r.deliver(m, r.now.Add(-2*time.Minute))
	n := r.khaMail()
	n.atts = nil
	r.deliver(n, r.now.Add(-time.Minute))
	res := r.mustSync()
	require.Equal(t, 1, res.Unmatched)
	var reason string
	r.one(&reason, `select unmatched_reason from submissions where course_id = $1 and source = 'IMAP'`, r.course)
	require.Equal(t, "NO_ASSIGNMENT_CODE", reason)
	page, err := r.svc.ListInbound(t.Context(), r.course, "IGNORED", httpx.PageParams{Limit: 10})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "NO_ATTACHMENT", *page.Items[0].Reason)
}

// ---- AC5: mail → khớp → chấm nháp ----

// TestImapToDraftGrade — AC5: thư S2 từ email đã xác minh → MATCHED source=IMAP → chấm → kết quả nháp ĐÚNG expected_grading.json (provider giả); sinh viên không thấy nháp.
func TestImapToDraftGrade(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.deliver(r.khaMail(), r.now.Add(-time.Hour))
	require.Equal(t, 1, r.mustSync().Matched)
	var sub, res uuid.UUID
	r.one(&sub, `select id from submissions where course_id = $1 and source = 'IMAP' and match_status = 'MATCHED'`, r.course)
	r.gradeLikeS2(sub, &res)
	var status, runState string
	r.one(&status, `select status from grading_results where id = $1`, res)
	r.one(&runState, `select run_state from grading_results where id = $1`, res)
	require.Equal(t, "DRAFT", status)
	require.Equal(t, "DONE", runState)
	var want []struct {
		ID       string `json:"id"`
		Expected struct {
			Scores map[string]string `json:"criteria_scores"`
			Total  string            `json:"total"`
		} `json:"expected"`
	}
	require.NoError(t, json.Unmarshal(fixture("expected_grading.json"), &want))
	var crit []struct {
		ID    string  `json:"id"`
		Score *string `json:"score"`
	}
	var raw []byte
	r.one(&raw, `select criteria from grading_results where id = $1`, res)
	require.NoError(t, json.Unmarshal(raw, &crit))
	var total string
	r.one(&total, `select total::text from grading_results where id = $1`, res)
	for _, w := range want {
		if w.ID != "S2" {
			continue
		}
		require.Len(t, crit, len(w.Expected.Scores))
		for _, c := range crit {
			require.NotNil(t, c.Score)
			require.True(t, decimal.RequireFromString(*c.Score).Equal(decimal.RequireFromString(w.Expected.Scores[c.ID])), "%s: %s ≠ %s", c.ID, *c.Score, w.Expected.Scores[c.ID])
		}
		require.True(t, decimal.RequireFromString(total).Equal(decimal.RequireFromString(w.Expected.Total)), "tổng %s ≠ %s", total, w.Expected.Total)
	}
	var visible int
	r.one(&visible, `select count(*)::int from grading_results where id = $1 and status = 'PUBLISHED'`, res)
	require.Zero(t, visible, "sinh viên không thấy nháp: chưa công bố")
}

// ---- AC6: nhánh lỗi ----

func (r *imapRig) failState() (fails int, next time.Time) {
	var raw []byte
	r.one(&raw, `select state from integration_configs where course_id = $1`, r.course)
	var st mailin.IMAPState
	require.NoError(r.t, json.Unmarshal(raw, &st))
	if st.NextAt != nil {
		next = *st.NextAt
	}
	return st.FailCount, next
}

// TestImapBackoff — AC6: lỗi kết nối → lùi 60 s → 120 s → 240 s … tối đa 15 phút; không vòng lặp nóng (chưa tới hạn thì Tick không gọi).
func TestImapBackoff(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.stop()
	want := []time.Duration{60 * time.Second, 120 * time.Second, 240 * time.Second, 480 * time.Second, 900 * time.Second, 900 * time.Second}
	for i, w := range want {
		_, err := r.sync()
		require.Error(t, err)
		fails, next := r.failState()
		require.Equal(t, i+1, fails)
		require.Equal(t, w, next.Sub(r.now), "lượt %d", i+1)
	}
	var due int
	r.one(&due, `select count(*)::int from integration_configs where course_id = $1 and (state->>'next_at')::timestamptz <= $2`, r.course, r.now.Add(899*time.Second))
	require.Zero(t, due)
	var lastErr, since string
	r.one(&lastErr, `select last_error from integration_configs where course_id = $1`, r.course)
	r.one(&since, `select coalesce(state->>'error_since', '') from integration_configs where course_id = $1`, r.course)
	require.Equal(t, "UNREACHABLE", lastErr)
	require.NotEmpty(t, since)
}

// TestImapNoMailLostOnOutage — AC6: máy chủ thư chết rồi hồi phục → thư vẫn nhập (không mất), last_error xoá, hạn poll về chu kỳ.
func TestImapNoMailLostOnOutage(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.deliver(r.khaMail(), r.now.Add(-time.Hour))
	r.stop()
	_, err := r.sync()
	require.Error(t, err)
	require.Zero(t, r.subs())
	r.restart()
	require.Equal(t, 1, r.mustSync().Created)
	fails, _ := r.failState()
	require.Zero(t, fails)
	var lastErr string
	r.one(&lastErr, `select coalesce(last_error, '') from integration_configs where course_id = $1`, r.course)
	require.Empty(t, lastErr)
}

// TestImapAuthFailPausesPoll — AC6: sai mật khẩu → tạm dừng poll ("Cần nhập lại mật khẩu"), không thử dồn dập; nhập lại mật khẩu thì poll tiếp.
func TestImapAuthFailPausesPoll(t *testing.T) {
	r := newImap(t)
	v := r.config(r.course, "sai", 0)
	_, err := r.sync()
	var ce *mailin.ConnError
	require.ErrorAs(t, err, &ce)
	require.Equal(t, mailin.KindAuth, ce.Kind)
	conns := r.ln.count()
	r.now = r.now.Add(24 * time.Hour)
	r.sy.Tick(t.Context()) // đã tạm dừng: không kết nối
	require.Equal(t, conns, r.ln.count(), "không thử lại khi chờ nhập lại mật khẩu")
	view, err := r.svc.Get(t.Context(), r.course, mailin.KindIMAP)
	require.NoError(t, err)
	require.Equal(t, "PAUSED", view.Status)
	require.True(t, view.Enabled)
	r.config(r.course, imapPass, v.Version) // nhập lại mật khẩu → mở poll
	r.deliver(r.khaMail(), r.now.Add(-time.Hour))
	r.sy.Tick(t.Context())
	require.Equal(t, 1, r.subs())
}

// TestImapBadMimeSkipped — AC6: thư MIME hỏng → FAILED PARSE_ERROR, bỏ qua, thư sau vẫn được xử lý (không kẹt hàng).
func TestImapBadMimeSkipped(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.deliver(mailSpec{raw: "From: x@truong.edu\r\nSubject: [BT03] hỏng\r\nContent-Type: multipart/mixed; boundary=zz\r\n\r\n--zz\r\nContent-Type: text/plain\r\nContent-Transfer-Encoding: base64\r\n\r\n!!!không-phải-base64!!!\r\n--zz\r\nContent-Disposition: attachment; filename=\"a.md\"\r\nContent-Transfer-Encoding: base64\r\n\r\n%%%%\r\n"}, r.now.Add(-2*time.Minute))
	r.deliver(r.khaMail(), r.now.Add(-time.Minute))
	res := r.mustSync()
	require.Equal(t, 1, res.Created)
	in := r.inbound()
	require.Len(t, in, 2)
	require.Equal(t, [2]string{"FAILED", "PARSE_ERROR"}, in[0])
	require.Equal(t, "ACCEPTED", in[1][0])
}

// ---- AC7: Đồng bộ ngay ----

func (r *imapRig) runJob(id uuid.UUID) {
	r.t.Helper()
	runner := jobs.NewRunner(r.pool, nil, clock.Real{}, slog.New(slog.DiscardHandler))
	r.sy.Register(runner)
	var payload []byte
	r.one(&payload, `select payload from outbox where topic = $1 and payload->>'job_id' = $2`, jobs.TopicEnqueue, id.String())
	require.NoError(r.t, runner.HandleMessage(r.t.Context(), outbox.Message{ID: uuid.New(), Topic: jobs.TopicEnqueue, Payload: payload}))
}

// TestImapSyncNowJob — AC7: 202 + job_id; chạy xong có kết quả {received, created, matched, unmatched, ignored} và tiến độ 100.
func TestImapSyncNowJob(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.deliver(r.khaMail(), r.now.Add(-2*time.Minute))
	n := r.khaMail()
	n.atts = nil
	r.deliver(n, r.now.Add(-time.Minute))
	job, err := r.svc.SyncNow(t.Context(), r.ta, r.course)
	require.NoError(t, err)
	r.runJob(job.JobID)
	var status string
	var progress int
	var result string
	r.one(&status, `select status::text from jobs where id = $1`, job.JobID)
	r.one(&progress, `select progress::int from jobs where id = $1`, job.JobID)
	r.one(&result, `select result::text from jobs where id = $1`, job.JobID)
	require.Equal(t, "SUCCEEDED", status)
	require.Equal(t, 100, progress)
	var got mailin.Result
	require.NoError(t, json.Unmarshal([]byte(result), &got))
	require.Equal(t, mailin.Result{Received: 2, Created: 1, Matched: 1, Ignored: 1}, got)
	exists, err := r.rdb.Exists(t.Context(), "ep:imapsync:"+r.course.String()+":job").Result()
	require.NoError(t, err)
	require.Zero(t, exists, "việc xong: bỏ dấu đang chạy")
}

// TestImapSyncNowRateLimit — AC7: ≤ 1 lần / 30 giây / lớp (429 + retry_after) khi việc trước đã xong.
func TestImapSyncNowRateLimit(t *testing.T) {
	r := newImap(t)
	r.ready()
	job, err := r.svc.SyncNow(t.Context(), r.ta, r.course)
	require.NoError(t, err)
	r.runJob(job.JobID)
	_, err = r.svc.SyncNow(t.Context(), r.ta, r.course)
	ae, ok := err.(*apierr.Error) //nolint:errorlint // trả thẳng
	require.True(t, ok)
	require.Equal(t, http.StatusTooManyRequests, ae.Status)
	require.Positive(t, ae.RetryAfter)
	require.LessOrEqual(t, ae.RetryAfter, 30)
}

// TestImapSyncNowNoOverlap — AC7: đang có một lần chạy → trả lại job_id đó (không chạy chồng, không xếp việc mới).
func TestImapSyncNowNoOverlap(t *testing.T) {
	r := newImap(t)
	r.ready()
	a, err := r.svc.SyncNow(t.Context(), r.ta, r.course)
	require.NoError(t, err)
	b, err := r.svc.SyncNow(t.Context(), r.teacher, r.course)
	require.NoError(t, err)
	require.Equal(t, a.JobID, b.JobID)
	var n int
	r.one(&n, `select count(*)::int from jobs where kind = $1 and id = $2`, mailin.JobKind, a.JobID)
	require.Equal(t, 1, n)
}

// ---- AC8: phân quyền / cô lập lớp ----

// TestImapCourseIsolation — AC8: thư của lớp này không bao giờ tạo bài nộp ở lớp khác; cấu hình theo course_id.
func TestImapCourseIsolation(t *testing.T) {
	r := newImap(t)
	r.ready()
	c2, asg2 := r.newCourse("BT03") // cùng mã bài, lớp khác
	r.deliver(r.khaMail(), r.now.Add(-time.Hour))
	r.mustSync()
	var other int
	r.one(&other, `select count(*)::int from submissions where course_id = $1 or assignment_id = $2`, c2, asg2)
	require.Zero(t, other)
	_, err := r.svc.Get(t.Context(), c2, mailin.KindIMAP)
	require.NoError(t, err)
	v, _ := r.svc.Get(t.Context(), c2, mailin.KindIMAP)
	require.False(t, v.Configured)
	_, err = r.sy.SyncCourse(t.Context(), c2, nil)
	require.Equal(t, apierr.NotFound, code(err), "lớp khác chưa cấu hình IMAP: không đọc được hộp thư của lớp này")
}

// ---- AC10: xác thực người gửi ----

// TestImapSpoofedFromNotMatched — AC10: thư giả `From: sv.kha@… [BT03]` không có xác thực → UNMATCHED SENDER_UNVERIFIED, KHÔNG thay bài thật của sinh viên đó.
func TestImapSpoofedFromNotMatched(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.deliver(r.khaMail(), r.now.Add(-2*time.Hour)) // bài thật
	r.mustSync()
	var real uuid.UUID
	r.one(&real, `select id from submissions where course_id = $1 and match_status = 'MATCHED'`, r.course)
	fake := r.khaMail()
	fake.auth = ""
	fake.atts = []att{{name: "gia.md", data: []byte("# bài giả mạo\nnội dung khác")}}
	r.deliver(fake, r.now.Add(-time.Hour))
	res := r.mustSync()
	require.Equal(t, 1, res.Unmatched)
	var reason string
	r.one(&reason, `select unmatched_reason from submissions where course_id = $1 and match_status = 'UNMATCHED'`, r.course)
	require.Equal(t, "SENDER_UNVERIFIED", reason)
	var n, cur int
	r.one(&n, `select count(*)::int from submissions where course_id = $1 and match_status = 'MATCHED'`, r.course)
	r.one(&cur, `select version::int from submissions where id = $1`, real)
	require.Equal(t, 1, n)
	require.Equal(t, 1, cur)
}

// TestImapAuthResultsTrustedOnly — AC10: authserv-id lạ, dmarc ≠ pass, header.from khác domain From → đều không xác thực; đúng cả ba mới khớp tự động.
func TestImapAuthResultsTrustedOnly(t *testing.T) {
	cases := map[string]bool{
		trustedID + "; dmarc=pass header.from=truong.edu":                      true,
		"mx.khac.vn; dmarc=pass header.from=truong.edu":                        false,
		trustedID + "; dmarc=fail header.from=truong.edu":                      false,
		trustedID + "; dmarc=none header.from=truong.edu":                      false,
		trustedID + "; dmarc=pass header.from=attacker.com":                    false,
		trustedID + "; dkim=pass header.d=truong.edu":                          false,
		trustedID + "; spf=pass; dmarc=pass (p=reject) header.from=truong.edu": true,
	}
	for hdr, want := range cases {
		require.Equal(t, want, mailin.SenderVerified([]string{hdr}, "sv.kha@truong.edu", trustedID), hdr)
	}
	r := newImap(t)
	r.ready()
	m := r.khaMail()
	m.auth = "mx.khac.vn; dmarc=pass header.from=truong.edu"
	r.deliver(m, r.now)
	res := r.mustSync()
	require.Equal(t, 1, res.Unmatched)
}

// TestImapAuthResultsFirstHeader — AC10: chỉ header TRÊN CÙNG (của máy chủ thư) được tin; header người gửi tự thêm nằm dưới bị bỏ qua — cả hai chiều.
func TestImapAuthResultsFirstHeader(t *testing.T) {
	r := newImap(t)
	r.ready()
	forged := "Authentication-Results: " + trustedID + "; dmarc=pass header.from=truong.edu"
	m := r.khaMail()
	m.auth = trustedID + "; dmarc=fail header.from=truong.edu" // máy chủ thư của trường: KHÔNG đạt; header giả đạt nằm dưới
	m.below = []string{forged}
	r.deliver(m, r.now.Add(-2*time.Minute))
	good := r.khaMail()
	good.below = []string{"Authentication-Results: " + trustedID + "; dmarc=fail header.from=truong.edu"} // dưới đáy: bỏ qua, header trên cùng vẫn đạt
	r.deliver(good, r.now.Add(-time.Minute))
	res := r.mustSync()
	require.Equal(t, 1, res.Matched, "chỉ thư có header trên cùng hợp lệ")
	require.Equal(t, 1, res.Unmatched)
	var reason string
	r.one(&reason, `select unmatched_reason from submissions where course_id = $1 and match_status = 'UNMATCHED'`, r.course)
	require.Equal(t, "SENDER_UNVERIFIED", reason)
}

// TestImapNoTrustedIdUnmatched — AC10: IMAP_TRUSTED_AUTHSERV_ID rỗng (mặc định production) → mọi thư UNMATCHED SENDER_UNVERIFIED.
func TestImapNoTrustedIdUnmatched(t *testing.T) {
	r := newImap(t)
	r.ready()
	r.sy.TrustedAuthservID = ""
	r.deliver(r.khaMail(), r.now)
	res := r.mustSync()
	require.Equal(t, 0, res.Matched)
	require.Equal(t, 1, res.Unmatched)
	var reason string
	r.one(&reason, `select unmatched_reason from submissions where course_id = $1`, r.course)
	require.Equal(t, "SENDER_UNVERIFIED", reason)
}

// TestImapReceiptBellToMatchedStudent — AC10: mỗi bài nộp qua thư được khớp gửi chuông cho CHÍNH sinh viên được gán ("Đã nhận bài nộp qua thư lúc HH:mm dd/MM"); thư không khớp không gửi cho ai.
func TestImapReceiptBellToMatchedStudent(t *testing.T) {
	r := newImap(t)
	r.ready()
	when := time.Date(2026, 11, 2, 9, 12, 0, 0, time.FixedZone("ICT", 7*3600))
	r.deliver(r.khaMail(), when)
	un := r.khaMail()
	un.auth = ""
	un.atts = []att{{name: "khac.md", data: []byte("# khác")}}
	r.deliver(un, when.Add(time.Minute))
	r.mustSync()
	var title string
	r.one(&title, `select title from notifications where user_id = $1 and type = 'SUBMISSION_MAIL_RECEIVED'`, r.kha)
	require.Equal(t, "Đã nhận bài nộp qua thư lúc 09:12 02/11", title)
	var n int
	r.one(&n, `select count(*)::int from notifications where type = 'SUBMISSION_MAIL_RECEIVED' and course_id = $1`, r.course)
	require.Equal(t, 1, n)
	require.Zero(t, func() int {
		var x int
		r.one(&x, `select count(*)::int from notifications where user_id = $1 and type = 'SUBMISSION_MAIL_RECEIVED'`, r.b)
		return x
	}())
}

// TestIntegrationsRBACMatrix — AC8: GET: ADMIN, TEACHER của lớp 200 (che bí mật), TA / STUDENT / giảng viên lớp khác 403; PUT / test: chỉ ADMIN (TEACHER 403); sync-now: TA, TEACHER 202, ADMIN 403.
func TestIntegrationsRBACMatrix(t *testing.T) {
	r := newRigDeps(t, func(d *httpapi.Deps) { d.Jobs = jobs.NewService(d.DB) })
	mk := func(role store.UserRole, c uuid.UUID, enrollAs string) session {
		u := r.addUser(uniq(strings.ToLower(string(role))), role, store.UserStatusACTIVE)
		if enrollAs != "" {
			r.enroll(c, u.ID, enrollAs, "ACTIVE")
		}
		return r.mustLogin(u.Email)
	}
	owner := r.addUser(uniq("owner"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	course := r.course(owner.ID, "RB"+uuid.NewString()[:6])
	other := r.course(owner.ID, "RC"+uuid.NewString()[:6])
	admin, teacher, ta, student := mk(store.UserRoleADMIN, course, ""), mk(store.UserRoleTEACHER, course, "TEACHER"), mk(store.UserRoleTA, course, "TA"), mk(store.UserRoleSTUDENT, course, "STUDENT")
	foreign := mk(store.UserRoleTEACHER, other, "TEACHER")
	path := "/courses/" + course.String() + "/integrations/imap"
	body := map[string]any{"host": "127.0.0.1", "port": 1, "security": "NONE", "username": "u", "password": "p", "enabled": false}
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": uuid.NewString()} }
	do := func(method, p string, s session, b any, h map[string]string) int {
		return r.do(req{method: method, path: p, bearer: s.access, body: b, hdr: h}).code
	}
	// GET
	for name, want := range map[string]int{"admin": 200, "giảng viên": 200, "ta": 403, "sinh viên": 403, "giảng viên lớp khác": 403} {
		s := map[string]session{"admin": admin, "giảng viên": teacher, "ta": ta, "sinh viên": student, "giảng viên lớp khác": foreign}[name]
		require.Equal(t, want, do(http.MethodGet, path, s, nil, nil), "GET "+name)
	}
	// PUT / test: chỉ ADMIN
	for name, s := range map[string]session{"giảng viên": teacher, "ta": ta, "sinh viên": student, "giảng viên lớp khác": foreign} {
		require.Equal(t, 403, do(http.MethodPut, path, s, body, nil), "PUT "+name)
		require.Equal(t, 403, do(http.MethodPost, path+"/test", s, nil, nil), "test "+name)
	}
	require.Equal(t, 200, do(http.MethodPut, path, admin, body, nil))
	require.Equal(t, 200, do(http.MethodPost, path+"/test", admin, nil, nil))
	// sync-now: TA, TEACHER; ADMIN, STUDENT, lớp khác 403
	for name, s := range map[string]session{"admin": admin, "sinh viên": student, "giảng viên lớp khác": foreign} {
		require.Equal(t, 403, do(http.MethodPost, path+"/sync-now", s, nil, idem()), "sync-now "+name)
	}
	require.Equal(t, 202, do(http.MethodPost, path+"/sync-now", ta, nil, idem()))
	r.rdb.Del(t.Context(), "ep:imapsync:"+course.String()+":job", "ep:imapsync:"+course.String()+":rl")
	require.Equal(t, 202, do(http.MethodPost, path+"/sync-now", teacher, nil, idem()))
	// thư không xử lý được: nhân viên đọc, sinh viên không
	for name, want := range map[string]int{"ta": 200, "giảng viên": 200, "sinh viên": 403, "admin": 403} {
		s := map[string]session{"ta": ta, "giảng viên": teacher, "sinh viên": student, "admin": admin}[name]
		require.Equal(t, want, do(http.MethodGet, "/courses/"+course.String()+"/mail-inbound", s, nil, nil), "mail-inbound "+name)
	}
	// GET của giảng viên không có mật khẩu
	res := r.do(req{method: http.MethodGet, path: path, bearer: teacher.access})
	require.NotContains(t, string(res.body), `"p"`)

	// Teams (US-P7-11): GET = ADMIN, TEACHER; PUT / test = chỉ ADMIN; đồng bộ = TA, TEACHER (ADMIN, STUDENT, lớp khác 403)
	tp := "/courses/" + course.String() + "/integrations/teams"
	tbody := map[string]any{"tenant_id": "t", "client_id": "c", "client_secret": "s", "enabled": true}
	for name, want := range map[string]int{"admin": 200, "giảng viên": 200, "ta": 403, "sinh viên": 403, "giảng viên lớp khác": 403} {
		s := map[string]session{"admin": admin, "giảng viên": teacher, "ta": ta, "sinh viên": student, "giảng viên lớp khác": foreign}[name]
		require.Equal(t, want, do(http.MethodGet, tp, s, nil, nil), "GET teams "+name)
	}
	for name, s := range map[string]session{"giảng viên": teacher, "ta": ta, "sinh viên": student, "giảng viên lớp khác": foreign} {
		require.Equal(t, 403, do(http.MethodPut, tp, s, tbody, nil), "PUT teams "+name)
		require.Equal(t, 403, do(http.MethodPost, tp+"/test", s, nil, nil), "test teams "+name)
	}
	require.Equal(t, 200, do(http.MethodPut, tp, admin, tbody, nil))
	res = r.do(req{method: http.MethodGet, path: tp, bearer: teacher.access})
	require.NotContains(t, string(res.body), `"client_secret"`)
	var asgID uuid.UUID
	require.NoError(t, r.pool.QueryRow(t.Context(), `insert into assignments (course_id, title, due_at, submission_code, created_by, channels, teams_class_id, teams_assignment_id, published_at, rubric)
		values ($1, 'BT', now() + interval '2 days', 'BT03', $2, ARRAY['IN_APP','TEAMS'], 'c', 'a', now(), '[]') returning id`, course, owner.ID).Scan(&asgID))
	sp := "/courses/" + course.String() + "/assignments/" + asgID.String() + "/teams/sync"
	for name, s := range map[string]session{"admin": admin, "sinh viên": student, "giảng viên lớp khác": foreign} {
		require.Equal(t, 403, do(http.MethodPost, sp, s, nil, idem()), "teams sync "+name)
	}
	require.Equal(t, 202, do(http.MethodPost, sp, ta, nil, idem()))
	r.rdb.Del(t.Context(), "ep:teamssync:"+asgID.String()+":job", "ep:teamssync:"+asgID.String()+":rl")
	require.Equal(t, 202, do(http.MethodPost, sp, teacher, nil, idem()))
}
