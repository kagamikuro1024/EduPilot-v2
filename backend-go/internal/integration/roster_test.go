package integration_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/store"
)

func (r *rig) importRoster(s session, courseID, csv string) resp {
	r.t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	p, err := mw.CreateFormFile("file", "roster.csv")
	require.NoError(r.t, err)
	_, err = p.Write([]byte(csv))
	require.NoError(r.t, err)
	require.NoError(r.t, mw.Close())
	hr := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseID+"/roster/import", &body)
	hr.RemoteAddr = r.ip + ":4444"
	hr.Header.Set("Content-Type", mw.FormDataContentType())
	hr.Header.Set("Authorization", "Bearer "+s.access)
	hr.Header.Set("Idempotency-Key", "k-"+uuid.NewString())
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, hr)
	return resp{code: w.Code, hdr: w.Header(), body: w.Body.Bytes()}
}

func (r *rig) scalar(sql string, args ...any) string {
	r.t.Helper()
	var s *string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&s))
	if s == nil {
		return ""
	}
	return *s
}

func (r *rig) count(sql string, args ...any) int {
	r.t.Helper()
	var n int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&n))
	return n
}

// issue phát token một lần như consumer thư (bản rõ chỉ ở đây, DB giữ băm).
func (r *rig) issue(uid uuid.UUID, kind auth.TokenKind) string {
	r.t.Helper()
	tx, err := r.pool.Begin(r.t.Context())
	require.NoError(r.t, err)
	plain, err := auth.Tokens{Clock: r.clk}.Issue(r.t.Context(), tx, uid, kind, 72*time.Hour, nil)
	require.NoError(r.t, err)
	require.NoError(r.t, tx.Commit(r.t.Context()))
	return plain
}

// classRoutes là mọi route lớp mà người ngoài / PENDING không được chạm (đọc và ghi).
func classRoutes(id string) []req {
	b := "/courses/" + id
	return []req{
		{method: http.MethodGet, path: b}, {method: http.MethodGet, path: b + "/members"}, {method: http.MethodGet, path: b + "/join-code"},
		{method: http.MethodGet, path: b + "/share-sources"}, {method: http.MethodGet, path: b + "/assistant-candidates"},
		{method: http.MethodPost, path: b + "/join-code/regenerate"}, {method: http.MethodPut, path: b + "/join-settings", body: map[string]any{"version": 1}},
		{method: http.MethodPut, path: b + "/assistants", body: map[string]any{"ta_ids": []string{}}}, {method: http.MethodGet, path: b + "/llm-budget"},
		{method: http.MethodPost, path: b + "/share-from", body: map[string]any{"source_course_id": uuid.NewString(), "what": []string{"documents"}}, hdr: map[string]string{"Idempotency-Key": "k-sweep-0001"}},
		{method: http.MethodPost, path: b + "/members/" + uuid.NewString() + "/approve"}, {method: http.MethodDelete, path: b + "/members/" + uuid.NewString()},
	}
}

func (r *rig) sweep403(s session, id, who string) {
	r.t.Helper()
	for _, q := range classRoutes(id) {
		q.bearer = s.access
		res := r.do(q)
		require.Equal(r.t, http.StatusForbidden, res.code, "%s %s %s: %s", who, q.method, q.path, res.body)
	}
}

