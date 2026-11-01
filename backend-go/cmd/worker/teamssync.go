package main

import (
	"os"
	"strconv"

	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/edupilot/backend-go/internal/submission"
	"github.com/edupilot/backend-go/internal/submission/source"
)

// newTeamsSync dựng bộ đồng bộ Teams (US-P7-11). nil khi thiếu object storage / Redis / khoá mã hoá (worker thử nghiệm): việc `teams.sync` không đăng ký.
func newTeamsSync(d Deps) *submission.TeamsSync {
	bs, ok := d.Blob.(submission.BlobAPI)
	if !ok || d.Redis == nil {
		return nil
	}
	cipher, err := crypto.ParseKey(d.Cfg.AppEncryptionKey)
	if err != nil {
		return nil
	}
	maxBytes := int64(20 << 20)
	if v, err := strconv.ParseInt(os.Getenv("ASSIGN_MAX_BYTES"), 10, 64); err == nil && v > 0 {
		maxBytes = v
	}
	return &submission.TeamsSync{Pool: d.DB, Redis: d.Redis, Blob: bs, Cipher: cipher, Jobs: jobs.NewService(d.DB), Log: d.Log, Enabled: d.Cfg.TeamsEnabled,
		Source: source.ConfigFrom(d.Cfg, maxBytes), Timeout: d.Cfg.TeamsSyncTimeout, MinInterval: d.Cfg.SyncNowMinInterval, MaxBytes: maxBytes}
}
