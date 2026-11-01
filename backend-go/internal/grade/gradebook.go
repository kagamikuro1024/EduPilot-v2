package grade

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Dạng JSON của bảng điểm (SRS 4.6.1). Số điểm là chuỗi decimal (dấu chấm); giao diện đổi sang dấu phẩy.
type (
	ColumnView struct {
		ItemID       uuid.UUID `json:"item_id"`
		Key          string    `json:"key"`
		Name         string    `json:"name"`
		ComponentKey string    `json:"component_key"`
		Grp          string    `json:"grp"`
		MaxScore     string    `json:"max_score"`
		Source       string    `json:"source"`
		Published    bool      `json:"published"`
	}
	EntryView struct {
		Score     string `json:"score"`
		Version   int    `json:"version"`
		Source    string `json:"source"`
		UpdatedBy string `json:"updated_by,omitempty"`
	}
	ResultView struct {
		Status  string   `json:"status"`
		QT      string   `json:"qt,omitempty"`
		HP      string   `json:"hp,omitempty"`
		Letter  string   `json:"letter,omitempty"`
		Verdict string   `json:"verdict"`
		Missing []string `json:"missing"`
	}
	RowView struct {
		Student StudentView          `json:"student"`
		Entries map[string]EntryView `json:"entries"`
		Bonus   string               `json:"bonus"`
		Penalty string               `json:"penalty"`
		Result  *ResultView          `json:"result"`
		Locked  *LockedView          `json:"locked,omitempty"`
		// Drift: sổ LOCKED mà điểm sống (Compute) khác bản chốt — chỉ Staff thấy (nhãn "Khác bản chốt").
		Drift bool `json:"drift,omitempty"`
	}
	StudentView struct {
		ID          uuid.UUID `json:"id"`
		FullName    string    `json:"full_name"`
		StudentCode string    `json:"student_code"`
	}
	LockedView struct {
		Version int `json:"version"`
	}
	WarnStudent struct {
		ID       uuid.UUID `json:"id"`
		FullName string    `json:"full_name"`
	}
	GradebookPage struct {
		Scheme *struct {
			ID       uuid.UUID `json:"id"`
			SchemeNo int       `json:"scheme_no"`
		} `json:"scheme"`
		State           GradebookStatus          `json:"state"`
		Columns         []ColumnView             `json:"columns"`
		Rows            []RowView                `json:"rows"`
		NextCursor      *string                  `json:"next_cursor"`
		Warnings        []string                 `json:"warnings,omitempty"`
		WarningStudents map[string][]WarnStudent `json:"warning_students,omitempty"`
	}
)

type pageCursor struct {
	Fold string `json:"f"`
	ID   string `json:"i"`
}

func encodePage(fold string, id uuid.UUID) string {
	raw, _ := json.Marshal(pageCursor{Fold: fold, ID: id.String()})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodePage(s string) (pageCursor, error) {
	var c pageCursor
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || json.Unmarshal(raw, &c) != nil {
		return c, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidCursor)
	}
	if _, err := uuid.Parse(c.ID); err != nil {
		return c, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidCursor)
	}
	return c, nil
}

func resultView(r Result) *ResultView {
	v := &ResultView{Status: string(r.Status), Verdict: r.Verdict, Missing: append([]string{}, r.Missing...)}
	if r.HasQT {
		v.QT = r.QT.String()
	}
	if r.Status == StatusComplete {
		v.HP, v.Letter = r.HP.String(), r.Letter
	}
	return v
}

