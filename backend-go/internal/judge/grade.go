package judge

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

// Test là một test của bài (đã đọc từ DB). Chỉ test `Approved` được chấm.
type Test struct {
	ID       string
	Position int
	IsSample bool
	Weight   int
	Input    string
	Expected string
	Approved bool
}

// Problem là phần của `code_problems` mà máy chấm cần.
type Problem struct {
	Limits       Limits
	Checker      CheckerKind
	FloatEps     float64
	TestsVersion int
}

// TestResult là một phần tử của `code_submissions.results`.
type TestResult struct {
	TestID   string  `json:"test_id"`
	Position int     `json:"position"`
	IsSample bool    `json:"is_sample"`
	Verdict  Verdict `json:"verdict"`
	TimeMS   int     `json:"time_ms"`
	MemoryKB int     `json:"memory_kb"`
	Weight   int     `json:"weight"`
	// StdoutExcerpt: đoạn đầu ra (≤ 200 byte) CHỈ cho test mẫu trượt — phía sinh viên không bao giờ nhận cho test ẩn.
	StdoutExcerpt string `json:"stdout_excerpt,omitempty"`
}

// Result là kết quả cả bản nộp (cột của `code_submissions`).
type Result struct {
	Verdict       Verdict
	CompileOK     bool
	CompileLog    string
	Results       []TestResult
	PassedWeight  int
	TotalWeight   int
	TimeMSMax     int
	MemoryKBMax   int
	TestsVersion  int
	ConfigProblem string // khác rỗng: lỗi cấu hình của bài (Σweight = 0 …) — verdict IE, KHÔNG thử lại
}

const stdoutExcerptMax = 200

// Judge chấm một bản nộp: biên dịch MỘT lần, chạy MỌI test approved theo `position` (một test TLE / MLE không dừng test khác), xoá fileId khi xong.
// Lỗi hệ thống (error bọc ErrSandbox) → trả error để hàng đợi thử lại; KHÔNG ghi lỗi hệ thống thành điểm 0.
func (c *Client) Judge(ctx context.Context, lang Language, source string, p Problem, tests []Test) (Result, error) {
	var approved []Test
	for _, t := range tests {
		if t.Approved {
			approved = append(approved, t)
		}
	}
	sort.SliceStable(approved, func(i, j int) bool { return approved[i].Position < approved[j].Position })
	res := Result{TestsVersion: p.TestsVersion}
	for _, t := range approved {
		res.TotalWeight += t.Weight
	}
	if res.TotalWeight <= 0 {
		res.Verdict = IE
		res.ConfigProblem = "Bài chưa có test nào có trọng số > 0 (tổng trọng số bằng 0)."
		res.CompileLog = res.ConfigProblem
		return res, nil
	}

	comp, err := c.Compile(ctx, lang, source)
	if err != nil {
		return Result{}, err
	}
	if !comp.OK {
		res.Verdict, res.CompileLog = CE, comp.Log
		res.TotalWeight = totalOf(approved)
		return res, nil
	}
	res.CompileOK = true
	defer func() { _ = c.DeleteFile(ctx, comp.FileID) }()

	var firstBad Verdict
	for _, t := range approved {
		rr, err := c.Run(ctx, comp.FileID, t.Input, p.Limits)
		if err != nil {
			return Result{}, err
		}
		v := MapRunStatus(rr.Status)
		if v == IE {
			return Result{}, fmt.Errorf("%w: %s", ErrSandbox, rr.Status)
		}
		if v == AC && !Check(p.Checker, p.FloatEps, rr.Stdout, t.Expected) {
			v = WA
		}
		tr := TestResult{TestID: t.ID, Position: t.Position, IsSample: t.IsSample, Verdict: v, TimeMS: rr.TimeMS, MemoryKB: rr.MemoryKB, Weight: t.Weight}
		if v != AC && t.IsSample {
			tr.StdoutExcerpt = truncateUTF8(rr.Stdout, stdoutExcerptMax)
		}
		res.Results = append(res.Results, tr)
		if v == AC {
			res.PassedWeight += t.Weight
		} else if firstBad == "" {
			firstBad = v
		}
		res.TimeMSMax = max(res.TimeMSMax, rr.TimeMS)
		res.MemoryKBMax = max(res.MemoryKBMax, rr.MemoryKB)
	}
	res.Verdict = AC
	if firstBad != "" {
		res.Verdict = firstBad // verdict của test lỗi ĐẦU TIÊN theo position
	}
	return res, nil
}

func totalOf(ts []Test) int {
	n := 0
	for _, t := range ts {
		n += t.Weight
	}
	return n
}

// IsSystemError cho biết lỗi có phải lỗi hệ thống của sandbox (cần thử lại) không.
func IsSystemError(err error) bool { return errors.Is(err, ErrSandbox) }
