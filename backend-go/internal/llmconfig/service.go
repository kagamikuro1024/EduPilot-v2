package llmconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/trace"
)

// Hạn mức (SRS 5.4).
const (
	MaxProviders         = 20
	MaxModelsPerProvider = 100
	maxChain             = 4
	embedDims            = 1536
)

// ProviderTypes là các loại nhà cung cấp hợp lệ (SRS 5.4).
func ProviderTypes() []string {
	return []string{"openai", "anthropic", "gemini", "openai_compatible", "fake"}
}

// Tasks là 7 tác vụ LLM (SRS 5.4), theo thứ tự hiển thị.
func Tasks() []string {
	return []string{"CHAT", "CLASSIFY", "UTILITY", "GRADING", "QUESTION_GEN", "INSIGHT", "EMBEDDING"}
}

// KeyStatus: tình trạng khoá của một nhà cung cấp (không bao giờ là bản thân khoá).
type KeyStatus string

// Giá trị của KeyStatus.
const (
	KeyOK         KeyStatus = "ok"
	KeyMissing    KeyStatus = "missing"
	KeyUnreadable KeyStatus = "unreadable"
)

// TestResult là kết quả Test gần nhất đã lưu.
type TestResult struct {
	OK        bool
	At        time.Time
	ErrorKind string
}

// Model là một mô hình của nhà cung cấp; giá = đ / 1 triệu token.
type Model struct {
	ID         uuid.UUID
	ProviderID uuid.UUID
	Model      string
	Kind       string
	Dims       *int
	PriceIn    decimal.Decimal
	PriceOut   decimal.Decimal
	Enabled    bool
}

// Provider là nhà cung cấp đã lưu. KHÔNG có trường khoá hay đuôi khoá.
type Provider struct {
	ID        uuid.UUID
	Type      string
	Name      string
	BaseURL   *string
	Enabled   bool
	HasKey    bool
	KeyStatus KeyStatus
	RPMLimit  *int
	TPMLimit  *int
	LastTest  *TestResult
	Version   int
	Models    []Model
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ModelInput là một mô hình trong yêu cầu tạo / sửa nhà cung cấp.
type ModelInput struct {
	Model    string
	Kind     string
	Dims     *int
	PriceIn  decimal.Decimal
	PriceOut decimal.Decimal
	Enabled  *bool
}

// ProviderInput là yêu cầu tạo / sửa. Sửa: APIKey nil = giữ khoá cũ; Models nil = giữ danh sách, khác nil = thay toàn bộ;
// Version bắt buộc khớp.
type ProviderInput struct {
	Type     string
	Name     string
	BaseURL  *string
	APIKey   *Secret
	Enabled  *bool
	RPMLimit *int
	TPMLimit *int
	Models   []ModelInput
	Version  int
}

// Service là dịch vụ cấu hình. Mọi hàm nhận danh tính từ ctx đã xác thực (auth.Middleware), không từ tham số.
type Service struct {
	pool     *pgxpool.Pool
	q        *store.Queries
	cipher   *crypto.Cipher
	clock    clock.Clock
	onChange func(context.Context)
}

// Option tuỳ chỉnh Service.
type Option func(*Service)

// WithClock thay đồng hồ (test).
func WithClock(c clock.Clock) Option { return func(s *Service) { s.clock = c } }

// WithOnChange đăng ký hàm gọi SAU mỗi thay đổi cấu hình đã commit (US-P1-02: PUBLISH ep:llm:reload).
func WithOnChange(f func(context.Context)) Option { return func(s *Service) { s.onChange = f } }

// New dựng Service.
func New(pool *pgxpool.Pool, c *crypto.Cipher, opts ...Option) *Service {
	s := &Service{pool: pool, q: store.New(pool), cipher: c, clock: clock.Real{}}
	for _, o := range opts {
		o(s)
	}
	return s
}

// actor là danh tính đã xác thực lấy từ ctx.
type actor struct {
	id   uuid.UUID
	role auth.Role
}

func actorFrom(ctx context.Context, write bool) (actor, error) {
	p, ok := auth.FromContext(ctx)
	if !ok {
		return actor{}, ErrForbidden
	}
	switch {
	case p.Role == auth.RoleAdmin:
	case !write && p.Role == auth.RoleTeacher:
	default:
		return actor{}, ErrForbidden
	}
	id, _ := uuid.Parse(p.Sub) // sub không phải uuid (token thử) → actor_id null trong audit_log
	return actor{id: id, role: p.Role}, nil
}

func aadFor(id uuid.UUID) []byte { return []byte("llm_providers:" + id.String()) }

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("llmconfig: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("llmconfig: commit: %w", err)
	}
	return nil
}

