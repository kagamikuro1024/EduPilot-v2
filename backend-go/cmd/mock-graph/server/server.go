// Package server là máy chủ Microsoft Graph GIẢ LẬP cho US-P7-11 (đồng bộ bài nộp từ Teams). Bám tên trường / chú thích @odata của tài liệu Microsoft
// (xem testdata/graph-docs); dữ liệu đọc từ testdata/mock-graph do `node scripts/seed.mjs --mock-graph` sinh từ fixtures/teams-plan.csv.
// Không nhập gói nào của internal/* và không gói nào của internal/* được nhập gói này.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Options là cấu hình của máy chủ.
type Options struct {
	DataDir   string
	TokenTTL  time.Duration // mặc định 60 giây (expires_in: 60 để luồng hết hạn xảy ra thật)
	SlowDelay time.Duration // mặc định 3 giây, chỉ ở kịch bản `slow`
	Now       func() time.Time
}

// Meta là thông tin tenant / ứng dụng giả.
type Meta struct {
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	ClassID      string `json:"class_id"`
	AssignmentID string `json:"assignment_id"`
}

type entry struct {
	Seq    int    `json:"seq"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query"`
	Status int    `json:"status"`
	Token  string `json:"token"`
	At     string `json:"at"`
}

type tokenInfo struct {
	expires time.Time
	dead    bool
}

// validScenario: tên kịch bản hợp lệ.
func validScenario(n string) bool {
	switch n {
	case "happy", "paged", "throttle", "expired_token", "no_consent", "slow", "partial_failure", "unauthorized":
		return true
	}
	return false
}

// Server là một máy chủ Graph giả; dùng như http.Handler.
type Server struct {
	opt     Options
	meta    Meta
	classes []map[string]any
	members []map[string]any
	assigns []map[string]any
	subs    []map[string]any
	files   map[string][]byte // theo id bài nộp
	sv14    string            // id bài nộp của sv14 (partial_failure)

	mu        sync.Mutex
	scenario  string
	log       []entry
	tokenReqs int
	throttled int
	graphReqs int
	tokens    map[string]*tokenInfo
	firstUsed string
	firstUses int
}

