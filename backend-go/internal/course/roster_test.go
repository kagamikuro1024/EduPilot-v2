package course_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

// upload gửi multipart `file` tới POST /courses/{id}/roster/import. key rỗng = tự sinh; "-" = không gửi Idempotency-Key.
func (r *rig) upload(s session, courseID, query string, data []byte, key string, fields map[string]string) resp {
	r.t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="file"; filename="roster.csv"`)
	h.Set("Content-Type", "application/octet-stream")
	p, err := mw.CreatePart(h)
	require.NoError(r.t, err)
	_, err = p.Write(data)
	require.NoError(r.t, err)
	for k, v := range fields {
		require.NoError(r.t, mw.WriteField(k, v))
	}
	require.NoError(r.t, mw.Close())
	path := "/api/v1/courses/" + courseID + "/roster/import"
	if query != "" {
		path += "?" + query
	}
	hr := httptest.NewRequest(http.MethodPost, path, &body)
	hr.RemoteAddr = r.ip + ":4444"
	hr.Header.Set("Content-Type", mw.FormDataContentType())
	hr.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120.0")
	if s.access != "" {
		hr.Header.Set("Authorization", "Bearer "+s.access)
	}
	switch key {
	case "-":
	case "":
		hr.Header.Set("Idempotency-Key", "k-"+uuid.NewString())
	default:
		hr.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, hr)
	return resp{code: w.Code, hdr: w.Header(), body: w.Body.Bytes()}
}

// csvOf dựng CSV từ các dòng "email|tên|mssv".
func csvOf(rows ...string) []byte {
	var b strings.Builder
	b.WriteString("Email,Họ và tên,MSSV\n")
	for _, r := range rows {
		b.WriteString(strings.ReplaceAll(r, "|", ",") + "\n")
	}
	return []byte(b.String())
}

func svRow(i int) string {
	return fmt.Sprintf("sv%02d.%s@example.test|Sinh Viên %02d|B20DC%05d", i, "x", i, i)
}

func (r *rig) importOK(k klass, data []byte, query string) map[string]any {
	r.t.Helper()
	res := r.upload(k.a.G, k.id, query, data, "", nil)
	require.Equal(r.t, http.StatusOK, res.code, string(res.body))
	return res.json()
}

func intOf(m map[string]any, k string) int { return int(m[k].(float64)) }

func errRows(rep map[string]any) (rows []int, codes []string) {
	for _, e := range rep["errors"].([]any) {
		m := e.(map[string]any)
		rows = append(rows, int(m["row"].(float64)))
		codes = append(codes, m["code"].(string))
	}
	return
}

// 30 dòng: dòng 7 email sai dạng, dòng 19 thiếu tên (số dòng trong tệp; tiêu đề = 1).
func roster30() []byte {
	var rows []string
	for i := 1; i <= 30; i++ {
		switch i {
		case 6: // dòng tệp 7
			rows = append(rows, "khong-phai-email|Sinh Viên 06|B20DC00006")
		case 18: // dòng tệp 19
			rows = append(rows, fmt.Sprintf("sv18.x@example.test||B20DC%05d", 18))
		default:
			rows = append(rows, svRow(i))
		}
	}
	return csvOf(rows...)
}

func TestRosterReportRowNumbers(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	rep := r.importOK(k, roster30(), "dry_run=true")
	rows, codes := errRows(rep)
	require.Equal(t, []int{7, 19}, rows)
	require.Equal(t, []string{"INVALID_EMAIL", "MISSING_NAME"}, codes)
	e := rep["errors"].([]any)[0].(map[string]any)
	require.Equal(t, "email", e["field"])
	require.Equal(t, "Email không đúng dạng.", e["message"])
	require.Equal(t, 30, intOf(rep, "total"))
}

func (r *rig) tableCounts() map[string]int {
	out := map[string]int{}
	for _, tb := range []string{"users", "enrollments", "mail_outbox", "outbox", "notifications", "audit_log"} {
		out[tb] = r.count(`select count(*) from ` + tb)
	}
	return out
}

func TestRosterDryRunWritesNothing(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	before := r.tableCounts()
	rep := r.importOK(k, roster30(), "dry_run=true")
	require.Equal(t, true, rep["dry_run"])
	require.Equal(t, 28, intOf(rep, "created_users"), "dry-run báo đúng số sẽ tạo")
	require.Equal(t, before, r.tableCounts(), "dry-run không ghi users, enrollments, mail_outbox, outbox, notifications, audit_log")
	rep = r.importOK(k, roster30(), "")
	require.Equal(t, false, rep["dry_run"])
	require.Equal(t, 28, intOf(rep, "created_users"))
	require.Equal(t, before["users"]+28, r.tableCounts()["users"])
}

func TestRosterPartialCommit(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	rep := r.importOK(k, roster30(), "")
	require.Len(t, rep["errors"], 2)
	require.Equal(t, 28, r.count(`select count(*) from enrollments where course_id = $1 and joined_via = 'ROSTER' and status = 'ACTIVE'`, k.id), "dòng hợp lệ ghi dù có dòng lỗi")
	require.Equal(t, 0, r.count(`select count(*) from users where email = 'khong-phai-email'`))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where course_id = $1 and action = 'roster_imported'`, k.id))
	require.NotContains(t, r.scalar(`select after::text from audit_log where course_id = $1 and action = 'roster_imported'`, k.id), "example.test", "audit không có email")
}

