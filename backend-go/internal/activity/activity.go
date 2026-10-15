// Package activity đo thời gian học (US-P5-03, SRS FEAT-crm-360 4.6): heartbeat CHỈ ghi Redis (một lời gọi Lua nguyên tử), worker cuộn lên `study_time_daily` mỗi phút
// bằng ghi TUYỆT ĐỐI (GREATEST). Dữ liệu tham khảo cho giảng viên (D17) — KHÔNG BAO GIỜ dùng tính điểm: `internal/grade` không được phụ thuộc gói này (TestGradeNeverImportsActivity).
package activity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Areas là TRACKING_AREAS (cố định): chỉ khu vực, không ghi nội dung / đường dẫn / tiêu đề / tên người.
var Areas = []string{"CHAT", "THREADS", "PRACTICE", "LIBRARY", "CALENDAR"} //nolint:gochecknoglobals // bảng tra cứu bất biến

// HeartbeatSeconds là HEARTBEAT_SECONDS (cố định): một nhịp = 30 giây và cũng là độ rộng khung khử trùng.
const HeartbeatSeconds = 30

// Redis key (SRS 5.9). Mọi khoá truyền qua KEYS của Lua.
var (
	dirtyKey   = appredis.Key("study", "dirty")   //nolint:gochecknoglobals // tên khoá cố định
	flushedKey = appredis.Key("study", "flushed") //nolint:gochecknoglobals
)

// Config là các hằng 4.1 (đã kiểm miền ở config.Load).
type Config struct {
	MaxPerMinute int           // HEARTBEAT_MAX_PER_MINUTE (4)
	DailyCap     int           // STUDY_DAILY_CAP (43200)
	CounterTTL   time.Duration // HEARTBEAT_COUNTER_TTL (192h)
}

func (c Config) withDefaults() Config {
	if c.MaxPerMinute == 0 {
		c.MaxPerMinute = 4
	}
	if c.DailyCap == 0 {
		c.DailyCap = 43200
	}
	if c.CounterTTL == 0 {
		c.CounterTTL = 192 * time.Hour
	}
	return c
}

// Service là heartbeat + rollup + đọc.
type Service struct {
	Pool  *pgxpool.Pool
	Redis *appredis.Client
	Clock clock.Clock
	Log   *slog.Logger
	Cfg   Config
	// Dropped đếm nhịp bị bỏ vì Redis lỗi (`activity_heartbeat_dropped_total`).
	Dropped atomic.Int64
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now()
}

// ictLoc là Asia/Ho_Chi_Minh (UTC+7, không đổi giờ mùa hè): ngày và tuần tính theo giờ Việt Nam.
var ictLoc = time.FixedZone("ICT", 7*3600) //nolint:gochecknoglobals // múi giờ cố định

func (s *Service) day() string { return s.now().In(ictLoc).Format("20060102") }

// ---- heartbeat ------------------------------------------------------------------------------------------------------------------------------

// beatLua — một lời gọi nguyên tử (SRS 4.6.1 bước 4). KEYS: 1 rate, 2 win, 3 total, 4 count, 5 dirty. ARGV: 1 tối đa/phút, 2 trần ngày, 3 TTL (giây), 4 giây mỗi nhịp.
// Trả 1 = đã tính, 0 = bỏ (vượt tốc độ / cùng khung / chạm trần). Nhịp bị bỏ ở bước (a) hay (b) không cộng nhưng vẫn chiếm quota của chính bước đó.
const beatLua = `
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('EXPIRE', KEYS[1], 60) end
if n > tonumber(ARGV[1]) then return 0 end
if not redis.call('SET', KEYS[2], '1', 'NX', 'EX', 40) then return 0 end
local total = tonumber(redis.call('GET', KEYS[3]) or '0')
if total >= tonumber(ARGV[2]) then return 0 end
local add = tonumber(ARGV[4])
redis.call('INCRBY', KEYS[3], add)
redis.call('EXPIRE', KEYS[3], ARGV[3])
redis.call('INCRBY', KEYS[4], add)
redis.call('EXPIRE', KEYS[4], ARGV[3])
redis.call('SADD', KEYS[5], KEYS[4])
return 1`

var beatScript = goredis.NewScript(beatLua) //nolint:gochecknoglobals // script nạp một lần

// Beat là thân `POST /activity/heartbeat`.
type Beat struct {
	CourseID uuid.UUID
	Area     string
	TabID    string
}

type trk struct {
	Ack   bool `json:"ack"`
	Track bool `json:"track"`
}

