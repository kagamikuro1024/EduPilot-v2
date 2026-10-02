package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/jobs"
)

// TestJobs_Ownership: `GET /api/v1/jobs/{id}` chỉ chủ việc hoặc ADMIN thấy; người khác / id lạ / id không phải
// uuid đều 404 NOT_FOUND (không lộ sự tồn tại) — US-PG-03 AC17, AC18.
func TestJobs_Ownership(t *testing.T) {
	t.Parallel()
	d, _ := idemDeps(t)
	svc := jobs.NewService(d.DB)
	owner := uuid.New()

	j, err := svc.Enqueue(t.Context(), owner, "test.progress", map[string]any{"steps": 4})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	h := newRouterWith(d, func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sub := r.Header.Get("X-Test-Sub")
				if sub == "" {
					next.ServeHTTP(w, r)
					return
				}
				p := auth.Principal{Sub: sub, Role: auth.Role(r.Header.Get("X-Test-Role")), JTI: "test"}
				next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
			})
		})
		r.Get("/jobs/{id}", jobs.Handler(svc))
	})

	get := func(sub, role, id string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+id, nil)
		if sub != "" {
			r.Header.Set("X-Test-Sub", sub)
			r.Header.Set("X-Test-Role", role)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}

	// Chủ việc: 200 + đúng tập trường của SRS 6.2.
	w := get(owner.String(), string(auth.RoleStudent), j.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("chủ việc: status = %d (%s)", w.Code, w.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("thân không phải JSON: %s", w.Body)
	}
	allowed := map[string]bool{"id": true, "kind": true, "status": true, "progress": true, "result": true,
		"error": true, "created_at": true, "updated_at": true, "finished_at": true}
	for k := range body {
		if !allowed[k] {
			t.Fatalf("trường lạ trong thân: %q (%s)", k, w.Body)
		}
	}
	for _, k := range []string{"id", "kind", "status", "progress", "created_at", "updated_at"} {
		if _, ok := body[k]; !ok {
			t.Fatalf("thiếu trường %q: %s", k, w.Body)
		}
	}
	if body["id"] != j.ID.String() || body["status"] != "QUEUED" {
		t.Fatalf("thân = %s", w.Body)
	}
	if _, ok := body["finished_at"]; ok {
		t.Fatalf("việc chưa xong không được có finished_at: %s", w.Body)
	}

	// Người khác: 404 (không phải 403).
	w = get(uuid.NewString(), string(auth.RoleStudent), j.ID.String())
	if w.Code != http.StatusNotFound || idemCode(t, w) != "NOT_FOUND" {
		t.Fatalf("người khác: status = %d, code = %q; muốn 404 NOT_FOUND", w.Code, idemCode(t, w))
	}

	// ADMIN: 200.
	if w := get(uuid.NewString(), string(auth.RoleAdmin), j.ID.String()); w.Code != http.StatusOK {
		t.Fatalf("ADMIN: status = %d (%s)", w.Code, w.Body)
	}

	// id không tồn tại / không phải uuid → 404.
	for _, id := range []string{uuid.NewString(), "not-a-uuid"} {
		w := get(owner.String(), string(auth.RoleStudent), id)
		if w.Code != http.StatusNotFound || idemCode(t, w) != "NOT_FOUND" {
			t.Fatalf("id %q: status = %d, code = %q; muốn 404 NOT_FOUND", id, w.Code, idemCode(t, w))
		}
	}

	// Không có danh tính → 401.
	if w := get("", "", j.ID.String()); w.Code != http.StatusUnauthorized {
		t.Fatalf("không token: status = %d, muốn 401", w.Code)
	}
}
