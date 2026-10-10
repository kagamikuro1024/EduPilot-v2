package integration_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/chat"
	coursepkg "github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
)

type student struct {
	name, code, email, phone, cccd string
}

func (s student) secrets() []string {
	return []string{s.name, strings.ToLower(foldVN(s.name)), strings.ToUpper(s.name), s.code, strings.ToLower(s.code), s.email, s.phone, s.cccd}
}

// foldVN bỏ dấu cho tên trong bảng thử (chỉ các chữ dùng ở dưới).
func foldVN(s string) string {
	r := strings.NewReplacer("ễ", "e", "ề", "e", "ệ", "e", "ê", "e", "ă", "a", "ạ", "a", "ầ", "a", "ấ", "a", "â", "a", "ư", "u", "ữ", "u", "ơ", "o", "ộ", "o", "ố", "o", "ô", "o", "ị", "i", "í", "i", "ì", "i", "Ễ", "E", "Ế", "E", "Ê", "E", "Ă", "A", "Â", "A", "Ư", "U", "Ơ", "O", "Ô", "O", "Đ", "D", "đ", "d", "ú", "u", "ù", "u", "ý", "y", "ó", "o", "ò", "o", "ổ", "o", "ẫ", "a", "ã", "a", "à", "a", "á", "a", "ả", "a", "é", "e", "è", "e", "ế", "e", "ể", "e", "ẻ", "e")
	return r.Replace(s)
}

func rosterOf(n int) []student {
	ho := []string{"Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Phan", "Vũ", "Đặng", "Bùi", "Đỗ"}
	dem := []string{"Văn", "Thị", "Minh", "Quốc", "Hữu"}
	ten := []string{"An", "Bình", "Cường", "Dũng", "Hà", "Khải", "Linh", "Nam", "Oanh", "Phúc"}
	out := make([]student, n)
	for i := range n {
		name := fmt.Sprintf("%s %s %s", ho[i%len(ho)], dem[(i/2)%len(dem)], ten[(i*3+i/10)%len(ten)])
		out[i] = student{name: name, code: fmt.Sprintf("ZZ77%06d", i+1), email: fmt.Sprintf("sv%02d.hoc@sv.edu.vn", i), phone: fmt.Sprintf("09123456%02d", i), cccd: fmt.Sprintf("0792030%05d", i)}
	}
	return out
}

