package course_test

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/course"
)

// ===== AC3: thời gian xử lý không phân biệt được nguyên nhân =====

func TestJoinFailureTimingEqualized(t *testing.T) {
	r := newRig(t)
	causes, _ := r.sixCauses()
	u, _ := r.student("")
	svc := course.Service{Pool: r.pool, Clock: r.clk} // không Redis: đo đúng đường tra mã + quyết định
	names := make([]string, 0, len(causes))
	for n := range causes {
		names = append(names, n)
	}
	sort.Strings(names)
	run := func(name string) time.Duration {
		start := time.Now()
		_, err := svc.Preview(context.Background(), u.ID, "", causes[name])
		d := time.Since(start)
		require.ErrorIs(t, err, course.ErrJoinInvalid, name)
		return d
	}
	for range 5 { // khởi động kết nối / plan cache
		for _, n := range names {
			run(n)
		}
	}
	samples := map[string][]time.Duration{}
	for range 20 { // xen kẽ để trôi tải làm lệch đều
		for _, n := range names {
			samples[n] = append(samples[n], run(n))
		}
	}
	med := map[string]time.Duration{}
	lo, hi := time.Duration(1<<62), time.Duration(0)
	for n, ds := range samples {
		sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
		med[n] = (ds[9] + ds[10]) / 2
		lo, hi = min(lo, med[n]), max(hi, med[n])
	}
	require.LessOrEqualf(t, float64(hi-lo)/float64(lo), 0.35, "trung vị mỗi nguyên nhân: %v", med)
}

// ===== AC4: giới hạn đoán mã =====

func TestJoinRateLimit5Per10Min(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, sv := r.student("")
	for i := range 5 {
		res := r.join(sv, fmt3("ZZZZZ", i))
		require.Equal(t, http.StatusNotFound, res.code, "lần %d", i)
	}
	blocked := r.join(sv, k.code) // mã ĐÚNG vẫn bị chặn
	require.Equal(t, http.StatusTooManyRequests, blocked.code, string(blocked.body))
	require.Equal(t, "RATE_LIMITED", blocked.errCode())
	require.NotEmpty(t, blocked.hdr.Get("Retry-After"))
	secs := int(blocked.json()["retry_after"].(float64))
	require.InDelta(t, 600, secs, 2)
	require.Equal(t, http.StatusTooManyRequests, r.preview(sv, k.code).code, "preview cũng bị chặn")

	r.clk.Advance(9 * time.Minute)
	left := r.join(sv, k.code)
	require.Equal(t, http.StatusTooManyRequests, left.code, "cửa sổ trượt: chưa hết 10 phút")
	require.InDelta(t, 60, left.json()["retry_after"].(float64), 2)
	r.clk.Advance(61 * time.Second)
	require.Equal(t, http.StatusOK, r.join(sv, k.code).code, "hết 10 phút thì thử lại được")
}

func fmt3(prefix string, i int) string { return prefix + string(rune('A'+i)) + "A" }

func TestJoinRateLimitPerIP(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	for i := range 20 { // mỗi người chỉ sai MỘT lần: vẫn dồn vào bộ đếm IP
		_, sv := r.student("")
		require.Equal(t, http.StatusNotFound, r.join(sv, fmt3("YYYYY", i%20)).code)
	}
	_, fresh := r.student("")
	res := r.join(fresh, k.code)
	require.Equal(t, http.StatusTooManyRequests, res.code, "IP đã sai 20 lần ⇒ người mới với mã đúng cũng bị chặn")
	require.Equal(t, "RATE_LIMITED", res.errCode())
}

func TestJoinRateLimitOnlyFailures(t *testing.T) {
	r := newRig(t)
	full := r.klass(map[string]any{"capacity": 1})
	u0, _ := r.student("")
	r.enroll(full.course, u0.ID, "STUDENT", "ACTIVE")
	open := r.klass(nil)
	_, sv := r.student("")
	for range 8 { // COURSE_FULL không phải thất bại đoán mã
		require.Equal(t, http.StatusConflict, r.join(sv, full.code).code)
	}
	require.Equal(t, http.StatusOK, r.join(sv, open.code).code)
	for range 25 { // thành công từ cùng IP không đếm
		_, s := r.student("")
		require.Equal(t, http.StatusOK, r.join(s, open.code).code)
	}
}

