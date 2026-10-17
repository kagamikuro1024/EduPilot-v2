package activity_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// US-P5-03 AC8 — LUẬT CỨNG (CLAUDE.md, TLR-2): `internal/grade` không bao giờ dùng thời gian học. Ba lớp kiểm; XANH khi gói `grade` chưa tồn tại (P6 chưa làm), ĐỎ ngay khi P6 vi phạm.

var forbiddenTokens = []string{"study_time_daily", "student_risk", "heartbeat", "v_student_360"} //nolint:gochecknoglobals // danh sách cố định

// queryMethods trả tên mọi phương thức `(*Queries)` sinh ra trong các tệp sqlc đã cho (sqlc gom mọi truy vấn vào một gói `store`, nên phải đọc tập tên từ chính tệp sinh ra).
func queryMethods(t *testing.T, files ...string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, f := range files {
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, 0)
		require.NoError(t, err, f)
		for _, d := range af.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv != nil && ast.IsExported(fd.Name.Name) {
				out[fd.Name.Name] = true
			}
		}
	}
	require.NotEmpty(t, out, "không đọc được tên truy vấn từ %v — bộ quét hỏng?", files)
	return out
}

// scanGoDir quét mọi .go dưới dir (nếu có): lời gọi `x.Method(...)` với Method ∈ banned và chuỗi chứa token cấm. dir không tồn tại → không vi phạm.
func scanGoDir(t *testing.T, dir string, banned map[string]bool) []string {
	t.Helper()
	var bad []string
	if _, err := os.Stat(dir); err != nil {
		return nil
	}
	require.NoError(t, filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		fs := token.NewFileSet()
		af, perr := parser.ParseFile(fs, path, nil, 0)
		require.NoError(t, perr, path)
		ast.Inspect(af, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if banned[x.Sel.Name] {
					bad = append(bad, path+": gọi "+x.Sel.Name)
				}
			case *ast.BasicLit:
				if x.Kind == token.STRING {
					for _, tok := range forbiddenTokens {
						if strings.Contains(strings.ToLower(x.Value), tok) {
							bad = append(bad, path+": chuỗi chứa "+tok)
						}
					}
				}
			}
			return true
		})
		return nil
	}))
	return bad
}

func TestGradeNeverImportsActivity(t *testing.T) {
	t.Parallel()
	// (a) phụ thuộc BẮC CẦU: bắt cả `grade` import gói trung gian (today, agent…) mà gói đó import crm / activity.
	cmd := exec.Command("go", "list", "-deps", "-tags", "testroutes", "./internal/grade/...")
	cmd.Dir = "../.."
	raw, _ := cmd.Output() // không khớp gói nào: stdout rỗng (cảnh báo ở stderr) = xanh
	for _, dep := range strings.Fields(string(raw)) {
		require.False(t, strings.HasSuffix(dep, "/internal/activity") || strings.HasSuffix(dep, "/internal/crm") ||
			strings.Contains(dep, "/internal/activity/") || strings.Contains(dep, "/internal/crm/"), "internal/grade phụ thuộc (bắc cầu) %s", dep)
	}
	// (b) quét AST: không gọi truy vấn sinh từ activity.sql.go / crm.sql.go, không chuỗi SQL nhắc tới thời gian học.
	banned := queryMethods(t, "../store/activity.sql.go", "../store/crm.sql.go")
	require.Empty(t, scanGoDir(t, "../grade", banned))
}

// TestGradeQueriesNeverReadStudyTime — (b, c): truy vấn / view / hàm của sổ điểm không nhắc thời gian học.
func TestGradeQueriesNeverReadStudyTime(t *testing.T) {
	t.Parallel()
	require.Empty(t, scanGoDir(t, "../grade", queryMethods(t, "../store/activity.sql.go", "../store/crm.sql.go")))
	require.Empty(t, gradeSQLViolations(t, "../store/queries", "../../db/migrations"))
}

var reGradeObject = regexp.MustCompile(`(?is)create\s+(?:or\s+replace\s+)?(?:view|function|materialized\s+view)\s+(?:public\.)?((?:v_)?grade\w*)\b(.*?)(?:;\s*\n|\$\$\s*;)`)

func gradeSQLViolations(t *testing.T, queriesDir, migrationsDir string) []string {
	t.Helper()
	var bad []string
	qs, err := filepath.Glob(filepath.Join(queriesDir, "grade*.sql"))
	require.NoError(t, err)
	ms, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, ms, "không thấy migration — đường dẫn sai?")
	for _, f := range qs {
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, tok := range forbiddenTokens {
			if strings.Contains(strings.ToLower(string(raw)), tok) {
				bad = append(bad, f+": nhắc "+tok)
			}
		}
	}
	for _, f := range ms {
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, m := range reGradeObject.FindAllStringSubmatch(string(raw), -1) {
			for _, tok := range forbiddenTokens {
				if strings.Contains(strings.ToLower(m[2]), tok) {
					bad = append(bad, f+": "+m[1]+" nhắc "+tok)
				}
			}
		}
	}
	return bad
}

// TestGradeRuleScannersCatchViolations — hai đường lách phải làm bộ quét ĐỎ (tự thử, không cần gói grade thật).
func TestGradeRuleScannersCatchViolations(t *testing.T) {
	t.Parallel()
	banned := queryMethods(t, "../store/activity.sql.go", "../store/crm.sql.go")
	require.True(t, banned["CrmGridStudents"] && banned["ActivityStudyTimeWeeks"], "tập tên đọc từ tệp sinh ra")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package grade\nfunc f(q interface{ CrmGridStudents() }) { q.CrmGridStudents() }\n"), 0o600))
	require.NotEmpty(t, scanGoDir(t, dir, banned), "gọi truy vấn của crm.sql.go")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.go"), []byte("package grade\nconst q = `select * from study_time_daily`\n"), 0o600))
	require.GreaterOrEqual(t, len(scanGoDir(t, dir, banned)), 2, "chuỗi SQL nhắc study_time_daily")
	mig, qdir := t.TempDir(), t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(mig, "1.sql"), []byte("-- +goose Up\nCREATE VIEW v_grade_sheet AS\nSELECT * FROM student_risk;\n"), 0o600))
	require.NotEmpty(t, gradeSQLViolations(t, qdir, mig), "view của sổ điểm đọc student_risk")
	require.NoError(t, os.WriteFile(filepath.Join(qdir, "grade_sheet.sql"), []byte("-- name: X :one\nselect 1 from v_student_360;\n"), 0o600))
	require.GreaterOrEqual(t, len(gradeSQLViolations(t, qdir, mig)), 2)
	require.Empty(t, gradeSQLViolations(t, "../store/queries", "../../db/migrations"), "mã thật sạch")
}
