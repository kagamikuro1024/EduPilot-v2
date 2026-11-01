package mailin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/submission/source"
)

// TeamsEnv là cấu hình Graph của máy chủ (GRAPH_*, TEAMS_*).
type TeamsEnv struct {
	Enabled bool
	Source  source.Config
}

// TeamsPublic là phần không bí mật của cấu hình Teams.
type TeamsPublic struct {
	TenantID string `json:"tenant_id"`
	ClientID string `json:"client_id"`
}

// TeamsPutIn là thân `PUT …/integrations/teams`. ClientSecret rỗng = giữ cũ.
type TeamsPutIn struct {
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Enabled      bool   `json:"enabled"`
	Version      int32  `json:"version"`
}

// TeamsSandbox: GRAPH_BASE_URL không trỏ graph.microsoft.com → nhãn "Môi trường giả lập".
func (e TeamsEnv) Sandbox() bool {
	u, err := url.Parse(e.Source.BaseURL)
	return err != nil || !strings.EqualFold(u.Hostname(), "graph.microsoft.com")
}

func (s *Service) teamsView(ctx context.Context, course uuid.UUID) (View, error) {
	sandbox, avail := s.Teams.Sandbox(), s.Teams.Enabled
	r, err := store.New(s.Pool).IntegrationGet(ctx, store.IntegrationGetParams{CourseID: course, Kind: KindTeams})
	if errors.Is(err, pgx.ErrNoRows) {
		return View{Kind: KindTeams, Status: "NOT_CONNECTED", Sandbox: &sandbox, Available: &avail}, nil
	}
	if err != nil {
		return View{}, fmt.Errorf("mailin: đọc cấu hình Teams: %w", err)
	}
	var pub TeamsPublic
	if err := json.Unmarshal(r.ConfigPublic, &pub); err != nil {
		return View{}, fmt.Errorf("mailin: cấu hình Teams hỏng: %w", err)
	}
	v := View{Kind: KindTeams, Configured: true, Enabled: r.Enabled, Config: pub, LastOKAt: r.LastOkAt, LastError: r.LastError, Version: r.Version, Sandbox: &sandbox, Available: &avail}
	var st IMAPState
	_ = json.Unmarshal(r.State, &st)
	v.LastSyncAt = st.LastSyncAt
	switch {
	case r.LastError != nil:
		v.Status = "ERROR"
	case r.LastOkAt != nil:
		v.Status = "CONNECTED"
	default:
		v.Status = "NOT_CONNECTED"
	}
	return v, nil
}

