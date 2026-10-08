// Package judge: máy chấm code C / C++ của bài thi (SRS FEAT-weekly-exam 4.5, D55, D58). Worker gọi sandbox `go-judge` qua REST:
// biên dịch MỘT lần, chạy từng test bằng `fileId` đã cache, so khớp bằng Go thuần (checker.go), phân loại (verdict.go).
// Không chạy tiến trình con trong worker, không ghi đĩa; không gửi danh tính sinh viên — chỉ mã nguồn, dữ liệu test và giới hạn.
package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Language là ngôn ngữ của bài (`code_problems.languages`).
type Language string

const (
	C11   Language = "c11"
	Cpp17 Language = "cpp17"
)

// Hằng số hợp đồng REST (SRS 4.5.2 / 4.5.3).
const (
	compileCPU       = 10 * time.Second
	compileClock     = 20 * time.Second
	compileMemory    = 512 << 20
	compileProcs     = 50
	compileStdout    = 4096
	compileStderr    = 8192
	runStderr        = 4096
	MinTokenLen      = 16
	defaultHTTPSlack = 5 * time.Second
	pathEnv          = "PATH=/usr/bin:/bin"
)

// ErrTokenRequired: JUDGE_TOKEN thiếu hoặc ngắn hơn MinTokenLen (worker thoát 1, nêu tên biến).
var ErrTokenRequired = errors.New("JUDGE_TOKEN phải có ít nhất 16 ký tự")

// ErrSandbox bọc mọi lỗi hệ thống của lời gọi sandbox (HTTP 5xx, mất kết nối, hết hạn, thân không giải mã được): verdict IE, thử lại theo hàng đợi.
var ErrSandbox = errors.New("judge: sandbox lỗi")

// Config của client.
type Config struct {
	URL   string
	Token string
	Slack time.Duration // JUDGE_HTTP_SLACK: cộng vào clockLimit làm hạn mỗi lời gọi (mặc định 5 s)
	HTTP  *http.Client  // tuỳ chọn (test); nil → client riêng không giữ kết nối quá hạn
}

// Client gọi go-judge. An toàn cho nhiều goroutine.
type Client struct {
	base  string
	token string
	slack time.Duration
	http  *http.Client
}

// NewClient dựng client; token ngắn → ErrTokenRequired. Token KHÔNG bao giờ được ghi vào log hay thông báo lỗi.
func NewClient(cfg Config) (*Client, error) {
	if len(cfg.Token) < MinTokenLen {
		return nil, ErrTokenRequired
	}
	if cfg.URL == "" {
		return nil, errors.New("JUDGE_URL trống")
	}
	h := cfg.HTTP
	if h == nil {
		h = &http.Client{}
	}
	slack := cfg.Slack
	if slack <= 0 {
		slack = defaultHTTPSlack
	}
	return &Client{base: strings.TrimRight(cfg.URL, "/"), token: cfg.Token, slack: slack, http: h}, nil
}

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
	CopyOutCached []string        `json:"copyOutCached,omitempty"`
}

// raw là phản hồi một lệnh của go-judge (thời gian ns, bộ nhớ byte).
type raw struct {
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

func (c *Client) do(ctx context.Context, method, path string, body any, deadline time.Duration, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSandbox, err)
		}
		rd = bytes.NewReader(b)
	}
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rd)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSandbox, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSandbox, scrub(err.Error(), c.token))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: HTTP %d", ErrSandbox, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: HTTP %d", ErrSandbox, resp.StatusCode) // 401 / 400: cấu hình sai, cũng là lỗi hệ thống (không phải lỗi của sinh viên)
	}
	if out != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<20)).Decode(out); err != nil {
			return fmt.Errorf("%w: phản hồi không giải mã được", ErrSandbox)
		}
	}
	return nil
}

func scrub(s, token string) string {
	if token == "" {
		return s
	}
	return strings.ReplaceAll(s, token, "***")
}

func (c *Client) run(ctx context.Context, cm cmd) (raw, error) {
	var rs []raw
	deadline := time.Duration(cm.ClockLimit) + c.slack
	if err := c.do(ctx, http.MethodPost, "/run", map[string]any{"cmd": []cmd{cm}}, deadline, &rs); err != nil {
		return raw{}, err
	}
	if len(rs) != 1 {
		return raw{}, fmt.Errorf("%w: phản hồi /run không có đúng một kết quả", ErrSandbox)
	}
	return rs[0], nil
}

