package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/testutil"
)

// TestRBAC_ClaimWinsOverDB: vai trò lấy từ claim, KHÔNG từ bảng `users` (04-AC6).
// Dùng DB thật để chứng minh giá trị trong DB không ảnh hưởng tới quyết định phân quyền.
func TestRBAC_ClaimWinsOverDB(t *testing.T) {
	testutil.RequireContainers(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn, err := pgx.Connect(ctx, testutil.MigratedPostgresURL(t))
	if err != nil {
		t.Fatalf("nối Postgres: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	// Bảng `users` thật của migration 00001 — chứng minh giá trị trong DB không được dùng để phân quyền.
	setRole := func(role Role) {
		t.Helper()
		if _, err := conn.Exec(ctx,
			`insert into users (id, email, full_name, role) values ($1, $2, $3, $4::user_role)
			 on conflict (id) do update set role = excluded.role`,
			testSub, "qc-claim@example.test", "QC Claim", string(role)); err != nil {
			t.Fatalf("ghi users.role=%s: %v", role, err)
		}
		var got string
		if err := conn.QueryRow(ctx, `select role from users where id = $1`, testSub).Scan(&got); err != nil {
			t.Fatalf("đọc users.role: %v", err)
		}
		if got != string(role) {
			t.Fatalf("users.role trong DB = %s, muốn %s", got, role)
		}
	}

	clk := fixedClock()
	iss := NewIssuer(testSecret, 0, clk)
	h := testRouter(NewVerifier(testSecret, clk), DenyAll{})

	tests := []struct {
		name   string
		dbRole Role
		claim  Role
		code   int
	}{
		{"DB=ADMIN, claim=STUDENT → 403", RoleAdmin, RoleStudent, http.StatusForbidden},
		{"DB=STUDENT, claim=ADMIN → 200", RoleStudent, RoleAdmin, http.StatusOK},
		{"DB=TEACHER, claim=ADMIN → 200", RoleTeacher, RoleAdmin, http.StatusOK},
		{"DB=ADMIN, claim=TA → 403", RoleAdmin, RoleTA, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRole(tt.dbRole)
			tok, err := iss.Issue(testSub, tt.claim, testEmail)
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}
			rec := do(t, h, "/rbac/admin", "Bearer "+tok)
			if rec.Code != tt.code {
				t.Fatalf("status=%d, muốn %d (thân %s)", rec.Code, tt.code, rec.Body.String())
			}
			if tt.code != http.StatusOK {
				d, _ := decodeBody(t, rec)["details"].(map[string]any)
				if d["reason"] != "role" {
					t.Fatalf("details=%v, muốn reason=role", d)
				}
				return
			}
			if body := decodeBody(t, rec); body["role"] != string(tt.claim) {
				t.Fatalf("role trả về=%v, muốn claim %s (không phải DB %s)", body["role"], tt.claim, tt.dbRole)
			}
		})
	}
}
