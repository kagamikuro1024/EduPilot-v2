// Package attendancehttp: điểm danh tự phục vụ (FEAT-attendance-selfcheck SRS 6). Handler mỏng, nghiệp vụ ở internal/attendance.
// Mọi phản hồi mang dữ liệu điểm danh có `Cache-Control: no-store` (luật 15). Danh tính của sinh viên lấy từ JWT, không từ thân.
package attendancehttp

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/attendance"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

// Handler gom route điểm danh. Guard: Staff = TEACHER / TA của lớp (ADMIN không qua, D39); StudentRole = STUDENT của lớp.
type Handler struct {
	Svc       *attendance.Service
	Guard     func(auth.GuardMode) func(http.Handler) http.Handler
	Idem      func(http.Handler) http.Handler // `Idempotency-Key` bắt buộc (mở cửa sổ, đánh vắng)
	PublicURL string                          // APP_PUBLIC_URL: gốc của qr_url
	ClientIP  func(*http.Request) string      // hàm client-IP của middleware hạn mức (tin proxy đã khai)
	Log       *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware.
func (h *Handler) Mount(r chi.Router) {
	staff := h.Guard(auth.Staff)
	student := h.Guard(auth.StudentRole)
	r.With(staff, h.Idem).Post("/courses/{id}/sessions/{sid}/checkin-windows", h.openWindow)
	r.With(staff).Get("/courses/{id}/sessions/{sid}/checkin-windows", h.listWindows)
	r.With(staff).Get("/courses/{id}/sessions/{sid}/checkin-windows/{wid}/display", h.display)
	r.With(staff).Patch("/courses/{id}/checkin-windows/{wid}", h.setHeadcount)
	r.With(staff).Post("/courses/{id}/checkin-windows/{wid}/close", h.closeWindow)
	r.With(staff).Post("/courses/{id}/checkin-windows/{wid}/spot-check", h.spotCheck)
	r.With(staff, h.Idem).Post("/courses/{id}/sessions/{sid}/absent-unchecked", h.absentUnchecked)
	r.With(staff).Get("/courses/{id}/sessions/{sid}/checkins", h.listCheckins)
	r.With(staff).Post("/courses/{id}/checkins/{cid}/clear-flags", h.clearFlags)
	r.With(student).Post("/courses/{id}/checkins", h.checkin)
	r.Post("/me/devices/challenge", h.deviceChallenge)
	r.Post("/me/devices", h.deviceRegister)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		apierr.Write(w, r, ae)
		return
	}
	if h.Log != nil {
		h.Log.ErrorContext(r.Context(), "attendance: lỗi không lường trước", "error", err.Error(), "path", r.URL.Path)
	}
	apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
}

func notFound(w http.ResponseWriter, r *http.Request) {
	apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
}

func actorOf(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	p, ok := auth.FromContext(r.Context())
	id, err := uuid.Parse(p.Sub)
	if !ok || err != nil {
		apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
		return uuid.Nil, false
	}
	return id, true
}

func idParam(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		notFound(w, r)
		return uuid.Nil, false
	}
	return id, true
}

// optionalBody giải mã thân JSON nếu có; thân rỗng là hợp lệ (mở cửa sổ không cần `minutes`).
func optionalBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil || r.ContentLength == 0 {
		return true
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusBadRequest, apierr.BadRequest))
		return false
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return true
	}
	r.Body = io.NopCloser(bytes.NewReader(b))
	return httpx.DecodeJSON(w, r, dst)
}

func noStore(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }

// ---- cửa sổ ----

type openBody struct {
	Minutes *int `json:"minutes"`
}

func (h *Handler) openWindow(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	sid, ok := idParam(w, r, "sid")
	if !ok {
		return
	}
	var b openBody
	if !optionalBody(w, r, &b) {
		return
	}
	out, err := h.Svc.OpenWindow(r.Context(), actor, cid, sid, b.Minutes)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": out.ID, "round": out.Round, "rotate_seconds": out.RotateSeconds, "closes_at": out.ClosesAt})
}

