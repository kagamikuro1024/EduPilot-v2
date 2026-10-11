package rag_test

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/rag"
	"github.com/edupilot/backend-go/internal/testutil"
)

const canary = "CANARY-7Q2X"

// axis trả vectơ đơn vị theo trục i (cosine giữa hai trục khác nhau = 0; cùng trục = 1), pha thêm một chút trục j để có thứ hạng.
func axis(i int, mix map[int]float32) []float32 {
	v := make([]float32, rag.Dims)
	v[i] = 1
	for j, w := range mix {
		v[j] = w
	}
	return v
}

type fx struct {
	pool    *pgxpool.Pool
	svc     *rag.Service
	c1, c2  uuid.UUID
	teacher uuid.UUID
}

func newFx(t *testing.T) *fx {
	t.Helper()
	testutil.RequireContainers(t)
	pool, err := pgxpool.New(t.Context(), testutil.MigratedPostgresURL(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	f := &fx{pool: pool, svc: &rag.Service{DB: pool}}
	ctx := t.Context()
	require.NoError(t, pool.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'GV', 'TEACHER') returning id`, uuid.NewString()+"@example.test").Scan(&f.teacher))
	const alpha = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	course := func(seed byte) uuid.UUID {
		id := uuid.New()
		jc := make([]byte, 7)
		for i := range jc {
			jc[i] = alpha[(int(id[i])+int(seed)*7)%len(alpha)]
		}
		var c uuid.UUID
		require.NoError(t, pool.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`,
			"RG"+id.String()[:8], string(jc), f.teacher).Scan(&c))
		return c
	}
	f.c1, f.c2 = course(1), course(2)
	return f
}

type docOpt struct {
	typ, status string
	visible     bool
	rag         bool
	audience    string
}

func (f *fx) doc(t *testing.T, course uuid.UUID, title string, o docOpt) uuid.UUID {
	t.Helper()
	if o.typ == "" {
		o.typ = "LECTURE"
	}
	if o.status == "" {
		o.status = "READY"
	}
	if o.audience == "" {
		o.audience = "ALL"
	}
	var id uuid.UUID
	vis := o.visible
	require.NoError(t, f.pool.QueryRow(t.Context(), `insert into documents (course_id, title, type, status, visible_to_students, use_for_rag) values ($1, $2, $3::document_type, $4::document_status, $5, $6) returning id`,
		course, title, o.typ, o.status, vis, o.rag).Scan(&id))
	return id
}

func (f *fx) chunk(t *testing.T, doc uuid.UUID, courses []uuid.UUID, ord int, audience, text string, vec []float32) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	var emb any
	if vec != nil {
		emb = pgvector.NewVector(vec)
	}
	require.NoError(t, f.pool.QueryRow(t.Context(), `insert into content_chunks (document_id, course_ids, audience, ord, text, embedding) values ($1, $2, $3::chunk_audience, $4, $5, $6) returning id`,
		doc, courses, audience, ord, text, emb).Scan(&id))
	return id
}

func ids(hs []rag.Hit) []uuid.UUID {
	out := make([]uuid.UUID, len(hs))
	for i, h := range hs {
		out[i] = h.ChunkID
	}
	return out
}

