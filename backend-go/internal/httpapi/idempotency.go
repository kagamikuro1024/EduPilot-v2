package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// idemHeader là tên header; GIÁ TRỊ của nó không bao giờ được ghi log (US-PG-03 AC12).
const idemHeader = "Idempotency-Key"

// idemScopeLen là số ký tự hex của sha256(endpoint) đi vào khoá Redis (SRS 5.6).
const idemScopeLen = 16

// Gửi đôi gần như đồng thời: chờ bản chạy trước tối đa idemWaitTotal trước khi trả 409.
const (
	idemWaitStep  = 50 * time.Millisecond
	idemWaitTotal = time.Second
)

// idemRecord là phản hồi đã lưu (giá trị của khoá `ep:idem:…`). Body giữ nguyên byte (JSON mã hoá base64).
type idemRecord struct {
	RequestHash string `json:"request_hash"`
	Status      int    `json:"status"`
	ContentType string `json:"content_type"`
	Body        []byte `json:"body"`
}

// idemScope là khoá logic (user_id, endpoint, key) + hash thân của một request.
type idemScope struct {
	user     uuid.UUID
	endpoint string // "POST /api/v1/_test/items"
	key      string
	hash     string
	respKey  string
	lockKey  string
}

// RequireIdempotencyKey bắt buộc header `Idempotency-Key` cho một route (SRS 6.6). Phải đặt SAU auth middleware:
// khoá logic là (user_id, endpoint, key). Gửi lại cùng khoá + cùng thân → phát lại nguyên phản hồi cũ;
// cùng khoá + thân khác → 422; đang chạy → 409; Redis chết → 503 (fail-closed: không bao giờ xử lý hai lần).
func RequireIdempotencyKey(d Deps) func(http.Handler) http.Handler {
	keyRE := regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(idemHeader)
			if key == "" {
				apierr.Write(w, r, apierr.New(http.StatusUnprocessableEntity, apierr.IdempotencyRequired))
				return
			}
			if !keyRE.MatchString(key) {
				apierr.Write(w, r, apierr.Validation(apierr.FieldError{
					Field: idemHeader, Code: "format",
					Message: "Khoá phải dài 8–128 ký tự trong [A-Za-z0-9._:-].",
				}))
				return
			}
			p, pok := auth.FromContext(r.Context())
			userID, err := uuid.Parse(p.Sub)
			if !pok || err != nil {
				apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
				return
			}
			body, ok := idemReadBody(w, r)
			if !ok {
				return
			}
			idemServe(d, next, w, r, idemScopeOf(r, userID, key, body))
		})
	}
}

// idemReadBody đọc hết thân (bodyLimitMiddleware đã chặn trên MAX_BODY_BYTES) rồi trả lại cho handler.
func idemReadBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	if r.Body == nil {
		return nil, true
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			apierr.Write(w, r, apierr.New(http.StatusRequestEntityTooLarge, apierr.PayloadTooLarge))
			return nil, false
		}
		apierr.Write(w, r, apierr.New(http.StatusBadRequest, apierr.BadRequest))
		return nil, false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, true
}

// idemScopeOf dựng khoá logic + request_hash = sha256(thân + query đã chuẩn hoá).
func idemScopeOf(r *http.Request, userID uuid.UUID, key string, body []byte) idemScope {
	pattern := ""
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		pattern = rctx.RoutePattern()
	}
	if pattern == "" {
		pattern = r.URL.Path
	}
	endpoint := r.Method + " " + pattern

	h := sha256.New()
	h.Write(body)
	h.Write([]byte("\n"))
	h.Write([]byte(r.URL.Query().Encode())) // Encode() sắp xếp theo tên tham số → chuẩn hoá
	epSum := sha256.Sum256([]byte(endpoint))

	resp := redis.Key("idem", userID.String(), hex.EncodeToString(epSum[:])[:idemScopeLen], key)
	return idemScope{
		user: userID, endpoint: endpoint, key: key,
		hash:    hex.EncodeToString(h.Sum(nil)),
		respKey: resp, lockKey: resp + ":lock",
	}
}

