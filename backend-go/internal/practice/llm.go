package practice

import (
	"context"
	"crypto/sha256"
	_ "embed" // lời nhắc cố định nằm cạnh mã
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/llm"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

//go:embed prompts/explain.txt
var explainPrompt string

// Giới hạn và hạn cache của phần dùng LLM (SRS FR-9, FR-10).
const (
	ExplainTTL     = 24 * time.Hour
	FeedbackPerHr  = 10
	feedbackWindow = time.Hour
)

// LLMClient là phần của `llm.Client` mà luyện đề dùng: một `Stream` (giải thích) và một `Structured` (nhận xét tự luận). Khác `internal/quiz` (không import llm), gói này được dùng LLM.
type LLMClient interface {
	Stream(ctx context.Context, r llm.Request) (<-chan llm.Chunk, error)
	Structured(ctx context.Context, r llm.Request, schema json.RawMessage) (json.RawMessage, error)
}

// ErrExplainFailed: LLM không trả được lời giải thích (chỉ phần giải thích lỗi, phản hồi đúng / sai còn nguyên).
var ErrExplainFailed = errors.New("practice: không giải thích được")

// ExplainJob là một lượt giải thích đã qua mọi kiểm tra: `Cached` khác rỗng thì phát thẳng, không gọi LLM.
type ExplainJob struct {
	Cached string
	svc    *Service
	req    llm.Request
	key    string
	ctx    context.Context
}

// PrepareExplain kiểm điều kiện (FR-9) rồi dựng việc: câu đã trả lời (409 NOT_ANSWERED); lượt MOCK phải đã nộp (409 NOT_SUBMITTED); chỉ câu có đáp án đúng (ESSAY → 422 INVALID_QUESTION).
func (s *Service) PrepareExplain(ctx context.Context, userID, courseID, attemptID, itemID uuid.UUID) (*ExplainJob, error) {
	q := store.New(s.Pool)
	att, err := owned(ctx, q, userID, courseID, attemptID, false)
	if err != nil {
		return nil, err
	}
	it, err := q.PracticeItemGet(ctx, store.PracticeItemGetParams{CourseID: courseID, AttemptID: attemptID, ItemID: itemID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound()
	}
	if err != nil {
		return nil, fmt.Errorf("practice: đọc câu để giải thích: %w", err)
	}
	sn, err := parseSnapshot(it.Question)
	if err != nil {
		return nil, err
	}
	switch {
	case sn.Type == "ESSAY":
		return nil, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidQuestion)
	case it.Answer == nil:
		return nil, apierr.New(http.StatusConflict, apierr.NotAnswered)
	case att.Mode == store.PracticeModeMOCK && att.Status == store.PracticeStatusINPROGRESS:
		return nil, apierr.New(http.StatusConflict, apierr.NotSubmitted)
	}
	if s.LLM == nil {
		return nil, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable)
	}
	sum := sha256.Sum256(compact(it.Answer))
	key := appredis.Key("practice", "explain", it.QuestionID.String(), fmt.Sprint(it.QuestionVersion), hex.EncodeToString(sum[:8]))
	if s.Redis != nil {
		if b, err := s.Redis.Get(ctx, key).Bytes(); err == nil && len(b) > 0 {
			return &ExplainJob{Cached: string(b)}, nil
		}
	}
	temp := 0.3
	course := courseID
	user := userID
	return &ExplainJob{svc: s, key: key, ctx: llm.WithIdentity(context.WithoutCancel(ctx), llm.Identity{UserID: &user, CourseID: &course}), req: llm.Request{
		Task: llm.TaskChat, Params: llm.Params{Temperature: &temp, MaxTokens: 400},
		Messages: []llm.Message{{Role: "system", Content: explainPrompt}, {Role: "user", Content: explainContext(sn, it.AnswerKey, it.Answer)}},
	}}, nil
}

// Run phát chữ qua emit (một lời gọi `llm.Stream`, làn INTERACTIVE của tác vụ CHAT); xong thì cache 24 giờ. Huỷ ctx của client dừng việc phát nhưng KHÔNG bỏ cache của lượt đã hoàn tất.
func (j *ExplainJob) Run(ctx context.Context, emit func(string) error) error {
	if j.Cached != "" {
		return emit(j.Cached)
	}
	gctx, cancel := context.WithCancel(j.ctx)
	defer cancel()
	go func() { // client ngắt → dừng sinh
		select {
		case <-ctx.Done():
			cancel()
		case <-gctx.Done():
		}
	}()
	ch, err := j.svc.LLM.Stream(gctx, j.req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExplainFailed, err)
	}
	var sb strings.Builder
	for c := range ch {
		switch {
		case c.Err != nil:
			return fmt.Errorf("%w: %v", ErrExplainFailed, c.Err)
		case c.Degraded: // câu suy giảm của gateway không phải lời giải thích
			return ErrExplainFailed
		case c.Text != "":
			sb.WriteString(c.Text)
			if err := emit(c.Text); err != nil {
				return err
			}
		}
	}
	if sb.Len() > 0 && j.svc.Redis != nil {
		_ = j.svc.Redis.Set(context.WithoutCancel(ctx), j.key, sb.String(), ExplainTTL).Err()
	}
	return nil
}

