//go:build testroutes

package testroutes

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// itemsPath là tiền tố `Location` của bản ghi vừa tạo (SRS 6.3 mục 2).
const itemsPath = "/api/v1/_test/items"

// item là biểu diễn JSON của một dòng `_test_items`.
type item struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const itemCols = `id, name, version, created_at, updated_at`

// Truy vấn phân trang: so sánh hàng `(created_at, id) <` + index `(created_at, id)`, không bao giờ bỏ qua dòng.
const (
	listFirstSQL = `select ` + itemCols + ` from _test_items where owner_id = $1
		order by created_at desc, id desc limit $2`
	listAfterSQL = `select ` + itemCols + ` from _test_items
		where owner_id = $1 and (created_at, id) < ($3::timestamptz, $4::uuid)
		order by created_at desc, id desc limit $2`
	getSQL    = `select ` + itemCols + ` from _test_items where id = $1 and owner_id = $2`
	insertSQL = `insert into _test_items (name, owner_id) values ($1, $2) returning ` + itemCols
	updateSQL = `update _test_items set name = $1, version = version + 1, updated_at = now()
		where id = $2 and owner_id = $3 and version = $4 returning ` + itemCols
)

// listItems: `GET /_test/items?cursor=&limit=` — chỉ bản ghi của chính người gọi, ETag băm thân (SRS 6.4, 6.5).
func listItems(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, e := ownerID(r)
		if e != nil {
			apierr.Write(w, r, e)
			return
		}
		p, e := httpx.ParsePageParams(r)
		if e != nil {
			apierr.Write(w, r, e)
			return
		}
		rows, err := queryItems(r.Context(), d, owner, p)
		if err != nil {
			writeDBError(w, r, err)
			return
		}
		page := httpx.Paginate(rows, p, func(it item) (time.Time, string) { return it.CreatedAt, it.ID })
		httpx.WriteJSONETag(w, r, page, "")
	}
}

func queryItems(ctx context.Context, d Deps, owner uuid.UUID, p httpx.PageParams) ([]item, error) {
	rows, err := func() (pgx.Rows, error) {
		if p.Cursor == nil {
			return d.DB.Query(ctx, listFirstSQL, owner, p.Fetch())
		}
		return d.DB.Query(ctx, listAfterSQL, owner, p.Fetch(), p.Cursor.CreatedAt, p.Cursor.ID)
	}()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// createItem: `POST /_test/items` (Idempotency-Key bắt buộc) → 201 + Location (SRS 6.3 mục 2).
func createItem(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, e := ownerID(r)
		if e != nil {
			apierr.Write(w, r, e)
			return
		}
		var in struct {
			Name string `json:"name" validate:"required,max=200"`
		}
		if !httpx.DecodeJSON(w, r, &in) {
			return
		}
		it, err := scanItem(d.DB.QueryRow(r.Context(), insertSQL, in.Name, owner))
		if err != nil {
			writeDBError(w, r, err)
			return
		}
		w.Header().Set("Location", itemsPath+"/"+it.ID)
		w.Header().Set("ETag", httpx.ETagVersion(it.Version))
		httpx.WriteJSON(w, http.StatusCreated, it)
	}
}

// getItem: `GET /_test/items/{id}` — bản ghi của người khác → 404; ETag `W/"v<version>"` + 304 (03-AC15).
func getItem(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		it, e := loadItem(r, d)
		if e != nil {
			apierr.Write(w, r, e)
			return
		}
		httpx.WriteJSONETag(w, r, it, httpx.ETagVersion(it.Version))
	}
}

// putItem: `PUT /_test/items/{id}` — khoá lạc quan theo `version` / `If-Match` (03-AC14).
func putItem(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, e := ownerID(r)
		if e != nil {
			apierr.Write(w, r, e)
			return
		}
		id, e := pathID(r)
		if e != nil {
			apierr.Write(w, r, e)
			return
		}
		var in struct {
			Name    string `json:"name" validate:"required,max=200"`
			Version *int   `json:"version" validate:"omitnil,min=1"`
		}
		if !httpx.DecodeJSON(w, r, &in) {
			return
		}
		want, e := httpx.WantedVersion(r, in.Version)
		if e != nil {
			apierr.Write(w, r, e)
			return
		}

		updated, err := scanItem(d.DB.QueryRow(r.Context(), updateSQL, in.Name, id, owner, want))
		switch {
		case err == nil:
			w.Header().Set("ETag", httpx.ETagVersion(updated.Version))
			httpx.WriteJSON(w, http.StatusOK, updated)
		case errors.Is(err, pgx.ErrNoRows):
			// 0 dòng: hoặc bản ghi không thuộc về người gọi / không tồn tại (404), hoặc sai version (409).
			current, err := scanItem(d.DB.QueryRow(r.Context(), getSQL, id, owner))
			if err != nil {
				writeDBError(w, r, err)
				return
			}
			httpx.WriteVersionConflict(w, r, current.Version, current)
		default:
			writeDBError(w, r, err)
		}
	}
}

func loadItem(r *http.Request, d Deps) (item, *apierr.Error) {
	owner, e := ownerID(r)
	if e != nil {
		return item{}, e
	}
	id, e := pathID(r)
	if e != nil {
		return item{}, e
	}
	it, err := scanItem(d.DB.QueryRow(r.Context(), getSQL, id, owner))
	if err != nil {
		return item{}, dbError(err)
	}
	return it, nil
}

// rowScanner gom pgx.Row và pgx.Rows (cùng chữ ký Scan).
type rowScanner interface{ Scan(dest ...any) error }

func scanItem(row rowScanner) (item, error) {
	var (
		it item
		id uuid.UUID
	)
	if err := row.Scan(&id, &it.Name, &it.Version, &it.CreatedAt, &it.UpdatedAt); err != nil {
		return item{}, err
	}
	it.ID = id.String()
	return it, nil
}

// ownerID là `sub` của người gọi: mọi bản ghi thử đều thuộc về đúng một người (#Q-QC-03-2).
func ownerID(r *http.Request) (uuid.UUID, *apierr.Error) {
	p := auth.MustFromContext(r.Context())
	id, err := uuid.Parse(p.Sub)
	if err != nil {
		return uuid.Nil, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	return id, nil
}

// pathID đọc `{id}`; không phải uuid → 404 (không lộ sự tồn tại).
func pathID(r *http.Request) (uuid.UUID, *apierr.Error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	return id, nil
}

func writeDBError(w http.ResponseWriter, r *http.Request, err error) {
	if e := dbError(err); e != nil {
		apierr.Write(w, r, e)
	}
}

// dbError đổi lỗi truy vấn thành lỗi API; ctx bị huỷ → nil (middleware timeout trả 504).
func dbError(err error) *apierr.Error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return apierr.New(http.StatusNotFound, apierr.NotFound)
	default:
		return apierr.New(http.StatusInternalServerError, apierr.Internal)
	}
}
