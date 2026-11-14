package llm_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TestDecisionPGSource: nguồn Postgres đọc đúng ba cổng seed (tắt) và hiệu lực chỉ khi cổng hệ thống bật + có hàng bật của lớp.
func TestDecisionPGSource(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var teacher, course uuid.UUID
	require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ('gv@x.vn', 'GV', 'TEACHER') returning id`).Scan(&teacher))
	require.NoError(t, pool.QueryRow(t.Context(), `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('IT1000', 'CLS001', 'Lớp', '2026-2027-HK1', 'ABCDEFG', $1) returning id`, teacher).Scan(&course))
	snap := llm.NewDecisionSnapshot(llm.PGDecisionSource(pool), nil)
	require.NoError(t, snap.Load(t.Context()))
	g, ok := snap.Gate(llm.GateRouting)
	require.True(t, ok)
	require.False(t, g.Enabled)
	require.InDelta(t, 0.95, g.Threshold, 1e-9)
	require.Equal(t, 800, int(g.Timeout.Milliseconds()))

	_, err = pool.Exec(t.Context(), `insert into course_decision_gates (course_id, gate, enabled) values ($1, 'CHAT_ROUTING', true)`, course)
	require.NoError(t, err)
	require.NoError(t, snap.Load(t.Context()))
	_, on := snap.Effective(llm.GateRouting, &course)
	require.False(t, on, "cổng hệ thống còn tắt")
	_, err = pool.Exec(t.Context(), `update decision_gates set enabled=true where gate='CHAT_ROUTING'`)
	require.NoError(t, err)
	require.NoError(t, snap.Load(t.Context()))
	_, on = snap.Effective(llm.GateRouting, &course)
	require.True(t, on)
	_, on = snap.Effective(llm.GateThreads, &course)
	require.False(t, on)
}
