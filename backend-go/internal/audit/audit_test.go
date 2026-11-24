package audit_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/audit"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/testutil"
)

type rig struct {
	t     *testing.T
	pool  *pgxpool.Pool
	svc   *audit.Service
	admin uuid.UUID
}

func newRig(t *testing.T) *rig {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	// đồng hồ giả chạy trước 1 phút: đồng hồ Postgres (Colima) có thể nhanh hơn máy chủ, dòng VIEW_LIST vừa ghi phải nằm trong khoảng «đến bây giờ» của lần gọi sau
	r := &rig{t: t, pool: pool, svc: &audit.Service{Pool: pool, Clock: clock.NewFake(time.Now().Add(time.Minute))}}
	require.NoError(t, pool.QueryRow(t.Context(), `insert into users (email, full_name, role) values ($1, 'Quản Trị', 'ADMIN') returning id`, "ad."+uuid.NewString()[:8]+"@example.test").Scan(&r.admin))
	return r
}

func (r *rig) exec(sql string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), sql, args...)
	require.NoError(r.t, err, sql)
}

// row chèn một dòng audit_log tại thời điểm `at`.
func (r *rig) row(at time.Time, actor *uuid.UUID, entity, action string, before, after any) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	enc := func(v any) any {
		if v == nil {
			return nil
		}
		b, err := json.Marshal(v)
		require.NoError(r.t, err)
		return string(b)
	}
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into audit_log (actor_id, entity, entity_id, action, before, after, created_at) values ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7) returning id`,
		actor, entity, uuid.NewString(), action, enc(before), enc(after), at).Scan(&id))
	return id
}

func (r *rig) viewCount() (n int) {
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select count(*) from audit_log where entity = 'audit_log' and action = 'VIEW_LIST'`).Scan(&n))
	return n
}

// TestAuditListNoValues — AC2: không bao giờ có giá trị before / after; before / after không phải đối tượng → changed_fields rỗng, không 500.
func TestAuditListNoValues(t *testing.T) {
	r := newRig(t)
	now := time.Now()
	r.row(now.Add(-time.Hour), &r.admin, "student_profile", "UPDATE", map[string]any{"email": "sv@canary.test", "note": "SECRET-CANARY", "mssv": "20225511"}, map[string]any{"note": "SECRET-CANARY-2", "zeta": 1})
	r.row(now.Add(-2*time.Hour), &r.admin, "student_profile", "UPDATE", nil, []any{"SECRET-CANARY"})
	r.row(now.Add(-3*time.Hour), &r.admin, "student_profile", "UPDATE", "SECRET-CANARY", 42)
	r.row(now.Add(-4*time.Hour), nil, "llm_request", "VIEW_DETAIL", nil, map[string]string{"reason": "SECRET-CANARY lý do"})
	big := map[string]any{}
	for i := range 30 {
		big[fmt.Sprintf("k%02d", i)] = i
	}
	r.row(now.Add(-5*time.Hour), nil, "big", "UPDATE", big, nil)

	page, err := r.svc.List(t.Context(), r.admin, audit.Query{}, "")
	require.NoError(t, err)
	raw, _ := json.Marshal(page)
	for _, banned := range []string{"SECRET-CANARY", "sv@canary.test", "20225511", `"before"`, `"after"`, `"payload"`} {
		require.NotContains(t, string(raw), banned)
	}
	require.Len(t, page.Items, 5)
	require.Equal(t, []string{"email", "mssv", "note", "zeta"}, page.Items[0].ChangedFields)
	require.Equal(t, []string{}, page.Items[1].ChangedFields)
	require.Equal(t, []string{}, page.Items[2].ChangedFields)
	require.Equal(t, []string{"reason"}, page.Items[3].ChangedFields)
	require.Len(t, page.Items[4].ChangedFields, 20)
	require.Nil(t, page.Items[3].ActorName, "dòng không có actor → actor_name / actor_role null")
	require.Equal(t, "Quản Trị", *page.Items[0].ActorName)
	require.Equal(t, "ADMIN", *page.Items[0].ActorRole)
}

