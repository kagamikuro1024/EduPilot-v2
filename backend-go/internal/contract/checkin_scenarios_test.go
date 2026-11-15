package contract

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// checkinScenarios gọi 12 thao tác điểm danh tự phục vụ (FEAT-attendance-selfcheck 6) với mọi status khai báo (trừ các mục miễn ở exempt.go).
// c2: lớp ACTIVE có sinh viên svIn đã vào lớp; cid: lớp đã lưu trữ.
func (r *runner) checkinScenarios(db *pgxpool.Pool, c2, gv, ta, svIn, admin string) {
	t := r.t
	ctx := context.Background()
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "ct-" + uuid.NewString()} }
	course := "/api/v1/courses/" + c2
	if _, err := db.Exec(ctx, `update courses set settings = settings || '{"attendance":{"self_checkin":true,"rotate_seconds":15,"late_after_min":15,"require":"WEB"}}'::jsonb where id = $1`, c2); err != nil {
		t.Fatal(err)
	}
	var s4, s5 string // hai buổi mới, chưa có dòng điểm danh nào
	for i, dst := range []*string{&s4, &s5} {
		if err := db.QueryRow(ctx, `insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, $2::int, now() + ($2::int * interval '1 day'), now() + ($2::int * interval '1 day') + interval '2 hours') returning id::text`, c2, 90+i).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	ws := course + "/sessions/" + s4 + "/checkin-windows"
	id := func(b []byte) string {
		var x struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(b, &x)
		return x.ID
	}

	// mở cửa sổ
	r.must(call{method: "POST", path: ws, headers: idem(), body: `{}`}, 401)
	r.must(call{method: "POST", path: ws, token: svIn, headers: idem(), body: `{}`}, 403)
	r.must(call{method: "POST", path: ws, token: admin, headers: idem(), body: `{}`}, 403)
	r.must(call{method: "POST", path: course + "/sessions/" + uuid.NewString() + "/checkin-windows", token: gv, headers: idem(), body: `{}`}, 404)
	r.must(call{method: "POST", path: ws, token: gv, headers: idem(), body: `{"minutes":0}`}, 422)
	_, ob := r.must(call{method: "POST", path: ws, token: gv, headers: idem(), body: `{"minutes":30}`}, 201)
	w1 := id(ob)
	r.must(call{method: "POST", path: ws, token: ta, headers: idem(), body: `{}`}, 409) // WINDOW_ALREADY_OPEN

	// danh sách + màn chiếu
	r.must(call{method: "GET", path: ws, token: gv}, 200)
	r.must(call{method: "GET", path: ws}, 401)
	r.must(call{method: "GET", path: ws, token: svIn}, 403)
	r.must(call{method: "GET", path: course + "/sessions/" + uuid.NewString() + "/checkin-windows", token: gv}, 404)
	disp := ws + "/" + w1 + "/display"
	_, db1 := r.must(call{method: "GET", path: disp, token: gv}, 200)
	r.must(call{method: "GET", path: disp}, 401)
	r.must(call{method: "GET", path: disp, token: svIn}, 403)
	r.must(call{method: "GET", path: ws + "/" + uuid.NewString() + "/display", token: gv}, 404)
	r.exhaust(call{method: "GET", path: disp, token: ta}) // 429: hạn mức theo người, nên dùng TA, không phải giảng viên
	var dsp struct {
		QR   string `json:"qr_url"`
		Code string `json:"short_code"`
	}
	_ = json.Unmarshal(db1, &dsp)
	_, token, _ := strings.Cut(dsp.QR, "#")

	// sinh viên điểm danh
	ck := course + "/checkins"
	r.must(call{method: "POST", path: ck, body: `{"code":"AAAAAA"}`}, 401)
	r.must(call{method: "POST", path: ck, token: gv, body: `{"code":"AAAAAA"}`}, 403)
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/checkins", token: svIn, body: `{"code":"AAAAAA"}`}, 404)
	r.must(call{method: "POST", path: ck, token: svIn, body: `{}`}, 422)
	r.must(call{method: "POST", path: ck, token: svIn, body: `{"token":"` + token + `"}`}, 201)
	r.must(call{method: "POST", path: ck, token: svIn, body: `{"token":"` + token + `"}`}, 200)

	// headcount, gọi tên, danh sách lượt, bỏ cờ
	pw := course + "/checkin-windows/" + w1
	r.must(call{method: "PATCH", path: pw, token: gv, body: `{"headcount":30}`}, 200)
	r.must(call{method: "PATCH", path: pw, body: `{"headcount":30}`}, 401)
	r.must(call{method: "PATCH", path: pw, token: svIn, body: `{"headcount":30}`}, 403)
	r.must(call{method: "PATCH", path: course + "/checkin-windows/" + uuid.NewString(), token: gv, body: `{"headcount":30}`}, 404)
	r.must(call{method: "PATCH", path: pw, token: gv, body: `{"headcount":5000}`}, 422)
	r.must(call{method: "POST", path: pw + "/spot-check", token: ta, body: `{"n":3}`}, 200)
	r.must(call{method: "POST", path: pw + "/spot-check", body: `{}`}, 401)
	r.must(call{method: "POST", path: pw + "/spot-check", token: svIn, body: `{}`}, 403)
	r.must(call{method: "POST", path: course + "/checkin-windows/" + uuid.NewString() + "/spot-check", token: gv, body: `{}`}, 404)
	r.must(call{method: "POST", path: pw + "/spot-check", token: gv, body: `{"n":0}`}, 422)
	lc := course + "/sessions/" + s4 + "/checkins"
	_, cb := r.must(call{method: "GET", path: lc, token: gv}, 200)
	r.must(call{method: "GET", path: lc}, 401)
	r.must(call{method: "GET", path: lc, token: svIn}, 403)
	r.must(call{method: "GET", path: course + "/sessions/" + uuid.NewString() + "/checkins", token: gv}, 404)
	r.must(call{method: "GET", path: lc + "?limit=101", token: gv}, 422)
	var cl struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(cb, &cl); err != nil || len(cl.Items) != 1 {
		t.Fatalf("cần đúng 1 lượt, có %d (%v): %s", len(cl.Items), err, cb)
	}
	cf := course + "/checkins/" + cl.Items[0].ID + "/clear-flags"
	r.must(call{method: "POST", path: cf, token: ta}, 200)
	r.must(call{method: "POST", path: cf}, 401)
	r.must(call{method: "POST", path: cf, token: svIn}, 403)
	r.must(call{method: "POST", path: course + "/checkins/" + uuid.NewString() + "/clear-flags", token: gv}, 404)

	// đóng; từ đây màn chiếu / điểm danh trả 409; đánh vắng
	ab := course + "/sessions/" + s4 + "/absent-unchecked"
	r.must(call{method: "POST", path: ab, token: gv, headers: idem()}, 409) // WINDOW_OPEN
	r.must(call{method: "POST", path: pw + "/close"}, 401)
	r.must(call{method: "POST", path: pw + "/close", token: svIn}, 403)
	r.must(call{method: "POST", path: course + "/checkin-windows/" + uuid.NewString() + "/close", token: gv}, 404)
	r.must(call{method: "POST", path: pw + "/close", token: gv}, 200)
	r.must(call{method: "GET", path: disp, token: gv}, 409) // WINDOW_CLOSED
	r.must(call{method: "POST", path: ck, token: svIn, body: `{"token":"` + token + `"}`}, 409)
	r.must(call{method: "POST", path: ab, token: gv, headers: idem()}, 200)
	r.must(call{method: "POST", path: ab, headers: idem()}, 401)
	r.must(call{method: "POST", path: ab, token: svIn, headers: idem()}, 403)
	r.must(call{method: "POST", path: course + "/sessions/" + uuid.NewString() + "/absent-unchecked", token: gv, headers: idem()}, 404)
	r.must(call{method: "POST", path: course + "/sessions/" + s5 + "/absent-unchecked", token: gv, headers: idem()}, 409) // NO_WINDOW

	// 5 mã sai liên tiếp → 429 (cuối cùng vì làm sinh viên bị khoá một phút)
	for range 5 {
		r.must(call{method: "POST", path: ck, token: svIn, body: `{"code":"ZZZZZZ"}`}, 422)
	}
	r.must(call{method: "POST", path: ck, token: svIn, body: `{"code":"ZZZZZZ"}`}, 429)

	// thiết bị Android: nonce (200), 5 lần / phút (429); đăng ký cần chuỗi chứng thực thật nên chỉ gọi các nhánh lỗi (200 / 201 / 503: exempt.go)
	dc := "/api/v1/me/devices/challenge"
	r.must(call{method: "POST", path: dc}, 401)
	r.must(call{method: "POST", path: dc, token: gv}, 403)
	r.must(call{method: "POST", path: dc, token: svIn}, 200)
	r.exhaust(call{method: "POST", path: dc, token: svIn})
	dv := "/api/v1/me/devices"
	r.must(call{method: "POST", path: dv, body: `{}`}, 401)
	r.must(call{method: "POST", path: dv, token: gv, body: `{}`}, 403)
	r.must(call{method: "POST", path: dv, token: svIn, body: `{}`}, 422)
	// openapi `attestation_chain` minItems 2 / maxItems 6 (D-AS-03): 1 và 7 chứng chỉ → 422 trước khi đụng nonce
	r.must(call{method: "POST", path: dv, token: svIn, body: `{"kind":"ANDROID_KEY","public_key":"AAAA","attestation_chain":["AAAA"]}`}, 422)
	r.must(call{method: "POST", path: dv, token: svIn, body: `{"kind":"ANDROID_KEY","public_key":"AAAA","attestation_chain":["AAAA","AAAA","AAAA","AAAA","AAAA","AAAA","AAAA"]}`}, 422)

	// lớp lưu trữ: mọi thao tác ghi của Staff → 409 COURSE_ARCHIVED (cửa sổ w2 còn mở lúc lưu trữ)
	_, o2 := r.must(call{method: "POST", path: course + "/sessions/" + s5 + "/checkin-windows", token: gv, headers: idem(), body: `{}`}, 201)
	w2 := id(o2)
	if _, err := db.Exec(ctx, `update courses set status = 'ARCHIVED', archived_at = now(), join_enabled = false where id = $1`, c2); err != nil {
		t.Fatal(err)
	}
	r.must(call{method: "POST", path: course + "/sessions/" + s5 + "/checkin-windows", token: gv, headers: idem(), body: `{}`}, 409)
	r.must(call{method: "PATCH", path: course + "/checkin-windows/" + w2, token: gv, body: `{"headcount":1}`}, 409)
	r.must(call{method: "POST", path: course + "/checkin-windows/" + w2 + "/close", token: gv}, 409)
	r.must(call{method: "POST", path: cf, token: gv}, 409)
	r.must(call{method: "POST", path: course + "/sessions/" + s5 + "/absent-unchecked", token: gv, headers: idem()}, 409)
	if _, err := db.Exec(ctx, `update courses set status = 'ACTIVE', archived_at = null where id = $1`, c2); err != nil {
		t.Fatal(err)
	}
}
