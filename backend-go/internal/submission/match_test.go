package submission_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/submission"
)

var (
	idA, idB, idC, idD, idX = uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	bt03, bt04              = uuid.New(), uuid.New()
)

func roster() []submission.RosterEntry {
	return []submission.RosterEntry{
		{StudentID: idA, Email: "a@school.test", EmailVerified: true, Code: "20229001", JoinedVia: "ROSTER"},
		{StudentID: idB, Email: "b@school.test", EmailVerified: true, Code: "20229002", JoinedVia: "ROSTER"},
		{StudentID: idC, Email: "c@school.test", EmailVerified: false, Code: "20229003", JoinedVia: "ADMIN"}, // email chưa xác minh
		{StudentID: idD, Email: "d@school.test", EmailVerified: true, Code: "B20DC111", JoinedVia: "ROSTER"},
		{StudentID: idX, Email: "x@school.test", EmailVerified: true, Code: "20229002", JoinedVia: "CODE"}, // X vào bằng mã, khai MSSV của B
	}
}

func asgs() []submission.AssignmentRef {
	return []submission.AssignmentRef{{ID: bt03, Code: "BT03"}, {ID: bt04, Code: "BT04"}}
}

func ptr(u uuid.UUID) *uuid.UUID { return &u }

// TestMatchTable — bảng ca của bộ khớp (SRS 4.4.2), gồm các dòng của seed-plan.csv (sv29 thư sai mẫu, sv30 tệp không MSSV, thư từ email lạ).
func TestMatchTable(t *testing.T) {
	cases := []struct {
		name      string
		sig       submission.Signals
		asg       *uuid.UUID
		student   *uuid.UUID
		method    string
		reason    string
		candidate *uuid.UUID
	}{
		{"email đã xác minh + mã bài", submission.Signals{FromEmail: "a@school.test", Subject: "[BT03] Bài của em", SenderVerified: true}, ptr(bt03), ptr(idA), "EMAIL", "", nil},
		{"email hoa thường", submission.Signals{FromEmail: "  A@School.TEST ", Subject: "bt03", SenderVerified: true}, ptr(bt03), ptr(idA), "EMAIL", "", nil},
		{"mã bài đứng riêng trong tên tệp", submission.Signals{FromEmail: "a@school.test", FileName: "nop_bt04_final.pdf", SenderVerified: true}, ptr(bt04), ptr(idA), "EMAIL", "", nil},
		{"MSSV trong tên tệp (seed sv05)", submission.Signals{Subject: "[BT03]", FileName: "20229001-nguyen.pdf", SenderVerified: true}, ptr(bt03), ptr(idA), "CODE", "", nil},
		{"MSSV có đuôi .0", submission.Signals{Subject: "[BT03] 20229001.0", SenderVerified: true}, ptr(bt03), ptr(idA), "CODE", "", nil},
		{"MSSV chữ thường", submission.Signals{Subject: "[BT03] b20dc111", SenderVerified: true}, ptr(bt03), ptr(idD), "CODE", "", nil},
		{"email + MSSV cùng sinh viên", submission.Signals{FromEmail: "b@school.test", Subject: "[BT03] 20229002", SenderVerified: true}, ptr(bt03), ptr(idB), "EMAIL", "", nil},
		{"email và MSSV mâu thuẫn", submission.Signals{FromEmail: "a@school.test", Subject: "[BT03] 20229002", SenderVerified: true}, ptr(bt03), nil, "", "EMAIL_CODE_CONFLICT", ptr(idA)},
		{"email chưa xác minh không dùng", submission.Signals{FromEmail: "c@school.test", Subject: "[BT03]", SenderVerified: true}, ptr(bt03), nil, "", "NO_STUDENT_SIGNAL", nil},
		{"email chưa xác minh nhưng có MSSV", submission.Signals{FromEmail: "c@school.test", Subject: "[BT03] 20229003", SenderVerified: true}, ptr(bt03), ptr(idC), "CODE", "", nil},
		{"thư từ email lạ, không MSSV (seed sv29)", submission.Signals{FromEmail: "em.ca.nhan@gmail.test", Subject: "Bài tập 3 của em", SenderVerified: true}, nil, nil, "", "NO_ASSIGNMENT_CODE", nil},
		{"thư từ email lạ, đúng mã bài, không MSSV", submission.Signals{FromEmail: "la@gmail.test", Subject: "[BT03]", SenderVerified: true}, ptr(bt03), nil, "", "NO_STUDENT_SIGNAL", nil},
		{"tệp ZIP không MSSV (seed sv30)", submission.Signals{FileName: "bai-cua-em.pdf", SenderVerified: true, AssignmentID: ptr(bt03)}, ptr(bt03), nil, "", "NO_STUDENT_SIGNAL", nil},
		{"MSSV không thuộc lớp (seed ngoài lớp)", submission.Signals{FromEmail: "la@gmail.test", Subject: "[BT03] 20250001", SenderVerified: true}, ptr(bt03), nil, "", "STUDENT_NOT_IN_COURSE", nil},
		{"MSSV tự khai của X (dòng CODE) không khớp X", submission.Signals{Subject: "[BT03] 20229002", SenderVerified: true}, ptr(bt03), ptr(idB), "CODE", "", nil},
		{"hai bài trong tiêu đề", submission.Signals{FromEmail: "a@school.test", Subject: "[BT03] [BT04]", SenderVerified: true}, nil, nil, "", "AMBIGUOUS_ASSIGNMENT", ptr(idA)},
		{"mã bài ở tên tệp khi tiêu đề không có", submission.Signals{FromEmail: "a@school.test", Subject: "bài nộp", FileName: "BT03.docx", SenderVerified: true}, ptr(bt03), ptr(idA), "EMAIL", "", nil},
		{"nguồn gọi theo bài bỏ qua bước chọn bài", submission.Signals{FromEmail: "a@school.test", AssignmentID: ptr(bt04), SenderVerified: true}, ptr(bt04), ptr(idA), "EMAIL", "", nil},
		{"thư chưa xác thực người gửi", submission.Signals{FromEmail: "a@school.test", Subject: "[BT03]", SenderVerified: false}, nil, nil, "", "SENDER_UNVERIFIED", nil},
		{"từ thường không bị nhầm là MSSV", submission.Signals{FromEmail: "la@gmail.test", Subject: "[BT03] Nguyen Document", SenderVerified: true}, ptr(bt03), nil, "", "NO_STUDENT_SIGNAL", nil},
		{"MSSV nằm ở tiêu đề, email ở người khác", submission.Signals{FromEmail: "d@school.test", Subject: "[BT03] 20229001", SenderVerified: true}, ptr(bt03), nil, "", "EMAIL_CODE_CONFLICT", ptr(idD)},
	}
	for _, c := range cases {
		v := submission.Match(c.sig, roster(), asgs())
		require.Equal(t, c.asg, v.AssignmentID, c.name)
		require.Equal(t, c.student, v.StudentID, c.name)
		require.Equal(t, c.method, v.Method, c.name)
		require.Equal(t, c.reason, v.Reason, c.name)
		require.Equal(t, c.candidate, v.Candidate, c.name)
	}
}