// AC4 — kẻ tấn công đăng ký bằng MSSV của B (email của riêng mình) không thấy gì của B / lớp.
func TestRosterLinkRequiresVerifiedEmail(t *testing.T) {
	r := newRig(t)
	r.worker()
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID, "ta_ids": []string{x.ta.ID.String()}})
	bMail := uniq("sv.kha")
	res := r.importRoster(x.G, id, "Email,Họ và tên,MSSV\n"+bMail+",Sinh Viên Khá,20229002\n")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	b := r.scalar(`select id::text from users where email = $1`, bMail)
	require.NotEmpty(t, b)

	atkMail := uniq("ke.tan.cong")
	reg := r.do(req{method: http.MethodPost, path: "/auth/register", body: map[string]string{"email": atkMail, "password": rigPassword, "full_name": "Kẻ Tấn Công", "student_code": "20229002"}, hdr: map[string]string{"Origin": rigOrigin}})
	require.Equal(t, http.StatusAccepted, reg.code, string(reg.body))
	atk := uuid.MustParse(r.scalar(`select id::text from users where email = $1`, atkMail))
	verify := r.do(req{method: http.MethodPost, path: "/auth/verify-email", body: map[string]string{"token": r.issue(atk, auth.TokenVerifyEmail)}, hdr: map[string]string{"Origin": rigOrigin}})
	require.Equal(t, http.StatusOK, verify.code, string(verify.body))
	sa := r.mustLogin(atkMail)

	r.sweep403(sa, id, "kẻ tấn công trước khi vào bằng mã")
	require.Equal(t, 0, r.count(`select count(*) from enrollments where course_id = $1 and user_id = $2`, id, atk), "MSSV không nối tài khoản vào bản ghi roster")

	join := r.joinWith(sa, r.codeOf(id))
	require.Equal(t, http.StatusOK, join.code, string(join.body))
	require.Equal(t, "PENDING", join.json()["status"])
	st, warn := r.enrollmentOf(id, atk)
	require.Equal(t, []string{"PENDING", "EMAIL_MISMATCH"}, []string{st, warn})
	r.sweep403(sa, id, "kẻ tấn công đã PENDING")

	// B vẫn ACTIVE và nguyên vẹn; kẻ tấn công không có thư / thông báo nào dính tới B hay lớp.
	bst, bwarn := r.enrollmentOf(id, uuid.MustParse(b))
	require.Equal(t, []string{"ACTIVE", ""}, []string{bst, bwarn})
	require.Empty(t, r.notifications(sa), "không thông báo nào cho kẻ tấn công")
	require.Equal(t, 0, r.count(`select count(*) from mail_outbox where to_addr = $1 and template = 'invite_student'`, atkMail))
	require.Equal(t, 0, r.count(`select count(*) from mail_outbox where payload::text like '%' || $1 || '%' and to_addr = $2`, bMail, atkMail))
	mine := r.do(req{method: http.MethodGet, path: "/me/courses", bearer: sa.access}).json()["items"].([]any)
	require.Len(t, mine, 1, "chỉ dòng chờ duyệt của chính mình, kèm thông tin tóm tắt như màn xem trước")
	require.Equal(t, "PENDING", mine[0].(map[string]any)["enrollment_status"])

	// "Hôm nay" của kẻ tấn công: chỉ yêu cầu chờ duyệt của CHÍNH họ; không có gì của B (tên, email, MSSV, id) và không có buổi học / việc của lớp.
	today := r.do(req{method: http.MethodGet, path: "/me/today", bearer: sa.access})
	require.Equal(t, http.StatusOK, today.code, string(today.body))
	require.Equal(t, "JOIN_PENDING", today.json()["recommended"].(map[string]any)["kind"])
	require.Empty(t, today.json()["timeline"])
	for _, leak := range []string{bMail, "Sinh Viên Khá", "20229002", b, x.gv.Email, "JOIN_REQUEST", "EMAIL_MISMATCH", "COURSE_SETUP"} {
		require.NotContains(t, string(today.body), leak)
	}

	// Giảng viên thấy cảnh báo ở hàng chờ.
	m := r.do(req{method: http.MethodGet, path: "/courses/" + id + "/members?status=PENDING", bearer: x.G.access}).json()["items"].([]any)
	require.Len(t, m, 1)
	require.Equal(t, "EMAIL_MISMATCH", m[0].(map[string]any)["warning"])
	waitFor(t, 10*time.Second, "JOIN_REQUEST cho giảng viên", func() bool { return len(r.notesOf(x.G, "JOIN_REQUEST")) == 1 })
}

// AC5 — đăng ký bằng email của người trong roster không tạo gì, không khác biệt, thư tới chủ hộp thư.
func TestRegisterOnRosterEmailGivesNothing(t *testing.T) {
	r := newRig(t)
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID})
	victim := uniq("victim")
	require.Equal(t, http.StatusOK, r.importRoster(x.G, id, "Email,Họ và tên,MSSV\n"+victim+",Nạn Nhân,20229003\n").code)
	users, mails := r.count(`select count(*) from users`), r.count(`select count(*) from mail_outbox where to_addr = $1 and template = 'invite_student'`, victim)
	require.Equal(t, 1, mails)

	fresh := r.do(req{method: http.MethodPost, path: "/auth/register", body: map[string]string{"email": uniq("moi"), "password": rigPassword, "full_name": "Người Lạ"}, hdr: map[string]string{"Origin": rigOrigin}})
	users++ // email mới tạo một tài khoản
	atk := r.do(req{method: http.MethodPost, path: "/auth/register", body: map[string]string{"email": victim, "password": rigPassword, "full_name": "Kẻ Tấn Công"}, hdr: map[string]string{"Origin": rigOrigin}})
	require.Equal(t, fresh.code, atk.code)
	require.Equal(t, http.StatusAccepted, atk.code)
	require.JSONEq(t, string(fresh.body), string(atk.body), "202 đồng nhất, không lộ email có trong roster")
	require.Equal(t, users, r.count(`select count(*) from users`), "không tạo tài khoản nào cho email của nạn nhân")
	require.Equal(t, 2, r.count(`select count(*) from mail_outbox where to_addr = $1 and template = 'invite_student'`, victim), "thư mời gửi lại cho CHỦ hộp thư")
	require.Equal(t, "Nạn Nhân", r.scalar(`select full_name from users where email = $1`, victim), "kẻ tấn công không đổi được tên")
	require.Equal(t, "", r.scalar(`select password_hash from users where email = $1`, victim), "không đặt được mật khẩu")
	require.Equal(t, http.StatusUnauthorized, r.login(victim, rigPassword).code, "không có phiên")
}

