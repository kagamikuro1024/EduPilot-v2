package server_test

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/cmd/mock-graph/server"
)

const (
	dataDir = "../../../testdata/mock-graph"
	docsDir = "../../../testdata/graph-docs/"
	planCSV = "../../../../docs/specs/FEAT-assignments/fixtures/teams-plan.csv"
)

type rig struct {
	t    *testing.T
	srv  *server.Server
	ts   *httptest.Server
	meta server.Meta
	now  time.Time
}

func newRig(t *testing.T) *rig {
	t.Helper()
	r := &rig{t: t, now: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)}
	srv, err := server.New(server.Options{DataDir: dataDir, SlowDelay: 120 * time.Millisecond, Now: func() time.Time { return r.now }})
	require.NoError(t, err)
	r.srv, r.meta = srv, srv.Meta()
	r.ts = httptest.NewServer(srv)
	t.Cleanup(r.ts.Close)
	return r
}

func (r *rig) do(method, path, token string, form url.Values) (int, http.Header, []byte) {
	r.t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(method, r.ts.URL+path, body)
	require.NoError(r.t, err)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(r.t, err)
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, b
}

func (r *rig) token() string {
	r.t.Helper()
	code, _, b := r.do(http.MethodPost, "/"+r.meta.TenantID+"/oauth2/v2.0/token", "", url.Values{"grant_type": {"client_credentials"}, "client_id": {r.meta.ClientID}, "client_secret": {r.meta.ClientSecret}, "scope": {"https://graph.microsoft.com/.default"}})
	require.Equal(r.t, 200, code, string(b))
	var tok struct {
		Access string `json:"access_token"`
	}
	require.NoError(r.t, json.Unmarshal(b, &tok))
	return tok.Access
}

func (r *rig) scenario(name string) {
	r.t.Helper()
	b, _ := json.Marshal(map[string]string{"name": name})
	req, _ := http.NewRequest(http.MethodPost, r.ts.URL+"/__mock/scenario", strings.NewReader(string(b)))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(r.t, err)
	_ = resp.Body.Close()
	require.Equal(r.t, 200, resp.StatusCode)
}

func (r *rig) subsPath() string {
	return "/v1.0/education/classes/" + r.meta.ClassID + "/assignments/" + r.meta.AssignmentID + "/submissions?$expand=submittedResources"
}

type pageOut struct {
	Value []map[string]any `json:"value"`
	Next  string           `json:"@odata.nextLink"`
}

func (r *rig) page(path, tok string) (int, pageOut) {
	r.t.Helper()
	code, _, b := r.do(http.MethodGet, path, tok, nil)
	var p pageOut
	_ = json.Unmarshal(b, &p)
	return code, p
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func doc(t *testing.T, name string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(docsDir + name)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	_, isSource := m["_source"].(string)
	require.True(t, isSource, name+": thiếu _source")
	delete(m, "_source")
	return m
}

// sameShape: cùng tập khoá, đệ quy xuống đối tượng và phần tử đầu của mảng.
func sameShape(t *testing.T, path string, want, got any) {
	t.Helper()
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		require.True(t, ok, path)
		require.Equal(t, keys(w), keys(g), path)
		for k := range w {
			sameShape(t, path+"."+k, w[k], g[k])
		}
	case []any:
		g, ok := got.([]any)
		require.True(t, ok, path)
		if len(w) > 0 && len(g) > 0 {
			sameShape(t, path+"[0]", w[0], g[0])
		}
	}
}

// TestMockGraphMatchesDocs — AC4: khoá JSON của phản hồi mock khớp từng chữ với ví dụ chép từ tài liệu Microsoft (testdata/graph-docs).
func TestMockGraphMatchesDocs(t *testing.T) {
	r := newRig(t)
	tok := r.token()
	code, p := r.page(r.subsPath(), tok)
	require.Equal(t, 200, code)
	var submitted, working map[string]any
	for _, v := range p.Value {
		switch v["status"] {
		case "submitted":
			if submitted == nil {
				submitted = v
			}
		case "working":
			if working == nil {
				working = v
			}
		}
	}
	want := doc(t, "list-submissions-expand-submittedresources.json")["value"].([]any)
	sameShape(t, "submission(expand)", want[0], any(submitted))
	plain := doc(t, "list-submissions.json")["value"].([]any)
	_, _, raw := r.do(http.MethodGet, strings.Split(r.subsPath(), "?")[0], tok, nil)
	var np pageOut
	require.NoError(t, json.Unmarshal(raw, &np))
	for _, v := range np.Value {
		if v["status"] == "working" {
			sameShape(t, "submission(không expand)", plain[0], any(v))
			break
		}
	}
	// thành viên: ví dụ tài liệu + `mail` (educationUser có thuộc tính mail; adapter dùng mail rồi userPrincipalName)
	_, mp := r.page("/v1.0/education/classes/"+r.meta.ClassID+"/members", tok)
	wm := doc(t, "list-class-members.json")["value"].([]any)[1].(map[string]any)
	gm := mp.Value[0]
	delete(gm, "mail")
	sameShape(t, "member", wm, any(gm))
	// token
	_, _, tb := r.do(http.MethodPost, "/"+r.meta.TenantID+"/oauth2/v2.0/token", "", url.Values{"grant_type": {"client_credentials"}, "client_id": {r.meta.ClientID}, "client_secret": {r.meta.ClientSecret}})
	var tm map[string]any
	require.NoError(t, json.Unmarshal(tb, &tm))
	sameShape(t, "token", doc(t, "token.json"), any(tm))
	// lớp
	_, lp := r.page("/v1.0/education/classes", tok)
	sameShape(t, "class", doc(t, "list-classes.json")["value"].([]any)[0], any(lp.Value[0]))
}

