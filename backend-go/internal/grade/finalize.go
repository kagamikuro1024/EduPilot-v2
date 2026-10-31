package grade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Hằng số chốt / mở khoá (SRS FEAT-gradebook 4.7).
const (
	// TopicFinalized: một bản chốt vừa được tạo → thông báo chuông cho sinh viên. Payload {course_id, version}.
	TopicFinalized = "grade.finalized"

	incompleteListMax = 30 // GRADES_INCOMPLETE liệt kê tối đa 30 sinh viên (kèm `total`)
	unlockReasonMin   = 10
	unlockReasonMax   = 500
	finalizeChunk     = 500
)

// FinalizeOut là phản hồi của `POST …/gradebook/finalize`.
type FinalizeOut struct {
	Version  int       `json:"version"`
	Students int       `json:"students"`
	LockedAt time.Time `json:"locked_at"`
}

// snapRow là một phần tử của lô `jsonb_to_recordset` của `GradeSnapshotsInsert`.
type snapRow struct {
	StudentID    uuid.UUID       `json:"student_id"`
	Breakdown    json.RawMessage `json:"breakdown"`
	NumericGrade string          `json:"numeric_grade"`
	Letter       string          `json:"letter"`
	FailedReason string          `json:"failed_reason"`
}

// breakdown dựng `final_grade_snapshots.breakdown`: đầu vào + trung gian + dòng giải trình có `source_quote`. Số là chuỗi.
func breakdown(conf Loaded, r Result, in Input) json.RawMessage {
	type comp struct {
		Key    string `json:"key"`
		Name   string `json:"name"`
		Grp    string `json:"grp"`
		Weight string `json:"weight"`
		Score  string `json:"score,omitempty"`
	}
	type line struct {
		Label       string `json:"label"`
		Expr        string `json:"expr"`
		Value       string `json:"value"`
		SourceQuote string `json:"source_quote,omitempty"`
		Page        int    `json:"page,omitempty"`
		FilledBy    bool   `json:"filled_by_user,omitempty"`
	}
	b := struct {
		Scheme     map[string]any    `json:"scheme"`
		Components []comp            `json:"components"`
		Items      map[string]string `json:"items"`
		Bonus      string            `json:"bonus"`
		Penalty    string            `json:"penalty"`
		Absent     int               `json:"absent"`
		QT         string            `json:"qt,omitempty"`
		CK         string            `json:"ck,omitempty"`
		HPRaw      string            `json:"hp_raw,omitempty"`
		HP         string            `json:"hp,omitempty"`
		Letter     string            `json:"letter,omitempty"`
		Failed     string            `json:"failed,omitempty"`
		Lines      []line            `json:"lines"`
	}{
		Scheme: map[string]any{"id": conf.ID, "scheme_no": conf.No, "version": conf.Version},
		Items:  map[string]string{}, Bonus: r.BonusApplied.String(), Penalty: r.Penalty.String(), Absent: in.Absent, Lines: []line{},
	}
	for _, c := range conf.Scheme.Components {
		for _, it := range c.Items {
			if v, ok := in.Scores[it.ID]; ok {
				b.Items[it.Key] = v.String()
			}
		}
	}
	for _, c := range r.Components {
		x := comp{Key: c.Key, Name: c.Name, Grp: string(c.Group), Weight: c.Weight.String()}
		if c.Computed {
			x.Score = c.Score.String()
		}
		b.Components = append(b.Components, x)
	}
	if r.HasQT {
		b.QT = r.QT.String()
	}
	if r.HasCK {
		b.CK = r.CK.String()
	}
	b.HPRaw, b.HP, b.Letter = r.HPRaw.String(), r.HP.String(), r.Letter
	if strings.HasPrefix(r.Verdict, "LIET") {
		b.Failed = r.Verdict
	}
	for _, l := range r.Lines {
		x := line{Label: l.Label, Expr: l.Expr, Value: l.Value, Page: l.Page, FilledBy: l.FilledByUser}
		if l.Quote != nil {
			x.SourceQuote = *l.Quote
		}
		b.Lines = append(b.Lines, x)
	}
	raw, _ := json.Marshal(b)
	return raw
}