func TestJoinRateLimitSharedAcrossInstances(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, _ := r.student("")
	ctx := context.Background()
	a := course.Service{Pool: r.pool, Clock: r.clk, Redis: r.rdb}
	b := course.Service{Pool: r.pool, Clock: r.clk, Redis: r.rdb} // "bản gateway thứ hai": không chung bộ nhớ, chỉ chung Redis
	for range 5 {
		_, err := a.Join(ctx, u.ID, "10.9.9.9", "ZZZZZZZ")
		require.ErrorIs(t, err, course.ErrJoinInvalid)
	}
	_, err := b.Join(ctx, u.ID, "10.9.9.9", k.code)
	var rl *course.RateLimitedError
	require.ErrorAs(t, err, &rl)
	require.Positive(t, rl.RetryAfter)
}

func TestJoinAttemptNotLogged(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, sv := r.student("")
	for _, c := range []string{"QWERTYU", "qwertyu", "zzzzzzz"} {
		r.join(sv, c)
		r.preview(sv, c)
	}
	r.join(sv, k.code)
	logs := r.logs.String()
	for _, secret := range []string{"QWERTYU", "qwertyu", "ZZZZZZZ", "zzzzzzz", k.code, strings.ToLower(k.code)} {
		require.NotContains(t, logs, secret, "mã thử không được nằm trong log")
	}
}

func TestJoinGuessSimulation(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, sv := r.student("")
	hits, blocked := 0, 0
	for i := range 100 {
		guess, err := course.NewJoinCode()
		require.NoError(t, err)
		if guess == k.code {
			continue
		}
		res := r.join(sv, guess)
		switch res.code {
		case http.StatusOK:
			hits++
		case http.StatusTooManyRequests:
			blocked++
			require.GreaterOrEqual(t, i, 5, "chỉ chặn sau lần thất bại thứ 5")
		default:
			require.Equal(t, http.StatusNotFound, res.code)
			require.Less(t, i, 5)
		}
	}
	require.Zero(t, hits)
	require.Equal(t, 95, blocked, "sau lần thứ 5 mọi yêu cầu đều 429")
}

// ===== AC5: tạo lại mã =====

