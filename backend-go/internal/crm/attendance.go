package crm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// TopicAttendanceSaved: điểm danh của một buổi vừa đổi → "Hôm nay" của nhân viên làm mới (≤ 60 s).
const TopicAttendanceSaved = "crm.attendance_saved"

const maxNote = 200

// Trạng thái điểm danh (khớp enum `attendance_status`).
var statuses = []string{"PRESENT", "LATE", "EXCUSED", "ABSENT"}

// GridSession là buổi của lưới.
type GridSession struct {
	ID               uuid.UUID
	SessionNo        int
	StartsAt, EndsAt time.Time
	Room, Topic      *string
	AttendanceDoneAt *time.Time
}

// Record là dòng điểm danh hiện có của một sinh viên.
type Record struct {
	Status   string
	Note     *string
	Version  int
	MarkedBy string // tên người điểm
	MarkedAt time.Time
}

// GridStudent là một hàng của lưới; Record nil = chưa có dòng (giao diện hiện mặc định "Có mặt" chưa lưu).
type GridStudent struct {
	ID          uuid.UUID
	FullName    string
	StudentCode *string
	Record      *Record
	SpeakCount  int
}

// Grid là `GET …/sessions/{sid}/attendance`.
type Grid struct {
	Session  GridSession
	Students []GridStudent
}

// GetGrid đọc lưới một buổi của lớp (TA / TEACHER). Buổi của lớp khác → 404.
func (s *Service) GetGrid(ctx context.Context, courseID, sid uuid.UUID) (Grid, error) {
	q := store.New(s.Pool)
	ses, err := q.CrmGetSession(ctx, store.CrmGetSessionParams{CourseID: courseID, ID: sid})
	if errors.Is(err, pgx.ErrNoRows) {
		return Grid{}, notFound()
	}
	if err != nil {
		return Grid{}, wrap("đọc buổi", err)
	}
	rows, err := q.CrmGridStudents(ctx, store.CrmGridStudentsParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return Grid{}, wrap("đọc lưới", err)
	}
	g := Grid{Session: GridSession{ID: ses.ID, SessionNo: int(ses.SessionNo), StartsAt: ses.StartsAt, EndsAt: ses.EndsAt, Room: ses.Room, Topic: ses.Topic, AttendanceDoneAt: ses.AttendanceDoneAt}, Students: make([]GridStudent, len(rows))}
	for i, r := range rows {
		st := GridStudent{ID: r.StudentID, FullName: r.FullName, StudentCode: r.StudentCode, SpeakCount: int(r.SpeakCount)}
		if r.RecStatus != nil && r.RecVersion != nil {
			rec := &Record{Status: string(*r.RecStatus), Note: r.RecNote, Version: int(*r.RecVersion)}
			if r.RecMarkedBy != nil {
				rec.MarkedBy = *r.RecMarkedBy
			}
			if r.RecMarkedAt != nil {
				rec.MarkedAt = *r.RecMarkedAt
			}
			st.Record = rec
		}
		g.Students[i] = st
	}
	return g, nil
}

// RecordIn là một hàng trong thân PUT. Version bắt buộc khi hàng đã có trong DB và nội dung khác.
type RecordIn struct {
	StudentID uuid.UUID
	Status    string
	Note      *string
	Version   *int
}

// PutGridInput là thân `PUT …/sessions/{sid}/attendance`.
type PutGridInput struct {
	Records  []RecordIn
	Complete bool
}

// RecordOut là hàng sau khi lưu.
type RecordOut struct {
	StudentID uuid.UUID
	Status    string
	Version   int
}

// PutGridResult là phản hồi 200.
type PutGridResult struct {
	Records []RecordOut
	DoneAt  *time.Time
}

type conflictMine struct {
	Status string  `json:"status"`
	Note   *string `json:"note"`
}
type conflictMarkedBy struct {
	FullName string `json:"full_name"`
}
type conflictCurrent struct {
	Status   string           `json:"status"`
	Note     *string          `json:"note"`
	Version  int              `json:"version"`
	MarkedBy conflictMarkedBy `json:"marked_by"`
	MarkedAt time.Time        `json:"marked_at"`
}
type conflict struct {
	StudentID uuid.UUID       `json:"student_id"`
	Mine      conflictMine    `json:"mine"`
	Current   conflictCurrent `json:"current"`
}

type change struct {
	StudentID uuid.UUID `json:"student_id"`
	From      *string   `json:"from"`
	To        string    `json:"to"`
}

func invalid(field, code, msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: field, Code: code, Message: msg})
}

