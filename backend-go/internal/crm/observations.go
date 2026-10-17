package crm

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn ghi chú quan sát (SRS 4.8.4, schema 5.4): ≤ 5 nhãn mỗi nhãn 1–30 ký tự; thân 1–2000 ký tự.
const (
	obsMaxTags    = 5
	obsTagMaxLen  = 30
	obsBodyMaxLen = 2000
)

// Observation là một ghi chú quan sát. Chỉ nhân viên của lớp đọc được; không bao giờ vào lời nhắc nào (chat / threads / RAG).
type Observation struct {
	ID         uuid.UUID
	AuthorID   uuid.UUID
	AuthorName string
	Kind       string
	Tags       []string
	Body       string
	AIEdited   bool
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ObservationIn là thân POST / PUT. Kind chỉ dùng khi tạo: "NOTE" (mặc định) hoặc "AI_SUMMARY" (lưu nhận xét AI đã soạn; AIEdited = bản lưu khác bản AI).
type ObservationIn struct {
	Kind     string
	Tags     []string
	Body     string
	AIEdited bool
	Version  int // chỉ PUT
}

func (in *ObservationIn) validate() error {
	in.Body = strings.TrimSpace(in.Body)
	var fe []apierr.FieldError
	if n := utf8.RuneCountInString(in.Body); n < 1 || n > obsBodyMaxLen {
		fe = append(fe, apierr.FieldError{Field: "body", Code: "length", Message: "Ghi chú dài từ 1 đến 2.000 ký tự."})
	}
	if len(in.Tags) > obsMaxTags {
		fe = append(fe, apierr.FieldError{Field: "tags", Code: "max", Message: "Tối đa 5 nhãn."})
	}
	for _, t := range in.Tags {
		if n := utf8.RuneCountInString(t); n < 1 || n > obsTagMaxLen {
			fe = append(fe, apierr.FieldError{Field: "tags", Code: "length", Message: "Mỗi nhãn dài từ 1 đến 30 ký tự."})
			break
		}
	}
	if in.Kind != "" && in.Kind != "NOTE" && in.Kind != "AI_SUMMARY" {
		fe = append(fe, apierr.FieldError{Field: "kind", Code: "invalid", Message: "Loại ghi chú không hợp lệ."})
	}
	if in.Kind != "AI_SUMMARY" && in.AIEdited {
		fe = append(fe, apierr.FieldError{Field: "ai_edited", Code: "invalid", Message: "Chỉ nhận xét AI mới có đánh dấu đã sửa."})
	}
	if len(fe) > 0 {
		return apierr.Validation(fe...)
	}
	if in.Tags == nil {
		in.Tags = []string{}
	}
	return nil
}

func obsOf(o store.StudentObservation, author string) Observation {
	return Observation{ID: o.ID, AuthorID: o.AuthorID, AuthorName: author, Kind: string(o.Kind), Tags: o.Tags, Body: o.Body, AIEdited: o.AiEdited, Version: int(o.Version), CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}
}

// ListObservations đọc ghi chú (mới nhất trước, con trỏ): trong MỘT giao dịch — ghi `audit_log` `view_observations` RỒI đọc; audit lỗi → 503. Sinh viên không ACTIVE → 404.
// Mọi nhân viên của lớp đọc được mọi ghi chú (guard Staff ở route).
func (s *Service) ListObservations(ctx context.Context, actor, courseID, studentID uuid.UUID, afterAt *time.Time, afterID *uuid.UUID, limit int) ([]Observation, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, wrap("mở giao dịch ghi chú", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := s.requireActive(ctx, q, courseID, studentID); err != nil {
		return nil, err
	}
	if err := audit(ctx, q, "student_profile", courseID, actor, studentID.String(), "view_observations", nil, nil); err != nil {
		return nil, unavailable()
	}
	rows, err := q.CrmObsList(ctx, store.CrmObsListParams{CourseID: courseID, StudentID: studentID, CurAt: afterAt, CurID: afterID, Lim: int32(limit + 1)}) //nolint:gosec // ≤ 101
	if err != nil {
		return nil, wrap("liệt kê ghi chú", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, wrap("commit ghi chú", err)
	}
	out := make([]Observation, len(rows))
	for i, r := range rows {
		out[i] = Observation{ID: r.ID, AuthorID: r.AuthorID, AuthorName: r.AuthorName, Kind: string(r.Kind), Tags: r.Tags, Body: r.Body, AIEdited: r.AiEdited, Version: int(r.Version), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	}
	return out, nil
}

// CreateObservation ghi một ghi chú (tác giả = người gọi) và `audit_log` KHÔNG kèm nội dung, cùng giao dịch.
func (s *Service) CreateObservation(ctx context.Context, actor, courseID, studentID uuid.UUID, in ObservationIn) (Observation, error) {
	if err := in.validate(); err != nil {
		return Observation{}, err
	}
	if in.Kind == "" {
		in.Kind = "NOTE"
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Observation{}, wrap("mở giao dịch ghi chú", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := s.requireActive(ctx, q, courseID, studentID); err != nil {
		return Observation{}, err
	}
	o, err := q.CrmObsInsert(ctx, store.CrmObsInsertParams{CourseID: courseID, StudentID: studentID, AuthorID: actor, Kind: store.ObservationKind(in.Kind), Tags: in.Tags, Body: in.Body, AiEdited: in.AIEdited})
	if err != nil {
		return Observation{}, wrap("ghi ghi chú", err)
	}
	if err := audit(ctx, q, "student_observation", courseID, actor, o.ID.String(), "create", nil, map[string]any{"student_id": studentID, "kind": in.Kind}); err != nil {
		return Observation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Observation{}, wrap("commit ghi chú", err)
	}
	return obsOf(o, ""), nil
}

// ObservationConflict: sai version khi sửa (hai nhân viên cùng sửa) → 409 kèm bản hiện tại để người dùng chọn.
type ObservationConflict struct{ Current Observation }

func (e *ObservationConflict) Error() string { return "crm: sai version ghi chú" }

// UpdateObservation sửa ghi chú: CHỈ tác giả (người khác → 403, kể cả TEACHER với ghi chú của TA); sai version → 409.
func (s *Service) UpdateObservation(ctx context.Context, actor, courseID, studentID, id uuid.UUID, in ObservationIn) (Observation, error) {
	if err := in.validate(); err != nil {
		return Observation{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Observation{}, wrap("mở giao dịch ghi chú", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.CrmObsGet(ctx, store.CrmObsGetParams{CourseID: courseID, StudentID: studentID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return Observation{}, notFound()
	}
	if err != nil {
		return Observation{}, wrap("đọc ghi chú", err)
	}
	if cur.AuthorID != actor {
		return Observation{}, apierr.New(http.StatusForbidden, apierr.Forbidden)
	}
	o, err := q.CrmObsUpdate(ctx, store.CrmObsUpdateParams{CourseID: courseID, StudentID: studentID, ID: id, AuthorID: actor, Tags: in.Tags, Body: in.Body, Version: int32(in.Version)}) //nolint:gosec // version nhỏ
	if errors.Is(err, pgx.ErrNoRows) {
		return Observation{}, &ObservationConflict{Current: obsOf(cur, "")}
	}
	if err != nil {
		return Observation{}, wrap("sửa ghi chú", err)
	}
	if err := audit(ctx, q, "student_observation", courseID, actor, id.String(), "update", nil, map[string]any{"student_id": studentID}); err != nil {
		return Observation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Observation{}, wrap("commit ghi chú", err)
	}
	return obsOf(o, ""), nil
}

// DeleteObservation xoá thật ghi chú của chính mình (người khác → 403) kèm audit không nội dung.
func (s *Service) DeleteObservation(ctx context.Context, actor, courseID, studentID, id uuid.UUID) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return wrap("mở giao dịch ghi chú", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.CrmObsGet(ctx, store.CrmObsGetParams{CourseID: courseID, StudentID: studentID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		return wrap("đọc ghi chú", err)
	}
	if cur.AuthorID != actor {
		return apierr.New(http.StatusForbidden, apierr.Forbidden)
	}
	n, err := q.CrmObsDelete(ctx, store.CrmObsDeleteParams{CourseID: courseID, StudentID: studentID, ID: id, AuthorID: actor})
	if err != nil {
		return wrap("xoá ghi chú", err)
	}
	if n == 0 {
		return notFound()
	}
	if err := audit(ctx, q, "student_observation", courseID, actor, id.String(), "delete", nil, map[string]any{"student_id": studentID}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return wrap("commit ghi chú", err)
	}
	return nil
}