// Gradebook trả một trang bảng điểm (SRS 4.6.1). Mỗi trang: một truy vấn sinh viên + đúng một lô cho điểm / sự kiện / điểm danh (không N+1).
func (s *Service) Gradebook(ctx context.Context, courseID uuid.UUID, cursor string, limit int, first bool) (GradebookPage, error) {
	q := store.New(s.Pool)
	out := GradebookPage{Columns: []ColumnView{}, Rows: []RowView{}}
	gb, err := q.GradebookState(ctx, courseID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		out.State = GradebookStatus{Status: "OPEN"}
	case err != nil:
		return out, fmt.Errorf("grade: trạng thái sổ điểm: %w", err)
	default:
		out.State = GradebookStatus{Status: string(gb.Status), SnapshotVersion: int(gb.SnapshotVersion)}
	}
	var cur pageCursor
	if cursor != "" {
		if cur, err = decodePage(cursor); err != nil {
			return out, err
		}
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}
	studs, err := q.GradebookStudentsPage(ctx, store.GradebookStudentsPageParams{CourseID: courseID, AfterFold: cur.Fold, AfterID: cur.ID, PageLimit: int32(limit + 1)}) //nolint:gosec // ≤ 101
	if err != nil {
		return out, fmt.Errorf("grade: sinh viên của trang: %w", err)
	}
	if len(studs) > limit {
		studs = studs[:limit]
		last := studs[len(studs)-1]
		c := encodePage(last.FoldName, last.StudentID)
		out.NextCursor = &c
	}
	conf, ok, err := LoadConfirmed(ctx, q, courseID)
	if err != nil {
		return out, err
	}
	ids := make([]uuid.UUID, len(studs))
	for i, st := range studs {
		ids[i] = st.StudentID
	}
	var inputs map[uuid.UUID]Input
	var detail map[uuid.UUID]map[uuid.UUID]EntryView
	snaps := map[uuid.UUID]store.GradeSnapshotsForStudentsRow{}
	if ok {
		out.Scheme = &struct {
			ID       uuid.UUID `json:"id"`
			SchemeNo int       `json:"scheme_no"`
		}{conf.ID, conf.No}
		for _, c := range conf.Scheme.Components {
			for _, it := range c.Items {
				out.Columns = append(out.Columns, ColumnView{ItemID: it.ID, Key: it.Key, Name: it.Name, ComponentKey: c.Key, Grp: string(c.Group), MaxScore: it.Max.String(), Source: string(it.Source), Published: it.Source == SourceExam && it.Closed || it.Source == SourceManual && it.Published})
			}
		}
		if inputs, err = LoadInputs(ctx, q, courseID, conf.ID, ids); err != nil {
			return out, err
		}
		rows, err := q.GradeEntriesDetail(ctx, store.GradeEntriesDetailParams{CourseID: courseID, SchemeID: conf.ID, StudentIds: idStrings(ids)})
		if err != nil {
			return out, fmt.Errorf("grade: điểm đã nhập: %w", err)
		}
		if out.State.Status == "LOCKED" {
			sn, err := q.GradeSnapshotsForStudents(ctx, store.GradeSnapshotsForStudentsParams{CourseID: courseID, Version: int32(out.State.SnapshotVersion), StudentIds: idStrings(ids)}) //nolint:gosec // phiên bản nhỏ
			if err != nil {
				return out, fmt.Errorf("grade: bản chốt của trang: %w", err)
			}
			for _, r := range sn {
				snaps[r.StudentID] = r
			}
		}
		detail = map[uuid.UUID]map[uuid.UUID]EntryView{}
		for _, r := range rows {
			if detail[r.StudentID] == nil {
				detail[r.StudentID] = map[uuid.UUID]EntryView{}
			}
			detail[r.StudentID][r.ItemID] = EntryView{Score: r.Score.String(), Version: int(r.Version), Source: string(r.Source), UpdatedBy: r.UpdatedBy}
		}
	}
	for _, st := range studs {
		row := RowView{Student: StudentView{ID: st.StudentID, FullName: st.FullName, StudentCode: derefStr(st.StudentCode)}, Entries: map[string]EntryView{}, Bonus: "0", Penalty: "0"}
		if ok {
			in := inputs[st.StudentID]
			for item, e := range detail[st.StudentID] {
				row.Entries[item.String()] = e
			}
			for _, c := range conf.Scheme.Components { // ô item ASSIGNMENT hiện điểm đã công bố (đọc thẳng, không sửa được — chỉ để xem)
				for _, it := range c.Items {
					if v, has := in.Scores[it.ID]; has && it.Source == SourceAssignment {
						row.Entries[it.ID.String()] = EntryView{Score: v.String(), Source: string(SourceAssignment)}
					}
				}
			}
			r := Compute(conf.Scheme, in)
			row.Result = resultView(r)
			row.Bonus, row.Penalty = r.BonusApplied.String(), r.Penalty.String()
			if sn, found := snaps[st.StudentID]; found { // sổ LOCKED: hiển thị bản chốt, so điểm sống để gắn cờ drift
				row.Result, row.Bonus, row.Penalty, row.Drift = snapshotResult(sn, r)
				row.Locked = &LockedView{Version: out.State.SnapshotVersion}
			} else if out.State.Status == "LOCKED" {
				row.Locked = &LockedView{Version: out.State.SnapshotVersion}
			}
		}
		out.Rows = append(out.Rows, row)
	}
	if first && ok {
		out.Warnings, out.WarningStudents, err = s.warnings(ctx, q, courseID, conf)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// warnings: luật điểm cộng / trừ `applicable = false` mà lớp vẫn có sự kiện khác 0 (TLR-17); kèm danh sách sinh viên liên quan (≤ 50 mỗi loại).
func (s *Service) warnings(ctx context.Context, q *store.Queries, courseID uuid.UUID, conf Loaded) ([]string, map[string][]WarnStudent, error) {
	pen, bon := conf.Scheme.Rules.Penalty, conf.Scheme.Rules.Bonus
	notPen := pen != nil && !pen.Applicable
	notBon := bon != nil && !bon.Applicable
	if !notPen && !notBon {
		return nil, nil, nil
	}
	rows, err := q.GradeEventWarnings(ctx, courseID)
	if err != nil {
		return nil, nil, fmt.Errorf("grade: sự kiện điểm cộng / trừ: %w", err)
	}
	by := map[string][]WarnStudent{}
	for _, r := range rows {
		code := WarnBonusNotInScheme
		if r.IsPenalty {
			code = WarnPenaltyNotInScheme
		}
		if (code == WarnPenaltyNotInScheme && !notPen) || (code == WarnBonusNotInScheme && !notBon) || len(by[code]) >= 50 {
			continue
		}
		by[code] = append(by[code], WarnStudent{ID: r.StudentID, FullName: r.FullName})
	}
	var codes []string
	for _, c := range []string{WarnPenaltyNotInScheme, WarnBonusNotInScheme} {
		if len(by[c]) > 0 {
			codes = append(codes, c)
		}
	}
	return codes, by, nil
}

// ---- sửa ô ------------------------------------------------------------------------------------------------------------------------------------

// PutEntryInput là thân `PUT …/grade-entries`. Score nil = xoá ô; Version nil = ô mới.
type PutEntryInput struct {
	ItemID    uuid.UUID `json:"item_id"`
	StudentID uuid.UUID `json:"student_id"`
	Score     *string   `json:"score"`
	Version   *int      `json:"version"`
}

// EntryOut là điểm sau khi lưu (nil khi ô bị xoá).
type EntryOut struct {
	ItemID    uuid.UUID `json:"item_id"`
	StudentID uuid.UUID `json:"student_id"`
	Score     string    `json:"score"`
	Version   int       `json:"version"`
	Source    string    `json:"source"`
}

// PutEntryOut: điểm đã lưu + kết quả `Compute` của hàng đó (xem trước tức thì).
type PutEntryOut struct {
	Entry     *EntryOut   `json:"entry"`
	RowResult *ResultView `json:"row_result"`
	Bonus     string      `json:"bonus"`
	Penalty   string      `json:"penalty"`
}

var (
	scoreRE     = regexp.MustCompile(`^\d{1,3}([.,]\d{1,2})?$`) //nolint:gochecknoglobals // biểu thức chính quy bất biến
	scoreManyRE = regexp.MustCompile(`^\d{1,3}[.,]\d{3,}$`)     //nolint:gochecknoglobals // biểu thức chính quy bất biến
)

// ParseScore đọc điểm giao diện nhập (`8,5` / `8.5`): ≤ 2 chữ số thập phân, KHÔNG tự làm tròn (TLR-7). Dùng chung với nhập XLSX.
func ParseScore(raw string) (decimal.Decimal, error) {
	t := strings.TrimSpace(raw)
	if scoreManyRE.MatchString(t) {
		return decimal.Zero, apierr.New(http.StatusUnprocessableEntity, apierr.ScorePrecision)
	}
	if !scoreRE.MatchString(t) {
		return decimal.Zero, apierr.Validation(apierr.FieldError{Field: "score", Code: "invalid", Message: "Điểm là một số, tối đa 2 chữ số thập phân."})
	}
	return decimal.RequireFromString(strings.Replace(t, ",", ".", 1)), nil
}

// PutEntry sửa / xoá một ô điểm (SRS 4.6.3). Một giao dịch: khoá sổ `FOR SHARE` (LOCKED → 409) → kiểm item / sinh viên / thang → khoá ô, so version →
// ghi → audit (lỗi audit → không ghi điểm) → `Compute` của hàng.
func (s *Service) PutEntry(ctx context.Context, actor, courseID uuid.UUID, in PutEntryInput) (PutEntryOut, error) {
	var score *decimal.Decimal
	if in.Score != nil && strings.TrimSpace(*in.Score) != "" {
		d, err := ParseScore(*in.Score)
		if err != nil {
			return PutEntryOut{}, err
		}
		score = &d
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return PutEntryOut{}, fmt.Errorf("grade: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := store.New(tx)

	gb, err := q.GradebookStatusShare(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return PutEntryOut{}, apierr.New(http.StatusConflict, apierr.SchemeNotConfirmed)
	}
	if err != nil {
		return PutEntryOut{}, fmt.Errorf("grade: trạng thái sổ điểm: %w", err)
	}
	if gb == store.GradebookStatusLOCKED {
		return PutEntryOut{}, apierr.New(http.StatusConflict, apierr.GradebookLocked)
	}
	item, err := q.GradeItemForEdit(ctx, store.GradeItemForEditParams{CourseID: courseID, ItemID: in.ItemID})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && item.SchemeStatus != store.GradeSchemeStatusCONFIRMED) {
		return PutEntryOut{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return PutEntryOut{}, fmt.Errorf("grade: đọc cột điểm: %w", err)
	}
	if item.Source != store.GradeItemSourceMANUAL {
		return PutEntryOut{}, apierr.New(http.StatusConflict, apierr.ItemNotEditable)
	}
	if score != nil && (score.Sign() < 0 || score.GreaterThan(item.MaxScore)) {
		return PutEntryOut{}, apierr.New(http.StatusUnprocessableEntity, apierr.ScoreOutOfRange)
	}
	if active, err := q.GradeStudentIsActive(ctx, store.GradeStudentIsActiveParams{CourseID: courseID, StudentID: in.StudentID}); err != nil {
		return PutEntryOut{}, fmt.Errorf("grade: sinh viên: %w", err)
	} else if !active {
		return PutEntryOut{}, apierr.Validation(apierr.FieldError{Field: "student_id", Code: "not_active", Message: "Sinh viên không còn trong lớp."})
	}

	cur, err := q.GradeEntryLock(ctx, store.GradeEntryLockParams{CourseID: courseID, ItemID: in.ItemID, StudentID: in.StudentID})
	exists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PutEntryOut{}, fmt.Errorf("grade: khoá ô điểm: %w", err)
	}
	if exists && (in.Version == nil || int(cur.Version) != *in.Version) || !exists && in.Version != nil {
		return PutEntryOut{}, s.entryConflict(ctx, q, courseID, in)
	}

	var before, after map[string]any
	if exists {
		before = map[string]any{"student_id": in.StudentID, "item_id": in.ItemID, "item_key": item.Key, "score": cur.Score.String(), "version": cur.Version}
	}
	var out PutEntryOut
	switch {
	case score == nil && !exists:
		// xoá ô chưa có: không đổi gì
	case score == nil:
		n, err := q.GradeEntryDelete(ctx, store.GradeEntryDeleteParams{CourseID: courseID, ID: cur.ID, Version: cur.Version})
		if err != nil || n != 1 {
			return PutEntryOut{}, s.entryConflict(ctx, q, courseID, in)
		}
		after = map[string]any{"student_id": in.StudentID, "item_id": in.ItemID, "item_key": item.Key, "score": nil}
	case exists:
		e, err := q.GradeEntryUpdate(ctx, store.GradeEntryUpdateParams{CourseID: courseID, ID: cur.ID, Version: cur.Version, Score: *score, EnteredBy: actor})
		if err != nil {
			return PutEntryOut{}, s.entryConflict(ctx, q, courseID, in)
		}
		out.Entry = &EntryOut{ItemID: in.ItemID, StudentID: in.StudentID, Score: e.Score.String(), Version: int(e.Version), Source: string(e.Source)}
		after = map[string]any{"student_id": in.StudentID, "item_id": in.ItemID, "item_key": item.Key, "score": e.Score.String(), "version": e.Version}
	default:
		e, err := q.GradeEntryInsert(ctx, store.GradeEntryInsertParams{CourseID: courseID, ItemID: in.ItemID, StudentID: in.StudentID, Score: *score, EnteredBy: actor})
		if err != nil {
			return PutEntryOut{}, fmt.Errorf("grade: ghi điểm: %w", err)
		}
		out.Entry = &EntryOut{ItemID: in.ItemID, StudentID: in.StudentID, Score: e.Score.String(), Version: int(e.Version), Source: string(e.Source)}
		after = map[string]any{"student_id": in.StudentID, "item_id": in.ItemID, "item_key": item.Key, "score": e.Score.String(), "version": e.Version}
	}
	if after != nil {
		b, _ := json.Marshal(before)
		a, _ := json.Marshal(after)
		if before == nil {
			b = nil
		}
		if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "grade_entries", EntityID: in.ItemID.String() + ":" + in.StudentID.String(), Action: "set", Before: b, After: a}); err != nil {
			return PutEntryOut{}, fmt.Errorf("grade: audit: %w", err) // fail-closed: không có audit thì không ghi điểm
		}
	}
	conf, ok, err := LoadConfirmed(ctx, q, courseID)
	if err != nil || !ok {
		return PutEntryOut{}, err
	}
	inputs, err := LoadInputs(ctx, q, courseID, conf.ID, []uuid.UUID{in.StudentID})
	if err != nil {
		return PutEntryOut{}, err
	}
	r := Compute(conf.Scheme, inputs[in.StudentID])
	out.RowResult, out.Bonus, out.Penalty = resultView(r), r.BonusApplied.String(), r.Penalty.String()
	if err := tx.Commit(ctx); err != nil {
		return PutEntryOut{}, fmt.Errorf("grade: lưu điểm: %w", err)
	}
	return out, nil
}

// entryConflict dựng 409 `VERSION_CONFLICT` với giá trị hiện tại của ô (kèm người sửa) để giao diện hỏi giữ bên nào.
func (s *Service) entryConflict(ctx context.Context, q *store.Queries, courseID uuid.UUID, in PutEntryInput) error {
	cur := map[string]any{"score": nil, "version": 0}
	rows, err := q.GradeEntryCurrent(ctx, store.GradeEntryCurrentParams{CourseID: courseID, ItemID: in.ItemID, StudentID: in.StudentID})
	if err == nil && len(rows) == 1 {
		r := rows[0]
		cur = map[string]any{"score": r.Score.String(), "version": r.Version, "updated_by": map[string]any{"name": r.UpdatedBy}, "updated_at": r.UpdatedAt}
	}
	return apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current_version": cur["version"], "current": cur})
}

// ---- công bố cột ------------------------------------------------------------------------------------------------------------------------------

// PublishItem đặt / gỡ `published_at` của cột nhập tay (idempotent; audit mỗi lần đổi). Cột bài thi theo bài thi: 409 `ITEM_NOT_PUBLISHABLE`.
func (s *Service) PublishItem(ctx context.Context, actor, courseID, itemID uuid.UUID, published bool) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("grade: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := store.New(tx)
	gb, err := q.GradebookStatusShare(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, apierr.New(http.StatusConflict, apierr.SchemeNotConfirmed)
	}
	if err != nil {
		return false, fmt.Errorf("grade: trạng thái sổ điểm: %w", err)
	}
	if gb == store.GradebookStatusLOCKED {
		return false, apierr.New(http.StatusConflict, apierr.GradebookLocked)
	}
	item, err := q.GradeItemForEdit(ctx, store.GradeItemForEditParams{CourseID: courseID, ItemID: itemID})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && item.SchemeStatus != store.GradeSchemeStatusCONFIRMED) {
		return false, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return false, fmt.Errorf("grade: đọc cột điểm: %w", err)
	}
	if item.Source != store.GradeItemSourceMANUAL {
		return false, apierr.New(http.StatusConflict, apierr.ItemNotPublishable)
	}
	was := item.PublishedAt != nil
	at, err := q.GradeItemSetPublished(ctx, store.GradeItemSetPublishedParams{CourseID: courseID, ItemID: itemID, Published: published})
	if err != nil {
		return false, fmt.Errorf("grade: công bố cột: %w", err)
	}
	if was != published {
		b, _ := json.Marshal(map[string]any{"item_key": item.Key, "published": was})
		a, _ := json.Marshal(map[string]any{"item_key": item.Key, "published": published, "published_at": at})
		if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "grade_items", EntityID: itemID.String(), Action: "publish", Before: b, After: a}); err != nil {
			return false, fmt.Errorf("grade: audit: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("grade: lưu công bố: %w", err)
	}
	return published, nil
}