type enr struct {
	Active bool `json:"active"`
	Allow  bool `json:"allow"`
}

func trkKey(uid uuid.UUID) string { return appredis.Key("trk", uid.String()) }
func enrKey(uid, course uuid.UUID) string {
	return appredis.Key("enr", uid.String(), course.String())
}

// RedisBudget là tổng thời gian một nhịp được dành cho MỌI lệnh Redis (#31): Redis chết thì nhịp bị bỏ và trả `204` trong chừng đó, không treo ~10 s.
const RedisBudget = 500 * time.Millisecond

// cached đọc JSON đệm 60 s; trượt đệm → load rồi ghi. Redis lỗi → load thẳng (không chặn nhịp, nhưng không ghi Postgres). `rctx` giới hạn riêng các lệnh Redis.
func cached[T any](ctx, rctx context.Context, rd *appredis.Client, key string, load func() (T, error)) (T, error) {
	var v T
	if rd != nil {
		if raw, err := rd.Get(rctx, key).Bytes(); err == nil && json.Unmarshal(raw, &v) == nil {
			return v, nil
		}
	}
	v, err := load()
	if err != nil {
		return v, err
	}
	if rd != nil {
		if raw, merr := json.Marshal(v); merr == nil {
			_ = rd.Set(rctx, key, raw, time.Minute).Err()
		}
	}
	return v, nil
}

// InvalidateTracking xoá đệm `ep:trk:{uid}` (sau ack hoặc đổi `track_study_time`) để nhịp kế tiếp có hiệu lực ngay.
func (s *Service) InvalidateTracking(ctx context.Context, uid uuid.UUID) {
	if s.Redis == nil {
		return
	}
	if err := s.Redis.Del(ctx, trkKey(uid)).Err(); err != nil && s.Log != nil {
		s.Log.WarnContext(ctx, "activity: không xoá được đệm ep:trk", "error", err.Error())
	}
}

func (s *Service) tracking(ctx context.Context, uid uuid.UUID) (trk, error) {
	return s.trackingR(ctx, ctx, uid)
}

func (s *Service) trackingR(ctx, rctx context.Context, uid uuid.UUID) (trk, error) {
	return cached(ctx, rctx, s.Redis, trkKey(uid), func() (trk, error) {
		r, err := store.New(s.Pool).ActivityUserTracking(ctx, uid)
		if err != nil {
			return trk{}, fmt.Errorf("activity: đọc ack: %w", err)
		}
		return trk{Ack: r.TrackingNoticeAckAt != nil, Track: r.TrackStudyTime}, nil
	})
}

func (s *Service) enrollment(ctx context.Context, uid, course uuid.UUID) (enr, error) {
	return s.enrollmentR(ctx, ctx, uid, course)
}

func (s *Service) enrollmentR(ctx, rctx context.Context, uid, course uuid.UUID) (enr, error) {
	return cached(ctx, rctx, s.Redis, enrKey(uid, course), func() (enr, error) {
		r, err := store.New(s.Pool).ActivityEnrollment(ctx, store.ActivityEnrollmentParams{CourseID: course, UserID: uid})
		if errors.Is(err, pgx.ErrNoRows) {
			return enr{}, nil
		}
		if err != nil {
			return enr{}, fmt.Errorf("activity: đọc ghi danh: %w", err)
		}
		return enr{Active: r.RoleInCourse == "STUDENT" && r.EnrollmentStatus == "ACTIVE", Allow: r.AllowStudyTimeOptOut}, nil
	})
}

// Heartbeat xử lý một nhịp. Trả nil = `204` (kể cả khi bỏ nhịp); lỗi apierr cho 403 / 422. KHÔNG ghi Postgres.
func (s *Service) Heartbeat(ctx context.Context, uid uuid.UUID, role string, b Beat) error {
	if !slices.Contains(Areas, b.Area) {
		return apierr.Validation(apierr.FieldError{Field: "area", Code: "invalid", Message: "Khu vực không hợp lệ."})
	}
	if role != "STUDENT" {
		return nil // nhân viên / Admin không bị đo
	}
	rctx, cancel := context.WithTimeout(ctx, RedisBudget)
	defer cancel()
	t, err := s.trackingR(ctx, rctx, uid)
	if err != nil {
		return s.drop(ctx, err)
	}
	if !t.Ack {
		return nil // chưa bấm "Đã hiểu": không đo
	}
	e, err := s.enrollmentR(ctx, rctx, uid, b.CourseID)
	if err != nil {
		return s.drop(ctx, err)
	}
	if !e.Active {
		return apierr.New(http.StatusForbidden, apierr.Forbidden)
	}
	if e.Allow && !t.Track {
		return nil // lớp cho tắt và sinh viên đã tắt: nhịp không chiếm khung 30 s (AC13)
	}
	now := s.now()
	day := now.In(ictLoc).Format("20060102")
	cfg := s.Cfg.withDefaults()
	uidS := uid.String()
	keys := []string{
		appredis.Key("hb", "rate", uidS),
		appredis.Key("hb", "win", uidS, strconv.FormatInt(now.Unix()/HeartbeatSeconds, 10)),
		appredis.Key("study", "total", uidS, day),
		appredis.Key("study", b.CourseID.String(), uidS, day, b.Area),
		dirtyKey,
	}
	if s.Redis == nil {
		return s.drop(ctx, errors.New("không có Redis"))
	}
	if err := beatScript.Run(rctx, s.Redis, keys, cfg.MaxPerMinute, cfg.DailyCap, int(cfg.CounterTTL.Seconds()), HeartbeatSeconds).Err(); err != nil {
		return s.drop(ctx, err)
	}
	return nil
}

