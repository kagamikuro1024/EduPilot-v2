//go:build integration

package judge_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/testutil"
)

func realClient(t *testing.T) *judge.Client {
	t.Helper()
	c, err := judge.NewClient(judge.Config{URL: testutil.JudgeURL(t), Token: testutil.JudgeToken})
	require.NoError(t, err)
	return c
}

const goodAB = `#include <cstdio>
int main(){long long a,b; if(scanf("%lld %lld",&a,&b)!=2) return 1; printf("%lld\n",a+b);}`

func abProblem() judge.Problem {
	return judge.Problem{Limits: judge.Limits{TimeMS: 1000, MemoryMB: 256, OutputLimitKB: 1024}, Checker: judge.Exact, TestsVersion: 1}
}

func oneTest(input, expected string) []judge.Test {
	return []judge.Test{{ID: "t1", Position: 1, Weight: 1, Input: input, Expected: expected, Approved: true}}
}

type attack struct {
	id       string
	lang     judge.Language
	src      string
	expected string
	want     []judge.Verdict // verdict cho phép; nil = chỉ cần "không AC"
	note     string
}

const cHeader = "#include <stdio.h>\n#include <stdlib.h>\n#include <string.h>\n#include <unistd.h>\n"

func attacks() []attack {
	chan5 := "CHAN\nCHAN\nCHAN\nCHAN\nCHAN\n"
	return []attack{
		{"A1", judge.C11, cHeader + `int main(){volatile long x=0; for(;;) x++;}`, "", []judge.Verdict{judge.TLE}, "vòng lặp vô hạn"},
		{"A2", judge.C11, cHeader + `int main(){for(;;) puts("hello world hello world hello world");}`, "", []judge.Verdict{judge.TLE, judge.OLE}, "in vô hạn: stdout cắt ở giới hạn"},
		{"A3", judge.C11, cHeader + `int main(){static char b[1024]; memset(b,'a',sizeof b); for(int i=0;i<5*1024;i++) fwrite(b,1,sizeof b,stdout); return 0;}`, "", []judge.Verdict{judge.OLE}, "in hữu hạn 5 MB"},
		{"A4", judge.C11, cHeader + `int main(){static char b[1<<20]; memset(b,'a',sizeof b); for(int i=0;i<1024;i++) fwrite(b,1,sizeof b,stdout); return 0;}`, "", []judge.Verdict{judge.OLE, judge.TLE}, "in 1 GB"},
		{"A5", judge.C11, cHeader + `int main(){for(int i=0;i<512;i++){char*p=malloc(1<<20); if(!p) return 3; memset(p,1,1<<20);} return 0;}`, "", []judge.Verdict{judge.MLE}, "cấp phát dần tới 512 MiB"},
		{"A6", judge.C11, cHeader + `int main(){size_t n=2UL<<30; volatile char*p=malloc(n); if(!p) return 3; for(size_t i=0;i<n;i+=4096) p[i]=1; printf("%d\n",p[n-4096]); return 0;}`, "", []judge.Verdict{judge.MLE, judge.RE}, "cấp phát 2 GB một lần"},
		{"A7", judge.C11, cHeader + `int main(){volatile int*p=0; *p=1; return 0;}`, "", []judge.Verdict{judge.RE}, "con trỏ NULL"},
		{"A8", judge.C11, cHeader + `int main(){for(;;) fork();}`, "", []judge.Verdict{judge.TLE, judge.RE, judge.MLE}, "fork bomb (bất biến thật: procPeak == 1, kiểm bên dưới; #19)"},
		{"A9", judge.C11, cHeader + `int main(){const char*p[]={"/etc/shadow","/etc/passwd","/proc/1/environ","/opt/go-judge","/root/.bashrc"}; for(int i=0;i<5;i++){FILE*f=fopen(p[i],"r"); if(f){puts("OPEN");fclose(f);} else puts("CHAN");} return 0;}`, chan5, []judge.Verdict{judge.AC}, "5 đường tệp hệ thống bị chặn"},
		{"A10", judge.C11, netHeader + `int main(){printf("connect=%d\n",tryc("1.1.1.1",53)); return 0;}`, "connect=-1\n", []judge.Verdict{judge.AC}, "connect 1.1.1.1:53"},
		{"A10b", judge.C11, netHeader + `int main(){printf("%d %d %d %d\n",tryc("127.0.0.1",5050),tryc("172.17.0.1",5432),tryc("172.17.0.2",6379),tryc("10.0.0.1",5432)); return 0;}`, "-1 -1 -1 -1\n", []judge.Verdict{judge.AC}, "connect tới cổng nội bộ / IP dịch vụ"},
		{"A11", judge.Cpp17, "#include \"/dev/random\"\nint main(){}", "", []judge.Verdict{judge.CE}, `#include "/dev/random"`},
		{"A12", judge.Cpp17, "constexpr long long f(long long n){long long s=0; for(long long i=0;i<n;i++) s+=i%7; return s;}\nconstexpr long long v=f(100000000000LL);\nint main(){return (int)v;}", "", []judge.Verdict{judge.CE}, "bom biên dịch constexpr"},
		{"A13", judge.C11, cHeader + `int main(){static char b[1<<20]; memset(b,'a',sizeof b); FILE*f=fopen("/tmp/big","w"); if(!f) {puts("denied"); return 0;} for(int i=0;i<1024;i++) if(fwrite(b,1,sizeof b,f)!=sizeof b){puts("full"); return 0;} puts("ok"); return 0;}`, "ok\n", nil, "ghi 1 GB vào /tmp"},
		{"A14", judge.C11, cHeader + `int main(){int r=system("ls / ; cat /etc/shadow"); puts(r==0?"RAN":"BLOCKED"); return 0;}`, "BLOCKED\n", []judge.Verdict{judge.AC}, "system() không tạo được tiến trình con"},
	}
}