// Finalize chốt điểm cả lớp thành snapshot bất biến (SRS 4.7.1). Một giao dịch: khoá hàng `gradebooks` `FOR UPDATE` → LOCKED 409 → công thức CONFIRMED →
// `Compute` mọi sinh viên ACTIVE → còn thiếu điểm thì 409 `GRADES_INCOMPLETE` (không ghi gì) → MỘT câu INSERT snapshot → khoá sổ → audit → outbox.
func (s *Service) Finalize(ctx context.Context, actor, courseID uuid.UUID) (FinalizeOut, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return FinalizeOut{}, fmt.Errorf("grade: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := store.New(tx)

	gb, err := q.GradebookLockForUpdate(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return FinalizeOut{}, apierr.New(http.StatusConflict, apierr.SchemeNotConfirmed) // chưa từng xác nhận công thức → chưa có hàng sổ điểm
	}
	if err != nil {
		return FinalizeOut{}, fmt.Errorf("grade: khoá sổ điểm: %w", err)
	}
	if gb.Status == store.GradebookStatusLOCKED {
		return FinalizeOut{}, apierr.New(http.StatusConflict, apierr.GradebookLocked)
	}
	conf, ok, err := LoadConfirmed(ctx, q, courseID)
	if err != nil {
		return FinalizeOut{}, err
	}
	if !ok {
		return FinalizeOut{}, apierr.New(http.StatusConflict, apierr.SchemeNotConfirmed)
	}
	studs, err := ActiveStudents(ctx, q, courseID)
	if err != nil {
		return FinalizeOut{}, err
	}

	type miss struct {
		ID      uuid.UUID `json:"id"`
		Missing []string  `json:"missing"`
	}
	var incomplete []miss
	total := 0
	rows := make([]snapRow, 0, len(studs))
	for lo := 0; lo < len(studs); lo += finalizeChunk {
		part := studs[lo:min(lo+finalizeChunk, len(studs))]
		ids := make([]uuid.UUID, len(part))
		for i, st := range part {
			ids[i] = st.ID
		}
		in, err := LoadInputs(ctx, q, courseID, conf.ID, ids)
		if err != nil {
			return FinalizeOut{}, err
		}
		for _, st := range part {
			r := Compute(conf.Scheme, in[st.ID])
			if r.Status != StatusComplete {
				total++
				if len(incomplete) < incompleteListMax {
					incomplete = append(incomplete, miss{ID: st.ID, Missing: append([]string{}, r.Missing...)})
				}
				continue
			}
			sr := snapRow{StudentID: st.ID, Breakdown: breakdown(conf, r, in[st.ID]), NumericGrade: r.HP.String(), Letter: r.Letter}
			if strings.HasPrefix(r.Verdict, "LIET") {
				sr.FailedReason = r.Verdict
			}
			rows = append(rows, sr)
		}
	}
	if total > 0 || len(studs) == 0 {
		return FinalizeOut{}, apierr.New(http.StatusConflict, apierr.GradesIncomplete).WithDetails(map[string]any{"students": append([]miss{}, incomplete...), "total": total})
	}

	version := gb.SnapshotVersion + 1
	raw, err := json.Marshal(rows)
	if err != nil {
		return FinalizeOut{}, fmt.Errorf("grade: mã hoá bản chốt: %w", err)
	}
	if n, err := q.GradeSnapshotsInsert(ctx, store.GradeSnapshotsInsertParams{CourseID: courseID, Version: version, SchemeID: conf.ID, LockedBy: actor, UnlockReason: gb.UnlockReason, Rows: raw}); err != nil {
		return FinalizeOut{}, fmt.Errorf("grade: ghi bản chốt: %w", err)
	} else if int(n) != len(rows) {
		return FinalizeOut{}, fmt.Errorf("grade: bản chốt ghi %d / %d dòng", n, len(rows))
	}
	at, err := q.GradebookSetLocked(ctx, store.GradebookSetLockedParams{CourseID: courseID, Version: version})
	if err != nil {
		return FinalizeOut{}, fmt.Errorf("grade: khoá sổ điểm: %w", err)
	}
	after, _ := json.Marshal(map[string]any{"version": version, "students": len(rows), "scheme_id": conf.ID})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "gradebook", EntityID: courseID.String(), Action: "finalize", After: after}); err != nil {
		return FinalizeOut{}, fmt.Errorf("grade: audit chốt: %w", err) // fail-closed
	}
	if _, err := outbox.Write(ctx, tx, TopicFinalized, map[string]any{"course_id": courseID.String(), "version": version}); err != nil {
		return FinalizeOut{}, fmt.Errorf("grade: outbox %s: %w", TopicFinalized, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return FinalizeOut{}, fmt.Errorf("grade: chốt điểm: %w", err)
	}
	return FinalizeOut{Version: int(version), Students: len(rows), LockedAt: at}, nil
}

