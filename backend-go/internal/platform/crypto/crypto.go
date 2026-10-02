// Package crypto: AES-256-GCM cho dữ liệu nhạy cảm trong DB (khoá API của nhà cung cấp LLM — SRS FEAT-llm-gateway 5.6).
//
// Định dạng bản mã: 0x01 ‖ nonce(12) ‖ ciphertext ‖ tag(16). Byte đầu là phiên bản (chừa chỗ xoay vòng khoá — Nợ PR).
// AAD gắn bản mã với bản ghi (ví dụ "llm_providers:"+id): chép bản mã sang bản ghi khác thì giải mã thất bại.
// Lỗi KHÔNG bao giờ chứa khoá, bản rõ hay bản mã.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	blobVersion = byte(0x01)
	keyLen      = 32
	nonceLen    = 12
	tagLen      = 16
)

// ErrInvalidKey: APP_ENCRYPTION_KEY thiếu, không phải base64 hoặc không giải ra đúng 32 byte. Thông báo không chứa giá trị.
var ErrInvalidKey = errors.New("APP_ENCRYPTION_KEY không hợp lệ: cần 32 byte (base64)")

// ErrDecrypt: giải mã thất bại (sai khoá, sai AAD, bản mã bị sửa hoặc sai định dạng). Cố ý không nêu nguyên nhân chi tiết.
var ErrDecrypt = errors.New("không giải mã được dữ liệu")

// Cipher giữ khóa AES-256-GCM. Không có phương thức nào trả khóa; mọi cách in đều ra [REDACTED].
type Cipher struct {
	aead cipher.AEAD
	rand io.Reader
}

// ParseKey dựng Cipher từ chuỗi base64 (chuẩn hoặc URL, có/không đệm) của đúng 32 byte.
func ParseKey(b64 string) (*Cipher, error) {
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		return nil, ErrInvalidKey
	}
	var raw []byte
	var err error
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if raw, err = enc.DecodeString(b64); err == nil {
			break
		}
	}
	if err != nil || len(raw) != keyLen {
		return nil, ErrInvalidKey
	}
	return NewCipher(raw)
}

// NewCipher dựng Cipher từ đúng 32 byte khoá thô.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != keyLen {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrInvalidKey
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("khởi tạo GCM: %w", err)
	}
	return &Cipher{aead: aead, rand: rand.Reader}, nil
}

// Encrypt mã hoá plain với AAD; nonce ngẫu nhiên mỗi lần gọi.
func (c *Cipher) Encrypt(plain, aad []byte) ([]byte, error) {
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(c.rand, nonce); err != nil {
		return nil, fmt.Errorf("sinh nonce: %w", err)
	}
	return c.seal(nonce, plain, aad), nil
}

func (c *Cipher) seal(nonce, plain, aad []byte) []byte {
	out := make([]byte, 0, 1+nonceLen+len(plain)+tagLen)
	out = append(out, blobVersion)
	out = append(out, nonce...)
	return c.aead.Seal(out, nonce, plain, aad)
}

// Decrypt giải mã bản mã do Encrypt tạo ra; sai bất cứ điều gì → ErrDecrypt.
func (c *Cipher) Decrypt(blob, aad []byte) ([]byte, error) {
	if len(blob) < 1+nonceLen+tagLen || blob[0] != blobVersion {
		return nil, ErrDecrypt
	}
	plain, err := c.aead.Open(nil, blob[1:1+nonceLen], blob[1+nonceLen:], aad)
	if err != nil {
		return nil, ErrDecrypt
	}
	return plain, nil
}

// String, GoString, LogValue: không bao giờ lộ khoá qua in ấn / log.
func (*Cipher) String() string   { return "[REDACTED]" }
func (*Cipher) GoString() string { return "[REDACTED]" }
