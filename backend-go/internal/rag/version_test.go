package rag_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TestDocumentChangedBumpsRagVersion — US-P8-01 AC14: document.changed / document.deleted tăng ep:rag:ver:{course} (không có TTL), lớp khác không đổi.
func TestDocumentChangedBumpsRagVersion(t *testing.T) {
	t.Parallel()
	testutil.RequireContainers(t)
	rdb, err := appredis.New(t.Context(), testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	a, b := uuid.New(), uuid.New()
	h := rag.BumpVersion(rdb)
	pa, _ := json.Marshal(map[string]string{"course_id": a.String()})
	v0, err := rag.Version(t.Context(), rdb, a)
	require.NoError(t, err)
	require.Zero(t, v0)
	for _, topic := range []string{"document.changed", "document.deleted"} {
		require.NoError(t, h(t.Context(), outbox.Message{Topic: topic, Payload: pa}))
	}
	v, _ := rag.Version(t.Context(), rdb, a)
	require.EqualValues(t, 2, v)
	vb, _ := rag.Version(t.Context(), rdb, b)
	require.Zero(t, vb)
	require.NoError(t, h(t.Context(), outbox.Message{Topic: "document.changed", Payload: []byte(`{}`)}), "payload không có lớp: bỏ qua")
}
