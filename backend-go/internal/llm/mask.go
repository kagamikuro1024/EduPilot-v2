package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/privacy"
)

// ErrMaskFailed: bước che lỗi (không có lớp trong ctx, quá hạn, panic, roster không nạp được) — KHÔNG gọi provider (SRS FEAT-private-chat-pii 4.3).
var ErrMaskFailed = privacy.ErrMaskFailed

// errNoMasker: Gateway dựng mà không có Masker và không bật NoMask.
var errNoMasker = errors.New("llm: thiếu Masker (cần cấp privacy.Masker hoặc NoMask cho test / ping)")

// maskErrorKind là giá trị llm_audit.error_kind của lỗi che (không đổi lược đồ).
const maskErrorKind = "MASK_FAILED"

// Masker là phần của privacy mà cổng LLM dùng. `*privacy.Masker` cài đặt. Đây là chỗ DUY NHẤT trong mã sản phẩm gọi Mask / Unmask / StreamUnmasker
// (kiểm bằng TestMaskOnlyInLLMGateway): internal/agent, chat, thread, ingest, rag không gọi.
type Masker interface {
	Mask(ctx context.Context, courseID uuid.UUID, s privacy.Session, texts []string) ([]string, int, error)
	Unmask(ctx context.Context, s privacy.Session, text string) string
	NewStreamUnmasker(ctx context.Context, s privacy.Session) *privacy.StreamUnmasker
}

// Validate kiểm cấu hình: gateway / worker phải cấp Masker, trừ khi NoMask (chỉ test và ping — không có nội dung người dùng).
func (o Options) Validate() error {
	if o.Masker == nil && !o.NoMask {
		return errNoMasker
	}
	return nil
}

// masked giữ trạng thái che của MỘT lời gọi để khôi phục đầu ra.
type masked struct {
	g       *Gateway
	ctx     context.Context
	sess    privacy.Session
	on      bool
	current int // số lần thay trong tin cuối (hiện tại + dữ kiện tool)
}

// maskTexts che mọi chuỗi người-dùng-có-thể-nhập của lời gọi MỘT LẦN, trước vòng dự phòng của registry: mọi nhà cung cấp trong chuỗi nhận cùng payload.
// Phạm vi roster = CourseID của Identity trong ctx; phiên = privacy.SessionFrom(ctx), không có thì phạm vi yêu cầu.
// CourseID == nil → ErrMaskFailed (nếu chỉ che bằng regex, họ tên lọt qua mà không báo lỗi).
func (g *Gateway) maskTexts(c *call, texts []string) ([]string, *masked, error) {
	m := &masked{g: g, ctx: c.ctx}
	out, err := m.mask(c, texts)
	return out, m, err
}

// mask che texts bằng CÙNG phiên của m (nhiều lần gọi trong một lời gọi logic dùng chung ánh xạ).
func (m *masked) mask(c *call, texts []string) ([]string, error) {
	g := m.g
	if g.o.NoMask || noMaskFrom(c.ctx) {
		return texts, nil
	}
	if g.o.Masker == nil {
		return nil, c.maskFail(errNoMasker)
	}
	if c.row.CourseID == nil {
		return nil, c.maskFail(fmt.Errorf("%w: lời gọi không gắn lớp (llm.WithIdentity)", ErrMaskFailed))
	}
	if m.sess == nil {
		m.sess = privacy.SessionFrom(c.ctx)
		if m.sess == nil {
			m.sess = privacy.NewSession("")
		}
	}
	out, n, err := g.o.Masker.Mask(c.ctx, *c.row.CourseID, m.sess, texts)
	if err != nil {
		if !errors.Is(err, ErrMaskFailed) {
			err = fmt.Errorf("%w: %w", ErrMaskFailed, err)
		}
		return nil, c.maskFail(err)
	}
	m.on = true
	c.row.PIIMaskedCount += n
	return out, nil
}

// maskFail ghi MỘT dòng llm_audit status='error', error_kind='MASK_FAILED' rồi trả lỗi (không gọi provider).
func (c *call) maskFail(err error) error {
	c.row.ErrorKind = maskErrorKind
	c.finish(nil, err)
	return err
}

// maskRequest che Request.Messages (system, lịch sử, người dùng, kết quả tool). Passages KHÔNG phải payload gửi provider nên không che.
func (g *Gateway) maskRequest(c *call, r Request) (Request, *masked, error) {
	m := &masked{g: g, ctx: c.ctx}
	out := make([]string, 0, len(r.Messages))
	if n := len(r.Messages); n > 0 {
		texts := make([]string, n)
		for i, msg := range r.Messages {
			texts[i] = msg.Content
		}
		hist, err := m.mask(c, texts[:n-1])
		if err != nil {
			return r, nil, err
		}
		before := c.row.PIIMaskedCount
		// tin cuối (hiện tại + dữ kiện tool) che riêng để đếm "đã ẩn N" của RIÊNG lượt này; ánh xạ ổn định theo phiên nên placeholder không đổi
		last, err := m.mask(c, texts[n-1:])
		if err != nil {
			return r, nil, err
		}
		m.current = c.row.PIIMaskedCount - before
		out = append(append(out, hist...), last...)
	}
	msgs := make([]Message, len(r.Messages))
	for i, msg := range r.Messages {
		msgs[i] = Message{Role: msg.Role, Content: out[i]}
	}
	r.Messages = msgs
	return r, m, nil
}

// unmask khôi phục một chuỗi (kèm quét placeholder sót). Không che thì trả nguyên.
func (m *masked) unmask(text string) string {
	if m == nil || !m.on {
		return text
	}
	return m.g.o.Masker.Unmask(m.ctx, m.sess, text)
}

// stream trả bộ khôi phục theo luồng; nil khi không che.
func (m *masked) stream() *privacy.StreamUnmasker {
	if m == nil || !m.on {
		return nil
	}
	return m.g.o.Masker.NewStreamUnmasker(m.ctx, m.sess)
}

// unmaskJSON khôi phục các chuỗi GIÁ TRỊ trong JSON đã qua kiểm schema (khoá giữ nguyên). Mã hoá lại bằng encoder không thoát HTML.
func (m *masked) unmaskJSON(raw json.RawMessage) (json.RawMessage, error) {
	if m == nil || !m.on {
		return raw, nil
	}
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("llm: đọc JSON để khôi phục: %w", err)
	}
	var walk func(any) any
	walk = func(x any) any {
		switch t := x.(type) {
		case string:
			return m.unmask(t)
		case []any:
			for i := range t {
				t[i] = walk(t[i])
			}
		case map[string]any:
			for k := range t {
				t[k] = walk(t[k])
			}
		}
		return x
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(walk(v)); err != nil {
		return nil, fmt.Errorf("llm: mã hoá JSON sau khôi phục: %w", err)
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}
