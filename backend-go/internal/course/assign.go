package course

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Changed là phần đã đổi của một lần gán.
type Changed struct {
	Teacher   bool
	AddedTA   []uuid.UUID
	RemovedTA []uuid.UUID
}

// AssignResult là kết quả `assign` / `assistants`.
type AssignResult struct {
	Teacher    *Person
	Assistants []Person
	Changed    Changed
}

// Assign: `teacher_id` vắng = giữ giảng viên; `ta_ids` vắng hoặc null = giữ trợ giảng, `[]` = gỡ hết, còn lại THAY TOÀN BỘ (SRS 4.4).
func (s Service) Assign(ctx context.Context, actor, courseID uuid.UUID, teacher *uuid.UUID, tas *[]uuid.UUID) (AssignResult, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return AssignResult{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	c, err := store.New(tx).LockCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AssignResult{}, ErrNotFound
	}
	if err != nil {
		return AssignResult{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	res, err := s.assignTx(ctx, tx, c, actor, teacher, tas)
	if err != nil {
		return AssignResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AssignResult{}, fmt.Errorf("course: commit gán lớp: %w", err)
	}
	return res, nil
}

// assignTx làm phần việc trong transaction của người gọi (lớp ĐÃ được khoá hoặc vừa tạo).
func (s Service) assignTx(ctx context.Context, tx pgx.Tx, c store.Course, actor uuid.UUID, teacher *uuid.UUID, tas *[]uuid.UUID) (AssignResult, error) {
	if c.Status == store.CourseStatusARCHIVED {
		return AssignResult{}, ErrArchived
	}
	q := store.New(tx)

	var want []uuid.UUID
	if tas != nil {
		want = dedupe(*tas)
	}
	ids := append([]uuid.UUID{}, want...)
	if teacher != nil {
		ids = append(ids, *teacher)
	}
	if err := s.checkPeople(ctx, q, teacher, want, ids); err != nil {
		return AssignResult{}, err
	}

	staff, err := q.ListStaffEnrollments(ctx, c.ID)
	if err != nil {
		return AssignResult{}, fmt.Errorf("course: đọc đội ngũ: %w", err)
	}
	var curTeacher *uuid.UUID
	curTA := map[uuid.UUID]bool{}
	for _, r := range staff {
		if r.RoleInCourse == store.EnrollmentRoleTEACHER {
			id := r.UserID
			curTeacher = &id
		} else {
			curTA[r.UserID] = true
		}
	}
	before := map[string]any{"teacher": curTeacher, "tas": sortedIDs(curTA)}

	var ch Changed
	assign := func(uid uuid.UUID, role store.EnrollmentRole) error {
		if _, err := q.ActivateStaffEnrollment(ctx, store.ActivateStaffEnrollmentParams{CourseID: c.ID, UserID: uid, RoleInCourse: role, Actor: &actor}); err != nil {
			return fmt.Errorf("course: gán người: %w", err)
		}
		_, err := outbox.Write(ctx, tx, TopicAssigned, map[string]string{"course_id": c.ID.String(), "user_id": uid.String(), "role": string(role), "assigned_by": actor.String()})
		return err
	}
	remove := func(uid uuid.UUID) error {
		return q.RemoveStaffEnrollment(ctx, store.RemoveStaffEnrollmentParams{CourseID: c.ID, UserID: uid, Actor: &actor})
	}

	// Giảng viên: gỡ cũ TRƯỚC khi gán mới (chỉ mục duy nhất một giảng viên ACTIVE).
	if teacher != nil && (curTeacher == nil || *curTeacher != *teacher) {
		if curTeacher != nil {
			if err := remove(*curTeacher); err != nil {
				return AssignResult{}, fmt.Errorf("course: gỡ giảng viên cũ: %w", err)
			}
		}
		if err := assign(*teacher, store.EnrollmentRoleTEACHER); err != nil {
			return AssignResult{}, err
		}
		ch.Teacher = true
	}
	if tas != nil {
		wantSet := map[uuid.UUID]bool{}
		for _, id := range want {
			wantSet[id] = true
		}
		for _, id := range sortedIDs(curTA) {
			if !wantSet[id] {
				if err := remove(id); err != nil {
					return AssignResult{}, fmt.Errorf("course: gỡ trợ giảng: %w", err)
				}
				ch.RemovedTA = append(ch.RemovedTA, id)
			}
		}
		for _, id := range want {
			if !curTA[id] {
				if err := assign(id, store.EnrollmentRoleTA); err != nil {
					return AssignResult{}, err
				}
				ch.AddedTA = append(ch.AddedTA, id)
			}
		}
	}

	if ch.Teacher || len(ch.AddedTA) > 0 || len(ch.RemovedTA) > 0 {
		after, err := q.ListStaffEnrollments(ctx, c.ID)
		if err != nil {
			return AssignResult{}, fmt.Errorf("course: đọc đội ngũ sau gán: %w", err)
		}
		var at *uuid.UUID
		taAfter := map[uuid.UUID]bool{}
		for _, r := range after {
			if r.RoleInCourse == store.EnrollmentRoleTEACHER {
				id := r.UserID
				at = &id
			} else {
				taAfter[r.UserID] = true
			}
		}
		if err := s.audit(ctx, q, c.ID, actor, c.ID.String(), "course_assigned", before, map[string]any{"teacher": at, "tas": sortedIDs(taAfter)}); err != nil {
			return AssignResult{}, err
		}
	}

	staffNow, err := q.ListStaffEnrollments(ctx, c.ID)
	if err != nil {
		return AssignResult{}, fmt.Errorf("course: đọc đội ngũ: %w", err)
	}
	res := AssignResult{Changed: ch, Assistants: []Person{}}
	for _, r := range staffNow {
		p := Person{ID: r.UserID, FullName: r.FullName}
		if r.RoleInCourse == store.EnrollmentRoleTEACHER {
			res.Teacher = &p
		} else {
			res.Assistants = append(res.Assistants, p)
		}
	}
	return res, nil
}

// checkPeople: teacher_id phải là TEACHER, mỗi ta_ids[i] phải là TA; ACTIVE hoặc INVITED (DISABLED / không tồn tại / sai vai ⇒ 422).
func (s Service) checkPeople(ctx context.Context, q *store.Queries, teacher *uuid.UUID, tas, all []uuid.UUID) error {
	if len(all) == 0 {
		return nil
	}
	txt := make([]string, len(all))
	for i, id := range all {
		txt[i] = id.String()
	}
	rows, err := q.GetUsersForAssign(ctx, txt)
	if err != nil {
		return fmt.Errorf("course: đọc người được gán: %w", err)
	}
	byID := map[uuid.UUID]store.GetUsersForAssignRow{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	ok := func(id uuid.UUID, role store.UserRole) bool {
		r, found := byID[id]
		return found && r.Role == role && (r.Status == store.UserStatusACTIVE || r.Status == store.UserStatusINVITED)
	}
	if teacher != nil && !ok(*teacher, store.UserRoleTEACHER) {
		return invalid("teacher_id", "ROLE_MISMATCH", "Người này không phải giảng viên.")
	}
	for _, id := range tas {
		if !ok(id, store.UserRoleTA) {
			return invalid("ta_ids", "ROLE_MISMATCH", "Người này không phải trợ giảng.")
		}
	}
	return nil
}

func dedupe(in []uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]bool{}
	out := make([]uuid.UUID, 0, len(in))
	for _, id := range in {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func sortedIDs(m map[uuid.UUID]bool) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	return out
}
