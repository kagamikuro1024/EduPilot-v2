// Package audit: xem nhật ký kiểm toán cho ADMIN (US-P10-14). Chỉ đọc; KHÔNG bao giờ trả giá trị `before` / `after` (chỉ tên khoá cấp 1).
// Mỗi lần xem thành công ghi một dòng VIEW_LIST trong cùng giao dịch, trước khi trả dữ liệu.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/store"
)

const (
	defaultSpan = 7 * 24 * time.Hour
	maxSpan     = 90 * 24 * time.Hour
)

var entityRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// Service đọc audit_log.
type Service struct {
	Pool  *pgxpool.Pool
	Clock clock.Clock
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now()
	}
	return s.Clock.Now()
}

// Query là tham số đã đọc từ URL (chuỗi rỗng = thiếu).
type Query struct {
	Entity, ActorID, From, To, Cursor string
	Limit                             int
}

// Row là một dòng nhật ký không có giá trị.
type Row struct {
	ID            uuid.UUID  `json:"id"`
	At            time.Time  `json:"at"`
	ActorID       *uuid.UUID `json:"actor_id"`
	ActorName     *string    `json:"actor_name"`
	ActorRole     *string    `json:"actor_role"`
	Entity        string     `json:"entity"`
	EntityID      string     `json:"entity_id"`
	Action        string     `json:"action"`
	CourseID      *uuid.UUID `json:"course_id"`
	TraceID       *string    `json:"trace_id"`
	ChangedFields []string   `json:"changed_fields"`
}

type cursorBody struct {
	V int    `json:"v"`
	T int64  `json:"t"`
	I string `json:"i"`
	F string `json:"f"` // dấu vân tay của bộ lọc đã chuẩn hoá (không gồm mặc định now / now − 7 ngày)
}

func fingerprint(entity string, actor *uuid.UUID, from, to *time.Time) string {
	a, f, t := "-", "-", "-"
	if actor != nil {
		a = actor.String()
	}
	if from != nil {
		f = from.UTC().Format(time.RFC3339Nano)
	}
	if to != nil {
		t = to.UTC().Format(time.RFC3339Nano)
	}
	h := sha256.Sum256([]byte("e=" + entity + "|a=" + a + "|f=" + f + "|t=" + t))
	return hex.EncodeToString(h[:6])
}

