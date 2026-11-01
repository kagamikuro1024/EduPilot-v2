//go:build testroutes

package httpapi

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

type asgCourse struct {
	id                  string
	teacher, student    string
	tTok, sTok, adminTk string
}

// tAsgCourse dựng lớp + giảng viên + sinh viên ACTIVE trên ngăn xếp dùng chung.
func tAsgCourse(t *testing.T) asgCourse {
	t.Helper()
	_, d := tStack(t)
	ctx := t.Context()
	user := func(role string) string {
		var id string
		if err := d.DB.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'Người Thử', $2::user_role) returning id`, strings.ToLower(role)+"."+uuid.NewString()[:8]+"@example.test", role).Scan(&id); err != nil {
			t.Fatalf("tạo người dùng: %v", err)
		}
		return id
	}
	c := asgCourse{teacher: user("TEACHER"), student: user("STUDENT")}
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, 7)
	for i := range b {
		b[i] = alpha[int(id[i])%len(alpha)]
	}
	if err := d.DB.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'ANM', '2026-2027-HK1', $2, $3) returning id`,
		"G-"+id.String()[:8], string(b), c.teacher).Scan(&c.id); err != nil {
		t.Fatalf("tạo lớp: %v", err)
	}
	for u, r := range map[string]string{c.teacher: "TEACHER", c.student: "STUDENT"} {
		if _, err := d.DB.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, $3, 'ACTIVE', 'ADMIN')`, c.id, u, r); err != nil {
			t.Fatalf("ghi danh: %v", err)
		}
	}
	c.tTok, c.sTok, c.adminTk = tTokenRole(t, c.teacher, auth.RoleTeacher), tTokenRole(t, c.student, auth.RoleStudent), tTokenRole(t, uuid.NewString(), auth.RoleAdmin)
	return c
}

func asgBody(code string) string {
	return `{"title":"BT03","submission_code":"` + code + `","due_at":"` + time.Now().UTC().Add(72*time.Hour).Format(time.RFC3339) + `","max_score":"10","reference_answer":"CANARY-BT03-ANSWER ` + strings.Repeat("x", 60) +
		`","rubric":[{"name":"Khái niệm","max_score":"4","guidance":"CANARY-GUIDANCE"},{"name":"Ví dụ","max_score":"6"}]}`
}

func TestCreateIdempotent(t *testing.T) {
	c := tAsgCourse(t)
	path, key := "/api/v1/courses/"+c.id+"/assignments", tIdemKey(t)
	first := tDo(t, http.MethodPost, path, c.tTok, asgBody("BT03"), "Idempotency-Key", key)
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("lần 1 → %d: %s", first.StatusCode, tBody(t, first))
	}
	id1 := tJSON(t, first)["id"]
	second := tDo(t, http.MethodPost, path, c.tTok, asgBody("BT03"), "Idempotency-Key", key)
	if second.StatusCode != http.StatusCreated || tJSON(t, second)["id"] != id1 {
		t.Fatalf("lần 2 phải phát lại đúng phản hồi (status %d)", second.StatusCode)
	}
	_, d := tStack(t)
	var n int
	if err := d.DB.QueryRow(t.Context(), `select count(*) from assignments where course_id = $1`, c.id).Scan(&n); err != nil || n != 1 {
		t.Fatalf("còn %d bài (cần 1): %v", n, err)
	}
	if r := tDo(t, http.MethodPost, path, c.tTok, asgBody("BT04")); r.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("thiếu Idempotency-Key → %d (cần 422)", r.StatusCode)
	}
}

func TestPublishIdempotent(t *testing.T) {
	c := tAsgCourse(t)
	base := "/api/v1/courses/" + c.id + "/assignments"
	id := tJSON(t, tDo(t, http.MethodPost, base, c.tTok, asgBody("BT03"), "Idempotency-Key", tIdemKey(t)))["id"].(string) //nolint:forcetypeassert // id là chuỗi
	key := tIdemKey(t)
	for i := range 2 {
		if r := tDo(t, http.MethodPost, base+"/"+id+"/publish", c.tTok, "", "Idempotency-Key", key); r.StatusCode != http.StatusOK {
			t.Fatalf("lần %d → %d (cần 200, lần 2 là phát lại): %s", i+1, r.StatusCode, tBody(t, r))
		}
	}
	_, d := tStack(t)
	var n int
	if err := d.DB.QueryRow(t.Context(), `select count(*) from outbox where topic = 'assignment.published' and payload->>'assignment_id' = $1`, id).Scan(&n); err != nil || n != 1 {
		t.Fatalf("outbox %d (cần 1): %v", n, err)
	}
}

