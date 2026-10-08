package exam

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// Giới hạn của câu hỏi (SRS 4.1.2).
const (
	MaxTitle       = 120
	MaxTopic       = 80
	MaxStem        = 8000
	MaxExplanation = 4000
	MaxOptionBody  = 1000
	MinOptions     = 2
	MaxOptions     = 8
)

// Loại câu (khớp enum `question_type`).
const (
	TypeMCQSingle = "MCQ_SINGLE"
	TypeMCQMulti  = "MCQ_MULTI"
	TypeTrueFalse = "TRUE_FALSE"
	TypeCode      = "CODE"
)

// OptionIn là một đáp án lúc gửi lên.
type OptionIn struct {
	Body       string `json:"body"`
	PinnedLast bool   `json:"pinned_last"`
}

// QuestionIn là thân tạo / sửa câu hỏi (`correct` là CHỈ SỐ trong `options` lúc gửi).
type QuestionIn struct {
	Type        string     `json:"type" validate:"required"`
	Title       string     `json:"title" validate:"required"`
	Topic       string     `json:"topic" validate:"required"`
	Difficulty  string     `json:"difficulty"`
	Stem        string     `json:"stem" validate:"required"`
	Options     []OptionIn `json:"options"`
	Correct     []int      `json:"correct"`
	Value       *bool      `json:"value"`
	Explanation *string    `json:"explanation"`
	Version     *int       `json:"version"`
}

// Checked là câu đã qua kiểm hợp lệ: đáp án đã chuẩn hoá, id đáp án CHƯA có (cấp lúc ghi).
type Checked struct {
	Type, Title, Topic, Difficulty, Stem string
	Explanation                          *string
	Options                              []OptionIn
	CorrectIdx                           []int // chỉ số đáp án đúng (MCQ)
	Value                                *bool // TRUE_FALSE
}

var compositeRE = regexp.MustCompile(`(?i)^(tất cả|cả |không có|a và|b và|cả hai)`)

// Validate kiểm một câu theo SRS 4.1.2; trả MỌI lỗi (không dừng ở lỗi đầu) và các cảnh báo không chặn.
func Validate(in QuestionIn) (Checked, []apierr.FieldError, []apierr.FieldError) {
	var errs, warns []apierr.FieldError
	bad := func(field, code, msg string) {
		errs = append(errs, apierr.FieldError{Field: field, Code: code, Message: msg})
	}
	c := Checked{Type: in.Type, Title: strings.TrimSpace(in.Title), Topic: strings.TrimSpace(in.Topic), Difficulty: in.Difficulty, Stem: in.Stem, Explanation: in.Explanation}

	switch in.Type {
	case TypeMCQSingle, TypeMCQMulti, TypeTrueFalse, TypeCode:
	case "SHORT", "ESSAY":
		bad("type", "TYPE_NOT_SUPPORTED", "Loại câu này chưa được hỗ trợ.")
	default:
		bad("type", "INVALID_TYPE", "Loại câu hỏi không hợp lệ.")
	}
	if n := utf8.RuneCountInString(c.Title); n < 1 || n > MaxTitle {
		bad("title", "TITLE_LENGTH", "Tiêu đề dài 1 đến 120 ký tự.")
	}
	if n := utf8.RuneCountInString(c.Topic); n < 1 || n > MaxTopic {
		bad("topic", "TOPIC_LENGTH", "Chủ đề dài 1 đến 80 ký tự.")
	}
	switch c.Difficulty {
	case "":
		c.Difficulty = "MEDIUM"
	case "EASY", "MEDIUM", "HARD":
	default:
		bad("difficulty", "INVALID_DIFFICULTY", "Độ khó phải là EASY, MEDIUM hoặc HARD.")
	}
	switch n := utf8.RuneCountInString(in.Stem); {
	case n > MaxStem:
		bad("stem", "STEM_TOO_LONG", "Đề bài tối đa 8.000 ký tự.")
	case strings.TrimSpace(in.Stem) == "":
		bad("stem", "STEM_EMPTY", "Đề bài không được để trống.")
	}
	if in.Explanation != nil && utf8.RuneCountInString(*in.Explanation) > MaxExplanation {
		bad("explanation", "EXPLANATION_TOO_LONG", "Giải thích tối đa 4.000 ký tự.")
	}

	switch in.Type {
	case TypeMCQSingle, TypeMCQMulti:
		validateOptions(in, &c, bad, &warns)
	case TypeTrueFalse:
		if len(in.Options) > 0 {
			bad("options", "OPTION_COUNT", "Câu đúng–sai không có đáp án lựa chọn.")
		}
		if in.Value == nil {
			bad("value", "VALUE_REQUIRED", "Cần chọn Đúng hoặc Sai.")
		}
		c.Value = in.Value
	case TypeCode:
		if len(in.Options) > 0 {
			bad("options", "OPTION_COUNT", "Bài lập trình không có đáp án lựa chọn.")
		}
	}
	return c, errs, warns
}

