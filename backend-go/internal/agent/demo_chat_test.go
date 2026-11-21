package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

// demoStream: nhà cung cấp giả với tệp bản ghi demo thật (seed/demo/fake_llm.json), đúng lời nhắc chat của agent (BuildMessages).
func demoStream(t *testing.T, ctxBlock, question string) string {
	t.Helper()
	p := fake.NewNamed(fake.NewController(fake.Settings{DemoFile: "../../../seed/demo/fake_llm.json"}), "demo", "")
	var msgs []provider.Message
	for _, m := range BuildMessages(nil, ctxBlock, question) {
		msgs = append(msgs, provider.Message{Role: m.Role, Content: m.Content})
	}
	ch, err := p.Stream(t.Context(), provider.ChatOpts{Messages: msgs})
	require.NoError(t, err)
	var b strings.Builder
	for d := range ch {
		b.WriteString(d.Text)
	}
	return b.String()
}

// TestDemoChatRecordings — DEMO_SCRIPT mục 5 (tầng 2): D1 (số liệu cá nhân) có câu dẫn không mang số — số lấy từ khối dữ liệu của sổ —, D3 (ngoài tài liệu) là lời từ chối để đường leo thang chạy.
func TestDemoChatRecordings(t *testing.T) {
	d1 := demoStream(t, "get_my_attendance.absent: 2", "Em là [[SV_1]], MSSV [[SV_2]]. Em đã nghỉ mấy buổi và được cộng bao nhiêu điểm phát biểu rồi ạ?")
	require.Contains(t, d1, "sổ của bạn")
	require.NotRegexp(t, `\d`, d1, "câu dẫn không được tự nêu số")
	require.NotContains(t, d1, "giả lập")

	d3 := demoStream(t, "[1] Quy chế\n- 2.1 Phát biểu được cộng điểm", "Thi cuối kỳ có được mang một tờ A4 ghi chú viết tay vào phòng thi không ạ?")
	require.True(t, IsRefusal(d3), d3)

	// bản ghi trích quy chế không được chặn chat: ngữ cảnh truy xuất chứa «40%», «25%» hay câu đầu quy chế vẫn phải ra câu trả lời chat, không phải JSON công thức
	for _, ctxBlock := range []string{"[1] Quy chế\n- Giữa kỳ 40%", "[1] Quy chế\n- Chuyên cần 25%", "[1] Quy chế\nĐiểm quá trình gồm ba thành phần"} {
		got := demoStream(t, ctxBlock, "Nộp bài muộn bị trừ điểm thế nào?")
		require.NotContains(t, got, `"components"`, ctxBlock)
		require.Contains(t, got, "giả lập", ctxBlock)
	}

	other := demoStream(t, "", "Hôm nay trời thế nào?")
	require.Contains(t, other, "giả lập", "câu lạ không có bản ghi: bản giả trung tính, không lỗi")
}