// TestMockTokenExpires60s — AC4: expires_in = 60; sau 61 giây token không còn dùng được (401) nên luồng hết hạn xảy ra thật.
func TestMockTokenExpires60s(t *testing.T) {
	r := newRig(t)
	code, _, b := r.do(http.MethodPost, "/"+r.meta.TenantID+"/oauth2/v2.0/token", "", url.Values{"grant_type": {"client_credentials"}, "client_id": {r.meta.ClientID}, "client_secret": {r.meta.ClientSecret}})
	require.Equal(t, 200, code)
	var tok struct {
		Access  string `json:"access_token"`
		Expires int    `json:"expires_in"`
	}
	require.NoError(t, json.Unmarshal(b, &tok))
	require.Equal(t, 60, tok.Expires)
	c, _ := r.page("/v1.0/education/classes", tok.Access)
	require.Equal(t, 200, c)
	r.now = r.now.Add(61 * time.Second)
	c, _ = r.page("/v1.0/education/classes", tok.Access)
	require.Equal(t, 401, c)
	// tenant / client sai
	code, _, _ = r.do(http.MethodPost, "/sai-tenant/oauth2/v2.0/token", "", url.Values{"grant_type": {"client_credentials"}, "client_id": {r.meta.ClientID}, "client_secret": {r.meta.ClientSecret}})
	require.Equal(t, 400, code)
	code, _, _ = r.do(http.MethodPost, "/"+r.meta.TenantID+"/oauth2/v2.0/token", "", url.Values{"grant_type": {"client_credentials"}, "client_id": {r.meta.ClientID}, "client_secret": {"sai"}})
	require.Equal(t, 401, code)
}

// TestMockDataFromPlan — AC4: dữ liệu sinh từ teams-plan.csv: 30 thành viên, 25 submitted (gồm 1 email ngoài lớp), 3 working, 2 returned; id = graph_user_id.
func TestMockDataFromPlan(t *testing.T) {
	f, err := os.Open(planCSV)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	rows, err := csv.NewReader(f).ReadAll()
	require.NoError(t, err)
	rows = rows[1:]
	r := newRig(t)
	tok := r.token()
	_, mp := r.page("/v1.0/education/classes/"+r.meta.ClassID+"/members", tok)
	require.Len(t, mp.Value, 30)
	_, sp := r.page(r.subsPath(), tok)
	require.Len(t, sp.Value, 30)
	status := map[string]int{}
	byUser := map[string]string{}
	for _, v := range sp.Value {
		status[v["status"].(string)]++
		byUser[v["recipient"].(map[string]any)["userId"].(string)] = v["status"].(string)
	}
	require.Equal(t, map[string]int{"submitted": 25, "working": 3, "returned": 2}, status)
	outside := 0
	for i, row := range rows {
		require.Equal(t, row[1], mp.Value[i]["id"])
		require.Equal(t, row[2], mp.Value[i]["mail"])
		require.Equal(t, row[3], byUser[row[1]])
		if !strings.HasSuffix(row[2], "@edupilot.local") {
			outside++
		}
	}
	require.Equal(t, 1, outside)
}

func (r *rig) requests() (log struct {
	Scenario      string `json:"scenario"`
	TokenRequests int    `json:"token_requests"`
	Throttled     int    `json:"throttled"`
	Requests      []struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Status int    `json:"status"`
	} `json:"requests"`
}) {
	r.t.Helper()
	_, _, b := r.do(http.MethodGet, "/__mock/requests", "", nil)
	require.NoError(r.t, json.Unmarshal(b, &log))
	return log
}

func (r *rig) all(tok string) (pages int, n int) {
	next := r.subsPath()
	for next != "" {
		code, p := r.page(strings.TrimPrefix(next, r.ts.URL), tok)
		if code != 200 {
			return pages, n
		}
		pages++
		n += len(p.Value)
		next = p.Next
	}
	return pages, n
}

func TestMockScenarioHappyAndPaged(t *testing.T) {
	r := newRig(t)
	tok := r.token()
	pages, n := r.all(tok)
	require.Equal(t, 1, pages)
	require.Equal(t, 30, n)
	r.scenario("paged")
	tok = r.token()
	pages, n = r.all(tok)
	require.Equal(t, 3, pages)
	require.Equal(t, 30, n)
	_, p := r.page(r.subsPath(), tok)
	require.True(t, strings.HasPrefix(p.Next, r.ts.URL), "nextLink tuyệt đối, cùng host")
	require.Len(t, p.Value, 10)
	_, mp := r.page("/v1.0/education/classes/"+r.meta.ClassID+"/members", tok)
	require.Len(t, mp.Value, 10)
	require.NotEmpty(t, mp.Next)
}

