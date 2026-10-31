package grade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// ConfirmResult là thân `POST …/grade-scheme/confirm` (200).
type ConfirmResult struct {
	SchemeID    uuid.UUID `json:"scheme_id"`
	SchemeNo    int       `json:"scheme_no"`
	ConfirmedAt time.Time `json:"confirmed_at"`
	Impact      Impact    `json:"impact"`
}

// ConfirmScheme xác nhận bản nháp (SRS 4.4.4) — CHỈ được gọi từ handler `POST …/grade-scheme/confirm` (TestConfirmOnlyFromHandler; luật: không đường nền nào
// tự xác nhận công thức). Một giao dịch, theo thứ tự: (1) hàng `gradebooks` + `FOR SHARE` (LOCKED → 409); (2) khoá bản nháp, so version, bộ kiểm (còn mục chưa rõ → 422);
// (3) bản CONFIRMED cũ → SUPERSEDED TRƯỚC, rồi DRAFT → CONFIRMED (chỉ mục duy nhất một phần không DEFERRABLE); (4) mang điểm nhập tay cùng key + nguồn + max_score;
// (5) audit + outbox `grade.scheme_confirmed`. Gửi lại cùng Idempotency-Key trả lại phản hồi cũ ở middleware.
func (s *Service) ConfirmScheme(ctx context.Context, actor, courseID uuid.UUID, version int) (ConfirmResult, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return ConfirmResult{}, fmt.Errorf("grade: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := store.New(tx)

	if err := q.GradebookEnsure(ctx, courseID); err != nil {
		return ConfirmResult{}, fmt.Errorf("grade: tạo sổ điểm: %w", err)
	}
	gb, err := q.GradebookStatusShare(ctx, courseID)
	if err != nil {
		return ConfirmResult{}, fmt.Errorf("grade: trạng thái sổ điểm: %w", err)
	}
	if gb == store.GradebookStatusLOCKED {
		return ConfirmResult{}, apierr.New(http.StatusConflict, apierr.GradebookLocked)
	}

	cur, err := q.GradeDraftLock(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConfirmResult{}, apierr.New(http.StatusConflict, apierr.NoDraft)
	}
	if err != nil {
		return ConfirmResult{}, fmt.Errorf("grade: khoá bản nháp: %w", err)
	}
	if int(cur.Version) != version {
		doc, err := schemeDocByID(ctx, q, courseID, cur.ID)
		if err != nil {
			return ConfirmResult{}, err
		}
		return ConfirmResult{}, versionConflict(doc)
	}
	draft, err := LoadScheme(ctx, q, courseID, cur.ID)
	if err != nil {
		return ConfirmResult{}, err
	}
	insp, err := InspectDrafts(ctx, q, []draftRef{{ID: cur.ID, CourseID: courseID}})
	if err != nil {
		return ConfirmResult{}, err
	}
	if un := insp[cur.ID].Unclear; len(un) > 0 {
		return ConfirmResult{}, apierr.New(http.StatusUnprocessableEntity, apierr.SchemeHasUnclear).WithDetails(map[string]any{"codes": codeList(un), "unclear_items": un})
	}

	imp, err := impactFor(ctx, q, courseID, parsedFromLoaded(draft))
	if err != nil {
		return ConfirmResult{}, err
	}
	var oldItems []Item
	if old, ok, err := LoadConfirmed(ctx, q, courseID); err != nil {
		return ConfirmResult{}, err
	} else if ok {
		for _, c := range old.Scheme.Components {
			oldItems = append(oldItems, c.Items...)
		}
	}

	if _, err := q.GradeSchemeSupersede(ctx, courseID); err != nil {
		return ConfirmResult{}, fmt.Errorf("grade: thay bản cũ: %w", err)
	}
	done, err := q.GradeSchemeConfirm(ctx, store.GradeSchemeConfirmParams{CourseID: courseID, ID: cur.ID, ActorID: &actor})
	if err != nil {
		return ConfirmResult{}, fmt.Errorf("grade: xác nhận: %w", err)
	}

	// Mang điểm nhập tay: cùng key, cùng nguồn nhập tay, cùng max_score (TLR-8). Item không mang vẫn nằm trong DB (ẩn khỏi bảng).
	oldBy := map[string]Item{}
	for _, it := range oldItems {
		if it.Source == SourceManual {
			oldBy[it.Key] = it
		}
	}
	moved := 0
	for _, c := range draft.Scheme.Components {
		for _, it := range c.Items {
			if o, ok := oldBy[it.Key]; ok && it.Source == SourceManual && o.Max.Equal(it.Max) {
				n, err := q.GradeCarryEntries(ctx, store.GradeCarryEntriesParams{CourseID: courseID, OldItemID: o.ID, NewItemID: it.ID})
				if err != nil {
					return ConfirmResult{}, fmt.Errorf("grade: mang điểm: %w", err)
				}
				moved += int(n)
				if err := q.GradeCarryPublished(ctx, store.GradeCarryPublishedParams{CourseID: courseID, OldItemID: o.ID, NewItemID: it.ID}); err != nil {
					return ConfirmResult{}, fmt.Errorf("grade: mang trạng thái công bố: %w", err)
				}
			}
		}
	}

	after, _ := json.Marshal(map[string]any{
		"scheme_no": done.SchemeNo, "version": version, "students_total": imp.StudentsTotal, "students_computable": imp.StudentsComputable,
		"changed_vs_confirmed": imp.ChangedVsConfirmed, "entries_carried": moved, "entries_dropped": imp.EntriesDropped,
	})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "grade_schemes", EntityID: cur.ID.String(), Action: "confirm", After: after}); err != nil {
		return ConfirmResult{}, fmt.Errorf("grade: audit: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, TopicSchemeConfirmed, map[string]string{"course_id": courseID.String(), "scheme_id": cur.ID.String()}); err != nil {
		return ConfirmResult{}, fmt.Errorf("grade: outbox %s: %w", TopicSchemeConfirmed, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ConfirmResult{}, fmt.Errorf("grade: xác nhận công thức: %w", err)
	}
	imp.EntriesCarried = moved
	at := s.now()
	if done.ConfirmedAt != nil {
		at = *done.ConfirmedAt
	}
	return ConfirmResult{SchemeID: cur.ID, SchemeNo: int(done.SchemeNo), ConfirmedAt: at, Impact: imp}, nil
}

func codeList(us []Unclear) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, u := range us {
		if u.Code != "" && !seen[u.Code] {
			seen[u.Code] = true
			out = append(out, u.Code)
		}
	}
	return out
}
