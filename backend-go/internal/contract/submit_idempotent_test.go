package contract

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

// TestSubmitIdempotent — P-11-42 (TC-P7-03-G03): POST /courses/{id}/assignments/{aid}/submit.
// Cùng Idempotency-Key + cùng thân ⇒ cùng phản hồi, đúng MỘT version và MỘT outbox submission.created;
// cùng khoá + thân khác ⇒ 422 IDEMPOTENCY_KEY_REUSED (không nộp thêm gì).
func TestSubmitIdempotent(t *testing.T) {
	r := &runner{t: t, rig: getRig(t), cache: map[string]string{}}
	r.prod, r.test = mustLoad(t)
	ctx := context.Background()
	db := r.rig.deps.DB
	mk := func(role string) uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(ctx, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Người Thử', $2::user_role, 'ACTIVE', 'x', now()) returning id`,
			"ct-si-"+uuid.NewString()[:8]+"@example.test", role).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	gv, sv := mk("TEACHER"), mk("STUDENT")
	var course uuid.UUID
	if err := db.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp nộp bài', '2026-2027-HK1', $2, $3) returning id`,
		"SI-"+uuid.NewString()[:6], "SJ"+strings.ToUpper(strings.NewReplacer("0", "X", "1", "Y", "O", "Z", "I", "W", "L", "V").Replace(uuid.NewString()[:5])), gv).Scan(&course); err != nil {
		t.Fatal(err)
	}
	for u, role := range map[uuid.UUID]string{gv: "TEACHER", sv: "STUDENT"} {
		if _, err := db.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, $3, 'ACTIVE', 'ADMIN')`, course, u, role); err != nil {
			t.Fatal(err)
		}
	}
	gvTok, svTok := r.rig.token(t, gv.String(), auth.RoleTeacher), r.rig.token(t, sv.String(), auth.RoleStudent)
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": uuid.NewString()} }
	base := "/api/v1/courses/" + course.String() + "/assignments"

	due := time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)
	_, b := r.must(call{method: "POST", path: base, token: gvTok, headers: idem(), body: `{"title":"BT-SI","submission_code":"BTSI","due_at":"` + due + `","max_score":"10","channels":["IN_APP"],"rubric":[{"name":"Một","max_score":"10"}]}`}, 201)
	var a struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(b, &a)
	A := base + "/" + a.ID
	r.must(call{method: "POST", path: A + "/publish", token: gvTok, headers: idem()}, 200)

	data := []byte("Bài làm của tôi.\n")
	_, b = r.must(call{method: "POST", path: A + "/submit/presign", token: svTok, body: fmt.Sprintf(`{"filename":"bai.txt","size_bytes":%d,"sha256":"%x"}`, len(data), sha256.Sum256(data))}, 200)
	var p struct {
		UploadID string            `json:"upload_id"`
		URL      string            `json:"url"`
		Headers  map[string]string `json:"headers"`
	}
	_ = json.Unmarshal(b, &p)
	req, _ := http.NewRequest(http.MethodPut, p.URL, bytes.NewReader(data))
	for k, v := range p.Headers {
		if k != "Content-Length" {
			req.Header.Set(k, v)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT bài nộp: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT bài nộp → %d", resp.StatusCode)
	}

	key := map[string]string{"Idempotency-Key": uuid.NewString()}
	same := `{"upload_id":"` + p.UploadID + `"}`
	_, b1 := r.must(call{method: "POST", path: A + "/submit", token: svTok, headers: key, body: same}, 201)
	_, b2 := r.must(call{method: "POST", path: A + "/submit", token: svTok, headers: key, body: same}, 201)
	if string(b1) != string(b2) {
		t.Fatalf("cùng khoá + cùng thân phải phát lại đúng phản hồi: %s ≠ %s", b1, b2)
	}
	var out struct {
		Submission struct {
			ID      string `json:"id"`
			Version int    `json:"version"`
		} `json:"submission"`
	}
	_ = json.Unmarshal(b1, &out)
	var versions, outbox int
	if err := db.QueryRow(ctx, `select count(*) from submissions where assignment_id = $1 and student_id = $2`, a.ID, sv).Scan(&versions); err != nil || versions != 1 || out.Submission.Version != 1 {
		t.Fatalf("submissions = %d, version phản hồi = %d (cần 1 và 1): %v", versions, out.Submission.Version, err)
	}
	if err := db.QueryRow(ctx, `select count(*) from outbox where topic = 'submission.created' and payload->>'submission_id' = $1`, out.Submission.ID).Scan(&outbox); err != nil || outbox != 1 {
		t.Fatalf("outbox submission.created = %d (cần 1): %v", outbox, err)
	}

	// cùng khoá, thân khác → 422 IDEMPOTENCY_KEY_REUSED; không nộp thêm
	_, rb := r.must(call{method: "POST", path: A + "/submit", token: svTok, headers: key, body: `{"upload_id":"` + uuid.NewString() + `"}`}, 422)
	if !strings.Contains(string(rb), "IDEMPOTENCY_KEY_REUSED") {
		t.Fatalf("thân khác cùng khoá: %s (cần IDEMPOTENCY_KEY_REUSED)", rb)
	}
	if err := db.QueryRow(ctx, `select count(*) from submissions where assignment_id = $1 and student_id = $2`, a.ID, sv).Scan(&versions); err != nil || versions != 1 {
		t.Fatalf("sau 422 submissions = %d (cần 1): %v", versions, err)
	}
}
