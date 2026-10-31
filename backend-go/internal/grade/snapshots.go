package grade

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

type (
	// SnapshotVersion là một phiên bản đã chốt (Staff xem lại: `Lịch sử chốt`).
	SnapshotVersion struct {
		Version      int       `json:"version"`
		Students     int       `json:"students"`
		LockedAt     time.Time `json:"locked_at"`
		LockedBy     string    `json:"locked_by"`
		UnlockReason *string   `json:"unlock_reason"`
	}
	// SnapshotRow là dòng của một bản chốt.
	SnapshotRow struct {
		Student      StudentView       `json:"student"`
		NumericGrade string            `json:"numeric_grade"`
		Letter       string            `json:"letter"`
		FailedReason *string           `json:"failed_reason"`
		QT           string            `json:"qt"`
		CK           string            `json:"ck"`
		Bonus        string            `json:"bonus"`
		Penalty      string            `json:"penalty"`
		Items        map[string]string `json:"items"`
	}
	SnapshotPage struct {
		Version    int           `json:"version"`
		LockedAt   time.Time     `json:"locked_at"`
		Items      []SnapshotRow `json:"items"`
		NextCursor *string       `json:"next_cursor"`
	}
)

// SnapshotVersions liệt kê các bản chốt của lớp, mới nhất trước.
func (s *Service) SnapshotVersions(ctx context.Context, courseID uuid.UUID) ([]SnapshotVersion, error) {
	rows, err := store.New(s.Pool).GradeSnapshotVersions(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("grade: các bản chốt: %w", err)
	}
	out := make([]SnapshotVersion, len(rows))
	for i, r := range rows {
		out[i] = SnapshotVersion{Version: int(r.Version), Students: int(r.Students), LockedAt: r.LockedAt, LockedBy: r.LockedBy}
		if r.UnlockReason != "" {
			reason := r.UnlockReason
			out[i].UnlockReason = &reason
		}
	}
	return out, nil
}

func breakdownOf(raw []byte) SnapBreakdown {
	var b SnapBreakdown
	_ = json.Unmarshal(raw, &b)
	if b.Items == nil {
		b.Items = map[string]string{}
	}
	return b
}

// Snapshot đọc một trang của bản chốt `version` (con trỏ như `Preview`). Bản không tồn tại → 404.
func (s *Service) Snapshot(ctx context.Context, courseID uuid.UUID, version int, cursor string, limit int) (SnapshotPage, error) {
	var cur pageCursor
	var err error
	if cursor != "" {
		if cur, err = decodePage(cursor); err != nil {
			return SnapshotPage{}, err
		}
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}
	rows, err := store.New(s.Pool).GradeSnapshotPage(ctx, store.GradeSnapshotPageParams{CourseID: courseID, Version: int32(version), AfterFold: cur.Fold, AfterID: cur.ID, PageLimit: int32(limit + 1)}) //nolint:gosec // phiên bản nhỏ, ≤ 101
	if err != nil {
		return SnapshotPage{}, fmt.Errorf("grade: đọc bản chốt: %w", err)
	}
	if len(rows) == 0 && cursor == "" {
		return SnapshotPage{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	out := SnapshotPage{Version: version, Items: []SnapshotRow{}}
	if len(rows) > limit {
		rows = rows[:limit]
		c := encodePage(rows[len(rows)-1].FoldName, rows[len(rows)-1].StudentID)
		out.NextCursor = &c
	}
	for _, r := range rows {
		b := breakdownOf(r.Breakdown)
		out.LockedAt = r.LockedAt
		out.Items = append(out.Items, SnapshotRow{Student: StudentView{ID: r.StudentID, FullName: r.FullName, StudentCode: derefStr(r.StudentCode)},
			NumericGrade: r.NumericGrade.String(), Letter: r.Letter, FailedReason: r.FailedReason, QT: b.QT, CK: b.CK, Bonus: b.Bonus, Penalty: b.Penalty, Items: b.Items})
	}
	return out, nil
}

// snapshotResult là kết quả hiển thị của một hàng khi sổ LOCKED (từ bản chốt) và cờ `drift` so với `Compute` sống.
func snapshotResult(sn store.GradeSnapshotsForStudentsRow, live Result) (res *ResultView, bonus, penalty string, drift bool) {
	b := breakdownOf(sn.Breakdown)
	res = &ResultView{Status: string(StatusComplete), QT: b.QT, HP: sn.NumericGrade.String(), Letter: sn.Letter, Verdict: "OK", Missing: []string{}}
	if sn.FailedReason != nil {
		res.Verdict = *sn.FailedReason
	}
	bonus, penalty = b.Bonus, b.Penalty
	if bonus == "" {
		bonus = "0"
	}
	if penalty == "" {
		penalty = "0"
	}
	drift = live.Status != StatusComplete || !live.HP.Equal(sn.NumericGrade) || live.Letter != sn.Letter
	return res, bonus, penalty, drift
}
