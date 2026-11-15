package practice

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/store"
)

// Cursor là con trỏ lịch sử (started_at, id) — mới nhất trước.
type Cursor struct {
	At time.Time
	ID uuid.UUID
}

// History là `GET …/practice/history` (FR-18): chỉ lượt của chính người gọi trong lớp này; `fetch` = limit + 1 (hàng dư để biết còn trang sau).
func (s *Service) History(ctx context.Context, userID, courseID uuid.UUID, cur *Cursor, fetch int) ([]HistoryRow, error) {
	arg := store.PracticeHistoryParams{CourseID: courseID, StudentID: userID, MaxRows: int32(fetch)} //nolint:gosec // ≤ 101
	if cur != nil {
		arg.CursorAt, arg.CursorID = &cur.At, &cur.ID
	}
	rows, err := store.New(s.Pool).PracticeHistory(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("practice: đọc lịch sử: %w", err)
	}
	out := make([]HistoryRow, len(rows))
	for i, r := range rows {
		h := HistoryRow{ID: r.ID, Mode: string(r.Mode), Topic: r.Topic, Status: string(r.Status), StartedAt: r.StartedAt, SubmittedAt: r.SubmittedAt, ItemCount: int(r.ItemCount)}
		if r.CorrectCount != nil {
			c := int(*r.CorrectCount)
			h.Correct = &c
		}
		out[i] = h
	}
	return out, nil
}

// minAnswered: dưới ngưỡng này khối "Chủ đề cần ôn" ẩn (US-P9-09 AC2).
const (
	minAnswered      = 10
	minTopicAnswered = 3
	maxWeakTopics    = 3
	maxDocsPerTopic  = 2
)

// Weakness là `GET …/practice/weakness` (FR-18, US-P9-09 AC2): tối đa 3 chủ đề có tỉ lệ sai cao nhất (≥ 3 câu đã làm), mỗi chủ đề tối đa 2 tài liệu gợi ý từ `citations` của câu sai.
func (s *Service) Weakness(ctx context.Context, userID, courseID uuid.UUID) (WeaknessView, error) {
	q := store.New(s.Pool)
	out := WeaknessView{Topics: []WeakTopic{}}
	stats, err := q.PracticeTopicStats(ctx, store.PracticeTopicStatsParams{CourseID: courseID, StudentID: userID})
	if err != nil {
		return out, fmt.Errorf("practice: đọc thống kê chủ đề: %w", err)
	}
	in := make([]TopicStat, len(stats))
	for i, st := range stats {
		in[i] = TopicStat{Topic: st.Topic, Answered: int(st.Answered), Wrong: int(st.Wrong)}
	}
	cand := WeakTopics(in)
	for _, st := range cand {
		w := WeakTopic{Topic: st.Topic, Answered: st.Answered, Wrong: st.Wrong, Documents: []DocSuggested{}}
		cites, err := q.PracticeWrongCitations(ctx, store.PracticeWrongCitationsParams{CourseID: courseID, StudentID: userID, Topic: st.Topic})
		if err != nil {
			return out, fmt.Errorf("practice: đọc trích dẫn câu sai: %w", err)
		}
		count := map[uuid.UUID]int{}
		for _, raw := range cites {
			var cs []struct {
				DocumentID uuid.UUID `json:"document_id"`
			}
			if json.Unmarshal(raw, &cs) != nil {
				continue
			}
			seen := map[uuid.UUID]bool{}
			for _, c := range cs {
				if c.DocumentID != uuid.Nil && !seen[c.DocumentID] {
					seen[c.DocumentID] = true
					count[c.DocumentID]++
				}
			}
		}
		ids := make([]string, 0, len(count))
		for id := range count {
			ids = append(ids, id.String())
		}
		if len(ids) > 0 {
			docs, err := q.PracticeDocTitles(ctx, store.PracticeDocTitlesParams{CourseID: courseID, Ids: ids})
			if err != nil {
				return out, fmt.Errorf("practice: đọc tên tài liệu: %w", err)
			}
			sort.Slice(docs, func(i, j int) bool {
				if count[docs[i].ID] != count[docs[j].ID] {
					return count[docs[i].ID] > count[docs[j].ID]
				}
				return docs[i].Title < docs[j].Title
			})
			for i, d := range docs {
				if i == maxDocsPerTopic {
					break
				}
				w.Documents = append(w.Documents, DocSuggested{DocumentID: d.ID, Title: d.Title})
			}
		}
		out.Topics = append(out.Topics, w)
	}
	return out, nil
}

// Signal là một tín hiệu lỗ hổng tổng hợp theo chủ đề — KHÔNG có định danh sinh viên (FR-20, TestWrongAnswerSignalsNoIdentity).
type Signal struct {
	Topic             string      `json:"topic"`
	WrongAnswers      int         `json:"wrong_answers"`
	StudentsAffected  int         `json:"students_affected"`
	SampleQuestionIDs []uuid.UUID `json:"sample_question_ids"`
}

// WrongAnswerSignals trả câu sai theo chủ đề trong [from, to); chủ đề có < KAnonymity sinh viên sai bị loại. Hàm Go (không route): đầu vào báo cáo lỗ hổng của P10.
func (s *Service) WrongAnswerSignals(ctx context.Context, courseID uuid.UUID, from, to time.Time) ([]Signal, error) {
	rows, err := store.New(s.Pool).PracticeSignals(ctx, store.PracticeSignalsParams{CourseID: courseID, FromAt: from, ToAt: to, MinStudents: KAnonymity})
	if err != nil {
		return nil, fmt.Errorf("practice: tín hiệu tổng hợp: %w", err)
	}
	out := make([]Signal, len(rows))
	for i, r := range rows {
		out[i] = Signal{Topic: r.Topic, WrongAnswers: int(r.WrongAnswers), StudentsAffected: int(r.StudentsAffected), SampleQuestionIDs: r.SampleQuestionIds}
	}
	return out, nil
}

// TopicStat là số câu đã chấm / sai của một chủ đề trong 30 ngày.
type TopicStat struct {
	Topic           string
	Answered, Wrong int
}

// WeakTopics chọn chủ đề yếu (FR-18): cần ≥ 10 câu đã chấm toàn lớp; chủ đề ≥ 3 câu và có sai; tỉ lệ sai giảm dần, hoà thì nhiều sai hơn; tối đa 3.
// Dùng chung cho `Weakness` và nguồn Hôm nay để hai nơi không lệch nhau.
func WeakTopics(stats []TopicStat) []TopicStat {
	total := 0
	for _, st := range stats {
		total += st.Answered
	}
	if total < minAnswered {
		return nil
	}
	var cand []TopicStat
	for _, st := range stats {
		if st.Answered >= minTopicAnswered && st.Wrong > 0 {
			cand = append(cand, st)
		}
	}
	sort.SliceStable(cand, func(i, j int) bool {
		a, b := cand[i], cand[j]
		if l, r := a.Wrong*b.Answered, b.Wrong*a.Answered; l != r {
			return l > r
		}
		if a.Wrong != b.Wrong {
			return a.Wrong > b.Wrong
		}
		return a.Topic < b.Topic
	})
	if len(cand) > maxWeakTopics {
		cand = cand[:maxWeakTopics]
	}
	return cand
}