func TestMatchEmailVerifiedOnly(t *testing.T) {
	sig := submission.Signals{FromEmail: "c@school.test", Subject: "[BT03]", SenderVerified: true}
	require.False(t, submission.Match(sig, roster(), asgs()).Matched(), "email chưa xác minh không phải tín hiệu")
	r := roster()
	r[2].EmailVerified = true
	require.Equal(t, ptr(idC), submission.Match(sig, r, asgs()).StudentID)
}

func TestMatchCodeFromRoster(t *testing.T) {
	for _, via := range []string{"ROSTER", "ADMIN"} {
		r := []submission.RosterEntry{{StudentID: idA, Code: "20229001", JoinedVia: via}}
		v := submission.Match(submission.Signals{Subject: "[BT03] 20229001", SenderVerified: true}, r, asgs())
		require.Equal(t, ptr(idA), v.StudentID, via)
	}
}

func TestMatchCodeJoinedRowNotVerified(t *testing.T) {
	r := []submission.RosterEntry{{StudentID: idX, Code: "20229777", JoinedVia: "CODE"}}
	v := submission.Match(submission.Signals{Subject: "[BT03] 20229777", SenderVerified: true}, r, asgs())
	require.False(t, v.Matched())
	require.Equal(t, "CODE_NOT_VERIFIED", v.Reason)
}

func TestMatchConflictUnmatched(t *testing.T) {
	v := submission.Match(submission.Signals{FromEmail: "a@school.test", Subject: "[BT03] 20229002", SenderVerified: true}, roster(), asgs())
	require.Equal(t, "EMAIL_CODE_CONFLICT", v.Reason)
	require.False(t, v.Matched())
	require.Equal(t, ptr(idA), v.Candidate, "gợi ý theo email")
}

func TestMatchAmbiguous(t *testing.T) {
	r := []submission.RosterEntry{{StudentID: idA, Code: "20229001", JoinedVia: "ROSTER"}, {StudentID: idB, Code: "20229001", JoinedVia: "ADMIN"}}
	v := submission.Match(submission.Signals{Subject: "[BT03] 20229001", SenderVerified: true}, r, asgs())
	require.Equal(t, "AMBIGUOUS_STUDENT_CODE", v.Reason)
	// hai bài cùng mã (không thể do UNIQUE, nhưng bộ khớp vẫn an toàn)
	dup := []submission.AssignmentRef{{ID: bt03, Code: "BT03"}, {ID: bt04, Code: "bt03"}}
	require.Equal(t, "AMBIGUOUS_ASSIGNMENT", submission.Match(submission.Signals{FromEmail: "a@school.test", Subject: "[BT03]", SenderVerified: true}, roster(), dup).Reason)
}

// TestSelfDeclaredCodeNeverMatches / TestCodeJoinedXWithYCodeNeverMatchesX — kịch bản TLR-1: X vào lớp bằng mã, khai MSSV của Y (dòng CODE) → không bao giờ khớp X;
// nếu Y có dòng ROSTER thì khớp Y.
func TestSelfDeclaredCodeNeverMatches(t *testing.T) {
	r := []submission.RosterEntry{{StudentID: idX, Email: "x@school.test", EmailVerified: false, Code: "20229002", JoinedVia: "CODE"}}
	v := submission.Match(submission.Signals{FromEmail: "sv.lech@school.test", Subject: "[BT03] 20229002", SenderVerified: true}, r, asgs())
	require.False(t, v.Matched())
	require.Equal(t, "CODE_NOT_VERIFIED", v.Reason)
}

func TestCodeJoinedXWithYCodeNeverMatchesX(t *testing.T) {
	v := submission.Match(submission.Signals{Subject: "[BT03]", FileName: "20229002.pdf", SenderVerified: true}, roster(), asgs())
	require.Equal(t, ptr(idB), v.StudentID, "khớp Y (ROSTER), không bao giờ X (CODE)")
	require.NotEqual(t, ptr(idX), v.StudentID)
}
