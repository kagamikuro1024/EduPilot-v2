package today_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/ingest"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/today"
)

// doc chèn một tài liệu của lớp c, tạo `age` trước đồng hồ giả. typ / status / visible tuỳ chỉnh qua extra (cột = giá trị SQL).
func (r *rig) newDoc(c cls, title string, age time.Duration, typ, status string, visible bool) string {
	r.t.Helper()
	return r.scalar(`insert into documents (course_id, title, type, status, visible_to_students, created_at) values ($1, $2, $3::document_type, $4::document_status, $5, $6) returning id::text`,
		c.id, title, typ, status, visible, r.clk.Now().Add(-age))
}

func (r *rig) openLibrary(s session, c cls, hdr map[string]string) resp {
	r.t.Helper()
	return r.do(req{method: http.MethodGet, path: "/courses/" + c.sid() + "/library", bearer: s.access, hdr: hdr})
}

// TestNewDocumentProvider — US-P2-11 AC15: «tài liệu mới» chỉ đếm tài liệu READY + hiện cho sinh viên + không ANSWER_KEY, vào lớp ≤ 72 giờ, của lớp ACTIVE; mở thư viện thì mục biến mất, tài liệu mới hơn thì hiện lại.
func TestNewDocumentProvider(t *testing.T) {
	r := newRig(t)
	a, b, pend, other := r.klass("762001"), r.klass("762002"), r.klass("762003"), r.klass("762004")
	u, s := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(a, u, "STUDENT", "ACTIVE", "", time.Hour)
	r.enrollAt(b, u, "STUDENT", "ACTIVE", "", time.Hour)
	r.enrollAt(pend, u, "STUDENT", "REMOVED", "", time.Hour) // PENDING sinh mục «Chờ giảng viên duyệt» (bậc 30) đứng trước; Viewer chỉ nạp lớp ACTIVE nên cả hai đều không đếm
	// bị loại: ANSWER_KEY, ẩn, quá 72 giờ, chưa READY (lớp A); tài liệu của lớp đã bị gỡ / không thuộc
	r.newDoc(a, "Đáp án", time.Hour, "ANSWER_KEY", "READY", false) // ràng buộc DB: ANSWER_KEY luôn ẩn; điều kiện loại thứ hai vẫn là type
	r.newDoc(a, "Tài liệu ẩn", time.Hour, "LECTURE", "READY", false)
	r.newDoc(a, "Tài liệu cũ", 73*time.Hour, "LECTURE", "READY", true)
	r.newDoc(a, "Đang xử lý", time.Hour, "LECTURE", "QUEUED", true)
	r.newDoc(pend, "Của lớp đã gỡ", time.Hour, "LECTURE", "READY", true)
	r.newDoc(other, "Của lớp khác", time.Hour, "LECTURE", "READY", true)
	require.Empty(t, kindItems(r.tget(s, "/me/today"), "NEW_DOCUMENT"), "không tài liệu nào đủ điều kiện")

	r.newDoc(a, "Chương 3 — TCP", 2*time.Hour, "LECTURE", "READY", true)
	items := kindItems(r.tget(s, "/me/today"), "NEW_DOCUMENT")
	require.Len(t, items, 1)
	require.Equal(t, "NEW_DOCUMENT:"+a.sid(), items[0]["id"])
	require.Equal(t, "1 tài liệu mới · lớp 762001", items[0]["title"])
	require.Equal(t, "Chương 3 — TCP", items[0]["reason"])
	require.Equal(t, "/library?course="+a.sid(), items[0]["href"])
	require.Equal(t, "normal", items[0]["urgency"])

	// tài liệu chia sẻ vào lớp A (do lớp khác sở hữu) cũng đếm, theo mốc chia sẻ; tiêu đề đếm tất cả
	shared := r.newDoc(other, "Bài đọc thêm", 100*time.Hour, "LECTURE", "READY", true)
	r.exec(`insert into document_courses (document_id, course_id, created_at) values ($1, $2, $3)`, shared, a.id, r.clk.Now().Add(-time.Hour))
	items = kindItems(r.tget(s, "/me/today"), "NEW_DOCUMENT")
	require.Len(t, items, 1)
	require.Equal(t, "2 tài liệu mới · lớp 762001", items[0]["title"])
	require.Equal(t, "Bài đọc thêm và 1 tài liệu khác", items[0]["reason"], "tên tài liệu mới nhất (theo mốc vào lớp)")

	// lớp B có thêm tài liệu → mỗi lớp một mục (theo từng lớp qua /courses/{id}/today)
	r.newDoc(b, "Slide tuần 2", time.Minute, "LECTURE", "READY", true)
	inB := kindItems(r.tget(s, "/courses/"+b.sid()+"/today"), "NEW_DOCUMENT")
	require.Len(t, inB, 1)
	require.Equal(t, "NEW_DOCUMENT:"+b.sid(), inB[0]["id"])

	// mở thư viện lớp A (200) → mục lớp A biến mất ở lần gọi today kế (cache đã xoá)
	require.Equal(t, http.StatusOK, r.openLibrary(s, a, nil).code)
	after := kindItems(r.tgetC(s, "/me/today"), "NEW_DOCUMENT")
	require.Len(t, after, 1)
	require.Equal(t, "NEW_DOCUMENT:"+b.sid(), after[0]["id"], "chỉ còn lớp B")
	require.Empty(t, kindItems(r.tgetC(s, "/courses/"+a.sid()+"/today"), "NEW_DOCUMENT"))

	// 304 cũng tính là đã mở
	first := r.openLibrary(s, b, nil)
	require.Equal(t, http.StatusOK, first.code)
	require.Equal(t, http.StatusNotModified, r.openLibrary(s, b, map[string]string{"If-None-Match": first.hdr.Get("ETag")}).code)
	require.Empty(t, kindItems(r.tgetC(s, "/me/today"), "NEW_DOCUMENT"))

	// tài liệu mới hơn lần mở → hiện lại, chỉ đếm tài liệu sau lần mở
	r.clk.Advance(time.Minute)
	r.newDoc(a, "Tuần 4", 0, "LECTURE", "READY", true)
	again := kindItems(r.tget(s, "/me/today"), "NEW_DOCUMENT")
	require.Len(t, again, 1)
	require.Equal(t, "1 tài liệu mới · lớp 762001", again[0]["title"])
	require.Equal(t, "Tuần 4", again[0]["reason"])
}