func TestRosterCreatesInvitedUser(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	email := "sv.moi.import@example.test"
	rep := r.importOK(k, csvOf(email+"|Sinh Viên Mới|B20DC00001"), "")
	require.Equal(t, 1, intOf(rep, "created_users"))
	require.Equal(t, "INVITED", r.scalar(`select status::text from users where email = $1`, email))
	require.Equal(t, "STUDENT", r.scalar(`select role::text from users where email = $1`, email))
	require.Equal(t, "B20DC00001", r.scalar(`select student_code from users where email = $1`, email))
	require.Equal(t, "", r.scalar(`select password_hash from users where email = $1`, email), "chưa có mật khẩu")
	require.Equal(t, "ACTIVE|ROSTER|B20DC00001", r.scalar(`select e.status||'|'||e.joined_via||'|'||e.student_code_snapshot from enrollments e join users u on u.id = e.user_id where u.email = $1 and e.course_id = $2`, email, k.id))
}

func TestRosterLinksVerifiedByEmail(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, _ := r.student("B20DC99999") // đã xác minh email, tự khai MSSV khác dòng roster
	rep := r.importOK(k, csvOf(u.Email+"|Tên Trong Roster|B20DC00001"), "")
	require.Equal(t, 1, intOf(rep, "linked_existing"))
	st, warn, snap, via := r.enrollment(k.id, u.ID)
	require.Equal(t, []string{"ACTIVE", "", "B20DC00001", "ROSTER"}, []string{st, warn, snap, via}, "nối bằng EMAIL; snapshot = MSSV của dòng roster")
	require.Equal(t, "B20DC99999", r.scalar(`select student_code from users where id = $1`, u.ID), "users.student_code không bị ghi đè")
	require.Equal(t, "Người Thử", r.scalar(`select full_name from users where id = $1`, u.ID), "tên tài khoản không bị ghi đè")
}

func TestRosterUnverifiedPending(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u := r.addUser(uniq("chua"), store.UserRoleSTUDENT, store.UserStatusPENDINGVERIFICATION)
	rep := r.importOK(k, csvOf(u.Email+"|Chưa Xác Minh|B20DC00002"), "")
	require.Equal(t, 1, intOf(rep, "pending_unverified"))
	st, warn, snap, _ := r.enrollment(k.id, u.ID)
	require.Equal(t, []string{"PENDING", "EMAIL_UNVERIFIED", "B20DC00002"}, []string{st, warn, snap})
}