func blank(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

func sameNote(a, b *string) bool {
	a, b = blank(a), blank(b)
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// PutGrid ghi cả lưới bằng MỘT giao dịch (SRS 4.3.3): khoá buổi → so version → ghi → hoàn tất → audit_log + outbox → commit.
// Có xung đột ⇒ rollback, 409 VERSION_CONFLICT (tất cả hoặc không). Hàng giống hệt DB bỏ qua (không đổi version, không audit).
func (s *Service) PutGrid(ctx context.Context, actor, courseID, sid uuid.UUID, in PutGridInput) (PutGridResult, error) {
	if len(in.Records) > s.maxRows() {
		return PutGridResult{}, invalid("records", "TOO_MANY", fmt.Sprintf("Mỗi lần lưu tối đa %d dòng.", s.maxRows()))
	}
	seen := make(map[uuid.UUID]bool, len(in.Records))
	for i, r := range in.Records {
		f := fmt.Sprintf("records[%d]", i)
		switch {
		case !slices.Contains(statuses, r.Status):
			return PutGridResult{}, invalid(f+".status", "INVALID", "Trạng thái là có mặt, muộn, vắng phép hoặc vắng.")
		case r.Note != nil && utf8.RuneCountInString(*r.Note) > maxNote:
			return PutGridResult{}, invalid(f+".note", "TOO_LONG", "Ghi chú tối đa 200 ký tự.")
		case seen[r.StudentID]:
			return PutGridResult{}, invalid(f+".student_id", "DUPLICATE", "Mỗi sinh viên chỉ một dòng.")
		}
		seen[r.StudentID] = true
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return PutGridResult{}, wrap("mở giao dịch", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	c, err := q.GetCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return PutGridResult{}, notFound()
	}
	if err != nil {
		return PutGridResult{}, wrap("đọc lớp", err)
	}
	if c.Status != store.CourseStatusACTIVE {
		return PutGridResult{}, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	ses, err := q.LockSession(ctx, store.LockSessionParams{CourseID: courseID, ID: sid}) // tuần tự hoá hai lần lưu cùng buổi
	if errors.Is(err, pgx.ErrNoRows) {
		return PutGridResult{}, notFound()
	}
	if err != nil {
		return PutGridResult{}, wrap("khoá buổi", err)
	}
	active, err := q.CrmActiveStudentIDs(ctx, courseID)
	if err != nil {
		return PutGridResult{}, wrap("sinh viên của lớp", err)
	}
	isActive := make(map[uuid.UUID]bool, len(active))
	for _, u := range active {
		isActive[u] = true
	}
	var outside []uuid.UUID
	for _, r := range in.Records {
		if !isActive[r.StudentID] {
			outside = append(outside, r.StudentID)
		}
	}
	if len(outside) > 0 {
		return PutGridResult{}, apierr.New(http.StatusUnprocessableEntity, apierr.StudentNotInCourse).WithDetails(map[string]any{"student_ids": outside})
	}
	recs, err := q.CrmSessionRecords(ctx, store.CrmSessionRecordsParams{CourseID: courseID, SessionID: sid})
	if err != nil {
		return PutGridResult{}, wrap("dòng điểm danh", err)
	}
	cur := make(map[uuid.UUID]store.CrmSessionRecordsRow, len(recs))
	before := map[string]int{}
	for _, r := range recs {
		cur[r.StudentID] = r
		before[string(r.Status)]++
	}

	var ins, upd batch
	var changes []change
	var conflicts []conflict
	out := make([]RecordOut, 0, len(in.Records))
	for _, r := range in.Records {
		old, has := cur[r.StudentID]
		switch {
		case !has:
			ins.add(r.StudentID, r.Status, str(r.Note))
			changes = append(changes, change{StudentID: r.StudentID, To: r.Status})
		case string(old.Status) == r.Status && sameNote(old.Note, r.Note):
			out = append(out, RecordOut{StudentID: r.StudentID, Status: r.Status, Version: int(old.Version)}) // giống hệt: bỏ qua
		case r.Version == nil || int32(*r.Version) != old.Version:
			conflicts = append(conflicts, conflict{StudentID: r.StudentID, Mine: conflictMine{Status: r.Status, Note: blank(r.Note)},
				Current: conflictCurrent{Status: string(old.Status), Note: old.Note, Version: int(old.Version), MarkedBy: conflictMarkedBy{FullName: old.MarkedByName}, MarkedAt: old.UpdatedAt}})
		default:
			upd.add(r.StudentID, r.Status, str(r.Note))
			from := string(old.Status)
			changes = append(changes, change{StudentID: r.StudentID, From: &from, To: r.Status})
		}
	}
	if len(conflicts) > 0 {
		return PutGridResult{}, apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"conflicts": conflicts})
	}

	for _, step := range []struct {
		name   string
		b      batch
		insert bool
	}{{"chèn dòng điểm danh", ins, true}, {"sửa dòng điểm danh", upd, false}} {
		rows, err := step.b.run(ctx, q, courseID, sid, actor, step.insert)
		if err != nil {
			return PutGridResult{}, wrap(step.name, err)
		}
		for _, x := range rows {
			out = append(out, RecordOut{StudentID: x.id, Status: step.b.status[x.id], Version: int(x.v)})
		}
	}

	// Hoàn tất: sinh viên ACTIVE còn thiếu dòng được ghi PRESENT cùng giao dịch (bất biến SRS 4.3.1).
	if in.Complete {
		have := make(map[uuid.UUID]bool, len(cur)+len(in.Records))
		for id := range cur {
			have[id] = true
		}
		for _, r := range in.Records {
			have[r.StudentID] = true
		}
		var fill batch
		for _, u := range active {
			if !have[u] {
				fill.add(u, "PRESENT", "")
				changes = append(changes, change{StudentID: u, To: "PRESENT"})
			}
		}
		rows, err := fill.run(ctx, q, courseID, sid, actor, true)
		if err != nil {
			return PutGridResult{}, wrap("điền có mặt", err)
		}
		for _, x := range rows {
			out = append(out, RecordOut{StudentID: x.id, Status: "PRESENT", Version: int(x.v)})
		}
	}
	doneAt := ses.AttendanceDoneAt
	firstComplete := false
	if in.Complete && ses.AttendanceDoneAt == nil {
		now := s.now()
		at, err := q.CrmCompleteSession(ctx, store.CrmCompleteSessionParams{At: &now, By: &actor, CourseID: courseID, ID: sid})
		if err != nil {
			return PutGridResult{}, wrap("hoàn tất buổi", err)
		}
		doneAt, firstComplete = at, true
	}
	if len(changes) == 0 && !firstComplete {
		return PutGridResult{Records: out, DoneAt: doneAt}, tx.Commit(ctx) // gửi lại cùng nội dung: không audit, không outbox
	}

	action := "save"
	switch {
	case firstComplete:
		action = "complete"
	case ses.AttendanceDoneAt != nil:
		action = "edit_after_complete"
	}
	after := map[string]int{}
	for id, r := range cur {
		_ = id
		after[string(r.Status)]++
	}
	for _, ch := range changes {
		if ch.From != nil {
			after[*ch.From]--
		}
		after[ch.To]++
	}
	if err := audit(ctx, q, "attendance", courseID, actor, sid.String(), action, map[string]any{"counts": before}, map[string]any{"counts": trim(after), "changes": changes}); err != nil {
		return PutGridResult{}, err
	}
	if _, err := outbox.Write(ctx, tx, TopicAttendanceSaved, map[string]string{"course_id": courseID.String(), "session_id": sid.String()}); err != nil {
		return PutGridResult{}, wrap("outbox", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PutGridResult{}, wrap("commit", err)
	}
	return PutGridResult{Records: out, DoneAt: doneAt}, nil
}

// trim bỏ trạng thái có số 0 (sau khi trừ bản cũ) để audit gọn.
func trim(m map[string]int) map[string]int {
	for k, v := range m {
		if v == 0 {
			delete(m, k)
		}
	}
	return m
}

// batch gom ba danh sách song song cho câu INSERT / UPDATE hàng loạt.
type batch struct {
	order  []uuid.UUID
	sts    []string
	notes  []string
	status map[uuid.UUID]string
}

func (b *batch) add(id uuid.UUID, status, note string) {
	if b.status == nil {
		b.status = map[uuid.UUID]string{}
	}
	b.order = append(b.order, id)
	b.sts = append(b.sts, status)
	b.notes = append(b.notes, note)
	b.status[id] = status
}

type rowVer struct {
	id uuid.UUID
	v  int32
}

// run chạy MỘT câu INSERT (insert) hoặc UPDATE hàng loạt cho cả lô; lô rỗng không chạm DB.
func (b batch) run(ctx context.Context, q *store.Queries, courseID, sid, actor uuid.UUID, insert bool) ([]rowVer, error) {
	if len(b.order) == 0 {
		return nil, nil
	}
	ids := b.ids()
	if insert {
		rows, err := q.CrmInsertRecords(ctx, store.CrmInsertRecordsParams{CourseID: courseID, SessionID: sid, MarkedBy: actor, StudentIds: ids, Statuses: b.sts, Notes: b.notes})
		out := make([]rowVer, len(rows))
		for i, r := range rows {
			out[i] = rowVer{r.StudentID, r.Version}
		}
		return out, err
	}
	rows, err := q.CrmUpdateRecords(ctx, store.CrmUpdateRecordsParams{CourseID: courseID, SessionID: sid, MarkedBy: actor, StudentIds: ids, Statuses: b.sts, Notes: b.notes})
	out := make([]rowVer, len(rows))
	for i, r := range rows {
		out[i] = rowVer{r.StudentID, r.Version}
	}
	return out, err
}

func (b batch) ids() []string {
	out := make([]string, len(b.order))
	for i, id := range b.order {
		out[i] = id.String()
	}
	return out
}
