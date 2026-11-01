// Package source chứa các nguồn bài nộp gọi dịch vụ ngoài. teams.go là MỘT đường mã cho Microsoft Graph: bản chạy thật và bản chạy thử chỉ khác địa chỉ
// (GRAPH_BASE_URL, GRAPH_AUTH_URL) — không có nhánh theo môi trường (US-P7-11 AC5).
package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Mã lỗi của nguồn Teams (SRS 4.11.2).
const (
	CodeAuthFailed      = "TEAMS_AUTH_FAILED"
	CodeConsentRequired = "TEAMS_CONSENT_REQUIRED"
	CodeResponseInvalid = "TEAMS_RESPONSE_INVALID"
	CodeUnreachable     = "UNREACHABLE"
	CodeTimeout         = "TIMEOUT"
	CodeFileDownload    = "FILE_DOWNLOAD_FAILED"
	CodeFileHost        = "FILE_HOST_NOT_ALLOWED"
	CodeNoFile          = "NO_FILE"
)

// Error là lỗi đã phân loại (Code ∈ Code*). Err không bao giờ chứa token hay bí mật.
type Error struct {
	Code string
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Code
	}
	return e.Code + ": " + e.Err.Error()
}
func (e *Error) Unwrap() error { return e.Err }

func fail(code string, err error) *Error { return &Error{Code: code, Err: err} }

var errRedirectDenied = errors.New("chuyển hướng tới host không được phép")

// Sleeper chờ `d` (huỷ được bằng ctx); test tiêm bản không ngủ thật.
type Sleeper interface {
	Sleep(ctx context.Context, d time.Duration) error
}

// RealSleeper ngủ thật.
type RealSleeper struct{}

// Sleep chờ d hoặc tới khi ctx kết thúc.
func (RealSleeper) Sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Config là cấu hình địa chỉ + giới hạn (từ GRAPH_* và TEAMS_*).
type Config struct {
	BaseURL          string        // GRAPH_BASE_URL, ví dụ https://graph.microsoft.com/v1.0
	AuthURL          string        // GRAPH_AUTH_URL, ví dụ https://login.microsoftonline.com
	FileHosts        []string      // GRAPH_FILE_HOSTS; rỗng = chỉ host của BaseURL
	DownloadParallel int           // TEAMS_DOWNLOAD_PARALLEL (4)
	FileRetry        int           // TEAMS_FILE_RETRY (2)
	RetryMax         int           // TEAMS_RETRY_MAX (3)
	MaxFileBytes     int64         // trần cỡ một tệp (ASSIGN_MAX_BYTES)
	Scope            string        // mặc định https://graph.microsoft.com/.default
	FileRetryWait    time.Duration // chờ giữa các lần tải lại (mặc định 1 giây)
}

// Credentials là thông tin ứng dụng (client_secret chỉ ở bộ nhớ).
type Credentials struct {
	Tenant, ClientID, ClientSecret string
}

// Fetched là một bài nộp đã tải xong tệp.
type Fetched struct {
	SubmissionID string
	SubmittedAt  time.Time
	UserID       string
	Email        string
	FileName     string
	Data         []byte
}

// Failure là lỗi riêng của một bài (đợt vẫn tiếp tục).
type Failure struct {
	SubmissionID string `json:"graph_submission_id"`
	Code         string `json:"code"`
}

// Report là kết quả một đợt: số bài `submitted` liệt kê được và các bài lỗi riêng.
type Report struct {
	Listed int
	Failed []Failure
}

// Teams là nguồn Teams qua Microsoft Graph.
type Teams struct {
	Cfg   Config
	HTTP  *http.Client // nil → client có hạn 60 giây
	Sleep Sleeper      // nil → ngủ thật
	Now   func() time.Time

	mu     sync.Mutex
	tokens map[string]token
}

type token struct {
	value   string
	expires time.Time
}

func (t *Teams) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

func (t *Teams) sleeper() Sleeper {
	if t.Sleep != nil {
		return t.Sleep
	}
	return RealSleeper{}
}