// ---- giải trình, lịch sử, xem trước ---------------------------------------------------------------------------------------------------------

type (
	LineView struct {
		Label        string `json:"label"`
		Expr         string `json:"expr"`
		Value        string `json:"value"`
		Quote        string `json:"quote,omitempty"`
		Page         int    `json:"page,omitempty"`
		FilledByUser bool   `json:"filled_by_user"`
	}
	ExplainOut struct {
		Student StudentView `json:"student"`
		Result  *ResultView `json:"result"`
		Lines   []LineView  `json:"lines"`
		Locked  *LockedView `json:"locked,omitempty"`
	}
)

// Explain trả giải trình của một sinh viên (bản sống) — các dòng do `Compute` sinh, không có công thức thứ hai.
func (s *Service) Explain(ctx context.Context, courseID, studentID uuid.UUID) (ExplainOut, error) {
	q := store.New(s.Pool)
	studs, err := ActiveStudents(ctx, q, courseID)
	if err != nil {
		return ExplainOut{}, err
	}
	var st *Student
	for i := range studs {
		if studs[i].ID == studentID {
			st = &studs[i]
		}
	}
	if st == nil {
		return ExplainOut{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	conf, ok, err := LoadConfirmed(ctx, q, courseID)
	if err != nil {
		return ExplainOut{}, err
	}
	if !ok {
		return ExplainOut{}, apierr.New(http.StatusConflict, apierr.SchemeNotConfirmed)
	}
	inputs, err := LoadInputs(ctx, q, courseID, conf.ID, []uuid.UUID{studentID})
	if err != nil {
		return ExplainOut{}, err
	}
	r := Compute(conf.Scheme, inputs[studentID])
	out := ExplainOut{Student: StudentView{ID: st.ID, FullName: st.FullName, StudentCode: st.Code}, Result: resultView(r), Lines: []LineView{}}
	for _, l := range r.Lines {
		lv := LineView{Label: l.Label, Expr: l.Expr, Value: l.Value, Page: l.Page, FilledByUser: l.FilledByUser}
		if l.Quote != nil {
			lv.Quote = *l.Quote
		}
		out.Lines = append(out.Lines, lv)
	}
	if gb, err := q.GradebookState(ctx, courseID); err == nil && gb.Status == store.GradebookStatusLOCKED {
		out.Locked = &LockedView{Version: int(gb.SnapshotVersion)}
	}
	return out, nil
}

// HistoryItem là một mục lịch sử điểm của sinh viên (từ audit_log).
type HistoryItem struct {
	At     time.Time `json:"at"`
	By     string    `json:"by"`
	Action string    `json:"action"`
	Item   string    `json:"item"`
	From   *string   `json:"from"`
	To     *string   `json:"to"`
}

// HistoryPage: ≤ 50 mục mới nhất, phân trang con trỏ (created_at, id).
type HistoryPage struct {
	Items      []HistoryItem `json:"items"`
	NextCursor *string       `json:"next_cursor"`
}

func scoreOf(raw []byte) (item string, score *string) {
	var m struct {
		Item      string  `json:"item_key"`
		Score     *string `json:"score"`
		Published *bool   `json:"published"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return "", nil
	}
	if m.Published != nil {
		v := map[bool]string{true: "Đã công bố", false: "Chưa công bố"}[*m.Published]
		return m.Item, &v
	}
	return m.Item, m.Score
}

// History trả lịch sử điểm của một sinh viên từ audit_log. `cursor` là chuỗi `httpx.EncodeCursor` (created_at + id).
func (s *Service) History(ctx context.Context, courseID, studentID uuid.UUID, at *time.Time, id *uuid.UUID, limit int) (HistoryPage, error) {
	if limit < 1 || limit > 50 {
		limit = 50
	}
	rows, err := store.New(s.Pool).GradeHistory(ctx, store.GradeHistoryParams{CourseID: &courseID, StudentID: studentID.String(), CursorAt: at, CursorID: id, PageLimit: int32(limit + 1)}) //nolint:gosec // ≤ 51
	if err != nil {
		return HistoryPage{}, fmt.Errorf("grade: lịch sử điểm: %w", err)
	}
	out := HistoryPage{Items: []HistoryItem{}}
	if len(rows) > limit {
		rows = rows[:limit]
		c := base64.RawURLEncoding.EncodeToString([]byte(rows[len(rows)-1].CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + rows[len(rows)-1].ID.String()))
		out.NextCursor = &c
	}
	for _, r := range rows {
		it := HistoryItem{At: r.CreatedAt, By: r.ActorName, Action: r.Action}
		it.Item, it.To = scoreOf(r.After)
		if _, from := scoreOf(r.Before); from != nil {
			it.From = from
		}
		if it.Item == "" {
			it.Item, _ = scoreOf(r.Before)
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// ParseHistoryCursor giải mã con trỏ của `History`.
func ParseHistoryCursor(s string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, uuid.Nil, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidCursor)
	}
	a, b, ok := strings.Cut(string(raw), "|")
	t, e1 := time.Parse(time.RFC3339Nano, a)
	id, e2 := uuid.Parse(b)
	if !ok || e1 != nil || e2 != nil {
		return time.Time{}, uuid.Nil, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidCursor)
	}
	return t, id, nil
}

type (
	PreviewItem struct {
		StudentID   uuid.UUID `json:"student_id"`
		StudentCode string    `json:"student_code"`
		FullName    string    `json:"full_name"`
		Status      string    `json:"status"`
		QT          string    `json:"qt,omitempty"`
		HPRaw       string    `json:"hp_raw,omitempty"`
		HP          string    `json:"hp,omitempty"`
		Letter      string    `json:"letter,omitempty"`
		Verdict     string    `json:"verdict"`
	}
	PreviewSummary struct {
		Students     int            `json:"students"`
		Complete     int            `json:"complete"`
		Incomplete   int            `json:"incomplete"`
		Liet         int            `json:"liet"`
		Distribution map[string]int `json:"distribution"`
	}
	PreviewPage struct {
		Summary    *PreviewSummary `json:"summary,omitempty"`
		Items      []PreviewItem   `json:"items"`
		NextCursor *string         `json:"next_cursor"`
	}
)

// Preview trả kết quả `Compute` cho lớp theo trang (nguồn của hộp thoại chốt và của kiểm tra seed). `summary` chỉ ở trang đầu và tính trên CẢ lớp.
func (s *Service) Preview(ctx context.Context, courseID uuid.UUID, cursor string, limit int) (PreviewPage, error) {
	q := store.New(s.Pool)
	conf, ok, err := LoadConfirmed(ctx, q, courseID)
	if err != nil {
		return PreviewPage{}, err
	}
	if !ok {
		return PreviewPage{}, apierr.New(http.StatusConflict, apierr.SchemeNotConfirmed)
	}
	var cur pageCursor
	if cursor != "" {
		if cur, err = decodePage(cursor); err != nil {
			return PreviewPage{}, err
		}
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}
	out := PreviewPage{Items: []PreviewItem{}}
	if cursor == "" {
		sum := PreviewSummary{Distribution: map[string]int{}}
		all, err := ActiveStudents(ctx, q, courseID)
		if err != nil {
			return out, err
		}
		for lo := 0; lo < len(all); lo += 500 {
			hi := min(lo+500, len(all))
			ids := make([]uuid.UUID, 0, hi-lo)
			for _, st := range all[lo:hi] {
				ids = append(ids, st.ID)
			}
			in, err := LoadInputs(ctx, q, courseID, conf.ID, ids)
			if err != nil {
				return out, err
			}
			for _, id := range ids {
				r := Compute(conf.Scheme, in[id])
				sum.Students++
				if r.Status == StatusComplete {
					sum.Complete++
					sum.Distribution[r.Letter]++
					if strings.HasPrefix(r.Verdict, "LIET") {
						sum.Liet++
					}
				} else {
					sum.Incomplete++
				}
			}
		}
		out.Summary = &sum
	}
	studs, err := q.GradebookStudentsPage(ctx, store.GradebookStudentsPageParams{CourseID: courseID, AfterFold: cur.Fold, AfterID: cur.ID, PageLimit: int32(limit + 1)}) //nolint:gosec // ≤ 101
	if err != nil {
		return out, fmt.Errorf("grade: sinh viên của trang: %w", err)
	}
	if len(studs) > limit {
		studs = studs[:limit]
		c := encodePage(studs[len(studs)-1].FoldName, studs[len(studs)-1].StudentID)
		out.NextCursor = &c
	}
	ids := make([]uuid.UUID, len(studs))
	for i, st := range studs {
		ids[i] = st.StudentID
	}
	in, err := LoadInputs(ctx, q, courseID, conf.ID, ids)
	if err != nil {
		return out, err
	}
	for _, st := range studs {
		r := Compute(conf.Scheme, in[st.StudentID])
		it := PreviewItem{StudentID: st.StudentID, StudentCode: derefStr(st.StudentCode), FullName: st.FullName, Status: string(r.Status), Verdict: r.Verdict}
		if r.HasQT {
			it.QT = r.QT.String()
		}
		if r.Status == StatusComplete {
			it.HPRaw, it.HP, it.Letter = r.HPRaw.String(), r.HP.String(), r.Letter
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}
