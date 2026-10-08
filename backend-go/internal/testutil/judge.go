package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	judgeName = "edupilot-test-judge"
	// JudgeToken là token thử của container judge dùng chung (≥ 16 ký tự).
	JudgeToken = "edupilot-test-judge-token"
	// JudgeParallelism khớp -parallelism của container thử (P = 2).
	JudgeParallelism = 2
)

var (
	judgeOnce sync.Once
	judgeURL  string
	judgeErr  error
	judgeCtr  testcontainers.Container
)

// JudgeURL dựng (hoặc dùng lại) container `judge` THẬT từ deploy/judge (go-judge v1.13.0 + g++ 14.2.0), privileged + cgroup host (D58),
// P = 2, 2 GiB RAM; seccomp bật trừ khi chạy trên arm64 (colima: v1.13.0 + seccomp hỏng ⇒ `-no-seccomp`) hoặc JUDGE_EXTRA_ARGS đặt tay.
func JudgeURL(t testing.TB) string {
	t.Helper()
	RequireContainers(t)
	judgeOnce.Do(func() {
		judgeErr = withLock(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			_, file, _, _ := runtime.Caller(0)
			buildCtx := filepath.Join(filepath.Dir(file), "..", "..", "..", "deploy", "judge")
			extra, ok := os.LookupEnv("JUDGE_EXTRA_ARGS")
			if !ok && runtime.GOARCH == "arm64" {
				extra = "-no-seccomp"
			}
			req := testcontainers.ContainerRequest{
				Name:           judgeName,
				FromDockerfile: testcontainers.FromDockerfile{Context: buildCtx, Repo: "edupilot-judge", Tag: "test", KeepImage: true},
				ExposedPorts:   []string{"5050/tcp"},
				Env:            map[string]string{"JUDGE_TOKEN": JudgeToken, "JUDGE_PARALLELISM": fmt.Sprint(JudgeParallelism), "JUDGE_EXTRA_ARGS": extra},
				HostConfigModifier: func(hc *container.HostConfig) {
					hc.Privileged = true
					hc.CgroupnsMode = container.CgroupnsMode("host")
					hc.ShmSize = 256 << 20
					hc.Memory = 2 << 30
					hc.NanoCPUs = 2_000_000_000
				},
				WaitingFor: wait.ForHTTP("/version").WithPort("5050/tcp").WithStartupTimeout(2 * time.Minute),
			}
			c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true, Reuse: true})
			if err != nil {
				return fmt.Errorf("start judge: %w", err)
			}
			host, err := c.Host(ctx)
			if err != nil {
				return err
			}
			mp, err := c.MappedPort(ctx, "5050/tcp")
			if err != nil {
				return err
			}
			judgeCtr = c
			judgeURL = "http://" + host + ":" + mp.Port()
			return nil
		})
	})
	if judgeErr != nil {
		t.Fatalf("judge container: %v", judgeErr)
	}
	return judgeURL
}

// JudgeRestarts trả RestartCount của container judge (điều kiện chung của TestSandboxAttacks: không đổi).
func JudgeRestarts(t testing.TB) int {
	t.Helper()
	JudgeURL(t)
	info, err := judgeCtr.Inspect(context.Background())
	if err != nil {
		t.Fatalf("inspect judge: %v", err)
	}
	return info.RestartCount
}