func TestMockScenarioThrottle(t *testing.T) {
	r := newRig(t)
	r.scenario("throttle")
	tok := r.token()
	for range 2 {
		code, h, _ := r.do(http.MethodGet, "/v1.0/education/classes", tok, nil)
		require.Equal(t, 429, code)
		require.Equal(t, "1", h.Get("Retry-After"))
	}
	code, _ := r.page("/v1.0/education/classes", tok)
	require.Equal(t, 200, code)
	require.Equal(t, 2, r.requests().Throttled)
}

func TestMockScenarioExpiredToken(t *testing.T) {
	r := newRig(t)
	r.scenario("expired_token")
	t1 := r.token()
	c, _ := r.page("/v1.0/education/classes", t1)
	require.Equal(t, 200, c)
	c, _ = r.page("/v1.0/education/classes", t1)
	require.Equal(t, 401, c, "yêu cầu thứ hai của token đầu")
	c, _ = r.page("/v1.0/education/classes", t1)
	require.Equal(t, 401, c, "token đầu đã chết")
	t2 := r.token()
	pages, n := r.all(t2)
	require.Equal(t, 1, pages)
	require.Equal(t, 30, n)
	require.Equal(t, 2, r.requests().TokenRequests)
}

func TestMockScenarioUnauthorizedAndNoConsent(t *testing.T) {
	r := newRig(t)
	r.scenario("unauthorized")
	c, _ := r.page("/v1.0/education/classes", r.token())
	require.Equal(t, 401, c)
	r.scenario("no_consent")
	code, _, b := r.do(http.MethodGet, "/v1.0/education/classes", r.token(), nil)
	require.Equal(t, 403, code)
	require.Contains(t, string(b), "Authorization_RequestDenied")
	code, _, _ = r.do(http.MethodGet, "/v1.0/education/classes", "", nil)
	require.Equal(t, 401, code, "không có token")
}

func TestMockScenarioSlow(t *testing.T) {
	r := newRig(t)
	r.scenario("slow")
	start := time.Now()
	tok := r.token()
	r.page("/v1.0/education/classes", tok)
	require.GreaterOrEqual(t, time.Since(start), 240*time.Millisecond, "token và mỗi yêu cầu đều trễ SlowDelay")
}

func TestMockScenarioPartialFailureAndFiles(t *testing.T) {
	r := newRig(t)
	tok := r.token()
	_, sp := r.page(r.subsPath(), tok)
	fileOf := func(account string) string {
		for _, v := range sp.Value {
			uid := v["recipient"].(map[string]any)["userId"].(string)
			_, mp := r.page("/v1.0/education/classes/"+r.meta.ClassID+"/members", tok)
			for _, m := range mp.Value {
				if m["id"] == uid && m["displayName"] == account && v["status"] == "submitted" {
					return v["submittedResources"].([]any)[0].(map[string]any)["resource"].(map[string]any)["fileUrl"].(string)
				}
			}
		}
		return ""
	}
	u14, u1 := fileOf("sv14"), fileOf("sv.gioi")
	require.NotEmpty(t, u14)
	require.True(t, strings.HasPrefix(u14, r.ts.URL+"/files/"))
	code, _, b := r.do(http.MethodGet, strings.TrimPrefix(u1, r.ts.URL), tok, nil)
	require.Equal(t, 200, code)
	require.NotEmpty(t, b)
	r.scenario("partial_failure")
	tok = r.token()
	code, _, _ = r.do(http.MethodGet, strings.TrimPrefix(u14, r.ts.URL), tok, nil)
	require.Equal(t, 500, code)
	code, _, _ = r.do(http.MethodGet, strings.TrimPrefix(u1, r.ts.URL), tok, nil)
	require.Equal(t, 200, code)
	r.scenario("happy")
	code, _, _ = r.do(http.MethodGet, strings.TrimPrefix(u14, r.ts.URL), r.token(), nil)
	require.Equal(t, 200, code, "hết lỗi")
	code, _, _ = r.do(http.MethodPost, "/__mock/scenario", "", nil)
	require.Equal(t, 400, code)
}

// TestMockRequestLog — nhật ký ghi phương thức, đường dẫn, mã trả, số lần xin token.
func TestMockRequestLog(t *testing.T) {
	r := newRig(t)
	tok := r.token()
	r.all(tok)
	log := r.requests()
	require.Equal(t, "happy", log.Scenario)
	require.Equal(t, 1, log.TokenRequests)
	require.Len(t, log.Requests, 2)
	require.Equal(t, http.MethodPost, log.Requests[0].Method)
	require.Equal(t, 200, log.Requests[1].Status)
}
