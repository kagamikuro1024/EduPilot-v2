package attendance

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var testRoot = []byte("0123456789abcdef0123456789abcdef")

func TestTokenRoundTripAndTamper(t *testing.T) {
	wid := uuid.New()
	tok := MakeToken(testRoot, wid, 1234)
	p, ok := ParseToken(tok)
	require.True(t, ok)
	require.Equal(t, wid, p.WindowID)
	require.Equal(t, int64(1234), p.Step)
	require.True(t, VerifyMAC(testRoot, p))

	p.Step++ // đổi bước nhưng giữ MAC
	require.False(t, VerifyMAC(testRoot, p))
	other := MakeToken(testRoot, uuid.New(), 1234) // MAC của cửa sổ khác
	q, _ := ParseToken(tok)
	o, _ := ParseToken(other)
	q.MAC = o.MAC
	require.False(t, VerifyMAC(testRoot, q))
	for _, bad := range []string{"", "x", "a.b.c", wid.String() + ".12"} {
		_, ok := ParseToken(bad)
		require.False(t, ok, bad)
	}
}

func TestTokenStepWindow(t *testing.T) {
	for _, c := range []struct {
		tok, now int64
		ok       bool
	}{{10, 10, true}, {9, 10, true}, {8, 10, false}, {11, 10, false}} {
		require.Equal(t, c.ok, StepAccepted(c.tok, c.now), "%d/%d", c.tok, c.now)
	}
}

func TestShortCodeNormalize(t *testing.T) {
	wid := uuid.New()
	c := ShortCode(testRoot, wid, 77, 6)
	require.Len(t, c, 6)
	n, ok := NormalizeCode("  " + FormatShortCode(c) + " ")
	require.True(t, ok)
	require.Equal(t, c, n)
	require.True(t, CodeMatches(testRoot, wid, 77, 6, c))
	require.False(t, CodeMatches(testRoot, wid, 78, 6, c))
	// Crockford: O→0, I/L→1.
	n, ok = NormalizeCode("o1l-iOO")
	require.True(t, ok)
	require.Equal(t, "011100", n)
	_, ok = NormalizeCode("A!C")
	require.False(t, ok)
}