func TestRegenerateOldCodeDeadImmediately(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	res := r.post(k.a.G, "/courses/"+k.id+"/join-code/regenerate", nil, nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.NotEqual(t, k.code, res.json()["join_code"])
	_, sv := r.student("")
	for _, op := range []func(session, string) resp{r.preview, r.join} {
		got := op(sv, k.code)
		require.Equal(t, http.StatusNotFound, got.code)
		require.Equal(t, "JOIN_CODE_INVALID", got.errCode())
	}
}

func TestRegenerateNewCodeWorks(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	res := r.post(k.a.A, "/courses/"+k.id+"/join-code/regenerate", nil, nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	code := res.json()["join_code"].(string)
	require.Regexp(t, `^[ABCDEFGHJKMNPQRSTUVWXYZ23456789]{7}$`, code)
	require.True(t, strings.HasSuffix(res.json()["join_url"].(string), "/join/"+code))
	require.Equal(t, code, r.joinCodeOf(k.id))
	_, sv := r.student("")
	require.Equal(t, http.StatusOK, r.join(sv, code).code)
	require.Len(t, res.json(), 3)
}

func TestRegenerateExistingMembersUnaffected(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, sv := r.student("")
	require.Equal(t, http.StatusOK, r.join(sv, k.code).code)
	require.Equal(t, http.StatusOK, r.post(k.a.G, "/courses/"+k.id+"/join-code/regenerate", nil, nil).code)
	require.Equal(t, http.StatusOK, r.get(sv, "/courses/"+k.id).code)
	st, _, _, _ := r.enrollment(k.id, u.ID)
	require.Equal(t, "ACTIVE", st)
}

func TestRegenerateTAForbidden(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, sv := r.student("")
	for name, s := range map[string]session{"TA": k.a.T, "GV lớp khác": k.a.G2, "SV": sv} {
		res := r.post(s, "/courses/"+k.id+"/join-code/regenerate", nil, nil)
		require.Equal(t, http.StatusForbidden, res.code, name)
	}
	require.Equal(t, k.code, r.joinCodeOf(k.id), "mã không đổi")
	require.Equal(t, http.StatusOK, r.post(k.a.G, "/courses/"+k.id+"/join-code/regenerate", nil, nil).code)
	require.Equal(t, http.StatusOK, r.post(k.a.A, "/admin/courses/"+k.id+"/archive", nil, nil).code)
	res := r.post(k.a.G, "/courses/"+k.id+"/join-code/regenerate", nil, nil)
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "COURSE_ARCHIVED", res.errCode())
}

func TestRegenerateAuditNoFullCode(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	res := r.post(k.a.G, "/courses/"+k.id+"/join-code/regenerate", nil, nil)
	newCode := res.json()["join_code"].(string)
	var text, actor string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select coalesce(before::text,'') || coalesce(after::text,''), actor_id::text from audit_log where entity = 'course' and entity_id = $1 and action = 'join_code_regenerated'`, k.id).Scan(&text, &actor))
	require.Equal(t, k.a.gv.ID.String(), actor)
	require.NotContains(t, text, k.code)
	require.NotContains(t, text, newCode)
	require.Contains(t, text, k.code[:2])
	require.Contains(t, text, newCode[:2])
}

// ===== AC6: cài đặt tham gia =====

func TestJoinSettingsGetShape(t *testing.T) {
	r := newRig(t)
	k := r.klass(map[string]any{"capacity": 40})
	_, sv := r.student("")
	for name, s := range map[string]session{"TA": k.a.T, "GV": k.a.G, "Admin": k.a.A} {
		res := r.get(s, "/courses/"+k.id+"/join-code")
		require.Equal(t, http.StatusOK, res.code, name+" "+string(res.body))
		m := res.json()
		require.Len(t, m, 10, name)
		require.Equal(t, k.code, m["join_code"])
		require.Equal(t, true, m["enabled"])
		require.Equal(t, false, m["require_approval"])
		require.EqualValues(t, 40, m["capacity"])
		require.EqualValues(t, 0, m["active_students"])
		require.True(t, strings.HasSuffix(m["join_url"].(string), "/join/"+k.code))
		require.Equal(t, `W/"v1"`, res.hdr.Get("ETag"))
	}
	require.Equal(t, http.StatusForbidden, r.get(sv, "/courses/"+k.id+"/join-code").code)
	require.Equal(t, http.StatusForbidden, r.get(k.a.G2, "/courses/"+k.id+"/join-code").code)
}

func TestJoinSettingsValidation(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	for i := range 3 {
		u, _ := r.student("")
		_ = i
		r.enroll(k.course, u.ID, "STUDENT", "ACTIVE")
	}
	day := 24 * time.Hour
	at := func(d time.Duration) string { return r.clk.Now().Add(d).UTC().Format(time.RFC3339) }
	for name, tc := range map[string]struct {
		body  map[string]any
		field string
	}{
		"hạn trong quá khứ":      {map[string]any{"expires_at": at(-time.Hour)}, "expires_at"},
		"hạn quá 366 ngày":       {map[string]any{"expires_at": at(367 * day)}, "expires_at"},
		"sĩ số nhỏ hơn đang học": {map[string]any{"capacity": 2}, "capacity"},
		"sĩ số 1001":             {map[string]any{"capacity": 1001}, "capacity"},
		"sĩ số 0":                {map[string]any{"capacity": 0}, "capacity"},
		"tên miền sai":           {map[string]any{"allowed_email_domain": "@ptit"}, "allowed_email_domain"},
		"khoá lạ":                {map[string]any{"join_code": "AAAAAAA"}, "join_code"},
	} {
		res := r.setJoin(k, tc.body)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, name+" "+string(res.body))
		require.Contains(t, string(res.body), tc.field, name)
	}
	ok := r.setJoin(k, map[string]any{"expires_at": at(30 * day), "capacity": 3, "allowed_email_domain": "PTIT.edu.vn", "require_approval": true})
	require.Equal(t, http.StatusOK, ok.code, string(ok.body))
	require.Equal(t, "ptit.edu.vn", ok.json()["allowed_email_domain"], "chuẩn hoá chữ thường")
	require.EqualValues(t, 2, ok.json()["version"])
	// xoá bằng null / rỗng
	clear := r.setJoin(k, map[string]any{"expires_at": nil, "capacity": nil, "allowed_email_domain": ""})
	require.Equal(t, http.StatusOK, clear.code, string(clear.body))
	require.Nil(t, clear.json()["expires_at"])
	require.Nil(t, clear.json()["capacity"])
	require.Nil(t, clear.json()["allowed_email_domain"])
	require.Equal(t, true, clear.json()["require_approval"], "khoá vắng = giữ nguyên")
	// sai version
	conflict := r.setJoin(k, map[string]any{"enabled": false, "version": 1})
	require.Equal(t, http.StatusConflict, conflict.code)
	require.Equal(t, "VERSION_CONFLICT", conflict.errCode())
	require.EqualValues(t, 3, conflict.details()["current_version"])
}

func TestJoinSettingsEffectDisabled(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"enabled": false}).code)
	_, sv := r.student("")
	require.Equal(t, "JOIN_CODE_INVALID", r.join(sv, k.code).errCode())
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"enabled": true}).code)
	require.Equal(t, http.StatusOK, r.join(sv, k.code).code)
}

func TestJoinSettingsEffectExpired(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"expires_at": r.clk.Now().Add(time.Hour).UTC().Format(time.RFC3339)}).code)
	u, sv := r.student("")
	require.Equal(t, http.StatusOK, r.preview(sv, k.code).code)
	r.clk.Advance(2 * time.Hour)
	sv = r.mustLogin(u.Email) // token cũ đã hết hạn theo đồng hồ giả
	require.Equal(t, "JOIN_CODE_INVALID", r.join(sv, k.code).errCode())
}

func TestJoinSettingsEffectDomain(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"allowed_email_domain": "example.test"}).code)
	_, ok := r.student("")
	require.Equal(t, http.StatusOK, r.join(ok, k.code).code, "email @example.test khớp")
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"allowed_email_domain": "ptit.edu.vn"}).code)
	_, other := r.student("")
	require.Equal(t, "JOIN_CODE_INVALID", r.join(other, k.code).errCode(), "không khớp tên miền ⇒ lỗi đồng nhất")
}

func TestJoinSettingsEffectCapacity(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"capacity": 2}).code)
	for range 2 {
		_, s := r.student("")
		require.Equal(t, http.StatusOK, r.join(s, k.code).code)
	}
	_, third := r.student("")
	require.Equal(t, "COURSE_FULL", r.join(third, k.code).errCode())
}

func TestJoinSettingsEffectApproval(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"require_approval": true}).code)
	u, s := r.student("")
	require.Equal(t, "PENDING", r.join(s, k.code).json()["status"])
	st, _, _, _ := r.enrollment(k.id, u.ID)
	require.Equal(t, "PENDING", st)
}

func TestJoinSettingsTAForbidden(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, sv := r.student("")
	for name, s := range map[string]session{"TA": k.a.T, "SV": sv, "GV lớp khác": k.a.G2} {
		res := r.put(s, "/courses/"+k.id+"/join-settings", map[string]any{"enabled": false, "version": 1})
		require.Equal(t, http.StatusForbidden, res.code, name)
	}
	require.True(t, r.count(`select count(*) from courses where id = $1 and join_enabled`, k.id) == 1)
	res := r.put(k.a.A, "/courses/"+k.id+"/join-settings", map[string]any{"enabled": false, "version": 1})
	require.Equal(t, http.StatusOK, res.code, "Admin được")
	_ = uuid.Nil
}