func (s *Service) changed(ctx context.Context) {
	if s.onChange != nil {
		s.onChange(ctx)
	}
}

// writeAudit ghi MỘT dòng audit_log trong cùng giao dịch; before/after không bao giờ chứa khoá.
func (s *Service) writeAudit(ctx context.Context, q *store.Queries, a actor, entity, entityID, action string, before, after any) error {
	arg := store.InsertAuditLogParams{Entity: entity, EntityID: entityID, Action: action}
	if a.id != uuid.Nil {
		arg.ActorID = &a.id
	}
	if before != nil {
		b, _ := json.Marshal(before)
		arg.Before = b
	}
	if after != nil {
		b, _ := json.Marshal(after)
		arg.After = b
	}
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		t := sc.TraceID().String()
		arg.TraceID = &t
	}
	if _, err := q.InsertAuditLog(ctx, arg); err != nil {
		return fmt.Errorf("llmconfig: ghi audit_log: %w", err)
	}
	return nil
}

// ---- đọc ----

// ListProviders trả mọi nhà cung cấp kèm mô hình (ADMIN, TEACHER).
func (s *Service) ListProviders(ctx context.Context) ([]Provider, error) {
	if _, err := actorFrom(ctx, false); err != nil {
		return nil, err
	}
	rows, err := s.q.ListLLMProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("llmconfig: liệt kê nhà cung cấp: %w", err)
	}
	models, err := s.q.ListLLMModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("llmconfig: liệt kê mô hình: %w", err)
	}
	byProvider := map[uuid.UUID][]Model{}
	for _, m := range models {
		byProvider[m.ProviderID] = append(byProvider[m.ProviderID], toModel(m))
	}
	out := make([]Provider, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toProvider(r, byProvider[r.ID]))
	}
	return out, nil
}

// GetProvider trả một nhà cung cấp (ADMIN, TEACHER).
func (s *Service) GetProvider(ctx context.Context, id uuid.UUID) (Provider, error) {
	if _, err := actorFrom(ctx, false); err != nil {
		return Provider{}, err
	}
	return s.getProvider(ctx, s.q, id)
}

func (s *Service) getProvider(ctx context.Context, q *store.Queries, id uuid.UUID) (Provider, error) {
	r, err := q.GetLLMProvider(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Provider{}, ErrNotFound
	}
	if err != nil {
		return Provider{}, fmt.Errorf("llmconfig: đọc nhà cung cấp: %w", err)
	}
	ms, err := q.ListLLMModelsByProvider(ctx, id)
	if err != nil {
		return Provider{}, fmt.Errorf("llmconfig: đọc mô hình: %w", err)
	}
	models := make([]Model, 0, len(ms))
	for _, m := range ms {
		models = append(models, toModel(m))
	}
	return s.toProvider(r, models), nil
}

func (s *Service) keyStatus(r store.LlmProvider) KeyStatus {
	if len(r.ApiKeyEnc) == 0 {
		return KeyMissing
	}
	if _, err := s.cipher.Decrypt(r.ApiKeyEnc, aadFor(r.ID)); err != nil {
		return KeyUnreadable
	}
	return KeyOK
}

func (s *Service) toProvider(r store.LlmProvider, models []Model) Provider {
	p := Provider{
		ID: r.ID, Type: r.Type, Name: r.Name, BaseURL: r.BaseUrl, Enabled: r.Enabled,
		HasKey: len(r.ApiKeyEnc) > 0, KeyStatus: s.keyStatus(r),
		RPMLimit: intPtr(r.RpmLimit), TPMLimit: intPtr(r.TpmLimit),
		Version: int(r.Version), Models: models, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if p.Models == nil {
		p.Models = []Model{}
	}
	if r.LastTestOk != nil && r.LastTestAt != nil {
		t := &TestResult{OK: *r.LastTestOk, At: *r.LastTestAt}
		if r.LastTestError != nil {
			t.ErrorKind = *r.LastTestError
		}
		p.LastTest = t
	}
	return p
}

func toModel(m store.LlmModel) Model {
	return Model{ID: m.ID, ProviderID: m.ProviderID, Model: m.Model, Kind: m.Kind, Dims: intPtr(m.Dims),
		PriceIn: m.PriceIn, PriceOut: m.PriceOut, Enabled: m.Enabled}
}

func intPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}

