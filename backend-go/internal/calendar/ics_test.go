package calendar_test

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/calendar"
)

func (f *fx) storedHash(u any) *string {
	f.t.Helper()
	var h *string
	require.NoError(f.t, f.pool.QueryRow(f.t.Context(), `select ics_token from users where id = $1`, u).Scan(&h))
	return h
}

// TestICSTokenStoredHashed — AC7: chỉ lưu hex(sha256(token)); token thô không có trong DB; URL đúng dạng, 43 ký tự.
func TestICSTokenStoredHashed(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	tok, u := f.icsToken(f.sv, auth.RoleStudent)
	require.Len(t, tok, 43)
	require.Equal(t, "http://app.test/api/v1/calendar/feed.ics?token="+tok, u)
	h := f.storedHash(f.sv)
	require.NotNil(t, h)
	require.Equal(t, calendar.HashToken(tok), *h)
	require.Regexp(t, `^[0-9a-f]{64}$`, *h)
	require.NotEqual(t, tok, *h)
	require.Zero(t, count(f, `select count(*) from users where ics_token = $1`, tok))
	require.Zero(t, count(f, `select count(*) from audit_log where after::text like '%' || $1 || '%' or before::text like '%' || $1 || '%'`, tok))
}

// TestICSTokenRawRejectedByCheck — AC7: CHECK của 00011 từ chối mọi giá trị không phải 64 chữ hex (23514).
func TestICSTokenRawRejectedByCheck(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	for _, bad := range []string{"raw-token", strings.Repeat("A", 64), strings.Repeat("a", 63), strings.Repeat("g", 64)} {
		_, err := f.pool.Exec(t.Context(), `update users set ics_token = $2 where id = $1`, f.sv, bad)
		var pg *pgconn.PgError
		require.ErrorAs(t, err, &pg, bad)
		require.Equal(t, "23514", pg.Code, bad)
	}
}

// TestICSTokenShownOnce — AC7: URL chỉ có ở phản hồi POST; GET chỉ trả {exists}, không có token.
func TestICSTokenShownOnce(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	tk := f.tok(f.sv, auth.RoleStudent)
	st, _, b := f.do("GET", "/api/v1/me/calendar/ics-token", tk, "")
	require.Equal(t, 200, st)
	require.JSONEq(t, `{"exists":false}`, string(b))
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	st, h, b := f.do("GET", "/api/v1/me/calendar/ics-token", tk, "")
	require.Equal(t, 200, st)
	require.JSONEq(t, `{"exists":true}`, string(b))
	require.NotContains(t, string(b)+fmt.Sprint(h), tok)
	st, h, b = f.do("POST", "/api/v1/me/calendar/ics-token", tk, "")
	require.Equal(t, 201, st)
	require.Equal(t, "no-store", h.Get("Cache-Control"))
	require.NotContains(t, string(b), tok)
}

// TestICSTokenRotateInvalidatesOld — AC7: gọi lại POST = xoay; token cũ mất hiệu lực ngay, token mới dùng được.
func TestICSTokenRotateInvalidatesOld(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	old, _ := f.icsToken(f.sv, auth.RoleStudent)
	st, _, _ := f.feed(old)
	require.Equal(t, 200, st)
	fresh, _ := f.icsToken(f.sv, auth.RoleStudent)
	require.NotEqual(t, old, fresh)
	st, _, _ = f.feed(old)
	require.Equal(t, 404, st)
	st, _, _ = f.feed(fresh)
	require.Equal(t, 200, st)
}

// TestICSTokenRevoke — AC7: DELETE đặt NULL, idempotent, feed → 404.
func TestICSTokenRevoke(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	tk := f.tok(f.sv, auth.RoleStudent)
	for range 2 {
		st, _, _ := f.do("DELETE", "/api/v1/me/calendar/ics-token", tk, "")
		require.Equal(t, 204, st)
	}
	require.Nil(t, f.storedHash(f.sv))
	st, _, _ := f.feed(tok)
	require.Equal(t, 404, st)
}