// TestNoPayloadLeak — US-P3-03 AC6 (cổng của G1) ở tầng cổng LLM: roster 30 sinh viên; kịch bản chat, Threads (có cấu trúc), nhúng, kết quả tool, đường suy giảm;
// provider giả ghi MỌI payload → 0 họ tên (3 biến thể) / MSSV / email / SĐT / CCCD của bất kỳ sinh viên nào, payload có placeholder.
// Từ US-P3-05 có thêm kịch bản qua internal/chat thật (agent + cổng llm + che thật) vào CÙNG bảng secrets; Threads thêm ở US-P3-06.
func TestNoPayloadLeak(t *testing.T) {
	t.Parallel()
	testutil.RequireContainers(t)
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	uid := uuid.New()
	var teacher, course uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, uid.String()+"@example.test").Scan(&teacher))
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(uid[i])%len(alpha)]
	}
	require.NoError(t, pool.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp 1', '2026-2027-HK1', $2, $3) returning id`,
		"NP"+uid.String()[:8], string(jc), teacher).Scan(&course))
	students := rosterOf(30)
	uids := make([]uuid.UUID, len(students))
	for i, s := range students {
		var u uuid.UUID
		require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, $2, 'STUDENT') returning id`, s.email, s.name).Scan(&u))
		uids[i] = u
		_, err := pool.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, 'STUDENT', 'ACTIVE', 'ROSTER', $3)`, course, u, s.code)
		require.NoError(t, err, i)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := llm.NewRegistry(nil, llm.EnvConfig{Provider: "fake", Fake: fake.Settings{StreamDelay: 1}}, nil, log)
	require.NoError(t, reg.Load(ctx))
	rc := &appredis.Client{Client: rdb.Client}
	masker := &privacy.Masker{Detector: &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: pool}, Redis: rc, Log: log}}, Redis: rc, Log: log}
	g := llm.New(llm.Options{Registry: reg, Log: log, Masker: masker})

	ictx := llm.WithIdentity(ctx, llm.Identity{CourseID: &course})
	structured := []byte(`{"type":"object","required":["answer"],"properties":{"answer":{"type":"string"}}}`)
	for i, me := range students {
		other := students[(i+7)%len(students)]
		sess := privacy.WithSession(ictx, privacy.NewSession(uuid.NewString()))
		// chat riêng: tên, MSSV, email, SĐT, CCCD của mình và của người khác; ba biến thể tên
		priv := fmt.Sprintf("Em là %s, mssv %s, mail %s, sđt %s, cccd %s. Hỏi giúp %s (%s) và %s: điểm cộng thế nào? mã %s",
			me.name, me.code, me.email, me.phone, me.cccd, strings.ToLower(foldVN(other.name)), other.code, strings.ToUpper(other.name), strings.ToLower(other.code))
		hist := []llm.Message{{Role: "system", Content: "Gọi người dùng là bạn. Hồ sơ: " + me.name}, {Role: "user", Content: priv},
			{Role: "assistant", Content: "Chào " + me.name + ", đã ghi nhận."}, {Role: "user", Content: "Kết quả tool get_my_grades cho " + other.name + ": 8,5"}}
		_, err := g.Chat(sess, llm.Request{Task: llm.TaskChat, Messages: hist})
		require.NoError(t, err)
		ch, err := g.Stream(sess, llm.Request{Task: llm.TaskChat, Messages: hist})
		require.NoError(t, err)
		for c := range ch {
			require.NoError(t, c.Err)
			require.NotContains(t, c.Text, "[[")
		}
		// Threads: AI trả lời có cấu trúc, phạm vi yêu cầu (không phiên)
		_, err = g.Structured(ictx, llm.Request{Task: llm.TaskUtility, Messages: []llm.Message{{Role: "user", Content: "Bài đăng của " + other.name + " (" + other.email + "): xin hỏi về Điều 5"}}}, structured)
		require.NoError(t, err)
		// nhúng câu hỏi
		_, err = g.Embed(ictx, llm.EmbedRequest{Inputs: []string{"điểm của " + me.name, "mail " + me.email + " mssv " + me.code}})
		require.NoError(t, err)
	}
	// đường suy giảm: mọi provider lỗi → câu trích nguyên văn; payload vẫn đã che
	reg.Fake().Set(fake.Settings{ErrorRate: 1, ErrorKind: "SERVER", StreamDelay: 1})
	_, err = g.Chat(privacy.WithSession(ictx, privacy.NewSession(uuid.NewString())), llm.Request{Task: llm.TaskChat, Messages: []llm.Message{{Role: "user", Content: "Em " + students[3].name + " " + students[3].code}},
		Passages: []llm.Passage{{Text: "Điều 5", Source: "Quy chế", Page: 1, Score: 0.9}}})
	require.NoError(t, err)

	// qua internal/chat THẬT (agent thật + cổng llm thật + che thật): mỗi sinh viên một phiên, tin có tên / MSSV / email / SĐT của mình và của người khác
	det := &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: pool}, Redis: rc, Log: log}}
	ag := &agent.Agent{An: &agent.Analyzer{Classifier: &privacy.Classifier{Detector: det}, Detector: det, Self: agent.StoreSelf{Pool: pool}, Embed: func(context.Context, string) ([]float32, error) { return constVec(), nil }, Log: log},
		Private: agent.DefaultPrivateRegistry(nil, nil, nil, nil, nil), Rag: &rag.Service{DB: pool}, Gen: g, Events: agent.StoreEvents{Pool: pool}, Log: log}
	svc := &chat.Service{Pool: pool, Redis: rc, Agent: ag, PII: det, Lock: noLock{}, Members: coursepkg.Resolver{Pool: pool}, Log: log, Cfg: chat.Config{RatePerMin: 1000}}
	// một tài liệu READY có chunk trùng vectơ câu hỏi (Embed giả trả hằng số) để truy xuất có ngữ cảnh → lời gọi sinh chữ thật
	var doc uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `insert into documents (course_id, title, status) values ($1, 'Quy chế học vụ', 'READY') returning id`, course).Scan(&doc))
	_, err = pool.Exec(ctx, `insert into content_chunks (document_id, course_ids, audience, ord, text, embedding) values ($1, array[$2::uuid], 'ALL', 0, 'Điều 5. Sinh viên được thi lại một lần.', array_fill(0.25::real, array[1536])::vector)`, doc, course)
	require.NoError(t, err)
	reg.Fake().Set(fake.Settings{StreamDelay: 1})
	before := len(reg.Fake().Payloads())
	for i := range 12 {
		me, other := students[i], students[(i+5)%len(students)]
		a := chat.Actor{UserID: uids[i], Role: auth.RoleStudent}
		sess, err := svc.CreateSession(ctx, a, chat.CreateSessionIn{CourseID: course})
		require.NoError(t, err)
		for _, text := range []string{
			fmt.Sprintf("Quy chế thi lại thế nào? Mình là %s (%s), mail %s, sđt %s.", me.name, me.code, me.email, me.phone),
			fmt.Sprintf("Điều 5 nói gì? Bạn của mình là %s, cccd %s, có được thi lại không?", strings.ToLower(foldVN(other.name)), other.cccd),
		} {
			m, err := svc.Send(ctx, a, sess.ID, uuid.New(), text)
			require.NoError(t, err)
			sk := &sink{}
			require.NoError(t, svc.Tail(ctx, sk, m, ""))
			require.Contains(t, sk.evs[len(sk.evs)-1], "done:", sk.evs)
			for _, e := range sk.evs {
				require.NotRegexp(t, `(?i)\[\[\s*(SV|MSSV|EMAIL|SDT|PHONE|CCCD)`, e, "placeholder lộ ra khung SSE")
			}
			svc.Wait()
		}
	}
	require.Greater(t, len(reg.Fake().Payloads())-before, 20, "chat thật phải tạo lời gọi LLM / nhúng để quét có nghĩa")

	payloads := reg.Fake().Payloads()
	require.GreaterOrEqual(t, len(payloads), 30)
	scanned, placeholders := 0, 0
	for _, p := range payloads {
		parts := append(append([]string(nil), p.Messages...), p.Inputs...)
		for _, part := range parts {
			scanned++
			placeholders += strings.Count(part, "[[")
			for _, s := range students {
				for _, secret := range s.secrets() {
					require.NotContains(t, part, secret, "rò %q trong payload %s", secret, p.Task)
				}
			}
		}
	}
	require.GreaterOrEqual(t, placeholders, 10, "payload phải chứa placeholder (kiểm có nghĩa)")
	t.Logf("payload đã quét: %d, phần nội dung: %d, placeholder: %d", len(payloads), scanned, placeholders)
}

type noLock struct{}

func (noLock) IsLocked(context.Context, uuid.UUID) (exam.Lock, bool, error) {
	return exam.Lock{}, false, nil
}
func (noLock) RecordChatBlocked(context.Context, uuid.UUID, uuid.UUID) error { return nil }

// sink bỏ khung SSE: kiểm chữ trên dây ở TestChatNoPlaceholderOnWire; ở đây chỉ đẩy luồng chạy hết.
func constVec() []float32 {
	v := make([]float32, llm.EmbedDims)
	for i := range v {
		v[i] = 0.25
	}
	return v
}

type sink struct{ evs []string }

func (s *sink) Frame(_, ev string, d []byte) error {
	s.evs = append(s.evs, ev+":"+string(d))
	return nil
}
