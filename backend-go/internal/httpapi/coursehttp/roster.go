package coursehttp

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
)

type rowErrorOut struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type rosterOut struct {
	Total             int           `json:"total"`
	CreatedUsers      int           `json:"created_users"`
	LinkedExisting    int           `json:"linked_existing"`
	AlreadyMember     int           `json:"already_member"`
	PendingUnverified int           `json:"pending_unverified"`
	SkippedRemoved    int           `json:"skipped_removed"`
	Errors            []rowErrorOut `json:"errors"`
	DryRun            bool          `json:"dry_run"`
}

// rosterImport: multipart `file` (CSV / XLSX). Thân đã bị bodyLimitMiddleware chặn ở 2 MiB + phần bọc multipart; tệp > 2 MiB ⇒ 413.
func (h *Handler) rosterImport(w http.ResponseWriter, r *http.Request) {
	courseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	teacher, ok := actorOf(w, r)
	if !ok {
		return
	}
	// maxMemory > giới hạn thân ⇒ không bao giờ tràn ra tệp tạm: tệp roster không chạm đĩa.
	if err := r.ParseMultipartForm(course.RosterMaxFileBytes + 1<<20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			apierr.Write(w, r, apierr.New(http.StatusRequestEntityTooLarge, apierr.PayloadTooLarge).WithDetails(map[string]any{"max_bytes": course.RosterMaxFileBytes}))
			return
		}
		h.fail(w, r, "roster-import", &course.InvalidError{Field: "file", Code: "UNSUPPORTED_FILE", Message: "Không đọc được tệp. Hãy dùng CSV UTF-8 hoặc XLSX."})
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()
	f, hdr, err := r.FormFile("file")
	if err != nil {
		h.fail(w, r, "roster-import", &course.InvalidError{Field: "file", Code: "REQUIRED", Message: "Hãy chọn một tệp CSV hoặc XLSX."})
		return
	}
	defer func() { _ = f.Close() }()
	if hdr.Size > course.RosterMaxFileBytes {
		apierr.Write(w, r, apierr.New(http.StatusRequestEntityTooLarge, apierr.PayloadTooLarge).WithDetails(map[string]any{"max_bytes": course.RosterMaxFileBytes}))
		return
	}
	data, err := io.ReadAll(io.LimitReader(f, course.RosterMaxFileBytes+1))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusBadRequest, apierr.BadRequest))
		return
	}
	dry := r.URL.Query().Get("dry_run") == "true"
	invites := !strings.EqualFold(strings.TrimSpace(r.FormValue("send_invites")), "false")
	rep, err := h.Courses.ImportRoster(r.Context(), courseID, teacher, data, course.RosterOptions{DryRun: dry, SendInvites: invites})
	if errors.Is(err, course.ErrFileTooLarge) {
		apierr.Write(w, r, apierr.New(http.StatusRequestEntityTooLarge, apierr.PayloadTooLarge).WithDetails(map[string]any{"max_bytes": course.RosterMaxFileBytes}))
		return
	}
	if h.fail(w, r, "roster-import", err) {
		return
	}
	out := rosterOut{Total: rep.Total, CreatedUsers: rep.CreatedUsers, LinkedExisting: rep.LinkedExisting, AlreadyMember: rep.AlreadyMember,
		PendingUnverified: rep.PendingUnverified, SkippedRemoved: rep.SkippedRemoved, DryRun: dry, Errors: make([]rowErrorOut, len(rep.Errors))}
	for i, e := range rep.Errors {
		out.Errors[i] = rowErrorOut{Row: e.Row, Field: e.Field, Code: e.Code, Message: e.Message}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

type shareSourceOut struct {
	ID        uuid.UUID `json:"id"`
	ClassCode string    `json:"class_code"`
	Name      string    `json:"name"`
	Semester  string    `json:"semester"`
	Documents int       `json:"documents"`
}

func (h *Handler) shareSources(w http.ResponseWriter, r *http.Request) {
	courseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	teacher, ok := actorOf(w, r)
	if !ok {
		return
	}
	rows, err := h.Courses.ShareSources(r.Context(), courseID, teacher)
	if h.fail(w, r, "share-sources", err) {
		return
	}
	items := make([]shareSourceOut, len(rows))
	for i, s := range rows {
		items[i] = shareSourceOut{ID: s.ID, ClassCode: s.ClassCode, Name: s.Name, Semester: s.Semester, Documents: s.Documents}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

type shareBody struct {
	SourceCourseID string   `json:"source_course_id"`
	What           []string `json:"what"`
}

type skippedOut struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

func (h *Handler) shareFrom(w http.ResponseWriter, r *http.Request) {
	courseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	var b shareBody
	if !httpx.DecodeJSON(w, r, &b) {
		return
	}
	teacher, ok := actorOf(w, r)
	if !ok {
		return
	}
	src, err := uuid.Parse(strings.TrimSpace(b.SourceCourseID))
	if err != nil {
		apierr.Write(w, r, apierr.Validation(apierr.FieldError{Field: "source_course_id", Code: "INVALID_ID", Message: "Mã lớp nguồn không hợp lệ."}))
		return
	}
	res, err := h.Courses.ShareFrom(r.Context(), courseID, teacher, src, b.What)
	if h.fail(w, r, "share-from", err) {
		return
	}
	sk := make([]skippedOut, len(res.Skipped))
	for i, s := range res.Skipped {
		sk[i] = skippedOut{Kind: s.Kind, Reason: s.Reason}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"shared": map[string]int{"documents": res.Documents}, "skipped": sk})
}
