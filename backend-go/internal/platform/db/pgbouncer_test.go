package db_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/edupilot/backend-go/internal/platform/config"
	"github.com/edupilot/backend-go/internal/platform/db"
	applog "github.com/edupilot/backend-go/internal/platform/log"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/testutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/require"
)

// bouncerPool dựng pool theo ĐÚNG đường runtime: cfg.PgBouncerURL → db.NewPool (QueryExecModeExec), tới PgBouncer thật
// (transaction mode) đặt trước Postgres. Dùng chung một database đã migrate cho cả các test dưới đây.
func bouncerPool(t *testing.T, maxConns int32) *pgxpool.Pool {
	t.Helper()
	cfg := config.Config{
		Role:          config.Gateway,
		DatabaseURL:   testutil.MigratedPostgresURL(t),
		DBMaxConns:    maxConns,
		DBSlowQueryMS: 200,
		LogLevel:      "error",
		InstanceID:    "pgb-test",
	}
	cfg.PgBouncerURL = viaBouncer(t, cfg.DatabaseURL)
	require.Equal(t, "pgbouncer", cfg.DBVia())
	pool, err := db.NewPool(context.Background(), cfg, applog.NewTo(&safeBuf{}, cfg, "gateway"))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

// viaBouncer đổi host:port của URL trực tiếp sang PgBouncer, giữ nguyên database.
func viaBouncer(t *testing.T, direct string) string {
	t.Helper()
	at := strings.Index(direct, "@")
	slash := strings.Index(direct[at:], "/") + at
	return direct[:at+1] + testutil.PgBouncerHostPort(t) + direct[slash:]
}

// TestPgBouncer_TransactionMode — US-PG-07 AC9: 50 goroutine × 20 vòng, hỗn hợp truy vấn sqlc, transaction nhiều câu,
// truy vấn vector và pg_sleep ngắn, qua PgBouncer THẬT ở transaction mode: không lỗi `prepared statement … already
// exists / does not exist`, không lỗi giao thức. Đồng thời chứng minh PgBouncer đang ở transaction mode (SHOW CONFIG).
func TestPgBouncer_TransactionMode(t *testing.T) {
	pool := bouncerPool(t, 20)
	ctx := context.Background()

	// Đúng chế độ: hỏi chính PgBouncer (console `pgbouncer`, giao thức đơn giản).
	admin, err := pgx.Connect(ctx, adminURL(t, viaBouncer(t, testutil.PostgresURL(t))))
	require.NoError(t, err)
	t.Cleanup(func() { _ = admin.Close(ctx) })
	rows, err := admin.Query(ctx, "SHOW CONFIG", pgx.QueryExecModeSimpleProtocol)
	require.NoError(t, err)
	cfgVals := map[string]string{}
	for rows.Next() {
		vals, err := rows.Values()
		require.NoError(t, err)
		cfgVals[fmt.Sprint(vals[0])] = fmt.Sprint(vals[1])
	}
	require.NoError(t, rows.Err())
	require.Equal(t, "transaction", cfgVals["pool_mode"])
	require.Equal(t, "0", cfgVals["max_prepared_statements"])

	q := store.New(pool)
	_, err = pool.Exec(ctx, `create table if not exists pgb_vec (id uuid primary key default uuidv7(), embedding vector(3) not null)`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `insert into pgb_vec (embedding) values ('[1,0,0]'),('[0,1,0]'),('[0,0,1]')`)
	require.NoError(t, err)

	const workers, loops = 50, 20
	var wg sync.WaitGroup
	errs := make(chan error, workers*loops*4)
	for w := range workers {
		wg.Go(func() {
			for i := range loops {
				email := fmt.Sprintf("pgb-%d-%d-%s@example.test", w, i, uuid.NewString()[:8])
				// 1) truy vấn sqlc (insert + get).
				u, err := q.InsertUser(ctx, store.InsertUserParams{Email: email, FullName: "Người Thử", Role: store.UserRoleSTUDENT, Status: store.UserStatusINVITED})
				if err != nil {
					errs <- fmt.Errorf("InsertUser: %w", err)
					continue
				}
				if got, err := q.GetUser(ctx, u.ID); err != nil || got.Email != email {
					errs <- fmt.Errorf("GetUser: %v (email %q)", err, got.Email)
				}
				// 2) transaction nhiều câu (ghi + đọc + rollback/commit luân phiên).
				if err := func() error {
					tx, err := pool.Begin(ctx)
					if err != nil {
						return err
					}
					defer func() { _ = tx.Rollback(ctx) }()
					tq := store.New(tx)
					if _, err := tq.InsertAuditLog(ctx, store.InsertAuditLogParams{Entity: "pgb", EntityID: u.ID.String(), Action: "create"}); err != nil {
						return err
					}
					var n int
					if err := tx.QueryRow(ctx, `select count(*) from audit_log where entity_id = $1`, u.ID.String()).Scan(&n); err != nil {
						return err
					}
					if n != 1 {
						return fmt.Errorf("audit_log: %d dòng (cần 1)", n)
					}
					if i%2 == 0 {
						return nil // rollback
					}
					return tx.Commit(ctx)
				}(); err != nil {
					errs <- fmt.Errorf("tx: %w", err)
				}
				// 3) truy vấn vector.
				var id uuid.UUID
				if err := pool.QueryRow(ctx, `select id from pgb_vec order by embedding <-> $1 limit 1`, pgvector.NewVector([]float32{0, 1, 0})).Scan(&id); err != nil {
					errs <- fmt.Errorf("vector: %w", err)
				}
				// 4) pg_sleep ngắn.
				if _, err := pool.Exec(ctx, `select pg_sleep($1)`, 0.002); err != nil {
					errs <- fmt.Errorf("pg_sleep: %w", err)
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NotContains(t, err.Error(), "prepared statement")
		require.NoError(t, err)
	}
}

func adminURL(t *testing.T, u string) string {
	t.Helper()
	// đổi database thành `pgbouncer` (console quản trị)
	slash := strings.LastIndex(u[:strings.Index(u, "?")], "/")
	return u[:slash+1] + "pgbouncer" + u[strings.Index(u, "?"):]
}

// TestPgBouncer_NoPreparedStatements — US-PG-07 AC9: pool runtime KHÔNG dùng prepared statement ngầm
// (QueryExecModeExec, cấm CacheStatement/CacheDescribe) và server không giữ prepared statement nào sau tải hỗn hợp.
func TestPgBouncer_NoPreparedStatements(t *testing.T) {
	pool := bouncerPool(t, 10)
	ctx := context.Background()
	require.Equal(t, pgx.QueryExecModeExec, pool.Config().ConnConfig.DefaultQueryExecMode,
		"runtime qua PgBouncer phải dùng QueryExecModeExec (mỗi lệnh gửi kèm tham số, không Parse/Describe riêng)")

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 30 {
				var one int
				if err := pool.QueryRow(ctx, `select $1::int + 1`, 41).Scan(&one); err != nil || one != 42 {
					t.Errorf("select: %v (%d)", err, one)
					return
				}
			}
		})
	}
	wg.Wait()

	// Mỗi lần lấy một kết nối server (transaction mode) đều không có prepared statement nào.
	for range 40 {
		var n int
		require.NoError(t, pool.QueryRow(ctx, `select count(*) from pg_prepared_statements`).Scan(&n))
		require.Zero(t, n, "không được có prepared statement ở phía server")
	}

	// Đối chứng: chế độ cache statement bị cấm — pool tự dựng bằng CacheStatement gặp lỗi hoặc để lại statement,
	// nên cấu hình runtime cấm dùng nó (không khẳng định lỗi cụ thể vì phụ thuộc phiên bản PgBouncer).
	require.NotEqual(t, pgx.QueryExecModeCacheStatement, pool.Config().ConnConfig.DefaultQueryExecMode)
	require.NotEqual(t, pgx.QueryExecModeCacheDescribe, pool.Config().ConnConfig.DefaultQueryExecMode)
}

// TestVector_ThroughPgBouncer — US-PG-02 AC10 (lệnh 2) + US-PG-07 AC9: kiểu vector/halfvec và truy vấn `<=>` có lọc
// course_id chạy đúng qua PgBouncer transaction mode (pgvector-go + AfterConnect đăng ký kiểu).
func TestVector_ThroughPgBouncer(t *testing.T) {
	pool := bouncerPool(t, 8)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	const table = "vec_through_pgb"
	_, err := pool.Exec(ctx, `create table `+table+` (
		id uuid primary key default uuidv7(), course_id uuid not null, embedding vector(1536) not null)`)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `drop table if exists `+table) }) //nolint:contextcheck // dọn cuối test
	_, err = pool.Exec(ctx, `create index `+table+`_hnsw on `+table+`
		using hnsw ((embedding::halfvec(1536)) halfvec_cosine_ops) with (m = 16, ef_construction = 64)`)
	require.NoError(t, err)

	course := uuid.New()
	_, err = pool.Exec(ctx, `insert into `+table+` (course_id, embedding)
		select $1::uuid, (select array_agg(case when s = i % 1536 then 1::real else 0.001::real end order by s)
		                    from generate_series(0, 1535) s)::vector(1536)
		  from generate_series(1, 300) i`, course)
	require.NoError(t, err)

	probeSlice := make([]float32, 1536)
	for i := range probeSlice {
		probeSlice[i] = 0.001
	}
	probeSlice[7] = 1
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 10 {
				rows, err := pool.Query(ctx, `select id from `+table+`
					where course_id = $2 order by embedding::halfvec(1536) <=> $1::halfvec(1536) limit 3`,
					pgvector.NewHalfVector(probeSlice), course)
				if err != nil {
					t.Errorf("query: %v", err)
					return
				}
				n := 0
				for rows.Next() {
					n++
				}
				if err := rows.Err(); err != nil || n != 3 {
					t.Errorf("rows: %v (%d dòng)", err, n)
					return
				}
			}
		})
	}
	wg.Wait()

	// Round-trip kiểu vector.
	var back pgvector.Vector
	require.NoError(t, pool.QueryRow(ctx, `select $1::vector(1536)`, pgvector.NewVector(probeSlice)).Scan(&back))
	require.Equal(t, probeSlice, back.Slice())
}