// TestRetrievalFiltersInSQL — AC11: mỗi điều kiện loại đoạn đúng nơi (READY, use_for_rag, ANSWER_KEY, visible, audience, embedding, lớp, document_ids).
func TestRetrievalFiltersInSQL(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	q := axis(0, nil)
	okDoc := f.doc(t, f.c1, "Bài giảng", docOpt{visible: true, rag: true})
	good := f.chunk(t, okDoc, []uuid.UUID{f.c1}, 0, "ALL", "nội dung hợp lệ về mạng máy tính", q)
	staffOnly := f.doc(t, f.c1, "Nội bộ", docOpt{visible: false, rag: true})
	staffChunk := f.chunk(t, staffOnly, []uuid.UUID{f.c1}, 0, "STAFF", "ghi chú nội bộ của giảng viên", q)
	f.chunk(t, f.doc(t, f.c1, "Đang xử lý", docOpt{visible: true, rag: true, status: "PROCESSING"}), []uuid.UUID{f.c1}, 0, "ALL", "chưa READY", q)
	f.chunk(t, f.doc(t, f.c1, "Không RAG", docOpt{visible: true, rag: false}), []uuid.UUID{f.c1}, 0, "ALL", "use_for_rag=false", q)
	f.chunk(t, f.doc(t, f.c1, "Đáp án", docOpt{typ: "ANSWER_KEY", visible: false, rag: true}), []uuid.UUID{f.c1}, 0, "GRADING", "đáp án "+canary, q)
	f.chunk(t, okDoc, []uuid.UUID{f.c1}, 1, "ALL", "chưa nhúng", nil)
	f.chunk(t, f.doc(t, f.c2, "Lớp khác", docOpt{visible: true, rag: true}), []uuid.UUID{f.c2}, 0, "ALL", "thuộc lớp 2", q)

	hs, err := f.svc.SearchStudent(t.Context(), rag.Query{CourseID: f.c1, Vec: q, Text: "mạng máy tính"})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{good}, ids(hs), "sinh viên chỉ thấy đoạn hợp lệ")
	require.InDelta(t, 1.0, hs[0].Cosine, 1e-6)

	hs, err = f.svc.SearchStaff(t.Context(), rag.Query{CourseID: f.c1, Vec: q, Text: "mạng máy tính"})
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{good, staffChunk}, ids(hs), "Staff thêm audience STAFF, vẫn không có ANSWER_KEY")

	hs, err = f.svc.SearchStaff(t.Context(), rag.Query{CourseID: f.c1, Vec: q, DocumentIDs: []uuid.UUID{staffOnly}})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{staffChunk}, ids(hs), "giới hạn document_id")
}

// TestRetrievalStillFullAfterForbiddenNeighbors — AC11: 40 láng giềng gần nhất đều bị cấm mà vẫn đủ K đoạn hợp lệ.
func TestRetrievalStillFullAfterForbiddenNeighbors(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	q := axis(0, nil)
	key := f.doc(t, f.c1, "Đáp án", docOpt{typ: "ANSWER_KEY", rag: true})
	for i := range 60 { // 60 > 40: nếu lọc SAU khi lấy top-40 thì không còn gì
		f.chunk(t, key, []uuid.UUID{f.c1}, i, "ALL", fmt.Sprintf("đáp án %d %s", i, canary), axis(0, map[int]float32{1: 0.001 * float32(i)}))
	}
	ok := f.doc(t, f.c1, "Bài giảng", docOpt{visible: true, rag: true})
	for i := range 12 {
		f.chunk(t, ok, []uuid.UUID{f.c1}, i, "ALL", fmt.Sprintf("bài giảng đoạn %d", i), axis(0, map[int]float32{2: 0.5 + 0.01*float32(i)}))
	}
	hs, err := f.svc.SearchStudent(t.Context(), rag.Query{CourseID: f.c1, Vec: q, Text: "đáp án"})
	require.NoError(t, err)
	require.Len(t, hs, rag.DefaultTopK)
	for _, h := range hs {
		require.NotContains(t, h.Text, canary)
	}
}

