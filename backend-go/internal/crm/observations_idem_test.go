package crm_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestObservationIdempotencyKey (#40, luật 14): cùng Idempotency-Key + cùng thân → một ghi chú, phản hồi phát lại y hệt; cùng khoá + thân khác → 422, vẫn một dòng;
// khoá khác → dòng mới; không khoá → chạy như cũ (client cũ, seed).
func TestObservationIdempotencyKey(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	s := w.students[0].ID
	post := func(body, key string) resp {
		h := map[string]string{}
		if key != "" {
			h["Idempotency-Key"] = key
		}
		return r.do(req{method: http.MethodPost, path: w.sp(0, "/observations"), bearer: w.T.access, body: map[string]any{"body": body, "tags": []string{}}, hdr: h})
	}
	rows := func() int { return r.count(`select count(*) from student_observations where student_id = $1`, s) }

	key := uuid.NewString()
	first := post("Hay hỏi bài.", key)
	require.Equal(t, http.StatusCreated, first.code, string(first.body))
	again := post("Hay hỏi bài.", key)
	require.Equal(t, http.StatusCreated, again.code, string(again.body))
	require.Equal(t, first.json()["id"], again.json()["id"], "phát lại đúng phản hồi cũ")
	require.Equal(t, 1, rows(), "gửi lại cùng khoá không ghi thêm")

	require.Equal(t, http.StatusUnprocessableEntity, post("Nội dung khác.", key).code, "cùng khoá, thân khác")
	require.Equal(t, 1, rows())

	require.Equal(t, http.StatusCreated, post("Hay hỏi bài.", uuid.NewString()).code, "khoá khác là ghi chú mới")
	require.Equal(t, http.StatusCreated, post("Không khoá.", "").code, "không khoá vẫn chạy")
	require.Equal(t, 3, rows())
}
