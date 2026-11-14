package llm

import "github.com/shopspring/decimal"

// DecideGate là điểm dùng decision trong luồng (SRS FEAT-decision-model 4.1). Giá trị cũng là nhãn `llm_audit.cascade_gate`.
type DecideGate string

// Ba cổng.
const (
	GateThreads   DecideGate = "THREADS_PRECHECK"
	GateInjection DecideGate = "INJECTION_FLAG"
	GateRouting   DecideGate = "CHAT_ROUTING"
)

// Valid báo cổng có thuộc ba cổng đã định nghĩa không.
func (g DecideGate) Valid() bool { return g == GateThreads || g == GateInjection || g == GateRouting }

// Option là một nhãn của câu hỏi `choice`. Key khớp bằng id (nhà cung cấp tự sắp khoá theo bảng chữ cái); ≤ 128 ký tự [A-Za-z0-9_ -].
type Option struct{ Key, Text string }

// Question là câu hỏi của decision: kiểu Go ĐÓNG (không JSON tự do, không chỉ thị từ người dùng; Text là chữ tĩnh của hệ thống).
type Question interface {
	QuestionID() string
	isQuestion()
}

// NoulQ hỏi có / không: trả P(có).
type NoulQ struct{ ID, Text string }

// ChoiceQ hỏi chọn một trong Options (≥ 2).
type ChoiceQ struct {
	ID, Text string
	Options  []Option
}

// ScoreQ khai báo cho đủ hợp đồng; 10.5 không dùng (D60 (5)) nên Decide từ chối.
type ScoreQ struct {
	ID, Text string
	Levels   []string
}

// QuestionID trả id câu hỏi.
func (q NoulQ) QuestionID() string { return q.ID }

// QuestionID trả id câu hỏi.
func (q ChoiceQ) QuestionID() string { return q.ID }

// QuestionID trả id câu hỏi.
func (q ScoreQ) QuestionID() string { return q.ID }

func (NoulQ) isQuestion()   {}
func (ChoiceQ) isQuestion() {}
func (ScoreQ) isQuestion()  {}

// Answer là câu trả lời của một câu hỏi.
type Answer interface {
	AnswerID() string
	isAnswer()
}

// NoulA: Noul ∈ [0,01; 0,99] = P(có); Confidence = max(Noul, 1−Noul).
type NoulA struct {
	ID         string
	Noul       float64
	Confidence float64
}

// ChoiceA: Choice là nhãn đứng đầu; Confidence = xác suất của nhãn đó. Xác suất là float64: không dùng cho điểm (luật 5 chỉ áp cho điểm số).
type ChoiceA struct {
	ID            string
	Choice        string
	Probabilities map[string]float64
	Confidence    float64
}

// AnswerID trả id câu hỏi tương ứng.
func (a NoulA) AnswerID() string { return a.ID }

// AnswerID trả id câu hỏi tương ứng.
func (a ChoiceA) AnswerID() string { return a.ID }

func (NoulA) isAnswer()   {}
func (ChoiceA) isAnswer() {}

// Usage là token của lời gọi decision; TokensOut luôn 0.
type Usage struct{ TokensIn, TokensOut int }

// DecideRequest là một lời gọi decision. Input là nội dung người dùng: gateway CHE danh tính rồi cắt theo trần của cổng trước khi gửi.
type DecideRequest struct {
	Task      Task // TaskDecide (bỏ trống = TaskDecide)
	Lane      Lane // INTERACTIVE | BATCH
	Gate      DecideGate
	Input     string
	Questions []Question
}

// DecideResponse mang câu trả lời theo THỨ TỰ câu hỏi.
type DecideResponse struct {
	Answers  []Answer
	Usage    Usage
	Provider string
	Model    string
	CostEst  decimal.Decimal
}

// Noul tìm câu trả lời `noul` theo id.
func (r DecideResponse) Noul(id string) (NoulA, bool) {
	for _, a := range r.Answers {
		if n, ok := a.(NoulA); ok && n.ID == id {
			return n, true
		}
	}
	return NoulA{}, false
}

// Choice tìm câu trả lời `choice` theo id.
func (r DecideResponse) Choice(id string) (ChoiceA, bool) {
	for _, a := range r.Answers {
		if c, ok := a.(ChoiceA); ok && c.ID == id {
			return c, true
		}
	}
	return ChoiceA{}, false
}
