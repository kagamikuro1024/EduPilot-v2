// Package seedtool: ba việc của seed sprint 7 mà API không làm được vì phải ghi thời gian QUÁ KHỨ (FEAT-crm-360 SRS 4.10).
// Chỉ gọi từ `gateway admin …` và chỉ khi APP_ENV nằm trong danh sách cho phép (xem cmd/gateway/seedcmd.go). Không đổi nội dung nghiệp vụ, chỉ đổi mốc thời gian.
package seedtool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ict = time.FixedZone("Asia/Ho_Chi_Minh", 7*3600) //nolint:gochecknoglobals // múi giờ cố định, bất biến

// areas là năm khu vực đo của `study_time_daily` (CHECK ở 00015).
var areas = [...]string{"CHAT", "THREADS", "PRACTICE", "LIBRARY", "CALENDAR"} //nolint:gochecknoglobals // hằng

// CourseID tìm lớp theo mã lớp.
func CourseID(ctx context.Context, pool *pgxpool.Pool, classCode string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `select id from courses where class_code = $1`, classCode).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("không có lớp %q", classCode)
	}
	return id, err
}

// StudyPlan là số phút mỗi tuần của một sinh viên; `Weeks` ghi đè từng tuần (khoá "1".."N", 1 = tuần xa nhất của khoảng, N = tuần hiện tại).
type StudyPlan struct {
	MinutesPerWeek int            `json:"minutes_per_week"`
	Weeks          map[string]int `json:"weeks"`
}

// StudyProfile là `seed/study_profile.json`: `default` cho mọi sinh viên ACTIVE, `students` ghi đè theo email.
type StudyProfile struct {
	Default  StudyPlan            `json:"default"`
	Students map[string]StudyPlan `json:"students"`
}

// ParseProfile đọc JSON; khoá lạ bị từ chối để lỗi chính tả không âm thầm thành "mặc định".
func ParseProfile(raw []byte) (StudyProfile, error) {
	var p StudyProfile
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return p, fmt.Errorf("study_profile: %w", err)
	}
	return p, nil
}

func (p StudyProfile) minutes(email string, week int) int {
	plan, ok := p.Students[email]
	base := p.Default.MinutesPerWeek
	if ok && plan.MinutesPerWeek > 0 {
		base = plan.MinutesPerWeek
	}
	key := fmt.Sprint(week)
	if v, has := p.Default.Weeks[key]; has {
		base = v
	}
	if ok {
		if v, has := plan.Weeks[key]; has {
			base = v
		}
	}
	return base
}

type studyRow struct {
	student uuid.UUID
	day     time.Time
	area    string
	seconds int
}