// TeamsPut lưu cấu hình Teams (chỉ ADMIN ở tầng route): client_secret mã hoá AES-GCM (AAD "{course}:TEAMS"), không bao giờ trả lại.
func (s *Service) TeamsPut(ctx context.Context, actor, course uuid.UUID, in TeamsPutIn) (View, error) {
	in.TenantID, in.ClientID = strings.TrimSpace(in.TenantID), strings.TrimSpace(in.ClientID)
	var fe []apierr.FieldError
	if in.TenantID == "" || len(in.TenantID) > 255 || strings.ContainsAny(in.TenantID, " /\\") {
		fe = append(fe, apierr.FieldError{Field: "tenant_id", Code: "invalid", Message: "Tenant không hợp lệ."})
	}
	if in.ClientID == "" || len(in.ClientID) > 255 {
		fe = append(fe, apierr.FieldError{Field: "client_id", Code: "invalid", Message: "Client ID không hợp lệ."})
	}
	if len(in.ClientSecret) > 2048 {
		fe = append(fe, apierr.FieldError{Field: "client_secret", Code: "invalid", Message: "Client secret quá dài."})
	}
	if len(fe) > 0 {
		return View{}, apierr.Validation(fe...)
	}
	pub, _ := json.Marshal(TeamsPublic{TenantID: in.TenantID, ClientID: in.ClientID})
	var enc []byte
	if in.ClientSecret != "" {
		sec, _ := json.Marshal(map[string]string{"client_secret": in.ClientSecret})
		var err error
		if enc, err = s.Cipher.Encrypt(sec, aad(course, KindTeams)); err != nil {
			return View{}, fmt.Errorf("mailin: mã hoá client secret: %w", err)
		}
	}
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.IntegrationGet(ctx, store.IntegrationGetParams{CourseID: course, Kind: KindTeams})
		var out store.IntegrationConfig
		before := []byte(`null`)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			if enc == nil {
				return apierr.Validation(apierr.FieldError{Field: "client_secret", Code: "required", Message: "Nhập client secret."})
			}
			if out, err = q.IntegrationInsert(ctx, store.IntegrationInsertParams{CourseID: course, Kind: KindTeams, ConfigPublic: pub, ConfigEnc: enc, Enabled: in.Enabled}); err != nil {
				return fmt.Errorf("mailin: ghi cấu hình Teams: %w", err)
			}
			if out.ID == uuid.Nil {
				return apierr.New(http.StatusConflict, apierr.VersionConflict)
			}
		case err != nil:
			return fmt.Errorf("mailin: đọc cấu hình Teams: %w", err)
		default:
			before = cur.ConfigPublic
			if in.Version != cur.Version {
				return apierr.New(http.StatusConflict, apierr.VersionConflict)
			}
			if out, err = q.IntegrationUpdate(ctx, store.IntegrationUpdateParams{CourseID: course, Kind: KindTeams, ConfigPublic: pub, ConfigEnc: enc, Enabled: in.Enabled, Version: in.Version}); errors.Is(err, pgx.ErrNoRows) {
				return apierr.New(http.StatusConflict, apierr.VersionConflict)
			} else if err != nil {
				return fmt.Errorf("mailin: ghi cấu hình Teams: %w", err)
			}
		}
		after, _ := json.Marshal(map[string]any{"config": json.RawMessage(pub), "enabled": in.Enabled, "secret_changed": enc != nil})
		_, err = q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &actor, Entity: "integration_configs", EntityID: out.ID.String(), Action: "teams.put", Before: before, After: after})
		return err
	})
	if err != nil {
		return View{}, err
	}
	return s.teamsView(ctx, course)
}

// TeamsTest xin token + `GET /education/classes` bằng cấu hình đã lưu; ghi last_ok_at / last_error (mã ngắn, không bí mật).
func (s *Service) TeamsTest(ctx context.Context, course uuid.UUID) (TestResult, error) {
	q := store.New(s.Pool)
	r, err := q.IntegrationGet(ctx, store.IntegrationGetParams{CourseID: course, Kind: KindTeams})
	if errors.Is(err, pgx.ErrNoRows) {
		return TestResult{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return TestResult{}, fmt.Errorf("mailin: đọc cấu hình Teams: %w", err)
	}
	var pub TeamsPublic
	if err := json.Unmarshal(r.ConfigPublic, &pub); err != nil {
		return TestResult{}, fmt.Errorf("mailin: cấu hình Teams hỏng: %w", err)
	}
	plain, err := s.Cipher.Decrypt(r.ConfigEnc, aad(course, KindTeams))
	if err != nil {
		return TestResult{}, fmt.Errorf("mailin: giải mã client secret: %w", err)
	}
	var sec struct {
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(plain, &sec); err != nil {
		return TestResult{}, fmt.Errorf("mailin: client secret hỏng: %w", err)
	}
	to := s.TestTimeout
	if to <= 0 {
		to = 10 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	t := &source.Teams{Cfg: s.Teams.Source}
	perr := t.Probe(cctx, source.Credentials{Tenant: pub.TenantID, ClientID: pub.ClientID, ClientSecret: sec.ClientSecret})
	if perr != nil {
		kind := source.CodeUnreachable
		var se *source.Error
		if errors.As(perr, &se) {
			kind = se.Code
		}
		if err := q.IntegrationResult(ctx, store.IntegrationResultParams{CourseID: course, Kind: KindTeams, StatePatch: []byte(`{}`), Ok: false, LastError: &kind}); err != nil {
			return TestResult{}, fmt.Errorf("mailin: ghi kết quả: %w", err)
		}
		return TestResult{Kind: &kind}, nil
	}
	if err := q.IntegrationResult(ctx, store.IntegrationResultParams{CourseID: course, Kind: KindTeams, StatePatch: []byte(`{}`), Ok: true}); err != nil {
		return TestResult{}, fmt.Errorf("mailin: ghi kết quả: %w", err)
	}
	return TestResult{OK: true}, nil
}
