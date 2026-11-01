package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/mailin"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/blob"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/testutil"
)

const (
	imapUser   = "nop@truong.edu"
	imapPass   = "mat-khau-thu"
	trustedID  = "mx.truong.edu"
	authPass   = trustedID + "; dkim=pass header.d=truong.edu; dmarc=pass (p=none) header.from=truong.edu"
	systemFrom = "no-reply@edupilot.local"
)

type literal struct {
	io.Reader
	n int64
}

func (l literal) Size() int64 { return l.n }

// countingListener đếm kết nối nhận được (mỗi lượt poll = một kết nối).
type countingListener struct {
	net.Listener
	mu sync.Mutex
	n  int
}

func (c *countingListener) Accept() (net.Conn, error) {
	conn, err := c.Listener.Accept()
	if err == nil {
		c.mu.Lock()
		c.n++
		c.mu.Unlock()
	}
	return conn, err
}

func (c *countingListener) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// imapRig: máy chủ IMAP THỬ chạy trong test (imapserver + imapmemserver; Mailpit không có IMAP), Postgres + Redis + MinIO thật, một lớp có hai sinh viên (MSSV 20229001 / 20229002) và bài BT03.
type imapRig struct {
	t       *testing.T
	pool    *pgxpool.Pool
	rdb     *appredis.Client
	blob    *blob.Store
	cipher  *crypto.Cipher
	course  uuid.UUID
	teacher uuid.UUID
	admin   uuid.UUID
	ta      uuid.UUID
	asg     uuid.UUID
	kha, b  uuid.UUID // sinh viên MSSV 20229001 / 20229002
	mem     *imapmemserver.Server
	box     *imapmemserver.User
	addr    string
	ln      *countingListener
	srv     *imapserver.Server
	svc     *mailin.Service
	sy      *mailin.Syncer
	now     time.Time
}

func (r *imapRig) exec(sql string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), sql, args...)
	require.NoError(r.t, err, sql)
}

func (r *imapRig) one(dst any, sql string, args ...any) {
	r.t.Helper()
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(dst), sql)
}