// idemServe chạy đúng quy trình 6 bước của SRS 6.6.
func idemServe(d Deps, next http.Handler, w http.ResponseWriter, r *http.Request, sc idemScope) {
	ctx := r.Context()
	if d.Redis == nil {
		idemUnavailable(d, w, r)
		return
	}

	rec, err := idemFromRedis(ctx, d, sc)
	if err != nil {
		idemUnavailable(d, w, r)
		return
	}
	if rec != nil {
		idemReplay(w, r, sc, rec)
		return
	}

	locked, err := d.Redis.SetNX(ctx, sc.lockKey, "1", redis.TTLIdempotencyLock).Result()
	if err != nil {
		idemUnavailable(d, w, r)
		return
	}
	if !locked {
		// "Hai tab gửi trùng": chờ ngắn cho bản chạy trước ghi xong rồi phát lại; vẫn chạy dở → 409.
		if rec := idemWait(ctx, d, sc); rec != nil {
			idemReplay(w, r, sc, rec)
			return
		}
		apierr.Write(w, r, apierr.New(http.StatusConflict, apierr.IdempotencyInProg).WithRetryAfter(1))
		return
	}
	defer func() {
		if err := d.Redis.Del(context.WithoutCancel(ctx), sc.lockKey).Err(); err != nil && d.Log != nil {
			d.Log.WarnContext(ctx, "không nhả được khoá idempotency", "endpoint", sc.endpoint, "error", err.Error())
		}
	}()

	// (4) Redis có thể đã mất dữ liệu: bảng idempotency_keys là bản bền.
	if rec := idemFromTable(ctx, d, sc); rec != nil {
		idemToRedis(ctx, d, sc, rec)
		idemReplay(w, r, sc, rec)
		return
	}

	// (5) chạy handler, đệm phản hồi để lưu lại được.
	rw := &idemRecorder{ResponseWriter: w, limit: d.Cfg.MaxBodyBytes}
	next.ServeHTTP(rw, r)
	if !rw.wrote {
		return // handler không ghi gì (ví dụ deadline) — middleware ngoài xử lý tiếp
	}

	// (6) chỉ lưu phản hồi < 500 và ≠ 429; còn lại chỉ nhả khoá (gửi lại sẽ xử lý lại).
	if !rw.overflow && rw.code < http.StatusInternalServerError && rw.code != http.StatusTooManyRequests {
		idemSave(ctx, d, sc, &idemRecord{
			RequestHash: sc.hash, Status: rw.code,
			ContentType: w.Header().Get("Content-Type"), Body: rw.buf.Bytes(),
		})
	}
	rw.flush()
}

// idemUnavailable: Redis không trả lời → 503 (fail-closed, SRS 3.4 + 6.6).
func idemUnavailable(d Deps, w http.ResponseWriter, r *http.Request) {
	if d.Log != nil {
		d.Log.WarnContext(r.Context(), "idempotency fail-closed: Redis không khả dụng",
			"method", r.Method, "path", r.URL.Path)
	}
	apierr.Write(w, r, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable).
		WithDetails(map[string]any{"redis": "down"}))
}

// idemWait chờ tối đa idemWaitTotal cho bản chạy trước lưu xong phản hồi (gửi đôi "hai tab" gần như luôn
// kết thúc trong vài chục ms). Hết thời gian mà chưa có → nil để người gọi trả 409.
func idemWait(ctx context.Context, d Deps, sc idemScope) *idemRecord {
	for waited := time.Duration(0); waited < idemWaitTotal; waited += idemWaitStep {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(idemWaitStep):
		}
		rec, err := idemFromRedis(ctx, d, sc)
		if err != nil {
			return nil
		}
		if rec != nil {
			return rec
		}
	}
	return nil
}

// idemFromRedis đọc phản hồi đã lưu; (nil, nil) = chưa có; lỗi ≠ "không có khoá" = Redis chết.
func idemFromRedis(ctx context.Context, d Deps, sc idemScope) (*idemRecord, error) {
	raw, err := d.Redis.Get(ctx, sc.respKey).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec idemRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, nil // bản lưu hỏng: coi như chưa có, xử lý lại dưới khoá
	}
	return &rec, nil
}

// idemFromTable đọc bản bền; lỗi DB → nil (handler phía sau cũng cần DB nên sẽ tự hỏng đúng cách).
func idemFromTable(ctx context.Context, d Deps, sc idemScope) *idemRecord {
	if d.DB == nil {
		return nil
	}
	row, err := store.New(d.DB).GetIdempotencyKey(ctx, store.GetIdempotencyKeyParams{
		UserID: sc.user, Endpoint: sc.endpoint, Key: sc.key,
	})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) && d.Log != nil {
			d.Log.WarnContext(ctx, "không đọc được idempotency_keys", "endpoint", sc.endpoint, "error", err.Error())
		}
		return nil
	}
	return &idemRecord{
		RequestHash: row.RequestHash, Status: int(row.StatusCode),
		ContentType: "application/json; charset=utf-8", Body: row.Response,
	}
}

