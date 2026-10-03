// Package auth: danh tính lấy từ claim JWT (HS256), RBAC theo vai và khung CourseAccessGuard (US-PG-04, SRS 4.4).
// Nguyên tắc: không truy DB mỗi request (FR-39) — Principal dựng hoàn toàn từ claim và chỉ đi qua context.Context.
package auth

import (
	"context"
	"log/slog"
	"time"
)

// Role là vai trò trong claim `role` (SRS 2); phân biệt hoa thường.
type Role string

// Bốn vai trò runtime của EduPilot.
const (
	RoleAdmin   Role = "ADMIN"
	RoleTeacher Role = "TEACHER"
	RoleTA      Role = "TA"
	RoleStudent Role = "STUDENT"
)

// Valid cho biết r có đúng một trong bốn vai trò không.
func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleTeacher, RoleTA, RoleStudent:
		return true
	default:
		return false
	}
}

// Principal là danh tính của một request đã xác thực; mọi trường lấy từ claim, không từ DB.
type Principal struct {
	Sub       string
	Role      Role
	Email     string
	JTI       string
	ExpiresAt time.Time
}

// LogValue là dạng an toàn để ghi log: chỉ `sub`, `role` và 8 ký tự đầu của `jti`
// (US-PG-04 AC9) — không email (PII), không token, không chữ ký.
func (p Principal) LogValue() slog.Value {
	jti := p.JTI
	if len(jti) > jtiLogLen {
		jti = jti[:jtiLogLen]
	}
	return slog.GroupValue(
		slog.String("sub", p.Sub),
		slog.String("role", string(p.Role)),
		slog.String("jti", jti),
	)
}

const jtiLogLen = 8

// CourseAccess là kết quả của CourseAccessGuard cho request hiện tại.
type CourseAccess struct {
	CourseID   string
	CourseRole Role
}

type ctxKey int

const (
	principalKey ctxKey = iota
	courseKey
)

// WithPrincipal gắn danh tính vào ctx — đường DUY NHẤT Principal đi qua hệ thống (FR-39, 04-AC12).
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// FromContext lấy danh tính; ok=false khi request chưa qua Middleware.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

// MustFromContext như FromContext nhưng panic khi thiếu — chỉ dùng trong handler chắc chắn nằm sau Middleware.
func MustFromContext(ctx context.Context) Principal {
	p, ok := FromContext(ctx)
	if !ok {
		panic("auth: context không có Principal (thiếu auth.Middleware?)")
	}
	return p
}

// WithCourseAccess gắn quyền lớp học đã giải vào ctx (CourseAccessGuard gọi).
func WithCourseAccess(ctx context.Context, a CourseAccess) context.Context {
	return context.WithValue(ctx, courseKey, a)
}

// CourseFromContext lấy quyền lớp học; ok=false khi request chưa qua CourseAccessGuard.
func CourseFromContext(ctx context.Context) (CourseAccess, bool) {
	a, ok := ctx.Value(courseKey).(CourseAccess)
	return a, ok
}
