//go:build integration

package exam_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TestVerifyReferenceRealJudge — AC6 trên go-judge THẬT: lời giải mẫu đúng → mọi test AC và cờ được ghi; lời giải mẫu sai ở một test → lệch, không ghi cờ.
func TestVerifyReferenceRealJudge(t *testing.T) {
	r := newRig(t)
	c, err := judge.NewClient(judge.Config{URL: testutil.JudgeURL(t), Token: testutil.JudgeToken})
	require.NoError(t, err)
	run := newRunner(t, r, &exam.Worker{Sandbox: c})
	q, _, _ := r.codeWithTests("verify thật")
	good := `#include <cstdio>
int main(){long long a,b; if(scanf("%lld %lld",&a,&b)!=2) return 1; printf("%lld\n",a+b);}`
	put := func(src string) {
		_, err := r.svc.PutCode(t.Context(), r.course, q.ID, exam.CodeIn{Languages: []string{"cpp17"}, Reference: &exam.Reference{Language: "cpp17", Source: src}}, r.version(q.ID))
		require.NoError(t, err)
	}
	verify := func() exam.VerifyResult {
		id, err := r.svc.EnqueueVerify(t.Context(), r.teacher, r.course, q.ID)
		require.NoError(t, err)
		j := runJob(t, r, run, id)
		require.Equal(t, "SUCCEEDED", j.Status)
		var res exam.VerifyResult
		require.NoError(t, json.Unmarshal(j.Result, &res))
		return res
	}
	put(good)
	res := verify()
	require.True(t, res.OK, "%+v", res)
	require.NotNil(t, r.problem(q.ID).Verified)
	put(`#include <cstdio>
int main(){long long a,b; scanf("%lld %lld",&a,&b); printf("%lld\n",a*b);}`) // sai: nhân
	res = verify()
	require.False(t, res.OK)
	require.Nil(t, r.problem(q.ID).Verified)
	var wa int
	for _, p := range res.PerTest {
		if p.Verdict == judge.WA {
			wa++
		}
	}
	require.GreaterOrEqual(t, wa, 1)
}