func TestRosterInvitedAccountLinkedStaysInvited(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u := r.addUser(uniq("moi"), store.UserRoleSTUDENT, store.UserStatusINVITED)
	rep := r.importOK(k, csvOf(u.Email+"|Đã Được Mời|B20DC00003"), "")
	require.Equal(t, 1, intOf(rep, "linked_existing"))
	st, _, snap, _ := r.enrollment(k.id, u.ID)
	require.Equal(t, "ACTIVE", st)
	require.Equal(t, "B20DC00003", snap)
	require.Equal(t, "INVITED", r.scalar(`select status::text from users where id = $1`, u.ID), "không kích hoạt hộ")
	require.Equal(t, 1, r.count(`select count(*) from mail_outbox where to_addr = $1 and template = 'invite_student'`, u.Email), "thư mời mới nêu lớp vừa thêm")
}

func TestRosterStaffEmailError(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	invitedStaff := r.addUser(uniq("gvmoi"), store.UserRoleTEACHER, store.UserStatusINVITED)
	rep := r.importOK(k, csvOf(k.a.gv.Email+"|A|B20DC00004", k.a.ta.Email+"|B|B20DC00005", invitedStaff.Email+"|C|B20DC00006", k.a.adm.Email+"|D|B20DC00007"), "")
	rows, codes := errRows(rep)
	require.Equal(t, []int{2, 3, 4, 5}, rows)
	require.Equal(t, []string{"EMAIL_BELONGS_TO_STAFF", "EMAIL_BELONGS_TO_STAFF", "EMAIL_BELONGS_TO_STAFF", "EMAIL_BELONGS_TO_STAFF"}, codes, "kể cả tài khoản INVITED mang vai khác sinh viên (Q-QC-P210-2)")
	require.Equal(t, 0, r.count(`select count(*) from enrollments where course_id = $1 and joined_via = 'ROSTER'`, k.id))
	require.Equal(t, 0, r.count(`select count(*) from mail_outbox where template = 'invite_student'`))
}

func TestRosterSkipsRemoved(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, _ := r.student("")
	r.enroll(k.course, u.ID, "STUDENT", "REMOVED")
	rep := r.importOK(k, csvOf(u.Email+"|Đã Bị Mời Ra|B20DC00008"), "")
	require.Equal(t, 1, intOf(rep, "skipped_removed"))
	st, _, _, _ := r.enrollment(k.id, u.ID)
	require.Equal(t, "REMOVED", st, "không tự khôi phục")
}

func TestRosterRowErrorCodes(t *testing.T) {
	r := newRig(t)
	k := r.klass(map[string]any{"capacity": 3})
	disabled := r.addUser(uniq("off"), store.UserRoleSTUDENT, store.UserStatusDISABLED)
	data := csvOf(
		"a-b|Tên|B20DC00001",                            // 2 INVALID_EMAIL
		"ok1@example.test||B20DC00002",                  // 3 MISSING_NAME
		"ok2@example.test|Tên|123",                      // 4 INVALID_STUDENT_CODE (ngắn)
		"ok3@example.test|Tên|B20-DC00003",              // 5 INVALID_STUDENT_CODE (ký tự lạ)
		"ok4@example.test|Tên|",                         // 6 INVALID_STUDENT_CODE (rỗng)
		"hop.le@example.test|Hợp Lệ|B20DC00010",         // 7 tạo
		"HOP.LE@example.test|Hợp Lệ Lần Hai|B20DC00011", // 8 DUPLICATE_EMAIL_IN_FILE (không phân biệt hoa thường)
		"khac@example.test|Người Khác|b20dc00010",       // 9 STUDENT_CODE_CONFLICT (cùng MSSV, email khác)
		k.a.gv.Email+"|Giảng viên|B20DC00012",           // 10 EMAIL_BELONGS_TO_STAFF
		disabled.Email+"|Đã Khoá|B20DC00013",            // 11 EMAIL_DISABLED
		"ok5@example.test|Thứ hai|B20DC00014",           // 12 tạo
		"ok6@example.test|Thứ ba|B20DC00015",            // 13 tạo (đủ 3)
		"ok7@example.test|Thứ tư|B20DC00016",            // 14 COURSE_FULL
	)
	rep := r.importOK(k, data, "")
	rows, codes := errRows(rep)
	require.Equal(t, []int{2, 3, 4, 5, 6, 8, 9, 10, 11, 14}, rows)
	require.Equal(t, []string{"INVALID_EMAIL", "MISSING_NAME", "INVALID_STUDENT_CODE", "INVALID_STUDENT_CODE", "INVALID_STUDENT_CODE",
		"DUPLICATE_EMAIL_IN_FILE", "STUDENT_CODE_CONFLICT", "EMAIL_BELONGS_TO_STAFF", "EMAIL_DISABLED", "COURSE_FULL"}, codes)
	require.Equal(t, 3, intOf(rep, "created_users"))
	require.Equal(t, 3, r.count(`select count(*) from enrollments where course_id = $1 and joined_via = 'ROSTER'`, k.id))
	require.Equal(t, 0, r.count(`select count(*) from users where email in ('khac@example.test', 'ok7@example.test')`), "dòng lỗi không để lại tài khoản")
}

