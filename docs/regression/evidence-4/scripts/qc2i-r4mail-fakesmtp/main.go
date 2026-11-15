// SMTP giả của QC2-I (r4mail): chế độ đọc từ tệp $CTL/mode mỗi kết nối — ok | 550 | 451 | drop.
// Mỗi kết nối ghi $CTL/log: "<RFC3339Nano> conn=<n> mode=<m> <sự kiện>". Chỉ stdlib, dựng: GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o fakesmtp main.go
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	ctl = os.Getenv("CTL")
	mu  sync.Mutex
)

func logf(format string, a ...any) {
	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(ctl+"/log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339Nano), fmt.Sprintf(format, a...))
}

func mode() string {
	b, err := os.ReadFile(ctl + "/mode")
	if err != nil {
		return "ok"
	}
	return strings.TrimSpace(string(b))
}

func main() {
	l, err := net.Listen("tcp", ":1025")
	if err != nil {
		panic(err)
	}
	n := 0
	for {
		c, err := l.Accept()
		if err != nil {
			continue
		}
		n++
		go serve(c, n, mode())
	}
}

func serve(c net.Conn, n int, m string) {
	defer c.Close()
	if m == "drop" {
		logf("conn=%d mode=%s DROP", n, m)
		return
	}
	r := bufio.NewReader(c)
	w := func(s string) { fmt.Fprintf(c, "%s\r\n", s) }
	w("220 fake ESMTP")
	data := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if data {
			if line == "." {
				data = false
				logf("conn=%d mode=%s ACCEPTED", n, m)
				w("250 2.0.0 queued")
			}
			continue
		}
		u := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(u, "EHLO"), strings.HasPrefix(u, "HELO"):
			w("250 fake")
		case strings.HasPrefix(u, "MAIL FROM"):
			w("250 2.1.0 ok")
		case strings.HasPrefix(u, "RCPT TO"):
			switch m {
			case "550":
				logf("conn=%d mode=%s RCPT->550", n, m)
				w("550 5.1.1 mailbox unavailable")
			case "451":
				logf("conn=%d mode=%s RCPT->451", n, m)
				w("451 4.3.0 try again later")
			default:
				w("250 2.1.5 ok")
			}
		case strings.HasPrefix(u, "DATA"):
			data = true
			w("354 go")
		case strings.HasPrefix(u, "RSET"):
			w("250 ok")
		case strings.HasPrefix(u, "QUIT"):
			w("221 bye")
			return
		default:
			w("250 ok")
		}
	}
}