// New nạp dữ liệu và dựng máy chủ ở kịch bản `happy`.
func New(opt Options) (*Server, error) {
	if opt.TokenTTL <= 0 {
		opt.TokenTTL = 60 * time.Second
	}
	if opt.SlowDelay <= 0 {
		opt.SlowDelay = 3 * time.Second
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	s := &Server{opt: opt, scenario: "happy", tokens: map[string]*tokenInfo{}, files: map[string][]byte{}}
	read := func(name string, dst any) error {
		b, err := os.ReadFile(filepath.Join(opt.DataDir, name))
		if err != nil {
			return fmt.Errorf("mock-graph: đọc %s: %w", name, err)
		}
		return json.Unmarshal(b, dst)
	}
	for name, dst := range map[string]any{"meta.json": &s.meta, "classes.json": &s.classes, "members.json": &s.members, "assignments.json": &s.assigns, "submissions.json": &s.subs} {
		if err := read(name, dst); err != nil {
			return nil, err
		}
	}
	for _, sub := range s.subs {
		id, _ := sub["id"].(string)
		if b, err := os.ReadFile(filepath.Join(opt.DataDir, "files", id+".md")); err == nil {
			s.files[id] = b
		}
		if sub["account"] == "sv14" {
			s.sv14 = id
		}
	}
	return s, nil
}

// Close giải phóng tài nguyên (không có gì phải đóng; để chữ ký đối xứng với httptest).
func (s *Server) Close() {}

// Meta trả thông tin tenant / ứng dụng giả (test đọc để cấu hình adapter).
func (s *Server) Meta() Meta { return s.meta }

func (s *Server) reset(name string) {
	s.scenario, s.log, s.tokenReqs, s.throttled, s.graphReqs, s.firstUsed, s.firstUses = name, nil, 0, 0, 0, "", 0
}

func (s *Server) record(r *http.Request, status int, tok string) {
	if len(tok) > 8 {
		tok = tok[:8]
	}
	s.log = append(s.log, entry{Seq: len(s.log) + 1, Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Status: status, Token: tok, At: s.opt.Now().UTC().Format(time.RFC3339Nano)})
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(c int) { w.code = c; w.ResponseWriter.WriteHeader(c) }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func gerr(w http.ResponseWriter, code int, c, msg string) {
	writeJSON(w, code, map[string]any{"error": map[string]any{"code": c, "message": msg}})
}

// ServeHTTP định tuyến: điều khiển (/__mock), token, Graph (/v1.0), tệp (/files, /files-cdn).
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if strings.HasPrefix(p, "/__mock/") {
		s.control(w, r)
		return
	}
	sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
	tok := ""
	defer func() {
		s.mu.Lock()
		s.record(r, sw.code, tok)
		s.mu.Unlock()
	}()
	s.mu.Lock()
	scn := s.scenario
	s.mu.Unlock()
	if scn == "slow" {
		select {
		case <-time.After(s.opt.SlowDelay):
		case <-r.Context().Done():
			sw.code = 499
			return
		}
	}
	if strings.HasSuffix(p, "/oauth2/v2.0/token") && r.Method == http.MethodPost {
		s.token(sw, r)
		return
	}
	tok = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if code := s.gate(scn, tok, p); code != 0 {
		s.deny(sw, code)
		return
	}
	switch {
	case strings.HasPrefix(p, "/files/") || strings.HasPrefix(p, "/files-cdn/"):
		s.file(sw, r, scn)
	case strings.HasPrefix(p, "/v1.0/education/classes"):
		s.education(sw, r, scn)
	default:
		gerr(sw, http.StatusNotFound, "itemNotFound", "Resource not found.")
	}
}

// gate kiểm token và áp hành vi kịch bản chung; trả mã lỗi (0 = cho qua).
func (s *Server) gate(scn, tok, _ string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	ti := s.tokens[tok]
	if tok == "" || ti == nil || ti.dead || !s.opt.Now().Before(ti.expires) {
		return http.StatusUnauthorized
	}
	s.graphReqs++
	switch scn {
	case "unauthorized":
		return http.StatusUnauthorized
	case "no_consent":
		return http.StatusForbidden
	case "throttle":
		if s.graphReqs <= 2 {
			s.throttled++
			return http.StatusTooManyRequests
		}
	case "expired_token":
		if s.firstUsed == "" {
			s.firstUsed = tok
		}
		if tok == s.firstUsed {
			s.firstUses++
			if s.firstUses >= 2 { // yêu cầu thứ hai của token đầu
				ti.dead = true
				return http.StatusUnauthorized
			}
		}
	}
	return 0
}

func (s *Server) deny(w http.ResponseWriter, code int) {
	switch code {
	case http.StatusTooManyRequests:
		w.Header().Set("Retry-After", "1")
		gerr(w, code, "TooManyRequests", "Too many requests.")
	case http.StatusForbidden:
		gerr(w, code, "Authorization_RequestDenied", "Insufficient privileges to complete the operation.")
	default:
		gerr(w, code, "InvalidAuthenticationToken", "Access token is empty, expired or invalid.")
	}
}

func (s *Server) token(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.tokenReqs++
	s.mu.Unlock()
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_request"})
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch {
	case len(parts) != 4 || parts[0] != s.meta.TenantID:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_request", "error_description": "AADSTS90002: Tenant not found."})
		return
	case r.PostForm.Get("grant_type") != "client_credentials":
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unsupported_grant_type"})
		return
	case r.PostForm.Get("client_id") != s.meta.ClientID || r.PostForm.Get("client_secret") != s.meta.ClientSecret:
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid_client", "error_description": "AADSTS7000215: Invalid client secret provided."})
		return
	}
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	tok := "mock." + hex.EncodeToString(b)
	s.mu.Lock()
	s.tokens[tok] = &tokenInfo{expires: s.opt.Now().Add(s.opt.TokenTTL)}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"token_type": "Bearer", "expires_in": int(s.opt.TokenTTL.Seconds()), "ext_expires_in": int(s.opt.TokenTTL.Seconds()), "access_token": tok})
}

