package privacy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/testutil"
)

type fx struct {
	pool   *pgxpool.Pool
	rdb    *appredis.Client
	course uuid.UUID
	logs   *bytes.Buffer
	log    *slog.Logger
}

func newFx(t *testing.T) *fx {
	t.Helper()
	testutil.RequireContainers(t)
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	warmRedis(t, rdb)
	f := &fx{pool: pool, rdb: rdb, logs: &bytes.Buffer{}}
	f.log = slog.New(slog.NewJSONHandler(f.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	uid := uuid.New()
	id := uid.String()[:6]
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789" // mã tham gia không có 0 O 1 I L
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(uid[i])%len(alpha)]
	}
	var teacher uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'Giảng Viên Thử', 'TEACHER') returning id`, "gv."+id+"@example.test").Scan(&teacher))
	require.NoError(t, pool.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`,
		"PV-"+strings.ToUpper(id), string(jc), teacher).Scan(&f.course))
	f.enroll(t, teacher, "TEACHER", "ACTIVE", "", "Giảng Viên Thử")
	return f
}

// warmRedis mở sẵn 8 kết nối: mã chạy thật chỉ cho Redis 30 ms mỗi lệnh (redisOpTimeout, rơi về bộ nhớ khi quá hạn), nhưng kết nối LẠNH dưới -race + CI tải nặng
// (container Redis vừa dựng, nhiều gói test chạy song song) có thể mất hơn thế → test khẳng định trạng thái Redis thấy ánh xạ bộ nhớ / khoá chưa ghi.
func warmRedis(t *testing.T, rdb *appredis.Client) {
	t.Helper()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { _ = rdb.Ping(t.Context()).Err() })
	}
	wg.Wait()
}

func (f *fx) enroll(t *testing.T, user uuid.UUID, role, status, code, _ string) {
	t.Helper()
	var codeArg any
	if code != "" {
		codeArg = code
	}
	_, err := f.pool.Exec(t.Context(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot, removed_at) values ($1, $2, $3::enrollment_role, $4::enrollment_status, 'ADMIN', $5, case when $4::text = 'REMOVED' then now() end)`, f.course, user, role, status, codeArg)
	require.NoError(t, err)
}

func (f *fx) user(t *testing.T, name, role string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, f.pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, $2, $3::user_role) returning id`, uuid.NewString()+"@example.test", name, role).Scan(&id))
	return id
}

func (f *fx) masker() *privacy.Masker {
	d := &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: f.pool}, Redis: f.rdb, Log: f.log}}
	return &privacy.Masker{Detector: d, Redis: f.rdb, Log: f.log}
}

func (f *fx) members(t *testing.T) []string {
	t.Helper()
	ms, err := privacy.StoreRoster{Pool: f.pool}.Members(t.Context(), f.course)
	require.NoError(t, err)
	var out []string
	for _, m := range ms {
		out = append(out, m.Name)
	}
	return out
}

// TestRosterExcludesStaff — AC3: roster chỉ gồm sinh viên ACTIVE; giảng viên / TA / sinh viên chờ duyệt / đã bị mời ra không vào từ điển.
func TestRosterExcludesStaff(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.enroll(t, f.user(t, "Nguyễn Văn An", "STUDENT"), "STUDENT", "ACTIVE", "B21DCCN001", "")
	f.enroll(t, f.user(t, "Trợ Giảng Hai", "TA"), "TA", "ACTIVE", "", "")
	f.enroll(t, f.user(t, "Chờ Duyệt Ba", "STUDENT"), "STUDENT", "PENDING", "", "")
	f.enroll(t, f.user(t, "Đã Rời Bốn", "STUDENT"), "STUDENT", "REMOVED", "", "")
	require.Equal(t, []string{"Nguyễn Văn An"}, f.members(t))
}

