//go:build testroutes

package testroutes

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/httpapi/sse"
)

// publishEvent: `POST /_test/events` — phát một sự kiện SSE cho chính mình;
// `user_id` khác mình chỉ ADMIN được (SRS 6.3 mục 15, #Q-QC-05-3).
func publishEvent(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := auth.MustFromContext(r.Context())
		var in struct {
			Type   string          `json:"type" validate:"required"`
			Data   json.RawMessage `json:"data" validate:"required"`
			UserID string          `json:"user_id" validate:"omitempty,uuid"`
		}
		if !httpx.DecodeJSON(w, r, &in) {
			return
		}
		target := p.Sub
		if in.UserID != "" && in.UserID != p.Sub {
			if p.Role != auth.RoleAdmin {
				apierr.Write(w, r, apierr.New(http.StatusForbidden, apierr.Forbidden).
					WithDetails(map[string]string{"reason": "role"}))
				return
			}
			target = in.UserID
		}

		id, err := d.Publisher.Publish(r.Context(), target, in.Type, in.Data)
		switch {
		case err == nil:
			httpx.WriteJSON(w, http.StatusAccepted, map[string]string{"event_id": id})
		case errors.Is(err, sse.ErrInvalidType):
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{
				Field: "type", Code: "format", Message: "type phải khớp ^[a-z][a-z0-9_.]{0,63}$.",
			}))
		case errors.Is(err, sse.ErrDataTooLarge):
			apierr.Write(w, r, apierr.Validation(apierr.FieldError{
				Field: "data", Code: "max", Message: "data vượt quá 64 KiB.",
			}))
		default:
			apierr.Write(w, r, apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable))
		}
	}
}
