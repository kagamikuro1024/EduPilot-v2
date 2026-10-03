//go:build testroutes

package httpapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/google/uuid"
)

const itemsURL = "/api/v1/_test/items"

// tList gọi GET /_test/items và trả phản hồi + thân đã giải mã.
func tList(t *testing.T, token, query string) (*http.Response, map[string]any) {
	t.Helper()
	resp := tDo(t, http.MethodGet, itemsURL+query, token, "")
	raw := tBody(t, resp)
	var body map[string]any
	if resp.StatusCode != http.StatusNotModified {
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("thân danh sách không phải JSON: %v (%s)", err, raw)
		}
	}
	return resp, body
}

// tIDs lấy danh sách id của một trang.
func tIDs(t *testing.T, body map[string]any) []string {
	t.Helper()
	items, _ := body["items"].([]any)
	out := make([]string, 0, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		out = append(out, fmt.Sprint(m["id"]))
	}
	return out
}

// tWalk đi hết các trang với limit cho trước và trả mọi id đọc được (theo thứ tự).
func tWalk(t *testing.T, token string, limit int) []string {
	t.Helper()
	var (
		all    []string
		cursor string
	)
	for page := 0; page < 80; page++ {
		query := fmt.Sprintf("?limit=%d", limit)
		if cursor != "" {
			query += "&cursor=" + cursor
		}
		resp, body := tList(t, token, query)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("trang %d → %d", page, resp.StatusCode)
		}
		all = append(all, tIDs(t, body)...)
		next, ok := body["next_cursor"].(string)
		if !ok || next == "" {
			return all
		}
		cursor = next
	}
	t.Fatal("đi quá 80 trang — nghi vòng lặp cursor")
	return nil
}

// TestPagination_Defaults: mặc định 30 mục, còn trang thì có next_cursor (03-AC8).
func TestPagination_Defaults(t *testing.T) {
	t.Parallel()
	owner, tok := tUser(t)
	tSeed(t, owner, 35)

	resp, body := tList(t, tok, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET items → %d", resp.StatusCode)
	}
	if n := len(tIDs(t, body)); n != httpx.DefaultLimit {
		t.Fatalf("số mục mặc định = %d, cần %d", n, httpx.DefaultLimit)
	}
	if next, ok := body["next_cursor"].(string); !ok || next == "" {
		t.Fatalf("next_cursor = %v (còn trang thì phải có)", body["next_cursor"])
	}
}

// TestPagination_LimitBounds: limit 1 và 100 được; ngoài 1..100 hoặc không phải số → 422 field "limit" (03-AC8).
func TestPagination_LimitBounds(t *testing.T) {
	t.Parallel()
	owner, tok := tUser(t)
	tSeed(t, owner, 120)

	for _, limit := range []int{1, 100} {
		_, body := tList(t, tok, fmt.Sprintf("?limit=%d", limit))
		if n := len(tIDs(t, body)); n != limit {
			t.Errorf("limit=%d → %d mục", limit, n)
		}
	}
	for _, bad := range []string{"0", "101", "abc", "-1", "1.5"} {
		resp, body := tList(t, tok, "?limit="+bad)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("limit=%s → %d, cần 422", bad, resp.StatusCode)
			continue
		}
		if body["code"] != apierr.ValidationFailed {
			t.Errorf("limit=%s: code = %v", bad, body["code"])
		}
		details, _ := body["details"].([]any)
		if len(details) == 0 {
			t.Errorf("limit=%s: thiếu details", bad)
			continue
		}
		m, _ := details[0].(map[string]any)
		if m["field"] != "limit" {
			t.Errorf("limit=%s: details[0].field = %v", bad, m["field"])
		}
	}
}

