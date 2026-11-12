package practice

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DTO sinh viên (FR-22, US-P9-01 AC5): struct RIÊNG, không bao giờ trả thẳng dòng DB. Trường lộ đáp án (`feedback`, `result.items[].key / explanation / citations`)
// chỉ được điền khi đến lúc: TOPIC — câu đã trả lời; MOCK — lượt đã nộp / hết giờ. `testdata/student_dto_allowlist.json` + TestPracticeStudentDTOAllowlist khoá danh sách thẻ JSON.

// OptionView là một phương án của câu (đã xáo theo `option_order`).
type OptionView struct {
	ID   uuid.UUID `json:"id"`
	Body string    `json:"body"`
}

// QuestionView là câu hỏi sinh viên thấy khi làm bài: KHÔNG đáp án, KHÔNG giải thích, KHÔNG trích dẫn.
type QuestionView struct {
	Type    string       `json:"type"`
	Topic   string       `json:"topic"`
	Stem    string       `json:"stem"`
	Options []OptionView `json:"options"`
}

// AnswerView là đáp án sinh viên đã lưu cho câu.
type AnswerView struct {
	Answer  json.RawMessage `json:"answer"`
	Version int             `json:"version"`
	Locked  bool            `json:"locked"`
}

// FeedbackView chỉ có khi được phép lộ (xem trên). `Correct` nil với câu tự luận (không chấm).
type FeedbackView struct {
	Correct     *bool           `json:"correct"`
	Key         json.RawMessage `json:"key"`
	Explanation *string         `json:"explanation"`
	Citations   json.RawMessage `json:"citations"`
}

// ItemView là một câu của lượt.
type ItemView struct {
	ID       uuid.UUID     `json:"id"`
	Position int           `json:"position"`
	Question QuestionView  `json:"question"`
	Answer   *AnswerView   `json:"answer"`
	Feedback *FeedbackView `json:"feedback"`
}

// TopicResult là kết quả theo chủ đề của một lượt đã nộp.
type TopicResult struct {
	Topic   string `json:"topic"`
	Correct int    `json:"correct"`
	Total   int    `json:"total"`
}

// ResultView là kết quả lượt đã đóng: `{đúng}/{tổng}` là chính; điểm thang 10 (một chữ số) là phụ, không vào sổ điểm.
type ResultView struct {
	Correct int           `json:"correct"`
	Total   int           `json:"total"`
	Score   string        `json:"score"`
	Topics  []TopicResult `json:"topics"`
	Weakest *string       `json:"weakest_topic"`
	Expired bool          `json:"expired"`
}

// AttemptView là một lượt (làm bài hoặc xem lại).
type AttemptView struct {
	ID           uuid.UUID   `json:"id"`
	Mode         string      `json:"mode"`
	Topic        *string     `json:"topic"`
	Status       string      `json:"status"`
	StartedAt    time.Time   `json:"started_at"`
	DeadlineAt   *time.Time  `json:"deadline_at"`
	ServerNow    time.Time   `json:"server_now"`
	ItemCount    int         `json:"item_count"`
	CurrentIndex int         `json:"current_index"`
	Notes        []string    `json:"notes"`
	Version      int         `json:"version"`
	Items        []ItemView  `json:"items"`
	Result       *ResultView `json:"result"`
}

// SaveView là phản hồi của `PUT …/answers/{itemId}`: TOPIC kèm phản hồi ngay; MOCK chỉ xác nhận đã lưu.
type SaveView struct {
	Version  int           `json:"version"`
	Locked   bool          `json:"locked"`
	Feedback *FeedbackView `json:"feedback"`
}

// TopicInfo là một chủ đề luyện được.
type TopicInfo struct {
	Topic     string `json:"topic"`
	Available int    `json:"available"`
}

// Recommended là khối luyện tập khuyến nghị (chủ đề + số câu).
type Recommended struct {
	Topic string `json:"topic"`
	Count int    `json:"count"`
}

// InProgressView là lượt dở ("Tiếp tục").
type InProgressView struct {
	ID           uuid.UUID  `json:"id"`
	Mode         string     `json:"mode"`
	Topic        *string    `json:"topic"`
	DeadlineAt   *time.Time `json:"deadline_at"`
	ItemCount    int        `json:"item_count"`
	CurrentIndex int        `json:"current_index"`
}

// OverviewView là `GET …/practice/overview`.
type OverviewView struct {
	Recommended *Recommended     `json:"recommended"`
	Topics      []TopicInfo      `json:"topics"`
	InProgress  []InProgressView `json:"in_progress"`
}

// HistoryRow là một hàng lịch sử.
type HistoryRow struct {
	ID          uuid.UUID  `json:"id"`
	Mode        string     `json:"mode"`
	Topic       *string    `json:"topic"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	SubmittedAt *time.Time `json:"submitted_at"`
	ItemCount   int        `json:"item_count"`
	Correct     *int       `json:"correct"`
}

// WeakTopic là một chủ đề cần ôn kèm tối đa 2 tài liệu gợi ý.
type WeakTopic struct {
	Topic     string         `json:"topic"`
	Answered  int            `json:"answered"`
	Wrong     int            `json:"wrong"`
	Documents []DocSuggested `json:"documents"`
}

// DocSuggested trỏ tới một tài liệu (từ `citations` của các câu sai).
type DocSuggested struct {
	DocumentID uuid.UUID `json:"document_id"`
	Title      string    `json:"title"`
}

// WeaknessView là `GET …/practice/weakness`: rỗng khi chưa đủ dữ liệu (< 10 câu đã làm).
type WeaknessView struct {
	Topics []WeakTopic `json:"topics"`
}