func int32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	n := int32(*v) //nolint:gosec // đã kiểm khoảng ở validate
	return &n
}

// ---- ghi: nhà cung cấp ----

// auditProvider là ảnh chụp KHÔNG chứa khoá của một nhà cung cấp cho audit_log.
type auditProvider struct {
	Type       string   `json:"type"`
	Name       string   `json:"name"`
	BaseURL    *string  `json:"base_url,omitempty"`
	Enabled    bool     `json:"enabled"`
	HasKey     bool     `json:"has_key"`
	KeyChanged bool     `json:"key_changed,omitempty"`
	RPMLimit   *int     `json:"rpm_limit,omitempty"`
	TPMLimit   *int     `json:"tpm_limit,omitempty"`
	Models     []string `json:"models"`
}

func snapshot(p Provider, keyChanged bool) auditProvider {
	names := make([]string, 0, len(p.Models))
	for _, m := range p.Models {
		names = append(names, m.Model)
	}
	slices.Sort(names)
	return auditProvider{Type: p.Type, Name: p.Name, BaseURL: p.BaseURL, Enabled: p.Enabled, HasKey: p.HasKey, KeyChanged: keyChanged,
		RPMLimit: p.RPMLimit, TPMLimit: p.TPMLimit, Models: names}
}

func invalid(field, code, msg string) error {
	return &ErrInvalid{Field: field, Code: code, Message: msg}
}

func validateProvider(in ProviderInput) error {
	if !slices.Contains(ProviderTypes(), in.Type) {
		return invalid("type", "INVALID_TYPE", "loại nhà cung cấp không hợp lệ")
	}
	if n := len([]rune(strings.TrimSpace(in.Name))); n < 1 || n > 60 {
		return invalid("name", "INVALID_NAME", "tên dài 1–60 ký tự")
	}
	if in.BaseURL != nil {
		u, err := url.Parse(strings.TrimSpace(*in.BaseURL))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return invalid("base_url", "INVALID_URL", "địa chỉ phải là URL http(s)")
		}
	}
	if in.Type == "openai_compatible" && (in.BaseURL == nil || strings.TrimSpace(*in.BaseURL) == "") {
		return invalid("base_url", "REQUIRED", "nhà cung cấp tương thích OpenAI cần địa chỉ gốc")
	}
	if in.RPMLimit != nil && *in.RPMLimit <= 0 {
		return invalid("rpm_limit", "OUT_OF_RANGE", "giới hạn lượt/phút phải > 0")
	}
	if in.TPMLimit != nil && *in.TPMLimit <= 0 {
		return invalid("tpm_limit", "OUT_OF_RANGE", "giới hạn token/phút phải > 0")
	}
	if in.APIKey != nil && strings.TrimSpace(in.APIKey.v) == "" {
		// Không có cách "xoá khoá" ngầm: chuỗi rỗng là lỗi (SRS 6.3).
		return invalid("api_key", "EMPTY", "khoá API không được rỗng; bỏ trường này để giữ khoá cũ")
	}
	if len(in.Models) > MaxModelsPerProvider {
		return fmt.Errorf("%w: tối đa %d mô hình mỗi nhà cung cấp", ErrLimit, MaxModelsPerProvider)
	}
	seen := map[string]bool{}
	for i, m := range in.Models {
		f := fmt.Sprintf("models[%d]", i)
		if n := len([]rune(m.Model)); n < 1 || n > 120 {
			return invalid(f+".model", "INVALID_NAME", "tên mô hình dài 1–120 ký tự")
		}
		if seen[m.Model] {
			return invalid(f+".model", "DUPLICATE", "trùng tên mô hình trong cùng nhà cung cấp")
		}
		seen[m.Model] = true
		if m.Kind != "chat" && m.Kind != "embedding" {
			return invalid(f+".kind", "INVALID_KIND", "loại mô hình là chat hoặc embedding")
		}
		if m.Kind == "embedding" && (m.Dims == nil || *m.Dims <= 0) {
			return invalid(f+".dims", "REQUIRED", "mô hình nhúng cần số chiều")
		}
		if m.Dims != nil && *m.Dims <= 0 {
			return invalid(f+".dims", "OUT_OF_RANGE", "số chiều phải > 0")
		}
		if m.PriceIn.IsNegative() || m.PriceOut.IsNegative() {
			return invalid(f+".price", "OUT_OF_RANGE", "giá không âm")
		}
	}
	return nil
}

