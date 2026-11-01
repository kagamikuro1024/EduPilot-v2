// Package server là máy chủ IMAP THỬ cho seed / demo (US-S9-SEED AC1): imapserver + imapmemserver của go-imap v2 kèm một cổng HTTP điều khiển.
// Mailpit chỉ có SMTP / POP3 / HTTP nên không thay được. Chỉ chạy ở compose local / test-seed; không nhập internal/*.
package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

// Options là cấu hình: tài khoản IMAP và authserv-id đặt vào header Authentication-Results của mọi thư.
type Options struct {
	Username, Password string
	AuthServID         string // mặc định mx.edupilot.test
}

// Attachment là một đính kèm (Base64 của nội dung).
type Attachment struct {
	Name   string `json:"name"`
	Base64 string `json:"base64"`
	Type   string `json:"content_type"`
}

// Message là thân `POST /__mock/append`.
type Message struct {
	From          string       `json:"from"`
	Subject       string       `json:"subject"`
	MessageID     string       `json:"message_id"`                       // khoá chống thêm trùng (seed chạy lại)
	InternalDate  time.Time    `json:"internal_date"`                    // INTERNALDATE
	AuthResults   *string      `json:"authentication_results,omitempty"` // nil = mặc định `{authserv-id}; dmarc=pass header.from={domain của From}`; "" = không thêm
	AutoSubmitted string       `json:"auto_submitted,omitempty"`
	Attachments   []Attachment `json:"attachments"`
}

// Server giữ máy chủ IMAP trong bộ nhớ và cổng điều khiển.
type Server struct {
	opt  Options
	mem  *imapmemserver.Server
	box  *imapmemserver.User
	imap *imapserver.Server

	mu   sync.Mutex
	uids map[string]uint32 // message_id → UID
}

// New dựng máy chủ với hộp thư INBOX rỗng.
func New(opt Options) (*Server, error) {
	if opt.AuthServID == "" {
		opt.AuthServID = "mx.edupilot.test"
	}
	s := &Server{opt: opt, mem: imapmemserver.New(), uids: map[string]uint32{}}
	s.box = imapmemserver.NewUser(opt.Username, opt.Password)
	if err := s.box.Create("INBOX", nil); err != nil {
		return nil, fmt.Errorf("mock-imap: tạo INBOX: %w", err)
	}
	s.mem.AddUser(s.box)
	s.imap = imapserver.New(&imapserver.Options{NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
		return s.mem.NewSession(), nil, nil
	}, InsecureAuth: true})
	return s, nil
}

// ServeIMAP phục vụ IMAP trên ln tới khi Close.
func (s *Server) ServeIMAP(ln net.Listener) error { return s.imap.Serve(ln) }

// Close dừng máy chủ IMAP.
func (s *Server) Close() error { return s.imap.Close() }

type literal struct {
	io.Reader
	n int64
}

func (l literal) Size() int64 { return l.n }

// Raw dựng thư MIME thô: header Authentication-Results do "máy chủ thư của trường" thêm TRÊN CÙNG (SRS 4.5.4), rồi các header người gửi.
func (s *Server) Raw(m Message) (string, error) {
	if m.From == "" || m.MessageID == "" {
		return "", errors.New("thiếu from hoặc message_id")
	}
	var h strings.Builder
	switch {
	case m.AuthResults == nil:
		at := strings.LastIndex(m.From, "@")
		if at < 0 {
			return "", errors.New("from không có @")
		}
		h.WriteString("Authentication-Results: " + s.opt.AuthServID + "; dmarc=pass header.from=" + m.From[at+1:] + "\r\n")
	case *m.AuthResults != "":
		h.WriteString("Authentication-Results: " + *m.AuthResults + "\r\n")
	}
	h.WriteString("Received: from mail.edupilot.test by " + s.opt.AuthServID + "\r\n")
	h.WriteString("From: <" + m.From + ">\r\nTo: <" + s.opt.Username + ">\r\nSubject: " + encodeSubject(m.Subject) + "\r\nMessage-ID: <" + m.MessageID + ">\r\nMIME-Version: 1.0\r\n")
	if m.AutoSubmitted != "" {
		h.WriteString("Auto-Submitted: " + m.AutoSubmitted + "\r\n")
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	p, _ := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=utf-8"}})
	_, _ = p.Write([]byte("Em nộp bài ạ."))
	for _, a := range m.Attachments {
		raw, err := base64.StdEncoding.DecodeString(a.Base64)
		if err != nil {
			return "", fmt.Errorf("đính kèm %s: %w", a.Name, err)
		}
		ct := a.Type
		if ct == "" {
			ct = "application/octet-stream"
		}
		ap, _ := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {ct + `; name="` + a.Name + `"`}, "Content-Disposition": {`attachment; filename="` + a.Name + `"`}, "Content-Transfer-Encoding": {"base64"}})
		enc := base64.StdEncoding.EncodeToString(raw)
		for len(enc) > 76 {
			_, _ = ap.Write([]byte(enc[:76] + "\r\n"))
			enc = enc[76:]
		}
		_, _ = ap.Write([]byte(enc))
	}
	_ = mw.Close()
	h.WriteString("Content-Type: multipart/mixed; boundary=" + mw.Boundary() + "\r\n\r\n")
	return h.String() + body.String(), nil
}

// encodeSubject mã hoá RFC 2047 khi có ký tự ngoài ASCII (tiêu đề tiếng Việt).
func encodeSubject(s string) string {
	for _, r := range s {
		if r > 127 {
			return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
		}
	}
	return s
}

// Append thêm thư vào INBOX (INTERNALDATE = m.InternalDate); cùng message_id thêm lần hai → trả UID cũ (idempotent).
func (s *Server) Append(m Message) (uid uint32, created bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.uids[m.MessageID]; ok {
		return u, false, nil
	}
	raw, err := s.Raw(m)
	if err != nil {
		return 0, false, err
	}
	at := m.InternalDate
	if at.IsZero() {
		at = time.Now()
	}
	d, err := s.box.Append("INBOX", literal{strings.NewReader(raw), int64(len(raw))}, &imap.AppendOptions{Time: at})
	if err != nil {
		return 0, false, fmt.Errorf("mock-imap: thêm thư: %w", err)
	}
	s.uids[m.MessageID] = uint32(d.UID)
	return uint32(d.UID), true, nil
}

// Handler là cổng điều khiển HTTP: `POST /__mock/append`, `GET /__mock/health`, `GET /__mock/state`.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /__mock/health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]bool{"ok": true}) })
	mux.HandleFunc("GET /__mock/state", func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		writeJSON(w, 200, map[string]int{"messages": len(s.uids)})
	})
	mux.HandleFunc("POST /__mock/append", func(w http.ResponseWriter, r *http.Request) {
		var m Message
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&m); err != nil {
			writeJSON(w, 400, map[string]string{"error": "JSON không hợp lệ"})
			return
		}
		uid, created, err := s.Append(m)
		if err != nil {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"uid": uid, "created": created})
	})
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
