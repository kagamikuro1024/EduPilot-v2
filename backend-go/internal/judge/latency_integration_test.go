//go:build integration

package judge_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/judge"
)

const bitsAB = "#include <bits/stdc++.h>\nusing namespace std;\nint main(){long long a,b;cin>>a>>b;cout<<a+b<<endl;}\n"

// TestRunLatencyIdle — US-PE-02 AC12: `Chạy thử` (biên dịch `<bits/stdc++.h>` + 3 test mẫu) lúc judge rảnh có p95 ≤ 5 s (20 lượt tuần tự).
func TestRunLatencyIdle(t *testing.T) {
	c := realClient(t)
	tests := []judge.Test{
		{ID: "t1", Position: 1, IsSample: true, Weight: 1, Input: "1 2", Expected: "3", Approved: true},
		{ID: "t2", Position: 2, IsSample: true, Weight: 1, Input: "10 20", Expected: "30", Approved: true},
		{ID: "t3", Position: 3, IsSample: true, Weight: 1, Input: "5 5", Expected: "10", Approved: true},
	}
	const n = 20
	took := make([]time.Duration, 0, n)
	for i := range n {
		start := time.Now()
		res, err := c.Judge(t.Context(), judge.Cpp17, bitsAB+fmt.Sprintf("// %d\n", i), abProblem(), tests)
		require.NoError(t, err)
		require.Equal(t, judge.AC, res.Verdict)
		took = append(took, time.Since(start))
	}
	slices.Sort(took)
	p95 := took[(n*95+99)/100-1]
	t.Logf("Chạy thử rảnh: min=%s p50=%s p95=%s max=%s (n=%d)", took[0].Round(time.Millisecond), took[n/2].Round(time.Millisecond), p95.Round(time.Millisecond), took[n-1].Round(time.Millisecond), n)
	require.LessOrEqual(t, p95, 5*time.Second)
}
