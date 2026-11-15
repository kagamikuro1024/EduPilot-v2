package today_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/today"
)

// US-P9-09 AC3 (L3): `practice.updated` xoá cache "Hôm nay" của CHÍNH sinh viên ở cả khoá `all` lẫn khoá theo lớp (`/courses/{id}/today`); cache của staff giữ nguyên.
func TestPracticeUpdatedInvalidatesCourseScope(t *testing.T) {
	r := newRig(t)
	c := r.klass("764002")
	su, _ := r.student("")
	r.enrollAt(c, su, "STUDENT", "ACTIVE", "", time.Hour)
	gv, _ := r.teaching(c)
	ctx := t.Context()

	keys := []string{today.CacheKey(su.ID, "all"), today.CacheKey(su.ID, c.sid()), today.CacheKey(gv.ID, c.sid())}
	for _, k := range keys {
		require.NoError(t, r.rdb.Set(ctx, k, "{}", time.Minute).Err())
	}
	payload, _ := json.Marshal(map[string]string{"user_id": su.ID.String(), "student_id": su.ID.String(), "course_id": c.sid()})
	require.NoError(t, today.Invalidator{Pool: r.pool, Redis: r.rdb}.Handle(ctx, outbox.Message{Topic: "practice.updated", Payload: payload}))

	for _, k := range keys[:2] {
		n, err := r.rdb.Exists(ctx, k).Result()
		require.NoError(t, err)
		require.Zero(t, n, k)
	}
	n, err := r.rdb.Exists(ctx, keys[2]).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, n, "cache của giảng viên không bị xoá mỗi lần sinh viên luyện")
}