func (t *Teams) client() *http.Client {
	if t.HTTP != nil {
		return t.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (t *Teams) base() *url.URL {
	u, _ := url.Parse(strings.TrimRight(t.Cfg.BaseURL, "/"))
	return u
}

// allowedHost: host được tải tệp (GRAPH_FILE_HOSTS; rỗng = host của BaseURL).
func (t *Teams) allowedHost(h string) bool {
	h = strings.ToLower(h)
	if len(t.Cfg.FileHosts) == 0 {
		return h == strings.ToLower(t.base().Host)
	}
	for _, a := range t.Cfg.FileHosts {
		if strings.EqualFold(strings.TrimSpace(a), h) {
			return true
		}
	}
	return false
}

func (c Credentials) key() string { return c.Tenant + "|" + c.ClientID }

// accessToken trả token còn hạn (xin mới khi hết hạn − 10 giây hoặc khi `renew`); chỉ giữ ở bộ nhớ.
func (t *Teams) accessToken(ctx context.Context, c Credentials, renew bool) (string, error) {
	t.mu.Lock()
	cur, ok := t.tokens[c.key()]
	t.mu.Unlock()
	if ok && !renew && t.now().Before(cur.expires.Add(-10*time.Second)) {
		return cur.value, nil
	}
	scope := t.Cfg.Scope
	if scope == "" {
		scope = "https://graph.microsoft.com/.default"
	}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {c.ClientID}, "client_secret": {c.ClientSecret}, "scope": {scope}}
	u := strings.TrimRight(t.Cfg.AuthURL, "/") + "/" + url.PathEscape(c.Tenant) + "/oauth2/v2.0/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fail(CodeUnreachable, errors.New("địa chỉ xin token không hợp lệ"))
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client().Do(req)
	if err != nil {
		return "", netError(err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return "", fail(CodeAuthFailed, fmt.Errorf("máy chủ xác thực từ chối (%d)", resp.StatusCode))
	case resp.StatusCode != http.StatusOK:
		return "", fail(CodeUnreachable, fmt.Errorf("máy chủ xác thực trả %d", resp.StatusCode))
	}
	var body struct {
		Access  string `json:"access_token"`
		Expires int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil || body.Access == "" || body.Expires <= 0 {
		return "", fail(CodeResponseInvalid, errors.New("phản hồi xin token thiếu trường"))
	}
	tok := token{value: body.Access, expires: t.now().Add(time.Duration(body.Expires) * time.Second)}
	t.mu.Lock()
	if t.tokens == nil {
		t.tokens = map[string]token{}
	}
	t.tokens[c.key()] = tok
	t.mu.Unlock()
	return tok.value, nil
}

func netError(err error) *Error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fail(CodeTimeout, errors.New("quá thời gian đồng bộ"))
	case errors.Is(err, context.Canceled):
		return fail(CodeUnreachable, errors.New("đã huỷ"))
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return fail(CodeTimeout, errors.New("máy chủ không phản hồi kịp"))
	}
	return fail(CodeUnreachable, errors.New("không kết nối được máy chủ"))
}

// call GET một địa chỉ Graph có token: 429 → chờ Retry-After (mặc định 1, trần 30) rồi thử lại ≤ RetryMax; 401 → xin token mới MỘT lần rồi thử lại, 401 nữa → TEAMS_AUTH_FAILED;
// 403 → TEAMS_CONSENT_REQUIRED. Trả thân phản hồi 200 (đã giới hạn cỡ).
func (t *Teams) call(ctx context.Context, c Credentials, u string) ([]byte, error) {
	renewed, throttled := false, 0
	for {
		tok, err := t.accessToken(ctx, c, false)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, fail(CodeResponseInvalid, errors.New("địa chỉ không hợp lệ"))
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Accept", "application/json")
		resp, err := t.client().Do(req)
		if err != nil {
			return nil, netError(err)
		}
		body, rerr := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		_ = resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			if rerr != nil {
				return nil, netError(rerr)
			}
			return body, nil
		case http.StatusTooManyRequests:
			if throttled >= t.retryMax() {
				return nil, fail(CodeUnreachable, errors.New("máy chủ vẫn giới hạn tần suất sau khi thử lại"))
			}
			throttled++
			if err := t.sleeper().Sleep(ctx, retryAfter(resp.Header.Get("Retry-After"))); err != nil {
				return nil, netError(err)
			}
		case http.StatusUnauthorized:
			if renewed {
				return nil, fail(CodeAuthFailed, errors.New("token mới vẫn bị từ chối"))
			}
			renewed = true
			if _, err := t.accessToken(ctx, c, true); err != nil {
				return nil, err
			}
		case http.StatusForbidden:
			return nil, fail(CodeConsentRequired, errors.New("chưa được cấp quyền"))
		default:
			return nil, fail(CodeUnreachable, fmt.Errorf("máy chủ trả %d", resp.StatusCode))
		}
	}
}