// TestPagination_InvalidCursor: cursor rác / JSON hỏng / sai phiên bản / sai kiểu → 422 INVALID_CURSOR (03-AC8).
func TestPagination_InvalidCursor(t *testing.T) {
	t.Parallel()
	_, tok := tUser(t)
	enc := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	bad := []string{
		"@@@",
		enc(`{"v":1,"t":`),
		enc(`{"v":2,"t":1767225600000000,"i":"00000000-0000-7000-8000-000000000001"}`),
		enc(`{"v":1,"t":"abc","i":"00000000-0000-7000-8000-000000000001"}`),
		enc(`{"v":1,"t":1767225600000000,"i":"khong-phai-uuid"}`),
	}
	for _, c := range bad {
		resp, body := tList(t, tok, "?cursor="+c)
		if resp.StatusCode != http.StatusUnprocessableEntity || body["code"] != apierr.InvalidCursor {
			t.Errorf("cursor %q → %d %v, cần 422 INVALID_CURSOR", c, resp.StatusCode, body["code"])
		}
	}
}

// TestPagination_Shape: thân chỉ có {items, next_cursor}; danh sách rỗng là [] (không null);
// trang cuối có next_cursor = null (03-AC8).
func TestPagination_Shape(t *testing.T) {
	t.Parallel()
	owner, tok := tUser(t)

	_, body := tList(t, tok, "")
	if len(body) != 2 || body["items"] == nil {
		t.Fatalf("thân danh sách rỗng = %v (cần items [] + next_cursor null)", body)
	}
	if items, ok := body["items"].([]any); !ok || len(items) != 0 {
		t.Errorf("items = %v, cần mảng rỗng", body["items"])
	}
	if _, has := body["next_cursor"]; !has || body["next_cursor"] != nil {
		t.Errorf("next_cursor = %v, cần null nhưng vẫn có trường", body["next_cursor"])
	}

	tSeed(t, owner, 5)
	_, last := tList(t, tok, "?limit=100")
	for k := range last {
		if k != "items" && k != "next_cursor" {
			t.Errorf("khoá lạ trong thân danh sách: %q", k)
		}
	}
	if last["next_cursor"] != nil {
		t.Errorf("trang cuối: next_cursor = %v, cần null", last["next_cursor"])
	}
	if n := len(tIDs(t, last)); n != 5 {
		t.Errorf("số mục = %d, cần 5", n)
	}

	// Chỉ trả bản ghi của chính người gọi (#Q-QC-03-2).
	_, other := tUser(t)
	_, otherBody := tList(t, other, "?limit=100")
	if n := len(tIDs(t, otherBody)); n != 0 {
		t.Errorf("người dùng khác thấy %d bản ghi, cần 0", n)
	}
}

// TestCursor_NoSkipNoDup: 250 bản ghi, limit 100 → đi hết trang được đúng 250 id duy nhất,
// đúng thứ tự (created_at DESC, id DESC) như truy vấn trực tiếp (03-AC8, 03-AC9).
func TestCursor_NoSkipNoDup(t *testing.T) {
	t.Parallel()
	owner, tok := tUser(t)
	_, d := tStack(t)
	tSeed(t, owner, 250)

	ids := tWalk(t, tok, 100)
	tAssertUnique(t, ids, 250)

	rows, err := d.DB.Query(t.Context(),
		`select id from _test_items where owner_id = $1::uuid order by created_at desc, id desc`, owner)
	if err != nil {
		t.Fatalf("truy vấn thứ tự: %v", err)
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if i >= len(ids) || ids[i] != id.String() {
			t.Fatalf("thứ tự lệch ở vị trí %d: API %q, SQL %q", i, ids[min(i, len(ids)-1)], id)
		}
		i++
	}
	if i != len(ids) {
		t.Fatalf("SQL trả %d dòng, API trả %d", i, len(ids))
	}
}

// TestCursor_SameTimestamp: 300 bản ghi CÙNG created_at vẫn không sót, không lặp (phân định bằng id) (03-AC9).
func TestCursor_SameTimestamp(t *testing.T) {
	t.Parallel()
	owner, tok := tUser(t)
	_, d := tStack(t)
	if _, err := d.DB.Exec(t.Context(),
		`insert into _test_items (name, owner_id, created_at)
		 select 'same-'||g, $1::uuid, timestamptz '2026-01-01 00:00:00+00' from generate_series(1, 300) g`,
		owner); err != nil {
		t.Fatalf("seed cùng created_at: %v", err)
	}

	ids := tWalk(t, tok, 100)
	tAssertUnique(t, ids, 300)
}

