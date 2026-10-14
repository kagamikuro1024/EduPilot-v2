package course

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/crm"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Ngưỡng chuyển cho giảng viên (SRS FEAT-escalation-notify 4.2.8): [0,50; 0,95], bước 0,05, so sánh bằng decimal.
var (
	thresholdMin  = decimal.New(50, -2) //nolint:gochecknoglobals // hằng số tra cứu bất biến
	thresholdMax  = decimal.New(95, -2) //nolint:gochecknoglobals
	thresholdStep = decimal.New(5, -2)  //nolint:gochecknoglobals
)

// ThresholdConflictError: sai version khi đặt ngưỡng; Current là bản hiện hành của lớp (để giao diện hoàn lại).
type ThresholdConflictError struct{ Current Detail }

func (e *ThresholdConflictError) Error() string { return "course: sai version cài đặt lớp" }

// ErrHasStudents: lớp đã có sinh viên ghi danh (mọi trạng thái) nên cờ `allow_study_time_opt_out` bị khoá (Q25) → 409 COURSE_HAS_STUDENTS.
var ErrHasStudents = errors.New("course: lớp đã có sinh viên")

// SettingsIn là thân `PUT /courses/{id}/settings`: trường nil = giữ nguyên; ít nhất một trường.
type SettingsIn struct {
	EscalationThreshold  *string
	AllowStudyTimeOptOut *bool
	Risk                 *crm.RiskIn // ngưỡng at-risk (US-P5-04): lưu vào `courses.settings.risk`, rồi outbox `course.settings_changed`
}

// PutSettings đặt `courses.escalation_threshold`, `courses.allow_study_time_opt_out` và / hoặc `courses.settings.risk` trong MỘT giao dịch (chỉ giảng viên của lớp; guard ở route).
// Ngưỡng: có hiệu lực ở tin kế tiếp (chat đọc mỗi lần), ngoài khoảng / sai bước → 422. Cờ đo thời gian học (Q25): `SELECT … FOR UPDATE`, rồi kiểm KHÔNG có ghi danh sinh viên nào (mọi trạng thái;
// có → ErrHasStudents, kể cả đổi cùng giá trị cũ) rồi `UPDATE`; ghi danh sinh viên đầu tiên lấy `FOR SHARE` cùng hàng nên hai việc tuần tự hoá. Sai version → ThresholdConflictError.
// Ghi audit_log (cũ / mới). `courses.version + 1`.
func (s Service) PutSettings(ctx context.Context, actor, courseID uuid.UUID, version int, in SettingsIn) (Detail, error) {
	var thr decimal.Decimal
	if in.EscalationThreshold != nil {
		v, err := decimal.NewFromString(*in.EscalationThreshold)
		if err != nil || v.LessThan(thresholdMin) || v.GreaterThan(thresholdMax) || !v.Mod(thresholdStep).IsZero() {
			return Detail{}, invalid("escalation_threshold", "range", "Ngưỡng từ 0,50 đến 0,95, mỗi bước 0,05.")
		}
		thr = v
	}
	if in.Risk != nil { // kiểm khoảng trước khi mở giao dịch: trộn lên settings rỗng chỉ để bắt trường sai
		if _, _, _, err := crm.MergeRisk(nil, *in.Risk); err != nil {
			return Detail{}, riskInvalid(err)
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	cur, err := q.LockCourse(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if cur.Status == store.CourseStatusARCHIVED {
		return Detail{}, ErrArchived
	}
	if int(cur.Version) != version {
		now, gerr := s.Get(ctx, courseID, "TEACHER", true)
		if gerr != nil {
			return Detail{}, gerr
		}
		return Detail{}, &ThresholdConflictError{Current: now}
	}
	before, after := map[string]any{"version": cur.Version}, map[string]any{}
	upd := cur
	if in.EscalationThreshold != nil {
		if upd, err = q.UpdateCourseThreshold(ctx, store.UpdateCourseThresholdParams{ID: courseID, Threshold: thr}); err != nil {
			return Detail{}, fmt.Errorf("course: đặt ngưỡng: %w", err)
		}
		before["escalation_threshold"], after["escalation_threshold"] = cur.EscalationThreshold.StringFixed(2), upd.EscalationThreshold.StringFixed(2)
	}
	if in.AllowStudyTimeOptOut != nil {
		has, err := q.CourseHasStudentEnrollment(ctx, courseID)
		if err != nil {
			return Detail{}, fmt.Errorf("course: kiểm sinh viên: %w", err)
		}
		if has {
			return Detail{}, ErrHasStudents
		}
		if upd, err = q.UpdateCourseOptOut(ctx, store.UpdateCourseOptOutParams{ID: courseID, Allow: *in.AllowStudyTimeOptOut}); err != nil {
			return Detail{}, fmt.Errorf("course: đặt cờ đo thời gian học: %w", err)
		}
		before["allow_study_time_opt_out"], after["allow_study_time_opt_out"] = cur.AllowStudyTimeOptOut, upd.AllowStudyTimeOptOut
	}
	if in.Risk != nil {
		merged, was, now, err := crm.MergeRisk(upd.Settings, *in.Risk)
		if err != nil {
			return Detail{}, riskInvalid(err)
		}
		if upd, err = q.UpdateCourseSettings(ctx, store.UpdateCourseSettingsParams{ID: courseID, Settings: merged}); err != nil {
			return Detail{}, fmt.Errorf("course: đặt ngưỡng cần chú ý: %w", err)
		}
		before["risk"], after["risk"] = was.Values(), now.Values()
		if _, err := outbox.Write(ctx, tx, crm.TopicSettingsChanged, map[string]string{"course_id": courseID.String()}); err != nil {
			return Detail{}, fmt.Errorf("course: outbox %s: %w", crm.TopicSettingsChanged, err)
		}
	}
	after["version"] = upd.Version
	action := "course_threshold_updated"
	switch {
	case in.EscalationThreshold != nil:
	case in.AllowStudyTimeOptOut != nil:
		action = "course_study_time_opt_out_updated"
	default:
		action = "course_risk_updated"
	}
	if err := s.audit(ctx, q, courseID, actor, courseID.String(), action, before, after); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("course: commit cài đặt: %w", err)
	}
	return s.Get(ctx, courseID, "TEACHER", true)
}

// riskInvalid đổi lỗi khoảng của crm.MergeRisk thành 422 theo trường (`risk.<tên>`); lỗi khác giữ nguyên.
func riskInvalid(err error) error {
	var re *crm.RiskError
	if errors.As(err, &re) {
		return invalid("risk."+re.Field, "range", re.Message)
	}
	return err
}