const netHeader = cHeader + `#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
static int tryc(const char*ip,int port){int s=socket(AF_INET,SOCK_STREAM,0); if(s<0) return -2; struct sockaddr_in a; memset(&a,0,sizeof a); a.sin_family=AF_INET; a.sin_port=htons(port); inet_pton(AF_INET,ip,&a.sin_addr); int r=connect(s,(struct sockaddr*)&a,sizeof a); close(s); return r<0?-1:r;}
`

// procPeakOf biên dịch + chạy một lần mã tấn công với giới hạn của abProblem rồi trả đỉnh số tiến trình và bộ nhớ (KiB) go-judge ghi nhận.
func procPeakOf(t *testing.T, c *judge.Client, a attack) (peak, memKB int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cp, err := c.Compile(ctx, a.lang, a.src)
	require.NoErrorf(t, err, "%s: biên dịch lại", a.id)
	require.Truef(t, cp.OK, "%s: biên dịch lại: %s", a.id, cp.Log)
	defer func() { _ = c.DeleteFile(ctx, cp.FileID) }()
	lim := abProblem().Limits
	rr, err := c.Run(ctx, cp.FileID, "", lim)
	require.NoErrorf(t, err, "%s: chạy lại", a.id)
	return rr.ProcPeak, rr.MemoryKB
}

