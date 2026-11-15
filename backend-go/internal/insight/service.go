package insight

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/clock"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/store"
)

// Hằng của báo cáo.
const (
	KindGenerate    = "insight.knowledge_gap"
	TypeReady       = "INSIGHT_READY"
	MinQuestions    = 20   // dưới mức này: 409 NOT_ENOUGH_DATA (SRS 3)
	MaxRows         = 3000 // trần số câu hỏi mỗi nguồn / kỳ
	MinStudents     = 3    // chủ đề < 3 sinh viên: không câu mẫu (k-ẩn danh, FR-13)
	MaxExamples     = 5
	TopRanked       = 8   // `/insights` hiển thị tối đa 8 chủ đề đầu
	DefaultPeriod   = 14  // Q-E-7: kỳ ∈ {7, 14, 30, toàn kỳ}, mặc định 14
	WholeTerm       = 120 // «toàn kỳ» ≈ một học kỳ: ponytail — số cố định, lớp không có ngày bắt đầu / kết thúc kỳ trong schema
	RatePerHour     = 3   // việc generate / 60 phút / lớp
	DefaultTau      = 0.80
	practiceTau     = 0.75
	namedByLLM      = "llm"
	namedByFallback = "fallback"
)

// Service: tạo / đọc báo cáo. Gateway chỉ dùng Generate / Latest / Export; worker dùng Run (đăng ký qua Register).
type Service struct {
	Pool   *pgxpool.Pool
	Redis  *appredis.Client // trần 3 việc / giờ / lớp; nil = không giới hạn
	Jobs   *jobs.Service
	Runner *jobs.Runner // nil = chỉ `Progress(pct)`, không có chi tiết giai đoạn
	LLM    llm.Client
	Det    *privacy.Detector
	Notify *notify.Service
	Clock  clock.Clock
	Log    *slog.Logger
	// Tau: ngưỡng cosine gom cụm (0 = DefaultTau). Phụ thuộc mô hình embedding: hiệu chỉnh cùng P-11-19.
	Tau float64
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now()
	}
	return s.Clock.Now()
}

func (s *Service) tau() float64 {
	if s.Tau > 0 {
		return s.Tau
	}
	return DefaultTau
}

// Register đăng ký việc trên worker.
func (s *Service) Register(r *jobs.Runner) {
	s.Runner = r
	r.Register(KindGenerate, s.run)
}

type payload struct {
	CourseID   uuid.UUID `json:"course_id"`
	PeriodDays int       `json:"period_days"`
}

// GenerateOut: 202 (JobID) hoặc 200 reused (ReportID).
type GenerateOut struct {
	JobID    *uuid.UUID `json:"job_id,omitempty"`
	ReportID *uuid.UUID `json:"report_id,omitempty"`
	Reused   bool       `json:"reused,omitempty"`
}

func (s *Service) window(days int) (time.Time, time.Time) {
	to := s.now().Add(time.Minute) // biên trên +1 phút: đồng hồ gateway có thể chậm hơn Postgres, câu hỏi vừa gửi không được rơi ngoài kỳ
	return to.AddDate(0, 0, -days), to
}

