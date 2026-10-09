package judge

import (
	"fmt"
	"strconv"
	"time"
)

// Settings là cấu hình JUDGE_* của worker (SRS 4.5.8; `.env.example`).
type Settings struct {
	Consumer        bool          // JUDGE_CONSUMER: đúng MỘT bản worker chấm bài
	URL             string        // JUDGE_URL
	Token           string        // JUDGE_TOKEN (≥ 16 ký tự)
	Parallelism     int           // JUDGE_PARALLELISM
	MemLimit        string        // JUDGE_MEM_LIMIT (khớp mem_limit của container judge)
	Lease           time.Duration // JUDGE_LEASE
	LeaseRenew      time.Duration // JUDGE_LEASE_RENEW
	ClaimIdle       time.Duration // JUDGE_CLAIM_IDLE
	HTTPSlack       time.Duration // JUDGE_HTTP_SLACK
	MaxTotalSeconds int           // JUDGE_MAX_TOTAL_SECONDS (kiểm ở `schedule`)
}

// SettingsFromEnv đọc JUDGE_* (mặc định theo SRS). Với `JUDGE_CONSUMER=true` còn kiểm token và công thức bộ nhớ: sai → lỗi NÊU TÊN BIẾN
// để worker thoát 1 (không tự sửa giá trị).
func SettingsFromEnv(getenv func(string) string) (Settings, error) {
	s := Settings{
		Consumer:        getenv("JUDGE_CONSUMER") == "true",
		URL:             orDefault(getenv("JUDGE_URL"), "http://judge:5050"),
		Token:           getenv("JUDGE_TOKEN"),
		Parallelism:     2,
		MemLimit:        orDefault(getenv("JUDGE_MEM_LIMIT"), "2g"),
		Lease:           30 * time.Second,
		LeaseRenew:      10 * time.Second,
		ClaimIdle:       10 * time.Minute,
		HTTPSlack:       5 * time.Second,
		MaxTotalSeconds: 300,
	}
	var err error
	if v := getenv("JUDGE_PARALLELISM"); v != "" {
		if s.Parallelism, err = strconv.Atoi(v); err != nil || s.Parallelism < 1 || s.Parallelism > 64 {
			return Settings{}, fmt.Errorf("JUDGE_PARALLELISM=%q không hợp lệ (số nguyên 1…64)", v)
		}
	}
	for name, dst := range map[string]*time.Duration{"JUDGE_LEASE": &s.Lease, "JUDGE_LEASE_RENEW": &s.LeaseRenew, "JUDGE_CLAIM_IDLE": &s.ClaimIdle, "JUDGE_HTTP_SLACK": &s.HTTPSlack} {
		if v := getenv(name); v != "" {
			d, perr := time.ParseDuration(v)
			if perr != nil || d <= 0 {
				return Settings{}, fmt.Errorf("%s=%q không hợp lệ (ví dụ 30s)", name, v)
			}
			*dst = d
		}
	}
	if v := getenv("JUDGE_MAX_TOTAL_SECONDS"); v != "" {
		if s.MaxTotalSeconds, err = strconv.Atoi(v); err != nil || s.MaxTotalSeconds < 20 {
			return Settings{}, fmt.Errorf("JUDGE_MAX_TOTAL_SECONDS=%q không hợp lệ (≥ 20)", v)
		}
	}
	if s.Lease < s.LeaseRenew*2 {
		return Settings{}, fmt.Errorf("JUDGE_LEASE (%s) phải ≥ 2 × JUDGE_LEASE_RENEW (%s)", s.Lease, s.LeaseRenew)
	}
	if s.Consumer {
		if len(s.Token) < MinTokenLen {
			return Settings{}, ErrTokenRequired
		}
		if err := CheckMemLimit(s.Parallelism, s.MemLimit); err != nil {
			return Settings{}, err
		}
	}
	return s, nil
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