// UnlockOut là phản hồi của `POST …/gradebook/unlock`.
type UnlockOut struct {
	Status          string `json:"status"`
	SnapshotVersion int    `json:"snapshot_version"`
}

// Unlock mở khoá sổ đã chốt (SRS 4.7.4): lý do 10–500 ký tự, đang OPEN → 409 `NOT_LOCKED`. Bản chốt cũ không đổi.
func (s *Service) Unlock(ctx context.Context, actor, courseID uuid.UUID, reason string) (UnlockOut, error) {
	reason = strings.TrimSpace(reason)
	if n := utf8.RuneCountInString(reason); n < unlockReasonMin || n > unlockReasonMax {
		return UnlockOut{}, apierr.Validation(apierr.FieldError{Field: "reason", Code: "length", Message: "Lý do dài 10–500 ký tự."})
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return UnlockOut{}, fmt.Errorf("grade: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := store.New(tx)
	gb, err := q.GradebookLockForUpdate(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && gb.Status != store.GradebookStatusLOCKED) {
		return UnlockOut{}, apierr.New(http.StatusConflict, apierr.NotLocked)
	}
	if err != nil {
		return UnlockOut{}, fmt.Errorf("grade: khoá sổ điểm: %w", err)
	}
	if n, err := q.GradebookSetOpen(ctx, store.GradebookSetOpenParams{CourseID: courseID, Reason: &reason, ActorID: &actor}); err != nil || n != 1 {
		return UnlockOut{}, fmt.Errorf("grade: mở khoá: %w", err)
	}
	before, _ := json.Marshal(map[string]any{"status": "LOCKED", "snapshot_version": gb.SnapshotVersion})
	after, _ := json.Marshal(map[string]any{"status": "OPEN", "snapshot_version": gb.SnapshotVersion, "reason": reason})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "gradebook", EntityID: courseID.String(), Action: "unlock", Before: before, After: after}); err != nil {
		return UnlockOut{}, fmt.Errorf("grade: audit mở khoá: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return UnlockOut{}, fmt.Errorf("grade: mở khoá: %w", err)
	}
	return UnlockOut{Status: "OPEN", SnapshotVersion: int(gb.SnapshotVersion)}, nil
}

// HandleFinalized là outbox.Handler của `grade.finalized` (SRS 4.7.5): lần đầu GRADE_FINALIZED cho cả lớp; chốt lại chỉ báo GRADE_CHANGED cho sinh viên có
// điểm / chữ khác bản trước. Không có con số trong chữ; chỉ chuông; mỗi nhóm là MỘT câu INSERT (khử trùng `grade.finalized:{lớp}:v{n}:{user}`).
func (s *Service) HandleFinalized(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID string `json:"course_id"`
		Version  int32  `json:"version"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil {
		return fmt.Errorf("grade: payload %s: %w", m.Topic, err)
	}
	course, err := uuid.Parse(p.CourseID)
	if err != nil || p.Version < 1 || s.Notify == nil {
		return nil
	}
	q := store.New(s.Pool)
	info, err := q.GradeCourseInfo(ctx, course)
	if err != nil {
		return fmt.Errorf("grade: thông tin lớp: %w", err)
	}
	sp := notify.Spec{CourseID: &course, Type: TypeFinalized, Title: fmt.Sprintf("Điểm học phần lớp %s đã được chốt.", info.ClassCode), Link: new("/me"), DedupeKey: fmt.Sprintf("grade.finalized:%s:v%d", course, p.Version)}
	var ids []uuid.UUID
	if p.Version == 1 {
		ids, err = q.GradeSnapshotUsers(ctx, store.GradeSnapshotUsersParams{CourseID: course, Version: p.Version})
	} else {
		sp.Type, sp.Title = TypeChanged, fmt.Sprintf("Điểm học phần lớp %s đã được cập nhật.", info.ClassCode)
		ids, err = q.GradeSnapshotChangedUsers(ctx, store.GradeSnapshotChangedUsersParams{CourseID: course, Version: p.Version})
	}
	if err != nil {
		return fmt.Errorf("grade: người nhận thông báo chốt: %w", err)
	}
	if _, err := s.Notify.SendMany(ctx, ids, sp); err != nil {
		return fmt.Errorf("grade: gửi thông báo chốt: %w", err)
	}
	return nil
}
