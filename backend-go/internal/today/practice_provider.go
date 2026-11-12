package today

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/practice"
	"github.com/edupilot/backend-go/internal/store"
)

// PracticeRecent: chủ đề đã luyện trong chừng này thì không nhắc ôn lại (SRS FR-19, US-P9-09 AC3).
const PracticeRecent = 3 * 24 * time.Hour

// PracticeProvider: PRACTICE_WEAK_TOPIC (bậc 85, chỉ STUDENT) — mỗi lớp ACTIVE của sinh viên tối đa một mục "Ôn chủ đề {tên}" cho chủ đề yếu nhất chưa luyện trong 3 ngày.
// Dữ liệu chỉ của chính người xem. Luyện / nộp → outbox `practice.updated` → mục biến mất ≤ 5 s.
type PracticeProvider struct {
	Pool *pgxpool.Pool
	Now  func() time.Time
}

// Name implements Provider.
func (PracticeProvider) Name() string { return "practice" }

// Items implements Provider.
func (p PracticeProvider) Items(ctx context.Context, v Viewer, s Scope) ([]Item, error) {
	courses := v.Scoped(s, "STUDENT")
	if v.Role != RoleStudent || len(courses) == 0 {
		return nil, nil
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	rows, err := store.New(p.Pool).PracticeTopicStatsMany(ctx, store.PracticeTopicStatsManyParams{StudentID: v.UserID, CourseIds: courseIDs(courses)})
	if err != nil {
		return nil, fmt.Errorf("today: điểm yếu luyện đề: %w", err)
	}
	by := map[uuid.UUID][]practice.TopicStat{}
	last := map[string]time.Time{}
	for _, r := range rows {
		by[r.CourseID] = append(by[r.CourseID], practice.TopicStat{Topic: r.Topic, Answered: int(r.Answered), Wrong: int(r.Wrong)})
		last[r.CourseID.String()+"|"+r.Topic] = r.LastStartedAt
	}
	var out []Item
	for _, c := range courses {
		for _, t := range practice.WeakTopics(by[c.ID]) {
			if now().Sub(last[c.ID.String()+"|"+t.Topic]) < PracticeRecent {
				continue
			}
			out = append(out, mk(Item{
				ID: "PRACTICE_WEAK_TOPIC:" + c.ID.String(), Kind: KindPracticeWeakTopic, Tier: TierPracticeWeakTopic, Course: courseRef(c),
				Href: "/practice?topic=" + url.QueryEscape(t.Topic) + "&course=" + c.ID.String(), Title: "Ôn chủ đề " + t.Topic,
				Reason: fmt.Sprintf("Sai %d / %d câu gần đây.", t.Wrong, t.Answered), EstimateMinutes: 10,
			}))
			break
		}
	}
	return out, nil
}

func courseIDs(cs []CourseRef) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.ID.String()
	}
	return out
}