// TestRosterInvalidatedOnMemberChange — AC3: cache ở ep:roster:{course} (TTL ≤ 1 giờ); sự kiện outbox xoá khoá; sinh viên mới bị bắt ở yêu cầu kế tiếp.
func TestRosterInvalidatedOnMemberChange(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ctx := t.Context()
	f.enroll(t, f.user(t, "Nguyễn Văn An", "STUDENT"), "STUDENT", "ACTIVE", "", "")
	m := f.masker()
	det := m.Detector
	fs, err := det.Detect(ctx, f.course, "Lê Thị Bình hỏi")
	require.NoError(t, err)
	require.Empty(t, fs)
	ttl, err := f.rdb.TTL(ctx, privacy.RosterKey(f.course)).Result()
	require.NoError(t, err)
	require.True(t, ttl > 0 && ttl <= time.Hour, ttl)

	f.enroll(t, f.user(t, "Lê Thị Bình", "STUDENT"), "STUDENT", "ACTIVE", "", "")
	fs, _ = det.Detect(ctx, f.course, "Lê Thị Bình hỏi")
	require.Empty(t, fs, "còn cache cũ tới khi có sự kiện")

	payload, _ := json.Marshal(map[string]string{"course_id": f.course.String()})
	for _, topic := range []string{"course.member_changed", "roster.imported"} {
		require.NoError(t, det.Roster.Invalidate(ctx, outbox.Message{Topic: topic, Payload: payload}))
	}
	fs, _ = det.Detect(ctx, f.course, "Lê Thị Bình hỏi")
	require.Len(t, fs, 1)

	// sinh viên bị mời ra không còn trong từ điển
	_, err = f.pool.Exec(ctx, `update enrollments set status='REMOVED', removed_at=now() where course_id=$1 and user_id=(select id from users where full_name='Lê Thị Bình' order by created_at desc limit 1)`, f.course)
	require.NoError(t, err)
	require.NoError(t, det.Roster.Invalidate(ctx, outbox.Message{Topic: "course.member_changed", Payload: payload}))
	fs, _ = det.Detect(ctx, f.course, "Lê Thị Bình hỏi")
	require.Empty(t, fs)

	// payload không có lớp: bỏ qua, không lỗi
	require.NoError(t, det.Roster.Invalidate(ctx, outbox.Message{Topic: "roster.imported", Payload: []byte(`{}`)}))
}

// TestMaskMappingTTL — AC7: ánh xạ ở HASH ep:mask:{sid}, TTL ≤ 24 giờ và được gia hạn mỗi lượt; tra được từ một Masker khác (tiến trình khác).
func TestMaskMappingTTL(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ctx := t.Context()
	f.enroll(t, f.user(t, "Nguyễn Văn An", "STUDENT"), "STUDENT", "ACTIVE", "", "")
	sid := "sess-" + uuid.NewString()
	m := f.masker()

	out, n, err := m.Mask(ctx, f.course, privacy.NewSession(sid), []string{"Nguyễn Văn An, 20201234"})
	require.NoError(t, err)
	require.Equal(t, []string{"[[SV_1]], [[MSSV_1]]"}, out)
	require.Equal(t, 2, n)
	ttl, err := f.rdb.TTL(ctx, privacy.MaskKey(sid)).Result()
	require.NoError(t, err)
	require.True(t, ttl > 23*time.Hour && ttl <= 24*time.Hour, ttl)

	require.NoError(t, f.rdb.Expire(ctx, privacy.MaskKey(sid), time.Minute).Err())
	_, _, err = m.Mask(ctx, f.course, privacy.NewSession(sid), []string{"Nguyễn Văn An"})
	require.NoError(t, err)
	ttl, _ = f.rdb.TTL(ctx, privacy.MaskKey(sid)).Result()
	require.True(t, ttl > 23*time.Hour, "mỗi lượt EXPIRE lại: %v", ttl)

	// một Masker + Sess mới (không có bộ nhớ) vẫn khôi phục được nhờ Redis; số thứ tự ổn định giữa các lượt
	m2 := f.masker()
	require.Equal(t, "Chào Nguyễn Văn An", m2.Unmask(ctx, privacy.NewSession(sid), "Chào [[SV_1]]"))
	out, _, err = m2.Mask(ctx, f.course, privacy.NewSession(sid), []string{"nguyen van an và 20201234"})
	require.NoError(t, err)
	require.Equal(t, []string{"[[SV_1]] và [[MSSV_1]]"}, out)
}

