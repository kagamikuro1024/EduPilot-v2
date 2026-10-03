package llm

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/edupilot/backend-go/internal/auth"
)

type identityKey struct{}

// Identity là người và lớp của lời gọi, dùng cho llm_audit và ngân sách theo lớp.
type Identity struct {
	UserID   *uuid.UUID
	CourseID *uuid.UUID
}

// WithIdentity gắn danh tính cho việc nền (worker không có Principal). Request HTTP không cần: lấy từ auth.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// IdentityFrom lấy danh tính từ ctx: WithIdentity trước, rồi Principal / CourseAccess của middleware.
func IdentityFrom(ctx context.Context) Identity {
	var id Identity
	if v, ok := ctx.Value(identityKey{}).(Identity); ok {
		id = v
	}
	if id.UserID == nil {
		if p, ok := auth.FromContext(ctx); ok {
			if u, err := uuid.Parse(p.Sub); err == nil {
				id.UserID = &u
			}
		}
	}
	if id.CourseID == nil {
		if c, ok := auth.CourseFromContext(ctx); ok {
			if u, err := uuid.Parse(c.CourseID); err == nil {
				id.CourseID = &u
			}
		}
	}
	return id
}

// TraceID là 32 hex của span trong ctx; rỗng nếu không có span (llm_audit.trace_id NOT NULL → người gọi dùng giá trị dự phòng).
func TraceID(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}