func isUnique(err error, constraint string) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505" && (constraint == "" || pg.ConstraintName == constraint)
}

// CreateProvider tạo nhà cung cấp (ADMIN). Khoá được mã hoá với AAD = id sinh ở Go trước khi chèn.
func (s *Service) CreateProvider(ctx context.Context, in ProviderInput) (Provider, error) {
	a, err := actorFrom(ctx, true)
	if err != nil {
		return Provider{}, err
	}
	if err := validateProvider(in); err != nil {
		return Provider{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Provider{}, fmt.Errorf("llmconfig: sinh id: %w", err)
	}
	var out Provider
	err = s.inTx(ctx, func(q *store.Queries) error {
		n, err := q.CountLLMProviders(ctx)
		if err != nil {
			return fmt.Errorf("llmconfig: đếm nhà cung cấp: %w", err)
		}
		if n >= MaxProviders {
			return fmt.Errorf("%w: tối đa %d nhà cung cấp", ErrLimit, MaxProviders)
		}
		arg := store.InsertLLMProviderParams{
			ID: id, Type: in.Type, Name: strings.TrimSpace(in.Name), BaseUrl: trimPtr(in.BaseURL),
			Enabled: in.Enabled == nil || *in.Enabled, RpmLimit: int32Ptr(in.RPMLimit), TpmLimit: int32Ptr(in.TPMLimit),
		}
		if in.APIKey != nil {
			if arg.ApiKeyEnc, err = s.cipher.Encrypt([]byte(in.APIKey.v), aadFor(id)); err != nil {
				return fmt.Errorf("llmconfig: mã hoá khoá: %w", err)
			}
		}
		if _, err := q.InsertLLMProvider(ctx, arg); err != nil {
			if isUnique(err, "llm_providers_name_uq") {
				return ErrDuplicateName
			}
			return fmt.Errorf("llmconfig: chèn nhà cung cấp: %w", err)
		}
		if err := upsertModels(ctx, q, id, in.Models); err != nil {
			return err
		}
		if out, err = s.getProvider(ctx, q, id); err != nil {
			return err
		}
		return s.writeAudit(ctx, q, a, "llm_provider", id.String(), "create", nil, snapshot(out, in.APIKey != nil))
	})
	if err != nil {
		return Provider{}, err
	}
	s.changed(ctx)
	return out, nil
}

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	t := strings.TrimSpace(*p)
	if t == "" {
		return nil
	}
	return &t
}

func upsertModels(ctx context.Context, q *store.Queries, providerID uuid.UUID, ms []ModelInput) error {
	for _, m := range ms {
		arg := store.UpsertLLMModelParams{ProviderID: providerID, Model: m.Model, Kind: m.Kind, PriceIn: m.PriceIn, PriceOut: m.PriceOut,
			Enabled: m.Enabled == nil || *m.Enabled}
		arg.Dims = int32Ptr(m.Dims)
		if _, err := q.UpsertLLMModel(ctx, arg); err != nil {
			return fmt.Errorf("llmconfig: lưu mô hình: %w", err)
		}
	}
	return nil
}

