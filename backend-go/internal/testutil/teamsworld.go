package testutil

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlanRow là một dòng của fixtures/teams-plan.csv.
type PlanRow struct {
	Account, GraphUserID, Mail, Status, Sample string
}

// TeamsWorld là một lớp có bài BT03 kênh TEAMS và đúng các sinh viên trong teams-plan.csv (trừ người ngoài lớp), email đã xác minh — đáp án "24 khớp + 1 UNMATCHED" của US-P7-11.
type TeamsWorld struct {
	Course, Teacher, Asg uuid.UUID
	Students             map[string]uuid.UUID // tài khoản (sv14, sv.gioi, …) → user id
	Plan                 []PlanRow
}

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

// MockGraphDataDir là thư mục dữ liệu của mock-graph (backend-go/testdata/mock-graph).
func MockGraphDataDir() string {
	return filepath.Join(repoRoot(), "backend-go", "testdata", "mock-graph")
}

// ReadPlan đọc teams-plan.csv.
func ReadPlan(t testing.TB) []PlanRow {
	t.Helper()
	f, err := os.Open(filepath.Join(repoRoot(), "docs", "specs", "FEAT-assignments", "fixtures", "teams-plan.csv"))
	if err != nil {
		t.Fatalf("teams-plan.csv: %v", err)
	}
	defer func() { _ = f.Close() }()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("teams-plan.csv: %v", err)
	}
	var out []PlanRow
	for _, r := range rows[1:] {
		out = append(out, PlanRow{Account: r[0], GraphUserID: r[1], Mail: r[2], Status: r[3], Sample: r[4]})
	}
	return out
}

// NewTeamsWorld dựng lớp + bài (teams_class_id / teams_assignment_id lấy từ meta.json của mock-graph).
func NewTeamsWorld(t testing.TB, pool *pgxpool.Pool) *TeamsWorld {
	t.Helper()
	ctx := context.Background()
	one := func(dst any, sql string, args ...any) {
		t.Helper()
		if err := pool.QueryRow(ctx, sql, args...).Scan(dst); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	var meta struct {
		ClassID      string `json:"class_id"`
		AssignmentID string `json:"assignment_id"`
	}
	raw, err := os.ReadFile(filepath.Join(MockGraphDataDir(), "meta.json"))
	if err != nil {
		t.Fatalf("meta.json: %v", err)
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("meta.json: %v", err)
	}
	w := &TeamsWorld{Students: map[string]uuid.UUID{}, Plan: ReadPlan(t)}
	one(&w.Teacher, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'GV', 'TEACHER', 'ACTIVE', 'x', now()) returning id`, "gv."+uuid.NewString()[:6]+"@edupilot.local")
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	one(&w.Course, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'ANM', '2026-2027-HK1', $2, $3) returning id`, "T-"+id.String()[:8], string(jc), w.Teacher)
	exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'TEACHER', 'ACTIVE', 'ADMIN')`, w.Course, w.Teacher)
	for i, p := range w.Plan {
		if strings.HasPrefix(p.Account, "(") {
			continue // người ngoài lớp: không có tài khoản trong lớp
		}
		var u uuid.UUID
		// mỗi test một database mới nên email cố định của kế hoạch không đụng nhau; hai lớp trong cùng test dùng lại cùng tài khoản (on conflict).
		one(&u, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, $2, 'STUDENT', 'ACTIVE', 'x', now()) on conflict (email) do update set full_name = excluded.full_name returning id`, p.Mail, p.Account)
		exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ROSTER', $3)`, w.Course, u, "2022"+leftPad(i))
		w.Students[p.Account] = u
	}
	one(&w.Asg, `insert into assignments (course_id, title, due_at, submission_code, created_by, channels, teams_class_id, teams_assignment_id, published_at, rubric)
		values ($1, 'BT03', '2026-12-01T16:59:00Z', 'BT03', $2, ARRAY['IN_APP','TEAMS'], $3, $4, now(), '[]') returning id`, w.Course, w.Teacher, meta.ClassID, meta.AssignmentID)
	return w
}

func leftPad(i int) string {
	s := "0000" + string(rune('0'+i/10)) + string(rune('0'+i%10))
	return s[len(s)-4:]
}

// PutTeamsConfig ghi cấu hình Teams (client_secret mã hoá bằng `seal`, AAD "{course}:TEAMS").
func PutTeamsConfig(t testing.TB, pool *pgxpool.Pool, seal func(plain, aad []byte) ([]byte, error), course uuid.UUID, tenant, clientID, secret string, enabled bool) {
	t.Helper()
	sec, _ := json.Marshal(map[string]string{"client_secret": secret})
	enc, err := seal(sec, []byte(course.String()+":TEAMS"))
	if err != nil {
		t.Fatalf("mã hoá: %v", err)
	}
	pub, _ := json.Marshal(map[string]string{"tenant_id": tenant, "client_id": clientID})
	if _, err := pool.Exec(context.Background(), `insert into integration_configs (course_id, kind, config_public, config_enc, enabled) values ($1, 'TEAMS', $2::jsonb, $3, $4)
		on conflict (course_id, kind) do update set config_public = excluded.config_public, config_enc = excluded.config_enc, enabled = excluded.enabled`, course, string(pub), enc, enabled); err != nil {
		t.Fatalf("cấu hình Teams: %v", err)
	}
}