// Generate: kiểm kỳ → so data_watermark (không có dữ liệu mới → reused, 0 lời gọi LLM, không tính trần) → đủ ≥ 20 câu → trần 3 việc / giờ / lớp → xếp việc.
func (s *Service) Generate(ctx context.Context, actor, courseID uuid.UUID, periodDays int) (GenerateOut, error) {
	if periodDays == 0 {
		periodDays = DefaultPeriod
	}
	if !slices.Contains([]int{7, 14, 30, WholeTerm}, periodDays) {
		return GenerateOut{}, apierr.Validation(apierr.FieldError{Field: "period_days", Code: "invalid", Message: "Kỳ là 7, 14, 30 hoặc 120 (toàn kỳ) ngày."})
	}
	q := store.New(s.Pool)
	from, to := s.window(periodDays)
	wm, err := q.InsightWatermark(ctx, store.InsightWatermarkParams{CourseID: courseID, FromTs: from, ToTs: to})
	if err != nil {
		return GenerateOut{}, fmt.Errorf("insight: mốc dữ liệu: %w", err)
	}
	if last, err := q.InsightReportLatestFor(ctx, store.InsightReportLatestForParams{CourseID: courseID, PeriodDays: int32(periodDays)}); err == nil { //nolint:gosec // 1..90
		if !wm.After(last.DataWatermark) {
			return GenerateOut{Reused: true, ReportID: &last.ID}, nil
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return GenerateOut{}, fmt.Errorf("insight: báo cáo gần nhất: %w", err)
	}
	n, err := s.countQuestions(ctx, q, courseID, from, to)
	if err != nil {
		return GenerateOut{}, err
	}
	if n < MinQuestions {
		return GenerateOut{}, apierr.New(http.StatusConflict, apierr.NotEnoughData).WithDetails(map[string]int{"count": n, "required": MinQuestions})
	}
	if err := s.rate(ctx, courseID); err != nil {
		return GenerateOut{}, err
	}
	j, err := s.Jobs.Enqueue(ctx, actor, KindGenerate, payload{CourseID: courseID, PeriodDays: periodDays})
	if err != nil {
		return GenerateOut{}, fmt.Errorf("insight: xếp việc: %w", err)
	}
	return GenerateOut{JobID: &j.ID}, nil
}

func (s *Service) countQuestions(ctx context.Context, q *store.Queries, courseID uuid.UUID, from, to time.Time) (int, error) {
	chat, err := q.InsightChatQuestions(ctx, store.InsightChatQuestionsParams{CourseID: courseID, FromTs: from, ToTs: to, MaxRows: MaxRows})
	if err != nil {
		return 0, fmt.Errorf("insight: đếm câu hỏi chat: %w", err)
	}
	th, err := q.InsightThreadQuestions(ctx, store.InsightThreadQuestionsParams{CourseID: courseID, FromTs: from, ToTs: to, MaxRows: MaxRows})
	if err != nil {
		return 0, fmt.Errorf("insight: đếm thread: %w", err)
	}
	return len(chat) + len(th), nil
}

// rate: cửa sổ cố định theo giờ, ≤ RatePerHour việc / lớp. Redis lỗi → cho qua (như hạn mức luyện đề).
func (s *Service) rate(ctx context.Context, courseID uuid.UUID) error {
	if s.Redis == nil {
		return nil
	}
	win := int64(time.Hour.Seconds())
	now := s.now()
	key := appredis.Key("rl", "insight-generate", courseID.String(), fmt.Sprint(now.Unix()/win))
	n, err := s.Redis.Incr(ctx, key).Result()
	if err != nil {
		return nil //nolint:nilerr // fail-open có chủ ý
	}
	if n == 1 {
		_ = s.Redis.Expire(ctx, key, time.Hour).Err()
	}
	if n > RatePerHour {
		return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(int(win - now.Unix()%win))
	}
	return nil
}

// ---- đọc ----

// Report là một báo cáo trả ra API.
type Report struct {
	ID            uuid.UUID `json:"id"`
	PeriodFrom    *string   `json:"period_from"`
	PeriodTo      *string   `json:"period_to"`
	DataWatermark time.Time `json:"data_watermark"`
	CreatedAt     time.Time `json:"created_at"`
	Payload       Payload   `json:"payload"`
}

// LatestOut: báo cáo mới nhất + so với bản trước + có dữ liệu mới chưa.
type LatestOut struct {
	Report     Report `json:"report"`
	Previous   *Diff  `json:"previous,omitempty"`
	HasNewData bool   `json:"has_new_data"`
}

// Diff so với báo cáo ngay trước (cùng lớp).
type Diff struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	New       []string  `json:"new_topics"`      // tên chủ đề không có ở bản trước
	Moved     []Move    `json:"moved,omitempty"` // đổi hạng
}

