package insight

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edupilot/backend-go/internal/llm"
)

const nameSystem = `Bạn giúp giảng viên đọc nhóm câu hỏi của sinh viên. Mỗi nhóm gồm vài câu mẫu đã ẩn danh. Với MỖI nhóm trả về: "name" (tên chủ đề ngắn ≤ 8 từ, tiếng Việt), "recommendation" (một câu khuyến nghị cho giảng viên, ≤ 30 từ, không nêu tên người). Giữ nguyên "id". Không bịa số liệu.`

const nameSchema = `{"type":"object","additionalProperties":false,"required":["topics"],"properties":{"topics":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["id","name","recommendation"],"properties":{"id":{"type":"integer"},"name":{"type":"string"},"recommendation":{"type":"string"}}}}}}`

// name đặt tên + khuyến nghị cho TopRanked chủ đề đầu bằng MỘT lời gọi Structured (BATCH). Lỗi / thiếu → giữ tên tạm từ từ khoá, named_by="fallback" (AC10).
// Đầu vào chỉ là câu mẫu đã che + từ khoá; điểm và hạng không do LLM quyết.
func (s *Service) name(ctx context.Context, pl *Payload) {
	n := min(len(pl.Topics), TopRanked)
	if n == 0 || s.LLM == nil {
		return
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "Nhóm %d (từ khoá: %s):\n", pl.Topics[i].Rank, strings.Join(pl.Topics[i].Keywords, ", "))
		for _, t := range pl.Topics[i].samples {
			fmt.Fprintf(&b, "- %s\n", t)
		}
	}
	lane := llm.LaneBatch
	raw, err := s.LLM.Structured(ctx, llm.Request{Task: llm.TaskInsight, Lane: &lane, Shareable: false, Messages: []llm.Message{{Role: "system", Content: nameSystem}, {Role: "user", Content: b.String()}}}, json.RawMessage(nameSchema))
	if err != nil {
		if s.Log != nil {
			s.Log.WarnContext(ctx, "insight: đặt tên lỗi, dùng tên tạm", "err", err)
		}
		return
	}
	var out struct {
		Topics []struct {
			ID             int    `json:"id"`
			Name           string `json:"name"`
			Recommendation string `json:"recommendation"`
		} `json:"topics"`
	}
	if json.Unmarshal(raw, &out) != nil {
		return
	}
	for _, o := range out.Topics {
		name := strings.TrimSpace(o.Name)
		if o.ID < 1 || o.ID > n || name == "" {
			continue
		}
		t := &pl.Topics[o.ID-1]
		t.Name, t.Recommendation, t.NamedBy = trim(name), trim(strings.TrimSpace(o.Recommendation)), namedByLLM
	}
}