// explainContext dựng tin người dùng: chỉ nội dung câu hỏi, không định danh ai.
func explainContext(sn snapshot, key, answer json.RawMessage) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ĐỀ: %s\n", strings.TrimSpace(sn.Stem))
	for i, o := range sn.Options {
		fmt.Fprintf(&b, "%c. %s\n", 'A'+i, o.Body)
	}
	fmt.Fprintf(&b, "ĐÁP ÁN ĐÚNG: %s\n", describe(sn, key))
	fmt.Fprintf(&b, "SINH VIÊN CHỌN: %s\n", describe(sn, answer))
	if sn.Explanation != nil && strings.TrimSpace(*sn.Explanation) != "" {
		fmt.Fprintf(&b, "LỜI GIẢI CÓ SẴN: %s\n", strings.TrimSpace(*sn.Explanation))
	}
	var cites []struct {
		Quote string `json:"quote"`
	}
	if json.Unmarshal(sn.Citations, &cites) == nil {
		for _, c := range cites {
			if q := strings.TrimSpace(c.Quote); q != "" {
				fmt.Fprintf(&b, "TRÍCH ĐOẠN TÀI LIỆU: %s\n", q)
			}
		}
	}
	return b.String()
}

// describe chuyển đáp án (khoá hoặc câu trả lời) thành chữ: MCQ → chữ cái + nội dung; đúng–sai; trả lời ngắn.
func describe(sn snapshot, raw json.RawMessage) string {
	var v struct {
		OptionIDs []uuid.UUID `json:"option_ids"`
		Value     *bool       `json:"value"`
		Text      string      `json:"text"`
		Accepted  []string    `json:"accepted"`
	}
	if json.Unmarshal(raw, &v) != nil {
		return "(không rõ)"
	}
	switch {
	case len(v.OptionIDs) > 0:
		var parts []string
		for _, id := range v.OptionIDs {
			for i, o := range sn.Options {
				if o.ID == id {
					parts = append(parts, fmt.Sprintf("%c. %s", 'A'+i, o.Body))
				}
			}
		}
		return strings.Join(parts, "; ")
	case v.Value != nil:
		if *v.Value {
			return "Đúng"
		}
		return "Sai"
	case len(v.Accepted) > 0:
		return strings.Join(v.Accepted, " / ")
	case v.Text != "":
		return v.Text
	}
	return "(bỏ trống)"
}

// FeedbackOut là phản hồi `POST …/feedback`: nhận xét gợi ý, không điểm.
type FeedbackOut struct {
	Overall  string                      `json:"overall"`
	Criteria []grading.FeedbackCriterion `json:"criteria"`
	Injected bool                        `json:"injected"`
}

// Feedback nhận xét một câu ESSAY đã trả lời (FR-10, US-P9-08): quá dài → 422 TOO_LONG, quá 10 lần / giờ → 429, cả hai trước khi tốn lời gọi LLM.
// Bài viết không bao giờ vào log; không ghi `grade_entries`.
func (s *Service) Feedback(ctx context.Context, userID, courseID, attemptID, itemID uuid.UUID) (FeedbackOut, error) {
	q := store.New(s.Pool)
	att, err := owned(ctx, q, userID, courseID, attemptID, false)
	if err != nil {
		return FeedbackOut{}, err
	}
	it, err := q.PracticeItemGet(ctx, store.PracticeItemGetParams{CourseID: courseID, AttemptID: attemptID, ItemID: itemID})
	if errors.Is(err, pgx.ErrNoRows) {
		return FeedbackOut{}, notFound()
	}
	if err != nil {
		return FeedbackOut{}, fmt.Errorf("practice: đọc câu để nhận xét: %w", err)
	}
	sn, err := parseSnapshot(it.Question)
	if err != nil {
		return FeedbackOut{}, err
	}
	switch {
	case sn.Type != "ESSAY":
		return FeedbackOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidQuestion)
	case it.Answer == nil:
		return FeedbackOut{}, apierr.New(http.StatusConflict, apierr.NotAnswered)
	case att.Mode == store.PracticeModeMOCK && att.Status == store.PracticeStatusINPROGRESS:
		return FeedbackOut{}, apierr.New(http.StatusConflict, apierr.NotSubmitted)
	}
	var ans struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal(it.Answer, &ans)
	if utf8.RuneCountInString(ans.Text) > MaxEssayChars {
		return FeedbackOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.TooLong)
	}
	var key struct {
		ModelAnswer string `json:"model_answer"`
		Rubric      string `json:"rubric"`
	}
	_ = json.Unmarshal(it.AnswerKey, &key)
	if s.LLM == nil {
		return FeedbackOut{}, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable)
	}
	if err := s.feedbackLimit(ctx, userID); err != nil {
		return FeedbackOut{}, err
	}
	user, course := userID, courseID
	res, err := grading.Feedback(llm.WithIdentity(ctx, llm.Identity{UserID: &user, CourseID: &course}), s.LLM,
		grading.FeedbackInput{Stem: sn.Stem, ModelAnswer: key.ModelAnswer, Rubric: key.Rubric, Text: ans.Text})
	if err != nil {
		return FeedbackOut{}, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable)
	}
	return FeedbackOut{Overall: res.Overall, Criteria: res.Criteria, Injected: res.Injected}, nil
}

// feedbackLimit: ≤ 10 lần / giờ / sinh viên (cửa sổ cố định theo giờ). Redis lỗi hoặc không có → cho qua (fail-open như hạn mức khác của gateway).
func (s *Service) feedbackLimit(ctx context.Context, userID uuid.UUID) error {
	if s.Redis == nil {
		return nil
	}
	now := s.now()
	key := appredis.Key("rl", "practice-feedback", userID.String(), fmt.Sprint(now.Unix()/int64(feedbackWindow.Seconds())))
	n, err := s.Redis.Incr(ctx, key).Result()
	if err != nil {
		return nil
	}
	if n == 1 {
		_ = s.Redis.Expire(ctx, key, feedbackWindow).Err()
	}
	if n > FeedbackPerHr {
		wait := int(feedbackWindow.Seconds()) - int(now.Unix()%int64(feedbackWindow.Seconds()))
		return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(wait)
	}
	return nil
}
