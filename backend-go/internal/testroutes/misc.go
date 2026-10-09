//go:build testroutes

package testroutes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// errorRetryAfter là `retry_after` cố định của `GET /_test/error/429|503` (SRS 6.1).
const errorRetryAfter = 7

// slowHandler ngủ `?ms=` rồi trả 200 — dùng để kiểm tắt êm (request đang chạy phải xong).
func slowHandler(w http.ResponseWriter, r *http.Request) {
	ms, _ := strconv.Atoi(r.URL.Query().Get("ms"))
	select {
	case <-time.After(time.Duration(ms) * time.Millisecond):
	case <-r.Context().Done():
		return // middleware timeout trả 504 DEADLINE_EXCEEDED
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// dbSleepHandler đẩy deadline của request xuống tận Postgres (`pg_sleep` bị huỷ khi ctx hết hạn).
func dbSleepHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		seconds, err := strconv.ParseFloat(r.URL.Query().Get("seconds"), 64)
		if err != nil || seconds < 0 {
			apierr.Write(w, r, apierr.New(http.StatusBadRequest, apierr.BadRequest))
			return
		}
		var n int
		if err := d.DB.QueryRow(r.Context(),
			"-- name: TestDBSleep :one\nselect 1 from pg_sleep($1::float)", seconds).Scan(&n); err != nil {
			return // ctx hết hạn / bị huỷ → middleware timeout trả 504
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]int{"slept": int(seconds)})
	}
}

// redisBlockHandler chặn trên một lệnh Redis để kiểm deadline xuống Redis (BLPOP huỷ theo ctx).
func redisBlockHandler(d Deps) http.HandlerFunc {
	key := redis.Key("test", "block")
	return func(w http.ResponseWriter, r *http.Request) {
		seconds, err := strconv.ParseFloat(r.URL.Query().Get("seconds"), 64)
		if err != nil || seconds < 0 {
			apierr.Write(w, r, apierr.New(http.StatusBadRequest, apierr.BadRequest))
			return
		}
		if err := d.Redis.BLPop(r.Context(), time.Duration(seconds*float64(time.Second)), key).Err(); err != nil {
			return // ctx hết hạn / bị huỷ → 504
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// panicHandler kiểm middleware recover (03-AC2).
func panicHandler(http.ResponseWriter, *http.Request) {
	panic("route thử /_test/panic")
}

// errorHandler trả đúng mã mặc định của status theo bảng SRS 6.1; status ngoài bảng → 404.
func errorHandler(w http.ResponseWriter, r *http.Request) {
	status, err := strconv.Atoi(chi.URLParam(r, "status"))
	if err != nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	e := apierr.ByStatus(status)
	if e == nil {
		apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
		return
	}
	if status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable {
		e = e.WithRetryAfter(errorRetryAfter)
	}
	apierr.Write(w, r, e)
}

// whoami trả danh tính lấy từ claim — KHÔNG truy DB (04-AC4).
func whoami(w http.ResponseWriter, r *http.Request) {
	p := auth.MustFromContext(r.Context())
	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"sub": p.Sub, "role": string(p.Role), "email": p.Email, "jti": p.JTI,
	})
}

// chatGate (US-PE-07 AC2): mẫu cổng chat cho P3 — gọi `exam.Locker` và trả `{allowed: !locked}`; lỗi cả Redis lẫn DB → `allowed:false` (từ chối khi nghi ngờ).
func chatGate(d Deps) http.HandlerFunc {
	lk := &exam.Locker{Pool: d.DB, Redis: d.Redis, Clock: d.Clock}
	return func(w http.ResponseWriter, r *http.Request) {
		p := auth.MustFromContext(r.Context())
		uid, err := uuid.Parse(p.Sub)
		if err != nil {
			apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
			return
		}
		_, locked, err := lk.IsLocked(r.Context(), uid)
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"allowed": err == nil && !locked})
	}
}
