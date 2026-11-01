package textfold_test

import (
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/testutil"
	"github.com/edupilot/backend-go/internal/textfold"

	"github.com/jackc/pgx/v5"
)

const vnLower = "àáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ"

// TestFoldParitySQL — AC5: Fold của Go cho đúng kết quả `vn_fold` của SQL trên cả bảng 67 ký tự, chữ hoa, số, ký hiệu và chuỗi trộn.
func TestFoldParitySQL(t *testing.T) {
	testutil.RequireContainers(t)
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(ctx) })
	upper := ""
	for _, r := range vnLower {
		upper += string([]rune{r - 32})
	}
	cases := []string{vnLower, upper, "ĐƯỜNG Đi LÊN", "Abc 123 !@# ñ ß", "Bề mặt tấn công — SolarWinds, 40 triệu đô la.", "", "tab\there"}
	for _, c := range cases {
		var want string
		require.NoError(t, conn.QueryRow(ctx, `select vn_fold($1)`, c).Scan(&want))
		require.Equal(t, want, textfold.Fold(c), c)
	}
	require.Len(t, []rune(vnLower), 67)
}

func TestCollapseAndIndex(t *testing.T) {
	t.Parallel()
	require.Equal(t, "chi phi ung pho khoang 40 trieu do la", textfold.Collapse("  Chi phí   ứng phó\nkhoảng 40 triệu đô  la "))
	_, _, ok := textfold.Index("abc", "   ")
	require.False(t, ok)
	text := "Mở đầu.\nTác nhân là   nhóm APT29."
	s, e, ok := textfold.Index(text, "tac nhan la nhom apt29")
	require.True(t, ok)
	u := utf16.Encode([]rune(text))
	require.Equal(t, "Tác nhân là   nhóm APT29", string(utf16.Decode(u[s:e])))
}

// TestQuoteOffsetsUTF16 — AC5: offset tính theo UTF-16 code unit (ký tự ngoài BMP chiếm 2 đơn vị).
func TestQuoteOffsetsUTF16(t *testing.T) {
	t.Parallel()
	text := "😀 mở đầu 😀 Chi phí 40 triệu."
	s, e, ok := textfold.Index(text, "chi phi 40 trieu")
	require.True(t, ok)
	u := utf16.Encode([]rune(text))
	require.Equal(t, "Chi phí 40 triệu", string(utf16.Decode(u[s:e])))
	require.Equal(t, 13, s, "hai emoji = 4 đơn vị")
}

func TestWords(t *testing.T) {
	t.Parallel()
	require.Equal(t, []string{"be", "mat", "bi", "khai", "thac", "he_thong"}, textfold.Words("Bề mặt bị khai-thác, hệ_thống."))
}
