package agent_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAgentHasNoRiskOrObservationAccess: agent không import internal/crm và không nhắc tới bảng rủi ro / ghi chú quan sát — tool không có đường đọc chúng (US-P5-04 AC9).
func TestAgentHasNoRiskOrObservationAccess(t *testing.T) {
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, bad := range []string{"student_risk", "student_observations", "v_student_360", "CrmRisk", "CrmStudents", "CrmAttention"} {
			require.NotContains(t, string(b), bad, f)
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, im := range af.Imports {
			require.NotContains(t, im.Path.Value, "internal/crm", f)
		}
	}
}