// drop: Redis lỗi → `204` + đếm; không `5xx` làm trang giật, không log `area` kèm `uid`.
func (s *Service) drop(ctx context.Context, cause error) error {
	s.Dropped.Add(1)
	if s.Log != nil {
		s.Log.WarnContext(ctx, "activity_heartbeat_dropped_total", "error", cause.Error(), "total", s.Dropped.Load())
	}
	return nil
}

// Ack đặt `tracking_notice_ack_at` (idempotent, giữ lần đầu) rồi xoá đệm `ep:trk:{uid}` — không phụ thuộc claim JWT (TLR-9).
func (s *Service) Ack(ctx context.Context, uid uuid.UUID) error {
	if _, err := store.New(s.Pool).ActivityAckTracking(ctx, uid); err != nil {
		return fmt.Errorf("activity: ack: %w", err)
	}
	s.InvalidateTracking(ctx, uid)
	return nil
}

// ---- rollup ---------------------------------------------------------------------------------------------------------------------------------

type counter struct {
	key             string
	course, student uuid.UUID
	date, area      string
	secs            int
}

// parseKey: `ep:study:{course}:{uid}:{yyyymmdd}:{AREA}`.
func parseKey(key string) (counter, bool) {
	p := strings.Split(strings.TrimPrefix(key, appredis.Key("study")+":"), ":")
	if len(p) != 4 {
		return counter{}, false
	}
	c, err1 := uuid.Parse(p[0])
	u, err2 := uuid.Parse(p[1])
	d, err3 := time.Parse("20060102", p[2])
	if err1 != nil || err2 != nil || err3 != nil || !slices.Contains(Areas, p[3]) {
		return counter{}, false
	}
	return counter{key: key, course: c, student: u, date: d.Format("2006-01-02"), area: p[3]}, true
}

const rollupBatch = 500