// TestMaskConcurrentSameSession — số thứ tự không trùng khi hai lượt che song song trên cùng phiên.
func TestMaskConcurrentSameSession(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ctx := t.Context()
	sid := "sess-" + uuid.NewString()
	m := f.masker()
	done := make(chan []string, 8)
	for i := range 8 {
		go func() {
			out, _, err := m.Mask(ctx, f.course, privacy.NewSession(sid), []string{"mail x" + string(rune('a'+i)) + "@b.vn"})
			if err != nil {
				done <- nil
				return
			}
			done <- out
		}()
	}
	seen := map[string]bool{}
	for range 8 {
		out := <-done
		require.Len(t, out, 1)
		require.False(t, seen[out[0]], "placeholder trùng: %s", out[0])
		seen[out[0]] = true
	}
}

// TestMaskingNeverLogged — AC7: log (mức debug) của một lượt chat đầy đủ không chứa tên, MSSV, email, nội dung ánh xạ.
func TestMaskingNeverLogged(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ctx := t.Context()
	f.enroll(t, f.user(t, "Nguyễn Văn An", "STUDENT"), "STUDENT", "ACTIVE", "ZZ99887766", "")
	sid := "sess-" + uuid.NewString()
	m := f.masker()
	s := privacy.NewSession(sid)
	out, _, err := m.Mask(ctx, f.course, s, []string{"Em Nguyễn Văn An, mssv 20201234, mail an@sv.edu.vn, sđt 0912345678, zz99887766"})
	require.NoError(t, err)
	_ = m.Unmask(ctx, s, "Chào "+out[0]+" và [[SV_9]]")
	u := m.NewStreamUnmasker(ctx, s)
	_ = u.Write("[[SV_")
	_ = u.Write("1]] [[EMAIL_7")
	_ = u.Flush()
	m.Redis = deadRedis(t) // nhánh Redis hỏng cũng không được log nội dung
	_, _, err = m.Mask(ctx, f.course, privacy.NewSession(sid), []string{"Nguyễn Văn An"})
	require.NoError(t, err)
	logs := f.logs.String()
	require.NotEmpty(t, logs, "phải có log để kiểm có nghĩa")
	for _, secret := range []string{"Nguyễn", "nguyen", "20201234", "an@sv.edu.vn", "0912345678", "ZZ99887766", "zz99887766"} {
		require.NotContains(t, logs, secret)
	}
}

func deadRedis(t *testing.T) *appredis.Client {
	t.Helper()
	c, err := appredis.New(t.Context(), "redis://127.0.0.1:1/0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// TestSessionlessMaskNoRedisKey — AC7: không có phiên → ánh xạ trong bộ nhớ của yêu cầu, không tạo khoá Redis.
func TestSessionlessMaskNoRedisKey(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ctx := t.Context()
	m := f.masker()
	s := privacy.NewSession("")
	out, _, err := m.Mask(ctx, f.course, s, []string{"mail an@sv.edu.vn"})
	require.NoError(t, err)
	require.Equal(t, []string{"mail [[EMAIL_1]]"}, out)
	require.Equal(t, "mail an@sv.edu.vn", m.Unmask(ctx, s, out[0]))
	n, err := f.rdb.Exists(ctx, privacy.MaskKey("")).Result()
	require.NoError(t, err)
	require.Zero(t, n)
}

var _ = context.Background
