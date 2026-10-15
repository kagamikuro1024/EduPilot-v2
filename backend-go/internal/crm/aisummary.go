package crm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/store"
)

const aiSystem = `Bạn là trợ lý của giảng viên. Dựa DUY NHẤT vào các con số trong dữ liệu JSON được gửi, hãy viết nhận xét tổng hợp ngắn bằng tiếng Việt về MỘT sinh viên (gọi là "sinh viên").
Quy tắc: không bịa số liệu; không đoán nguyên nhân cá nhân; không nêu hay đoán tên người; không đánh giá điểm số; giọng trung tính, tôn trọng.
Trả về JSON: summary (≤ 600 ký tự), strengths, concerns, suggestions (mỗi mảng 0–3 mục ngắn).`

const aiSchema = `{"type":"object","additionalProperties":false,"required":["summary","strengths","concerns","suggestions"],"properties":{
"summary":{"type":"string","maxLength":600},
"strengths":{"type":"array","maxItems":3,"items":{"type":"string"}},
"concerns":{"type":"array","maxItems":3,"items":{"type":"string"}},
"suggestions":{"type":"array","maxItems":3,"items":{"type":"string"}}}}`

// AISummaryDraft là bản nháp nhận xét tổng hợp. KHÔNG được lưu: `Lưu` đi qua POST observations kind=AI_SUMMARY.
type AISummaryDraft struct {
	Summary     string   `json:"summary"`
	Strengths   []string `json:"strengths"`
	Concerns    []string `json:"concerns"`
	Suggestions []string `json:"suggestions"`
}

// aiInput là TOÀN BỘ thứ gửi cho LLM: chỉ số liệu. Không tên, MSSV, email, nội dung ghi chú / chat (SRS 4.8.5).
type aiInput struct {
	Attendance         map[string]int `json:"attendance"`
	Participation      map[string]any `json:"participation"`
	StudyMinutesByWeek []*int         `json:"study_minutes_by_week"` // 4 tuần gần nhất, cũ → mới; null = không đo
	AsksByWeek         []int          `json:"asks_by_week"`
	PostsByWeek        []int          `json:"posts_by_week"`
	RiskReasonCodes    []string       `json:"risk_reason_codes"`
}

func (s *Service) aiTimeout() time.Duration {
	if s.AITimeout > 0 {
		return s.AITimeout
	}
	return 20 * time.Second
}

func (s *Service) aiPerHour() int {
	if s.AIPerHour > 0 {
		return s.AIPerHour
	}
	return 10
}

