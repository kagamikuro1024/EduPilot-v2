package ingest

import (
	"strconv"
	"time"
)

// Settings là cấu hình nạp tài liệu (SRS FEAT-docs-calendar 5.6). Worker đọc riêng từ môi trường, như judge.Settings.
type Settings struct {
	DoclingURL     string        // DOCLING_URL
	MaxBytes       int64         // DOC_MAX_BYTES
	MaxPages       int           // DOC_MAX_PAGES
	ChunkChars     int           // CHUNK_CHARS
	ExtractTimeout time.Duration // INGEST_EXTRACT_TIMEOUT
	Workers        int           // INGEST_WORKERS
	ClaimIdle      time.Duration // INGEST_CLAIM_IDLE: thuê tài liệu hết hạn sau từng này kể từ nhịp gia hạn cuối
	PollInterval   time.Duration
	LeaseRenew     time.Duration
	Reclaim        time.Duration // tin treo (bận / consumer chết) được giao lại sau từng này
	Retries        int           // lần thử với lỗi tạm thời
}

// LoadSettings đọc biến môi trường; thiếu hoặc sai thì dùng mặc định.
func LoadSettings(getenv func(string) string) Settings {
	i := func(k string, def int64) int64 {
		if v, err := strconv.ParseInt(getenv(k), 10, 64); err == nil && v > 0 {
			return v
		}
		return def
	}
	d := func(k string, def time.Duration) time.Duration {
		if v, err := time.ParseDuration(getenv(k)); err == nil && v > 0 {
			return v
		}
		return def
	}
	s := Settings{
		DoclingURL:     getenv("DOCLING_URL"),
		MaxBytes:       i("DOC_MAX_BYTES", 50<<20),
		MaxPages:       int(i("DOC_MAX_PAGES", 400)),
		ChunkChars:     int(i("CHUNK_CHARS", ChunkChars)),
		ExtractTimeout: d("INGEST_EXTRACT_TIMEOUT", 25*time.Minute),
		Workers:        int(i("INGEST_WORKERS", 1)),
		PollInterval:   2 * time.Second,
		LeaseRenew:     time.Minute,
		Reclaim:        10 * time.Second,
		Retries:        3,
	}
	s.ClaimIdle = d("INGEST_CLAIM_IDLE", s.ExtractTimeout+5*time.Minute)
	if s.DoclingURL == "" {
		s.DoclingURL = "http://docling:5001"
	}
	return s
}