// TestAuditViewIsAudited — AC4: mỗi lần xem ghi một VIEW_LIST với khoảng ĐÃ ÁP DỤNG; dòng vừa ghi không nằm trong chính phản hồi.
func TestAuditViewIsAudited(t *testing.T) {
	r := newRig(t)
	r.row(time.Now().Add(-time.Hour), &r.admin, "x", "UPDATE", nil, map[string]any{"a": 1})

	p1, err := r.svc.List(t.Context(), r.admin, audit.Query{Entity: "x"}, "")
	require.NoError(t, err)
	require.Len(t, p1.Items, 1)
	require.Equal(t, 1, r.viewCount())
	p2, err := r.svc.List(t.Context(), r.admin, audit.Query{}, "")
	require.NoError(t, err)
	require.Equal(t, 2, r.viewCount())
	var seen int
	for _, it := range p2.Items {
		if it.Action == "VIEW_LIST" {
			seen++
		}
	}
	require.Equal(t, 1, seen, "lần gọi sau thấy dòng của lần trước, không thấy dòng của chính nó")

	var after map[string]any
	var raw []byte
	require.NoError(t, r.pool.QueryRow(t.Context(), `select after from audit_log where action = 'VIEW_LIST' order by created_at, id limit 1`).Scan(&raw))
	require.NoError(t, json.Unmarshal(raw, &after))
	require.Equal(t, "x", after["entity"])
	require.Nil(t, after["actor_id"])
	require.Equal(t, false, after["has_cursor"])
	require.EqualValues(t, 1, after["returned"])
	from, _ := time.Parse(time.RFC3339, after["from"].(string))
	to, _ := time.Parse(time.RFC3339, after["to"].(string))
	require.Equal(t, 7*24*time.Hour, to.Sub(from), "ghi khoảng đã áp dụng dù client không gửi")
}

// TestAuditViewWriteFailClosed — AC5: không ghi được VIEW_LIST thì không trả dòng nào và không có dòng VIEW_LIST.
func TestAuditViewWriteFailClosed(t *testing.T) {
	r := newRig(t)
	r.row(time.Now().Add(-time.Hour), &r.admin, "x", "UPDATE", nil, nil)
	r.exec(`create function audit_fail() returns trigger language plpgsql as $$ begin if new.action = 'VIEW_LIST' then raise exception 'ghi hỏng'; end if; return new; end $$`)
	r.exec(`create trigger audit_fail_t before insert on audit_log for each row execute function audit_fail()`)
	page, err := r.svc.List(t.Context(), r.admin, audit.Query{}, "")
	require.Error(t, err)
	require.Empty(t, page.Items)
	var ae *apierr.Error
	require.False(t, errors.As(err, &ae), "lỗi ghi là lỗi hệ thống (500), không phải lỗi người dùng")
	require.Zero(t, r.viewCount())
}

// TestAuditListFilters — AC1, AC7: lọc AND, khoảng mặc định, tham số rỗng = thiếu, mọi tham số sai → 422 và không ghi VIEW_LIST.
func TestAuditListFilters(t *testing.T) {
	r := newRig(t)
	now := time.Now()
	other := uuid.New()
	r.exec(`insert into users (id, email, full_name, role) values ($1, 'o@example.test', 'Người Khác', 'TEACHER')`, other)
	r.row(now.Add(-1*time.Hour), &r.admin, "a", "UPDATE", nil, nil)
	r.row(now.Add(-2*time.Hour), &other, "a", "UPDATE", nil, nil)
	r.row(now.Add(-3*time.Hour), &r.admin, "b", "UPDATE", nil, nil)
	r.row(now.Add(-10*24*time.Hour), &r.admin, "a", "OLD", nil, nil) // ngoài 7 ngày mặc định

	count := func(q audit.Query) int {
		p, err := r.svc.List(t.Context(), r.admin, q, "")
		require.NoError(t, err)
		n := 0
		for _, it := range p.Items {
			if it.Action != "VIEW_LIST" {
				n++
			}
		}
		return n
	}
	require.Equal(t, 3, count(audit.Query{}), "mặc định 7 ngày")
	require.Equal(t, 2, count(audit.Query{Entity: "a"}))
	require.Equal(t, 1, count(audit.Query{Entity: "a", ActorID: strings.ToUpper(other.String())}), "uuid viết hoa = thường; lọc AND")
	require.Equal(t, 4, count(audit.Query{From: now.Add(-11 * 24 * time.Hour).Format(time.RFC3339)}), "chỉ from → to = now")
	require.Equal(t, 1, count(audit.Query{From: now.Add(-90 * time.Minute).Format(time.RFC3339), To: now.Format(time.RFC3339)}))
	require.Equal(t, 3, count(audit.Query{Entity: "", ActorID: "", From: "", To: "", Cursor: ""}), "tham số rỗng như không gửi")
	require.Equal(t, 3, count(audit.Query{To: now.Format(time.RFC3339)}), "chỉ to → from = to − 7 ngày")

	before := r.viewCount()
	for name, q := range map[string]audit.Query{
		"entity sai mẫu":    {Entity: "Bad-Entity"},
		"entity khoảng":     {Entity: "  "},
		"actor_id sai":      {ActorID: "abc"},
		"from sai":          {From: "hôm qua"},
		"to sai":            {To: "2026-13-45"},
		"from >= to":        {From: now.Format(time.RFC3339), To: now.Add(-time.Hour).Format(time.RFC3339)},
		"quá 90 ngày":       {From: now.Add(-91 * 24 * time.Hour).Format(time.RFC3339), To: now.Format(time.RFC3339)},
		"chỉ from quá 90":   {From: now.Add(-95 * 24 * time.Hour).Format(time.RFC3339)},
		"limit âm / bằng 0": {Limit: -1},
		"limit 101":         {Limit: 101},
	} {
		_, err := r.svc.List(t.Context(), r.admin, q, "")
		var ae *apierr.Error
		require.ErrorAs(t, err, &ae, name)
		require.Equal(t, http.StatusUnprocessableEntity, ae.Status, name)
		require.Equal(t, apierr.ValidationFailed, ae.Code, name)
	}
	require.Equal(t, before, r.viewCount(), "lỗi tham số không ghi VIEW_LIST")
	p, err := r.svc.List(t.Context(), r.admin, audit.Query{Entity: "zzz"}, "")
	require.NoError(t, err)
	require.Empty(t, p.Items)
	require.Nil(t, p.NextCursor)
}