// UpdateProvider sửa nhà cung cấp (ADMIN) với khoá lạc quan in.Version.
func (s *Service) UpdateProvider(ctx context.Context, id uuid.UUID, in ProviderInput) (Provider, error) {
	a, err := actorFrom(ctx, true)
	if err != nil {
		return Provider{}, err
	}
	if err := validateProvider(in); err != nil {
		return Provider{}, err
	}
	var out Provider
	err = s.inTx(ctx, func(q *store.Queries) error {
		before, err := s.getProvider(ctx, q, id)
		if err != nil {
			return err
		}
		if before.Version != in.Version {
			return &ErrVersionConflict{Current: before.Version}
		}
		enabled := before.Enabled
		if in.Enabled != nil {
			enabled = *in.Enabled
		}
		arg := store.UpdateLLMProviderParams{
			ID: id, Version: int32(in.Version), //nolint:gosec // version nhỏ
			Type: in.Type, Name: strings.TrimSpace(in.Name), BaseUrl: trimPtr(in.BaseURL), Enabled: enabled,
			RpmLimit: int32Ptr(in.RPMLimit), TpmLimit: int32Ptr(in.TPMLimit),
			// kết quả Test cũ hết giá trị khi đổi khoá / địa chỉ / loại
			ResetTest: in.APIKey != nil || in.Type != before.Type || !samePtr(trimPtr(in.BaseURL), before.BaseURL),
		}
		if in.APIKey != nil {
			if arg.ApiKeyEnc, err = s.cipher.Encrypt([]byte(in.APIKey.v), aadFor(id)); err != nil {
				return fmt.Errorf("llmconfig: mã hoá khoá: %w", err)
			}
		}
		if _, err := q.UpdateLLMProvider(ctx, arg); err != nil {
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				cur, gerr := q.GetLLMProvider(ctx, id)
				if gerr != nil {
					return ErrNotFound
				}
				return &ErrVersionConflict{Current: int(cur.Version)}
			case isUnique(err, "llm_providers_name_uq"):
				return ErrDuplicateName
			}
			return fmt.Errorf("llmconfig: sửa nhà cung cấp: %w", err)
		}
		if in.Models != nil {
			if err := replaceModels(ctx, q, id, before.Models, in.Models); err != nil {
				return err
			}
		}
		if out, err = s.getProvider(ctx, q, id); err != nil {
			return err
		}
		return s.writeAudit(ctx, q, a, "llm_provider", id.String(), "update", snapshot(before, false), snapshot(out, in.APIKey != nil))
	})
	if err != nil {
		return Provider{}, err
	}
	s.changed(ctx)
	return out, nil
}

func samePtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// replaceModels thay toàn bộ danh sách mô hình; mô hình bị bỏ mà đang được tuyến dùng → ErrProviderInUse.
func replaceModels(ctx context.Context, q *store.Queries, providerID uuid.UUID, existing []Model, want []ModelInput) error {
	keep := make(map[string]bool, len(want))
	for _, m := range want {
		keep[m.Model] = true
	}
	var drop []uuid.UUID
	for _, m := range existing {
		if !keep[m.Model] {
			drop = append(drop, m.ID)
		}
	}
	if len(drop) > 0 {
		tasks, err := q.LLMTasksUsingModels(ctx, drop)
		if err != nil {
			return fmt.Errorf("llmconfig: kiểm tra mô hình đang dùng: %w", err)
		}
		if len(tasks) > 0 {
			return &ErrProviderInUse{Tasks: tasks}
		}
	}
	names := make([]string, 0, len(want))
	for _, m := range want {
		names = append(names, m.Model)
	}
	if _, err := q.DeleteLLMModelsNotIn(ctx, store.DeleteLLMModelsNotInParams{ProviderID: providerID, Keep: names}); err != nil {
		return fmt.Errorf("llmconfig: xoá mô hình thừa: %w", err)
	}
	return upsertModels(ctx, q, providerID, want)
}

// DeleteProvider xoá nhà cung cấp cùng mô hình của nó; đang được tuyến dùng → ErrProviderInUse.
func (s *Service) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	a, err := actorFrom(ctx, true)
	if err != nil {
		return err
	}
	err = s.inTx(ctx, func(q *store.Queries) error {
		before, err := s.getProvider(ctx, q, id)
		if err != nil {
			return err
		}
		tasks, err := q.LLMTasksUsingProvider(ctx, id)
		if err != nil {
			return fmt.Errorf("llmconfig: kiểm tra nhà cung cấp đang dùng: %w", err)
		}
		if len(tasks) > 0 {
			return &ErrProviderInUse{Tasks: tasks}
		}
		if _, err := q.DeleteLLMProvider(ctx, id); err != nil {
			return fmt.Errorf("llmconfig: xoá nhà cung cấp: %w", err)
		}
		return s.writeAudit(ctx, q, a, "llm_provider", id.String(), "delete", snapshot(before, false), nil)
	})
	if err != nil {
		return err
	}
	s.changed(ctx)
	return nil
}

// RecordTest lưu kết quả Test kết nối của một nhà cung cấp đã lưu (ADMIN). errorKind chỉ là loại lỗi, không thân lỗi.
func (s *Service) RecordTest(ctx context.Context, id uuid.UUID, ok bool, errorKind string) error {
	if _, err := actorFrom(ctx, true); err != nil {
		return err
	}
	now := s.clock.Now()
	arg := store.UpdateLLMProviderTestParams{ID: id, LastTestOk: &ok, LastTestAt: &now}
	if errorKind != "" {
		arg.LastTestError = &errorKind
	}
	if _, err := s.q.UpdateLLMProviderTest(ctx, arg); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("llmconfig: lưu kết quả test: %w", err)
	}
	return nil
}
