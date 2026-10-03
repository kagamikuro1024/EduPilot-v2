package course

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/mail"
	"github.com/edupilot/backend-go/internal/store"
)

const inviteTemplate = "invite_student"

// ErrRosterRace: email vừa được tạo bởi yêu cầu khác giữa lúc phân loại và ghi; gửi lại là xong (409).
var ErrRosterRace = errors.New("course: email vừa được tạo bởi yêu cầu khác")

var rosterCodeRE = regexp.MustCompile(`^[A-Z0-9]{6,15}$`)

// RowError là lỗi của MỘT dòng roster. Row = số dòng trong tệp (tiêu đề = 1).
type RowError struct {
	Row                  int
	Field, Code, Message string
}

// RosterReport là kết quả nhập. Total = CreatedUsers + LinkedExisting + AlreadyMember + PendingUnverified + SkippedRemoved + len(Errors).
type RosterReport struct {
	Total, CreatedUsers, LinkedExisting, AlreadyMember, PendingUnverified, SkippedRemoved int
	Errors                                                                                []RowError
}

// RosterOptions: DryRun chạy đủ phân loại rồi hoàn tác (không ghi gì); SendInvites xếp thư `invite_student`.
type RosterOptions struct{ DryRun, SendInvites bool }

// ImportRoster nạp danh sách lớp. Nối CHỈ bằng email (SRS 4.5): MSSV là thông tin khai báo, không bao giờ là khoá nối.
// Giảng viên của lớp đã được CourseAccessGuard(Teacher) kiểm; ở đây chỉ kiểm lớp còn mở. Dòng hợp lệ ghi trong MỘT transaction;
// dòng lỗi chỉ được báo. teacherID là người nhập (tên vào thư mời).
func (s Service) ImportRoster(ctx context.Context, courseID, teacherID uuid.UUID, data []byte, opt RosterOptions) (RosterReport, error) {
	rows, err := ParseRoster(data)
	if err != nil {
		return RosterReport{}, err
	}
	rep := RosterReport{Total: len(rows), Errors: []RowError{}}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return rep, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := store.New(tx)
	c, err := tq.LockCourse(ctx, courseID) // tuần tự hoá ghi danh cùng lớp: sĩ số đúng, import đồng thời không đua nhau
	if errors.Is(err, pgx.ErrNoRows) {
		return rep, ErrNotFound
	}
	if err != nil {
		return rep, fmt.Errorf("course: khoá lớp: %w", err)
	}
	if c.Status != store.CourseStatusACTIVE {
		return rep, ErrArchived
	}
	teacher, err := tq.GetUser(ctx, teacherID)
	if err != nil {
		return rep, fmt.Errorf("course: đọc giảng viên: %w", err)
	}
	counts, err := tq.CountCourseStudents(ctx, courseID)
	if err != nil {
		return rep, fmt.Errorf("course: đếm sinh viên: %w", err)
	}
	active := int(counts.Active)
	at := s.now()

	seenEmail, seenCode := map[string]bool{}, map[string]bool{}
	for _, row := range rows {
		email := auth.NormalizeEmail(row.Email)
		name := truncateRunes(row.Name, 100)
		code := strings.ToUpper(row.Code)
		fail := func(field, code, msg string) {
			rep.Errors = append(rep.Errors, RowError{Row: row.Line, Field: field, Code: code, Message: msg})
		}

		// 1. Dạng dữ liệu. Mỗi dòng báo MỘT lỗi đầu tiên (email → tên → MSSV): đủ để sửa theo thứ tự cột.
		switch {
		case auth.ValidateEmail(email) != nil:
			fail("email", "INVALID_EMAIL", "Email không đúng dạng.")
			continue
		case name == "" || auth.ValidateName(name) != nil:
			fail("full_name", "MISSING_NAME", "Thiếu họ và tên.")
			continue
		case !rosterCodeRE.MatchString(code):
			fail("student_code", "INVALID_STUDENT_CODE", "MSSV gồm 6–15 chữ và số.")
			continue
		case seenEmail[email]:
			fail("email", "DUPLICATE_EMAIL_IN_FILE", "Email này đã có ở dòng phía trên.")
			continue
		case seenCode[code]:
			fail("student_code", "DUPLICATE_STUDENT_CODE_IN_FILE", "MSSV này đã có ở dòng phía trên.")
			continue
		}
		seenEmail[email], seenCode[code] = true, true

		// 2. Tài khoản theo EMAIL (không bao giờ theo MSSV).
		u, err := tq.GetUserByEmail(ctx, email)
		exists := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return rep, fmt.Errorf("course: tra tài khoản: %w", err)
		}
		uid := uuid.Nil
		if exists {
			uid = u.ID
			switch {
			case u.Role != store.UserRoleSTUDENT:
				fail("email", "EMAIL_BELONGS_TO_STAFF", "Email này là của giảng viên hoặc trợ giảng.")
				continue
			case u.Status == store.UserStatusDISABLED:
				fail("email", "EMAIL_DISABLED", "Tài khoản này đã bị vô hiệu hoá.")
				continue
			}
			// 3. Ghi danh sẵn có (ACTIVE / PENDING / REMOVED) ⇒ không đổi.
			en, err := tq.LockEnrollment(ctx, store.LockEnrollmentParams{CourseID: courseID, UserID: u.ID})
			if err == nil {
				if en.Status == store.EnrollmentStatusREMOVED {
					rep.SkippedRemoved++
				} else {
					rep.AlreadyMember++
				}
				continue
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return rep, fmt.Errorf("course: đọc ghi danh: %w", err)
			}
		}

		// 4. MSSV đã là ảnh chụp của NGƯỜI KHÁC đang ở / chờ trong lớp ⇒ lỗi dòng (người đến sau bị chặn), không nối, không ghi.
		held, err := tq.EnrollmentConflictByStudentCode(ctx, store.EnrollmentConflictByStudentCodeParams{CourseID: courseID, Code: &code, UserID: uid})
		if err != nil {
			return rep, fmt.Errorf("course: kiểm MSSV: %w", err)
		}
		if held {
			fail("student_code", "STUDENT_CODE_CONFLICT", "MSSV này đã thuộc một sinh viên khác trong lớp.")
			continue
		}

		// 5. Chưa xác minh email ⇒ PENDING (tự ACTIVE khi xác minh). Còn lại ACTIVE nếu còn chỗ (đếm ACTIVE như vào bằng mã).
		status, warning := store.EnrollmentStatusACTIVE, (*string)(nil)
		if exists && u.Status != store.UserStatusINVITED && u.EmailVerifiedAt == nil {
			w := "EMAIL_UNVERIFIED"
			status, warning = store.EnrollmentStatusPENDING, &w
		}
		if status == store.EnrollmentStatusACTIVE && c.Capacity != nil && active >= int(*c.Capacity) {
			fail("email", "COURSE_FULL", "Lớp đã đủ sĩ số.")
			continue
		}
		created := false
		if !exists {
			id, err := tq.InsertRosterStudent(ctx, store.InsertRosterStudentParams{Email: email, FullName: name, StudentCode: &code})
			if errors.Is(err, pgx.ErrNoRows) {
				return rep, ErrRosterRace // người khác vừa tạo cùng email: phân loại lại từ đầu bằng cách gửi lại
			}
			if err != nil {
				return rep, fmt.Errorf("course: tạo tài khoản: %w", err)
			}
			if u, err = tq.GetUser(ctx, id); err != nil {
				return rep, fmt.Errorf("course: đọc tài khoản vừa tạo: %w", err)
			}
			created = true
		}

		en, err := tq.InsertRosterEnrollment(ctx, store.InsertRosterEnrollmentParams{CourseID: courseID, UserID: u.ID, Status: status, Snapshot: &code, Warning: warning, At: at, Actor: &teacherID})
		if err != nil {
			return rep, fmt.Errorf("course: ghi danh: %w", err)
		}
		if status == store.EnrollmentStatusACTIVE {
			active++
		}
		switch {
		case created:
			rep.CreatedUsers++
		case warning != nil:
			rep.PendingUnverified++
		default:
			rep.LinkedExisting++
		}
		if err := s.auditEnrollment(ctx, tq, courseID, &teacherID, en, "member_imported", nil, map[string]any{"status": status, "warning": warning, "via": "ROSTER"}); err != nil {
			return rep, err
		}
		// Thư mời: chỉ tài khoản INVITED (vừa tạo, hoặc tạo từ lớp khác) — người đã xác minh / đang chờ xác minh đã có đường vào riêng.
		if opt.SendInvites && u.Status == store.UserStatusINVITED {
			if _, _, err := mail.Enqueue(ctx, tx, mail.Message{
				To: email, Template: inviteTemplate, DedupeKey: fmt.Sprintf("invite_student:%s:%s", u.ID, courseID),
				Payload: map[string]any{"user_id": u.ID.String(), "full_name": u.FullName, "teacher_name": teacher.FullName, "course_name": c.Name, "class_code": c.ClassCode},
			}); err != nil {
				return rep, fmt.Errorf("course: xếp thư mời: %w", err)
			}
		}
	}

	if opt.DryRun {
		return rep, nil // tx hoàn tác ở defer: không users, enrollments, mail_outbox, outbox, notifications, audit_log
	}
	if _, err := tq.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &teacherID, Entity: "course", EntityID: courseID.String(), Action: "roster_imported",
		After: []byte(fmt.Sprintf(`{"total":%d,"created":%d,"linked":%d,"already":%d,"pending":%d,"skipped":%d,"errors":%d}`,
			rep.Total, rep.CreatedUsers, rep.LinkedExisting, rep.AlreadyMember, rep.PendingUnverified, rep.SkippedRemoved, len(rep.Errors)))}); err != nil {
		return rep, fmt.Errorf("course: ghi audit_log: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return rep, fmt.Errorf("course: commit roster: %w", err)
	}
	return rep, nil
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
