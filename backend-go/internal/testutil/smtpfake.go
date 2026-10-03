package testutil

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// Delivered là một thư mà FakeSMTP đã nhận trọn (qua DATA).
type Delivered struct{ Rcpt, Data string }

// FakeSMTP là máy chủ SMTP tối giản cho test lỗi mà Mailpit không mô phỏng được (550 vĩnh viễn, 451 tạm thời, mất kết nối).
// Dừng/bật lại trên CÙNG cổng bằng Stop/Start để thử "SMTP không với tới rồi có lại".
type FakeSMTP struct {
	t    testing.TB
	addr string

	mu        sync.Mutex
	ln        net.Listener
	delivered []Delivered
	rcptCode  atomic.Int32
	wg        sync.WaitGroup
}

// NewFakeSMTP dựng máy chủ trên cổng ngẫu nhiên, mặc định chấp nhận mọi người nhận (250).
func NewFakeSMTP(t testing.TB) *FakeSMTP {
	t.Helper()
	f := &FakeSMTP{t: t}
	f.rcptCode.Store(250)
	ln, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // máy chủ giả của test
	if err != nil {
		t.Fatalf("fake smtp listen: %v", err)
	}
	f.ln, f.addr = ln, ln.Addr().String()
	f.serve(ln)
	t.Cleanup(f.Stop)
	return f
}

// Addr là host:port đang (hoặc từng) lắng nghe.
func (f *FakeSMTP) Addr() (host string, port int) {
	h, p, _ := net.SplitHostPort(f.addr)
	_, _ = fmt.Sscanf(p, "%d", &port)
	return h, port
}

// SetRcptCode đặt mã trả lời cho RCPT TO (250 = nhận, 550 = vĩnh viễn, 451 = tạm thời).
func (f *FakeSMTP) SetRcptCode(code int) { f.rcptCode.Store(int32(code)) } //nolint:gosec // mã SMTP 3 chữ số

// Stop đóng cổng (kết nối mới bị từ chối). Gọi nhiều lần an toàn.
func (f *FakeSMTP) Stop() {
	f.mu.Lock()
	ln := f.ln
	f.ln = nil
	f.mu.Unlock()
	if ln != nil {
		_ = ln.Close()
	}
	f.wg.Wait()
}

// Start mở lại đúng cổng cũ sau Stop.
func (f *FakeSMTP) Start() {
	f.t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ln != nil {
		return
	}
	ln, err := net.Listen("tcp", f.addr) //nolint:noctx // máy chủ giả của test
	if err != nil {
		f.t.Fatalf("fake smtp relisten: %v", err)
	}
	f.ln = ln
	f.serve(ln)
}

// Delivered trả bản sao các thư đã nhận.
func (f *FakeSMTP) Delivered() []Delivered {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Delivered(nil), f.delivered...)
}

func (f *FakeSMTP) serve(ln net.Listener) {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go f.conn(c)
		}
	}()
}

func (f *FakeSMTP) conn(c net.Conn) {
	defer func() { _ = c.Close() }()
	r, w := bufio.NewReader(c), bufio.NewWriter(c)
	reply := func(s string) { _, _ = w.WriteString(s + "\r\n"); _ = w.Flush() }
	reply("220 fake ESMTP")
	rcpt := ""
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			reply("250 fake")
		case strings.HasPrefix(cmd, "MAIL FROM"), strings.HasPrefix(cmd, "RSET"), strings.HasPrefix(cmd, "NOOP"):
			reply("250 ok")
		case strings.HasPrefix(cmd, "RCPT TO"):
			code := int(f.rcptCode.Load())
			if code != 250 {
				reply(fmt.Sprintf("%d rejected", code))
				continue
			}
			rcpt = strings.Trim(strings.TrimSpace(line[len("RCPT TO:"):]), "<>")
			reply("250 ok")
		case strings.HasPrefix(cmd, "DATA"):
			reply("354 go")
			var b strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				b.WriteString(l)
			}
			f.mu.Lock()
			f.delivered = append(f.delivered, Delivered{Rcpt: rcpt, Data: b.String()})
			f.mu.Unlock()
			reply("250 queued")
		case strings.HasPrefix(cmd, "QUIT"):
			reply("221 bye")
			return
		default:
			reply("502 unsupported")
		}
	}
}
