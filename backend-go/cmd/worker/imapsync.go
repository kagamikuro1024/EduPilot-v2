package main

import (
	"context"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi/sse"
	"github.com/edupilot/backend-go/internal/mailin"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/edupilot/backend-go/internal/submission"
)

// newImapSyncer dựng bộ đọc thư (US-P7-05). nil khi thiếu object storage / Redis / khoá mã hoá (worker thử nghiệm): việc `imap.sync` không đăng ký, không poll.
func newImapSyncer(d Deps) *mailin.Syncer {
	bs, ok := d.Blob.(submission.BlobAPI)
	if !ok || d.Redis == nil {
		return nil
	}
	cipher, err := crypto.ParseKey(d.Cfg.AppEncryptionKey)
	if err != nil {
		return nil
	}
	from := ""
	if a, err := mail.ParseAddress(d.Cfg.MailFrom); err == nil {
		from = strings.ToLower(a.Address)
	}
	maxBytes := int64(20 << 20)
	if v, err := strconv.ParseInt(os.Getenv("ASSIGN_MAX_BYTES"), 10, 64); err == nil && v > 0 {
		maxBytes = v
	}
	pub := sse.NewPublisher(d.Redis, d.Cfg.SSEBufferMaxLen, d.Cfg.SSEBufferTTL)
	return &mailin.Syncer{Pool: d.DB, Redis: d.Redis, Cipher: cipher, Blob: bs, Notify: notify.New(d.DB, pub, d.Log), Log: d.Log,
		TrustedAuthservID: d.Cfg.IMAPTrustedAuthservID, SystemFrom: from, MaxBytes: maxBytes, DefaultPoll: time.Duration(d.Cfg.IMAPPollSeconds) * time.Second, BackoffMax: d.Cfg.IMAPBackoffMax}
}

// imapTask poll hộp thư IMAP của các lớp đã bật; một poller mỗi lớp nhờ khoá `ep:lock:imap:{course_id}`.
type imapTask struct{ y *mailin.Syncer }

func (imapTask) Name() string                    { return "imap.poller" }
func (t imapTask) Run(ctx context.Context) error { return t.y.Run(ctx, 5*time.Second) }