// gatherAIInput đọc số liệu của MỘT sinh viên (bốn tuần trọn gần nhất). hasData = có ít nhất một số liệu nào.
func (s *Service) gatherAIInput(ctx context.Context, courseID, studentID uuid.UUID) (aiInput, bool, error) {
	q := store.New(s.Pool)
	prof, err := q.CrmProfileGet(ctx, store.CrmProfileGetParams{CourseID: courseID, StudentID: studentID})
	if err != nil {
		return aiInput{}, false, wrap("đọc số liệu", err)
	}
	cw := weekStart(s.now())
	from := cw.AddDate(0, 0, -7*weeksShown)
	fromDay := pgtype.Date{Time: time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
	sid := &studentID
	act, err := q.CrmRiskActivity(ctx, store.CrmRiskActivityParams{CourseID: courseID, StudentID: sid, FromDay: fromDay, FromAt: from, ToAt: cw})
	if err != nil {
		return aiInput{}, false, wrap("hoạt động theo tuần", err)
	}
	study, err := q.CrmRiskStudy(ctx, store.CrmRiskStudyParams{CourseID: courseID, StudentID: sid, FromDay: fromDay, ToDay: pgtype.Date{Time: fromDay.Time.AddDate(0, 0, 7*weeksShown), Valid: true}})
	if err != nil {
		return aiInput{}, false, wrap("thời gian học theo tuần", err)
	}
	in := aiInput{
		Attendance:         map[string]int{"recorded": int(prof.Recorded), "present": int(prof.Present), "late": int(prof.Late), "excused": int(prof.Excused), "absent": int(prof.Absent)},
		Participation:      map[string]any{"speak_count": int(prof.SpeakCount), "points_total": prof.PointsTotal.StringFixed(2)},
		StudyMinutesByWeek: make([]*int, weeksShown), AsksByWeek: make([]int, weeksShown), PostsByWeek: make([]int, weeksShown), RiskReasonCodes: []string{},
	}
	has := prof.Recorded > 0 || prof.SpeakCount > 0
	for _, a := range act {
		in.AsksByWeek[a.K], in.PostsByWeek[a.K] = int(a.Asks), int(a.Posts)
		has = has || a.Asks+a.Posts > 0
	}
	if prof.Tracked { // chỉ khi đang được đo mới có số phút (null = không đo, không phải 0)
		for i := range in.StudyMinutesByWeek {
			m := 0
			in.StudyMinutesByWeek[i] = &m
		}
		for _, w := range study {
			m := int(w.Seconds / 60)
			in.StudyMinutesByWeek[w.K] = &m
			has = has || w.Seconds > 0
		}
	}
	var rs []Reason
	if json.Unmarshal(prof.Reasons, &rs) == nil {
		for _, r := range rs {
			in.RiskReasonCodes = append(in.RiskReasonCodes, r.Code)
		}
	}
	return in, has, nil
}

// aiRateKey: một người tối đa AI_SUMMARY_PER_HOUR lượt mỗi giờ (cửa sổ cố định theo giờ).
func aiRateKey(actor uuid.UUID, now time.Time) string {
	return "ep:aisum:" + actor.String() + ":" + strconv.FormatInt(now.Unix()/3600, 10)
}

// AISummary soạn nhận xét tổng hợp (SRS 4.8.5): chưa có dữ liệu → 409 NOT_ENOUGH_DATA (không gọi LLM, không tính lượt); > AI_SUMMARY_PER_HOUR → 429; đúng MỘT llm.Structured (INSIGHT, làn
// NEAR_REALTIME) với đầu vào chỉ là số liệu; quá AI_SUMMARY_TIMEOUT → 504; LLM hỏng → 503 LLM_UNAVAILABLE / OVERLOADED (kèm retry_after). Đầu ra quét `privacy.Detect` theo roster rồi `Redact`;
// KHÔNG lưu gì.
func (s *Service) AISummary(ctx context.Context, actor, courseID, studentID uuid.UUID) (AISummaryDraft, error) {
	q := store.New(s.Pool)
	if err := s.requireActive(ctx, q, courseID, studentID); err != nil {
		return AISummaryDraft{}, err
	}
	in, has, err := s.gatherAIInput(ctx, courseID, studentID)
	if err != nil {
		return AISummaryDraft{}, err
	}
	if !has {
		return AISummaryDraft{}, apierr.New(http.StatusConflict, apierr.NotEnoughData)
	}
	if s.Redis != nil {
		k := aiRateKey(actor, s.now())
		n, rerr := s.Redis.Incr(ctx, k).Result()
		if rerr == nil {
			_ = s.Redis.Expire(ctx, k, time.Hour).Err()
			if int(n) > s.aiPerHour() {
				return AISummaryDraft{}, apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(int(time.Hour.Seconds()) - int(s.now().Unix()%3600))
			}
		}
	}
	if s.LLM == nil {
		return AISummaryDraft{}, apierr.New(http.StatusServiceUnavailable, apierr.LLMUnavailable).WithRetryAfter(30)
	}
	payload, err := json.Marshal(in)
	if err != nil {
		return AISummaryDraft{}, fmt.Errorf("crm: mã hoá số liệu: %w", err)
	}
	lane := llm.LaneNearRealtime
	temp := 0.3
	cctx, cancel := context.WithTimeout(llm.WithIdentity(ctx, llm.Identity{UserID: &actor, CourseID: &courseID}), s.aiTimeout())
	defer cancel()
	raw, err := s.LLM.Structured(cctx, llm.Request{Task: llm.TaskInsight, Lane: &lane, Params: llm.Params{Temperature: &temp},
		Messages: []llm.Message{{Role: "system", Content: aiSystem}, {Role: "user", Content: string(payload)}}}, json.RawMessage(aiSchema))
	if err != nil {
		var over *llm.ErrOverloaded
		switch {
		case errors.Is(err, context.DeadlineExceeded) || errors.Is(cctx.Err(), context.DeadlineExceeded):
			return AISummaryDraft{}, apierr.New(http.StatusGatewayTimeout, apierr.DeadlineExceeded)
		case errors.As(err, &over):
			return AISummaryDraft{}, apierr.New(http.StatusServiceUnavailable, apierr.Overloaded).WithRetryAfter(max(1, int(over.RetryAfter.Seconds())))
		default:
			return AISummaryDraft{}, apierr.New(http.StatusServiceUnavailable, apierr.LLMUnavailable).WithRetryAfter(30)
		}
	}
	var d AISummaryDraft
	if err := json.Unmarshal(raw, &d); err != nil {
		return AISummaryDraft{}, apierr.New(http.StatusServiceUnavailable, apierr.LLMUnavailable).WithRetryAfter(30)
	}
	return s.cleanDraft(ctx, courseID, d)
}

// cleanDraft cắt về giới hạn của schema và che mọi tên / MSSV / email mà mô hình lỡ bịa ra (quét theo roster của lớp).
func (s *Service) cleanDraft(ctx context.Context, courseID uuid.UUID, d AISummaryDraft) (AISummaryDraft, error) {
	scrub := func(t string, maxRunes int) (string, error) {
		if r := []rune(t); len(r) > maxRunes {
			t = string(r[:maxRunes])
		}
		if s.PII == nil {
			return t, nil
		}
		fs, err := s.PII.Detect(ctx, courseID, t)
		if err != nil {
			return "", unavailable()
		}
		return privacy.Redact(t, fs), nil
	}
	var err error
	if d.Summary, err = scrub(d.Summary, 600); err != nil {
		return AISummaryDraft{}, err
	}
	for _, list := range []*[]string{&d.Strengths, &d.Concerns, &d.Suggestions} {
		if len(*list) > 3 {
			*list = (*list)[:3]
		}
		for i, t := range *list {
			if (*list)[i], err = scrub(t, 200); err != nil {
				return AISummaryDraft{}, err
			}
		}
		if *list == nil {
			*list = []string{}
		}
	}
	return d, nil
}