// TestICSTokenNotInLogs — AC7: log truy cập không chứa giá trị `token` (kể cả khi lỗi 404 và 304).
func TestICSTokenNotInLogs(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	f.feed(tok)
	f.feed(strings.Repeat("z", 43))
	require.NotContains(t, f.logs.String(), tok)
	require.NotContains(t, f.logs.String(), "zzzzzzzzzz")
	require.Contains(t, f.logs.String(), "feed.ics") // có log truy cập, chỉ không có giá trị token
}

// TestICSTokenOnlySelf — AC15: token thuộc người gọi; token của người khác không đọc, không xoay, không thu hồi được qua tài khoản mình.
func TestICSTokenOnlySelf(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	tok1, _ := f.icsToken(f.sv, auth.RoleStudent)
	_, _, b := f.do("GET", "/api/v1/me/calendar/ics-token", f.tok(f.sv2, auth.RoleStudent), "")
	require.JSONEq(t, `{"exists":false}`, string(b))
	f.do("DELETE", "/api/v1/me/calendar/ics-token", f.tok(f.sv2, auth.RoleStudent), "")
	f.icsToken(f.sv2, auth.RoleStudent)
	st, _, _ := f.feed(tok1)
	require.Equal(t, 200, st, "thao tác của sv2 không được đụng token của sv1")
	f.icsToken(f.teacher, auth.RoleTeacher) // mọi người dùng đã đăng nhập đều có token của mình
	f.icsToken(f.ta, auth.RoleTA)
	st, _, _ = f.do("POST", "/api/v1/me/calendar/ics-token", "", "")
	require.Equal(t, 401, st)
}

func hasLine(body, line string) bool {
	for _, l := range strings.Split(body, "\r\n") {
		if l == line {
			return true
		}
	}
	return false
}

// TestICSFormat — AC8: khung VCALENDAR, UID ổn định, DTSTART/DTEND UTC, DTEND +1 giờ khi trống, CRLF, hợp ba nguồn, Content-Type, Cache-Control.
func TestICSFormat(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	s := f.session(f.course, 1, 24*time.Hour, "Mật mã")
	e := f.exam(f.course, "Thi giữa kỳ", "SCHEDULED", 48*time.Hour)
	ev := f.event(f.course, "OTHER", "Nộp báo cáo", 72*time.Hour)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	st, h, body := f.feed(tok)
	require.Equal(t, 200, st)
	require.Equal(t, "text/calendar; charset=utf-8", h.Get("Content-Type"))
	require.Equal(t, "private, max-age=300", h.Get("Cache-Control"))
	require.NotEmpty(t, h.Get("ETag"))
	require.True(t, strings.HasPrefix(body, "BEGIN:VCALENDAR\r\n"))
	require.True(t, strings.HasSuffix(body, "END:VCALENDAR\r\n"))
	require.NotContains(t, strings.ReplaceAll(body, "\r\n", ""), "\n")
	for _, l := range []string{"VERSION:2.0", "PRODID:-//EduPilot//Calendar//VI", "CALSCALE:GREGORIAN", "X-WR-CALNAME:EduPilot",
		"UID:class_session-" + s.String() + "@edupilot", "UID:weekly_exam-" + e.String() + "@edupilot", "UID:calendar_event-" + ev.String() + "@edupilot",
		"SUMMARY:Buổi 1 · Mật mã", "SUMMARY:Thi giữa kỳ", "LOCATION:P.301"} {
		require.True(t, hasLine(body, l), "thiếu dòng %q trong\n%s", l, body)
	}
	require.Equal(t, 3, strings.Count(body, "BEGIN:VEVENT"))
	start := f.clk.Now().Add(72 * time.Hour)
	var evStart, evEnd string
	for _, blk := range strings.Split(body, "BEGIN:VEVENT")[1:] {
		if strings.Contains(blk, "calendar_event-") {
			for _, l := range strings.Split(blk, "\r\n") {
				if v, ok := strings.CutPrefix(l, "DTSTART:"); ok {
					evStart = v
				}
				if v, ok := strings.CutPrefix(l, "DTEND:"); ok {
					evEnd = v
				}
			}
		}
	}
	require.Equal(t, start.UTC().Truncate(time.Second).Format("20060102T150405Z"), evStart)
	require.Equal(t, start.UTC().Truncate(time.Second).Add(time.Hour).Format("20060102T150405Z"), evEnd) // DTEND bù +1 giờ
	require.NotContains(t, body, "DESCRIPTION")
}

