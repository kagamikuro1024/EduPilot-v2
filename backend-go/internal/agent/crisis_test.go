package agent

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// US-P4-07 (SRS FEAT-escalation-notify 4.9, 9.5): bộ luật khủng hoảng hai tầng. Mọi câu trong testdata đều BỊA (D44); test khẳng định `DetectIntent`, không gọi LLM.

func load(t *testing.T, name string) []string {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	var out []string
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

func TestCrisis20Recall(t *testing.T) {
	t.Parallel()
	pos := load(t, "crisis_20.json")
	require.Len(t, pos, 20)
	hit := 0
	for _, s := range pos {
		if DetectIntent(s) == IntentCrisis {
			hit++
		} else {
			t.Logf("bỏ sót: %q", s)
		}
	}
	t.Logf("recall %d/20", hit)
	require.GreaterOrEqual(t, hit, 19)
}

func TestCrisisNegatives17(t *testing.T) {
	t.Parallel()
	neg := load(t, "crisis_neg_17.json")
	require.Len(t, neg, 17)
	for _, s := range neg {
		require.NotEqual(t, IntentCrisis, DetectIntent(s), s)
	}
}

func TestCrisisAmbiguousNeedsFirstPerson(t *testing.T) {
	t.Parallel()
	require.Equal(t, IntentCrisis, DetectIntent("em tu tu cho xong"))
	require.Equal(t, IntentCrisis, DetectIntent("Em muốn tự tử"))
	for _, s := range []string{"em doc tu tu", "join tu hai bang", "Thầy giải thích từ từ giúp em", "Process cha chết đi thì sao", "cách cắt tay cầm"} {
		require.NotEqual(t, IntentCrisis, DetectIntent(s), s)
	}
	// dạng NFD (dấu kết hợp) không so được dấu: rơi về luật cũ, thà bắt nhầm còn hơn bỏ sót
	require.Equal(t, IntentCrisis, DetectIntent("toi nghi toi tu\u0323 tu\u031B\u0309"))
}

func TestCrisisBeatsOtherIntents(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"Cho em hỏi điểm giữa kỳ. Mà em muốn chết thật rồi", "Em vắng mấy buổi, em không muốn sống nữa", "Lịch thi khi nào? Em muốn tự tử", "Cách tính điểm ra sao, em chán sống lắm rồi"} {
		require.Equal(t, IntentCrisis, DetectIntent(s), s)
	}
}

// TestCrisisHyperboleStillSupport — AC7: lời nói quá vẫn trả câu hỗ trợ (chọn an toàn; Q10).
func TestCrisisHyperboleStillSupport(t *testing.T) {
	t.Parallel()
	require.Equal(t, IntentCrisis, DetectIntent("deadline này làm em muốn chết"))
}

func TestCrisisReplyTone(t *testing.T) {
	t.Parallel()
	for _, support := range []string{"", "Phòng Công tác sinh viên, phòng 101 nhà A1, giờ hành chính"} {
		r := CrisisText(support)
		require.Contains(t, r, "Mình rất tiếc")
		require.Contains(t, r, "giảng viên")
		require.LessOrEqual(t, utf8.RuneCountInString(r), 400)
		low := strings.ToLower(r)
		for _, bad := range []string{"không thể", "xin lỗi, tôi", "tôi chỉ có thể trả lời câu hỏi về môn học", "ngoài phạm vi"} {
			require.NotContains(t, low, bad)
		}
	}
}

func TestCrisisReplyNoAdvice(t *testing.T) {
	t.Parallel()
	low := strings.ToLower(CrisisText(""))
	for _, bad := range []string{"thuốc", "liệu pháp", "chẩn đoán", "bạn nên uống", "bạn nên dùng", "trầm cảm"} {
		require.NotContains(t, low, bad)
	}
}

func TestCrisisReplyNoJargon(t *testing.T) {
	t.Parallel()
	r := CrisisText("")
	for _, bad := range []string{"AI", "RAG", "PII", "độ tin cậy", "[[", "]]", "provider"} {
		require.NotContains(t, r, bad)
	}
}

func TestCrisisReplyUsesSupportResources(t *testing.T) {
	t.Parallel()
	const sup = "Phòng Công tác sinh viên, phòng 101 nhà A1, giờ hành chính"
	require.Contains(t, CrisisText(sup), sup)
	a := &Agent{SupportResources: sup}
	require.Contains(t, a.crisisReply(), sup)
}

func TestCrisisReplyDefaultSupport(t *testing.T) {
	t.Parallel()
	require.Contains(t, CrisisText("  "), DefaultSupport)
	require.Contains(t, (&Agent{}).crisisReply(), DefaultSupport)
}
