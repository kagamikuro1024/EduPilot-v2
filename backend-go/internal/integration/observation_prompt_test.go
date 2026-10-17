package integration_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/chat"
	coursepkg "github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
)

// TestPromptNeverContainsObservations — US-P5-05 AC8: ghi chú quan sát và nhãn rủi ro của sinh viên KHÔNG bao giờ vào lời nhắc LLM của chat, kể cả khi chính sinh viên hỏi
// "Giảng viên ghi gì về em?" / "Em có bị coi là cần chú ý không?". Canary `CANARY-OBS-4F` nằm trong ghi chú thường, nhận xét AI đã lưu và lý do rủi ro.
func TestPromptNeverContainsObservations(t *testing.T) {
	t.Parallel()
	testutil.RequireContainers(t)
	ctx := t.Context()
	pool := testutil.RuntimePool(t)
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	uid := uuid.New()
	var teacher, course, stu uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, uid.String()+"@example.test").Scan(&teacher))
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"[int(uid[i])%31]
	}
	require.NoError(t, pool.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp 1', '2026-2027-HK1', $2, $3) returning id`, "OB"+uid.String()[:8], string(jc), teacher).Scan(&stu))
	course = stu
	require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'Lê Văn Nam', 'STUDENT') returning id`, "nam."+uid.String()[:8]+"@example.test").Scan(&stu))
	_, err = pool.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ROSTER', 'ZZ770001')`, course, stu)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `insert into student_observations (course_id, student_id, author_id, kind, body) values ($1, $2, $3, 'NOTE', 'CANARY-OBS-4F hay trễ hạn'), ($1, $2, $3, 'AI_SUMMARY', 'CANARY-OBS-4F nhận xét AI đã lưu')`, course, stu, teacher)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `insert into student_risk (course_id, student_id, level, reasons, entered_at, computed_at) values ($1, $2, 'AT_RISK', '[{"code":"ATTENDANCE_LOW","values":{"absent":3,"recorded":4,"weeks":2}}]', now(), now())`, course, stu)
	require.NoError(t, err)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := llm.NewRegistry(nil, llm.EnvConfig{Provider: "fake", Fake: fake.Settings{StreamDelay: 1}}, nil, log)
	require.NoError(t, reg.Load(ctx))
	rc := &appredis.Client{Client: rdb.Client}
	det := &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: pool}, Redis: rc, Log: log}}
	g := llm.New(llm.Options{Registry: reg, Log: log, Masker: &privacy.Masker{Detector: det, Redis: rc, Log: log, Timeout: time.Second}})
	ag := &agent.Agent{An: &agent.Analyzer{Classifier: &privacy.Classifier{Detector: det}, Detector: det, Self: agent.StoreSelf{Pool: pool}, Embed: func(context.Context, string) ([]float32, error) { return constVec(), nil }, Log: log},
		Private: agent.DefaultPrivateRegistry(nil, nil, nil, nil, nil), Rag: &rag.Service{DB: pool}, Gen: g, Events: agent.StoreEvents{Pool: pool}, Log: log}
	svc := &chat.Service{Pool: pool, Redis: rc, Agent: ag, PII: det, Lock: noLock{}, Members: coursepkg.Resolver{Pool: pool}, Log: log, Cfg: chat.Config{RatePerMin: 1000}}
	var doc uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `insert into documents (course_id, title, status) values ($1, 'Quy chế học vụ', 'READY') returning id`, course).Scan(&doc))
	_, err = pool.Exec(ctx, `insert into content_chunks (document_id, course_ids, audience, ord, text, embedding) values ($1, array[$2::uuid], 'ALL', 0, 'Điều 5. Sinh viên được thi lại một lần.', array_fill(0.25::real, array[1536])::vector)`, doc, course)
	require.NoError(t, err)

	reg.Fake().ResetPayloads()
	a := chat.Actor{UserID: stu, Role: auth.RoleStudent}
	sess, err := svc.CreateSession(ctx, a, chat.CreateSessionIn{CourseID: course})
	require.NoError(t, err)
	var wire []string
	for _, text := range []string{"Giảng viên ghi gì về em?", "Em có bị coi là cần chú ý không?", "Quy chế thi lại thế nào ạ?"} {
		m, err := svc.Send(ctx, a, sess.ID, uuid.New(), text)
		require.NoError(t, err)
		sk := &sink{}
		require.NoError(t, svc.Tail(ctx, sk, m, ""))
		svc.Wait()
		wire = append(wire, sk.evs...)
	}
	payloads := reg.Fake().Payloads()
	require.NotEmpty(t, payloads, "chat phải gọi LLM / nhúng để quét có nghĩa")
	for _, p := range payloads {
		for _, part := range append(append([]string(nil), p.Messages...), p.Inputs...) {
			for _, bad := range []string{"CANARY-OBS-4F", "AT_RISK", "ATTENDANCE_LOW", "trễ hạn"} {
				require.NotContains(t, part, bad, "ghi chú / nhãn rủi ro lọt vào lời nhắc %s", p.Task)
			}
		}
	}
	for _, e := range wire {
		require.NotContains(t, e, "CANARY-OBS-4F")
		require.False(t, strings.Contains(e, "AT_RISK"), e)
	}
}
