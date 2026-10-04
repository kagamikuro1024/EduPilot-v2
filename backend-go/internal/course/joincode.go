package course

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
)

// joinCodeAlphabet: 31 ký tự, bỏ 0 O 1 I L (dễ đọc nhầm khi chép tay). Không gian 31⁷ ≈ 2,75·10¹⁰.
const joinCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const joinCodeLen = 7

var joinCodeRE = regexp.MustCompile(`^[` + joinCodeAlphabet + `]{7}$`)

// NewJoinCode sinh một mã tham gia bằng crypto/rand (`rand.Int` với big.Int — không thiên lệch modulo).
func NewJoinCode() (string, error) {
	n := big.NewInt(int64(len(joinCodeAlphabet)))
	b := make([]byte, joinCodeLen)
	for i := range b {
		v, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", fmt.Errorf("course: sinh mã tham gia: %w", err)
		}
		b[i] = joinCodeAlphabet[v.Int64()]
	}
	return string(b), nil
}
