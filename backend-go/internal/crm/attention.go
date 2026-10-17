package crm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/today"
)

var _ today.AttentionSource = (*Service)(nil)

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// Snapshot cài today.AttentionSource: MỘT truy vấn cho số sinh viên mới vào AT_RISK mỗi lớp và ≤ limit sinh viên AT_RISK (mới vào trước) + tổng, mỗi người một cụm lý do ngắn.
func (s *Service) Snapshot(ctx context.Context, courseIDs []uuid.UUID, since time.Time, limit int) (today.AttentionSnapshot, error) {
	rows, err := store.New(s.Pool).CrmAttention(ctx, store.CrmAttentionParams{CourseIds: uuidStrings(courseIDs), Since: since, Lim: int32(limit)}) //nolint:gosec // 5
	if err != nil {
		return today.AttentionSnapshot{}, wrap("sinh viên AT_RISK", err)
	}
	out := today.AttentionSnapshot{NewByCourse: map[uuid.UUID]int{}}
	for _, r := range rows {
		if r.Kind == "N" {
			out.NewByCourse[r.CourseID] = int(r.N)
			continue
		}
		out.Total = int(r.Total)
		out.Rows = append(out.Rows, today.NewAttentionRow(r.StudentID, r.FullName, r.CourseID, r.ClassCode, shortReason(r.Reasons)))
	}
	return out, nil
}

// shortReason: cụm ngắn theo thứ tự chuyên cần, thời gian học, hoạt động ("vắng 3/4 buổi; học ít").
func shortReason(raw json.RawMessage) string {
	var rs []Reason
	if json.Unmarshal(raw, &rs) != nil {
		return ""
	}
	var parts []string
	for _, code := range []string{ReasonAttendanceLow, ReasonStudyLow, ReasonActivityLow} {
		for _, r := range rs {
			if r.Code != code {
				continue
			}
			switch code {
			case ReasonAttendanceLow:
				parts = append(parts, fmt.Sprintf("vắng %d/%d buổi", r.Values["absent"], r.Values["recorded"]))
			case ReasonStudyLow:
				parts = append(parts, "học ít")
			case ReasonActivityLow:
				parts = append(parts, "ít hoạt động")
			}
		}
	}
	return strings.Join(parts, "; ")
}
