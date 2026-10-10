package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Lỗi của lớp đọc tài liệu.
var (
	// ErrExtractUnavailable: docling chết / không trả lời / mất việc giữa chừng — lỗi TẠM THỜI, thử lại được.
	ErrExtractUnavailable = errors.New("ingest: docling không dùng được")
	// ErrExtractFailed: docling đọc xong nhưng kết quả `failure` (tệp hỏng, vượt số trang…). Không thử lại.
	ErrExtractFailed = errors.New("ingest: docling không đọc được tệp")
)

// ExtractFailure mang lý do ngắn (đã cắt, không URL / đường dẫn) và cờ vượt số trang.
type ExtractFailure struct {
	Message  string
	PageSize bool // docling báo vượt MAX_NUM_PAGES
}

func (e *ExtractFailure) Error() string { return ErrExtractFailed.Error() + ": " + e.Message }
func (e *ExtractFailure) Unwrap() error { return ErrExtractFailed }

// Docling là client của docling-serve (hợp đồng chốt theo PoC, proposals #7): gọi bất đồng bộ MỘT lần cho cả tệp, poll, kiểm `status` trong thân kết quả.
type Docling struct {
	BaseURL string
	HTTP    *http.Client  // nil = http.DefaultClient
	Poll    time.Duration // mặc định 2 s
}

// Result là văn bản Markdown đọc được.
type Result struct{ Markdown string }

// ConvertOptions: OCR là lượt 2 cho bản scan.
type ConvertOptions struct {
	OCR             bool
	DocumentTimeout time.Duration
}

type taskResp struct {
	TaskID     string `json:"task_id"`
	TaskStatus string `json:"task_status"`
}

type resultResp struct {
	Document struct {
		MD string `json:"md_content"`
	} `json:"document"`
	Status string `json:"status"`
	Errors []struct {
		Message string `json:"error_message"`
	} `json:"errors"`
}

func (d *Docling) client() *http.Client {
	if d.HTTP != nil {
		return d.HTTP
	}
	return http.DefaultClient
}

// Convert gửi `body` (stream, không ghi đĩa) tới /v1/convert/file/async rồi chờ kết quả. progress (nếu có) được gọi mỗi lần poll.
func (d *Docling) Convert(ctx context.Context, filename string, body io.Reader, o ConvertOptions, progress func()) (Result, error) {
	id, err := d.submit(ctx, filename, body, o)
	if err != nil {
		return Result{}, err
	}
	poll := d.Poll
	if poll <= 0 {
		poll = 2 * time.Second
	}
	for {
		select {
		case <-ctx.Done():
			return Result{}, fmt.Errorf("%w: %w", ErrExtractUnavailable, ctx.Err())
		case <-time.After(poll):
		}
		var t taskResp
		code, err := d.getJSON(ctx, "/v1/status/poll/"+url.PathEscape(id), &t)
		if err != nil || code == http.StatusNotFound { // 404: server khởi động lại → mất việc
			return Result{}, fmt.Errorf("%w: poll", ErrExtractUnavailable)
		}
		if progress != nil {
			progress()
		}
		switch t.TaskStatus {
		case "success":
			return d.result(ctx, id)
		case "failure":
			return Result{}, &ExtractFailure{Message: "docling báo lỗi xử lý"}
		}
	}
}

func (d *Docling) submit(ctx context.Context, filename string, body io.Reader, o ConvertOptions) (string, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		fields := [][2]string{
			{"to_formats", "md"}, {"pdf_backend", "pypdfium2"}, {"do_table_structure", "true"}, {"table_mode", "fast"},
			{"include_images", "false"}, {"image_export_mode", "placeholder"}, {"md_page_break_placeholder", PageMarker},
		}
		if o.OCR {
			fields = append(fields, [2]string{"do_ocr", "true"}, [2]string{"ocr_preset", "tesseract"}, [2]string{"ocr_lang", "vie"})
		} else {
			fields = append(fields, [2]string{"do_ocr", "false"})
		}
		if o.DocumentTimeout > 0 {
			fields = append(fields, [2]string{"document_timeout", fmt.Sprint(int(o.DocumentTimeout.Seconds()))})
		}
		for _, f := range fields {
			if err := mw.WriteField(f[0], f[1]); err != nil { // WriteField (không phải curl -F): không hiểu `<` là "đọc tệp"
				_ = pw.CloseWithError(err)
				return
			}
		}
		fw, err := mw.CreateFormFile("files", filename)
		if err == nil {
			_, err = io.Copy(fw, body)
		}
		if err == nil {
			err = mw.Close()
		}
		_ = pw.CloseWithError(err)
	}()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(d.BaseURL, "/")+"/v1/convert/file/async", pr)
	if err != nil {
		return "", fmt.Errorf("ingest: dựng yêu cầu docling: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := d.client().Do(req)
	if err != nil {
		_ = pr.CloseWithError(err)
		return "", fmt.Errorf("%w: gửi tệp", ErrExtractUnavailable)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("%w: HTTP %d", ErrExtractUnavailable, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", &ExtractFailure{Message: "docling từ chối tệp", PageSize: strings.Contains(strings.ToLower(string(raw)), "page")}
	}
	var t taskResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&t); err != nil || t.TaskID == "" {
		return "", fmt.Errorf("%w: phản hồi gửi tệp", ErrExtractUnavailable)
	}
	return t.TaskID, nil
}

func (d *Docling) result(ctx context.Context, id string) (Result, error) {
	var r resultResp
	code, err := d.getJSON(ctx, "/v1/result/"+url.PathEscape(id), &r)
	if err != nil || code != http.StatusOK {
		return Result{}, fmt.Errorf("%w: lấy kết quả", ErrExtractUnavailable)
	}
	// task_status=success vẫn có thể đi với kết quả failure (nguồn bị chặn, tệp hỏng): kiểm status trong thân.
	if r.Status != "success" && r.Status != "partial_success" {
		var msgs []string
		for _, e := range r.Errors {
			msgs = append(msgs, e.Message)
		}
		msg := truncRunes(strings.Join(msgs, "; "), 1000)
		low := strings.ToLower(msg)
		return Result{}, &ExtractFailure{Message: msg, PageSize: strings.Contains(low, "page") && (strings.Contains(low, "max") || strings.Contains(low, "limit") || strings.Contains(low, "exceed"))}
	}
	return Result{Markdown: r.Document.MD}, nil
}

func (d *Docling) getJSON(ctx context.Context, path string, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(d.BaseURL, "/")+path, http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("ingest: dựng yêu cầu docling: %w", err)
	}
	resp, err := d.client().Do(req)
	if err != nil {
		return 0, fmt.Errorf("gọi docling: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return resp.StatusCode, nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256<<20)).Decode(out); err != nil {
		return resp.StatusCode, fmt.Errorf("đọc phản hồi docling: %w", err)
	}
	return resp.StatusCode, nil
}

// Ready gọi GET /ready (kiểm sức khoẻ).
func (d *Docling) Ready(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(d.BaseURL, "/")+"/ready", http.NoBody)
	if err != nil {
		return false
	}
	resp, err := d.client().Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