// TestAuditCursor — AC3: > 100 dòng, duyệt hết không trùng không sót kể cả khi có dòng mới chen; cursor gắn với bộ lọc.
func TestAuditCursor(t *testing.T) {
	r := newRig(t)
	now := time.Now()
	r.exec(`insert into audit_log (actor_id, entity, entity_id, action, created_at)
		select $1, 'bulk', gen_random_uuid()::text, 'UPDATE', $2::timestamptz - (g % 40) * interval '1 second' from generate_series(1, 230) g`, r.admin, now.Add(-time.Hour)) // nhiều dòng cùng giây
	want := map[uuid.UUID]bool{}
	cursor := ""
	for pages := 0; ; pages++ {
		p, err := r.svc.List(t.Context(), r.admin, audit.Query{Entity: "bulk", Limit: 100, Cursor: cursor}, "")
		require.NoError(t, err)
		for _, it := range p.Items {
			require.False(t, want[it.ID], "trùng")
			want[it.ID] = true
		}
		if pages == 0 {
			r.row(now.Add(-time.Minute), &r.admin, "bulk", "UPDATE", nil, nil) // chen giữa hai lần gọi (mới hơn → không lọt vào các trang sau)
		}
		if p.NextCursor == nil {
			break
		}
		cursor = *p.NextCursor
		require.Less(t, pages, 5)
	}
	require.Len(t, want, 230)

	// khoảng thời gian cùng thời điểm viết Z hay +07:00 = cùng bộ lọc
	z := now.UTC().Format(time.RFC3339)
	p, err := r.svc.List(t.Context(), r.admin, audit.Query{Entity: "bulk", To: z, Limit: 5}, "")
	require.NoError(t, err)
	require.NotNil(t, p.NextCursor)
	loc := time.FixedZone("x", 7*3600)
	_, err = r.svc.List(t.Context(), r.admin, audit.Query{Entity: "bulk", To: now.In(loc).Format(time.RFC3339), Limit: 5, Cursor: *p.NextCursor}, "")
	require.NoError(t, err)

	before := r.viewCount()
	for name, q := range map[string]audit.Query{
		"entity khác": {Entity: "other", Limit: 5, Cursor: *p.NextCursor},
		"bỏ entity":   {Limit: 5, Cursor: *p.NextCursor},
		"actor khác":  {Entity: "bulk", ActorID: uuid.NewString(), Limit: 5, Cursor: *p.NextCursor},
		"to khác":     {Entity: "bulk", To: now.Add(-time.Minute).UTC().Format(time.RFC3339), Limit: 5, Cursor: *p.NextCursor},
		"rác":         {Entity: "bulk", Cursor: "not-a-cursor"},
	} {
		_, err := r.svc.List(t.Context(), r.admin, q, "")
		var ae *apierr.Error
		require.ErrorAs(t, err, &ae, name)
		require.Equal(t, apierr.InvalidCursor, ae.Code, name)
	}
	require.Equal(t, before, r.viewCount())
}

// TestAuditListUsesIndex — AC9: truy vấn đi qua chỉ mục mới (không Seq Scan trên audit_log) khi bảng đủ lớn.
func TestAuditListUsesIndex(t *testing.T) {
	r := newRig(t)
	r.exec(`insert into audit_log (actor_id, entity, entity_id, action, created_at)
		select $1, 'e' || (g % 50), gen_random_uuid()::text, 'UPDATE', now() - g * interval '3 seconds' from generate_series(1, 60000) g`, r.admin)
	r.exec(`analyze audit_log`)
	explain := func(where string) string {
		rows, err := r.pool.Query(t.Context(), `explain select a.id from audit_log a where a.created_at >= now() - interval '7 days' and a.created_at < now() `+where+` order by a.created_at desc, a.id desc limit 31`)
		require.NoError(t, err)
		defer rows.Close()
		var b strings.Builder
		for rows.Next() {
			var l string
			require.NoError(t, rows.Scan(&l))
			b.WriteString(l + "\n")
		}
		return b.String()
	}
	for name, w := range map[string]string{"không lọc": "", "entity": "and a.entity = 'e7'", "actor": "and a.actor_id is not null"} {
		plan := explain(w)
		require.NotContains(t, plan, "Seq Scan on audit_log", name+"\n"+plan)
		require.Contains(t, plan, "audit_log_", name+"\n"+plan)
	}
}