// idemToRedis khôi phục bản bền vào Redis (bước 4 của SRS 6.6).
func idemToRedis(ctx context.Context, d Deps, sc idemScope, rec *idemRecord) {
	raw, err := json.Marshal(rec)
	if err != nil {
		return
	}
	if err := d.Redis.Set(ctx, sc.respKey, raw, redis.TTLIdempotency).Err(); err != nil && d.Log != nil {
		d.Log.WarnContext(ctx, "không khôi phục được phản hồi idempotency vào Redis",
			"endpoint", sc.endpoint, "error", err.Error())
	}
}

// idemSave ghi phản hồi vào Redis (24 h) và bảng (ON CONFLICT DO NOTHING).
func idemSave(ctx context.Context, d Deps, sc idemScope, rec *idemRecord) {
	idemToRedis(ctx, d, sc, rec)
	if d.DB == nil || !json.Valid(rec.Body) {
		return // cột `response` là jsonb: chỉ lưu được thân JSON
	}
	if _, err := store.New(d.DB).InsertIdempotencyKey(ctx, store.InsertIdempotencyKeyParams{
		UserID: sc.user, Endpoint: sc.endpoint, Key: sc.key, RequestHash: sc.hash,
		StatusCode: int16(rec.Status), Response: rec.Body,
	}); err != nil && d.Log != nil {
		d.Log.WarnContext(ctx, "không ghi được idempotency_keys", "endpoint", sc.endpoint, "error", err.Error())
	}
}

// idemReplay phát lại phản hồi cũ; thân khác với lần đầu → 422 IDEMPOTENCY_KEY_REUSED.
func idemReplay(w http.ResponseWriter, r *http.Request, sc idemScope, rec *idemRecord) {
	if rec.RequestHash != sc.hash {
		apierr.Write(w, r, apierr.New(http.StatusUnprocessableEntity, apierr.IdempotencyReused))
		return
	}
	h := w.Header()
	if rec.ContentType != "" {
		h.Set("Content-Type", rec.ContentType)
	}
	h.Set("Idempotent-Replayed", "true")
	w.WriteHeader(rec.Status)
	_, _ = w.Write(rec.Body)
}

// idemRecorder đệm phản hồi của handler để lưu lại được. Vượt `limit` (MAX_BODY_BYTES) thì xả thẳng ra client
// và bỏ việc lưu — không giữ phản hồi lớn trong RAM.
type idemRecorder struct {
	http.ResponseWriter
	limit    int64
	buf      bytes.Buffer
	code     int
	wrote    bool
	overflow bool
}

func (rc *idemRecorder) WriteHeader(code int) {
	if rc.wrote {
		return
	}
	rc.code, rc.wrote = code, true
	if rc.overflow {
		rc.ResponseWriter.WriteHeader(code)
	}
}

func (rc *idemRecorder) Write(b []byte) (int, error) {
	if !rc.wrote {
		rc.WriteHeader(http.StatusOK)
	}
	if rc.overflow {
		return rc.ResponseWriter.Write(b)
	}
	if int64(rc.buf.Len()+len(b)) > rc.limit {
		rc.overflow = true
		rc.ResponseWriter.WriteHeader(rc.code)
		if _, err := rc.ResponseWriter.Write(rc.buf.Bytes()); err != nil {
			return 0, err
		}
		rc.buf.Reset()
		return rc.ResponseWriter.Write(b)
	}
	return rc.buf.Write(b)
}

// flush ghi phản hồi đã đệm ra client (gọi một lần, sau khi đã lưu).
func (rc *idemRecorder) flush() {
	if rc.overflow {
		return
	}
	rc.ResponseWriter.WriteHeader(rc.code)
	_, _ = rc.ResponseWriter.Write(rc.buf.Bytes())
}

// Unwrap cho http.ResponseController (handler phía sau có thể cần Flush).
func (rc *idemRecorder) Unwrap() http.ResponseWriter { return rc.ResponseWriter }