// List: kiểm tham số → đọc trang → ghi VIEW_LIST (cùng giao dịch) → trả. Ghi hỏng → lỗi, không trả dòng nào.
func (s *Service) List(ctx context.Context, admin uuid.UUID, qy Query, traceID string) (httpx.Page[Row], error) {
	var (
		errs     []apierr.FieldError
		entity   string
		actor    *uuid.UUID
		from, to *time.Time
		limit    = httpx.DefaultLimit
	)
	add := func(f, m string) { errs = append(errs, apierr.FieldError{Field: f, Code: "invalid", Message: m}) }
	if qy.Entity != "" {
		if !entityRE.MatchString(qy.Entity) {
			add("entity", "entity gồm chữ thường, số và dấu gạch dưới.")
		}
		entity = qy.Entity
	}
	if qy.ActorID != "" {
		id, err := uuid.Parse(qy.ActorID)
		if err != nil {
			add("actor_id", "actor_id không hợp lệ.")
		} else {
			actor = &id
		}
	}
	for _, f := range []struct {
		name, raw string
		dst       **time.Time
	}{{"from", qy.From, &from}, {"to", qy.To, &to}} {
		if f.raw == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, f.raw)
		if err != nil {
			add(f.name, f.name+" phải theo RFC 3339.")
			continue
		}
		t = t.UTC()
		*f.dst = &t
	}
	if qy.Limit != 0 {
		if qy.Limit < 1 || qy.Limit > httpx.MaxLimit {
			add("limit", "limit phải từ 1 đến 100.")
		} else {
			limit = qy.Limit
		}
	}
	// khoảng áp dụng
	now := s.now().UTC()
	af, at := now.Add(-defaultSpan), now
	switch {
	case from != nil && to != nil:
		af, at = *from, *to
	case from != nil:
		af = *from
	case to != nil:
		at, af = *to, to.Add(-defaultSpan)
	}
	if len(errs) == 0 && !af.Before(at) {
		add("from", "from phải trước to.")
	}
	if len(errs) == 0 && at.Sub(af) > maxSpan {
		add("from", "Khoảng thời gian tối đa 90 ngày.")
	}
	if len(errs) > 0 {
		return httpx.Page[Row]{}, apierr.Validation(errs...)
	}
	fp := fingerprint(entity, actor, from, to)
	var curAt *time.Time
	var curID *uuid.UUID
	if qy.Cursor != "" {
		var c cursorBody
		raw, err := base64.RawURLEncoding.DecodeString(qy.Cursor)
		if err != nil || json.Unmarshal(raw, &c) != nil || c.V != 1 || c.F != fp {
			return httpx.Page[Row]{}, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidCursor)
		}
		id, err := uuid.Parse(c.I)
		if err != nil {
			return httpx.Page[Row]{}, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidCursor)
		}
		t := time.UnixMicro(c.T).UTC()
		curAt, curID = &t, &id
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return httpx.Page[Row]{}, fmt.Errorf("audit: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	arg := store.AuditListParams{FromTs: af, ToTs: at, CursorAt: curAt, CursorID: curID, MaxRows: int32(limit + 1)} //nolint:gosec // ≤ 101
	if entity != "" {
		arg.Entity = &entity
	}
	arg.ActorID = actor
	rows, err := q.AuditList(ctx, arg)
	if err != nil {
		return httpx.Page[Row]{}, fmt.Errorf("audit: đọc nhật ký: %w", err)
	}
	out := make([]Row, len(rows))
	for i, r := range rows {
		var role *string
		if r.ActorRole != nil {
			v := string(*r.ActorRole)
			role = &v
		}
		out[i] = Row{ID: r.ID, At: r.CreatedAt, ActorID: r.ActorID, ActorName: r.ActorName, ActorRole: role, Entity: r.Entity, EntityID: r.EntityID, Action: r.Action, CourseID: r.CourseID, TraceID: r.TraceID, ChangedFields: r.ChangedFields}
		if out[i].ChangedFields == nil {
			out[i].ChangedFields = []string{}
		}
	}
	page := httpx.Paginate(out, httpx.PageParams{Limit: limit}, func(r Row) (time.Time, string) { return r.At, r.ID.String() })
	if page.NextCursor != nil {
		last := page.Items[len(page.Items)-1]
		raw, _ := json.Marshal(cursorBody{V: 1, T: last.At.UTC().UnixMicro(), I: last.ID.String(), F: fp}) //nolint:errchkjson // kiểu cố định
		c := base64.RawURLEncoding.EncodeToString(raw)
		page.NextCursor = &c
	}
	type nz = *string
	var ent, act nz
	if entity != "" {
		ent = &entity
	}
	if actor != nil {
		a := actor.String()
		act = &a
	}
	after, _ := json.Marshal(map[string]any{"entity": ent, "actor_id": act, "from": af.Format(time.RFC3339), "to": at.Format(time.RFC3339), "has_cursor": qy.Cursor != "", "returned": len(page.Items)}) //nolint:errchkjson // kiểu cố định
	ins := store.InsertAuditLogParams{ActorID: &admin, Entity: "audit_log", EntityID: admin.String(), Action: "VIEW_LIST", After: after}
	if traceID != "" {
		ins.TraceID = &traceID
	}
	if _, err := q.InsertAuditLog(ctx, ins); err != nil {
		return httpx.Page[Row]{}, fmt.Errorf("audit: ghi VIEW_LIST: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return httpx.Page[Row]{}, fmt.Errorf("audit: chốt giao dịch: %w", err)
	}
	return page, nil
}
