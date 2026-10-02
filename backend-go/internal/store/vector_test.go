package store_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/require"
)

// TestVectorConventions — US-PG-02 AC10: quy ước embedding của khối chú thích trong 00001 chạy thật:
// cột vector(1536), index HNSW trên biểu thức halfvec(1536) (halfvec_cosine_ops, m=16, ef_construction=64),
// truy vấn `<=>` có lọc course_id trong CÙNG câu, plan dùng Index Scan … hnsw, bảng tạm bị xoá cuối test.
func TestVectorConventions(t *testing.T) {
	t.Parallel()
	ctx, conn := connect(t)

	const table = "vec_convention_demo"
	t.Cleanup(func() {
		_, _ = conn.Exec(ctx, `drop table if exists `+table) //nolint:contextcheck // ctx của test
	})

	_, err := conn.Exec(ctx, `create table `+table+` (
		id         uuid         not null default uuidv7(),
		course_id  uuid         not null,
		content    text         not null,
		embedding  vector(1536) not null,
		constraint `+table+`_pkey primary key (id))`)
	require.NoError(t, err)

	_, err = conn.Exec(ctx, `create index `+table+`_embedding_hnsw_idx on `+table+`
		using hnsw ((embedding::halfvec(1536)) halfvec_cosine_ops) with (m = 16, ef_construction = 64)`)
	require.NoError(t, err)

	// HNSW là xấp xỉ: nâng ef_search tới mức tối đa để top-1 chắc chắn đúng. Bảng 1.000 dòng nhỏ nên planner có thể chọn
	// Seq Scan khi ef_search lớn → ép dùng index để kiểm đúng đường chạy thật.
	_, err = conn.Exec(ctx, `set hnsw.ef_search = 1000`)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `set enable_seqscan = off`)
	require.NoError(t, err)

	courseID, otherCourse := uuid.New(), uuid.New()
	// 1.000 dòng, mỗi dòng một vector ngẫu nhiên (hạt giống cố định → tái lập). Không dùng vector "một chiều bật" (cơ sở trực
	// giao): đồ thị HNSW trên dữ liệu suy biến như vậy có nút không tới được (CI x86 không tìm thấy chính vector truy vấn).
	_, err = conn.Exec(ctx, `select setseed(0.42)`)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `insert into `+table+` (course_id, content, embedding)
		select case when i % 5 = 0 then $2::uuid else $1::uuid end, 'chunk ' || i,
		       (select array_agg(random()::real - 0.5 order by s) from generate_series(0, 1535) s where i > 0)::vector(1536)
		  from generate_series(1, 1000) i`, courseID, otherCourse)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `analyze `+table)
	require.NoError(t, err)

	// Lấy một dòng thuộc course_id làm vector truy vấn (kết quả đầu phải là chính nó).
	var wantID uuid.UUID
	var probe pgvector.Vector
	require.NoError(t, conn.QueryRow(ctx,
		`select id, embedding from `+table+` where course_id = $1 order by content limit 1`, courseID).Scan(&wantID, &probe))

	query := `select id, embedding::halfvec(1536) <=> $1::halfvec(1536) as dist from ` + table + `
		 where course_id = $2
		 order by embedding::halfvec(1536) <=> $1::halfvec(1536)
		 limit 5`
	arg := pgvector.NewHalfVector(probe.Slice())

	rows, err := conn.Query(ctx, query, arg, courseID)
	require.NoError(t, err)
	var ids []uuid.UUID
	var dists []float64
	for rows.Next() {
		var id uuid.UUID
		var d float64
		require.NoError(t, rows.Scan(&id, &d))
		ids = append(ids, id)
		dists = append(dists, d)
	}
	require.NoError(t, rows.Err())
	require.Len(t, ids, 5)
	// Kết quả đầu là chính vector truy vấn: khoảng cosine ≈ 0 (so khoảng cách thay vì id — làm tròn dấu phẩy động của
	// pgvector khác nhau giữa CPU, nên không đòi trùng id tuyệt đối). Vector ngẫu nhiên khác cách xa (≈ 0,5).
	require.Contains(t, ids, wantID, "vector truy vấn phải nằm trong top 5 (dist=%v)", dists)
	require.Less(t, dists[0], 0.001, "kết quả đầu phải là chính vector truy vấn (dist=%v)", dists)

	// Plan phải dùng index HNSW (không quét tuần tự).
	plan := explain(t, ctx, conn, `explain (analyze, costs off) `+query, arg, courseID)
	require.Contains(t, plan, "Index Scan using "+table+"_embedding_hnsw_idx")
	require.NotContains(t, plan, "Seq Scan")
}

func explain(t *testing.T, ctx context.Context, conn *pgx.Conn, sql string, args ...any) string {
	t.Helper()
	rows, err := conn.Query(ctx, sql, args...)
	require.NoError(t, err)
	defer rows.Close()
	var b strings.Builder
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		fmt.Fprintln(&b, line)
	}
	require.NoError(t, rows.Err())
	return b.String()
}
