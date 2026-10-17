package escalation_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/escalation"
)

func TestFailedMailAdminOnly(t *testing.T) {
	r := newRig(t, escalation.Config{})
	h := r.api(t)
	for who, want := range map[string]int{"admin": 200, "gv": 403, "ta": 403, "sv": 403} {
		code, _ := call(h, r.whoAll()[who], "GET", "/admin/observability/failed-mail", "")
		require.Equal(t, want, code, who)
	}
	code, _ := call(h, who{}, "GET", "/admin/observability/failed-mail", "")
	require.Equal(t, 401, code)
}

func TestFailedMailMaskedRecipient(t *testing.T) {
	r := newRig(t, escalation.Config{})
	h := r.api(t)
	r.exec(`insert into mail_outbox (to_addr, template, payload, status, attempts, last_error) values ('nguyen.van.a@example.test', 'escalation_answered', '{"ticket_id":"CANARY-PAYLOAD"}', 'DEAD', 4, 'smtp_temporary')`)
	r.exec(`insert into mail_outbox (to_addr, template, payload, status, sent_at) values ('ok@example.test', 'ticket_new', '{}', 'SENT', now())`)
	code, body := call(h, r.whoAll()["admin"], "GET", "/admin/observability/failed-mail", "")
	require.Equal(t, 200, code)
	var out struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &out))
	require.Len(t, out.Items, 1, "chỉ thư DEAD")
	require.Equal(t, "n***@example.test", out.Items[0]["to_masked"])
	require.Equal(t, "smtp_temporary", out.Items[0]["last_error"])
	require.EqualValues(t, 4, out.Items[0]["attempts"])
	require.NotContains(t, body, "CANARY-PAYLOAD", "không payload")
	require.NotContains(t, body, "nguyen.van.a", "địa chỉ đã che")
	for _, k := range []string{"payload", "body", "text", "to_addr"} {
		require.NotContains(t, out.Items[0], k)
	}
}

func TestFailedMailUsesDeadIndex(t *testing.T) {
	r := newRig(t, escalation.Config{})
	r.exec(`insert into mail_outbox (to_addr, template, status, sent_at, updated_at, created_at)
		select 'u' || g || '@example.test', 'ticket_new', case when g % 200 = 0 then 'DEAD'::mail_status else 'SENT'::mail_status end, case when g % 200 = 0 then null else now() end, now() - (g || ' seconds')::interval, now() - (g || ' seconds')::interval
		from generate_series(1, 100000) g`)
	r.exec(`analyze mail_outbox`)
	rows, err := r.pool.Query(t.Context(), `explain select id, template, to_addr, attempts, last_error, updated_at from mail_outbox where status = 'DEAD' order by updated_at desc, id desc limit 31`)
	require.NoError(t, err)
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var l string
		require.NoError(t, rows.Scan(&l))
		plan.WriteString(l + "\n")
	}
	require.NotContains(t, plan.String(), "Seq Scan on mail_outbox", plan.String())
	require.Contains(t, plan.String(), "mail_outbox_dead_idx")
}