// TestCursor_InsertAndDeleteDuringScan: chen bản ghi mới hơn / cũ hơn con trỏ và xoá bản đã đọc giữa
// lúc đi trang — mọi bản ghi gốc còn lại và bản chen vào vùng chưa đọc xuất hiện đúng 1 lần (03-AC9).
func TestCursor_InsertAndDeleteDuringScan(t *testing.T) {
	t.Parallel()
	owner, tok := tUser(t)
	_, d := tStack(t)
	tSeed(t, owner, 50)

	_, first := tList(t, tok, "?limit=20")
	page1 := tIDs(t, first)
	if len(page1) != 20 {
		t.Fatalf("trang 1 có %d mục, cần 20", len(page1))
	}
	cursor, _ := first["next_cursor"].(string)
	if cursor == "" {
		t.Fatal("trang 1 không có next_cursor")
	}

	// Xoá 3 bản ghi ĐÃ đọc, chen 5 bản mới hơn (vùng đã qua) và 5 bản cũ hơn con trỏ (vùng chưa đọc).
	if _, err := d.DB.Exec(t.Context(), `delete from _test_items where id = any($1::uuid[])`, page1[:3]); err != nil {
		t.Fatalf("xoá 3 bản đã đọc: %v", err)
	}
	for _, spec := range []struct{ name, shift string }{{"newer", "+ interval '1 hour'"}, {"older", "- interval '1 hour'"}} {
		if _, err := d.DB.Exec(t.Context(),
			`insert into _test_items (name, owner_id, created_at)
			 select '`+spec.name+`-'||g, $1::uuid, now() `+spec.shift+` from generate_series(1, 5) g`, owner); err != nil {
			t.Fatalf("chen bản ghi %s: %v", spec.name, err)
		}
	}

	rest := []string{cursor}
	var seen []string
	seen = append(seen, page1...)
	for page := 0; page < 20 && len(rest) > 0; page++ {
		resp, body := tList(t, tok, "?limit=20&cursor="+rest[0])
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("trang kế → %d", resp.StatusCode)
		}
		seen = append(seen, tIDs(t, body)...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			break
		}
		rest[0] = next
	}
	counts := map[string]int{}
	for _, id := range seen {
		counts[id]++
		if counts[id] > 1 {
			t.Fatalf("id %s xuất hiện %d lần", id, counts[id])
		}
	}
	// Mọi bản ghi gốc chưa xoá xuất hiện đúng 1 lần; 5 bản chen vào vùng CHƯA đọc cũng vậy;
	// 5 bản chen vào vùng ĐÃ đọc (mới hơn con trỏ) không xuất hiện.
	kept := 0
	for _, id := range tQueryIDs(t, d, owner, "seed-%") {
		if counts[id] != 1 {
			t.Fatalf("bản ghi gốc %s xuất hiện %d lần", id, counts[id])
		}
		kept++
	}
	if kept != 47 {
		t.Fatalf("còn %d bản ghi gốc, cần 47 (50 − 3 đã xoá)", kept)
	}
	older := tQueryIDs(t, d, owner, "older-%")
	if len(older) != 5 {
		t.Fatalf("số bản ghi cũ hơn con trỏ = %d", len(older))
	}
	for _, id := range older {
		if counts[id] != 1 {
			t.Fatalf("bản chen vào vùng chưa đọc %s xuất hiện %d lần", id, counts[id])
		}
	}
	for _, id := range tQueryIDs(t, d, owner, "newer-%") {
		if counts[id] != 0 {
			t.Fatalf("bản chen vào vùng đã đọc %s lại xuất hiện %d lần", id, counts[id])
		}
	}
}