// TestStudentSeesNoAnswerOverHTTP — AC10: canary đặt ở đáp án chuẩn và guidance không xuất hiện ở bất kỳ phản hồi nào gửi sinh viên; ADMIN không qua.
func TestStudentSeesNoAnswerOverHTTP(t *testing.T) {
	c := tAsgCourse(t)
	base := "/api/v1/courses/" + c.id + "/assignments"
	id := tJSON(t, tDo(t, http.MethodPost, base, c.tTok, asgBody("BT03"), "Idempotency-Key", tIdemKey(t)))["id"].(string) //nolint:forcetypeassert // id là chuỗi
	if r := tDo(t, http.MethodGet, base+"/"+id, c.sTok, ""); r.StatusCode != http.StatusNotFound {
		t.Fatalf("bài nháp với sinh viên → %d (cần 404)", r.StatusCode)
	}
	if r := tDo(t, http.MethodPost, base+"/"+id+"/publish", c.tTok, "", "Idempotency-Key", tIdemKey(t)); r.StatusCode != http.StatusOK {
		t.Fatalf("công bố → %d", r.StatusCode)
	}
	for _, p := range []string{base, base + "/" + id} {
		r := tDo(t, http.MethodGet, p, c.sTok, "")
		body := string(tBody(t, r))
		if r.StatusCode != http.StatusOK || strings.Contains(body, "CANARY") || strings.Contains(body, "guidance") || strings.Contains(body, "reference_answer") || strings.Contains(body, "submission_code") {
			t.Fatalf("GET %s → %d, lộ dữ liệu nhân viên: %s", p, r.StatusCode, body)
		}
	}
	if r := tDo(t, http.MethodGet, base+"/"+id, c.tTok, ""); !strings.Contains(string(tBody(t, r)), "CANARY-BT03-ANSWER") {
		t.Fatal("giảng viên phải thấy đáp án chuẩn")
	}
	if r := tDo(t, http.MethodGet, base+"/"+id, c.adminTk, ""); r.StatusCode != http.StatusForbidden {
		t.Fatalf("ADMIN → %d (cần 403)", r.StatusCode)
	}
}

// pubAsgCourse: lớp + một bài đã công bố; trả (course, assignmentID).
func pubAsgCourse(t *testing.T) (asgCourse, string) {
	t.Helper()
	c := tAsgCourse(t)
	base := "/api/v1/courses/" + c.id + "/assignments"
	id := tJSON(t, tDo(t, http.MethodPost, base, c.tTok, asgBody("BT03"), "Idempotency-Key", tIdemKey(t)))["id"].(string) //nolint:forcetypeassert // id là chuỗi
	if r := tDo(t, http.MethodPost, base+"/"+id+"/publish", c.tTok, "", "Idempotency-Key", tIdemKey(t)); r.StatusCode != http.StatusOK {
		t.Fatalf("công bố → %d", r.StatusCode)
	}
	return c, id
}

// TestStudentRoutesNoStore — mọi route sinh viên của bài tập trả `Cache-Control: no-store` (TLR-17).
func TestStudentRoutesNoStore(t *testing.T) {
	c, id := pubAsgCourse(t)
	base := "/api/v1/courses/" + c.id + "/assignments"
	for _, p := range []string{base, base + "/" + id, base + "/" + id + "/mine"} {
		r := tDo(t, http.MethodGet, p, c.sTok, "")
		if r.StatusCode != http.StatusOK || r.Header.Get("Cache-Control") != "no-store" {
			t.Fatalf("GET %s → %d, Cache-Control %q", p, r.StatusCode, r.Header.Get("Cache-Control"))
		}
	}
	body := `{"filename":"a.txt","size_bytes":5,"sha256":"` + strings.Repeat("a", 64) + `"}`
	if r := tDo(t, http.MethodPost, base+"/"+id+"/submit/presign", c.sTok, body); r.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("presign Cache-Control %q (status %d)", r.Header.Get("Cache-Control"), r.StatusCode)
	}
}

