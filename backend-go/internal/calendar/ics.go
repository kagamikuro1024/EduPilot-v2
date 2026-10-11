package calendar

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn feed (SRS 4.8): now−30 ngày … now+365 ngày, tối đa 2.000 sự kiện; 60 lượt / phút / IP.
const (
	feedBack    = 30 * 24 * time.Hour
	feedAhead   = 365 * 24 * time.Hour
	FeedMax     = 2000
	FeedPerMin  = 60
	foldOctets  = 75
	icsTimeFmt  = "20060102T150405Z"
	defaultSpan = time.Hour
)

// HashToken là `hex(sha256(token))` — thứ duy nhất lưu ở users.ics_token.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// IssueToken tạo / xoay token ICS của CHÍNH người gọi; trả URL (token thô chỉ hiện ở đây, một lần).
func (s *Service) IssueToken(ctx context.Context, uid uuid.UUID) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("calendar: sinh token: %w", err)
	}
	tok := base64.RawURLEncoding.EncodeToString(b)
	h := HashToken(tok)
	if _, err := store.New(s.Pool).IcsTokenSet(ctx, store.IcsTokenSetParams{TokenHash: &h, Owner: uid}); err != nil {
		return "", fmt.Errorf("calendar: lưu băm token: %w", err)
	}
	return strings.TrimRight(s.PublicURL, "/") + "/api/v1/calendar/feed.ics?token=" + tok, nil
}

// RevokeToken đặt NULL (idempotent).
func (s *Service) RevokeToken(ctx context.Context, uid uuid.UUID) error {
	if _, err := store.New(s.Pool).IcsTokenClear(ctx, uid); err != nil {
		return fmt.Errorf("calendar: thu hồi token: %w", err)
	}
	return nil
}

// TokenExists cho GET /me/calendar/ics-token.
func (s *Service) TokenExists(ctx context.Context, uid uuid.UUID) (bool, error) {
	ok, err := store.New(s.Pool).IcsTokenExists(ctx, uid)
	if err != nil {
		return false, fmt.Errorf("calendar: đọc token: %w", err)
	}
	return ok, nil
}

// ErrFeedNotFound: token sai / đã xoay / thu hồi / tài khoản không ACTIVE — một lỗi, một thân (chống dò).
var ErrFeedNotFound = apierr.New(http.StatusNotFound, apierr.NotFound)

// RateOK đếm 1 lượt feed của IP trong phút hiện tại; Redis nil / lỗi → cho qua.
func (s *Service) RateOK(ctx context.Context, ip string) bool {
	if s.Redis == nil {
		return true
	}
	k := fmt.Sprintf("ep:rl:ics:%s:%d", ip, s.now().Unix()/60)
	n, err := s.Redis.Incr(ctx, k).Result()
	if err != nil {
		return true
	}
	if n == 1 {
		_ = s.Redis.Expire(ctx, k, 70*time.Second).Err()
	}
	return n <= FeedPerMin
}

// Feed dựng thân ICS cho chủ token (không JWT). Chỉ dữ liệu của lớp người đó ACTIVE.
func (s *Service) Feed(ctx context.Context, token string) ([]byte, error) {
	if len(token) != 43 {
		return nil, ErrFeedNotFound
	}
	q := store.New(s.Pool)
	h := HashToken(token)
	uid, err := q.IcsUserByHash(ctx, &h)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFeedNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("calendar: tra token: %w", err)
	}
	now := s.now()
	rs, err := q.CalFeed(ctx, store.CalFeedParams{Owner: uid, FromAt: now.Add(-feedBack), ToAt: now.Add(feedAhead), PageLimit: FeedMax})
	if err != nil {
		return nil, fmt.Errorf("calendar: feed: %w", err)
	}
	var b strings.Builder
	line := func(s string) { b.WriteString(fold(s)); b.WriteString("\r\n") }
	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//EduPilot//Calendar//VI")
	line("CALSCALE:GREGORIAN")
	line("X-WR-CALNAME:EduPilot")
	for _, r := range rs {
		title := r.Title
		if r.CourseCount >= 2 {
			title = r.ClassCode + " · " + title
		}
		end := r.StartsAt.Add(defaultSpan)
		if r.HasEnd {
			end = r.EndsAt
		}
		line("BEGIN:VEVENT")
		line("UID:" + r.Src + "-" + r.ID.String() + "@edupilot")
		line("DTSTAMP:" + r.UpdatedAt.UTC().Format(icsTimeFmt)) // ổn định giữa hai lần gọi ⇒ ETag không đổi khi dữ liệu không đổi
		line("DTSTART:" + r.StartsAt.UTC().Format(icsTimeFmt))
		line("DTEND:" + end.UTC().Format(icsTimeFmt))
		line("SUMMARY:" + escape(title))
		if r.Location != nil && *r.Location != "" {
			line("LOCATION:" + escape(*r.Location))
		}
		if r.Src == "weekly_exam" {
			line("URL:" + strings.TrimRight(s.PublicURL, "/") + "/exams/" + r.ID.String() + "/take")
		} else {
			line("URL:" + strings.TrimRight(s.PublicURL, "/") + "/calendar")
		}
		line("END:VEVENT")
	}
	line("END:VCALENDAR")
	return []byte(b.String()), nil
}

// escape theo RFC 5545 §3.3.11: `\`, `;`, `,`, xuống dòng.
func escape(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", `\n`).Replace(s)
}

// fold gập dòng ở 75 octet (CRLF + một khoảng trắng), không cắt giữa ký tự UTF-8.
func fold(s string) string {
	if len(s) <= foldOctets {
		return s
	}
	var b strings.Builder
	limit := foldOctets
	n := 0
	for _, r := range s {
		w := utf8.RuneLen(r)
		if n+w > limit {
			b.WriteString("\r\n ")
			n, limit = 1, foldOctets // dòng tiếp có 1 octet khoảng trắng đứng đầu
		}
		b.WriteRune(r)
		n += w
	}
	return b.String()
}