func (s *Server) control(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case r.URL.Path == "/__mock/scenario" && r.Method == http.MethodPost:
		var in struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<10)).Decode(&in); err != nil || !validScenario(in.Name) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "kịch bản không hợp lệ"})
			return
		}
		s.reset(in.Name)
		writeJSON(w, http.StatusOK, map[string]any{"scenario": in.Name})
	case r.URL.Path == "/__mock/requests" && r.Method == http.MethodGet:
		log := s.log
		if log == nil {
			log = []entry{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"scenario": s.scenario, "requests": log, "token_requests": s.tokenReqs, "throttled": s.throttled})
	case r.URL.Path == "/__mock/reset" && r.Method == http.MethodPost:
		s.reset("happy")
		s.tokens = map[string]*tokenInfo{}
		writeJSON(w, http.StatusOK, map[string]any{"scenario": "happy"})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) file(w http.ResponseWriter, r *http.Request, scn string) {
	id := path0(r.URL.Path)
	if scn == "partial_failure" && id == s.sv14 {
		gerr(w, http.StatusInternalServerError, "generalException", "General exception while processing.")
		return
	}
	b, ok := s.files[id]
	if !ok {
		gerr(w, http.StatusNotFound, "itemNotFound", "The specified file does not exist.")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	_, _ = w.Write(b)
}

func path0(p string) string { return p[strings.LastIndex(p, "/")+1:] }

// education phục vụ /v1.0/education/classes[/{id}/(members|assignments[/{aid}/submissions])].
func (s *Server) education(w http.ResponseWriter, r *http.Request, scn string) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1.0/education/classes"), "/")
	base := "http://" + r.Host + "/v1.0"
	if rest == "" {
		writeJSON(w, http.StatusOK, map[string]any{"value": s.classes})
		return
	}
	seg := strings.Split(rest, "/")
	if seg[0] != s.meta.ClassID {
		gerr(w, http.StatusNotFound, "itemNotFound", "The requested class was not found.")
		return
	}
	switch {
	case len(seg) == 2 && seg[1] == "members":
		s.paged(w, r, scn, base, "members", "#Collection(microsoft.graph.educationUser)", s.members, false)
	case len(seg) == 2 && seg[1] == "assignments":
		writeJSON(w, http.StatusOK, map[string]any{"@odata.context": "https://graph.microsoft.com/v1.0/$metadata#education/classes('" + s.meta.ClassID + "')/assignments", "value": s.assigns})
	case len(seg) == 4 && seg[1] == "assignments" && seg[2] == s.meta.AssignmentID && seg[3] == "submissions":
		s.paged(w, r, scn, base, "assignments/"+s.meta.AssignmentID+"/submissions", "#education/classes('"+s.meta.ClassID+"')/assignments('"+s.meta.AssignmentID+"')/submissions", s.subs, true)
	default:
		gerr(w, http.StatusNotFound, "itemNotFound", "Resource not found.")
	}
}

// paged cắt trang (kịch bản `paged`: 10 bản ghi / trang; hoặc `$top`) và dựng `@odata.nextLink` tuyệt đối cùng host.
func (s *Server) paged(w http.ResponseWriter, r *http.Request, scn, base, tail, ctx string, all []map[string]any, submissions bool) {
	q := r.URL.Query()
	size := len(all)
	if scn == "paged" {
		size = 10
	}
	if top, err := strconv.Atoi(q.Get("$top")); err == nil && top > 0 {
		size = min(size, top)
	}
	skip, _ := strconv.Atoi(q.Get("$skiptoken"))
	skip = max(0, min(skip, len(all)))
	end := min(len(all), skip+size)
	expand := q.Get("$expand")
	items := make([]map[string]any, 0, end-skip)
	for _, it := range all[skip:end] {
		items = append(items, s.shape(it, expand, base, r.Host, submissions))
	}
	out := map[string]any{"@odata.context": "https://graph.microsoft.com/v1.0/$metadata" + ctx, "value": items}
	if end < len(all) {
		nq := url.Values{"$skiptoken": {strconv.Itoa(end)}}
		for _, k := range []string{"$expand", "$top"} {
			if v := q.Get(k); v != "" {
				nq.Set(k, v)
			}
		}
		out["@odata.nextLink"] = base + "/education/classes/" + s.meta.ClassID + "/" + tail + "?" + nq.Encode()
	}
	writeJSON(w, http.StatusOK, out)
}

// shape bỏ trường riêng của mock (`account`), áp $expand và đổi fileUrl tương đối thành tuyệt đối trên chính host của yêu cầu.
func (s *Server) shape(it map[string]any, expand, _, host string, submissions bool) map[string]any {
	out := make(map[string]any, len(it))
	for k, v := range it {
		if k == "account" {
			continue
		}
		out[k] = v
	}
	if !submissions {
		return out
	}
	want := func(n string) bool { return strings.Contains(expand, n) || strings.Contains(expand, "*") }
	for _, k := range []string{"submittedResources", "resources"} {
		if !want(k) {
			delete(out, k)
			continue
		}
		list, _ := it[k].([]any)
		abs := make([]any, 0, len(list))
		for _, e := range list {
			m, _ := e.(map[string]any)
			cp := map[string]any{}
			for kk, vv := range m {
				cp[kk] = vv
			}
			if res, ok := m["resource"].(map[string]any); ok {
				rc := map[string]any{}
				for kk, vv := range res {
					rc[kk] = vv
				}
				if u, _ := rc["fileUrl"].(string); strings.HasPrefix(u, "/") {
					rc["fileUrl"] = "http://" + host + u
				}
				cp["resource"] = rc
			}
			abs = append(abs, cp)
		}
		out[k] = abs
		out[k+"@odata.context"] = "https://graph.microsoft.com/v1.0/$metadata#education/classes('" + s.meta.ClassID + "')/assignments('" + s.meta.AssignmentID + "')/submissions('" + fmt.Sprint(it["id"]) + "')/" + k
	}
	if u, _ := out["resourcesFolderUrl"].(string); strings.HasPrefix(u, "/") {
		out["resourcesFolderUrl"] = "http://" + host + u
	}
	return out
}