// TestSandboxAttacks — US-PE-02 AC11: 15 ca tấn công đi qua ĐƯỜNG CHẤM THẬT (Client.Judge → go-judge thật trong container privileged);
// mỗi ca ra đúng verdict / dấu hiệu, và TRONG LÚC chạy một bài đúng của sinh viên khác vẫn AC ≤ 20 s; container không khởi động lại.
func TestSandboxAttacks(t *testing.T) {
	c := realClient(t)
	restarts := testutil.JudgeRestarts(t)

	good := func() (judge.Result, time.Duration, error) {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		res, err := c.Judge(ctx, judge.Cpp17, goodAB, abProblem(), oneTest("1 2\n", "3\n"))
		return res, time.Since(start), err
	}
	// đối chứng: bài đúng chấm AC khi rảnh
	res, _, err := good()
	require.NoError(t, err)
	require.Equal(t, judge.AC, res.Verdict, "đối chứng: bài đúng phải AC")

	for _, a := range attacks() {
		var wg sync.WaitGroup
		var gRes judge.Result
		var gDur time.Duration
		var gErr error
		wg.Add(1)
		go func() { defer wg.Done(); time.Sleep(300 * time.Millisecond); gRes, gDur, gErr = good() }()

		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		r, err := c.Judge(ctx, a.lang, a.src, abProblem(), oneTest("", a.expected))
		cancel()
		wg.Wait()
		require.NoErrorf(t, err, "%s: lỗi hệ thống của sandbox", a.id)
		require.NotEqualf(t, judge.IE, r.Verdict, "%s: IE (lỗi sandbox, không phải verdict của bài)", a.id)
		if len(a.want) > 0 {
			require.Containsf(t, a.want, r.Verdict, "%s (%s): verdict %s", a.id, a.note, r.Verdict)
		} else {
			require.NotEqualf(t, judge.AC, r.Verdict, "%s (%s): phải bị chặn", a.id, a.note)
		}
		if a.id == "A11" || a.id == "A12" {
			require.Lessf(t, time.Since(start), 25*time.Second, "%s: biên dịch phải bị cắt ≤ 25 s", a.id)
		}
		require.NoErrorf(t, gErr, "%s: bài của sinh viên khác bị ảnh hưởng", a.id)
		require.Equalf(t, judge.AC, gRes.Verdict, "%s: bài đúng của sinh viên khác phải vẫn AC", a.id)
		require.Lessf(t, gDur, 20*time.Second, "%s: bài đúng chấm quá chậm trong lúc bị tấn công", a.id)
		line := fmt.Sprintf("%s %s %s (%.1fs; bài khác AC trong %.1fs)", a.id, r.Verdict, a.note, time.Since(start).Seconds(), gDur.Seconds())
		if a.id == "A8" || a.id == "A14" { // bất biến thật của "không fork được": đỉnh tiến trình = 1 (Result không mang procPeak → chạy lại qua Compile + Run); TL-4
			peak, memKB := procPeakOf(t, c, a)
			require.Equalf(t, 1, peak, "%s: sandbox để bài tạo tiến trình thứ hai", a.id)
			line += fmt.Sprintf(" · procPeak=%d, bộ nhớ %d KiB", peak, memKB)
		}
		fmt.Println(line)
		t.Log(line)
	}

	// A6 đồng thời P bản: container sống, không khởi động lại (góp ý #5).
	var wg sync.WaitGroup
	verdicts := make([]judge.Verdict, testutil.JudgeParallelism)
	for i := 0; i < testutil.JudgeParallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := c.Judge(context.Background(), judge.C11, cHeader+`int main(){size_t n=2UL<<30; volatile char*p=malloc(n); if(!p) return 3; for(size_t i=0;i<n;i+=4096) p[i]=1; printf("%d\n",p[n-4096]); return 0;}`, abProblem(), oneTest("", ""))
			if err == nil {
				verdicts[i] = r.Verdict
			}
		}()
	}
	wg.Wait()
	for i, v := range verdicts {
		require.Containsf(t, []judge.Verdict{judge.MLE, judge.RE}, v, "A6 đồng thời bản %d", i)
	}
	fmt.Printf("A6 đồng thời x%d %v container sống\n", testutil.JudgeParallelism, verdicts)

	// A15: rò dữ liệu giữa hai lượt chạy: lượt 1 ghi /tmp/leak và /w/leak, lượt 2 (ngay sau) đọc.
	w, err := c.Judge(context.Background(), judge.C11, cHeader+`int main(){FILE*a=fopen("/tmp/leak","w"); if(a){fputs("secret",a);fclose(a);} FILE*b=fopen("/w/leak","w"); if(b){fputs("secret",b);fclose(b);} puts("wrote"); return 0;}`, abProblem(), oneTest("", "wrote\n"))
	require.NoError(t, err)
	require.Equal(t, judge.AC, w.Verdict)
	r, err := c.Judge(context.Background(), judge.C11, cHeader+`int main(){const char*p[]={"/tmp/leak","/w/leak"}; for(int i=0;i<2;i++){FILE*f=fopen(p[i],"r"); if(f){puts("LEAK");fclose(f);} else puts("CHAN");} return 0;}`, abProblem(), oneTest("", "CHAN\nCHAN\n"))
	require.NoError(t, err)
	require.Equal(t, judge.AC, r.Verdict, "A15: lượt 2 không được thấy tệp của lượt 1")
	fmt.Println("A15 AC CHAN×2 (lượt 2 không thấy dữ liệu lượt 1)")

	require.Equal(t, restarts, testutil.JudgeRestarts(t), "container judge không được khởi động lại")
}
