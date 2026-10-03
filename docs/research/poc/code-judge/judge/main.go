// PoC client chấm C/C++ qua go-judge REST (mô phỏng worker Go): biên dịch 1 lần, chạy từng test, so khớp, phân loại.
// go run . -url http://localhost:15050 [-burst N]
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type file struct {
	Content *string `json:"content,omitempty"`
	FileID  string  `json:"fileId,omitempty"`
	Name    string  `json:"name,omitempty"`
	Max     int64   `json:"max,omitempty"`
	Pipe    bool    `json:"pipe,omitempty"`
}

type cmd struct {
	Args          []string        `json:"args"`
	Env           []string        `json:"env"`
	Files         []file          `json:"files"`
	CPULimit      int64           `json:"cpuLimit"`
	ClockLimit    int64           `json:"clockLimit"`
	MemoryLimit   int64           `json:"memoryLimit"`
	ProcLimit     int             `json:"procLimit"`
	CopyIn        map[string]file `json:"copyIn,omitempty"`
	CopyOut       []string        `json:"copyOut,omitempty"`
	CopyOutCached []string        `json:"copyOutCached,omitempty"`
}

type result struct {
	Status     string            `json:"status"`
	ExitStatus int               `json:"exitStatus"`
	Error      string            `json:"error"`
	Time       int64             `json:"time"`
	Memory     int64             `json:"memory"`
	RunTime    int64             `json:"runTime"`
	ProcPeak   int               `json:"procPeak"`
	Files      map[string]string `json:"files"`
	FileIDs    map[string]string `json:"fileIds"`
}

var base string
func run(c cmd) (result, error) {
	b, _ := json.Marshal(map[string]any{"cmd": []cmd{c}})
	resp, err := http.Post(base+"/run", "application/json", bytes.NewReader(b))
	if err != nil {
		return result{}, err
	}
	defer resp.Body.Close()
	var rs []result
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil || len(rs) != 1 {
		return result{}, fmt.Errorf("decode %d: %v", resp.StatusCode, err)
	}
	return rs[0], nil
}

func del(id string) {
	req, _ := http.NewRequest(http.MethodDelete, base+"/file/"+id, nil)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
	}
}

var env = []string{"PATH=/usr/bin:/bin"}

const sec = int64(time.Second)

type test struct{ in, out string }

// verdict: CE / AC / WA / TLE / MLE / OLE / RE và điểm % test đạt.
func judge(src string, tests []test, tlMs int64, mlMB int64) (string, int, []string, time.Duration) {
	start := time.Now()
	// biên dịch: giới hạn riêng (CPU 10 s, đồng hồ 20 s, 512 MiB, 50 tiến trình cho cc1plus/as/ld)
	cr, err := run(cmd{
		Args:        []string{"/usr/bin/g++", "-O2", "-std=c++17", "-pipe", "-o", "a", "a.cc"},
		Env:         env,
		Files:       []file{{Content: new("")}, {Name: "stdout", Max: 4096, Pipe: true}, {Name: "stderr", Max: 8192, Pipe: true}},
		CPULimit:    10 * sec, ClockLimit: 20 * sec, MemoryLimit: 512 << 20, ProcLimit: 50,
		CopyIn:      map[string]file{"a.cc": {Content: new(src)}},
		CopyOutCached: []string{"a"},
	})
	if err != nil {
		return "IE", 0, []string{err.Error()}, time.Since(start)
	}
	if cr.Status != "Accepted" {
		msg := cr.Status
		if s := cr.Files["stderr"]; s != "" {
			msg += ": " + firstLine(s)
		}
		return "CE", 0, []string{msg}, time.Since(start)
	}
	exe := cr.FileIDs["a"]
	defer del(exe)
	var notes []string
	pass := 0
	worst := "AC"
	for i, t := range tests {
		r, err := run(cmd{
			Args:     []string{"a"},
			Env:      env,
			Files:    []file{{Content: new(t.in)}, {Name: "stdout", Max: 1 << 20, Pipe: true}, {Name: "stderr", Max: 4096, Pipe: true}},
			CPULimit: tlMs * int64(time.Millisecond), ClockLimit: 3 * tlMs * int64(time.Millisecond),
			MemoryLimit: mlMB << 20, ProcLimit: 1,
			CopyIn:   map[string]file{"a": {FileID: exe}},
		})
		v := classify(r, err, t.out)
		if v == "AC" {
			pass++
		} else if worst == "AC" {
			worst = v
		}
		notes = append(notes, fmt.Sprintf("t%d=%s(%s cpu=%dms mem=%dKiB proc=%d%s)", i+1, v, r.Status, r.Time/1e6, r.Memory>>10, r.ProcPeak, errNote(r, err)))
	}
	return worst, pass * 100 / len(tests), notes, time.Since(start)
}