// TestAnswerKeyNeverRetrieved — AC12 (cổng của M9): kể cả khi audience bị gán nhầm ALL, use_for_rag=true và visible_to_students bị sửa thẳng trong DB.
func TestAnswerKeyNeverRetrieved(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	q := axis(3, nil)
	key := f.doc(t, f.c1, "Đáp án giữa kỳ", docOpt{typ: "ANSWER_KEY", visible: false, rag: true})
	// audience gán nhầm ALL, vectơ trùng hệt câu hỏi, văn bản trùng từ khoá: mọi điều kiện khác đều cho qua
	f.chunk(t, key, []uuid.UUID{f.c1}, 0, "ALL", "Đáp án câu 3 là B "+canary, q)
	f.chunk(t, key, []uuid.UUID{f.c1}, 1, "STAFF", "Đáp án câu 4 là C "+canary, q)
	f.chunk(t, key, []uuid.UUID{f.c1}, 2, "GRADING", "Đáp án câu 5 là D "+canary, q)
	ok := f.doc(t, f.c1, "Bài giảng", docOpt{visible: true, rag: true})
	f.chunk(t, ok, []uuid.UUID{f.c1}, 0, "ALL", "đáp án trắc nghiệm được giải thích trên lớp", axis(3, map[int]float32{4: 0.3}))

	// DB không cho đổi visible_to_students của ANSWER_KEY (documents_answer_key_chk)
	_, err := f.pool.Exec(t.Context(), `update documents set visible_to_students = true where id = $1`, key)
	require.Error(t, err)

	for name, search := range map[string]func(context.Context, rag.Query) ([]rag.Hit, error){"student": f.svc.SearchStudent, "staff": f.svc.SearchStaff} {
		for _, docs := range [][]uuid.UUID{nil, {key}} { // cả lớp và chat giới hạn document_id của chính tài liệu đáp án
			hs, err := search(t.Context(), rag.Query{CourseID: f.c1, Vec: q, Text: "đáp án câu 3 " + canary, DocumentIDs: docs})
			require.NoError(t, err, name)
			for _, h := range hs {
				require.NotContains(t, h.Text, canary, name)
				require.NotEqual(t, key, h.DocumentID, name)
			}
			if docs != nil {
				require.Empty(t, hs, name+": giới hạn document_id của tài liệu đáp án không trả gì")
			}
		}
	}
}

// TestHybridRRF — AC11: đoạn chỉ khớp từ khoá (vectơ xa) và đoạn chỉ khớp vectơ đều ra; đoạn khớp cả hai xếp đầu.
func TestHybridRRF(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	d := f.doc(t, f.c1, "Quy chế", docOpt{visible: true, rag: true})
	both := f.chunk(t, d, []uuid.UUID{f.c1}, 0, "ALL", "Điều kiện bị cảnh báo học vụ được quy định tại Điều 5", axis(0, nil))
	vecOnly := f.chunk(t, d, []uuid.UUID{f.c1}, 1, "ALL", "đoạn không liên quan chữ nào", axis(0, map[int]float32{1: 0.2}))
	kwOnly := f.chunk(t, d, []uuid.UUID{f.c1}, 2, "ALL", "cảnh báo học vụ lần hai", axis(5, nil))
	hs, err := f.svc.SearchStudent(t.Context(), rag.Query{CourseID: f.c1, Vec: axis(0, nil), Text: "cảnh báo học vụ"})
	require.NoError(t, err)
	require.Equal(t, both, hs[0].ChunkID)
	require.Contains(t, ids(hs), vecOnly)
	require.Contains(t, ids(hs), kwOnly)
}

// TestKeywordBigramFindsNaturalQuestion — AC11: câu hỏi tự nhiên (không dấu, có từ thừa) vẫn đưa đúng đoạn lên đầu nhờ cặp âm tiết liền nhau.
func TestKeywordBigramFindsNaturalQuestion(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	d := f.doc(t, f.c1, "Quy chế", docOpt{visible: true, rag: true})
	target := f.chunk(t, d, []uuid.UUID{f.c1}, 0, "ALL", "Sinh viên bị cảnh báo học vụ khi điểm trung bình dưới 1,0", axis(7, nil))
	for i := range 20 {
		f.chunk(t, d, []uuid.UUID{f.c1}, i+1, "ALL", fmt.Sprintf("sinh viên đăng ký học phần số %d điều kiện khác", i), axis(8+i, nil))
	}
	hs, err := f.svc.SearchStudent(t.Context(), rag.Query{CourseID: f.c1, Vec: axis(500, nil), Text: "em bi canh bao hoc vu khi nao"})
	require.NoError(t, err)
	require.NotEmpty(t, hs)
	require.Equal(t, target, hs[0].ChunkID)
}