// TestNewDocumentSeenRedisDown — AC15: mất khoá «đã mở» (Redis lỗi / hết TTL) thì mục hiện lại, không lỗi.
func TestNewDocumentSeenRedisDown(t *testing.T) {
	r := newRig(t)
	a := r.klass("762011")
	u, s := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(a, u, "STUDENT", "ACTIVE", "", time.Hour)
	r.newDoc(a, "Chương 1", time.Hour, "LECTURE", "READY", true)
	require.Equal(t, http.StatusOK, r.openLibrary(s, a, nil).code)
	require.Empty(t, kindItems(r.tgetC(s, "/me/today"), "NEW_DOCUMENT"), "đã mở → không còn")
	svc := today.NewService(r.pool, r.rdb, r.clk, slog.New(slog.NewTextHandler(&strings.Builder{}, nil)), nil)
	require.NoError(t, r.rdb.Close()) // mọi lệnh Redis lỗi
	body, err := svc.Get(t.Context(), u.ID, today.RoleStudent, today.Scope{})
	require.NoError(t, err, "Redis chết: vẫn trả")
	require.Contains(t, string(body), "NEW_DOCUMENT", "mất khoá → hiện lại")
}

// TestNewDocumentRBAC — AC15: giảng viên, TA, Admin không có mục này dù lớp có tài liệu mới.
func TestNewDocumentRBAC(t *testing.T) {
	r := newRig(t)
	a := r.klass("762021")
	r.newDoc(a, "Chương 1", time.Hour, "LECTURE", "READY", true)
	gv, gs := r.person(store.UserRoleTEACHER, true)
	r.enrollAt(a, gv, "TEACHER", "ACTIVE", "", time.Hour)
	ta, ts := r.person(store.UserRoleTA, true)
	r.enrollAt(a, ta, "TA", "ACTIVE", "", time.Hour)
	_, as := r.person(store.UserRoleADMIN, true)
	for name, s := range map[string]session{"giảng viên": gs, "TA": ts, "Admin": as} {
		require.NotContains(t, string(r.tget(s, "/me/today").body), "NEW_DOCUMENT", name)
	}
}

// TestNewDocumentInvalidate — AC15: tài liệu sẵn sàng / chia sẻ / đổi cờ / xoá (outbox document.changed) xoá cache today của sinh viên lớp đó; TTL chỉ là lưới an toàn.
func TestNewDocumentInvalidate(t *testing.T) {
	r := newRig(t)
	a := r.klass("762031")
	u, s := r.person(store.UserRoleSTUDENT, true)
	r.enrollAt(a, u, "STUDENT", "ACTIVE", "", time.Hour)
	require.Empty(t, kindItems(r.tget(s, "/me/today"), "NEW_DOCUMENT")) // cache rỗng
	r.newDoc(a, "Vừa sẵn sàng", time.Minute, "LECTURE", "READY", true)
	require.Empty(t, kindItems(r.tgetC(s, "/me/today"), "NEW_DOCUMENT"), "chưa có sự kiện: cache cũ")
	payload, _ := json.Marshal(map[string]string{"course_id": a.sid()})
	require.NoError(t, today.Invalidator{Pool: r.pool, Redis: r.rdb}.Handle(t.Context(), outbox.Message{Topic: ingest.TopicDocumentChanged, Payload: payload}))
	require.Len(t, kindItems(r.tgetC(s, "/me/today"), "NEW_DOCUMENT"), 1, "document.changed xoá cache của sinh viên")
}

// TestNewDocumentQueryNotPerCourse — AC17: số truy vấn của sinh viên không tăng theo số lớp (1 → 5), mỗi lớp đều có tài liệu mới.
func TestNewDocumentQueryNotPerCourse(t *testing.T) {
	r := newRig(t)
	u1, s1 := r.person(store.UserRoleSTUDENT, true)
	u5, s5 := r.person(store.UserRoleSTUDENT, true)
	for i := range 5 {
		c := r.klass(fmt.Sprintf("76204%d", i))
		r.newDoc(c, fmt.Sprintf("Tài liệu %d", i), time.Hour, "LECTURE", "READY", true)
		r.enrollAt(c, u5, "STUDENT", "ACTIVE", "", time.Hour)
		if i == 0 {
			r.enrollAt(c, u1, "STUDENT", "ACTIVE", "", time.Hour)
		}
	}
	require.Len(t, kindItems(r.tget(s5, "/me/today"), "NEW_DOCUMENT"), 1, "recommended là việc đầu bảng")
	one := r.queries(s1, "/me/today")
	five := r.queries(s5, "/me/today")
	require.LessOrEqual(t, five-one, int64(0), "không N+1 theo số lớp")
}
