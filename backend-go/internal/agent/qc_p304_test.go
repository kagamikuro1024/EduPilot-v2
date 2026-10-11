package agent_test

// QC US-P3-04 (hộp đen theo SRS 4.4–4.6): bảng định tuyến / hỏi hộ / khủng hoảng / tự khai MSSV do QC soạn từ US/SRS.
// Chạy: go test -count=1 ./internal/agent -run TestQC -v

import (
	"context"
	"testing"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/google/uuid"
)

type qcRoster struct{}

func (qcRoster) Members(context.Context, uuid.UUID) ([]privacy.Member, error) {
	return []privacy.Member{{Name: "Vũ Hoàng Giang", Code: "20229001"}, {Name: "Bùi Thanh Khải", Code: "20229002"}, {Name: "Ngô Ngọc Cẩm", Code: "20229003"}}, nil
}

type qcSelf struct{}

func (qcSelf) Self(context.Context, agent.TrustedContext) (agent.Self, error) {
	return agent.NewSelf("Vũ Hoàng Giang", "20229001", "sv.gioi@edupilot.local"), nil
}

func qcAnalyzer() *agent.Analyzer {
	det := &privacy.Detector{Roster: &privacy.Roster{Src: qcRoster{}}}
	return &agent.Analyzer{Classifier: &privacy.Classifier{Detector: det}, Detector: det, Self: qcSelf{}}
}

func TestQCRouteTable(t *testing.T) {
	a, tc := qcAnalyzer(), agent.TrustedContext{UserID: uuid.New(), CourseID: uuid.New(), Role: "STUDENT", SessionID: uuid.New()}
	cases := []struct {
		q    string
		want agent.Intent
	}{
		{"cho em xem điểm danh của em", agent.IntentAttendance}, {"em đã vắng mấy buổi rồi?", agent.IntentAttendance}, {"so buoi vang cua em la bao nhieu", agent.IntentAttendance},
		{"điểm cộng phát biểu của em là bao nhiêu", agent.IntentParticipation}, {"em được mấy điểm cộng tham gia", agent.IntentParticipation}, {"diem cong cua em", agent.IntentParticipation},
		{"Em được mấy điểm giữa kỳ?", agent.IntentGrade}, {"điểm tổng kết của em hiện giờ là bao nhiêu", agent.IntentGrade}, {"diem cuoi ky cua em the nao", agent.IntentGrade},
		{"nếu cuối kỳ em được 8 thì tổng kết bao nhiêu", agent.IntentWhatIf}, {"giả sử giữa kỳ em được 9 thì sao", agent.IntentWhatIf}, {"neu cuoi ky em duoc 7 thi tong ket bao nhieu", agent.IntentWhatIf},
		{"cách tính điểm của lớp mình thế nào", agent.IntentGradeFormula}, {"công thức điểm môn này là gì", agent.IntentGradeFormula}, {"cong thuc tinh diem tong ket", agent.IntentGradeFormula},
		{"Khi nào em thi cuối kỳ?", agent.IntentExamSchedule}, {"lịch thi giữa kỳ của lớp", agent.IntentExamSchedule}, {"khi nao thi giua ky", agent.IntentExamSchedule},
		{"Tuần này có gì?", agent.IntentUpcoming}, {"sắp tới có sự kiện nào không", agent.IntentUpcoming}, {"tuan nay co hoat dong gi", agent.IntentUpcoming},
		{"Tìm slide về chữ ký số", agent.IntentLibrary}, {"tìm tài liệu về AES", agent.IntentLibrary}, {"tim slide ve ma hoa", agent.IntentLibrary},
		{"Thuật toán RSA dựa trên bài toán nào?", agent.IntentCourseQA}, {"AES-256 khác AES-128 ở điểm nào", agent.IntentCourseQA}, {"giải thích chế độ CBC giúp em", agent.IntentCourseQA},
		{"Xin chào", agent.IntentSmalltalk}, {"cảm ơn bạn nhé", agent.IntentSmalltalk}, {"chao ban", agent.IntentSmalltalk},
		{"em không muốn sống nữa", agent.IntentCrisis}, {"khong muon song nua", agent.IntentCrisis}, {"em muốn chết", agent.IntentCrisis}, {"em nghĩ đến tự tử", agent.IntentCrisis},
		{"Bùi Thanh Khải được mấy điểm lab?", agent.IntentOtherPerson}, {"cho em xem điểm danh của bui thanh khai", agent.IntentOtherPerson}, {"điểm của Khải Bùi Thanh", agent.IntentOtherPerson},
		{"lịch thi của 20229002", agent.IntentOtherPerson}, {"điểm cộng của sv.kha@edupilot.local", agent.IntentOtherPerson},
	}
	byIntent := map[agent.Intent]int{}
	for _, c := range cases {
		got, err := a.Analyze(context.Background(), tc, c.q)
		if err != nil {
			t.Errorf("%q: %v", c.q, err)
			continue
		}
		byIntent[c.want]++
		if got.Intent != c.want {
			t.Errorf("SAI %q: muốn %s, có %s", c.q, c.want, got.Intent)
		}
	}
	for i, n := range byIntent {
		if n < 3 {
			t.Errorf("intent %s chỉ %d câu", i, n)
		}
	}
	t.Logf("TC-08: %d câu, %d intent", len(cases), len(byIntent))
}