// TestICSSummaryClassPrefix — AC8: tiền tố `{mã lớp} · ` khi ≥ 2 lớp ACTIVE; 1 lớp thì không.
func TestICSSummaryClassPrefix(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.event(f.course, "OTHER", "Thi giữa kỳ", time.Hour)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	_, _, one := f.feed(tok)
	require.True(t, hasLine(one, "SUMMARY:Thi giữa kỳ"))
	c2 := f.newCourse("ACTIVE")
	f.enroll(c2, f.sv, "STUDENT", "ACTIVE")
	f.event(c2, "OTHER", "Báo cáo", time.Hour)
	var code1, code2 string
	require.NoError(t, f.pool.QueryRow(t.Context(), `select class_code from courses where id = $1`, f.course).Scan(&code1))
	require.NoError(t, f.pool.QueryRow(t.Context(), `select class_code from courses where id = $1`, c2).Scan(&code2))
	_, _, two := f.feed(tok)
	require.True(t, hasLine(two, "SUMMARY:"+code1+" · Thi giữa kỳ"), two)
	require.True(t, hasLine(two, "SUMMARY:"+code2+" · Báo cáo"), two)
}

// TestICSOnlyActiveCourses — AC8: chỉ lớp người đó ACTIVE; bị mời ra khỏi lớp thì sự kiện biến mất ở lần kế.
func TestICSOnlyActiveCourses(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	mine := f.event(f.course, "OTHER", "Của tôi", time.Hour)
	other := f.event(f.newCourse("ACTIVE"), "OTHER", "Lớp người khác", time.Hour)
	pend := f.newCourse("ACTIVE")
	f.enroll(pend, f.sv, "STUDENT", "PENDING")
	pe := f.event(pend, "OTHER", "Chờ duyệt", time.Hour)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	_, _, body := f.feed(tok)
	require.Contains(t, body, mine.String())
	require.NotContains(t, body, other.String())
	require.NotContains(t, body, pe.String())
	require.NotContains(t, body, "Lớp người khác")
	_, err := f.pool.Exec(t.Context(), `update enrollments set status = 'REMOVED', removed_at = now() where user_id = $1 and course_id = $2`, f.sv, f.course)
	require.NoError(t, err)
	_, _, body = f.feed(tok)
	require.NotContains(t, body, mine.String())
}

// TestICSNoDraftExam — AC8: không bài thi DRAFT; không description; không tên người.
func TestICSNoDraftExam(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	d := f.exam(f.course, "Đề nháp", "DRAFT", 0)
	_, err := f.pool.Exec(t.Context(), `insert into calendar_events (course_id, type, title, starts_at, description, created_by) values ($1, 'OTHER', 'Có mô tả', now() + interval '1 hour', 'Mô tả riêng tư', $2)`, f.course, f.teacher)
	require.NoError(t, err)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	_, _, body := f.feed(tok)
	require.NotContains(t, body, d.String())
	require.NotContains(t, body, "Đề nháp")
	require.NotContains(t, body, "Mô tả riêng tư")
	for _, name := range []string{"GV", "TA", "SV1", "SV2"} {
		require.NotContains(t, body, name+"@")
	}
	require.Contains(t, body, "Có mô tả")
}

// TestICSStableUID — AC8: hai lần gọi giống hệt nhau (UID, DTSTAMP, ETag); đổi giờ giữ UID.
func TestICSStableUID(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ev := f.event(f.course, "OTHER", "A", time.Hour)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	_, h1, b1 := f.feed(tok)
	f.clk.Advance(time.Minute)
	_, h2, b2 := f.feed(tok)
	require.Equal(t, b1, b2)
	require.Equal(t, h1.Get("ETag"), h2.Get("ETag"))
	_, err := f.pool.Exec(t.Context(), `update calendar_events set starts_at = starts_at + interval '1 day' where id = $1`, ev)
	require.NoError(t, err)
	_, h3, b3 := f.feed(tok)
	require.NotEqual(t, h1.Get("ETag"), h3.Get("ETag"))
	require.True(t, hasLine(b3, "UID:calendar_event-"+ev.String()+"@edupilot"))
}