func (r *imapRig) user(role, email string, verified bool) uuid.UUID {
	var id uuid.UUID
	r.one(&id, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Người Thử', $2::user_role, 'ACTIVE', 'x', case when $3 then now() end) returning id`, email, role, verified)
	return id
}

func (r *imapRig) enroll(course, user uuid.UUID, role, code string) {
	var snap any
	if code != "" {
		snap = code
	}
	r.exec(`insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) values ($1, $2, $3::enrollment_role, 'ACTIVE', 'ROSTER', $4)`, course, user, role, snap)
}

// newCourse dựng lớp + bài BT03 (kênh IMAP bật); trả lớp và bài.
func (r *imapRig) newCourse(code string) (course, asg uuid.UUID) {
	id := uuid.New()
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	jc := make([]byte, 7)
	for i := range jc {
		jc[i] = alpha[int(id[i])%len(alpha)]
	}
	r.one(&course, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'ANM', '2026-2027-HK1', $2, $3) returning id`, "M-"+id.String()[:8], string(jc), r.teacher)
	r.enroll(course, r.teacher, "TEACHER", "")
	r.one(&asg, `insert into assignments (course_id, title, due_at, submission_code, created_by, channels, published_at, rubric) values ($1, 'BT03', now() + interval '3 days', $2, $3, ARRAY['IN_APP','IMAP'], now(), '[]') returning id`, course, code, r.teacher)
	return course, asg
}

func newImap(t *testing.T) *imapRig {
	t.Helper()
	testutil.RequireContainers(t)
	pool := testutil.RuntimePool(t)
	ctx := t.Context()
	rdb, err := appredis.New(ctx, testutil.RedisURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })
	st, err := blob.New(ctx, blob.Config{Endpoint: testutil.MinIOEndpoint(t), Bucket: "imap-" + strings.ToLower(testutil.TestPrefix(t)), AccessKey: testutil.MinIOAccessKey, SecretKey: testutil.MinIOSecretKey, EnsureBucket: true})
	require.NoError(t, err)
	cipher, err := crypto.ParseKey("ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk=")
	require.NoError(t, err)
	r := &imapRig{t: t, pool: pool, rdb: rdb, blob: st, cipher: cipher, now: time.Now().UTC()}
	r.teacher = r.user("TEACHER", "gv."+uuid.NewString()[:6]+"@truong.edu", true)
	r.admin = r.user("ADMIN", "ad."+uuid.NewString()[:6]+"@truong.edu", true)
	r.ta = r.user("TA", "ta."+uuid.NewString()[:6]+"@truong.edu", true)
	r.course, r.asg = r.newCourse("BT03")
	r.enroll(r.course, r.ta, "TA", "")
	tag := uuid.NewString()[:6]
	r.kha = r.user("STUDENT", "sv.kha."+tag+"@truong.edu", true)
	r.b = r.user("STUDENT", "sv.gioi."+tag+"@truong.edu", true)
	r.enroll(r.course, r.kha, "STUDENT", "20229001")
	r.enroll(r.course, r.b, "STUDENT", "20229002")

	r.mem = imapmemserver.New()
	r.box = imapmemserver.NewUser(imapUser, imapPass)
	require.NoError(t, r.box.Create("INBOX", nil))
	r.mem.AddUser(r.box)
	r.listen("127.0.0.1:0")
	t.Cleanup(r.stop)

	log := slog.New(slog.DiscardHandler)
	r.svc = &mailin.Service{Pool: pool, Cipher: cipher, Redis: rdb, Jobs: jobs.NewService(pool), TestTimeout: 500 * time.Millisecond, MinSync: 30 * time.Second}
	r.sy = &mailin.Syncer{Pool: pool, Redis: rdb, Cipher: cipher, Blob: st, Notify: &notify.Service{Pool: pool, Log: log}, Log: log, TrustedAuthservID: trustedID, SystemFrom: systemFrom,
		DefaultPoll: time.Minute, BackoffMax: 15 * time.Minute, ConnTimeout: 3 * time.Second, Now: func() time.Time { return r.now }}
	return r
}

func (r *imapRig) listen(addr string) {
	r.t.Helper()
	ln, err := net.Listen("tcp", addr)
	require.NoError(r.t, err)
	r.ln = &countingListener{Listener: ln}
	r.addr = ln.Addr().String()
	r.srv = imapserver.New(&imapserver.Options{NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
		return r.mem.NewSession(), nil, nil
	}, InsecureAuth: true})
	go func() { _ = r.srv.Serve(r.ln) }()
}

func (r *imapRig) stop() {
	if r.srv != nil {
		_ = r.srv.Close()
		r.srv = nil
	}
}

func (r *imapRig) restart() {
	r.t.Helper()
	r.stop()
	var err error
	for range 50 {
		var ln net.Listener
		if ln, err = net.Listen("tcp", r.addr); err == nil {
			_ = ln.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(r.t, err)
	r.listen(r.addr)
}

func (r *imapRig) config(course uuid.UUID, password string, version int32) mailin.View {
	r.t.Helper()
	host, port, _ := net.SplitHostPort(r.addr)
	var p int
	_, _ = fmt.Sscanf(port, "%d", &p)
	v, err := r.svc.Put(r.t.Context(), r.admin, course, mailin.PutIn{Host: host, Port: p, Security: mailin.SecNone, Username: imapUser, Password: password, Mailbox: "INBOX", PollSeconds: 60, Enabled: true, Version: version})
	require.NoError(r.t, err)
	return v
}

func (r *imapRig) ready() {
	r.t.Helper()
	r.config(r.course, imapPass, 0)
}

// att là một đính kèm.
type att struct {
	name string
	data []byte
	ctyp string
}

// mailSpec mô tả một thư thử.
type mailSpec struct {
	from, subject string
	auth          string   // `Authentication-Results` do MÁY CHỦ THƯ thêm (trên cùng); rỗng = không có
	below         []string // header người gửi tự thêm (nằm DƯỚI header của máy chủ thư)
	autoSubmitted string
	atts          []att
	raw           string // thân thô thay hoàn toàn (thư hỏng)
	msgID         string // Message-ID cố định (gửi lại đúng thư cũ); rỗng = ngẫu nhiên
}

func (m mailSpec) messageID() string {
	if m.msgID != "" {
		return m.msgID
	}
	return uuid.NewString() + "@truong.edu"
}

func (m mailSpec) build() string {
	if m.raw != "" {
		return m.raw
	}
	var h strings.Builder
	if m.auth != "" {
		h.WriteString("Authentication-Results: " + m.auth + "\r\n")
	}
	h.WriteString("Received: from mail.truong.edu by mx.truong.edu\r\n")
	for _, l := range m.below {
		h.WriteString(l + "\r\n")
	}
	from := m.from
	if from == "" {
		from = "sv.kha@truong.edu"
	}
	h.WriteString("From: SV <" + from + ">\r\nTo: " + imapUser + "\r\nSubject: " + m.subject + "\r\nMessage-ID: <" + m.messageID() + ">\r\nMIME-Version: 1.0\r\n")
	if m.autoSubmitted != "" {
		h.WriteString("Auto-Submitted: " + m.autoSubmitted + "\r\n")
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	p, _ := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=utf-8"}})
	_, _ = p.Write([]byte("Em nộp bài ạ."))
	for _, a := range m.atts {
		ct := a.ctyp
		if ct == "" {
			ct = "application/octet-stream"
		}
		ap, _ := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {ct + `; name="` + a.name + `"`}, "Content-Disposition": {`attachment; filename="` + a.name + `"`}, "Content-Transfer-Encoding": {"base64"}})
		enc := base64.StdEncoding.EncodeToString(a.data)
		for len(enc) > 76 {
			_, _ = ap.Write([]byte(enc[:76] + "\r\n"))
			enc = enc[76:]
		}
		_, _ = ap.Write([]byte(enc))
	}
	_ = mw.Close()
	h.WriteString("Content-Type: multipart/mixed; boundary=" + mw.Boundary() + "\r\n\r\n")
	return h.String() + body.String()
}

// deliver đặt một thư vào INBOX (INTERNALDATE = `at`) và trả UID.
func (r *imapRig) deliver(m mailSpec, at time.Time) uint32 {
	r.t.Helper()
	raw := m.build()
	d, err := r.box.Append("INBOX", literal{strings.NewReader(raw), int64(len(raw))}, &imap.AppendOptions{Time: at})
	require.NoError(r.t, err)
	return uint32(d.UID)
}

func (r *imapRig) sync() (mailin.Result, error) {
	return r.sy.SyncCourse(r.t.Context(), r.course, nil)
}

func (r *imapRig) mustSync() mailin.Result {
	r.t.Helper()
	res, err := r.sync()
	require.NoError(r.t, err)
	return res
}

var s2 = func() []byte { return fixture("submissions/s2-kha.md") }

// goodMail là thư đúng mẫu `[BT03] 20229001` từ sinh viên kha, máy chủ thư xác thực.
func goodMail() mailSpec {
	return mailSpec{from: "", subject: "[BT03] 20229001", auth: authPass, atts: []att{{name: "bai.md", data: s2(), ctyp: "text/markdown"}}}
}

// khaMail đặt From đúng email của sv kha của rig.
func (r *imapRig) khaMail() mailSpec {
	m := goodMail()
	m.from = r.email(r.kha)
	m.auth = trustedID + "; dmarc=pass header.from=truong.edu"
	return m
}

func (r *imapRig) email(u uuid.UUID) (s string) {
	r.one(&s, `select email from users where id = $1`, u)
	return s
}

// inbound trả (status, reason) của các thư theo UID.
func (r *imapRig) inbound() (out [][2]string) {
	rows, err := r.pool.Query(r.t.Context(), `select status, coalesce(reason, '') from mail_inbound where course_id = $1 order by message_uid`, r.course)
	require.NoError(r.t, err)
	defer rows.Close()
	for rows.Next() {
		var a, b string
		require.NoError(r.t, rows.Scan(&a, &b))
		out = append(out, [2]string{a, b})
	}
	return out
}

func (r *imapRig) subs() int {
	var n int
	r.one(&n, `select count(*)::int from submissions where course_id = $1 and source = 'IMAP'`, r.course)
	return n
}

func fixture(rel string) []byte {
	b, err := os.ReadFile(asgFixDir + rel)
	if err != nil {
		panic(err)
	}
	return b
}

// gradeLikeS2 chấm bài `sub` bằng Grader thật + LLM giả trả đúng hai lượt mẫu S2 của expected_grading.json; trả id kết quả.
func (r *imapRig) gradeLikeS2(sub uuid.UUID, res *uuid.UUID) {
	r.t.Helper()
	var samples []struct {
		ID string          `json:"id"`
		P1 json.RawMessage `json:"llm_pass1"`
		P2 json.RawMessage `json:"llm_pass2"`
	}
	require.NoError(r.t, json.Unmarshal(fixture("expected_grading.json"), &samples))
	var s2 = samples[1]
	require.Equal(r.t, "S2", s2.ID)
	var rb struct {
		Criteria json.RawMessage `json:"criteria"`
	}
	require.NoError(r.t, json.Unmarshal(fixture("rubric-bt03.json"), &rb))
	r.exec(`update assignments set rubric = $2::jsonb, reference_answer = $3, max_score = 10 where id = $1`, r.asg, string(rb.Criteria), string(fixture("dap-an-chuan-bt03.md")))
	r.one(res, `insert into grading_results (course_id, submission_id) select course_id, id from submissions where id = $1 returning id`, sub)
	g := &grading.Grader{Pool: r.pool, Blob: r.blob, Extract: noExtract{}, Cfg: grading.DefaultEssayConfig(),
		LLM: structFn(func(_ context.Context, req llm.Request, _ json.RawMessage) (json.RawMessage, error) {
			if strings.Contains(strings.SplitN(req.Messages[1].Content, "\n", 2)[0], "c4, c3, c2, c1") {
				return s2.P2, nil
			}
			return s2.P1, nil
		})}
	_, err := g.Run(r.t.Context(), r.course, *res)
	require.NoError(r.t, err)
}