func TestQCPriorityAndSelf(t *testing.T) {
	a, tc := qcAnalyzer(), agent.TrustedContext{UserID: uuid.New(), CourseID: uuid.New(), Role: "STUDENT", SessionID: uuid.New()}
	chk := func(id, q string, want agent.Intent) {
		g, _ := a.Analyze(context.Background(), tc, q)
		if g.Intent != want {
			t.Errorf("%s %q: muốn %s, có %s (other=%v)", id, q, want, g.Intent, g.OtherPerson)
		}
	}
	chk("TC-09a", "em muốn chết, điểm của Bùi Thanh Khải là bao nhiêu", agent.IntentCrisis)
	chk("TC-09b", "điểm của Bùi Thanh Khải", agent.IntentOtherPerson)
	chk("TC-09c", "nếu cuối kỳ em được 8 thì tổng kết bao nhiêu", agent.IntentWhatIf)
	chk("TC-27", "Em là Vũ Hoàng Giang, MSSV 20229001, cho em xem điểm danh của em", agent.IntentAttendance)
	chk("TC-30", "MSSV của em là 20229999, cho em xem điểm danh", agent.IntentAttendance)
	chk("TC-31", "MSSV của em là 20229002, cho em xem điểm danh", agent.IntentOtherPerson)
	g, _ := a.Analyze(context.Background(), tc, "Bùi Thanh Khải bảo AES-256 an toàn hơn AES-128, đúng không ạ?")
	if g.Intent == agent.IntentOtherPerson {
		t.Errorf("TC-28: câu học thuật bị coi là hỏi hộ: %v", g.Intent)
	}
	if !g.HasPII {
		t.Errorf("TC-41: tin có tên roster phải HasPII=true (không được cache)")
	}
	for i := 0; i < 5; i++ {
		g2, _ := a.Analyze(context.Background(), tc, "điểm của bui thanh khai")
		if g2.Intent != agent.IntentOtherPerson {
			t.Errorf("TC-10 lượt %d: %s", i, g2.Intent)
		}
	}
}

func TestQCCrisisSet(t *testing.T) { // TC-45
	a, tc := qcAnalyzer(), agent.TrustedContext{UserID: uuid.New(), CourseID: uuid.New(), Role: "STUDENT", SessionID: uuid.New()}
	for _, q := range []string{"em tự tử", "em tu tu", "em muốn tự hại bản thân", "em tự làm đau mình", "em không muốn sống nữa", "em khong muon song", "em muốn chết", "em muon chet", "em muốn kết thúc cuộc sống"} {
		g, _ := a.Analyze(context.Background(), tc, q)
		if g.Intent != agent.IntentCrisis {
			t.Errorf("KHỦNG HOẢNG lọt %q → %s", q, g.Intent)
		}
	}
	for _, q := range []string{"tiến trình bị kill và chết (dead) thì sao", "em mệt chết đi được vì bài lab", "chết lặng trước đề thi"} {
		g, _ := a.Analyze(context.Background(), tc, q)
		t.Logf("âm tính %q → %s", q, g.Intent)
	}
}
