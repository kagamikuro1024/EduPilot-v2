package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestCipher(t *testing.T) *Cipher {
	t.Helper()
	key := make([]byte, keyLen)
	_, err := rand.Read(key)
	require.NoError(t, err)
	c, err := NewCipher(key)
	require.NoError(t, err)
	return c
}

func TestRoundtrip(t *testing.T) {
	t.Parallel()
	c := newTestCipher(t)
	for _, plain := range []string{"", "sk-LEAK-CANARY-7f3a9c1e", strings.Repeat("x", 5000), "tiếng Việt ✓"} {
		blob, err := c.Encrypt([]byte(plain), []byte("llm_providers:id-1"))
		require.NoError(t, err)
		require.Equal(t, byte(0x01), blob[0])
		require.Len(t, blob, 1+nonceLen+len(plain)+tagLen)
		got, err := c.Decrypt(blob, []byte("llm_providers:id-1"))
		require.NoError(t, err)
		require.Equal(t, plain, string(got))
	}
}

func TestNonceUnique1000(t *testing.T) {
	t.Parallel()
	c := newTestCipher(t)
	seen := map[string]bool{}
	nonces := map[string]bool{}
	for range 1000 {
		blob, err := c.Encrypt([]byte("cùng một bản rõ"), []byte("aad"))
		require.NoError(t, err)
		seen[string(blob)] = true
		nonces[string(blob[1:1+nonceLen])] = true
	}
	require.Len(t, seen, 1000)
	require.Len(t, nonces, 1000)
}

func TestTamper(t *testing.T) {
	t.Parallel()
	c := newTestCipher(t)
	blob, err := c.Encrypt([]byte("secret-value"), []byte("aad"))
	require.NoError(t, err)
	for _, tc := range []struct {
		name string
		pos  int
	}{
		{"phiên bản", 0},
		{"nonce đầu", 1},
		{"nonce cuối", nonceLen},
		{"ciphertext đầu", 1 + nonceLen},
		{"ciphertext cuối", len(blob) - tagLen - 1},
		{"tag đầu", len(blob) - tagLen},
		{"tag cuối", len(blob) - 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := bytes.Clone(blob)
			bad[tc.pos] ^= 0x01
			_, err := c.Decrypt(bad, []byte("aad"))
			require.ErrorIs(t, err, ErrDecrypt)
		})
	}
	_, err = c.Decrypt(blob[:10], []byte("aad"))
	require.ErrorIs(t, err, ErrDecrypt)
	_, err = c.Decrypt(nil, nil)
	require.ErrorIs(t, err, ErrDecrypt)
}

func TestAADMismatch(t *testing.T) {
	t.Parallel()
	c := newTestCipher(t)
	blob, err := c.Encrypt([]byte("v"), []byte("llm_providers:A"))
	require.NoError(t, err)
	_, err = c.Decrypt(blob, []byte("llm_providers:B"))
	require.ErrorIs(t, err, ErrDecrypt)
	_, err = c.Decrypt(blob, nil)
	require.ErrorIs(t, err, ErrDecrypt)
}

func TestWrongKey(t *testing.T) {
	t.Parallel()
	a, b := newTestCipher(t), newTestCipher(t)
	blob, err := a.Encrypt([]byte("v"), []byte("aad"))
	require.NoError(t, err)
	_, err = b.Decrypt(blob, []byte("aad"))
	require.ErrorIs(t, err, ErrDecrypt)
}

// TestVector: vectơ thử công bố của NIST (GCM spec, Test Case 16: AES-256, IV 96 bit, có AAD) — bản tham chiếu độc lập
// với mã của dự án. Chèn nonce cố định qua seal.
func TestVector(t *testing.T) {
	t.Parallel()
	h := func(s string) []byte {
		b, err := hex.DecodeString(s)
		require.NoError(t, err)
		return b
	}
	key := h("feffe9928665731c6d6a8f9467308308feffe9928665731c6d6a8f9467308308")
	nonce := h("cafebabefacedbaddecaf888")
	aad := h("feedfacedeadbeeffeedfacedeadbeefabaddad2")
	plain := h("d9313225f88406e5a55909c5aff5269a86a7a9531534f7da2e4c303d8a318a721c3c0c95956809532fcf0e2449a6b525b16aedf5aa0de657ba637b39")
	want := h("522dc1f099567d07f47f37a32a84427d643a8cdcbfe5c0c97598a2bd2555d1aa8cb08e48590dbb3da7b08b1056828838c5f61e6393ba7a0abcc9f662" +
		"76fc6ece0f4e1768cddf8853bb2d551b")

	c, err := NewCipher(key)
	require.NoError(t, err)
	blob := c.seal(nonce, plain, aad)
	require.Equal(t, append(append([]byte{0x01}, nonce...), want...), blob)
	got, err := c.Decrypt(blob, aad)
	require.NoError(t, err)
	require.Equal(t, plain, got)
}

func TestParseKey(t *testing.T) {
	t.Parallel()
	good := make([]byte, keyLen)
	for _, tc := range []struct {
		name, in string
		ok       bool
	}{
		{"đủ 32 byte, std", base64.StdEncoding.EncodeToString(good), true},
		{"đủ 32 byte, url", base64.URLEncoding.EncodeToString(good), true},
		{"không đệm", base64.RawStdEncoding.EncodeToString(good), true},
		{"rỗng", "", false},
		{"toàn khoảng trắng", "   ", false},
		{"không phải base64", "abc$%^", false},
		{"16 byte", base64.StdEncoding.EncodeToString(good[:16]), false},
		{"33 byte", base64.StdEncoding.EncodeToString(append(good, 1)), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := ParseKey(tc.in)
			if tc.ok {
				require.NoError(t, err)
				require.NotNil(t, c)
				return
			}
			require.ErrorIs(t, err, ErrInvalidKey)
			require.NotContains(t, err.Error(), strings.TrimSpace(tc.in)+"x")
		})
	}
}

func TestCipherRedacted(t *testing.T) {
	t.Parallel()
	c := newTestCipher(t)
	for _, s := range []string{c.String(), c.GoString(), fmt.Sprintf("%v %+v %#v %s", c, c, c, c)} {
		require.Contains(t, s, "[REDACTED]")
	}
}
