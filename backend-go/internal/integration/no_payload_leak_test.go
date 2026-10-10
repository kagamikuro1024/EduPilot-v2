package integration_test

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/fake"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
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
// ponytail: chưa đi qua internal/chat / internal/thread (US-P3-05, 06 sẽ thêm kịch bản dùng dịch vụ thật vào CÙNG bảng secrets).
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
	for i, s := range students {
		var u uuid.UUID
		require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, $2, 'STUDENT') returning id`, s.email, s.name).Scan(&u))
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