// Rollup cuộn một lượt (SRS 4.6.2). Ghi TUYỆT ĐỐI (`GREATEST`), chỉ các khoá ĐỔI từ lần trước (so `ep:study:flushed`); khoá của ngày đã qua > 1 ngày bị `SREM` khỏi `dirty` và `HDEL` ở `flushed`.
// Chạy lại / giao đôi / chết giữa chừng không đổi kết quả; Redis mất khoá thì DB giữ giá trị cũ (không âm, không đôi). Trả số hàng đã ghi.
func (s *Service) Rollup(ctx context.Context) (int, error) {
	if s.Redis == nil {
		return 0, nil
	}
	var keys []string
	var cur uint64
	for {
		ks, next, err := s.Redis.SScan(ctx, dirtyKey, cur, "*", 1000).Result()
		if err != nil {
			return 0, fmt.Errorf("activity: quét dirty: %w", err)
		}
		keys = append(keys, ks...)
		if cur = next; cur == 0 {
			break
		}
	}
	if len(keys) == 0 {
		return 0, nil
	}
	today, _ := time.ParseInLocation("20060102", s.day(), ictLoc)
	written := 0
	for chunk := range slices.Chunk(keys, rollupBatch) {
		vals, err := s.Redis.MGet(ctx, chunk...).Result()
		if err != nil {
			return written, fmt.Errorf("activity: đọc bộ đếm: %w", err)
		}
		seen, err := s.Redis.HMGet(ctx, flushedKey, chunk...).Result()
		if err != nil {
			return written, fmt.Errorf("activity: đọc flushed: %w", err)
		}
		var rows []counter
		stale := map[string]bool{}   // khoá ngày đã qua > 1 ngày: ứng viên dọn
		settled := map[string]bool{} // đã ghi, không đổi, hoặc không còn gì để ghi — chỉ những khoá này mới được dọn
		for i, k := range chunk {
			c, ok := parseKey(k)
			if !ok {
				stale[k], settled[k] = true, true // khoá lạ: dọn khỏi dirty
				continue
			}
			if d, _ := time.ParseInLocation("2006-01-02", c.date, ictLoc); today.Sub(d) > 24*time.Hour {
				stale[k] = true
			}
			v, ok := vals[i].(string)
			n, aerr := strconv.Atoi(v)
			if !ok || aerr != nil || n <= 0 { // Redis mất khoá (hết hạn / khởi động lại không lưu): DB giữ giá trị cũ
				settled[k] = true
				continue
			}
			if prev, _ := seen[i].(string); prev == v {
				settled[k] = true // không đổi từ lần trước
				continue
			}
			c.secs = min(n, 86400)
			rows = append(rows, c)
		}
		if len(rows) > 0 {
			done, err := s.write(ctx, rows)
			if err != nil {
				return written, err
			}
			written += len(done)
			for _, c := range done { // chỉ đánh dấu "đã ghi" các hàng THỰC SỰ vào DB: cặp không có ghi danh ở DB này được bỏ ngỏ
				settled[c.key] = true
				if err := s.Redis.HSet(ctx, flushedKey, c.key, strconv.Itoa(c.secs)).Err(); err != nil {
					return written, fmt.Errorf("activity: ghi flushed: %w", err)
				}
			}
		}
		for k := range stale {
			if !settled[k] {
				continue
			}
			if err := s.Redis.SRem(ctx, dirtyKey, k).Err(); err != nil {
				return written, fmt.Errorf("activity: dọn dirty: %w", err)
			}
			if err := s.Redis.HDel(ctx, flushedKey, k).Err(); err != nil {
				return written, fmt.Errorf("activity: dọn flushed: %w", err)
			}
		}
	}
	return written, nil
}

// write upsert một lô, bỏ cặp (lớp, sinh viên) không có ghi danh (khoá ngoại phức hợp sẽ làm cả lô lỗi); trả các hàng đã ghi.
func (s *Service) write(ctx context.Context, rows []counter) ([]counter, error) {
	q := store.New(s.Pool)
	cs, ss := make([]string, len(rows)), make([]string, len(rows))
	for i, r := range rows {
		cs[i], ss[i] = r.course.String(), r.student.String()
	}
	pairs, err := q.ActivityEnrolledPairs(ctx, store.ActivityEnrolledPairsParams{CourseIds: cs, StudentIds: ss})
	if err != nil {
		return nil, fmt.Errorf("activity: kiểm ghi danh: %w", err)
	}
	ok := make(map[[2]uuid.UUID]bool, len(pairs))
	for _, p := range pairs {
		ok[[2]uuid.UUID{p.CourseID, p.UserID}] = true
	}
	var in store.ActivityUpsertStudyTimeParams
	var done []counter
	for _, r := range rows {
		if !ok[[2]uuid.UUID{r.course, r.student}] {
			continue
		}
		done = append(done, r)
		in.CourseIds = append(in.CourseIds, r.course.String())
		in.StudentIds = append(in.StudentIds, r.student.String())
		in.Dates = append(in.Dates, r.date)
		in.Areas = append(in.Areas, r.area)
		in.Seconds = append(in.Seconds, int32(r.secs)) //nolint:gosec // ≤ 86400
	}
	if len(in.CourseIds) == 0 {
		return nil, nil
	}
	if err := q.ActivityUpsertStudyTime(ctx, in); err != nil {
		return nil, fmt.Errorf("activity: ghi study_time_daily: %w", err)
	}
	return done, nil
}

// ---- đọc ------------------------------------------------------------------------------------------------------------------------------------

// Week là một tuần ISO (thứ Hai `Asia/Ho_Chi_Minh`) của một sinh viên.
type Week struct {
	WeekStart string         `json:"week_start"`
	Minutes   int            `json:"minutes"`
	ByArea    map[string]int `json:"by_area"`
}

// StudyTime là phản hồi của hai route đọc. Sinh viên: `Tracking`; nhân viên: `Tracked`.
type StudyTime struct {
	Weeks           []Week    `json:"weeks"`
	ThisWeekMinutes int       `json:"this_week_minutes"`
	Tracking        *Tracking `json:"tracking,omitempty"`
	Tracked         *bool     `json:"tracked,omitempty"`
}

