package grade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Service là nghiệp vụ công thức điểm / sổ điểm. Không giữ trạng thái giữa các yêu cầu; điểm KHÔNG cache phía máy chủ (luật 15).
type Service struct {
	Pool   *pgxpool.Pool
	Clock  clock.Clock
	Log    *slog.Logger
	Notify *notify.Service  // chuông GRADE_*; nil = chỉ job nhắc không chạy được
	Redis  *appredis.Client // vé tải lên của nhập XLSX (`ep:upload:{id}`)
	Blob   BlobReader       // đọc lại tệp nhập từ object storage
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now()
}

// Trạng thái công thức của lớp (SRS 4.5.1).
const (
	StateNoPolicy     = "NO_POLICY"
	StateNotExtracted = "NOT_EXTRACTED"
	StateDraftUnclear = "DRAFT_UNCLEAR"
	StateDraftReady   = "DRAFT_READY"
	StateConfirmed    = "CONFIRMED"

	// BlockedNoScheme là lý do khoá `Chốt điểm` khi chưa có công thức đã xác nhận.
	BlockedNoScheme = "Chưa có công thức điểm đã xác nhận"
)

// GradebookStatus là sổ điểm ở `…/grade-scheme/status`.
type GradebookStatus struct {
	Status          string `json:"status"`
	SnapshotVersion int    `json:"snapshot_version"`
}

// TopicSchemeConfirmed: công thức vừa được xác nhận (US-P6-04) → "Hôm nay" của Staff làm mới (mục GRADE_SCHEME_UNCONFIRMED biến mất ≤ 5 s). Payload {course_id}.
const TopicSchemeConfirmed = "grade.scheme_confirmed"

// SchemeStatus là thân `GET …/grade-scheme/status` (Staff).
type SchemeStatus struct {
	State                 string          `json:"state"`
	UnclearCount          int             `json:"unclear_count"`
	HasPolicy             bool            `json:"has_policy"`
	ConfirmedAt           *time.Time      `json:"confirmed_at,omitempty"`
	SchemeNo              *int            `json:"scheme_no,omitempty"`
	PendingDraft          bool            `json:"pending_draft"`
	Gradebook             GradebookStatus `json:"gradebook"`
	FinalizeBlockedReason string          `json:"finalize_blocked_reason,omitempty"`
	ClassCode             string          `json:"-"`
}

// Inspection là kết quả bộ kiểm cho một bản nháp: mục chưa rõ (chặn xác nhận) + cảnh báo không chặn.
type Inspection struct {
	Unclear  []Unclear
	Warnings []string
}

type storedCite struct {
	Page  int    `json:"page"`
	Quote string `json:"quote"`
}

// draftOf dựng Draft thuần từ các dòng DB của MỘT công thức.
func draftOf(comps []store.GradeComponentsOfRow, rules []store.GradeRulesOfRow) Draft {
	var d Draft
	for _, c := range comps {
		var cs []storedCite
		_ = json.Unmarshal(c.Citations, &cs)
		dc := DraftComponent{Key: c.Key, Name: c.Name, Group: Group(c.Grp), Weight: c.Weight.String(), FilledByUser: c.FilledByUser}
		for _, x := range cs {
			dc.Citations = append(dc.Citations, Citation(x))
		}
		d.Components = append(d.Components, dc)
	}
	for _, r := range rules {
		dr := DraftRule{Kind: RuleKind(r.Kind), Applicable: r.Applicable, Params: r.Params, FilledByUser: r.FilledByUser}
		if r.SourceQuote != nil {
			dr.Quote = *r.SourceQuote
		}
		if r.SourcePage != nil {
			dr.Page = int(*r.SourcePage)
		}
		d.Rules = append(d.Rules, dr)
	}
	return d
}

type draftRef struct {
	ID, CourseID uuid.UUID
}