// Version thăm dò sức khoẻ (`GET /version`, không cần token — vẫn gửi header).
func (c *Client) Version(ctx context.Context, timeout time.Duration) error {
	return c.do(ctx, http.MethodGet, "/version", nil, timeout, nil)
}

// DeleteFile xoá một fileId đã cache (gọi cả khi lỗi / huỷ; dùng context riêng không bị huỷ).
func (c *Client) DeleteFile(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	return c.do(context.WithoutCancel(ctx), http.MethodDelete, "/file/"+id, nil, c.slack, nil)
}

// Compiled là kết quả biên dịch.
type Compiled struct {
	OK      bool
	FileID  string // fileId của tệp chạy (rỗng khi lỗi); người gọi PHẢI DeleteFile
	Verdict Verdict
	Log     string // đã thay đường dẫn, cắt 8 KiB
}

func sourceFile(l Language) (name string, args []string) {
	if l == C11 {
		return "a.c", []string{"/usr/bin/gcc", "-O2", "-std=c11", "-pipe", "-o", "a", "a.c", "-lm"}
	}
	return "a.cc", []string{"/usr/bin/g++", "-O2", "-std=c++17", "-pipe", "-o", "a", "a.cc"}
}

// Compile biên dịch một lần. Lỗi biên dịch của sinh viên → (Compiled{OK:false, Verdict:CE}, nil); lỗi hệ thống → error bọc ErrSandbox.
func (c *Client) Compile(ctx context.Context, lang Language, source string) (Compiled, error) {
	name, args := sourceFile(lang)
	empty := ""
	r, err := c.run(ctx, cmd{
		Args:          args,
		Env:           []string{pathEnv},
		Files:         []file{{Content: &empty}, {Name: "stdout", Max: compileStdout, Pipe: true}, {Name: "stderr", Max: compileStderr, Pipe: true}},
		CPULimit:      int64(compileCPU),
		ClockLimit:    int64(compileClock),
		MemoryLimit:   compileMemory,
		ProcLimit:     compileProcs,
		CopyIn:        map[string]file{name: {Content: &source}},
		CopyOutCached: []string{"a"},
	})
	if err != nil {
		return Compiled{}, err
	}
	if r.Status == statusInternal || r.Status == statusFile {
		return Compiled{}, fmt.Errorf("%w: %s", ErrSandbox, r.Status)
	}
	if r.Status != statusAccepted {
		return Compiled{Verdict: CE, Log: CompileLog(r.Status, r.Files["stderr"], lang)}, nil
	}
	return Compiled{OK: true, FileID: r.FileIDs["a"]}, nil
}

// Limits là giới hạn chạy một test (từ `code_problems`).
type Limits struct {
	TimeMS        int
	MemoryMB      int
	OutputLimitKB int
}

// RunResult là kết quả chạy một test, chưa so khớp.
type RunResult struct {
	Status   string
	Stdout   string
	TimeMS   int
	MemoryKB int
	ProcPeak int
}

// Run chạy một test: stdin = input; `copyIn` bằng fileId. Chỉ lỗi hệ thống trả error.
func (c *Client) Run(ctx context.Context, fileID, input string, lim Limits) (RunResult, error) {
	cpu := time.Duration(lim.TimeMS) * time.Millisecond
	r, err := c.run(ctx, cmd{
		Args:        []string{"a"},
		Env:         []string{pathEnv},
		Files:       []file{{Content: &input}, {Name: "stdout", Max: int64(lim.OutputLimitKB) * 1024, Pipe: true}, {Name: "stderr", Max: runStderr, Pipe: true}},
		CPULimit:    int64(cpu),
		ClockLimit:  int64(3 * cpu),
		MemoryLimit: int64(lim.MemoryMB) << 20,
		ProcLimit:   1,
		CopyIn:      map[string]file{"a": {FileID: fileID}},
	})
	if err != nil {
		return RunResult{}, err
	}
	return RunResult{Status: r.Status, Stdout: r.Files["stdout"], TimeMS: int(r.Time / int64(time.Millisecond)), MemoryKB: int(r.Memory >> 10), ProcPeak: r.ProcPeak}, nil
}
