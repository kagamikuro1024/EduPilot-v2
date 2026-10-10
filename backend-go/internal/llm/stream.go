package llm

import (
	"context"
	"errors"
	"strings"

	"github.com/edupilot/backend-go/internal/llm/provider"
)

type opened struct {
	ch     <-chan provider.Delta
	first  provider.Delta
	cancel context.CancelFunc
}

// Stream sinh văn bản và phát từng mẩu. Lỗi TRƯỚC token đầu → thử lại / chuyển dự phòng như Chat; sau token đầu mà nhà cung cấp chết
// → phần tử cuối mang Err (người gọi gửi `event: error` rồi đóng — SRS 3.1). ctx huỷ → kênh đóng, lời gọi nhà cung cấp dừng, audit `cancelled`.
func (g *Gateway) Stream(ctx context.Context, r Request) (<-chan Chunk, error) {
	c, err := g.begin(ctx, r.Task, r.Lane, r.PIIMaskedCount)
	if err != nil {
		return nil, err
	}
	if err := validRequest(r); err != nil {
		c.row.ErrorKind = string(provider.KindBadRequest)
		c.finish(nil, err)
		return nil, err
	}
	rm, mk, err := g.maskRequest(c, r)
	if err != nil {
		return nil, err
	}
	ch, err := runChain(c,
		func(t Target) int { return estTokens(c.opts(t, rm, nil)) },
		func(ctx context.Context, t Target) (opened, int, error) {
			sctx, cancel := context.WithCancel(ctx)
			d, err := t.P.Stream(sctx, c.opts(t, rm, nil))
			if err != nil {
				cancel()
				return opened{}, 0, err
			}
			select {
			case first, ok := <-d:
				if !ok {
					cancel()
					return opened{}, 0, &provider.Error{Kind: provider.KindBadResponse}
				}
				if first.Err != nil {
					cancel()
					return opened{}, 0, first.Err
				}
				return opened{ch: d, first: first, cancel: cancel}, 0, nil
			case <-ctx.Done():
				cancel()
				return opened{}, 0, ctx.Err()
			}
		})
	if errors.Is(err, ErrAllProvidersFailed) {
		if c.lane != LaneInteractive {
			err = &ErrUnavailable{Reason: ReasonAllFailed}
		} else { // chưa phát byte nào: trả câu suy giảm trích nguyên văn
			resp := degraded(r)
			c.finish(&resp, nil)
			out := make(chan Chunk, 2)
			out <- Chunk{Text: resp.Text}
			out <- Chunk{Done: true, Response: &resp}
			close(out)
			return out, nil
		}
	}
	if err != nil {
		c.finish(nil, err)
		return nil, err
	}
	out := make(chan Chunk, 16)
	go g.forward(c.ctx, c, rm, mk, ch, out) //nolint:contextcheck // c.ctx là con của ctx người gọi (begin: WithDeadline)
	return out, nil
}

func (g *Gateway) forward(ctx context.Context, c *call, r Request, mk *masked, ch chosen[opened], out chan<- Chunk) {
	defer close(out)
	defer ch.val.cancel()
	var text strings.Builder
	var usage *provider.Result
	var streamErr error
	un := mk.stream() // nil khi không che

	emit := func(x Chunk) bool {
		select {
		case out <- x:
			return true
		case <-c.ctx.Done():
			return false
		}
	}
	consume := func(d provider.Delta) bool { // false = dừng
		switch {
		case d.Err != nil:
			streamErr = d.Err
			return false
		case d.Usage != nil:
			usage = d.Usage
		case d.Text != "":
			text.WriteString(d.Text)
			if un == nil {
				return emit(Chunk{Text: d.Text})
			}
			if t := un.Write(d.Text); t != "" { // người gọi chỉ nhận chữ đã khôi phục; phần có thể là nửa placeholder được giữ lại
				return emit(Chunk{Text: t})
			}
			return true
		}
		return true
	}
	ok := consume(ch.val.first)
	for ok {
		select {
		case d, more := <-ch.val.ch:
			if !more {
				ok = false
				continue
			}
			ok = consume(d)
		case <-c.ctx.Done():
			ok = false
		}
	}

	in, outTok := estTokensIn(c, r), (text.Len()+3)/4
	if usage != nil {
		in, outTok = usage.TokensIn, usage.TokensOut
	}
	switch {
	case streamErr != nil:
		kind, _ := c.kindOf(streamErr)
		c.row.ErrorKind = string(kind)
		if kind.CountsToBreaker() && !c.ownDeadlineExpired() {
			g.gate.BreakerReport(ctx, ch.target.ProviderID, kind)
		}
		c.settle(ctx, ch, in, outTok)
		if un != nil {
			if t := un.Flush(); t != "" {
				emitFinal(out, Chunk{Text: t})
			}
		}
		emitFinal(out, Chunk{Err: ErrStream})
		c.finish(nil, ErrStream)
	case c.ctx.Err() != nil:
		err := c.ctxErr(c.ctx.Err())
		c.settle(ctx, ch, in, outTok) // token đã sinh vẫn tính ngân sách
		if errors.Is(err, ErrDeadline) {
			emitFinal(out, Chunk{Err: ErrDeadline})
		}
		c.finish(nil, err)
	default:
		resp := c.response(ch.target, ch.idx, mk.unmask(text.String()), in, outTok, ch.permit)
		if un != nil { // Chunk.Done đi sau Flush()
			if t := un.Flush(); t != "" {
				emit(Chunk{Text: t})
			}
		}
		emitFinal(out, Chunk{Done: true, Response: &resp})
		c.finish(&resp, nil)
	}
}

// settle trả chỗ và tính chi phí token đã sinh khi luồng kết thúc không trọn vẹn; ghi token vào dòng audit.
func (c *call) settle(ctx context.Context, ch chosen[opened], in, out int) {
	c.row.TokensIn, c.row.TokensOut = in, out
	ch.permit.Done(in + out)
	c.g.gate.BudgetCharge(context.WithoutCancel(ctx), c.row.CourseID, costOf(ch.target, in, out))
}

// emitFinal gửi phần tử cuối không chặn nếu người nhận đã đi (kênh có đệm).
func emitFinal(out chan<- Chunk, x Chunk) {
	select {
	case out <- x:
	default:
	}
}

func estTokensIn(c *call, r Request) int {
	n := 0
	for _, m := range r.Messages {
		n += len(m.Content)
	}
	return (n + 3) / 4
}
