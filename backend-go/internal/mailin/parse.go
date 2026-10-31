// Package mailin là nguồn nhận bài qua thư (ImapSource, US-P7-05): cấu hình IMAP theo lớp, poller, phân tích một thư và đưa vào đường ống chung `submission.Ingest`.
package mailin

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"

	"github.com/edupilot/backend-go/internal/submission"
)

// Attach là một đính kèm hợp lệ về đuôi và cỡ (nội dung chưa được sniff).
type Attach struct {
	Name string
	Data []byte
	Kind submission.Kind
}

// Parsed là phần của một thư mà ImapSource cần.
type Parsed struct {
	From        string // địa chỉ (chữ thường), rỗng nếu không đọc được
	Subject     string
	MessageID   string   // `Message-ID` (rỗng nếu thiếu): khử trùng khi UIDVALIDITY đổi
	AutoReply   bool     // `Auto-Submitted` có và ≠ `no`
	AuthResults []string // giá trị các header `Authentication-Results`, TRÊN CÙNG trước
	Valid       []Attach // đính kèm đúng đuôi (PDF / DOCX / TXT / MD) và ≤ max
	Rejected    int      // đính kèm sai loại hoặc quá cỡ
	Attachments int      // tổng đính kèm (Valid + Rejected)
}

// ErrParse: MIME hỏng (thư bị bỏ qua: FAILED / PARSE_ERROR, đi tiếp thư sau).
var ErrParse = errors.New("mailin: không đọc được thư")

// Parse đọc một thư thô. `max` là cỡ tối đa của một đính kèm.
func Parse(raw []byte, max int64) (Parsed, error) {
	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil && !message.IsUnknownCharset(err) && !message.IsUnknownEncoding(err) {
		return Parsed{}, fmt.Errorf("%w: %w", ErrParse, err)
	}
	var p Parsed
	h := mr.Header
	if addrs, aerr := h.AddressList("From"); aerr == nil && len(addrs) == 1 {
		p.From = strings.ToLower(strings.TrimSpace(addrs[0].Address))
	}
	p.Subject, _ = h.Subject()
	p.MessageID = strings.TrimSpace(h.Get("Message-Id"))
	if v := strings.TrimSpace(h.Get("Auto-Submitted")); v != "" && !strings.EqualFold(v, "no") {
		p.AutoReply = true
	}
	fields := h.Fields()
	for fields.Next() {
		if strings.EqualFold(fields.Key(), "Authentication-Results") {
			v, _ := fields.Text()
			p.AuthResults = append(p.AuthResults, v)
		}
	}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return p, nil
		}
		if err != nil && !message.IsUnknownCharset(err) && !message.IsUnknownEncoding(err) {
			return Parsed{}, fmt.Errorf("%w: %w", ErrParse, err)
		}
		name := nameOf(part.Header)
		if name == "" {
			continue // thân thư (chữ / HTML), không phải đính kèm
		}
		p.Attachments++
		data, rerr := io.ReadAll(io.LimitReader(part.Body, max+1))
		if rerr != nil {
			return Parsed{}, fmt.Errorf("%w: %w", ErrParse, rerr)
		}
		kind, ok := submission.KindOf(strings.ToLower(path.Ext(name)))
		if !ok || int64(len(data)) > max || len(data) == 0 {
			p.Rejected++
			continue
		}
		p.Valid = append(p.Valid, Attach{Name: name, Data: data, Kind: kind})
	}
}

// nameOf: tên tệp của phần đính kèm (Content-Disposition `filename` hoặc Content-Type `name`); rỗng nếu là thân thư.
func nameOf(h mail.PartHeader) string {
	switch t := h.(type) {
	case *mail.AttachmentHeader:
		if n, err := t.Filename(); err == nil && n != "" {
			return n
		}
		return "attachment"
	case *mail.InlineHeader:
		if _, params, err := t.ContentDisposition(); err == nil && params["filename"] != "" {
			return params["filename"]
		}
		if _, params, err := t.ContentType(); err == nil && params["name"] != "" {
			return params["name"]
		}
	}
	return ""
}