// AC5 (nửa sau): chủ hộp thư đặt mật khẩu bằng lời mời và vào đúng bản ghi roster — không sinh bản ghi thứ hai.
func TestRosterInviteAcceptLinksOwnRow(t *testing.T) {
	r := newRig(t)
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID})
	victim := uniq("chunha")
	require.Equal(t, http.StatusOK, r.importRoster(x.G, id, "Email,Họ và tên,MSSV\n"+victim+",Chủ Hộp Thư,20229004\n").code)
	uid := uuid.MustParse(r.scalar(`select id::text from users where email = $1`, victim))
	enr := r.scalar(`select id::text from enrollments where course_id = $1 and user_id = $2`, id, uid)

	acc := r.do(req{method: http.MethodPost, path: "/auth/accept-invite", body: map[string]string{"token": r.issue(uid, auth.TokenInvite), "password": "Mat-khau-chu-hop-thu-2026"}, hdr: map[string]string{"Origin": rigOrigin}})
	require.Equal(t, http.StatusOK, acc.code, string(acc.body))
	require.Equal(t, "ACTIVE", r.scalar(`select status::text from users where id = $1`, uid))
	require.Equal(t, 1, r.count(`select count(*) from enrollments where user_id = $1`, uid))
	require.Equal(t, enr, r.scalar(`select id::text from enrollments where course_id = $1 and user_id = $2`, id, uid), "đúng bản ghi roster")
	s := session{access: acc.json()["access_token"].(string)}
	mine := r.do(req{method: http.MethodGet, path: "/me/courses", bearer: s.access}).json()["items"].([]any)
	require.Len(t, mine, 1)
	require.Equal(t, id, mine[0].(map[string]any)["course"].(map[string]any)["id"])
	require.Equal(t, "ACTIVE", mine[0].(map[string]any)["enrollment_status"])
}

// AC6 — sinh viên có trong roster tự nhập mã lớp ⇒ nối vào enrollment sẵn có.
func TestJoinByCodeAfterRosterNoDuplicate(t *testing.T) {
	r := newRig(t)
	x := r.f2()
	id := r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID})
	u, su := r.student("")
	require.Equal(t, http.StatusOK, r.importRoster(x.G, id, "Email,Họ và tên,MSSV\n"+u.Email+",Sinh Viên,20229005\n").code)
	enr := r.scalar(`select id::text from enrollments where course_id = $1 and user_id = $2`, id, u.ID)
	users := r.count(`select count(*) from users`)
	res := r.joinWith(su, r.codeOf(id))
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, true, res.json()["already_member"])
	require.Equal(t, "ACTIVE", res.json()["status"])
	require.Equal(t, 1, r.count(`select count(*) from enrollments where course_id = $1 and user_id = $2`, id, u.ID))
	require.Equal(t, enr, r.scalar(`select id::text from enrollments where course_id = $1 and user_id = $2`, id, u.ID))
	require.Equal(t, users, r.count(`select count(*) from users`))
}

// AC9/AC10 — chia sẻ giữ audience, không lộ sang lớp thứ ba, một chiều.
func (r *rig) sharedFixture() (x f2, src, dst, third string) {
	r.t.Helper()
	x = r.f2()
	src = r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID})
	dst = r.openCourse(x.A, map[string]any{"teacher_id": x.gv.ID})
	third = r.openCourse(x.A, map[string]any{})
	return
}

