package activity_test

import (
	"io"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/testutil"
)

// redisProxy là proxy TCP trước Redis dùng chung: bình thường chuyển tiếp; Hang() ngắt mọi kết nối đang có và từ đó nhận kết nối mới nhưng KHÔNG BAO GIỜ trả lời —
// mô phỏng Redis treo / chết giữa chừng cho cả router thật (không đụng container dùng chung của máy).
type redisProxy struct {
	ln     net.Listener
	target string
	mu     sync.Mutex
	hung   bool
	conns  []net.Conn
}

func newRedisProxy(t *testing.T) *redisProxy {
	t.Helper()
	u, err := url.Parse(testutil.RedisURL(t))
	require.NoError(t, err)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	p := &redisProxy{ln: ln, target: u.Host}
	t.Cleanup(func() {
		_ = ln.Close()
		p.mu.Lock()
		defer p.mu.Unlock()
		for _, c := range p.conns {
			_ = c.Close()
		}
	})
	go p.serve()
	return p
}

func (p *redisProxy) URL() string { return "redis://" + p.ln.Addr().String() }

func (p *redisProxy) serve() {
	for {
		c, err := p.ln.Accept()
		if err != nil {
			return
		}
		p.mu.Lock()
		p.conns = append(p.conns, c)
		hung := p.hung
		p.mu.Unlock()
		if hung {
			continue // giữ mở, im lặng
		}
		up, err := net.DialTimeout("tcp", p.target, 2*time.Second)
		if err != nil {
			_ = c.Close()
			continue
		}
		p.mu.Lock()
		p.conns = append(p.conns, up)
		p.mu.Unlock()
		go func() { _, _ = io.Copy(up, c); _ = up.Close() }()
		go func() { _, _ = io.Copy(c, up); _ = c.Close() }()
	}
}

// Hang: từ giờ Redis "chết": ngắt kết nối cũ, kết nối mới không được trả lời.
func (p *redisProxy) Hang() {
	p.mu.Lock()
	p.hung = true
	conns := p.conns
	p.conns = nil
	p.mu.Unlock()
	for _, c := range conns {
		_ = c.Close()
	}
}