// TestICS304 — AC8: If-None-Match trùng → 304 không thân.
func TestICS304(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.event(f.course, "OTHER", "A", time.Hour)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	_, h, _ := f.feed(tok)
	st, _, b := f.do("GET", "/api/v1/calendar/feed.ics?token="+tok, "", "", "If-None-Match", h.Get("ETag"))
	require.Equal(t, http.StatusNotModified, st)
	require.Empty(t, b)
}

// TestICSBadTokenUniform404 — AC9: thiếu / sai / sai độ dài / đã xoay / đã thu hồi → cùng một 404, thân giống hệt (trừ trace_id), không bao giờ 401.
func TestICSBadTokenUniform404(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	old, _ := f.icsToken(f.sv, auth.RoleStudent)
	f.icsToken(f.sv, auth.RoleStudent)
	revoked, _ := f.icsToken(f.sv2, auth.RoleStudent)
	f.do("DELETE", "/api/v1/me/calendar/ics-token", f.tok(f.sv2, auth.RoleStudent), "")
	var bodies []string
	for _, tok := range []string{"", "x", strings.Repeat("a", 43), strings.Repeat("a", 64), old, revoked} {
		st, _, b := f.do("GET", "/api/v1/calendar/feed.ics?token="+tok, "", "")
		require.Equal(t, 404, st, tok)
		bodies = append(bodies, regexpTrace.ReplaceAllString(string(b), ""))
	}
	st, _, b := f.do("GET", "/api/v1/calendar/feed.ics", "", "")
	require.Equal(t, 404, st)
	bodies = append(bodies, regexpTrace.ReplaceAllString(string(b), ""))
	for _, b := range bodies[1:] {
		require.Equal(t, bodies[0], b)
	}
}

// TestICSDisabledUser404 — AC9: tài khoản DISABLED → 404 như token sai.
func TestICSDisabledUser404(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	st, _, _ := f.feed(tok)
	require.Equal(t, 200, st)
	_, err := f.pool.Exec(t.Context(), `update users set status = 'DISABLED' where id = $1`, f.sv)
	require.NoError(t, err)
	st, _, _ = f.feed(tok)
	require.Equal(t, 404, st)
}

// TestICSRateLimitPerIP — AC9: 60 yêu cầu / phút / IP, lượt 61 → 429 + Retry-After, kể cả khi token đúng; IP khác không bị ảnh hưởng.
func TestICSRateLimitPerIP(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	for i := range calendar.FeedPerMin {
		st, _, _ := f.feed(tok)
		require.Equal(t, 200, st, "lượt %d", i+1)
	}
	st, h, _ := f.feed(tok)
	require.Equal(t, 429, st)
	require.NotEmpty(t, h.Get("Retry-After"))
	require.True(t, f.svc.RateOK(t.Context(), "203.0.113.9"), "IP khác có hạn mức riêng")
	f.clk.Advance(time.Minute)
	st, _, _ = f.feed(tok)
	require.Equal(t, 200, st)
}

var regexpTrace = regexp.MustCompile(`"trace_id":"[^"]*",?`)

// TestICSCapAndSpeed — AC8 / SRS 8: tối đa 2.000 sự kiện trong feed, dựng ≤ 500 ms (bỏ kiểm giờ khi EP_SKIP_TIMING).
func TestICSCapAndSpeed(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	_, err := f.pool.Exec(t.Context(), `insert into calendar_events (course_id, type, title, starts_at, created_by)
		select $1, 'OTHER', 'Sự kiện ' || g, now() + g * interval '1 minute', $2 from generate_series(1, 2100) g`, f.course, f.teacher)
	require.NoError(t, err)
	tok, _ := f.icsToken(f.sv, auth.RoleStudent)
	start := time.Now()
	body, err := f.svc.Feed(t.Context(), tok)
	require.NoError(t, err)
	require.Equal(t, calendar.FeedMax, strings.Count(string(body), "BEGIN:VEVENT"))
	if os.Getenv("EP_SKIP_TIMING") == "" {
		require.Less(t, time.Since(start), 500*time.Millisecond)
	}
}