func TestRosterReportTotals(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	verified, _ := r.student("")
	pending := r.addUser(uniq("p"), store.UserRoleSTUDENT, store.UserStatusPENDINGVERIFICATION)
	member, _ := r.student("")
	r.enroll(k.course, member.ID, "STUDENT", "ACTIVE")
	removed, _ := r.student("")
	r.enroll(k.course, removed.ID, "STUDENT", "REMOVED")
	rep := r.importOK(k, csvOf(
		svRow(1), svRow(2), verified.Email+"|V|B20DC00021", pending.Email+"|P|B20DC00022",
		member.Email+"|M|B20DC00023", removed.Email+"|R|B20DC00024", "xx|E|B20DC00025",
	), "")
	require.Equal(t, []int{2, 1, 1, 1, 1, 1}, []int{intOf(rep, "created_users"), intOf(rep, "linked_existing"), intOf(rep, "already_member"), intOf(rep, "pending_unverified"), intOf(rep, "skipped_removed"), len(rep["errors"].([]any))})
	sum := 0
	for _, f := range []string{"created_users", "linked_existing", "already_member", "pending_unverified", "skipped_removed"} {
		sum += intOf(rep, f)
	}
	require.Equal(t, intOf(rep, "total"), sum+len(rep["errors"].([]any)))
}

func TestRosterReimportNoDuplicate(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	data := roster30()
	r.importOK(k, data, "")
	users, enrolls, mails := r.count(`select count(*) from users`), r.count(`select count(*) from enrollments where course_id = $1`, k.id), r.count(`select count(*) from mail_outbox`)
	rep := r.importOK(k, data, "")
	require.Equal(t, 28, intOf(rep, "already_member"))
	require.Equal(t, 0, intOf(rep, "created_users"))
	require.Equal(t, []int{users, enrolls, mails}, []int{r.count(`select count(*) from users`), r.count(`select count(*) from enrollments where course_id = $1`, k.id), r.count(`select count(*) from mail_outbox`)})
}

func TestRosterSameStudentCodeConflict(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	rep := r.importOK(k, csvOf("a.one@example.test|Một|B20DC00031", "b.two@example.test|Hai|B20DC00031"), "")
	rows, codes := errRows(rep)
	require.Equal(t, []int{3}, rows, "dòng sau bị báo, dòng trước vào bình thường")
	require.Equal(t, []string{"STUDENT_CODE_CONFLICT"}, codes)
	require.Equal(t, 1, intOf(rep, "created_users"))
}

