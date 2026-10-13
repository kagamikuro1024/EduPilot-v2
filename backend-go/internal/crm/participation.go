package crm

import (
	"context"
	"errors"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

var (
	minPoints = decimal.NewFromInt(-5)
	maxPoints = decimal.NewFromInt(5)
)

// ParticipationIn là thân `POST …/participation`. Points nil ⇒ SPEAK dùng điểm mặc định. ClientEventID bắt buộc (khử trùng khi thử lại).
type ParticipationIn struct {
	StudentID     uuid.UUID
	SessionID     *uuid.UUID
	Type          string
	Points        *decimal.Decimal
	Note          *string
	ClientEventID uuid.UUID
}

// Participation là một sự kiện tham gia.
type Participation struct {
	ID            uuid.UUID
	CourseID      uuid.UUID
	StudentID     uuid.UUID
	SessionID     *uuid.UUID
	Type          string
	Points        decimal.Decimal
	Note          *string
	ClientEventID uuid.UUID
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
}

// ParticipationRow là một dòng danh sách (kèm tên sinh viên — chỉ nhân viên đọc).
type ParticipationRow struct {
	ID        uuid.UUID
	StudentID uuid.UUID
	FullName  string
	SessionID *uuid.UUID
	Type      string
	Points    decimal.Decimal
	Note      *string
	CreatedBy uuid.UUID
	CreatedAt time.Time
}

func validPoints(typ string, p decimal.Decimal) *apierr.Error {
	switch {
	case p.LessThan(minPoints) || p.GreaterThan(maxPoints) || !p.Equal(p.Round(2)):
		return invalid("points", "OUT_OF_RANGE", "Điểm từ -5 đến 5, bước 0,01.")
	case typ == "SPEAK" && p.IsNegative():
		return invalid("points", "SIGN", "Phát biểu không âm.")
	case typ == "BONUS" && !p.IsPositive():
		return invalid("points", "SIGN", "Điểm cộng phải lớn hơn 0.")
	case typ == "PENALTY" && !p.IsNegative():
		return invalid("points", "SIGN", "Điểm trừ phải nhỏ hơn 0.")
	}
	return nil
}

// AddParticipation ghi một sự kiện (SRS 4.4). Trùng client_event_id của CHÍNH người này ở CHÍNH lớp này ⇒ trả sự kiện cũ (created=false);
// trùng của lớp / người khác ⇒ 409 IDEMPOTENCY_KEY_REUSED (không bao giờ lộ sự kiện đó — TLR-8).
func (s *Service) AddParticipation(ctx context.Context, actor, courseID uuid.UUID, in ParticipationIn) (p Participation, created bool, err error) {
	if in.Type != "SPEAK" && in.Type != "BONUS" && in.Type != "PENALTY" {
		return Participation{}, false, invalid("type", "INVALID", "Loại là phát biểu, điểm cộng hoặc điểm trừ.")
	}
	if in.ClientEventID == uuid.Nil {
		return Participation{}, false, invalid("client_event_id", "REQUIRED", "Thiếu mã sự kiện.")
	}
	if in.Note != nil && utf8.RuneCountInString(*in.Note) > maxNote {
		return Participation{}, false, invalid("note", "TOO_LONG", "Ghi chú tối đa 200 ký tự.")
	}
	var pts decimal.Decimal
	switch {
	case in.Points != nil:
		pts = *in.Points
	case in.Type == "SPEAK":
		pts = s.speakPoints()
	default:
		return Participation{}, false, invalid("points", "REQUIRED", "Cần số điểm.")
	}
	if verr := validPoints(in.Type, pts); verr != nil {
		return Participation{}, false, verr
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Participation{}, false, wrap("mở giao dịch", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	c, err := q.GetCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Participation{}, false, notFound()
	}
	if err != nil {
		return Participation{}, false, wrap("đọc lớp", err)
	}
	if c.Status != store.CourseStatusACTIVE {
		return Participation{}, false, apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	ok, err := q.CrmStudentActive(ctx, store.CrmStudentActiveParams{CourseID: courseID, StudentID: in.StudentID})
	if err != nil {
		return Participation{}, false, wrap("kiểm sinh viên", err)
	}
	if !ok {
		return Participation{}, false, apierr.New(http.StatusUnprocessableEntity, apierr.StudentNotInCourse).WithDetails(map[string]any{"student_ids": []uuid.UUID{in.StudentID}})
	}
	if in.SessionID != nil {
		exists, err := q.CrmSessionInCourse(ctx, store.CrmSessionInCourseParams{CourseID: courseID, ID: *in.SessionID})
		if err != nil {
			return Participation{}, false, wrap("kiểm buổi", err)
		}
		if !exists {
			return Participation{}, false, invalid("session_id", "NOT_FOUND", "Buổi này không thuộc lớp.")
		}
	}
	row, err := q.CrmInsertParticipation(ctx, store.CrmInsertParticipationParams{
		CourseID: courseID, StudentID: in.StudentID, SessionID: in.SessionID, Type: store.ParticipationType(in.Type), Points: pts, Note: blank(in.Note), ClientEventID: in.ClientEventID, CreatedBy: actor,
	})
	if errors.Is(err, pgx.ErrNoRows) { // trùng client_event_id
		_ = tx.Rollback(ctx)
		old, err := store.New(s.Pool).CrmParticipationByClientEvent(ctx, store.CrmParticipationByClientEventParams{ClientEventID: in.ClientEventID, CourseID: courseID, CreatedBy: actor})
		if errors.Is(err, pgx.ErrNoRows) {
			return Participation{}, false, apierr.New(http.StatusConflict, apierr.IdempotencyReused)
		}
		if err != nil {
			return Participation{}, false, wrap("đọc lại sự kiện", err)
		}
		return Participation{ID: old.ID, CourseID: old.CourseID, StudentID: old.StudentID, SessionID: old.SessionID, Type: string(old.Type), Points: old.Points, Note: old.Note, ClientEventID: old.ClientEventID, CreatedBy: old.CreatedBy, CreatedAt: old.CreatedAt}, false, nil
	}
	if err != nil {
		return Participation{}, false, wrap("ghi sự kiện", err)
	}
	if err := audit(ctx, q, "participation", courseID, actor, row.ID.String(), "create", nil, map[string]any{"student_id": in.StudentID, "type": in.Type, "points": pts.String()}); err != nil {
		return Participation{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Participation{}, false, wrap("commit", err)
	}
	return Participation{ID: row.ID, CourseID: row.CourseID, StudentID: row.StudentID, SessionID: row.SessionID, Type: string(row.Type), Points: row.Points, Note: row.Note, ClientEventID: row.ClientEventID, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt}, true, nil
}

// DeleteParticipation xoá một sự kiện: người tạo hoặc TEACHER của lớp (TA xoá của người khác → 403). Xoá lại → 404.
func (s *Service) DeleteParticipation(ctx context.Context, actor uuid.UUID, teacher bool, courseID, id uuid.UUID) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return wrap("mở giao dịch", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	c, err := q.GetCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		return wrap("đọc lớp", err)
	}
	if c.Status != store.CourseStatusACTIVE {
		return apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	ev, err := q.CrmParticipationGet(ctx, store.CrmParticipationGetParams{CourseID: courseID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		return wrap("đọc sự kiện", err)
	}
	if ev.CreatedBy != actor && !teacher {
		return apierr.New(http.StatusForbidden, apierr.Forbidden)
	}
	if n, err := q.CrmParticipationDelete(ctx, store.CrmParticipationDeleteParams{CourseID: courseID, ID: id}); err != nil {
		return wrap("xoá sự kiện", err)
	} else if n == 0 {
		return notFound()
	}
	if err := audit(ctx, q, "participation", courseID, actor, id.String(), "delete", map[string]any{"student_id": ev.StudentID, "type": string(ev.Type), "points": ev.Points.String()}, nil); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return wrap("commit", err)
	}
	return nil
}

// ListParticipation liệt kê sự kiện của lớp, mới nhất trước, con trỏ (created_at, id); lọc theo sinh viên / buổi. Trả limit+1 dòng.
func (s *Service) ListParticipation(ctx context.Context, courseID uuid.UUID, student, session *uuid.UUID, afterAt *time.Time, afterID *uuid.UUID, limit int) ([]ParticipationRow, error) {
	rows, err := store.New(s.Pool).CrmParticipationList(ctx, store.CrmParticipationListParams{CourseID: courseID, StudentID: student, SessionID: session, CurAt: afterAt, CurID: afterID, Lim: int32(limit + 1)})
	if err != nil {
		return nil, wrap("liệt kê sự kiện", err)
	}
	out := make([]ParticipationRow, len(rows))
	for i, r := range rows {
		out[i] = ParticipationRow{ID: r.ID, StudentID: r.StudentID, FullName: r.FullName, SessionID: r.SessionID, Type: string(r.Type), Points: r.Points, Note: r.Note, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt}
	}
	return out, nil
}