// tQueryIDs trả id các bản ghi của owner có tên khớp pattern.
func tQueryIDs(t *testing.T, d Deps, owner, pattern string) []string {
	t.Helper()
	rows, err := d.DB.Query(t.Context(),
		`select id from _test_items where owner_id = $1::uuid and name like $2`, owner, pattern)
	if err != nil {
		t.Fatalf("truy vấn id (%s): %v", pattern, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan id: %v", err)
		}
		out = append(out, id.String())
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("đọc id: %v", err)
	}
	return out
}

func tAssertUnique(t *testing.T, ids []string, want int) {
	t.Helper()
	uniq := map[string]bool{}
	for _, id := range ids {
		if uniq[id] {
			t.Fatalf("id lặp: %s", id)
		}
		uniq[id] = true
	}
	if len(uniq) != want {
		t.Fatalf("đọc được %d id duy nhất (tổng %d), cần %d", len(uniq), len(ids), want)
	}
}

// TestCursor_Format: cursor là base64url không đệm của {"v":1,"t":<micro-giây>,"i":<uuid>}, ≤ 120 ký tự (03-AC10).
func TestCursor_Format(t *testing.T) {
	t.Parallel()
	owner, tok := tUser(t)
	tSeed(t, owner, 10)

	_, body := tList(t, tok, "?limit=1")
	cursor, _ := body["next_cursor"].(string)
	if cursor == "" {
		t.Fatal("không có next_cursor")
	}
	if len(cursor) > 120 {
		t.Errorf("độ dài cursor = %d (> 120)", len(cursor))
	}
	if strings.ContainsAny(cursor, "=+/") {
		t.Errorf("cursor không phải base64url không đệm: %q", cursor)
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		t.Fatalf("giải base64url: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("thân cursor không phải JSON: %v", err)
	}
	if len(decoded) != 3 || decoded["v"] != float64(1) {
		t.Fatalf("cursor = %v (cần đúng {v,t,i} với v=1)", decoded)
	}
	micro, ok := decoded["t"].(float64)
	if !ok || micro < 1.7e15 {
		t.Errorf("t = %v (cần micro-giây)", decoded["t"])
	}
	if _, err := uuid.Parse(fmt.Sprint(decoded["i"])); err != nil {
		t.Errorf("i = %v (cần uuid)", decoded["i"])
	}

	// Cursor đi được vòng tròn: giải mã rồi mã hoá lại cho đúng chuỗi cũ.
	c, err := httpx.ParseCursor(cursor)
	if err != nil {
		t.Fatalf("ParseCursor: %v", err)
	}
	if got := httpx.EncodeCursor(c.CreatedAt, c.ID.String()); got != cursor {
		t.Errorf("mã hoá lại = %q, cần %q", got, cursor)
	}
}

// TestCursor_UsesIndex: trên 10.000 dòng, truy vấn trang dùng Index Scan và KHÔNG Sort (03-AC10).
func TestCursor_UsesIndex(t *testing.T) {
	t.Parallel()
	owner, _ := tUser(t)
	_, d := tStack(t)
	tSeed(t, owner, 10000)
	if _, err := d.DB.Exec(t.Context(), "analyze _test_items"); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	rows, err := d.DB.Query(t.Context(),
		`explain select id, name, created_at from _test_items
		 where (created_at, id) < (now(), 'ffffffff-ffff-ffff-ffff-ffffffffffff'::uuid)
		 order by created_at desc, id desc limit 31`)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		plan.WriteString(line + "\n")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("đọc plan: %v", err)
	}
	if !strings.Contains(plan.String(), "Index Scan") && !strings.Contains(plan.String(), "Index Only Scan") {
		t.Fatalf("plan không dùng index:\n%s", plan.String())
	}
	if strings.Contains(plan.String(), "Sort") {
		t.Fatalf("plan có Sort:\n%s", plan.String())
	}
}

