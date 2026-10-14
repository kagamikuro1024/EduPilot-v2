package crm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

const weeksShown = 4 // student_risk.weeks: 4 tuần trọn gần nhất; chỉ hai tuần đầu quyết định mức

// weekStart là 00:00 thứ Hai (Asia/Ho_Chi_Minh) của tuần chứa t.
func weekStart(t time.Time) time.Time {
	l := t.In(ict)
	d := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, ict)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// weekJSON là một phần tử của `student_risk.weeks` (SRS 5.5).
type weekJSON struct {
	WeekStart  string `json:"week_start"`
	Attendance *struct {
		Recorded int    `json:"recorded"`
		OK       int    `json:"ok"`
		Ratio    string `json:"ratio"`
	} `json:"attendance"`
	Activity struct {
		Count int `json:"count"`
	} `json:"activity"`
	Study *struct {
		Minutes int `json:"minutes"`
	} `json:"study"`
	Weak bool     `json:"weak"`
	Low  []string `json:"low"`
}

func weeksOut(ws []WeekResult) []weekJSON {
	out := make([]weekJSON, len(ws))
	for i, w := range ws {
		o := weekJSON{WeekStart: w.Start.In(ict).Format(time.DateOnly), Weak: w.Weak, Low: w.Low}
		o.Activity.Count = w.Activity
		if a := w.Attendance; a != nil && a.Recorded > 0 {
			o.Attendance = &struct {
				Recorded int    `json:"recorded"`
				OK       int    `json:"ok"`
				Ratio    string `json:"ratio"`
			}{a.Recorded, a.OK, decimal.NewFromInt(int64(a.OK)).Div(decimal.NewFromInt(int64(a.Recorded))).StringFixed(2)}
		}
		if w.StudyMinutes != nil {
			o.Study = &struct {
				Minutes int `json:"minutes"`
			}{*w.StudyMinutes}
		}
		out[i] = o
	}
	return out
}