// Tracking là cấu hình đo của sinh viên ở lớp này: Enabled = tracked (4.6.7), Locked = lớp không cho tắt.
type Tracking struct {
	Enabled bool `json:"enabled"`
	Locked  bool `json:"locked"`
}

// MaxWeeks / DefaultWeeks là miền `weeks` (SRS 4.6.5).
const (
	MaxWeeks     = 20
	DefaultWeeks = 8
)

func weekStartOf(t time.Time) time.Time {
	d := t.In(ictLoc)
	d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, ictLoc)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// Read trả thời gian học `weeks` tuần gần nhất (tuần có dữ liệu, tăng dần) của một sinh viên ở một lớp. `weeks` ngoài [1, 20] → 422.
func (s *Service) Read(ctx context.Context, courseID, studentID uuid.UUID, weeks int, staff bool) (StudyTime, error) {
	if weeks < 1 || weeks > MaxWeeks {
		return StudyTime{}, apierr.Validation(apierr.FieldError{Field: "weeks", Code: "range", Message: "weeks phải từ 1 đến 20."})
	}
	cur := weekStartOf(s.now())
	rows, err := store.New(s.Pool).ActivityStudyTimeWeeks(ctx, store.ActivityStudyTimeWeeksParams{CourseID: courseID, StudentID: studentID, FromDate: pgtype.Date{Time: cur.AddDate(0, 0, -7*(weeks-1)), Valid: true}})
	if err != nil {
		return StudyTime{}, fmt.Errorf("activity: đọc thời gian học: %w", err)
	}
	out := StudyTime{Weeks: []Week{}}
	secs := map[string]int64{}
	var order []string
	byArea := map[string]map[string]int{}
	for _, r := range rows {
		k := r.WeekStart.Time.Format("2006-01-02")
		if _, ok := secs[k]; !ok {
			order = append(order, k)
			byArea[k] = map[string]int{}
		}
		secs[k] += r.Seconds
		byArea[k][r.Area] = int(r.Seconds / 60)
	}
	for _, k := range order {
		w := Week{WeekStart: k, Minutes: int(secs[k] / 60), ByArea: byArea[k]}
		out.Weeks = append(out.Weeks, w)
		if k == cur.Format("2006-01-02") {
			out.ThisWeekMinutes = w.Minutes
		}
	}
	e, err := s.enrollment(ctx, studentID, courseID)
	if err != nil {
		return StudyTime{}, err
	}
	if staff && !e.Active {
		return StudyTime{}, apierr.New(http.StatusForbidden, apierr.Forbidden) // không phải sinh viên ACTIVE của lớp: 403 (AC9, #31)
	}
	t, err := s.tracking(ctx, studentID)
	if err != nil {
		return StudyTime{}, err
	}
	tracked := !e.Allow || t.Track
	if staff {
		out.Tracked = &tracked
	} else {
		out.Tracking = &Tracking{Enabled: tracked, Locked: !e.Allow}
	}
	return out, nil
}

// SettingsCourse là một dòng của nhóm "Thời gian học" ở `GET /me/settings` (chỉ lớp sinh viên đang ACTIVE).
type SettingsCourse struct {
	CourseID  uuid.UUID `json:"course_id"`
	ClassCode string    `json:"class_code"`
	Name      string    `json:"name"`
	Locked    bool      `json:"locked"`
}

// StudySettings là nhóm "Thời gian học" của `GET /me/settings`: đã bấm "Đã hiểu" chưa + lớp ACTIVE kèm cờ khoá (giao diện ẩn các dòng cho tới khi ack — chưa đo gì).
type StudySettings struct {
	Acked   bool             `json:"acked"`
	Courses []SettingsCourse `json:"courses"`
}

// Settings trả nhóm "Thời gian học" của một người. Không phải sinh viên → `courses` rỗng.
func (s *Service) Settings(ctx context.Context, uid uuid.UUID) (StudySettings, error) {
	t, err := s.tracking(ctx, uid)
	if err != nil {
		return StudySettings{}, err
	}
	out := StudySettings{Acked: t.Ack, Courses: []SettingsCourse{}}
	rows, err := store.New(s.Pool).ActivitySettingsCourses(ctx, uid)
	if err != nil {
		return StudySettings{}, fmt.Errorf("activity: lớp của tôi: %w", err)
	}
	for _, r := range rows {
		out.Courses = append(out.Courses, SettingsCourse{CourseID: r.CourseID, ClassCode: r.ClassCode, Name: r.Name, Locked: r.Locked})
	}
	return out, nil
}
