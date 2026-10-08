package judge_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/judge"
)

const token = "0123456789abcdef-token"

// fake là go-judge giả: ghi lại mọi yêu cầu, trả phản hồi theo kịch bản.
type fake struct {
	mu       sync.Mutex
	reqs     []map[string]any
	deleted  []string
	auth     []string
	compile  string // JSON một phần tử
	runs     []string
	runIdx   int
	compiles int
	hang     time.Duration // trễ mọi /run
	runDelay time.Duration // trễ riêng lệnh chạy test (không phải biên dịch)
	status5  bool
	srv      *httptest.Server
	versions int
	down     bool   // /version trả 503
	out      string // stdout mặc định của lệnh chạy test
}

func newFake(t *testing.T) *fake {
	f := &fake{compile: `{"status":"Accepted","exitStatus":0,"files":{"stdout":"","stderr":""},"fileIds":{"a":"FID1"}}`}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.auth = append(f.auth, r.Header.Get("Authorization"))
		hang, runDelay, status5, down := f.hang, f.runDelay, f.status5, f.down
		f.mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/version":
			f.mu.Lock()
			f.versions++
			f.mu.Unlock()
			if down {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = io.WriteString(w, `{"buildVersion":"v1.13.0"}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/file/"):
			f.mu.Lock()
			f.deleted = append(f.deleted, strings.TrimPrefix(r.URL.Path, "/file/"))
			f.mu.Unlock()
		case r.Method == http.MethodPost && r.URL.Path == "/run":
			if hang > 0 {
				time.Sleep(hang)
			}
			if status5 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var body map[string]any
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &body)
			cm := body["cmd"].([]any)[0].(map[string]any)
			args := cm["args"].([]any)
			f.mu.Lock()
			f.reqs = append(f.reqs, body)
			isRun := args[0] == "a"
			var resp string
			if isRun {
				resp = okRun(f.out)
				if f.runIdx < len(f.runs) {
					resp = f.runs[f.runIdx]
				}
				f.runIdx++
			} else {
				f.compiles++
				resp = f.compile
			}
			f.mu.Unlock()
			if isRun && runDelay > 0 {
				time.Sleep(runDelay)
			}
			_, _ = io.WriteString(w, "["+resp+"]")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fake) client(t *testing.T) *judge.Client {
	c, err := judge.NewClient(judge.Config{URL: f.srv.URL, Token: token, Slack: 300 * time.Millisecond})
	require.NoError(t, err)
	return c
}

func okRun(out string) string {
	b, _ := json.Marshal(out)
	return `{"status":"Accepted","exitStatus":0,"time":3000000,"memory":1228800,"procPeak":1,"files":{"stdout":` + string(b) + `}}`
}

func problem() judge.Problem {
	return judge.Problem{Limits: judge.Limits{TimeMS: 1000, MemoryMB: 256, OutputLimitKB: 1024}, Checker: judge.Exact, TestsVersion: 3}
}

func tests() []judge.Test {
	return []judge.Test{
		{ID: "t1", Position: 1, IsSample: true, Weight: 1, Input: "1 2\n", Expected: "3\n", Approved: true},
		{ID: "t2", Position: 2, Weight: 1, Input: "2 2\n", Expected: "4\n", Approved: true},
		{ID: "t3", Position: 3, Weight: 2, Input: "5 5\n", Expected: "10\n", Approved: true},
	}
}

// TestClientRequiresToken — token thiếu / ngắn → lỗi nêu tên biến; /run và /file luôn gửi Bearer.
func TestClientRequiresToken(t *testing.T) {
	t.Parallel()
	for _, tok := range []string{"", "short", "0123456789abcde"} {
		_, err := judge.NewClient(judge.Config{URL: "http://x", Token: tok})
		require.ErrorIs(t, err, judge.ErrTokenRequired)
		require.Contains(t, err.Error(), "JUDGE_TOKEN")
	}
	f := newFake(t)
	c := f.client(t)
	_, err := c.Judge(t.Context(), judge.Cpp17, "int main(){}", problem(), tests())
	require.NoError(t, err)
	for _, a := range f.auth {
		require.Equal(t, "Bearer "+token, a)
	}
}

// TestTokenNeverLogged — token không có trong thông báo lỗi (kể cả khi URL / lỗi mạng chứa nó).
func TestTokenNeverLogged(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	c, err := judge.NewClient(judge.Config{URL: "http://127.0.0.1:1", Token: token, Slack: 100 * time.Millisecond})
	require.NoError(t, err)
	_, err = c.Judge(t.Context(), judge.Cpp17, "x", problem(), tests())
	require.Error(t, err)
	log.Error("lỗi chấm", "error", err.Error())
	require.NotContains(t, err.Error(), token)
	require.NotContains(t, buf.String(), token)
	require.True(t, judge.IsSystemError(err))
}

// TestClientCompileRequest — thân POST /run biên dịch đúng hợp đồng 4.5.2 (C và C++).
func TestClientCompileRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		lang judge.Language
		args []any
		file string
	}{
		{judge.Cpp17, []any{"/usr/bin/g++", "-O2", "-std=c++17", "-pipe", "-o", "a", "a.cc"}, "a.cc"},
		{judge.C11, []any{"/usr/bin/gcc", "-O2", "-std=c11", "-pipe", "-o", "a", "a.c", "-lm"}, "a.c"},
	} {
		f := newFake(t)
		c := f.client(t)
		got, err := c.Compile(t.Context(), tc.lang, "SRC-CANARY")
		require.NoError(t, err)
		require.True(t, got.OK)
		require.Equal(t, "FID1", got.FileID)
		cm := f.reqs[0]["cmd"].([]any)[0].(map[string]any)
		require.Equal(t, tc.args, cm["args"])
		require.Equal(t, []any{"PATH=/usr/bin:/bin"}, cm["env"])
		require.EqualValues(t, 10e9, cm["cpuLimit"])
		require.EqualValues(t, 20e9, cm["clockLimit"])
		require.EqualValues(t, 512<<20, cm["memoryLimit"])
		require.EqualValues(t, 50, cm["procLimit"])
		require.Equal(t, []any{"a"}, cm["copyOutCached"])
		require.Equal(t, "SRC-CANARY", cm["copyIn"].(map[string]any)[tc.file].(map[string]any)["content"])
		files := cm["files"].([]any)
		require.EqualValues(t, 4096, files[1].(map[string]any)["max"])
		require.EqualValues(t, 8192, files[2].(map[string]any)["max"])
		for _, k := range []string{"user", "student", "email", "full_name", "student_code"} { // không gửi danh tính
			require.NotContains(t, strings.ToLower(mustJSON(f.reqs[0])), "\""+k+"\"")
		}
	}
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

// TestClientRunRequest — thân POST /run chạy một test: stdin = input, giới hạn theo bài, procLimit 1, copyIn bằng fileId.
func TestClientRunRequest(t *testing.T) {
	t.Parallel()
	f := newFake(t)
	c := f.client(t)
	rr, err := c.Run(t.Context(), "FID9", "INPUT-CANARY", judge.Limits{TimeMS: 1500, MemoryMB: 64, OutputLimitKB: 2048})
	require.NoError(t, err)
	require.Equal(t, "Accepted", rr.Status)
	cm := f.reqs[0]["cmd"].([]any)[0].(map[string]any)
	require.Equal(t, []any{"a"}, cm["args"])
	require.EqualValues(t, 1500e6, cm["cpuLimit"])
	require.EqualValues(t, 3*1500e6, cm["clockLimit"])
	require.EqualValues(t, 64<<20, cm["memoryLimit"])
	require.EqualValues(t, 1, cm["procLimit"])
	files := cm["files"].([]any)
	require.Equal(t, "INPUT-CANARY", files[0].(map[string]any)["content"])
	require.EqualValues(t, 2048*1024, files[1].(map[string]any)["max"])
	require.EqualValues(t, 4096, files[2].(map[string]any)["max"])
	require.Equal(t, "FID9", cm["copyIn"].(map[string]any)["a"].(map[string]any)["fileId"])
}

// TestClientDeletesCachedFile — fileId bị xoá sau khi chấm, cả khi một test gặp lỗi hệ thống.
func TestClientDeletesCachedFile(t *testing.T) {
	t.Parallel()
	f := newFake(t)
	f.runs = []string{okRun("3\n"), `{"status":"Internal Error"}`}
	c := f.client(t)
	_, err := c.Judge(t.Context(), judge.Cpp17, "x", problem(), tests())
	require.True(t, judge.IsSystemError(err))
	require.Equal(t, []string{"FID1"}, f.deleted)

	f2 := newFake(t)
	f2.runs = []string{okRun("3\n"), okRun("4\n"), okRun("10\n")}
	_, err = f2.client(t).Judge(t.Context(), judge.Cpp17, "x", problem(), tests())
	require.NoError(t, err)
	require.Equal(t, []string{"FID1"}, f2.deleted)
}

// TestClientDeadlineAndSemaphore — lời gọi có hạn = clockLimit + slack: judge treo không làm treo worker; 5xx → lỗi hệ thống.
func TestClientDeadlineAndSemaphore(t *testing.T) {
	t.Parallel()
	f := newFake(t)
	f.hang = 3 * time.Second
	c := f.client(t)
	start := time.Now()
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	_, err := c.Run(ctx, "F", "", judge.Limits{TimeMS: 100, MemoryMB: 16, OutputLimitKB: 1})
	require.Error(t, err)
	require.True(t, judge.IsSystemError(err))
	require.Less(t, time.Since(start), 2*time.Second)

	f2 := newFake(t)
	f2.status5 = true
	_, err = f2.client(t).Run(t.Context(), "F", "", judge.Limits{TimeMS: 100, MemoryMB: 16, OutputLimitKB: 1})
	require.True(t, errors.Is(err, judge.ErrSandbox))
}

// TestVerdictMapping — bảng 4.5.4 (≥ 16 ca, gồm chuỗi thật của PoC và ca `lstat stdout`).
func TestVerdictMapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status string
		want   judge.Verdict
	}{
		{"Accepted", judge.AC}, {"Time Limit Exceeded", judge.TLE}, {"Memory Limit Exceeded", judge.MLE}, {"Output Limit Exceeded", judge.OLE},
		{"Nonzero Exit Status", judge.RE}, {"Signalled", judge.RE}, {"Dangerous Syscall", judge.RE},
		{"Internal Error", judge.IE}, {"File Error", judge.IE}, {"", judge.IE}, {"container: stdout: lstat stdout: operation not permitted", judge.IE},
		{"Some New Status", judge.IE}, {"accepted", judge.IE}, {"Time limit exceeded", judge.IE}, {"Killed", judge.IE}, {"Invalid", judge.IE},
	}
	for _, c := range cases {
		require.Equal(t, c.want, judge.MapRunStatus(c.status), c.status)
	}
	require.GreaterOrEqual(t, len(cases), 16)
}

// TestCompileLogScrubbed — đường dẫn tuyệt đối và tên tệp nội bộ → main.cpp / main.c; cắt ≤ 8 KiB; không cắt giữa ký tự.
func TestCompileLogScrubbed(t *testing.T) {
	t.Parallel()
	in := "/w/a.cc: In function 'int main()':\n/w/a.cc:3:5: error: 'x' was not declared\na.cc:7: note: in /usr/include/c++/14/bits/stl.h"
	out := judge.CompileLog("Nonzero Exit Status", in, judge.Cpp17)
	require.Contains(t, out, "main.cpp:3:5: error")
	require.NotContains(t, out, "/w/")
	require.NotContains(t, out, "a.cc")
	require.NotContains(t, out, "/usr/include")
	c := judge.CompileLog("Nonzero Exit Status", "/tmp/x/a.c:1:1: error: lỗi", judge.C11)
	require.Contains(t, c, "main.c:1:1")
	require.NotContains(t, c, "/tmp")
	long := strings.Repeat("ế", 5000)
	got := judge.CompileLog("Nonzero Exit Status", long, judge.Cpp17)
	require.LessOrEqual(t, len(got), judge.MaxCompileLog)
	require.True(t, strings.HasSuffix(got, "ế"))
	require.Contains(t, judge.CompileLog("Time Limit Exceeded", "", judge.Cpp17), "quá thời gian")
	require.Contains(t, judge.CompileLog("Memory Limit Exceeded", "", judge.Cpp17), "quá bộ nhớ")
}

// TestCheckerExact — 14 ca.
func TestCheckerExact(t *testing.T) {
	t.Parallel()
	cases := []struct {
		got, want string
		ok        bool
	}{
		{"3\n", "3\n", true}, {"3", "3\n", true}, {"3\r\n", "3\n", true}, {"3  \n", "3\n", true}, {"3\t\n", "3\n", true},
		{"a\nb\n\n\n", "a\nb", true}, {"a  \nb \n", "a\nb\n", true}, {"", "", true}, {"\n\n", "", true},
		{"a\nb", "a\nc", false}, {"a\n\nb", "a\nb", false}, {"a b", "a  b", false}, {" a", "a", false}, {"3\n4", "3", false},
	}
	for i, c := range cases {
		require.Equalf(t, c.ok, judge.Check(judge.Exact, 0, c.got, c.want), "ca %d %q vs %q", i, c.got, c.want)
	}
}

// TestCheckerTokens — 12 ca.
func TestCheckerTokens(t *testing.T) {
	t.Parallel()
	cases := []struct {
		got, want string
		ok        bool
	}{
		{"1 2 3", "1\n2\n3", true}, {" 1   2 ", "1 2", true}, {"a\tb", "a b", true}, {"1 2", "1 2 3", false}, {"1 2 3", "1 2", false},
		{"abc", "abd", false}, {"", "", true}, {"  \n ", "", true}, {"x", "", false}, {"1e3", "1000", false}, {"A", "a", false}, {"1\u00a02", "1 2", true},
	}
	for i, c := range cases {
		require.Equalf(t, c.ok, judge.Check(judge.Tokens, 0, c.got, c.want), "ca %d %q vs %q", i, c.got, c.want)
	}
}

// TestCheckerFloatEps — 14 ca: 1e3 so 1000, biên eps, NaN / Inf, token chữ lẫn số.
func TestCheckerFloatEps(t *testing.T) {
	t.Parallel()
	cases := []struct {
		got, want string
		eps       float64
		ok        bool
	}{
		{"1e3", "1000", 1e-6, true}, {"3.1415926", "3.14159265", 1e-6, true}, {"3.14", "3.15", 1e-6, false}, {"1000.0005", "1000", 1e-6, true},
		{"1000.002", "1000", 1e-6, false}, {"0.0000001", "0", 1e-6, true}, {"0.000002", "0", 1e-6, false}, {"1.00001", "1", 1.1e-5, true},
		{"1.00002", "1", 1e-5, false}, {"NaN", "NaN", 1e-6, true}, {"NaN", "1", 1e-6, false}, {"Inf", "inf", 1e-6, false},
		{"abc 1.0", "abc 1.0000001", 1e-6, true}, {"abc 1.0", "abd 1.0", 1e-6, false},
	}
	for i, c := range cases {
		require.Equalf(t, c.ok, judge.Check(judge.FloatEps, c.eps, c.got, c.want), "ca %d %q vs %q (eps %g)", i, c.got, c.want, c.eps)
	}
}

// TestSubmissionResultAssembly — kết quả một lượt nộp: results theo position, trọng số, verdict của test lỗi ĐẦU TIÊN, thời gian / bộ nhớ lớn nhất.
func TestSubmissionResultAssembly(t *testing.T) {
	t.Parallel()
	f := newFake(t)
	f.runs = []string{okRun("3\n"), `{"status":"Time Limit Exceeded","time":1001000000,"memory":2048000,"procPeak":1,"files":{"stdout":""}}`, okRun("999\n")}
	res, err := f.client(t).Judge(t.Context(), judge.Cpp17, "x", problem(), tests())
	require.NoError(t, err)
	require.True(t, res.CompileOK)
	require.Equal(t, judge.TLE, res.Verdict, "test lỗi đầu tiên theo position là t2 (TLE), không phải t3 (WA)")
	require.Equal(t, 1, res.PassedWeight)
	require.Equal(t, 4, res.TotalWeight)
	require.Equal(t, 3, res.TestsVersion)
	require.Len(t, res.Results, 3)
	require.Equal(t, judge.AC, res.Results[0].Verdict)
	require.Equal(t, judge.TLE, res.Results[1].Verdict)
	require.Equal(t, judge.WA, res.Results[2].Verdict)
	require.Equal(t, 1001, res.TimeMSMax)
	require.Equal(t, 2000, res.MemoryKBMax)
	require.True(t, res.Results[0].IsSample)
	require.Empty(t, res.Results[2].StdoutExcerpt, "test ẩn không có đoạn đầu ra")

	fAC := newFake(t)
	fAC.runs = []string{okRun("3\n"), okRun("4\n"), okRun("10\n")}
	res, err = fAC.client(t).Judge(t.Context(), judge.Cpp17, "x", problem(), tests())
	require.NoError(t, err)
	require.Equal(t, judge.AC, res.Verdict)
	require.Equal(t, 4, res.PassedWeight)

	fCE := newFake(t)
	fCE.compile = `{"status":"Nonzero Exit Status","exitStatus":1,"files":{"stderr":"/w/a.cc:1:1: error: lỗi"}}`
	res, err = fCE.client(t).Judge(t.Context(), judge.Cpp17, "x", problem(), tests())
	require.NoError(t, err)
	require.Equal(t, judge.CE, res.Verdict)
	require.False(t, res.CompileOK)
	require.Contains(t, res.CompileLog, "main.cpp:1:1")
	require.Equal(t, 4, res.TotalWeight)
	require.Empty(t, res.Results)
}

// TestNoEarlyStop — một test TLE / MLE / RE không dừng các test còn lại: cả 3 test đều chạy.
func TestNoEarlyStop(t *testing.T) {
	t.Parallel()
	f := newFake(t)
	f.runs = []string{`{"status":"Time Limit Exceeded","time":1000000000,"files":{}}`, `{"status":"Memory Limit Exceeded","files":{}}`, okRun("10\n")}
	res, err := f.client(t).Judge(t.Context(), judge.Cpp17, "x", problem(), tests())
	require.NoError(t, err)
	require.Equal(t, 3, f.runIdx)
	require.Len(t, res.Results, 3)
	require.Equal(t, 2, res.PassedWeight)
	require.Equal(t, judge.TLE, res.Verdict)
}

// TestZeroTotalWeight — Σweight = 0: IE + lời giải thích, không chia cho 0, không gọi sandbox, không thử lại.
func TestZeroTotalWeight(t *testing.T) {
	t.Parallel()
	f := newFake(t)
	ts := tests()
	for i := range ts {
		ts[i].Weight = 0
	}
	res, err := f.client(t).Judge(t.Context(), judge.Cpp17, "x", problem(), ts)
	require.NoError(t, err)
	require.Equal(t, judge.IE, res.Verdict)
	require.NotEmpty(t, res.ConfigProblem)
	require.Empty(t, f.reqs)
	// test weight=0 lẫn test weight>0: vẫn chạy, không cộng điểm
	f2 := newFake(t)
	f2.runs = []string{okRun("3\n"), okRun("4\n"), okRun("10\n")}
	ts2 := tests()
	ts2[0].Weight = 0
	res, err = f2.client(t).Judge(t.Context(), judge.Cpp17, "x", problem(), ts2)
	require.NoError(t, err)
	require.Equal(t, 3, f2.runIdx)
	require.Equal(t, 3, res.PassedWeight)
	require.Equal(t, 3, res.TotalWeight)
}

// TestOnlyApprovedTests — test chưa duyệt (nháp AI) không được chấm.
func TestOnlyApprovedTests(t *testing.T) {
	t.Parallel()
	f := newFake(t)
	f.runs = []string{okRun("3\n"), okRun("10\n")}
	ts := tests()
	ts[1].Approved = false
	res, err := f.client(t).Judge(t.Context(), judge.Cpp17, "x", problem(), ts)
	require.NoError(t, err)
	require.Equal(t, 2, f.runIdx)
	require.Len(t, res.Results, 2)
	require.Equal(t, 3, res.TotalWeight)
}