// TestETag_NotModified: GET tài nguyên → ETag W/"v<version>" + Cache-Control + Vary; If-None-Match khớp
// → 304 thân rỗng kèm ETag (03-AC15).
func TestETag_NotModified(t *testing.T) {
	t.Parallel()
	_, tok := tUser(t)
	it := tNewItem(t, tok, "etag-304")
	path := itemsURL + "/" + fmt.Sprint(it["id"])

	resp := tDo(t, http.MethodGet, path, tok, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET item → %d", resp.StatusCode)
	}
	etag := resp.Header.Get("ETag")
	if etag != `W/"v1"` {
		t.Fatalf("ETag = %q", etag)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "private, no-cache" {
		t.Errorf("Cache-Control = %q", cc)
	}
	if !strings.Contains(strings.Join(resp.Header.Values("Vary"), ","), "Authorization") {
		t.Errorf("Vary = %v", resp.Header.Values("Vary"))
	}

	cached := tDo(t, http.MethodGet, path, tok, "", "If-None-Match", etag)
	if cached.StatusCode != http.StatusNotModified {
		t.Fatalf("If-None-Match khớp → %d, cần 304", cached.StatusCode)
	}
	if body := tBody(t, cached); len(body) != 0 {
		t.Errorf("304 có thân %d byte", len(body))
	}
	if cached.Header.Get("ETag") != etag {
		t.Errorf("304: ETag = %q", cached.Header.Get("ETag"))
	}
	if miss := tDo(t, http.MethodGet, path, tok, "", "If-None-Match", `W/"v999"`); miss.StatusCode != http.StatusOK {
		t.Errorf("ETag không khớp → %d, cần 200", miss.StatusCode)
	}
}

// TestETag_ChangesAfterUpdate: PUT đổi ETag; ETag cũ không còn khớp; If-None-Match bị bỏ qua trên POST/PUT (03-AC15).
func TestETag_ChangesAfterUpdate(t *testing.T) {
	t.Parallel()
	_, tok := tUser(t)
	it := tNewItem(t, tok, "etag-update")
	path := itemsURL + "/" + fmt.Sprint(it["id"])
	before := tDo(t, http.MethodGet, path, tok, "").Header.Get("ETag")

	put := tDo(t, http.MethodPut, path, tok, `{"name":"etag-update-2","version":1}`, "If-None-Match", "*")
	if put.StatusCode != http.StatusOK {
		t.Fatalf("PUT (If-None-Match: * phải bị bỏ qua) → %d", put.StatusCode)
	}
	after := tDo(t, http.MethodGet, path, tok, "").Header.Get("ETag")
	if after == before || after != `W/"v2"` {
		t.Fatalf("ETag sau PUT = %q (trước: %q)", after, before)
	}
	if stale := tDo(t, http.MethodGet, path, tok, "", "If-None-Match", before); stale.StatusCode != http.StatusOK {
		t.Errorf("ETag cũ → %d, cần 200", stale.StatusCode)
	}

	created := tDo(t, http.MethodPost, itemsURL, tok, `{"name":"etag-post"}`,
		"Idempotency-Key", tIdemKey(t), "If-None-Match", "*")
	if created.StatusCode != http.StatusCreated {
		t.Errorf("POST với If-None-Match: * → %d, cần 201", created.StatusCode)
	}
}

// TestETag_MultipleValues: If-None-Match dạng danh sách và `*` đều khớp; ETag danh sách = băm thân (03-AC15).
func TestETag_MultipleValues(t *testing.T) {
	t.Parallel()
	owner, tok := tUser(t)
	it := tNewItem(t, tok, "etag-list")
	path := itemsURL + "/" + fmt.Sprint(it["id"])
	etag := tDo(t, http.MethodGet, path, tok, "").Header.Get("ETag")

	for _, inm := range []string{`W/"v9", ` + etag, etag + `, W/"v9"`, "*"} {
		if resp := tDo(t, http.MethodGet, path, tok, "", "If-None-Match", inm); resp.StatusCode != http.StatusNotModified {
			t.Errorf("If-None-Match %q → %d, cần 304", inm, resp.StatusCode)
		}
	}

	tSeed(t, owner, 4)
	resp, _ := tList(t, tok, "?limit=5")
	listETag := resp.Header.Get("ETag")
	sum := sha256.Sum256(tBodyOf(t, tok, "?limit=5"))
	want := `W/"` + base64.RawURLEncoding.EncodeToString(sum[:])[:16] + `"`
	if listETag != want {
		t.Fatalf("ETag danh sách = %q, cần băm thân %q", listETag, want)
	}
	if again := tDo(t, http.MethodGet, itemsURL+"?limit=5", tok, "").Header.Get("ETag"); again != listETag {
		t.Errorf("ETag danh sách đổi dù thân không đổi: %q ≠ %q", again, listETag)
	}
	if cached := tDo(t, http.MethodGet, itemsURL+"?limit=5", tok, "", "If-None-Match", listETag); cached.StatusCode != http.StatusNotModified {
		t.Errorf("If-None-Match trên danh sách → %d, cần 304", cached.StatusCode)
	}
}

// tBodyOf lấy đúng byte thân của một lần gọi danh sách (để băm lại như client).
func tBodyOf(t *testing.T, token, query string) []byte {
	t.Helper()
	return tBody(t, tDo(t, http.MethodGet, itemsURL+query, token, ""))
}

// TestOptimisticLock_Stale409: PUT với version cũ → 409 VERSION_CONFLICT kèm current_version, current
// và header ETag hiện tại; dữ liệu không đổi (03-AC14).
func TestOptimisticLock_Stale409(t *testing.T) {
	t.Parallel()
	_, tok := tUser(t)
	it := tNewItem(t, tok, "lock-stale")
	path := itemsURL + "/" + fmt.Sprint(it["id"])

	if resp := tDo(t, http.MethodPut, path, tok, `{"name":"v2","version":1}`); resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT lần 1 → %d", resp.StatusCode)
	}
	resp := tDo(t, http.MethodPut, path, tok, `{"name":"v2-lan2","version":1}`)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("PUT version cũ → %d, cần 409", resp.StatusCode)
	}
	body := tJSON(t, resp)
	if body["code"] != apierr.VersionConflict {
		t.Fatalf("code = %v", body["code"])
	}
	details, _ := body["details"].(map[string]any)
	if details["current_version"] != float64(2) {
		t.Errorf("details.current_version = %v", details["current_version"])
	}
	if _, ok := details["current"].(map[string]any); !ok {
		t.Errorf("details.current = %v (cần object)", details["current"])
	}
	if etag := resp.Header.Get("ETag"); etag != `W/"v2"` {
		t.Errorf("ETag = %q", etag)
	}
	cur := tJSON(t, tDo(t, http.MethodGet, path, tok, ""))
	if cur["version"] != float64(2) || cur["name"] != "v2" {
		t.Errorf("bản ghi bị đổi: %v", cur)
	}
}

