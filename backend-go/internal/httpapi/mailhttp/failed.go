// Package mailhttp: danh sách thư gửi lỗi cho ADMIN (US-P4-04 AC9). Không có đường nào trả thân thư hay payload.
package mailhttp

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/store"
)

// Handler phục vụ `GET /admin/observability/failed-mail`.
type Handler struct {
	Pool *pgxpool.Pool
	Log  *slog.Logger
}

// Mount đăng ký trong nhóm đã qua auth.Middleware; chỉ ADMIN.
func (h *Handler) Mount(r chi.Router) {
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/admin/observability/failed-mail", h.list)
}

// Row là một thư DEAD: người nhận đã che, không thân thư.
type Row struct {
	ID        uuid.UUID `json:"id"`
	Template  string    `json:"template"`
	ToMasked  string    `json:"to_masked"`
	Attempts  int32     `json:"attempts"`
	LastError *string   `json:"last_error"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Mask: chữ đầu + `***` + `@domain` ("a***@domain").
func Mask(addr string) string {
	i := strings.LastIndex(addr, "@")
	if i <= 0 {
		return "***"
	}
	return addr[:1] + "***" + addr[i:]
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	pg, aerr := httpx.ParsePageParams(r)
	if aerr != nil {
		apierr.Write(w, r, aerr)
		return
	}
	arg := store.MailFailedPageParams{PageLimit: int32(pg.Fetch())} //nolint:gosec // ≤ 101
	if pg.Cursor != nil {
		arg.CursorAt, arg.CursorID = &pg.Cursor.CreatedAt, &pg.Cursor.ID
	}
	rows, err := store.New(h.Pool).MailFailedPage(r.Context(), arg)
	if err != nil {
		h.Log.ErrorContext(r.Context(), "mail: danh sách thư lỗi", "error", err.Error())
		apierr.Write(w, r, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable))
		return
	}
	out := struct {
		Items      []Row   `json:"items"`
		NextCursor *string `json:"next_cursor"`
	}{Items: []Row{}}
	if len(rows) > pg.Limit {
		rows = rows[:pg.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].UpdatedAt, rows[len(rows)-1].ID.String())
		out.NextCursor = &c
	}
	for _, m := range rows {
		out.Items = append(out.Items, Row{ID: m.ID, Template: m.Template, ToMasked: Mask(m.ToAddr), Attempts: m.Attempts, LastError: m.LastError, UpdatedAt: m.UpdatedAt})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
