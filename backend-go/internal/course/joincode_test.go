package course

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/testutil"
)

func TestJoinCodeAlphabet(t *testing.T) {
	t.Parallel()
	require.Len(t, joinCodeAlphabet, 31)
	require.NotRegexp(t, `[01OIL]`, joinCodeAlphabet)
	for range 2000 {
		c, err := NewJoinCode()
		require.NoError(t, err)
		require.Regexp(t, `^[ABCDEFGHJKMNPQRSTUVWXYZ23456789]{7}$`, c)
		require.True(t, joinCodeRE.MatchString(c))
	}
}

// 100.000 mã ⇒ mỗi ký tự 3,23 % ± 0,25 điểm phần trăm (tức đều, không thiên lệch modulo).
func TestJoinCodeDistribution(t *testing.T) {
	t.Parallel()
	count := map[rune]int{}
	const n = 100_000
	for range n {
		c, err := NewJoinCode()
		require.NoError(t, err)
		for _, ch := range c {
			count[ch]++
		}
	}
	require.Len(t, count, 31)
	for _, ch := range joinCodeAlphabet {
		pct := float64(count[ch]) / float64(n*joinCodeLen) * 100
		require.InDeltaf(t, 100.0/31, pct, 0.25, "ký tự %c: %.3f %%", ch, pct)
	}
}

func testService(t *testing.T, production bool) (Service, uuid.UUID) {
	t.Helper()
	testutil.RequireContainers(t)
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var admin uuid.UUID
	require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role, status, password_hash) values ($1, 'A', 'ADMIN', 'ACTIVE', 'x') returning id`, "jc."+uuid.NewString()[:8]+"@example.test").Scan(&admin))
	return Service{Pool: pool, Production: production}, admin
}

func in(class string) CreateInput {
	return CreateInput{SubjectCode: "INT1006", ClassCode: class, Name: "An ninh mạng", Semester: "2026-2027-HK1"}
}

func TestJoinCodeCollisionRetry(t *testing.T) {
	s, admin := testService(t, false)
	ctx := context.Background()
	first, err := s.Create(ctx, admin, in("JC-"+uuid.NewString()[:6]))
	require.NoError(t, err)
	var taken string
	require.NoError(t, s.Pool.QueryRow(ctx, `select join_code from courses where id = $1`, first.ID).Scan(&taken))

	// 4 lần đầu trùng mã đã có, lần thứ 5 là mã mới ⇒ thành công
	calls := 0
	s.genCode = func() (string, error) {
		calls++
		if calls < 5 {
			return taken, nil
		}
		return NewJoinCode()
	}
	v, err := s.Create(ctx, admin, in("JC-"+uuid.NewString()[:6]))
	require.NoError(t, err)
	require.Equal(t, 5, calls)
	var got string
	require.NoError(t, s.Pool.QueryRow(ctx, `select join_code from courses where id = $1`, v.ID).Scan(&got))
	require.NotEqual(t, taken, got)

	// trùng suốt ⇒ dừng sau 5 lần, không tạo lớp nào
	calls = 0
	s.genCode = func() (string, error) { calls++; return taken, nil }
	class := "JC-" + uuid.NewString()[:6]
	_, err = s.Create(ctx, admin, in(class))
	require.True(t, errors.Is(err, ErrCodeExhausted), "%v", err)
	require.Equal(t, maxCodeAttempts, calls)
	var n int
	require.NoError(t, s.Pool.QueryRow(ctx, `select count(*) from courses where class_code = $1`, class).Scan(&n))
	require.Zero(t, n)
}

func TestJoinCodeFixedOnlyOutsideProduction(t *testing.T) {
	ctx := context.Background()
	dev, admin := testService(t, false)
	fixed := in("FX-" + uuid.NewString()[:6])
	fixed.JoinCode = "AN7K2MQ"
	// ngoài production: mã cố định được dùng (seed); lớp thứ hai xin cùng mã ⇒ 422 TAKEN, không lặp vô hạn
	v, err := dev.Create(ctx, admin, fixed)
	if err != nil { // mã seed có thể đã được test khác dùng trên cùng DB mẫu
		var inv *InvalidError
		require.ErrorAs(t, err, &inv)
		require.Equal(t, "TAKEN", inv.Code)
	} else {
		var got string
		require.NoError(t, dev.Pool.QueryRow(ctx, `select join_code from courses where id = $1`, v.ID).Scan(&got))
		require.Equal(t, "AN7K2MQ", got)
	}
	second := in("FX-" + uuid.NewString()[:6])
	second.JoinCode = "AN7K2MQ"
	_, err = dev.Create(ctx, admin, second)
	var inv *InvalidError
	require.ErrorAs(t, err, &inv)
	require.Equal(t, "TAKEN", inv.Code)

	prod := Service{Pool: dev.Pool, Production: true}
	third := in("FX-" + uuid.NewString()[:6])
	third.JoinCode = "BX4P9TW"
	_, err = prod.Create(ctx, admin, third)
	require.ErrorAs(t, err, &inv)
	require.Equal(t, "join_code", inv.Field)
	require.Equal(t, "NOT_ALLOWED", inv.Code)
	var n int
	require.NoError(t, dev.Pool.QueryRow(ctx, `select count(*) from courses where class_code = $1`, third.ClassCode).Scan(&n))
	require.Zero(t, n)
	// sai bảng ký tự
	bad := in("FX-" + uuid.NewString()[:6])
	bad.JoinCode = strings.Repeat("0", 7)
	_, err = dev.Create(ctx, admin, bad)
	require.ErrorAs(t, err, &inv)
	require.Equal(t, "FORMAT", inv.Code)
}