// TestOptimisticLock_Concurrent20: 20 PUT song song cùng version → đúng 1 lần 200, 19 lần 409,
// không 5xx, version cuối = n+1 (03-AC14).
func TestOptimisticLock_Concurrent20(t *testing.T) {
	t.Parallel()
	_, tok := tUser(t)
	it := tNewItem(t, tok, "lock-race")
	path := itemsURL + "/" + fmt.Sprint(it["id"])

	var (
		mu     sync.Mutex
		status []int
		wg     sync.WaitGroup
	)
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := tDo(t, http.MethodPut, path, tok, fmt.Sprintf(`{"name":"race-%d","version":1}`, i))
			mu.Lock()
			status = append(status, resp.StatusCode)
			mu.Unlock()
		}()
	}
	wg.Wait()

	var ok, conflict, server int
	for _, s := range status {
		switch {
		case s == http.StatusOK:
			ok++
		case s == http.StatusConflict:
			conflict++
		case s >= 500:
			server++
		}
	}
	if ok != 1 || conflict != 19 || server != 0 {
		t.Fatalf("20 PUT song song: 200=%d 409=%d 5xx=%d (cần 1 / 19 / 0) — %v", ok, conflict, server, status)
	}
	if cur := tJSON(t, tDo(t, http.MethodGet, path, tok, "")); cur["version"] != float64(2) {
		t.Fatalf("version cuối = %v, cần 2", cur["version"])
	}
}

