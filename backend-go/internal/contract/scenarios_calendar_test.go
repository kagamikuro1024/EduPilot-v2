package contract

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

// calendarScenarios: thao tác 16–23 của SRS FEAT-docs-calendar 6 (lịch gộp, sự kiện, token và feed ICS — US-P8-03) với mọi status đã khai báo.
func (r *runner) calendarScenarios(x examRig) {
	base := "/api/v1/courses/" + x.cid + "/calendar"
	now := time.Now().UTC()
	q := "?from=" + url.QueryEscape(now.Add(-24*time.Hour).Format(time.RFC3339)) + "&to=" + url.QueryEscape(now.Add(30*24*time.Hour).Format(time.RFC3339))
	when := now.Add(72 * time.Hour).Format(time.RFC3339)
	ev := fmt.Sprintf(`{"type":"OTHER","title":"Nộp báo cáo","starts_at":%q,"location":"P.301"}`, when)

	// 16: danh sách.
	h, _ := r.must(call{method: "GET", path: base + q, token: x.sv}, 200)
	if tag := h.Get("ETag"); tag != "" {
		r.must(call{method: "GET", path: base + q, token: x.sv, headers: map[string]string{"If-None-Match": tag}}, 304)
	}
	r.must(call{method: "GET", path: base + q, token: x.gv}, 200)
	r.must(call{method: "GET", path: base + q}, 401)
	r.must(call{method: "GET", path: base + q, token: r.rig.token(r.t, uuid.NewString(), auth.RoleStudent)}, 403)                                                                              // không phải thành viên lớp
	r.must(call{method: "GET", path: base + "?from=" + url.QueryEscape(now.Format(time.RFC3339)) + "&to=" + url.QueryEscape(now.Add(63*24*time.Hour).Format(time.RFC3339)), token: x.sv}, 422) // RANGE_TOO_LARGE
	r.must(call{method: "GET", path: base + "?from=x&to=y", token: x.sv}, 422)

	// 17: tạo.
	_, b := r.must(call{method: "POST", path: base + "/events", token: x.ta, body: ev}, 201)
	var made struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal(b, &made); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "POST", path: base + "/events", body: ev}, 401)
	r.must(call{method: "POST", path: base + "/events", token: x.sv, body: ev}, 403)
	r.must(call{method: "POST", path: "/api/v1/courses/" + x.arch + "/calendar/events", token: x.gv, body: ev}, 409) // COURSE_ARCHIVED
	r.must(call{method: "POST", path: base + "/events", token: x.ta, body: strings.Replace(ev, `"OTHER"`, `"BAD"`, 1)}, 422)
	r.must(call{method: "POST", path: base + "/events", token: x.ta, body: fmt.Sprintf(`{"type":"OTHER","title":"x","starts_at":%q,"ends_at":%q}`, when, now.Format(time.RFC3339))}, 422) // EVENT_TIME_INVALID

	// 18: sửa.
	E := base + "/events/" + made.ID
	put := func(v int, title string) string {
		return fmt.Sprintf(`{"type":"EXAM","title":%q,"starts_at":%q,"version":%d}`, title, when, v)
	}
	r.must(call{method: "PUT", path: E, token: x.ta, body: put(made.Version, "Nộp báo cáo cuối kỳ")}, 200)
	r.must(call{method: "PUT", path: E, token: x.ta, body: put(made.Version, "Cũ")}, 409) // VERSION_CONFLICT
	r.must(call{method: "PUT", path: E, body: put(1, "x")}, 401)
	r.must(call{method: "PUT", path: E, token: x.sv, body: put(1, "x")}, 403)
	r.must(call{method: "PUT", path: base + "/events/" + uuid.NewString(), token: x.ta, body: put(1, "x")}, 404)
	r.must(call{method: "PUT", path: E, token: x.ta, body: put(made.Version+1, "")}, 422)
	r.must(call{method: "PUT", path: "/api/v1/courses/" + x.arch + "/calendar/events/" + made.ID, token: x.gv, body: put(1, "x")}, 409)

	// 21–23: token ICS, 20: feed.
	T := "/api/v1/me/calendar/ics-token"
	r.must(call{method: "GET", path: T}, 401)
	r.must(call{method: "POST", path: T}, 401)
	r.must(call{method: "DELETE", path: T}, 401)
	r.must(call{method: "GET", path: T, token: x.sv}, 200)
	_, tb := r.must(call{method: "POST", path: T, token: x.sv}, 201)
	var tok struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(tb, &tok); err != nil {
		r.t.Fatal(err)
	}
	u, err := url.Parse(tok.URL)
	if err != nil {
		r.t.Fatal(err)
	}
	F := "/api/v1/calendar/feed.ics?token=" + u.Query().Get("token")
	fh, _ := r.must(call{method: "GET", path: F}, 200)
	if tag := fh.Get("ETag"); tag != "" {
		r.must(call{method: "GET", path: F, headers: map[string]string{"If-None-Match": tag}}, 304)
	}
	r.must(call{method: "GET", path: "/api/v1/calendar/feed.ics?token=" + strings.Repeat("a", 43)}, 404)
	r.must(call{method: "DELETE", path: T, token: x.sv}, 204)
	r.must(call{method: "GET", path: F}, 404) // đã thu hồi
	got429 := false
	for range 80 {
		if st, _, _ := r.do(call{method: "GET", path: "/api/v1/calendar/feed.ics?token=" + strings.Repeat("b", 43)}); st == 429 {
			got429 = true
			break
		}
	}
	if !got429 {
		r.t.Fatal("feed.ics: không thấy 429 sau 80 lượt")
	}

	// 19: xoá.
	r.must(call{method: "DELETE", path: E}, 401)
	r.must(call{method: "DELETE", path: E, token: x.sv}, 403)
	r.must(call{method: "DELETE", path: E, token: x.ta}, 204)
	r.must(call{method: "DELETE", path: E, token: x.ta}, 404)
	r.must(call{method: "DELETE", path: "/api/v1/courses/" + x.arch + "/calendar/events/" + made.ID, token: x.gv}, 409)
}
