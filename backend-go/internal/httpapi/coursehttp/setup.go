package coursehttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
)

// dismissSetup: giảng viên bỏ qua mục "Thiết lập lớp mới" ở Hôm nay. 204, idempotent.
func (h *Handler) dismissSetup(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	teacher, ok := actorOf(w, r)
	if !ok {
		return
	}
	if h.fail(w, r, "setup-dismiss", h.Courses.DismissSetup(r.Context(), teacher, id)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