// Move: chủ đề đổi hạng.
type Move struct {
	Name string `json:"name"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

func notFound() error { return apierr.New(http.StatusNotFound, apierr.NotFound) }

func toReport(id uuid.UUID, from, to pgtype.Date, raw []byte, wm, created time.Time) (Report, error) {
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return Report{}, fmt.Errorf("insight: payload hỏng: %w", err)
	}
	return Report{ID: id, PeriodFrom: dateStr(from), PeriodTo: dateStr(to), DataWatermark: wm, CreatedAt: created, Payload: p}, nil
}

// Latest trả báo cáo mới nhất + so sánh với bản trước; chưa có → 404.
func (s *Service) Latest(ctx context.Context, courseID uuid.UUID) (LatestOut, error) {
	q := store.New(s.Pool)
	rows, err := q.InsightReportLatest(ctx, courseID)
	if err != nil {
		return LatestOut{}, fmt.Errorf("insight: đọc báo cáo: %w", err)
	}
	if len(rows) == 0 {
		return LatestOut{}, notFound()
	}
	cur, err := toReport(rows[0].ID, rows[0].PeriodFrom, rows[0].PeriodTo, rows[0].Payload, rows[0].DataWatermark, rows[0].CreatedAt)
	if err != nil {
		return LatestOut{}, err
	}
	out := LatestOut{Report: cur}
	if len(rows) > 1 {
		prev, err := toReport(rows[1].ID, rows[1].PeriodFrom, rows[1].PeriodTo, rows[1].Payload, rows[1].DataWatermark, rows[1].CreatedAt)
		if err != nil {
			return LatestOut{}, err
		}
		out.Previous = diff(prev, cur)
	}
	from, to := s.window(cur.Payload.PeriodDays)
	wm, err := q.InsightWatermark(ctx, store.InsightWatermarkParams{CourseID: courseID, FromTs: from, ToTs: to})
	if err != nil {
		return LatestOut{}, fmt.Errorf("insight: mốc dữ liệu: %w", err)
	}
	out.HasNewData = wm.After(cur.DataWatermark)
	return out, nil
}

func diff(prev, cur Report) *Diff {
	d := &Diff{ID: prev.ID, CreatedAt: prev.CreatedAt, New: []string{}}
	rank := map[string]int{}
	for _, t := range prev.Payload.Topics {
		rank[t.Name] = t.Rank
	}
	for _, t := range cur.Payload.Topics {
		if r, ok := rank[t.Name]; !ok {
			d.New = append(d.New, t.Name)
		} else if r != t.Rank {
			d.Moved = append(d.Moved, Move{Name: t.Name, From: r, To: t.Rank})
		}
	}
	return d
}

// Export dựng Markdown của một báo cáo (không có định danh). Không có → 404.
func (s *Service) Export(ctx context.Context, courseID, id uuid.UUID) (string, error) {
	r, err := store.New(s.Pool).InsightReportGet(ctx, store.InsightReportGetParams{CourseID: courseID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", notFound()
	}
	if err != nil {
		return "", fmt.Errorf("insight: đọc báo cáo: %w", err)
	}
	rep, err := toReport(r.ID, r.PeriodFrom, r.PeriodTo, r.Payload, r.DataWatermark, r.CreatedAt)
	if err != nil {
		return "", err
	}
	return markdown(rep), nil
}

func markdown(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Lỗ hổng kiến thức — %d ngày gần nhất\n\n", r.Payload.PeriodDays)
	fmt.Fprintf(&b, "Tạo lúc %s · %d câu hỏi\n\n", r.CreatedAt.Format("02/01/2006 15:04"), r.Payload.Counts.Questions)
	for _, t := range r.Payload.Topics {
		fmt.Fprintf(&b, "## %d. %s\n\n%s\n\n", t.Rank, t.Name, t.Why)
		if t.Recommendation != "" {
			fmt.Fprintf(&b, "**Khuyến nghị:** %s\n\n", t.Recommendation)
		}
		for _, e := range t.Examples {
			fmt.Fprintf(&b, "- %s\n", e)
		}
		if len(t.Examples) > 0 {
			b.WriteString("\n")
		}
	}
	if len(r.Payload.Uncovered) > 0 {
		b.WriteString("## Tài liệu chưa đề cập\n\n")
		for _, u := range r.Payload.Uncovered {
			fmt.Fprintf(&b, "- %s (%d lần)\n", u.Text, u.Count)
		}
	}
	return b.String()
}

// ---- việc nền ----

// run: bọc `generate` cho Runner; tiến độ qua SSE `job.progress` kèm giai đoạn.
func (s *Service) run(ctx context.Context, j jobs.JobCtx) (any, error) {
	var p payload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, fmt.Errorf("insight: payload việc: %w", err)
	}
	return s.generate(ctx, j.OwnerID, p, func(pct int, name string) {
		if s.Runner != nil {
			s.Runner.ReportDetail(ctx, j.ID, pct, map[string]string{"stage": name})
			return
		}
		j.Progress(pct)
	})
}

// generate: COLLECT → CLUSTER (+ xếp hạng bằng code) → NAME → ghi báo cáo + thông báo. Lỗi trước NAME → việc FAILED, không có báo cáo; lỗi ở NAME → tên tạm.
func (s *Service) generate(ctx context.Context, owner uuid.UUID, p payload, stage func(pct int, name string)) (any, error) {
	ctx = llm.WithIdentity(ctx, llm.Identity{UserID: &owner, CourseID: &p.CourseID})
	stage(5, "COLLECT")
	q := store.New(s.Pool)
	from, to := s.window(p.PeriodDays)
	d, err := s.collect(ctx, q, p.CourseID, from, to)
	if err != nil {
		return nil, err
	}
	if len(d.items) < MinQuestions {
		return nil, &jobs.UserError{Code: apierr.NotEnoughData, Message: "Chưa đủ câu hỏi để gom (cần ≥ 20)."}
	}
	stage(25, "CLUSTER")
	pl, err := s.build(ctx, q, p.CourseID, p.PeriodDays, d)
	if err != nil {
		return nil, err
	}
	stage(70, "NAME")
	s.name(ctx, &pl)
	raw, err := json.Marshal(pl)
	if err != nil {
		return nil, fmt.Errorf("insight: payload: %w", err)
	}
	row, err := s.insert(ctx, p, from, to, raw, d.watermark, owner)
	if err != nil {
		return nil, err
	}
	if s.Notify != nil {
		link := "/insights"
		if _, err := s.Notify.Send(ctx, notify.Spec{UserID: owner, CourseID: &p.CourseID, Type: TypeReady, Title: "Báo cáo lỗ hổng kiến thức đã sẵn sàng", Link: &link, DedupeKey: "insight.ready:" + row.ID.String()}); err != nil && s.Log != nil {
			s.Log.WarnContext(ctx, "insight: gửi thông báo", "err", err)
		}
	}
	stage(100, "DONE")
	return map[string]any{"report_id": row.ID}, nil
}

var selfIntro = regexp.MustCompile(`(?i)\b(tôi|em|mình|tớ)\s+(tên\s+là|tên|là)\s+\S+(\s+\S+){0,3}`)

// anonymize: che thông tin cá nhân theo danh sách lớp (privacy.Detect) rồi bỏ lời tự giới thiệu «em là …» — chữ đi vào embedding / LLM / payload luôn qua đây.
func (s *Service) anonymize(ctx context.Context, courseID uuid.UUID, text string) string {
	text = strings.TrimSpace(text)
	if s.Det != nil {
		if fs, err := s.Det.Detect(ctx, courseID, text); err == nil {
			text = privacy.Redact(text, fs)
		} else if s.Log != nil {
			s.Log.WarnContext(ctx, "insight: che chữ lỗi, bỏ câu", "err", err)
			return ""
		}
	}
	return strings.TrimSpace(selfIntro.ReplaceAllString(text, ""))
}

type item struct {
	student uuid.UUID
	text    string
	low, nh bool // tự tin thấp / escalate; Not helpful
	noCtx   bool
}

type collected struct {
	items     []item
	practice  []store.InsightPracticeTopicsRow
	watermark time.Time
}

func (s *Service) collect(ctx context.Context, q *store.Queries, courseID uuid.UUID, from, to time.Time) (collected, error) {
	var d collected
	chat, err := q.InsightChatQuestions(ctx, store.InsightChatQuestionsParams{CourseID: courseID, FromTs: from, ToTs: to, MaxRows: MaxRows})
	if err != nil {
		return d, fmt.Errorf("insight: câu hỏi chat: %w", err)
	}
	for _, r := range chat {
		if t := s.anonymize(ctx, courseID, r.Content); t != "" {
			d.items = append(d.items, item{student: r.UserID, text: t, low: r.LowConfidence || r.Escalated, nh: r.NotHelpful, noCtx: r.NoContext})
		}
		if r.CreatedAt.After(d.watermark) {
			d.watermark = r.CreatedAt
		}
	}
	th, err := q.InsightThreadQuestions(ctx, store.InsightThreadQuestionsParams{CourseID: courseID, FromTs: from, ToTs: to, MaxRows: MaxRows})
	if err != nil {
		return d, fmt.Errorf("insight: thread: %w", err)
	}
	for _, r := range th {
		if t := s.anonymize(ctx, courseID, r.Title+". "+r.Body); t != "" {
			d.items = append(d.items, item{student: r.UserID, text: t})
		}
		if r.CreatedAt.After(d.watermark) {
			d.watermark = r.CreatedAt
		}
	}
	if d.practice, err = q.InsightPracticeTopics(ctx, store.InsightPracticeTopicsParams{CourseID: courseID, FromTs: from, ToTs: to}); err != nil {
		return d, fmt.Errorf("insight: luyện đề: %w", err)
	}
	if wm, err := q.InsightWatermark(ctx, store.InsightWatermarkParams{CourseID: courseID, FromTs: from, ToTs: to}); err == nil && wm.After(d.watermark) {
		d.watermark = wm
	}
	return d, nil
}

// build: nhúng → gom cụm → tín hiệu → xếp hạng (code thuần) → câu mẫu k-ẩn danh + chương / tuần. Chưa có tên (name() điền sau).
func (s *Service) build(ctx context.Context, q *store.Queries, courseID uuid.UUID, days int, d collected) (Payload, error) {
	texts := make([]string, len(d.items))
	for i, it := range d.items {
		texts[i] = it.text
	}
	vecs, err := s.embed(ctx, texts)
	if err != nil {
		return Payload{}, err
	}
	// Câu ngoài tài liệu đi vào mục riêng; vẫn gom cùng để chủ đề lớn có tín hiệu.
	clusters := Cluster(vecs, s.tau())
	pt, ptVec, err := s.practiceVecs(ctx, d.practice)
	if err != nil {
		return Payload{}, err
	}
	pres := Present{Practice: len(d.practice) > 0}
	type cl struct {
		idx  []int
		sig  Signals
		keys []string
	}
	cls := make([]cl, len(clusters))
	for k, idx := range clusters {
		students := map[uuid.UUID]bool{}
		var low, nh int
		var tx []string
		for _, i := range idx {
			it := d.items[i]
			students[it.student] = true
			if it.low {
				low++
				pres.Low = true
			}
			if it.nh {
				nh++
				pres.NotHelpful = true
			}
			tx = append(tx, it.text)
		}
		n := float64(len(idx))
		cls[k] = cl{idx: idx, keys: Keywords(tx, 3), sig: Signals{Students: len(students), Questions: len(idx), LowRate: float64(low) / n, NotHelpfulRate: float64(nh) / n}}
		if len(pt) > 0 {
			if t, sim := nearest(Centroid(vecs, idx), ptVec); sim >= practiceTau {
				cls[k].sig.PracticeWrongRate = float64(pt[t].Wrong) / float64(pt[t].Answered)
			}
		}
	}
	sigs := make([]Signals, len(cls))
	for i := range cls {
		sigs[i] = cls[i].sig
	}
	scores := Score(sigs, pres)
	pl := Payload{PeriodDays: days, Counts: Counts{Questions: len(d.items), Clusters: len(cls)}, Topics: []Topic{}, Uncovered: uncovered(d.items)}
	all := map[uuid.UUID]bool{}
	for _, it := range d.items {
		all[it.student] = true
	}
	pl.Counts.Students = len(all)
	for rank, k := range Order(sigs, scores) {
		c := cls[k]
		t := Topic{Rank: rank + 1, Score: round3(scores[k]), Keywords: []string{}, Signals: TopicSignals{Students: c.sig.Students, Questions: c.sig.Questions, LowRate: round3(c.sig.LowRate), NotHelpfulRate: round3(c.sig.NotHelpfulRate)}, Examples: []string{}, NamedBy: namedByFallback}
		if pres.Practice {
			p := round3(c.sig.PracticeWrongRate)
			t.Signals.PracticeWrongRate = &p
		}
		t.Why = why(t.Signals)
		if c.keys != nil {
			t.Keywords = c.keys
		}
		if c.sig.Students < MinStudents {
			t.FewAsked = true
		} else {
			t.Examples = pickExamples(d.items, c.idx)
		}
		if ch, err := q.InsightNearestChunk(ctx, store.InsightNearestChunkParams{CourseID: courseID, Centroid: pgvector.NewVector(Centroid(vecs, c.idx))}); err == nil {
			title := ch.Title
			t.Chapter = &title
			if ch.WeekNo != nil {
				w := int(*ch.WeekNo)
				t.Week = &w
			}
		}
		t.Name = tempName(c.keys, rank+1)
		t.samples = sampleTexts(d.items, c.idx)
		pl.Topics = append(pl.Topics, t)
	}
	return pl, nil
}

func (s *Service) embed(ctx context.Context, texts []string) ([][]float32, error) {
	lane := llm.LaneBatch
	var out [][]float32
	for i := 0; i < len(texts); i += 100 {
		v, err := s.LLM.Embed(ctx, llm.EmbedRequest{Inputs: texts[i:min(i+100, len(texts))], Lane: &lane})
		if err != nil {
			return nil, fmt.Errorf("insight: nhúng: %w", err)
		}
		out = append(out, v...)
	}
	return out, nil
}

func (s *Service) practiceVecs(ctx context.Context, rows []store.InsightPracticeTopicsRow) ([]store.InsightPracticeTopicsRow, [][]float32, error) {
	var keep []store.InsightPracticeTopicsRow
	var names []string
	for _, r := range rows {
		if r.Topic != "" {
			keep, names = append(keep, r), append(names, r.Topic)
		}
	}
	if len(keep) == 0 {
		return nil, nil, nil
	}
	v, err := s.embed(ctx, names)
	return keep, v, err
}

func nearest(c []float32, cands [][]float32) (int, float64) {
	best, bs := 0, -2.0
	for i, v := range cands {
		if s := cosine(c, v); s > bs {
			best, bs = i, s
		}
	}
	return best, bs
}

func round3(x float64) float64 { return float64(int(x*1000+0.5)) / 1000 }

func why(s TopicSignals) string {
	t := fmt.Sprintf("%d sinh viên hỏi, %d câu", s.Students, s.Questions)
	if s.LowRate > 0 {
		t += fmt.Sprintf("; %d%% trả lời tự tin thấp", int(s.LowRate*100+0.5))
	}
	if s.NotHelpfulRate > 0 {
		t += fmt.Sprintf("; %d%% bị đánh giá chưa hữu ích", int(s.NotHelpfulRate*100+0.5))
	}
	if s.PracticeWrongRate != nil && *s.PracticeWrongRate > 0 {
		t += fmt.Sprintf("; %d%% sai khi luyện đề", int(*s.PracticeWrongRate*100+0.5))
	}
	return t
}

func tempName(keys []string, rank int) string {
	if len(keys) == 0 {
		return fmt.Sprintf("Nhóm câu hỏi %d", rank)
	}
	return strings.Join(keys, ", ")
}

// pickExamples: 3–5 câu mẫu, mỗi câu từ một sinh viên khác nhau khi có thể (vòng 1 mỗi người một câu, thiếu thì lấy thêm).
func pickExamples(items []item, idx []int) []string {
	seen := map[uuid.UUID]bool{}
	var out, rest []string
	for _, i := range idx {
		t := trim(items[i].text)
		if seen[items[i].student] {
			rest = append(rest, t)
			continue
		}
		seen[items[i].student] = true
		out = append(out, t)
	}
	if len(out) > MaxExamples {
		out = out[:MaxExamples]
	}
	for _, t := range rest {
		if len(out) >= 3 {
			break
		}
		out = append(out, t)
	}
	return out
}

func sampleTexts(items []item, idx []int) []string {
	var out []string
	for _, i := range idx {
		if len(out) == MaxExamples {
			break
		}
		out = append(out, trim(items[i].text))
	}
	return out
}

func trim(t string) string {
	if r := []rune(t); len(r) > 200 {
		return string(r[:200]) + "…"
	}
	return t
}

// uncovered: câu AI báo «không có trong tài liệu», gộp câu trùng (bỏ dấu, hoa thường), nhiều lần nhất trước, tối đa 20.
// k-ẩn danh như câu mẫu (FR-13): chỉ liệt kê câu do ≥ MinStudents sinh viên KHÁC NHAU hỏi — một người hỏi nhiều lần vẫn không hiện (P-11-31 F-1);
// nếu không, giảng viên / trợ giảng đọc nguyên văn chat riêng của một sinh viên khi chưa escalate. Count vẫn là tổng số lần hỏi.
func uncovered(items []item) []Uncovered {
	type agg struct {
		u        Uncovered
		students map[uuid.UUID]struct{}
	}
	cnt := map[string]*agg{}
	var order []string
	for _, it := range items {
		if !it.noCtx {
			continue
		}
		k := strings.ToLower(strings.Join(strings.Fields(it.text), " "))
		if a, ok := cnt[k]; ok {
			a.u.Count++
			a.students[it.student] = struct{}{}
			continue
		}
		cnt[k] = &agg{u: Uncovered{Text: trim(it.text), Count: 1}, students: map[uuid.UUID]struct{}{it.student: {}}}
		order = append(order, k)
	}
	out := make([]Uncovered, 0, len(order))
	for _, k := range order {
		if len(cnt[k].students) >= MinStudents {
			out = append(out, cnt[k].u)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if len(out) > 20 {
		out = out[:20]
	}
	return out
}

// insert ghi báo cáo; hai việc đồng thời của cùng lớp + kỳ không tạo hai bản trùng data_watermark (khoá tư vấn theo lớp, kiểm lại trong khoá).
func (s *Service) insert(ctx context.Context, p payload, from, to time.Time, raw []byte, wm time.Time, owner uuid.UUID) (store.InsightReportInsertRow, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return store.InsightReportInsertRow{}, fmt.Errorf("insight: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended('insight:' || $1::text, 0))`, p.CourseID); err != nil {
		return store.InsightReportInsertRow{}, fmt.Errorf("insight: khoá lớp: %w", err)
	}
	q := store.New(tx)
	if last, err := q.InsightReportLatestFor(ctx, store.InsightReportLatestForParams{CourseID: p.CourseID, PeriodDays: int32(p.PeriodDays)}); err == nil && !wm.After(last.DataWatermark) { //nolint:gosec // 7..120
		return store.InsightReportInsertRow{ID: last.ID}, nil // việc song song đã ghi cùng dữ liệu
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return store.InsightReportInsertRow{}, fmt.Errorf("insight: báo cáo gần nhất: %w", err)
	}
	row, err := q.InsightReportInsert(ctx, store.InsightReportInsertParams{CourseID: p.CourseID, PeriodFrom: dateIn(from), PeriodTo: dateIn(to), Payload: raw, DataWatermark: wm, GeneratedBy: owner})
	if err != nil {
		return store.InsightReportInsertRow{}, fmt.Errorf("insight: ghi báo cáo: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return store.InsightReportInsertRow{}, fmt.Errorf("insight: chốt báo cáo: %w", err)
	}
	return row, nil
}
