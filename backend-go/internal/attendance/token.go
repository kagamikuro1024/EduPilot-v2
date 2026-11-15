// Package attendance: điểm danh tự phục vụ chống gian lận (FEAT-attendance-selfcheck, ARCHITECTURE §11). Mã QR xoay vòng kiểm bằng HMAC
// không trạng thái, khoá thiết bị ký từng lượt, cờ bất thường chỉ gắn — không bao giờ từ chối vì tín hiệu. Không toạ độ, không IP, không
// token / chữ ký trong DB, log hay audit. `internal/grade` không đọc gì ở đây.
package attendance

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const (
	tokenMACLen  = 16                                 // ký tự base32 (80 bit) của MAC trong token
	crockford    = "0123456789ABCDEFGHJKMNPQRSTVWXYZ" // bảng mã ngắn: 0-9A-Z trừ I L O U
	signaturePre = "edupilot-checkin-v1|"
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding) //nolint:gochecknoglobals // bảng chữ bất biến

// WindowKey = HMAC-SHA256(K_root, "att-v1|" + window_id) — một HMAC mỗi request, không đệm, không lưu bí mật theo buổi.
func WindowKey(root []byte, windowID uuid.UUID) []byte {
	m := hmac.New(sha256.New, root)
	m.Write([]byte("att-v1|" + windowID.String()))
	return m.Sum(nil)
}

// Step = ⌊unix_s / rotate_seconds⌋.
func Step(unixSec int64, rotate int) int64 { return unixSec / int64(rotate) }

// StepExpiresAt = (step + 1) × rotate (giây Unix).
func StepExpiresAt(step int64, rotate int) int64 { return (step + 1) * int64(rotate) }

// TokenAgeMs = now_ms − step × rotate × 1000 (không âm).
func TokenAgeMs(nowMs, step int64, rotate int) int64 {
	age := nowMs - step*int64(rotate)*1000
	if age < 0 {
		return 0
	}
	return age
}

func macOf(kw []byte, step int64) string {
	m := hmac.New(sha256.New, kw)
	m.Write([]byte(strconv.FormatInt(step, 10)))
	return b32.EncodeToString(m.Sum(nil))[:tokenMACLen]
}

// MakeToken = window_id + "." + step + "." + base32(HMAC(K_w, decimal(step)))[:16].
func MakeToken(root []byte, windowID uuid.UUID, step int64) string {
	return windowID.String() + "." + strconv.FormatInt(step, 10) + "." + macOf(WindowKey(root, windowID), step)
}

// ParsedToken là kết quả tách cú pháp (chưa kiểm MAC).
type ParsedToken struct {
	WindowID uuid.UUID
	Step     int64
	MAC      string
}

// ParseToken tách `window.step.mac`; sai cú pháp → ok=false (người gọi trả TOKEN_INVALID, không phân biệt).
func ParseToken(s string) (ParsedToken, bool) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 || len(parts[2]) != tokenMACLen || len(s) > 160 {
		return ParsedToken{}, false
	}
	wid, err := uuid.Parse(parts[0])
	if err != nil {
		return ParsedToken{}, false
	}
	step, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || step < 0 {
		return ParsedToken{}, false
	}
	return ParsedToken{WindowID: wid, Step: step, MAC: parts[2]}, true
}

// VerifyMAC so MAC bằng hmac.Equal (thời gian không đổi).
func VerifyMAC(root []byte, p ParsedToken) bool {
	want := macOf(WindowKey(root, p.WindowID), p.Step)
	return hmac.Equal([]byte(want), []byte(p.MAC))
}

// StepAccepted: nhận step ∈ {s, s−1}; bước tương lai hoặc cũ hơn → TOKEN_EXPIRED.
func StepAccepted(tokenStep, nowStep int64) bool {
	return tokenStep == nowStep || tokenStep == nowStep-1
}

// ShortCode = 30 bit đầu của HMAC(K_w, "short|" + decimal(step)) mã hoá Crockford, độ dài n (6): đổi CÙNG nhịp với token.
func ShortCode(root []byte, windowID uuid.UUID, step int64, n int) string {
	m := hmac.New(sha256.New, WindowKey(root, windowID))
	m.Write([]byte("short|" + strconv.FormatInt(step, 10)))
	sum := m.Sum(nil)
	var bits uint64
	for _, b := range sum[:5] { // 40 bit đủ cho n ≤ 8 ký tự × 5 bit
		bits = bits<<8 | uint64(b)
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = crockford[(bits>>(uint(35-5*i)))&31]
	}
	return string(out)
}

// FormatShortCode nhóm 3 + 3 để hiển thị («7K2 9QX»).
func FormatShortCode(c string) string {
	if len(c) <= 3 {
		return c
	}
	return c[:3] + " " + c[3:]
}

// NormalizeCode: bỏ khoảng trắng / `-`, in hoa, O→0, I/L→1. Ký tự ngoài bảng mã → ok=false.
func NormalizeCode(s string) (string, bool) {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		switch r {
		case ' ', '-', '\t', '\n':
			continue
		case 'O':
			r = '0'
		case 'I', 'L':
			r = '1'
		}
		if r > 127 || !strings.ContainsRune(crockford, r) {
			return "", false
		}
		b.WriteRune(r)
	}
	return b.String(), b.Len() > 0
}

// CodeMatches so mã ngắn đã chuẩn hoá với mã của một bước bằng ConstantTimeCompare.
func CodeMatches(root []byte, windowID uuid.UUID, step int64, n int, norm string) bool {
	return subtle.ConstantTimeCompare([]byte(ShortCode(root, windowID, step, n)), []byte(norm)) == 1
}

// SignedMessage là chuỗi thiết bị ký: đường token "edupilot-checkin-v1|{token}|{user_id}"; đường mã ngắn "edupilot-checkin-v1|code:{CHUẨN_HOÁ}|{user_id}".
func SignedMessage(token, normCode string, userID uuid.UUID) []byte {
	if token == "" {
		return []byte(signaturePre + "code:" + normCode + "|" + userID.String())
	}
	return []byte(signaturePre + token + "|" + userID.String())
}