// RunCourse tính lại `student_risk` của MỘT lớp (SRS 4.7.4): mỗi sinh viên ACTIVE một dòng (UPSERT, `entered_at` giữ nguyên khi đã AT_RISK), xoá dòng của người không còn ACTIVE,
// rồi ghi outbox `crm.risk_updated` — tất cả trong một giao dịch. Chạy lại cùng dữ liệu → cùng kết quả.
func (s *Service) RunCourse(ctx context.Context, courseID uuid.UUID, now time.Time) error {
	cw := weekStart(now)
	from := cw.AddDate(0, 0, -7*weeksShown)
	fromDay := pgtype.Date{Time: time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
	q := store.New(s.Pool)
	crs, err := q.CrmRiskCourse(ctx, courseID)
	if err != nil {
		return wrap("đọc lớp", err)
	}
	th := ThresholdsFrom(crs.Settings)
	studs, err := q.CrmRiskStudents(ctx, courseID)
	if err != nil {
		return wrap("sinh viên ACTIVE", err)
	}
	att, err := q.CrmRiskAttendance(ctx, store.CrmRiskAttendanceParams{CourseID: courseID, FromDay: fromDay, FromAt: from, ToAt: cw})
	if err != nil {
		return wrap("chuyên cần theo tuần", err)
	}
	act, err := q.CrmRiskActivity(ctx, store.CrmRiskActivityParams{CourseID: courseID, FromDay: fromDay, FromAt: from, ToAt: cw})
	if err != nil {
		return wrap("hoạt động theo tuần", err)
	}
	study, err := q.CrmRiskStudy(ctx, store.CrmRiskStudyParams{CourseID: courseID, FromDay: fromDay, ToDay: pgtype.Date{Time: fromDay.Time.AddDate(0, 0, 7*weeksShown), Valid: true}})
	if err != nil {
		return wrap("thời gian học theo tuần", err)
	}
	type key struct {
		s uuid.UUID
		k int32
	}
	attBy, actBy, studyBy := map[key]store.CrmRiskAttendanceRow{}, map[key]store.CrmRiskActivityRow{}, map[key]int32{}
	for _, r := range att {
		attBy[key{r.StudentID, r.K}] = r
	}
	for _, r := range act {
		actBy[key{r.StudentID, r.K}] = r
	}
	for _, r := range study {
		studyBy[key{r.StudentID, r.K}] = r.Seconds
	}
	type row struct {
		StudentID uuid.UUID  `json:"student_id"`
		Level     string     `json:"level"`
		Reasons   []Reason   `json:"reasons"`
		Weeks     []weekJSON `json:"weeks"`
		Asks      int        `json:"asks"`
		Posts     int        `json:"posts"`
	}
	rows := make([]row, 0, len(studs))
	for _, st := range studs {
		weeks := make([]Week, weeksShown) // weeks[0] = W1 (k = 3)
		for i := range weeks {
			k := int32(weeksShown - 1 - i) //nolint:gosec // 0..3
			start := cw.AddDate(0, 0, -7*(i+1))
			w := Week{Start: start}
			if a, ok := attBy[key{st.StudentID, k}]; ok {
				w.Attendance = &Attendance{Recorded: int(a.Recorded), OK: int(a.Ok), Absent: int(a.Absent)}
			}
			if a, ok := actBy[key{st.StudentID, k}]; ok {
				w.Activity = int(a.Asks + a.Posts)
			}
			// STUDY có dữ liệu khi đã xác nhận thông báo từ trước tuần này VÀ (lớp không cho tắt đo HOẶC sinh viên chưa tắt) — SRS 4.7.1.
			if st.AckAt != nil && !st.AckAt.After(start) && (!crs.AllowStudyTimeOptOut || st.TrackStudyTime) {
				m := int(studyBy[key{st.StudentID, k}] / 60)
				w.StudyMinutes = &m
			}
			weeks[i] = w
		}
		res := Evaluate(weeks, th)
		r := row{StudentID: st.StudentID, Level: res.Level, Reasons: res.Reasons, Weeks: weeksOut(res.Weeks)}
		if a, ok := actBy[key{st.StudentID, weeksShown - 1}]; ok {
			r.Asks, r.Posts = int(a.Asks), int(a.Posts)
		}
		rows = append(rows, r)
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("crm: mã hoá student_risk: %w", err)
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return wrap("mở giao dịch", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := store.New(tx)
	if err := tq.CrmRiskUpsert(ctx, store.CrmRiskUpsertParams{CourseID: courseID, NowAt: now, Rows: raw}); err != nil {
		return wrap("ghi student_risk", err)
	}
	if err := tq.CrmRiskDeleteStale(ctx, courseID); err != nil {
		return wrap("xoá student_risk cũ", err)
	}
	if _, err := outbox.Write(ctx, tx, TopicRiskUpdated, map[string]string{"course_id": courseID.String()}); err != nil {
		return fmt.Errorf("crm: outbox %s: %w", TopicRiskUpdated, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return wrap("commit student_risk", err)
	}
	return nil
}

// RunAll chạy mọi lớp ACTIVE lần lượt (mỗi lớp một giao dịch). Gặp lỗi một lớp thì dừng và trả lỗi: job xoá dấu "đã chạy" để tick sau chạy lại (các lớp đã xong chạy lại vô hại).
func (s *Service) RunAll(ctx context.Context, now time.Time) error {
	ids, err := store.New(s.Pool).CrmRiskActiveCourses(ctx)
	if err != nil {
		return wrap("lớp ACTIVE", err)
	}
	for _, id := range ids {
		if err := s.RunCourse(ctx, id, now); err != nil {
			return fmt.Errorf("crm.risk lớp %s: %w", id, err)
		}
	}
	return nil
}

// HandleSettingsChanged là outbox.Handler của `course.settings_changed`: tính lại RIÊNG lớp đó (SRS 4.7.3). Idempotent.
func (s *Service) HandleSettingsChanged(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID uuid.UUID `json:"course_id"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil {
		return fmt.Errorf("crm: payload %s: %w", m.Topic, err)
	}
	return s.RunCourse(ctx, p.CourseID, s.now())
}

// RiskLeaderKey: khoá leader của `crm.risk` (chỉ chống hai bản chạy cùng lúc; "đã chạy hôm nay" do khoá `ep:crm.risk:done:{yyyymmdd}`).
var RiskLeaderKey = appredis.Key("lock", "crm.risk") //nolint:gochecknoglobals // tên khoá cố định

// RiskJob chạy `crm.risk` đúng MỘT lần mỗi ngày sau `At` (giờ ICT kể từ 00:00): mỗi tick, nếu `now ≥ At` hôm nay và SET NX dấu ngày thành công thì chạy — chạy bù khi worker lên lại,
// lỗi giữa chừng thì DEL dấu để tick sau thử lại (SRS 4.7.4, TLR-6).
type RiskJob struct {
	Svc   *Service
	Redis *appredis.Client
	Name  string
	At    time.Duration
	Every time.Duration // mỗi tick (1 phút)
	Log   *slog.Logger
}

// Run chạy tới khi ctx huỷ.
func (j *RiskJob) Run(ctx context.Context) error {
	tick := time.NewTicker(j.Every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			if _, err := j.Once(ctx); err != nil && j.Log != nil {
				j.Log.ErrorContext(ctx, "crm.risk lỗi", "error", err.Error())
			}
		}
	}
}

// Once là một tick: true khi lượt chạy của hôm nay được thực hiện ở tick này.
func (j *RiskJob) Once(ctx context.Context) (bool, error) {
	now := j.Svc.now().In(ict)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ict)
	if now.Before(day.Add(j.At)) {
		return false, nil
	}
	if !j.Redis.Leader(ctx, RiskLeaderKey, j.Name, 2*j.Every) {
		return false, nil
	}
	done := appredis.Key("crm.risk", "done", day.Format("20060102"))
	ok, err := j.Redis.SetNX(ctx, done, 1, 48*time.Hour).Result()
	if err != nil || !ok {
		return false, err
	}
	if err := j.Svc.RunAll(ctx, now); err != nil {
		_ = j.Redis.Del(context.WithoutCancel(ctx), done).Err() // tick sau chạy lại
		return false, err
	}
	return true, nil
}
