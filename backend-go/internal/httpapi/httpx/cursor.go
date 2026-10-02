package httpx

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/google/uuid"
)

// Giới hạn `limit` của mọi danh sách (SRS 6.4).
const (
	DefaultLimit = 30
	MaxLimit     = 100
)

// cursorVersion là `v` trong thân cursor; giá trị khác → 422 INVALID_CURSOR.
const cursorVersion = 1

// ErrInvalidCursor: chuỗi cursor không giải mã được (sai base64url, JSON hỏng, `v` lạ, `t`/`i` sai kiểu).
var ErrInvalidCursor = errors.New("httpx: cursor không hợp lệ")

// Cursor trỏ tới dòng CUỐI của trang trước; thứ tự danh sách luôn là (created_at DESC, id DESC).
type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// cursorJSON là dạng đi trên dây: {"v":1,"t":<micro-giây UTC>,"i":"<uuid>"} (SRS 6.4).
type cursorJSON struct {
	V int    `json:"v"`
	T int64  `json:"t"`
	I string `json:"i"`
}

// EncodeCursor dựng cursor base64url không đệm từ `created_at` + `id` của dòng cuối trang.
func EncodeCursor(createdAt time.Time, id string) string {
	raw, _ := json.Marshal(cursorJSON{V: cursorVersion, T: createdAt.UTC().UnixMicro(), I: id})
	return base64.RawURLEncoding.EncodeToString(raw)
}

// ParseCursor giải mã chuỗi cursor của client.
func ParseCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	var c cursorJSON
	if err := json.Unmarshal(raw, &c); err != nil || c.V != cursorVersion {
		return Cursor{}, ErrInvalidCursor
	}
	id, err := uuid.Parse(c.I)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{CreatedAt: time.UnixMicro(c.T).UTC(), ID: id}, nil
}

// PageParams là `?limit=&cursor=` đã kiểm của một request danh sách.
type PageParams struct {
	Limit  int
	Cursor *Cursor // nil = trang đầu
}

// Fetch là số dòng cần truy vấn (limit + 1) để biết còn trang sau hay không.
func (p PageParams) Fetch() int { return p.Limit + 1 }

// ParsePageParams đọc `limit` (1..100, mặc định 30) và `cursor` từ query (SRS 6.4).
// `limit` sai → 422 VALIDATION_FAILED field "limit"; cursor sai → 422 INVALID_CURSOR.
func ParsePageParams(r *http.Request) (PageParams, *apierr.Error) {
	p := PageParams{Limit: DefaultLimit}
	q := r.URL.Query()
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			return p, apierr.Validation(apierr.FieldError{
				Field: "limit", Code: "range",
				Message: "limit phải là số nguyên từ 1 đến " + strconv.Itoa(MaxLimit) + ".",
			})
		}
		p.Limit = n
	}
	if raw := q.Get("cursor"); raw != "" {
		c, err := ParseCursor(raw)
		if err != nil {
			return p, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidCursor)
		}
		p.Cursor = &c
	}
	return p, nil
}

// Page là thân chuẩn của mọi danh sách: `items` luôn là mảng, `next_cursor` luôn có mặt (null ở trang cuối).
type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// Paginate cắt kết quả đã truy vấn với LIMIT p.Fetch() thành một trang và dựng `next_cursor`
// từ dòng cuối của trang (key trả về `created_at` + `id` của một dòng).
func Paginate[T any](rows []T, p PageParams, key func(T) (time.Time, string)) Page[T] {
	items := rows
	var next *string
	if len(rows) > p.Limit {
		items = rows[:p.Limit]
		c := EncodeCursor(key(items[p.Limit-1]))
		next = &c
	}
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, NextCursor: next}
}