// TestEnableRAGBeforeReindexNoError — AC11: tài liệu use_for_rag=true nhưng đoạn chưa nhúng không làm lỗi và không ra kết quả.
func TestEnableRAGBeforeReindexNoError(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	d := f.doc(t, f.c1, "Chưa nhúng", docOpt{visible: true, rag: true})
	f.chunk(t, d, []uuid.UUID{f.c1}, 0, "ALL", "cảnh báo học vụ", nil)
	hs, err := f.svc.SearchStudent(t.Context(), rag.Query{CourseID: f.c1, Vec: axis(0, nil), Text: "cảnh báo học vụ"})
	require.NoError(t, err)
	require.Empty(t, hs)
}

// TestCourseIsolation — AC13: đoạn lớp B không ra ở lớp A; chia sẻ (course_ids chứa A) thì ra ở cả hai.
func TestCourseIsolation(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	d := f.doc(t, f.c2, "Của lớp B", docOpt{visible: true, rag: true})
	c := f.chunk(t, d, []uuid.UUID{f.c2}, 0, "ALL", "tài liệu riêng lớp B", axis(0, nil))
	q := rag.Query{CourseID: f.c1, Vec: axis(0, nil), Text: "tài liệu riêng"}
	hs, err := f.svc.SearchStudent(t.Context(), q)
	require.NoError(t, err)
	require.Empty(t, hs)
	_, err = f.pool.Exec(t.Context(), `update content_chunks set course_ids = array_append(course_ids, $1) where id = $2`, f.c1, c)
	require.NoError(t, err)
	hs, err = f.svc.SearchStudent(t.Context(), q)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{c}, ids(hs))
}

func TestBadQuery(t *testing.T) {
	t.Parallel()
	svc := &rag.Service{}
	_, err := svc.SearchStudent(context.Background(), rag.Query{Vec: axis(0, nil)})
	require.ErrorIs(t, err, rag.ErrBadQuery)
	_, err = svc.SearchStudent(context.Background(), rag.Query{CourseID: uuid.New(), Vec: []float32{1, 2}})
	require.ErrorIs(t, err, rag.ErrBadQuery)
}

// TestNoPublicGradingSearch — AC12: gói rag không xuất hàm nào tìm đoạn GRADING và không nhắc audience GRADING trong mã sản phẩm.
func TestNoPublicGradingSearch(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		require.NoError(t, err)
		file, err := parser.ParseFile(fset, f, src, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Name.IsExported() {
					require.NotContains(t, strings.ToLower(x.Name.Name), "grading", "%s: hàm xuất có tên chấm điểm", f)
				}
			case *ast.SelectorExpr:
				require.NotEqual(t, "ChunkAudienceGRADING", x.Sel.Name, "%s: nhắc audience GRADING", f)
			}
			return true
		})
	}
}

// TestDocScopedChatOnlyThatDoc — US-P8-02 AC11: truy xuất của phiên "Hỏi AI về tài liệu" (Query.DocumentIDs) chỉ trả đoạn của tài liệu đó, dù đoạn của tài liệu khác khớp hơn.
func TestDocScopedChatOnlyThatDoc(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	a := f.doc(t, f.c1, "A", docOpt{visible: true, rag: true})
	b := f.doc(t, f.c1, "B", docOpt{visible: true, rag: true})
	ca := f.chunk(t, a, []uuid.UUID{f.c1}, 0, "ALL", "nội dung A xa", axis(1, nil))
	f.chunk(t, b, []uuid.UUID{f.c1}, 0, "ALL", "nội dung B khớp hơn", axis(0, nil))
	q := rag.Query{CourseID: f.c1, Vec: axis(0, nil), Text: "nội dung", K: 8}
	all, err := f.svc.SearchStudent(t.Context(), q)
	require.NoError(t, err)
	require.Len(t, all, 2)
	q.DocumentIDs = []uuid.UUID{a}
	only, err := f.svc.SearchStudent(t.Context(), q)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{ca}, ids(only))
}