// TestOptimisticLock_IfMatch: If-Match W/"v<n>" tương đương trường version; lệch nhau → 422 (03-AC14).
func TestOptimisticLock_IfMatch(t *testing.T) {
	t.Parallel()
	_, tok := tUser(t)
	it := tNewItem(t, tok, "lock-ifmatch")
	path := itemsURL + "/" + fmt.Sprint(it["id"])

	if resp := tDo(t, http.MethodPut, path, tok, `{"name":"im-1"}`, "If-Match", `W/"v1"`); resp.StatusCode != http.StatusOK {
		t.Fatalf("If-Match đúng → %d, cần 200", resp.StatusCode)
	}
	if resp := tDo(t, http.MethodPut, path, tok, `{"name":"im-2"}`, "If-Match", `W/"v1"`); resp.StatusCode != http.StatusConflict {
		t.Fatalf("If-Match cũ → %d, cần 409", resp.StatusCode)
	}
	resp := tDo(t, http.MethodPut, path, tok, `{"name":"im-3","version":2}`, "If-Match", `W/"v9"`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("If-Match ≠ version trong thân → %d, cần 422", resp.StatusCode)
	}
	if f := tFirstField(t, resp); f != "version" {
		t.Errorf("details[0].field = %q", f)
	}
	if cur := tJSON(t, tDo(t, http.MethodGet, path, tok, "")); cur["version"] != float64(2) {
		t.Errorf("version trong DB = %v, cần 2", cur["version"])
	}
}

// TestOptimisticLock_Missing: thiếu version → 422 field "version"; id không có → 404;
// bản ghi của người khác → 404 (03-AC14, #Q-QC-03-2, #Q-QC-03-5).
func TestOptimisticLock_Missing(t *testing.T) {
	t.Parallel()
	_, tok := tUser(t)
	it := tNewItem(t, tok, "lock-missing")
	path := itemsURL + "/" + fmt.Sprint(it["id"])

	resp := tDo(t, http.MethodPut, path, tok, `{"name":"khong-version"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("thiếu version → %d, cần 422", resp.StatusCode)
	}
	if resp := tJSON(t, resp); resp["code"] != apierr.ValidationFailed {
		t.Errorf("code = %v", resp["code"])
	}
	for _, bad := range []string{"0", "-1", "1.5", `"a"`} {
		r := tDo(t, http.MethodPut, path, tok, `{"name":"x","version":`+bad+`}`)
		if r.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("version=%s → %d, cần 422", bad, r.StatusCode)
			continue
		}
		if f := tFirstField(t, r); f != "version" {
			t.Errorf("version=%s: details[0].field = %q", bad, f)
		}
	}
	if r := tDo(t, http.MethodPut, itemsURL+"/"+uuid.NewString(), tok,
		`{"name":"x","version":1}`); r.StatusCode != http.StatusNotFound {
		t.Errorf("id không tồn tại → %d, cần 404", r.StatusCode)
	}

	_, other := tUser(t)
	if r := tDo(t, http.MethodPut, path, other, `{"name":"cua-nguoi-khac","version":1}`); r.StatusCode != http.StatusNotFound {
		t.Errorf("PUT bản ghi của người khác → %d, cần 404", r.StatusCode)
	}
	if r := tDo(t, http.MethodGet, path, other, ""); r.StatusCode != http.StatusNotFound {
		t.Errorf("GET bản ghi của người khác → %d, cần 404", r.StatusCode)
	}
	if cur := tJSON(t, tDo(t, http.MethodGet, path, tok, "")); cur["version"] != float64(1) {
		t.Errorf("version trong DB = %v, cần 1 (không bị đổi)", cur["version"])
	}
}
