package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	appdb "github.com/edupilot/backend-go/internal/platform/db"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TestRosterInvalidateRegistered — US-P3-02 AC3: hai topic thành viên / nhập danh sách xoá ep:roster:{course} qua Chain trên dòng đăng ký sẵn có
// (đăng ký trùng topic sẽ panic nên newRegistry chạy được cũng là bằng chứng không đăng ký hai lần).
func TestRosterInvalidateRegistered(t *testing.T) {
	testutil.RequireContainers(t)
	d, _ := workerDeps(t)
	d.Cfg.DatabaseURL = testutil.MigratedPostgresURL(t) // inv.Handle đọc enrollments
	pool, err := appdb.NewPool(t.Context(), d.Cfg, d.Log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	d.DB = pool
	reg := newRegistry(d)
	for _, topic := range []string{"course.member_changed", "roster.imported"} {
		course := uuid.New()
		key := privacy.RosterKey(course)
		if err := d.Redis.Set(t.Context(), key, `[{"n":"Nguyễn Văn An"}]`, time.Hour).Err(); err != nil {
			t.Fatal(err)
		}
		h, ok := reg.Lookup(topic)
		if !ok {
			t.Fatalf("topic %s chưa đăng ký", topic)
		}
		payload, _ := json.Marshal(map[string]string{"course_id": course.String()})
		start := time.Now()
		if err := h(t.Context(), outbox.Message{Topic: topic, Payload: payload}); err != nil {
			t.Fatalf("%s: %v", topic, err)
		}
		if n, _ := d.Redis.Exists(t.Context(), key).Result(); n != 0 {
			t.Fatalf("%s: khoá roster còn sau sự kiện", topic)
		}
		if time.Since(start) > 5*time.Second {
			t.Fatalf("%s: vô hiệu quá 5 s", topic)
		}
	}
}