func (h *Handler) listWindows(w http.ResponseWriter, r *http.Request) {
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	sid, ok := idParam(w, r, "sid")
	if !ok {
		return
	}
	items, err := h.Svc.ListWindows(r.Context(), cid, sid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) display(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	wid, ok := idParam(w, r, "wid")
	if !ok {
		return
	}
	out, err := h.Svc.Display(r.Context(), actor, cid, wid, h.PublicURL)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	w.Header().Set("Referrer-Policy", "no-referrer")
	httpx.WriteJSON(w, http.StatusOK, out)
}

type headcountBody struct {
	Headcount *int `json:"headcount"`
}

func (h *Handler) setHeadcount(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	wid, ok := idParam(w, r, "wid")
	if !ok {
		return
	}
	var b headcountBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	out, err := h.Svc.SetHeadcount(r.Context(), actor, cid, wid, b.Headcount)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) closeWindow(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	wid, ok := idParam(w, r, "wid")
	if !ok {
		return
	}
	out, err := h.Svc.Close(r.Context(), &actor, cid, wid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, out)
}

type spotBody struct {
	N *int `json:"n"`
}

func (h *Handler) spotCheck(w http.ResponseWriter, r *http.Request) {
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	wid, ok := idParam(w, r, "wid")
	if !ok {
		return
	}
	var b spotBody
	if !optionalBody(w, r, &b) {
		return
	}
	out, err := h.Svc.SpotCheck(r.Context(), cid, wid, b.N)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"students": out})
}

func (h *Handler) absentUnchecked(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	sid, ok := idParam(w, r, "sid")
	if !ok {
		return
	}
	n, err := h.Svc.AbsentUnchecked(r.Context(), actor, cid, sid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"marked": n})
}

// ---- lượt ----

func (h *Handler) listCheckins(w http.ResponseWriter, r *http.Request) {
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	sid, ok := idParam(w, r, "sid")
	if !ok {
		return
	}
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	flagged := r.URL.Query().Get("flagged") == "true"
	round := 0
	if v := r.URL.Query().Get("round"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 3 {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "round", Code: "INVALID", Message: "Vòng từ 1 đến 3."}))
			return
		}
		round = n
	}
	page, err := h.Svc.ListCheckins(r.Context(), cid, sid, flagged, round, pg)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (h *Handler) clearFlags(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorOf(w, r)
	if !ok {
		return
	}
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	id, ok := idParam(w, r, "cid")
	if !ok {
		return
	}
	out, err := h.Svc.ClearFlags(r.Context(), actor, cid, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) checkin(w http.ResponseWriter, r *http.Request) {
	uid, ok := actorOf(w, r)
	if !ok {
		return
	}
	cid, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	var b attendance.CheckinIn
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	ip := ""
	if h.ClientIP != nil {
		ip = h.ClientIP(r)
	}
	out, created, err := h.Svc.Checkin(r.Context(), uid, cid, b, ip)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, out)
}

// ---- thiết bị ----

func studentOnly(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	p, ok := auth.FromContext(r.Context())
	if !ok || p.Role != auth.RoleStudent {
		apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden))
		return uuid.Nil, false
	}
	return actorOf(w, r)
}

func (h *Handler) deviceChallenge(w http.ResponseWriter, r *http.Request) {
	uid, ok := studentOnly(w, r)
	if !ok {
		return
	}
	nonce, err := h.Svc.DeviceChallenge(r.Context(), uid)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"nonce": base64.StdEncoding.EncodeToString(nonce)})
}

type registerBody struct {
	Kind             string   `json:"kind"`
	PublicKey        string   `json:"public_key"`
	AttestationChain []string `json:"attestation_chain"`
}

func (h *Handler) deviceRegister(w http.ResponseWriter, r *http.Request) {
	uid, ok := studentOnly(w, r)
	if !ok {
		return
	}
	var b registerBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	pk, err := base64.StdEncoding.DecodeString(b.PublicKey)
	if err != nil || b.Kind != attendance.KindAndroidKey || len(b.AttestationChain) < 2 || len(b.AttestationChain) > 6 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "attestation_chain", Code: "INVALID", Message: "Cần 2 đến 6 chứng chỉ và khoá công khai."}))
		return
	}
	chain := make([][]byte, 0, len(b.AttestationChain))
	total := 0
	for _, c := range b.AttestationChain {
		der, err := base64.StdEncoding.DecodeString(c)
		if err != nil {
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "attestation_chain", Code: "INVALID", Message: "Chứng chỉ không đọc được."}))
			return
		}
		total += len(der)
		chain = append(chain, der)
	}
	if total > 16<<10 {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "attestation_chain", Code: "TOO_LARGE", Message: "Chuỗi chứng chỉ quá lớn."}))
		return
	}
	out, err := h.Svc.DeviceRegister(r.Context(), uid, attendance.RegisterIn{Kind: b.Kind, PublicKey: pk, AttestationChain: chain})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	noStore(w)
	code := http.StatusOK
	if out.Created {
		code = http.StatusCreated
	}
	httpx.WriteJSON(w, code, map[string]string{"key_id": out.KeyID})
}
