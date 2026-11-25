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
func (s *Service) name(ctx context.Context, pl *Payload, piiMasked int) {
	n := min(len(pl.Topics), TopRanked)
	if n == 0 || s.LLM == nil {
		return
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		if len(pl.Topics[i].samples) == 0 { // DEF-GP10-08: không câu mẫu thì không có gì để đặt tên (LLM bịa «Nhóm chưa có câu hỏi»): tên «Chủ đề N», không khuyến nghị
			pl.Topics[i].Name, pl.Topics[i].Recommendation = fmt.Sprintf("Chủ đề %d", pl.Topics[i].Rank), ""
			continue
		}
		fmt.Fprintf(&b, "Nhóm %d (từ khoá: %s):\n", pl.Topics[i].Rank, strings.Join(pl.Topics[i].Keywords, ", "))
		for _, t := range pl.Topics[i].samples {
			fmt.Fprintf(&b, "- %s\n", t)
		}
	}
	if b.Len() == 0 { // mọi chủ đề đều không có câu mẫu: không gọi LLM
		return
	}
	lane := llm.LaneBatch
	raw, err := s.LLM.Structured(ctx, llm.Request{Task: llm.TaskInsight, Lane: &lane, Shareable: false, PIIMaskedCount: piiMasked, Messages: []llm.Message{{Role: "system", Content: nameSystem}, {Role: "user", Content: b.String()}}}, json.RawMessage(nameSchema))
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
		if o.ID < 1 || o.ID > n || name == "" || len(pl.Topics[o.ID-1].samples) == 0 {
			continue
		}
		t := &pl.Topics[o.ID-1]
		rec := strings.TrimSpace(o.Recommendation)
		t.Name, t.Recommendation, t.NamedBy = trim(name), trim(rec), namedByLLM
	}
}
