// Package grading: trích công thức điểm từ quy chế môn học bằng MỘT lời gọi llm.Structured (FEAT-gradebook US-P6-03). AI chỉ soạn bản nháp
// có trích dẫn trang; bộ kiểm (internal/grade.Check) và giảng viên quyết định; không có đường mã nào xác nhận công thức ở đây.
package grading

import "encoding/json"

// numStr là số điểm / ngưỡng: LUÔN chuỗi decimal (strict json_schema của provider openai không có kiểu số linh hoạt, và luật 5 cấm float).
const numStr = `{"type":"string","pattern":"^\\d+(\\.\\d+)?$"}`

// nullStr là chuỗi tuỳ chọn: ở strict mode mọi khoá đều `required`, nên tuỳ chọn khai bằng `["string","null"]`.
const nullStr = `{"type":["string","null"]}`

const idList = `{"type":"array","items":{"type":"string"}}`

// paramsSchema: `params` là anyOf sáu nhánh (BONUS, PENALTY, ROUNDING, LETTER_SCALE, FAIL_CONDITION, MISSING_SCORE). Mỗi nhánh là object đủ `required`,
// `additionalProperties:false`; khoá trùng fixtures/extraction-*.json. Kiểu luật (`kind`) nằm ở phần tử rules; nhánh được phân biệt bằng khoá và `const`.
var paramsSchema = `{"anyOf":[` + //nolint:gochecknoglobals // schema bất biến
	`{"type":"object","additionalProperties":false,"required":["source","per_speak","cap"],"properties":{"source":{"const":"PARTICIPATION"},"per_speak":` + numStr + `,"cap":` + numStr + `}},` +
	`{"type":"object","additionalProperties":false,"required":["source","free","per"],"properties":{"source":{"const":"ABSENCE"},"free":{"type":"integer","minimum":0,"maximum":1000},"per":` + numStr + `}},` +
	`{"type":"object","additionalProperties":false,"required":["step","mode"],"properties":{"step":` + numStr + `,"mode":{"const":"HALF_UP"}}},` +
	`{"type":"object","additionalProperties":false,"required":["bands"],"properties":{"bands":{"type":"array","minItems":1,"maxItems":20,"items":{"type":"object","additionalProperties":false,"required":["min","letter"],"properties":{"min":` + numStr + `,"letter":{"type":"string","minLength":1,"maxLength":4}}}}}},` +
	`{"type":"object","additionalProperties":false,"required":["kind","value"],"properties":{"kind":{"type":"string","enum":["FINAL_BELOW","ABSENT_AT_LEAST"]},"value":` + numStr + `}},` +
	`{"type":"object","additionalProperties":false,"required":["zero_components"],"properties":{"zero_components":{"type":"array","items":{"type":"string"}}}}` +
	`]}`

// ExtractSchema là JSON schema đầu ra của `llm.Structured` (tương thích `strict: true`: mọi object có `required` = mọi khoá, `additionalProperties:false`).
var ExtractSchema = json.RawMessage(`{"type":"object","additionalProperties":false,"required":["components","rules","citations","unclear_items"],"properties":{` + //nolint:gochecknoglobals // schema bất biến
	`"components":{"type":"array","maxItems":20,"items":{"type":"object","additionalProperties":false,"required":["key","name","group","weight","kind_hint","citation_ids"],"properties":{` +
	`"key":{"type":"string","pattern":"^[a-z][a-z0-9_]{0,39}$"},"name":{"type":"string","minLength":1,"maxLength":80},"group":{"type":"string","enum":["QT","CK"]},"weight":` + numStr + `,` +
	`"kind_hint":` + nullStr + `,"citation_ids":` + idList + `}}},` +
	`"rules":{"type":"array","maxItems":30,"items":{"type":"object","additionalProperties":false,"required":["kind","applicable","params","citation_ids"],"properties":{` +
	`"kind":{"type":"string","enum":["BONUS","PENALTY","ROUNDING","LETTER_SCALE","FAIL_CONDITION","MISSING_SCORE"]},"applicable":{"type":"boolean"},"params":` + paramsSchema + `,"citation_ids":` + idList + `}}},` +
	`"citations":{"type":"array","maxItems":60,"items":{"type":"object","additionalProperties":false,"required":["id","page","quote"],"properties":{"id":{"type":"string"},"page":{"type":"integer","minimum":1},"quote":{"type":"string","minLength":1}}}},` +
	`"unclear_items":{"type":"array","maxItems":30,"items":{"type":"object","additionalProperties":false,"required":["message","target"],"properties":{"message":{"type":"string"},"target":{"type":"string"}}}}}}`)

// Extraction là đầu ra đã giải mã của LLM (cùng hình với ExtractSchema).
type Extraction struct {
	Components []ExtractedComponent `json:"components"`
	Rules      []ExtractedRule      `json:"rules"`
	Citations  []ExtractedCitation  `json:"citations"`
	Unclear    []ExtractedUnclear   `json:"unclear_items"`
}

type ExtractedComponent struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Group       string   `json:"group"`
	Weight      string   `json:"weight"`
	KindHint    *string  `json:"kind_hint"`
	CitationIDs []string `json:"citation_ids"`
}

type ExtractedRule struct {
	Kind        string          `json:"kind"`
	Applicable  bool            `json:"applicable"`
	Params      json.RawMessage `json:"params"`
	CitationIDs []string        `json:"citation_ids"`
}

type ExtractedCitation struct {
	ID    string `json:"id"`
	Page  int    `json:"page"`
	Quote string `json:"quote"`
}

type ExtractedUnclear struct {
	Message string `json:"message"`
	Target  string `json:"target"`
}