// InspectDrafts chạy bộ kiểm 4.2.5 cho một lô bản nháp: đọc thành phần / luật theo lô, tra trích dẫn bằng SQL (theo tài liệu nguồn còn tồn tại),
// rồi thêm mục `unclear_items` do LLM đưa vào mà còn hiệu lực. Mục CODE luôn tính lại từ dữ liệu hiện tại (tự hết khi dữ liệu thoả).
func InspectDrafts(ctx context.Context, q *store.Queries, refs []draftRef) (map[uuid.UUID]Inspection, error) {
	out := make(map[uuid.UUID]Inspection, len(refs))
	if len(refs) == 0 {
		return out, nil
	}
	ids := make([]string, len(refs))
	for i, r := range refs {
		ids[i] = r.ID.String()
	}
	schemes, err := q.GradeSchemesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("grade: đọc công thức: %w", err)
	}
	comps, err := q.GradeComponentsOf(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("grade: đọc thành phần: %w", err)
	}
	rules, err := q.GradeRulesOf(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("grade: đọc luật: %w", err)
	}
	cBy, rBy := map[uuid.UUID][]store.GradeComponentsOfRow{}, map[uuid.UUID][]store.GradeRulesOfRow{}
	for _, c := range comps {
		cBy[c.SchemeID] = append(cBy[c.SchemeID], c)
	}
	for _, r := range rules {
		rBy[r.SchemeID] = append(rBy[r.SchemeID], r)
	}
	for _, sc := range schemes {
		d := draftOf(cBy[sc.ID], rBy[sc.ID])
		cited, err := citationVerifier(ctx, q, sc.CourseID, sc.SourceDocumentID, d)
		if err != nil {
			return nil, err
		}
		un, warn := Check(d, cited)
		un = append(un, activeLLM(sc.Extraction, int(sc.Version), cBy[sc.ID], rBy[sc.ID])...)
		out[sc.ID] = Inspection{Unclear: un, Warnings: warn}
	}
	return out, nil
}

// citationVerifier tra SQL một lần mọi trích dẫn của bản nháp trên đúng tài liệu nguồn. Không còn tài liệu nguồn (đã gỡ) → không kiểm (nil).
func citationVerifier(ctx context.Context, q *store.Queries, courseID uuid.UUID, doc *uuid.UUID, d Draft) (func(Citation) bool, error) {
	if doc == nil {
		return nil, nil
	}
	pages, _, _, err := DocPages(ctx, q, courseID, *doc, 0)
	if err != nil {
		return nil, err
	}
	textOf := map[int]string{}
	for _, p := range pages {
		textOf[p.No] = p.Text
	}
	return VerifyCitations(ctx, q, textOf, d)
}

// VerifyCitations tra SQL một lần mọi trích dẫn của bản nháp trên văn bản các trang (`textOf[trang]`). Trang không có trong `textOf` → không khớp.
func VerifyCitations(ctx context.Context, q *store.Queries, textOf map[int]string, d Draft) (func(Citation) bool, error) {
	type key struct {
		page  int
		quote string
	}
	idx := map[key]int{}
	type item struct {
		Quote string `json:"quote"`
		Text  string `json:"text"`
	}
	var list []item
	add := func(c Citation) {
		k := key{c.Page, c.Quote}
		text, ok := textOf[c.Page]
		if _, seen := idx[k]; !seen && ok && c.Quote != "" {
			idx[k] = len(list)
			list = append(list, item{Quote: c.Quote, Text: text})
		}
	}
	for _, c := range d.Components {
		for _, ct := range c.Citations {
			add(ct)
		}
	}
	for _, r := range d.Rules {
		add(Citation{Page: r.Page, Quote: r.Quote})
	}
	found := map[key]bool{}
	if len(list) > 0 {
		raw, _ := json.Marshal(list)
		rows, err := q.GradeCitationsCheck(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("grade: kiểm trích dẫn: %w", err)
		}
		for k, i := range idx {
			for _, r := range rows {
				if int(r.Idx) == i {
					found[k] = r.Found
				}
			}
		}
	}
	return func(c Citation) bool { return found[key{c.Page, c.Quote}] }, nil
}

// activeLLM trả các mục chưa rõ do LLM đưa vào mà còn hiệu lực: bản nháp chưa ai sửa (version 1), hoặc mục đích danh còn tồn tại và chưa được giảng viên tự điền.
func activeLLM(extraction json.RawMessage, version int, comps []store.GradeComponentsOfRow, rules []store.GradeRulesOfRow) []Unclear {
	var x struct {
		Items []Unclear `json:"unclear_items"`
	}
	if json.Unmarshal(extraction, &x) != nil {
		return nil
	}
	filled := map[string]bool{}
	for _, c := range comps {
		filled["components."+c.Key] = c.FilledByUser
	}
	for _, r := range rules {
		filled["rules."+string(r.Kind)] = r.FilledByUser
	}
	var out []Unclear
	for _, u := range x.Items {
		if u.Source != SourceLLM {
			continue
		}
		if f, known := filled[u.Target]; version == 1 || (known && !f) {
			out = append(out, u)
		}
	}
	return out
}