func (t *Teams) retryMax() int {
	if t.Cfg.RetryMax > 0 {
		return t.Cfg.RetryMax
	}
	return 3
}

func retryAfter(h string) time.Duration {
	n, err := strconv.Atoi(strings.TrimSpace(h))
	if err != nil || n < 1 {
		n = 1
	}
	return time.Duration(min(n, 30)) * time.Second
}

// sameHost: nextLink phải cùng host với GRAPH_BASE_URL.
func (t *Teams) sameHost(next string) bool {
	u, err := url.Parse(next)
	return err == nil && strings.EqualFold(u.Host, t.base().Host)
}

type page struct {
	Value []json.RawMessage `json:"value"`
	Next  string            `json:"@odata.nextLink"`
}

// pages đi hết `@odata.nextLink` (cùng host) và gọi fn cho từng phần tử.
func (t *Teams) pages(ctx context.Context, c Credentials, first string, fn func(json.RawMessage) error) error {
	next := first
	for hops := 0; next != "" && hops < 1000; hops++ {
		raw, err := t.call(ctx, c, next)
		if err != nil {
			return err
		}
		var p page
		if err := json.Unmarshal(raw, &p); err != nil || p.Value == nil {
			return fail(CodeResponseInvalid, errors.New("thiếu trường value"))
		}
		for _, v := range p.Value {
			if err := fn(v); err != nil {
				return err
			}
		}
		next = p.Next
		if next != "" && !t.sameHost(next) {
			return fail(CodeResponseInvalid, errors.New("nextLink trỏ sang host khác"))
		}
	}
	return nil
}