func TestRosterInviteMails(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	existing, _ := r.student("")
	var rows []string
	for i := 1; i <= 27; i++ {
		rows = append(rows, svRow(i))
	}
	rows = append(rows, existing.Email+"|Đã Có|B20DC00090", "sv91.x@example.test|Chín Mốt|B20DC00091", "sv92.x@example.test|Chín Hai|B20DC00092")
	rep := r.importOK(k, csvOf(rows...), "")
	require.Equal(t, []int{29, 1}, []int{intOf(rep, "created_users"), intOf(rep, "linked_existing")}, "30 dòng: 29 mới + 1 nối")
	require.Equal(t, 29, r.count(`select count(*) from mail_outbox where template = 'invite_student'`), "một thư mỗi tài khoản mới; người đã xác minh không nhận thư mời")
	r.importOK(k, csvOf(rows...), "")
	require.Equal(t, 29, r.count(`select count(*) from mail_outbox where template = 'invite_student'`), "import lại không gửi lại")
}

func TestRosterNoInvites(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	res := r.upload(k.a.G, k.id, "", csvOf(svRow(1), svRow(2)), "", map[string]string{"send_invites": "false"})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, 2, intOf(res.json(), "created_users"))
	require.Equal(t, 0, r.count(`select count(*) from mail_outbox`))
	require.Equal(t, 2, r.count(`select count(*) from users where status = 'INVITED'`), "tài khoản vẫn INVITED")
}

func TestRosterInviteMailNoStudentCode(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	r.importOK(k, csvOf("sv.mssv@example.test|Sinh Viên|B20DC00077"), "")
	payload := r.scalar(`select payload::text from mail_outbox where template = 'invite_student'`)
	require.NotEmpty(t, payload)
	require.NotContains(t, payload, "B20DC00077")
	require.Contains(t, payload, "An ninh mạng")
	require.Contains(t, payload, "Người Thử", "tên giảng viên nhập danh sách")
}

// AC7: chỉ TEACHER của lớp; ADMIN cũng 403 (không thêm sinh viên vào lớp). Ba biến thể × 7 người gọi + lưu trữ + thiếu khoá.
func TestRosterImportRBACMatrix(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, member := r.student("")
	callers := []struct {
		name string
		s    session
		want int
	}{
		{"không JWT", session{}, 401},
		{"sinh viên ngoài lớp", k.a.S, 403},
		{"sinh viên khác", member, 403},
		{"trợ giảng của lớp", k.a.T, 403},
		{"ADMIN", k.a.A, 403},
		{"giảng viên lớp khác", k.a.G2, 403},
		{"giảng viên của lớp", k.a.G, 200},
	}
	n := 0
	for _, c := range callers {
		for _, v := range []struct{ name, query string }{{"thật", ""}, {"dry_run", "dry_run=true"}, {"không thư", "send_invites=false"}} {
			n++
			res := r.upload(c.s, k.id, v.query, csvOf(fmt.Sprintf("rbac%d@example.test|Sinh Viên|B20DC%05d", n, n)), "", nil)
			require.Equal(t, c.want, res.code, c.name+" / "+v.name+": "+string(res.body))
		}
	}
	require.GreaterOrEqual(t, n, 21)
	// 24 ca: thêm lưu trữ, thiếu khoá, tệp thiếu, sai vai khi thiếu khoá (403 đến trước 422).
	require.Equal(t, http.StatusUnprocessableEntity, r.upload(k.a.G, k.id, "", csvOf(svRow(1)), "-", nil).code, "thiếu Idempotency-Key")
	require.Equal(t, http.StatusForbidden, r.upload(k.a.T, k.id, "", csvOf(svRow(1)), "-", nil).code, "403 đến trước 422 thiếu khoá")
	require.Equal(t, http.StatusForbidden, r.upload(k.a.A, k.id, "", csvOf(svRow(1)), "-", nil).code)
	require.Equal(t, http.StatusOK, r.post(k.a.A, "/admin/courses/"+k.id+"/archive", nil, nil).code)
	res := r.upload(k.a.G, k.id, "", csvOf(svRow(2)), "", nil)
	require.Equal(t, http.StatusConflict, res.code, string(res.body))
	require.Equal(t, "COURSE_ARCHIVED", res.errCode())
}

