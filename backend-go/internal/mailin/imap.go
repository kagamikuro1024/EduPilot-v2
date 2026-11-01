package mailin

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Nguyên nhân thất bại của `Test kết nối` (SRS 4.5.2).
const (
	KindUnreachable = "UNREACHABLE"
	KindAuth        = "AUTH"
	KindTLS         = "TLS"
	KindMailbox     = "MAILBOX"
	KindTimeout     = "TIMEOUT"
)

// Chế độ bảo mật.
const (
	SecTLS      = "TLS"
	SecStartTLS = "STARTTLS"
	SecNone     = "NONE"
)

// Settings là cấu hình đã giải mã để mở kết nối.
type Settings struct {
	Host, Username, Password, Mailbox, Security string
	Port                                        int
}

// ConnError là lỗi kết nối đã phân loại (Kind ∈ Kind*). Err không bao giờ chứa mật khẩu.
type ConnError struct {
	Kind string
	Err  error
}

func (e *ConnError) Error() string { return e.Kind + ": " + e.Err.Error() }
func (e *ConnError) Unwrap() error { return e.Err }

// session là một kết nối đã đăng nhập và chọn hộp thư (chỉ đọc: không đổi cờ thư).
type session struct {
	c           *imapclient.Client
	uidValidity uint32
}

func (s *session) close() {
	if s == nil || s.c == nil {
		return
	}
	_ = s.c.Close()
}

// open kết nối + đăng nhập + chọn hộp thư; mọi lỗi là *ConnError. `ctx` có hạn: hết hạn → TIMEOUT.
func open(ctx context.Context, st Settings) (*session, error) {
	type result struct {
		s   *session
		err error
	}
	done := make(chan result, 1)
	var cl atomic.Pointer[imapclient.Client]
	go func() {
		s, err := openBlocking(ctx, st, &cl)
		done <- result{s, err}
	}()
	select {
	case r := <-done:
		return r.s, r.err
	case <-ctx.Done():
		if c := cl.Load(); c != nil {
			_ = c.Close()
		}
		return nil, &ConnError{Kind: KindTimeout, Err: ctx.Err()}
	}
}

func openBlocking(ctx context.Context, st Settings, out *atomic.Pointer[imapclient.Client]) (*session, error) {
	addr := net.JoinHostPort(st.Host, strconv.Itoa(st.Port))
	dl, _ := ctx.Deadline()
	opts := &imapclient.Options{Dialer: &net.Dialer{Timeout: time.Until(dl)}, TLSConfig: &tls.Config{ServerName: st.Host, MinVersion: tls.VersionTLS12}}
	var (
		c   *imapclient.Client
		err error
	)
	switch st.Security {
	case SecTLS:
		c, err = imapclient.DialTLS(addr, opts)
	case SecStartTLS:
		c, err = imapclient.DialStartTLS(addr, opts)
	default:
		c, err = imapclient.DialInsecure(addr, opts)
	}
	if err != nil {
		return nil, &ConnError{Kind: dialKind(err), Err: err}
	}
	out.Store(c)
	if err := c.Login(st.Username, st.Password).Wait(); err != nil {
		_ = c.Close()
		return nil, &ConnError{Kind: KindAuth, Err: errors.New("đăng nhập bị từ chối")} // không giữ lỗi gốc: có thể chứa tên tài khoản
	}
	data, err := c.Select(st.Mailbox, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		_ = c.Close()
		return nil, &ConnError{Kind: KindMailbox, Err: errors.New("không mở được hộp thư")}
	}
	return &session{c: c, uidValidity: data.UIDValidity}, nil
}

func dialKind(err error) string {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return KindTimeout
	}
	var (
		unknown x509.UnknownAuthorityError
		invalid x509.CertificateInvalidError
		host    x509.HostnameError
		rec     tls.RecordHeaderError
		verify  *tls.CertificateVerificationError
	)
	if errors.As(err, &unknown) || errors.As(err, &invalid) || errors.As(err, &host) || errors.As(err, &rec) || errors.As(err, &verify) {
		return KindTLS
	}
	return KindUnreachable
}

// MessageData là một thư lấy về: UID, INTERNALDATE và nội dung thô.
type MessageData struct {
	UID  uint32
	Date time.Time
	Raw  []byte
}

// uidsAfter trả UID > last (tăng dần). `UID n:*` của IMAP có thể trả thư cuối dù UID ≤ last: lọc lại ở đây.
func (s *session) uidsAfter(last uint32) ([]uint32, error) {
	var set imap.UIDSet
	set.AddRange(imap.UID(last)+1, 0)
	data, err := s.c.UIDSearch(&imap.SearchCriteria{UID: []imap.UIDSet{set}}, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("mailin: tìm thư mới: %w", err)
	}
	var out []uint32
	for _, u := range data.AllUIDs() {
		if uint32(u) > last {
			out = append(out, uint32(u))
		}
	}
	slices.Sort(out)
	return out, nil
}

// fetchOne lấy một thư (nội dung thô + INTERNALDATE). Không đổi cờ thư (Peek).
func (s *session) fetchOne(uid uint32) (MessageData, error) {
	cmd := s.c.Fetch(imap.UIDSetNum(imap.UID(uid)), &imap.FetchOptions{UID: true, InternalDate: true, BodySection: []*imap.FetchItemBodySection{{Peek: true}}})
	defer func() { _ = cmd.Close() }()
	var md MessageData
	for {
		msg := cmd.Next()
		if msg == nil {
			break
		}
		for {
			item := msg.Next()
			if item == nil {
				break
			}
			switch it := item.(type) {
			case imapclient.FetchItemDataUID:
				md.UID = uint32(it.UID)
			case imapclient.FetchItemDataInternalDate:
				md.Date = it.Time
			case imapclient.FetchItemDataBodySection:
				b, err := io.ReadAll(it.Literal)
				if err != nil {
					return md, fmt.Errorf("mailin: đọc thư: %w", err)
				}
				md.Raw = b
			}
		}
	}
	if err := cmd.Close(); err != nil {
		return md, fmt.Errorf("mailin: lấy thư: %w", err)
	}
	if md.UID != uid {
		return md, fmt.Errorf("mailin: thư UID %d không còn", uid)
	}
	return md, nil
}
