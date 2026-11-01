package server_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/cmd/mock-imap/server"
	"github.com/edupilot/backend-go/internal/mailin"
)

func newSrv(t *testing.T) (*server.Server, string) {
	t.Helper()
	srv, err := server.New(server.Options{Username: "nop@edupilot.test", Password: "pw"})
	require.NoError(t, err)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = srv.ServeIMAP(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return srv, ln.Addr().String()
}

func msg(id string, at time.Time) server.Message {
	return server.Message{From: "sv07@edupilot.local", Subject: "[BT03] bài làm", MessageID: id, InternalDate: at,
		Attachments: []server.Attachment{{Name: "bai.md", Base64: base64.StdEncoding.EncodeToString([]byte("# bài làm\nnội dung đủ dài để qua kiểm tra")), Type: "text/markdown"}}}
}

// TestMockImapAuthHeaders — AC1 (US-S9-SEED): mọi thư seed mang `Authentication-Results: mx.edupilot.test; dmarc=pass header.from={domain}` TRÊN CÙNG và ImapSource tin nó khi IMAP_TRUSTED_AUTHSERV_ID = mx.edupilot.test.
func TestMockImapAuthHeaders(t *testing.T) {
	srv, _ := newSrv(t)
	raw, err := srv.Raw(msg("a@x", time.Now()))
	require.NoError(t, err)
	p, err := mailin.Parse([]byte(raw), 1<<20)
	require.NoError(t, err)
	require.Equal(t, []string{"mx.edupilot.test; dmarc=pass header.from=edupilot.local"}, p.AuthResults)
	require.True(t, mailin.SenderVerified(p.AuthResults, p.From, "mx.edupilot.test"))
	require.False(t, mailin.SenderVerified(p.AuthResults, p.From, ""), "biến rỗng → không tin")
	require.Equal(t, "[BT03] bài làm", p.Subject)
	require.Len(t, p.Valid, 1)
	none := ""
	m := msg("b@x", time.Now())
	m.AuthResults = &none
	raw, _ = srv.Raw(m)
	p, _ = mailin.Parse([]byte(raw), 1<<20)
	require.Empty(t, p.AuthResults)
}

// TestMockImapAppendIdempotentAndInternalDate — thêm cùng message_id lần hai không tạo thư; INTERNALDATE đặt được; qua HTTP.
func TestMockImapAppendIdempotentAndInternalDate(t *testing.T) {
	srv, addr := newSrv(t)
	h := httptest.NewServer(srv.Handler())
	t.Cleanup(h.Close)
	at := time.Date(2026, 12, 3, 8, 30, 0, 0, time.UTC)
	post := func() map[string]any {
		b, _ := json.Marshal(msg("c@x", at))
		resp, err := http.Post(h.URL+"/__mock/append", "application/json", bytes.NewReader(b))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, 200, resp.StatusCode)
		var out map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
		return out
	}
	first, second := post(), post()
	require.Equal(t, true, first["created"])
	require.Equal(t, false, second["created"])
	require.Equal(t, first["uid"], second["uid"])
	c, err := imapclient.DialInsecure(addr, nil)
	require.NoError(t, err)
	defer func() { _ = c.Close() }()
	require.NoError(t, c.Login("nop@edupilot.test", "pw").Wait())
	_, err = c.Select("INBOX", &imap.SelectOptions{ReadOnly: true}).Wait()
	require.NoError(t, err)
	msgs, err := c.Fetch(imap.UIDSetNum(imap.UID(1)), &imap.FetchOptions{InternalDate: true}).Collect()
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.True(t, at.Equal(msgs[0].InternalDate))
	resp, err := http.Get(h.URL + "/__mock/health")
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
}