func TestRosterFileSizeLimit(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	big := append(csvOf(svRow(1)), bytes.Repeat([]byte("# "), (2<<20)/2+10)...)
	res := r.upload(k.a.G, k.id, "", big, "", nil)
	require.Equal(t, http.StatusRequestEntityTooLarge, res.code, string(res.body))
	require.Equal(t, "PAYLOAD_TOO_LARGE", res.errCode())
	huge := bytes.Repeat([]byte("a"), 3<<20)
	require.Equal(t, http.StatusRequestEntityTooLarge, r.upload(k.a.G, k.id, "", huge, "", nil).code, "vượt xa giới hạn route")
	// đúng 2 MiB thì còn đọc được (không phải lỗi kích thước).
	edge := append(csvOf(svRow(1)), bytes.Repeat([]byte("\n"), 2<<20-len(csvOf(svRow(1))))...)
	require.Equal(t, http.StatusOK, r.upload(k.a.G, k.id, "", edge, "", nil).code)
	// Các route khác vẫn giữ MAX_BODY_BYTES 1 MiB.
	res = r.do(req{method: http.MethodPost, path: "/courses/join/preview", bearer: k.a.S.access, body: map[string]string{"code": strings.Repeat("A", 1100000)}})
	require.Equal(t, http.StatusRequestEntityTooLarge, res.code)
}

func TestRosterIdempotent(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	data := csvOf(svRow(1), svRow(2))
	a := r.upload(k.a.G, k.id, "", data, "idem-roster-0001", nil)
	require.Equal(t, http.StatusOK, a.code, string(a.body))
	b := r.upload(k.a.G, k.id, "", data, "idem-roster-0001", nil) // boundary multipart khác nhau, nội dung giống
	require.Equal(t, a.code, b.code)
	require.JSONEq(t, string(a.body), string(b.body), "cùng khoá, cùng tệp ⇒ cùng phản hồi (không thành already_member)")
	require.Equal(t, 2, intOf(b.json(), "created_users"))
	c := r.upload(k.a.G, k.id, "", csvOf(svRow(3)), "idem-roster-0001", nil)
	require.Equal(t, "IDEMPOTENCY_KEY_REUSED", c.errCode(), "cùng khoá, tệp khác")
}

func TestRosterMissingFileAndColumn(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	res := r.upload(k.a.G, k.id, "", []byte("Họ và tên,MSSV\nA,B20DC00001\n"), "", nil)
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
	require.Contains(t, string(res.body), "MISSING_COLUMN")
	require.Contains(t, string(res.body), "Email")
	res = r.upload(k.a.G, k.id, "", []byte("\x7fELF\x02\x01\x01\x00"), "", nil)
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
	require.Contains(t, string(res.body), "UNSUPPORTED_FILE")
	require.Contains(t, string(res.body), "Không đọc được tệp. Hãy dùng CSV UTF-8 hoặc XLSX.")
	require.Equal(t, 0, r.count(`select count(*) from enrollments where course_id = $1 and joined_via = 'ROSTER'`, k.id))
}

func TestRosterXLSXImport(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	rep := r.importOK(k, xlsxOf(t, [][]any{{"Email", "Họ và tên", "MSSV"}, {"x1@example.test", "=HYPERLINK(\"http://evil\")", 20229001}, {"x2@example.test", "Lê B", "20229002"}}), "")
	require.Equal(t, 2, intOf(rep, "created_users"))
	require.Equal(t, `=HYPERLINK("http://evil")`, r.scalar(`select full_name from users where email = 'x1@example.test'`), "công thức lưu như chữ")
}

func TestRosterLongNameTruncated(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	r.importOK(k, csvOf("long@example.test|"+strings.Repeat("Ư", 150)+"|B20DC00055"), "")
	require.Equal(t, 100, r.count(`select char_length(full_name) from users where email = 'long@example.test'`))
}