type member struct {
	ID                string `json:"id"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

// members: MỘT lệnh phân trang cho cả lớp → bảng id → email (mail, rồi userPrincipalName).
func (t *Teams) members(ctx context.Context, c Credentials, class string) (map[string]string, error) {
	out := map[string]string{}
	err := t.pages(ctx, c, t.Cfg.BaseURL+"/education/classes/"+url.PathEscape(class)+"/members", func(raw json.RawMessage) error {
		var m member
		if err := json.Unmarshal(raw, &m); err != nil || m.ID == "" {
			return fail(CodeResponseInvalid, errors.New("thành viên thiếu id"))
		}
		mail := strings.ToLower(strings.TrimSpace(m.Mail))
		if mail == "" {
			mail = strings.ToLower(strings.TrimSpace(m.UserPrincipalName))
		}
		out[m.ID] = mail
		return nil
	})
	return out, err
}

type resource struct {
	Resource struct {
		DisplayName string `json:"displayName"`
		FileURL     string `json:"fileUrl"`
	} `json:"resource"`
}

type submission struct {
	ID                string     `json:"id"`
	Status            string     `json:"status"`
	SubmittedDateTime *time.Time `json:"submittedDateTime"`
	Recipient         struct {
		UserID string `json:"userId"`
	} `json:"recipient"`
	SubmittedResources []resource `json:"submittedResources"`
}

// pending là một bài `submitted` chờ tải tệp.
type pending struct {
	sub   submission
	email string
	res   resource
}

// Sync đi chuỗi Graph của SRS 4.11.1: token → thành viên → bài nộp (`$expand=submittedResources`) → tải tệp song song. `onFile` được gọi tuần tự từng bài đã tải xong;
// trả lỗi từ onFile thì dừng đợt. Lỗi tải tệp của một bài thành Failure riêng, đợt vẫn tiếp tục. `progress(done, total)` báo số bài đã xử lý / tổng số bài `submitted`.
func (t *Teams) Sync(ctx context.Context, c Credentials, classID, assignmentID string, onFile func(context.Context, Fetched) error, progress func(done, total int)) (Report, error) {
	var rep Report
	mail, err := t.members(ctx, c, classID)
	if err != nil {
		return rep, err
	}
	var todo []pending
	listURL := t.Cfg.BaseURL + "/education/classes/" + url.PathEscape(classID) + "/assignments/" + url.PathEscape(assignmentID) + "/submissions?$expand=submittedResources"
	err = t.pages(ctx, c, listURL, func(raw json.RawMessage) error {
		var s submission
		if err := json.Unmarshal(raw, &s); err != nil || s.ID == "" || s.Status == "" {
			return fail(CodeResponseInvalid, errors.New("bài nộp thiếu id hoặc status"))
		}
		if s.Status != "submitted" { // bỏ working, returned, reassigned
			return nil
		}
		if s.SubmittedDateTime == nil || s.Recipient.UserID == "" {
			return fail(CodeResponseInvalid, errors.New("bài nộp thiếu submittedDateTime hoặc recipient.userId"))
		}
		p := pending{sub: s, email: mail[s.Recipient.UserID]}
		for _, r := range s.SubmittedResources {
			if r.Resource.FileURL != "" && r.Resource.DisplayName != "" {
				p.res = r
				break
			}
		}
		todo = append(todo, p)
		return nil
	})
	if err != nil {
		return rep, err
	}
	rep.Listed = len(todo)
	return t.download(ctx, c, todo, onFile, progress, rep)
}

type outcome struct {
	p    pending
	data []byte
	err  *Error
}

func (t *Teams) download(ctx context.Context, c Credentials, todo []pending, onFile func(context.Context, Fetched) error, progress func(done, total int), rep Report) (Report, error) {
	par := t.Cfg.DownloadParallel
	if par < 1 {
		par = 4
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan pending)
	results := make(chan outcome)
	var wg sync.WaitGroup
	for range par {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				o := outcome{p: p}
				if p.res.Resource.FileURL == "" {
					o.err = fail(CodeNoFile, nil)
				} else {
					o.data, o.err = t.fetchFile(ctx, c, p.res.Resource.FileURL)
				}
				select {
				case results <- o:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, p := range todo {
			select {
			case jobs <- p:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() { wg.Wait(); close(results) }()
	done := 0
	var firstErr error
	for o := range results {
		done++
		switch {
		case firstErr != nil:
		case o.err != nil && (o.err.Code == CodeTimeout || o.err.Code == CodeAuthFailed || o.err.Code == CodeConsentRequired || o.err.Code == CodeUnreachable):
			firstErr = o.err
			cancel()
		case o.err != nil:
			rep.Failed = append(rep.Failed, Failure{SubmissionID: o.p.sub.ID, Code: o.err.Code})
		default:
			err := onFile(ctx, Fetched{SubmissionID: o.p.sub.ID, SubmittedAt: o.p.sub.SubmittedDateTime.UTC(), UserID: o.p.sub.Recipient.UserID, Email: o.p.email,
				FileName: path.Base(o.p.res.Resource.DisplayName), Data: o.data})
			if err != nil {
				firstErr = err
				cancel()
			}
		}
		if progress != nil {
			progress(done, len(todo))
		}
	}
	if firstErr == nil && ctx.Err() != nil && !errors.Is(ctx.Err(), context.Canceled) {
		firstErr = netError(ctx.Err())
	}
	return rep, firstErr
}

// fetchFile tải một tệp: chỉ host trong danh sách cho phép; chuyển hướng chỉ tới host được liệt kê (≤ 5 lần) và KHÔNG gửi Authorization sang host khác host Graph;
// 5xx / lỗi mạng → thử lại FileRetry lần rồi FILE_DOWNLOAD_FAILED; 429 → chờ Retry-After.
func (t *Teams) fetchFile(ctx context.Context, c Credentials, fileURL string) ([]byte, *Error) {
	u, err := url.Parse(fileURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || !t.allowedHost(u.Host) {
		return nil, fail(CodeFileHost, nil)
	}
	graphHost := strings.ToLower(t.base().Host)
	cl := *t.client()
	cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !t.allowedHost(req.URL.Host) {
			return errRedirectDenied
		}
		if !strings.EqualFold(req.URL.Host, graphHost) {
			req.Header.Del("Authorization")
		}
		return nil
	}
	attempts, wait := 0, t.Cfg.FileRetryWait
	if wait <= 0 {
		wait = time.Second
	}
	throttled := 0
	lastNet := false
	for {
		tok, terr := t.accessToken(ctx, c, false)
		if terr != nil {
			var e *Error
			if errors.As(terr, &e) {
				return nil, e
			}
			return nil, fail(CodeUnreachable, nil)
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
		if strings.EqualFold(u.Host, graphHost) {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		resp, derr := cl.Do(req)
		retry := false
		lastNet = false
		var body []byte
		switch {
		case errors.Is(derr, errRedirectDenied):
			return nil, fail(CodeFileHost, nil)
		case derr != nil:
			ne := netError(derr)
			if ne.Code == CodeTimeout && ctx.Err() != nil {
				return nil, ne
			}
			retry, lastNet = true, true
		case resp.StatusCode == http.StatusOK:
			var rerr error
			body, rerr = io.ReadAll(io.LimitReader(resp.Body, t.maxFile()+1))
			_ = resp.Body.Close()
			switch {
			case rerr != nil:
				retry = true
			case int64(len(body)) > t.maxFile():
				return nil, fail(CodeFileDownload, errors.New("tệp quá cỡ"))
			default:
				return body, nil
			}
		case resp.StatusCode == http.StatusTooManyRequests:
			_ = resp.Body.Close()
			if throttled >= t.retryMax() {
				return nil, fail(CodeFileDownload, nil)
			}
			throttled++
			if err := t.sleeper().Sleep(ctx, retryAfter(resp.Header.Get("Retry-After"))); err != nil {
				return nil, netError(err)
			}
			continue
		case resp.StatusCode >= 500:
			_ = resp.Body.Close()
			retry = true
		case resp.StatusCode == http.StatusForbidden:
			_ = resp.Body.Close()
			return nil, fail(CodeConsentRequired, nil)
		case resp.StatusCode == http.StatusUnauthorized:
			_ = resp.Body.Close()
			if _, rerr := t.accessToken(ctx, c, true); rerr != nil {
				var e *Error
				if errors.As(rerr, &e) {
					return nil, e
				}
			}
			retry = true
		default:
			_ = resp.Body.Close()
			return nil, fail(CodeFileDownload, nil)
		}
		if !retry || attempts >= t.Cfg.FileRetry {
			if lastNet { // mạng đứt chứ không phải lỗi riêng của tệp: dừng đợt (UNREACHABLE)
				return nil, fail(CodeUnreachable, nil)
			}
			return nil, fail(CodeFileDownload, nil)
		}
		attempts++
		if err := t.sleeper().Sleep(ctx, wait); err != nil {
			return nil, netError(err)
		}
	}
}

func (t *Teams) maxFile() int64 {
	if t.Cfg.MaxFileBytes > 0 {
		return t.Cfg.MaxFileBytes
	}
	return 20 << 20
}

// Probe kiểm tra kết nối: xin token + `GET /education/classes`. nil = ok.
func (t *Teams) Probe(ctx context.Context, c Credentials) error {
	if _, err := t.accessToken(ctx, c, true); err != nil {
		return err
	}
	raw, err := t.call(ctx, c, t.Cfg.BaseURL+"/education/classes")
	if err != nil {
		return err
	}
	var p page
	if err := json.Unmarshal(raw, &p); err != nil || p.Value == nil {
		return fail(CodeResponseInvalid, errors.New("thiếu trường value"))
	}
	return nil
}