func errNote(r result, err error) string {
	if err != nil {
		return " err=" + err.Error()
	}
	if s := r.Files["stdout"]; s != "" && len(s) < 120 {
		return " out=" + strings.TrimSpace(s)
	}
	return ""
}

func classify(r result, err error, want string) string {
	if err != nil {
		return "IE"
	}
	switch r.Status {
	case "Accepted":
		if norm(r.Files["stdout"]) == norm(want) {
			return "AC"
		}
		return "WA"
	case "Time Limit Exceeded":
		return "TLE"
	case "Memory Limit Exceeded":
		return "MLE"
	case "Output Limit Exceeded":
		return "OLE"
	default: // Nonzero Exit Status, Signalled, Dangerous Syscall, Internal Error
		return "RE"
	}
}

// bỏ khoảng trắng cuối dòng và dòng trống cuối tệp
func norm(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i > 0 {
		s = s[:i]
	}
	if len(s) > 140 {
		s = s[:140]
	}
	return s
}

var abTests = []test{{"1 2\n", "3\n"}, {"-5 5\n", "0"}, {"1000000000 1000000000\n", "2000000000\n"}}

var subs = []struct{ name, src string }{
	{"AC a+b", `#include <cstdio>
int main(){long long a,b; if(scanf("%lld %lld",&a,&b)!=2) return 1; printf("%lld\n",a+b);}`},
	{"WA", `#include <cstdio>
int main(){int a,b; scanf("%d %d",&a,&b); printf("%d\n",a-b);}`},
	{"CE", `int main(){ return x; }`},
	{"TLE vòng lặp vô hạn", `int main(){ volatile unsigned long x=0; for(;;) x++; }`},
	{"MLE chạm dần 512 MiB", `#include <cstdio>
#include <cstdlib>
#include <cstring>
int main(){ long t=0; for(int i=0;i<512;i++){ char*p=(char*)malloc(1<<20); if(!p){ printf("null at %d\n", i); return 2; } memset(p,i,1<<20); t+=p[7]; } printf("%ld\n", t); }`},
	{"RE segfault", `int main(){ volatile int*p=0; *p=1; }`},
	{"OLE in vô hạn", `#include <cstdio>
int main(){ for(;;) puts("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"); }`},
	{"fork bomb", `#include <unistd.h>
#include <cstdio>
int main(){ int ok=0; for(int i=0;i<100000;i++){ pid_t p=fork(); if(p==0){ for(;;) fork(); } if(p>0) ok++; } printf("forked %d\n", ok); }`},
	{"đọc file hệ thống", `#include <cstdio>
int main(){ const char* fs[]={"/etc/shadow","/etc/passwd","/proc/1/environ","/opt/go-judge","/root/.bashrc"};
for(auto f: fs){ FILE* p=fopen(f,"r"); printf("%s=%s ", f, p?"MO":"CHAN"); if(p) fclose(p);} puts(""); }`},
	{"mạng ra ngoài", `#include <cstdio>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
int main(){ int s=socket(AF_INET,SOCK_STREAM,0); sockaddr_in a{}; a.sin_family=AF_INET; a.sin_port=htons(53); inet_pton(AF_INET,"1.1.1.1",&a.sin_addr);
int r=connect(s,(sockaddr*)&a,sizeof a); printf("socket=%d connect=%d\n", s, r); }`},
	{"CE include /dev/random", `#include "/dev/random"
int main(){}`},
}

func main() {
	flag.StringVar(&base, "url", "http://localhost:15050", "go-judge")
	burst := flag.Int("burst", 0, "số bài AC nộp đồng thời (đo thông lượng)")
	flag.Parse()
	if *burst > 0 {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var worst time.Duration
		ok := 0
		start := time.Now()
		tests := make([]test, 10)
		for i := range tests {
			tests[i] = abTests[i%len(abTests)]
		}
		for range *burst {
			wg.Go(func() {
				// bài thực tế của SV thường dùng <bits/stdc++.h> → biên dịch nặng hơn nhiều so với <cstdio>
				v, _, _, d := judge("#include <bits/stdc++.h>\nusing namespace std;\nint main(){long long a,b;cin>>a>>b;cout<<a+b<<\"\\n\";}", tests, 1000, 256)
				mu.Lock()
				if v == "AC" {
					ok++
				}
				worst = max(worst, d)
				mu.Unlock()
			})
		}
		wg.Wait()
		el := time.Since(start)
		fmt.Printf("burst=%d (1 biên dịch + 10 test mỗi bài): AC=%d tổng=%.1fs chậm nhất=%.1fs → %.0f bài/phút\n", *burst, ok, el.Seconds(), worst.Seconds(), float64(*burst)/el.Minutes())
		return
	}
	for _, s := range subs {
		v, score, notes, d := judge(s.src, abTests, 1000, 256)
		fmt.Printf("%-24s → %-4s %3d%%  %5.2fs  %s\n", s.name, v, score, d.Seconds(), strings.Join(notes, " "))
	}
	os.Exit(0)
}
