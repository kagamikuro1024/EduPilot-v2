package mail_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/mail"
)

// US-P4-04: năm mẫu mới — đủ biến thì dựng được, thiếu biến thì ErrMissingVar (DEAD ngay), tiêu đề không có tên người.
var p404 = map[string]map[string]string{
	"escalation_answered": {"FullName": "Trần An", "ClassCode": "761987", "QuestionText": "Điểm chuyên cần?", "AnswererRole": "Giảng viên", "AnswererName": "Lê Hoa", "AnswerText": "Xem bài 3.", "ChatURL": "https://app.test/chat?session=s&m=m"},
	"outreach_message":    {"FullName": "Trần An", "ClassCode": "761987", "AnswererRole": "Trợ giảng", "AnswererName": "Lê Hoa", "AnswerText": "Em nộp bài chưa?", "ChatURL": "https://app.test/chat?session=s"},
	"ticket_reminder":     {"ClassCode": "761987", "Count": "3", "InboxURL": "https://app.test/inbox?course=c"},
	"ticket_new":          {"ClassCode": "761987", "Count": "2", "InboxURL": "https://app.test/inbox?course=c"},
	"thread_reply":        {"Event": "Có bình luận mới", "ThreadTitle": "Cảnh báo học vụ", "ThreadURL": "https://app.test/threads/t"},
}

func testTemplate(t *testing.T, name, subject string, in ...string) {
	t.Helper()
	data := p404[name]
	r, err := mail.Render(name, data)
	require.NoError(t, err)
	require.Equal(t, subject, r.Subject)
	for _, v := range in {
		require.Contains(t, r.Text, v)
		require.Contains(t, r.HTML, v)
	}
	require.Contains(t, r.Text, "Thư tự động từ hệ thống")
	for k := range data {
		d := map[string]string{}
		for kk, vv := range data {
			if kk != k {
				d[kk] = vv
			}
		}
		_, err := mail.Render(name, d)
		require.ErrorIs(t, err, mail.ErrMissingVar, "thiếu %s", k)
	}
}

func TestTemplateTicketReminder(t *testing.T) {
	testTemplate(t, "ticket_reminder", "Có câu hỏi chờ giảng viên quá 24 giờ làm việc", "761987", "3 câu hỏi", "https://app.test/inbox?course=c")
}

func TestTemplateTicketNew(t *testing.T) {
	testTemplate(t, "ticket_new", "Có câu hỏi mới cần giảng viên", "761987", "2 câu hỏi", "https://app.test/inbox?course=c")
}

func TestTemplateThreadReply(t *testing.T) {
	testTemplate(t, "thread_reply", "Câu hỏi của bạn trên Threads có trả lời mới", "Có bình luận mới", "«Cảnh báo học vụ»", "https://app.test/threads/t")
}

func TestTemplateEscalationAnswered(t *testing.T) {
	testTemplate(t, "escalation_answered", "Giảng viên đã trả lời câu hỏi của bạn", "Chào Trần An,", "Điểm chuyên cần?", "Giảng viên Lê Hoa trả lời: Xem bài 3.", "https://app.test/chat?session=s")
}

func TestTemplateSubjectsHaveNoNames(t *testing.T) {
	for name, data := range p404 {
		r, err := mail.Render(name, data)
		require.NoError(t, err)
		for _, field := range []string{"FullName", "AnswererName", "ThreadTitle"} {
			if v := data[field]; v != "" {
				require.NotContains(t, r.Subject, v, name)
			}
		}
	}
	r, err := mail.Render("thread_reply", p404["thread_reply"])
	require.NoError(t, err)
	require.NotContains(t, r.Subject, "Cảnh báo", "tiêu đề thread không đi vào tiêu đề thư")
}

func TestTemplateEscapesHTML(t *testing.T) {
	d := map[string]string{}
	for k, v := range p404["escalation_answered"] {
		d[k] = v
	}
	d["AnswerText"] = `<script>alert(1)</script>`
	r, err := mail.Render("escalation_answered", d)
	require.NoError(t, err)
	require.NotContains(t, r.HTML, "<script>")
}

func TestSenderDeterministicMessageID(t *testing.T) {
	id := uuid.MustParse("01a135d5-9014-7fd6-9943-82655e1f913f")
	require.Equal(t, "01a135d5-9014-7fd6-9943-82655e1f913f@edupilot.local", mail.MessageID(id, "EduPilot <no-reply@edupilot.local>"))
	require.Equal(t, id.String()+"@ptit.edu.vn", mail.MessageID(id, "no-reply@ptit.edu.vn"))
	require.Equal(t, mail.MessageID(id, "x@a.b"), mail.MessageID(id, "x@a.b"), "tất định")
}

func TestClipLongText(t *testing.T) {
	require.Equal(t, "abc", mail.Clip("  abc ", 5, "…"))
	require.Equal(t, "ab… Xem đầy đủ", mail.Clip("abcdef", 2, "… Xem đầy đủ"))
	require.Equal(t, "ĐĐĐ…", mail.Clip("ĐĐĐĐĐ", 3, "…"), "cắt theo ký tự, không theo byte")
}

// TestNoHTTPReadsMailOutbox — AC8: chỉ `internal/mail`, `internal/store` và `mailhttp` (đã che, không thân thư) được nhắc tới mail_outbox; route nào khác không đọc.
func TestNoHTTPReadsMailOutbox(t *testing.T) {
	var offenders []string
	root := filepath.Join("..", "httpapi")
	require.NoError(t, filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		b, _ := os.ReadFile(p)
		s := string(b)
		if strings.Contains(s, "mail_outbox") || strings.Contains(s, "LockMailOutbox") || strings.Contains(s, "GetMailOutbox") {
			offenders = append(offenders, p)
		}
		return nil
	}))
	require.Empty(t, offenders)
	b, err := os.ReadFile(filepath.Join("..", "httpapi", "mailhttp", "failed.go"))
	require.NoError(t, err)
	require.False(t, strings.Contains(string(b), ".Payload") || strings.Contains(string(b), `json:"payload"`), "danh sách thư lỗi không có payload / thân thư")
}
