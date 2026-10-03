//go:build integration

package llm_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/provider"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/crypto"
	"github.com/edupilot/backend-go/internal/testutil"
)

func adminCtx(t *testing.T) context.Context {
	t.Helper()
	return auth.WithPrincipal(t.Context(), auth.Principal{Sub: "00000000-0000-7000-8000-0000000000a0", Role: auth.RoleAdmin})
}

const childKey = "ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk="

// TestReloadChild là tiến trình con (đặt LLM_RELOAD_CHILD=1): nạp cấu hình, nghe ep:llm:reload, in tên nhà chính của CHAT mỗi khi đổi.
func TestReloadChild(t *testing.T) {
	if os.Getenv("LLM_RELOAD_CHILD") != "1" {
		t.Skip("chỉ chạy như tiến trình con của TestReloadTwoProcesses")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	defer pool.Close()
	opt, err := goredis.ParseURL(os.Getenv("REDIS_URL"))
	require.NoError(t, err)
	rdb := goredis.NewClient(opt)
	defer func() { _ = rdb.Close() }()
	c, err := crypto.ParseKey(childKey)
	require.NoError(t, err)
	reg := llm.NewRegistry(llmconfig.NewResolver(pool, c), llm.EnvConfig{}, func(s llm.ProviderSpec) (provider.Provider, error) { return &stub{name: s.Name}, nil }, newDiscardLog())
	require.NoError(t, reg.Load(ctx))
	go reg.Watch(ctx, rdb)
	time.Sleep(300 * time.Millisecond) // chờ đăng ký pub/sub
	fmt.Println("READY")

	go func() { _, _ = io.Copy(io.Discard, os.Stdin); cancel() }() // cha đóng stdin → thoát
	last := ""
	for ctx.Err() == nil {
		if rt, err := reg.Route(llm.TaskChat); err == nil && rt.Targets[0].ProviderName != last {
			last = rt.Targets[0].ProviderName
			fmt.Println("CHAT=" + last)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

type child struct {
	cmd   *exec.Cmd
	lines chan string
	stdin io.WriteCloser
}

func startChild(t *testing.T, dbURL, redisURL string) *child {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestReloadChild$", "-test.v=false", "-test.timeout=60s") //nolint:gosec // chạy chính binary test
	cmd.Env = append(os.Environ(), "LLM_RELOAD_CHILD=1", "DATABASE_URL="+dbURL, "REDIS_URL="+redisURL)
	out, err := cmd.StdoutPipe()
	require.NoError(t, err)
	in, err := cmd.StdinPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	c := &child{cmd: cmd, lines: make(chan string, 64), stdin: in}
	go func() {
		sc := bufio.NewScanner(out)
		for sc.Scan() {
			c.lines <- strings.TrimSpace(sc.Text())
		}
		close(c.lines)
	}()
	t.Cleanup(func() { _ = in.Close(); _ = cmd.Wait() })
	return c
}

func (c *child) expect(t *testing.T, want string, within time.Duration) {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case l, ok := <-c.lines:
			if !ok {
				t.Fatalf("tiến trình con thoát trước khi in %q", want)
			}
			if l == want {
				return
			}
		case <-deadline:
			t.Fatalf("không thấy %q trong %v", want, within)
		}
	}
}

// TestReloadTwoProcesses — US-P1-02 AC14: đổi cấu hình rồi PUBLISH ep:llm:reload → cả hai tiến trình đổi trong ≤ 1 s.
func TestReloadTwoProcesses(t *testing.T) {
	dbURL, redisURL := testutil.MigratedPostgresURL(t), testutil.RedisURL(t)
	pool, err := pgxpool.New(t.Context(), dbURL)
	require.NoError(t, err)
	defer pool.Close()
	opt, err := goredis.ParseURL(redisURL)
	require.NoError(t, err)
	rdb := goredis.NewClient(opt)
	defer func() { _ = rdb.Close() }()
	c, err := crypto.ParseKey(childKey)
	require.NoError(t, err)
	svc := llmconfig.New(pool, c, llmconfig.WithOnChange(func(ctx context.Context) { _ = llm.PublishReload(ctx, rdb) }))
	ctx := adminCtx(t)

	a, err := svc.CreateProvider(ctx, llmconfig.ProviderInput{Type: "fake", Name: "A", Models: []llmconfig.ModelInput{cm("ma")}})
	require.NoError(t, err)
	b, err := svc.CreateProvider(ctx, llmconfig.ProviderInput{Type: "fake", Name: "B", Models: []llmconfig.ModelInput{cm("mb")}})
	require.NoError(t, err)
	_, err = svc.SetRoute(ctx, "CHAT", []uuid.UUID{a.Models[0].ID}, nil, 0)
	require.NoError(t, err)

	c1, c2 := startChild(t, dbURL, redisURL), startChild(t, dbURL, redisURL)
	for _, ch := range []*child{c1, c2} {
		ch.expect(t, "READY", 20*time.Second)
		ch.expect(t, "CHAT=A", 5*time.Second)
	}
	start := time.Now()
	_, err = svc.SetRoute(ctx, "CHAT", []uuid.UUID{b.Models[0].ID}, nil, 1) // OnChange → PUBLISH
	require.NoError(t, err)
	c1.expect(t, "CHAT=B", time.Second)
	c2.expect(t, "CHAT=B", time.Second)
	t.Logf("cả hai tiến trình đổi sau %v", time.Since(start))
}