// StatusBatch trả trạng thái công thức của một lô lớp (SRS 4.5.1): MỘT truy vấn dữ kiện (số mục chưa rõ của bản nháp đọc từ cột `unclear_count`, không chạy bộ kiểm khi đọc: hằng số truy vấn, không đọc toàn văn quy chế).
// Mọi lớp trong `courseIDs` có mặt trong kết quả (lớp không tồn tại bị bỏ).
func StatusBatch(ctx context.Context, q *store.Queries, courseIDs []uuid.UUID) (map[uuid.UUID]SchemeStatus, error) {
	out := make(map[uuid.UUID]SchemeStatus, len(courseIDs))
	if len(courseIDs) == 0 {
		return out, nil
	}
	ids := make([]string, len(courseIDs))
	for i, c := range courseIDs {
		ids[i] = c.String()
	}
	facts, err := q.GradeStatusFacts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("grade: dữ kiện công thức: %w", err)
	}
	for _, f := range facts {
		st := SchemeStatus{HasPolicy: f.HasPolicy, PendingDraft: f.DraftID != nil, ClassCode: f.ClassCode, Gradebook: GradebookStatus{Status: f.GbStatus, SnapshotVersion: int(f.GbVersion)}}
		switch {
		case f.ConfirmedID != nil:
			st.State = StateConfirmed
			st.ConfirmedAt = f.ConfirmedAt
			if f.ConfirmedNo != nil {
				n := int(*f.ConfirmedNo)
				st.SchemeNo = &n
			}
		case f.DraftID != nil:
			st.UnclearCount = int(f.UnclearCount) // ghi lúc lưu bản nháp / sửa / đổi tài liệu (#16), không tính lại khi đọc
			st.State = StateDraftReady
			if st.UnclearCount > 0 {
				st.State = StateDraftUnclear
			}
		case f.HasPolicy:
			st.State = StateNotExtracted
		default:
			st.State = StateNoPolicy
		}
		if st.State != StateConfirmed {
			st.FinalizeBlockedReason = BlockedNoScheme
		}
		out[f.CourseID] = st
	}
	return out, nil
}

// Status trả trạng thái công thức của MỘT lớp. Đọc mỗi lần, không cache.
func (s *Service) Status(ctx context.Context, courseID uuid.UUID) (SchemeStatus, error) {
	m, err := StatusBatch(ctx, store.New(s.Pool), []uuid.UUID{courseID})
	if err != nil {
		return SchemeStatus{}, err
	}
	st, ok := m[courseID]
	if !ok {
		return SchemeStatus{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	return st, nil
}

// RecomputeUnclear chạy bộ kiểm cho bản nháp hiện có của lớp và ghi `unclear_count` (SRS 4.5.3, #16). Gọi khi tài liệu của lớp đổi / bị gỡ
// (trích dẫn có thể thôi khớp hoặc không còn kiểm được). Bản nháp đổi trong lúc tính thì bỏ qua: người sửa đã ghi số mới.
func (s *Service) RecomputeUnclear(ctx context.Context, courseID uuid.UUID) error {
	q := store.New(s.Pool)
	id, err := q.GradeSchemeIDByStatus(ctx, store.GradeSchemeIDByStatusParams{CourseID: courseID, Status: store.GradeSchemeStatusDRAFT})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("grade: bản nháp: %w", err)
	}
	sc, err := q.GradeSchemeGet(ctx, store.GradeSchemeGetParams{CourseID: courseID, ID: id})
	if err != nil {
		return fmt.Errorf("grade: bản nháp: %w", err)
	}
	insp, err := InspectDrafts(ctx, q, []draftRef{{ID: id, CourseID: courseID}})
	if err != nil {
		return err
	}
	if _, err := q.GradeSetUnclear(ctx, store.GradeSetUnclearParams{CourseID: courseID, ID: id, Version: sc.Version, UnclearCount: int32(len(insp[id].Unclear))}); err != nil {
		return fmt.Errorf("grade: ghi số mục chưa rõ: %w", err)
	}
	return nil
}

// HandleDocumentChanged là outbox.Handler của `document.changed`: tính lại số mục chưa rõ của bản nháp lớp đó. Idempotent.
func (s *Service) HandleDocumentChanged(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID string `json:"course_id"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil {
		return fmt.Errorf("grade: payload %s: %w", m.Topic, err)
	}
	c, err := uuid.Parse(p.CourseID)
	if err != nil {
		return nil
	}
	return s.RecomputeUnclear(ctx, c)
}
