// Package jobs: việc dài (SRS 4.3 FR-35, 5.4). Gateway xếp hàng việc (dòng `jobs` + dòng `outbox` trong CÙNG
// transaction) và trả 202; worker chạy việc qua `Runner` (runner.go) rồi phát tiến độ qua SSE.
// Gói này KHÔNG import `internal/httpapi` (httpapi mount handler của gói này).
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// TopicEnqueue là topic outbox của một việc mới xếp hàng; worker đăng ký
// `registry.Register(jobs.TopicEnqueue, runner.HandleMessage)`.
const TopicEnqueue = "job.enqueue"

// Service là phía gateway: xếp hàng việc và đọc trạng thái việc.
type Service struct{ DB *pgxpool.Pool }

// NewService dựng Service trên pool đã có.
func NewService(db *pgxpool.Pool) *Service { return &Service{DB: db} }

// enqueuePayload là thân dòng outbox `job.enqueue`.
type enqueuePayload struct {
	JobID   uuid.UUID       `json:"job_id"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

// Enqueue ghi dòng `jobs` (QUEUED) và dòng `outbox` `job.enqueue` trong MỘT transaction (luật 14):
// commit cùng, rollback cùng — không bao giờ có việc không ai chạy, cũng không có tin cho việc không tồn tại.
func (s *Service) Enqueue(ctx context.Context, ownerID uuid.UUID, kind string, payload any) (store.Job, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return store.Job{}, fmt.Errorf("jobs enqueue: marshal payload: %w", err)
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return store.Job{}, fmt.Errorf("jobs enqueue: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	j, err := store.New(tx).InsertJob(ctx, store.InsertJobParams{Kind: kind, OwnerID: ownerID})
	if err != nil {
		return store.Job{}, fmt.Errorf("jobs enqueue: insert job: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, TopicEnqueue, enqueuePayload{JobID: j.ID, Kind: kind, Payload: raw}); err != nil {
		return store.Job{}, fmt.Errorf("jobs enqueue: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return store.Job{}, fmt.Errorf("jobs enqueue: commit: %w", err)
	}
	return j, nil
}

// Get trả việc nếu người gọi được xem (chủ việc hoặc ADMIN). Mọi trường hợp khác — id sai, việc không có,
// việc của người khác — trả pgx.ErrNoRows để người gọi trả 404 (không lộ sự tồn tại; US-PG-03 AC18).
func (s *Service) Get(ctx context.Context, p auth.Principal, id string) (store.Job, error) {
	jobID, err := uuid.Parse(id)
	if err != nil {
		return store.Job{}, pgx.ErrNoRows
	}
	j, err := store.New(s.DB).GetJob(ctx, jobID)
	if err != nil {
		return store.Job{}, err
	}
	if p.Role != auth.RoleAdmin && j.OwnerID.String() != p.Sub {
		return store.Job{}, pgx.ErrNoRows
	}
	return j, nil
}

// view là thân phản hồi của `GET /api/v1/jobs/{id}` (SRS 6.2) — không thêm trường nào khác.
type view struct {
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	Status     string          `json:"status"`
	Progress   int             `json:"progress"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      json.RawMessage `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
}

func newView(j store.Job) view {
	return view{
		ID: j.ID.String(), Kind: j.Kind, Status: string(j.Status), Progress: int(j.Progress),
		Result: jsonOrNil(j.Result), Error: jsonOrNil(j.Error),
		CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt, FinishedAt: j.FinishedAt,
	}
}

// jsonOrNil bỏ cột jsonb rỗng hoặc JSON `null` để `result`/`error` vắng mặt thay vì là null (SRS 6.2).
func jsonOrNil(b json.RawMessage) json.RawMessage {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return b
}

// Handler là `GET /api/v1/jobs/{id}`: cần đăng nhập (auth.Middleware chạy trước); chủ việc hoặc ADMIN → 200,
// còn lại → 404 NOT_FOUND.
func Handler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.FromContext(r.Context())
		if !ok {
			apierr.Write(w, r, apierr.New(http.StatusUnauthorized, apierr.Unauthenticated))
			return
		}
		j, err := svc.Get(r.Context(), p, chi.URLParam(r, "id"))
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			apierr.Write(w, r, apierr.New(http.StatusNotFound, apierr.NotFound))
			return
		case err != nil:
			apierr.Write(w, r, apierr.New(http.StatusInternalServerError, apierr.Internal))
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(newView(j))
	}
}
