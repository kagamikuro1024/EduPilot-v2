package testutil

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/moby/moby/client"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	pgbouncerImage = "edoburu/pgbouncer:v1.26.0-p0"
	pgbouncerName  = "edupilot-test-pgbouncer"
)

var (
	pgbOnce sync.Once
	pgbHost string
	pgbErr  error
)

// pgbouncerINI là cấu hình hiệu lực của SRS 8.3 (transaction mode, không prepared statement phía server),
// chuyển tiếp MỌI database tới container Postgres dùng chung.
func pgbouncerINI(pgIP string) string {
	return fmt.Sprintf(`[databases]
* = host=%s port=5432

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
unix_socket_dir =
auth_file = /etc/pgbouncer/userlist.txt
auth_type = scram-sha-256
admin_users = edupilot
pool_mode = transaction
max_client_conn = 200
default_pool_size = 20
reserve_pool_size = 5
reserve_pool_timeout = 3
query_wait_timeout = 30
max_prepared_statements = 0
ignore_startup_parameters = extra_float_digits
`, pgIP)
}

// containerIP lấy địa chỉ IP (mạng bridge mặc định) của một container đang chạy theo tên.
func containerIP(ctx context.Context, name string) (string, error) {
	p, err := testcontainers.NewDockerProvider()
	if err != nil {
		return "", err
	}
	defer func() { _ = p.Close() }()
	info, err := p.Client().ContainerInspect(ctx, name, client.ContainerInspectOptions{})
	if err != nil {
		return "", err
	}
	for _, n := range info.Container.NetworkSettings.Networks {
		if n.IPAddress.IsValid() {
			return n.IPAddress.String(), nil
		}
	}
	return "", fmt.Errorf("container %s không có IP", name)
}

// PgBouncerHostPort trả host:port của PgBouncer (transaction mode) đặt TRƯỚC container Postgres dùng chung.
func PgBouncerHostPort(t testing.TB) string {
	t.Helper()
	RequireContainers(t)
	PostgresHostPort(t) // bảo đảm Postgres đã chạy
	pgbOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		ip, err := containerIP(ctx, pgName)
		if err != nil {
			pgbErr = fmt.Errorf("ip postgres: %w", err)
			return
		}
		pgbHost, pgbErr = ensure(pgbouncerName, testcontainers.ContainerRequest{
			Image:        pgbouncerImage,
			ExposedPorts: []string{"6432/tcp"},
			Entrypoint:   []string{"/usr/bin/pgbouncer"},
			Cmd:          []string{"/etc/pgbouncer/pgbouncer.ini"},
			Files: []testcontainers.ContainerFile{
				{Reader: strings.NewReader(pgbouncerINI(ip)), ContainerFilePath: "/etc/pgbouncer/pgbouncer.ini", FileMode: 0o644},
				{Reader: strings.NewReader(fmt.Sprintf("%q %q\n", pgUser, pgPass)), ContainerFilePath: "/etc/pgbouncer/userlist.txt", FileMode: 0o644},
			},
			WaitingFor: wait.ForListeningPort("6432/tcp").WithStartupTimeout(time.Minute),
		}, "6432/tcp", func(hp string) error {
			cctx, ccancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer ccancel()
			c, err := pgx.Connect(cctx, pgDSN(hp, "postgres"))
			if err != nil {
				return err
			}
			defer func() { _ = c.Close(cctx) }()
			return c.Ping(cctx)
		})
	})
	if pgbErr != nil {
		t.Fatalf("pgbouncer container: %v", pgbErr)
	}
	return pgbHost
}

// PgBouncerURL trả URL tới MỘT database mới (đã migrate) đi QUA PgBouncer transaction mode.
func PgBouncerURL(t testing.TB) string {
	t.Helper()
	direct := MigratedPostgresURL(t)
	name := direct[strings.LastIndex(direct, "/")+1 : strings.Index(direct, "?")]
	return pgDSN(PgBouncerHostPort(t), name)
}