func validateOptions(in QuestionIn, c *Checked, bad func(field, code, msg string), warns *[]apierr.FieldError) {
	n := len(in.Options)
	if n < MinOptions || n > MaxOptions {
		bad("options", "OPTION_COUNT", "Câu trắc nghiệm có từ 2 đến 8 đáp án.")
	}
	seen := map[string]int{}
	for i, o := range in.Options {
		body := strings.TrimSpace(o.Body)
		field := "options[" + strconv.Itoa(i) + "].body"
		if body == "" || utf8.RuneCountInString(body) > MaxOptionBody {
			bad(field, "OPTION_BODY_INVALID", "Mỗi đáp án dài 1 đến 1.000 ký tự.")
			continue
		}
		key := strings.ToLower(strings.Join(strings.Fields(body), ""))
		if first, dup := seen[key]; dup {
			bad(field, "DUPLICATE_OPTION", "Đáp án trùng với đáp án "+strconv.Itoa(first+1)+".")
		} else {
			seen[key] = i
		}
		if !o.PinnedLast && compositeRE.MatchString(body) {
			*warns = append(*warns, apierr.FieldError{Field: field, Code: "UNPINNED_COMPOSITE", Message: "Đáp án kiểu \"Tất cả các đáp án trên\" nên được ghim ở cuối để không vỡ nghĩa khi xáo trộn."})
		}
		c.Options = append(c.Options, OptionIn{Body: body, PinnedLast: o.PinnedLast})
	}
	idx := map[int]bool{}
	for _, k := range in.Correct {
		switch {
		case k < 0 || k >= n:
			bad("correct", "INVALID_OPTION_ID", "Đáp án đúng nằm ngoài danh sách đáp án.")
		case idx[k]:
			bad("correct", "DUPLICATE_OPTION", "Đáp án đúng bị chọn hai lần.")
		default:
			idx[k] = true
			c.CorrectIdx = append(c.CorrectIdx, k)
		}
	}
	switch {
	case len(in.Correct) == 0:
		bad("correct", "NO_CORRECT_OPTION", "Cần chọn ít nhất một đáp án đúng.")
	case in.Type == TypeMCQSingle && len(c.CorrectIdx) > 1:
		bad("correct", "SINGLE_MULTIPLE_CORRECT", "Câu một đáp án chỉ có đúng một đáp án đúng.")
	case in.Type == TypeMCQMulti && n >= MinOptions && len(c.CorrectIdx) >= n:
		bad("correct", "NO_CORRECT_OPTION", "Câu nhiều đáp án phải có ít nhất một đáp án sai.")
	}
}

// AnswerKey dựng `answer_key` (jsonb) của câu: MCQ lưu theo id đáp án, TRUE_FALSE lưu giá trị; CODE không có (nil).
func AnswerKey(c Checked, optionIDs []uuid.UUID) json.RawMessage {
	switch c.Type {
	case TypeMCQSingle, TypeMCQMulti:
		ids := make([]string, 0, len(c.CorrectIdx))
		for _, k := range c.CorrectIdx {
			ids = append(ids, optionIDs[k].String())
		}
		b, _ := json.Marshal(map[string]any{"option_ids": ids})
		return b
	case TypeTrueFalse:
		b, _ := json.Marshal(map[string]any{"value": *c.Value})
		return b
	}
	return nil
}

// forbiddenRole là 403 FORBIDDEN `reason="role"` (thao tác dành cho Giảng viên, không phải TA).
func forbiddenRole() *apierr.Error {
	return apierr.New(http.StatusForbidden, apierr.Forbidden).WithDetails(map[string]string{"reason": "role"})
}