// monday trả 00:00 thứ Hai (giờ Việt Nam) của tuần chứa t.
func monday(t time.Time) time.Time {
	l := t.In(ict)
	d := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// studyRows dựng các dòng của một sinh viên: tuần `n` (1..weeks) có `minutes` phút, chia đều cho các ngày đã trôi qua (tuần hiện tại: chỉ tới hôm nay, số phút nhân tỉ lệ),
// mỗi ngày 1–3 khu vực luân phiên. Tổng giây mỗi tuần = đúng `minutes × 60` (× tỉ lệ) nên `floor(Σ/60)` đúng số phút.
func studyRows(student uuid.UUID, email string, p StudyProfile, weeks int, now time.Time) []studyRow {
	cur := monday(now)
	today := time.Date(now.In(ict).Year(), now.In(ict).Month(), now.In(ict).Day(), 0, 0, 0, 0, time.UTC)
	var out []studyRow
	for n := 1; n <= weeks; n++ {
		minutes := p.minutes(email, n)
		if minutes <= 0 {
			continue
		}
		start := cur.AddDate(0, 0, -7*(weeks-n))
		days := 7
		if n == weeks {
			days = int(today.Sub(start).Hours()/24) + 1
		}
		total := minutes * 60 * days / 7
		for d := range days {
			sec := total / days
			if d < total%days {
				sec++
			}
			if sec == 0 {
				continue
			}
			k := 1 + (n+d)%3
			first := (n*3 + d) % len(areas)
			for a := range k {
				s := sec / k
				if a < sec%k {
					s++
				}
				if s == 0 {
					continue
				}
				out = append(out, studyRow{student: student, day: start.AddDate(0, 0, d), area: areas[(first+a)%len(areas)], seconds: s})
			}
		}
	}
	return out
}

// SeedStudyTime ghi `study_time_daily` của mọi sinh viên ACTIVE của lớp (`ON CONFLICT … GREATEST`, chạy lại không nhân đôi) và trả số dòng đã ĐỔI (thêm mới hoặc tăng).
func SeedStudyTime(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, weeks int, p StudyProfile, now time.Time) (int64, error) {
	if weeks < 1 || weeks > 52 {
		return 0, errors.New("--weeks phải từ 1 đến 52")
	}
	rs, err := pool.Query(ctx, `select e.user_id, u.email from enrollments e join users u on u.id = e.user_id where e.course_id = $1 and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE' order by u.email`, courseID)
	if err != nil {
		return 0, err
	}
	type stu struct {
		id    uuid.UUID
		email string
	}
	var studs []stu
	for rs.Next() {
		var s stu
		if err := rs.Scan(&s.id, &s.email); err != nil {
			rs.Close()
			return 0, err
		}
		studs = append(studs, s)
	}
	rs.Close()
	if err := rs.Err(); err != nil {
		return 0, err
	}
	var ids []uuid.UUID
	var days []time.Time
	var ar []string
	var secs []int32
	for _, s := range studs {
		for _, r := range studyRows(s.id, s.email, p, weeks, now) {
			ids, days, ar, secs = append(ids, r.student), append(days, r.day), append(ar, r.area), append(secs, int32(r.seconds)) //nolint:gosec // ≤ 86400
		}
	}
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := pool.Exec(ctx, `
insert into study_time_daily (course_id, student_id, date, area, active_seconds)
select $1, x.s, x.d, x.a, x.n from unnest($2::uuid[], $3::date[], $4::text[], $5::int[]) as x(s, d, a, n)
on conflict (course_id, student_id, date, area) do update set active_seconds = greatest(study_time_daily.active_seconds, excluded.active_seconds)
where study_time_daily.active_seconds < excluded.active_seconds`, courseID, ids, days, ar, secs)
	if err != nil {
		return 0, fmt.Errorf("ghi study_time_daily: %w", err)
	}
	return tag.RowsAffected(), nil
}

// AgeTickets lùi `created_at`, `waiting_since`, `claimed_at`, `answered_at`, `closed_at` của MỘT ticket `hours` giờ (nếu chỉ lùi created_at thì ca "quá hạn" không bao giờ quá hạn: SLA tính từ waiting_since).
// Không đổi trạng thái, nội dung, `sla_stage`. Trả số dòng đổi (0 khi ticket không thuộc lớp).
func AgeTickets(ctx context.Context, pool *pgxpool.Pool, courseID, ticketID uuid.UUID, hours int) (int64, error) {
	if hours < 1 {
		return 0, errors.New("--hours phải ≥ 1")
	}
	tag, err := pool.Exec(ctx, `
update escalation_tickets set
  created_at = created_at - make_interval(hours => $3),
  waiting_since = waiting_since - make_interval(hours => $3),
  claimed_at = claimed_at - make_interval(hours => $3),
  answered_at = answered_at - make_interval(hours => $3),
  closed_at = closed_at - make_interval(hours => $3)
where course_id = $1 and id = $2`, courseID, ticketID, hours)
	if err != nil {
		return 0, fmt.Errorf("lùi ticket: %w", err)
	}
	return tag.RowsAffected(), nil
}

// AgeActivity đưa `created_at` của tin chat (USER), thread và bình luận của lớp nằm trong tuần hiện tại về một mốc ngẫu nhiên-nhưng-ổn-định (băm của `id`) trong `weeks` tuần TRƯỚC tuần hiện tại.
// Chạy lại không đổi gì thêm: dòng đã ở trước đầu tuần hiện tại không bị chạm. Không đổi nội dung; mọi truy vấn đếm theo tuần lọc bằng `created_at`, không bằng khoảng `id`.
func AgeActivity(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, weeks int, now time.Time) (int64, error) {
	if weeks < 1 || weeks > 52 {
		return 0, errors.New("--weeks phải từ 1 đến 52")
	}
	cur := monday(now)
	from := time.Date(cur.Year(), cur.Month(), cur.Day(), 0, 0, 0, 0, ict)
	var total int64
	for _, q := range []string{
		`update chat_messages m set created_at = $2 - make_interval(days => abs(hashtext(m.id::text)) % ($3 * 7), hours => 1 + abs(hashtext(m.id::text || 'h')) % 20)
		 where m.course_id = $1 and m.role = 'USER' and m.created_at >= $2`,
		`update forum_threads t set created_at = $2 - make_interval(days => abs(hashtext(t.id::text)) % ($3 * 7), hours => 1 + abs(hashtext(t.id::text || 'h')) % 20)
		 where t.course_id = $1 and t.created_at >= $2`,
		`update forum_posts p set created_at = $2 - make_interval(days => abs(hashtext(p.id::text)) % ($3 * 7), hours => 1 + abs(hashtext(p.id::text || 'h')) % 20)
		 where p.course_id = $1 and p.kind = 'HUMAN' and p.created_at >= $2`,
	} {
		tag, err := pool.Exec(ctx, q, courseID, from, weeks)
		if err != nil {
			return total, fmt.Errorf("rải hoạt động: %w", err)
		}
		total += tag.RowsAffected()
	}
	return total, nil
}