// TestMineIgnoresStudentParam — `student_id` (query hoặc thân) không làm đổi người được đọc: luôn là người trong JWT.
func TestMineIgnoresStudentParam(t *testing.T) {
	c, id := pubAsgCourse(t)
	_, d := tStack(t)
	var other string
	if err := d.DB.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, 'Bạn', 'STUDENT') returning id`, "o."+uuid.NewString()[:8]+"@example.test").Scan(&other); err != nil {
		t.Fatal(err)
	}
	if _, err := d.DB.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, c.id, other); err != nil {
		t.Fatal(err)
	}
	if _, err := d.DB.Exec(t.Context(), `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method)
		values ($1, $2, $3, 'IN_APP', 1, 'k', 'cua-nguoi-khac.txt', 1, 'text/plain', 'x', 'MATCHED', 'IN_APP')`, c.id, id, other); err != nil {
		t.Fatal(err)
	}
	r := tDo(t, http.MethodGet, "/api/v1/courses/"+c.id+"/assignments/"+id+"/mine?student_id="+other, c.sTok, "")
	body := string(tBody(t, r))
	if r.StatusCode != http.StatusOK || strings.Contains(body, "cua-nguoi-khac") || !strings.Contains(body, `"state":"NOT_SUBMITTED"`) {
		t.Fatalf("mine?student_id → %d: %s", r.StatusCode, body)
	}
}

// TestStudentCannotCallStaffRoutes — sinh viên gọi route ghi của nhân viên → 403; TA / TEACHER / ADMIN gọi route sinh viên → 403.
func TestStudentCannotCallStaffRoutes(t *testing.T) {
	c, id := pubAsgCourse(t)
	base := "/api/v1/courses/" + c.id + "/assignments"
	k := func() []string { return []string{"Idempotency-Key", tIdemKey(t)} }
	for _, call := range []struct{ m, p, b string }{
		{http.MethodPost, base, asgBody("BT09")}, {http.MethodPut, base + "/" + id, `{"version":1,"title":"x"}`}, {http.MethodDelete, base + "/" + id, ""},
		{http.MethodPost, base + "/" + id + "/publish", ""}, {http.MethodPut, base + "/" + id + "/rubric", `{"version":1,"criteria":[]}`}, {http.MethodPost, base + "/" + id + "/rubric/suggest", ""},
	} {
		if r := tDo(t, call.m, call.p, c.sTok, call.b, k()...); r.StatusCode != http.StatusForbidden {
			t.Fatalf("sinh viên %s %s → %d (cần 403)", call.m, call.p, r.StatusCode)
		}
	}
	for _, tok := range []string{c.tTok, c.adminTk} {
		if r := tDo(t, http.MethodGet, base+"/"+id+"/mine", tok, ""); r.StatusCode != http.StatusForbidden {
			t.Fatalf("mine với vai không phải sinh viên → %d (cần 403)", r.StatusCode)
		}
	}
}

// TestUnmatchedListRBAC / TestUploadRBAC / TestSignedGetChecksRole — route nhân viên về bài nộp: TEACHER 200; STUDENT, ADMIN, giảng viên lớp khác 403 (kể cả khi tệp / vé không tồn tại: guard chạy trước).
func TestUnmatchedListRBAC(t *testing.T) {
	c, id := pubAsgCourse(t)
	list := "/api/v1/courses/" + c.id + "/submissions?match_status=UNMATCHED"
	if r := tDo(t, http.MethodGet, list, c.tTok, ""); r.StatusCode != http.StatusOK || r.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("giảng viên → %d, Cache-Control %q", r.StatusCode, r.Header.Get("Cache-Control"))
	}
	outsider := tTokenRole(t, uuid.NewString(), auth.RoleTeacher)
	for name, tok := range map[string]string{"sinh viên": c.sTok, "admin": c.adminTk, "giảng viên lớp khác": outsider} {
		if r := tDo(t, http.MethodGet, list, tok, ""); r.StatusCode != http.StatusForbidden {
			t.Fatalf("%s → %d (cần 403)", name, r.StatusCode)
		}
		if r := tDo(t, http.MethodPut, "/api/v1/courses/"+c.id+"/submissions/"+uuid.NewString()+"/match", tok,
			`{"assignment_id":"`+id+`","student_id":"`+c.student+`","version":1}`); r.StatusCode != http.StatusForbidden {
			t.Fatalf("gán tay: %s → %d (cần 403)", name, r.StatusCode)
		}
	}
}

func TestUploadRBAC(t *testing.T) {
	c, id := pubAsgCourse(t)
	path := "/api/v1/courses/" + c.id + "/assignments/" + id + "/submissions/upload"
	body := `{"upload_id":"` + uuid.NewString() + `"}`
	outsider := tTokenRole(t, uuid.NewString(), auth.RoleTeacher)
	for name, tok := range map[string]string{"sinh viên": c.sTok, "admin": c.adminTk, "giảng viên lớp khác": outsider} {
		if r := tDo(t, http.MethodPost, path, tok, body, "Idempotency-Key", tIdemKey(t)); r.StatusCode != http.StatusForbidden {
			t.Fatalf("%s → %d (cần 403)", name, r.StatusCode)
		}
	}
	// giảng viên của lớp qua guard: vé không có → 404 UPLOAD_NOT_FOUND (hoặc 503 nếu ngăn xếp thử không có object storage), không phải 403
	if r := tDo(t, http.MethodPost, path, c.tTok, body, "Idempotency-Key", tIdemKey(t)); r.StatusCode == http.StatusForbidden || r.StatusCode == http.StatusUnauthorized {
		t.Fatalf("giảng viên lớp bị chặn: %d", r.StatusCode)
	}
}

func TestSignedGetChecksRole(t *testing.T) {
	c, _ := pubAsgCourse(t)
	path := "/api/v1/courses/" + c.id + "/submissions/" + uuid.NewString() + "/file"
	for name, tok := range map[string]string{"sinh viên": c.sTok, "admin": c.adminTk, "giảng viên lớp khác": tTokenRole(t, uuid.NewString(), auth.RoleTeacher)} {
		if r := tDo(t, http.MethodGet, path, tok, ""); r.StatusCode != http.StatusForbidden {
			t.Fatalf("%s → %d (cần 403): URL ký chỉ phát sau khi kiểm vai", name, r.StatusCode)
		}
	}
	if r := tDo(t, http.MethodGet, path, c.tTok, ""); r.StatusCode != http.StatusNotFound {
		t.Fatalf("giảng viên, id lạ → %d (cần 404)", r.StatusCode)
	}
}

// TestReviewRBACMatrix — US-P7-08 AC8: GET/PUT/review kết quả + hàng chờ: TEACHER qua; STUDENT, ADMIN, giảng viên lớp khác 403; chưa đăng nhập 401. TA qua đã được kiểm ở kịch bản contract.
func TestReviewRBACMatrix(t *testing.T) {
	c, _ := pubAsgCourse(t)
	rid := uuid.NewString()
	base := "/api/v1/courses/" + c.id
	outsider := tTokenRole(t, uuid.NewString(), auth.RoleTeacher)
	calls := []struct{ method, path, body string }{
		{http.MethodGet, base + "/submissions", ""},
		{http.MethodGet, base + "/grading-results/" + rid, ""},
		{http.MethodPut, base + "/grading-results/" + rid, `{"version":1}`},
		{http.MethodPost, base + "/grading-results/" + rid + "/review", `{"version":1}`},
	}
	for _, cl := range calls {
		for name, tok := range map[string]string{"sinh viên": c.sTok, "admin": c.adminTk, "giảng viên lớp khác": outsider} {
			if r := tDo(t, cl.method, cl.path, tok, cl.body); r.StatusCode != http.StatusForbidden {
				t.Fatalf("%s %s: %s → %d (cần 403)", cl.method, cl.path, name, r.StatusCode)
			}
		}
		if r := tDo(t, cl.method, cl.path, "", cl.body); r.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s không token → %d (cần 401)", cl.method, cl.path, r.StatusCode)
		}
	}
	if r := tDo(t, http.MethodGet, base+"/submissions", c.tTok, ""); r.StatusCode != http.StatusOK || r.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("giảng viên: hàng chờ → %d, Cache-Control %q", r.StatusCode, r.Header.Get("Cache-Control"))
	}
	if r := tDo(t, http.MethodGet, base+"/grading-results/"+rid, c.tTok, ""); r.StatusCode != http.StatusNotFound {
		t.Fatalf("giảng viên, id lạ → %d (cần 404)", r.StatusCode)
	}
}

// TestSignedGetFileRole — AC8: URL ký tệp gốc chỉ phát sau khi kiểm vai (cùng kiểm với TestSignedGetChecksRole, tên theo US-P7-08).
func TestSignedGetFileRole(t *testing.T) { TestSignedGetChecksRole(t) }