func (r *rig) chunk(doc, course, audience string, ord int) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), `insert into content_chunks (document_id, course_ids, audience, ord, text) values ($1, array[$2::uuid], $3::chunk_audience, $4, 'nội dung')`, doc, course, audience, ord)
	require.NoError(r.t, err)
}

func (r *rig) doc(course, typ string, visible bool) string {
	return r.scalar(`insert into documents (course_id, title, type, status, visible_to_students) values ($1, 'T', $2::document_type, 'READY', $3) returning id::text`, course, typ, visible)
}

func (r *rig) shareFrom(s session, target, source string) resp {
	return r.do(req{method: http.MethodPost, path: "/courses/" + target + "/share-from", bearer: s.access, body: map[string]any{"source_course_id": source, "what": []string{"documents"}}, hdr: map[string]string{"Idempotency-Key": "k-" + uuid.NewString()}})
}

func (r *rig) chunksOf(course string) map[string]string {
	rows, err := store.New(r.pool).ChunksForCourse(r.t.Context(), store.ChunksForCourseParams{CourseID: uuid.MustParse(course), Lim: 100})
	require.NoError(r.t, err)
	out := map[string]string{}
	for _, c := range rows {
		out[c.ID.String()] = string(c.Audience)
	}
	return out
}

func TestShareDoesNotLeakToOtherCourses(t *testing.T) {
	r := newRig(t)
	x, src, dst, third := r.sharedFixture()
	d := r.doc(src, "LECTURE", true)
	r.chunk(d, src, "ALL", 0)
	only1, so := r.student("")
	r.enroll(uuid.MustParse(src), only1.ID, "STUDENT", "ACTIVE")
	require.Equal(t, http.StatusOK, r.shareFrom(x.G, dst, src).code)

	require.Len(t, r.chunksOf(dst), 1, "lớp đích thấy chunk đã chia sẻ")
	require.Len(t, r.chunksOf(src), 1)
	require.Empty(t, r.chunksOf(third), "lớp thứ ba không chia sẻ ⇒ không thấy chunk của lớp 1")
	// Sinh viên chỉ ở lớp nguồn không vào được lớp đích; dữ liệu lớp đích không mở qua lớp nguồn.
	r.sweep403(so, dst, "sinh viên chỉ ở lớp nguồn")
}

func TestShareKeepsAudience(t *testing.T) {
	r := newRig(t)
	x, src, dst, _ := r.sharedFixture()
	d := r.doc(src, "ANSWER_KEY", false)
	r.chunk(d, src, "GRADING", 0)
	r.chunk(d, src, "STAFF", 1)
	pub := r.doc(src, "LECTURE", true)
	r.chunk(pub, src, "ALL", 0)
	before := r.chunksOf(src)
	require.Equal(t, http.StatusOK, r.shareFrom(x.G, dst, src).code)
	after := r.chunksOf(dst)
	require.Equal(t, before, after, "audience từng chunk không đổi khi sang lớp đích (đáp án vẫn GRADING)")
	require.Equal(t, 1, r.count(`select count(*) from documents where id = $1 and visible_to_students = false`, d), "cờ ẩn với sinh viên giữ nguyên")
	for _, a := range after {
		if a == "GRADING" || a == "STAFF" {
			continue
		}
		require.Equal(t, "ALL", a)
	}
}

func TestShareOneWay(t *testing.T) {
	r := newRig(t)
	x, src, dst, _ := r.sharedFixture()
	d := r.doc(dst, "LECTURE", true)
	r.chunk(d, dst, "ALL", 0)
	s := r.doc(src, "LECTURE", true)
	r.chunk(s, src, "ALL", 0)
	require.Equal(t, http.StatusOK, r.shareFrom(x.G, dst, src).code)
	require.Len(t, r.chunksOf(src), 1, "chia sẻ 1→2 không làm lớp nguồn thấy tài liệu riêng của lớp đích")
	// Giảng viên chỉ dạy lớp đích (không dạy nguồn) không dùng share-from để đọc lớp nguồn.
	gv2 := r.addUser(uniq("gv2"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	g2 := r.mustLogin(gv2.Email)
	other := r.openCourse(x.A, map[string]any{"teacher_id": gv2.ID}) // cùng học phần, giảng viên khác
	res := r.shareFrom(g2, other, src)
	require.Equal(t, http.StatusForbidden, res.code, string(res.body))
	require.Equal(t, "source_course", res.details()["reason"])
	require.Equal(t, http.StatusForbidden, r.do(req{method: http.MethodGet, path: "/courses/" + src + "/members", bearer: g2.access}).code)
}
