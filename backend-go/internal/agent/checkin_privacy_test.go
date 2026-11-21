package agent_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
)

// US-AS-06 (P-11-43): điểm danh tự phục vụ (cờ, mức tin cậy, thiết bị, vị trí) là dữ liệu của giảng viên — tool `get_my_attendance` của sinh viên không nhận
// tham số mới và không có đường tới chúng: (1) không tham số; (2) agent không nhắc bảng / cờ / không import internal/attendance; (3) hai câu SQL của nguồn dữ liệu
// (CrmMyAttendanceSummary, CrmMyRecentAbsent) chỉ đọc dòng điểm danh chính thức, không đọc attendance_checkins / checkin_windows.
func TestAgentToolNoCheckinFlags(t *testing.T) {
	tool := agent.NewAttendanceTool(nil)
	require.Equal(t, "get_my_attendance", tool.Name())
	at := tool.ArgsType()
	require.Equal(t, reflect.Struct, at.Kind())
	require.Zero(t, at.NumField(), "tool không nhận tham số nào (kể cả student_id / include_flags)")

	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, bad := range []string{"attendance_checkins", "checkin_windows", "user_devices", "TOKEN_LATE", "DEVICE_SHARED", "GEO_OUT", "flags_cleared", "assurance"} {
			require.NotContains(t, string(b), bad, f)
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, im := range af.Imports {
			require.NotContains(t, im.Path.Value, "internal/attendance", f)
		}
	}

	sql, err := os.ReadFile(filepath.Join("..", "store", "queries", "crm.sql"))
	require.NoError(t, err)
	for _, name := range []string{"CrmMyAttendanceSummary", "CrmMyRecentAbsent"} {
		m := regexp.MustCompile(`(?s)-- name: ` + name + ` .*?(?:\n-- name: |\z)`).Find(sql)
		require.NotEmpty(t, m, name)
		for _, bad := range []string{"attendance_checkins", "checkin_windows", "user_devices", "flags"} {
			require.NotContains(t, string(m), bad, name)
		}
	}
}
