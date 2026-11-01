package submission

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Lý do không khớp (cột `submissions.unmatched_reason`).
const (
	ReasonNoAssignmentCode    = "NO_ASSIGNMENT_CODE"
	ReasonAmbiguousAssignment = "AMBIGUOUS_ASSIGNMENT"
	ReasonNoStudentSignal     = "NO_STUDENT_SIGNAL"
	ReasonEmailCodeConflict   = "EMAIL_CODE_CONFLICT"
	ReasonAmbiguousCode       = "AMBIGUOUS_STUDENT_CODE"
	ReasonStudentNotInCourse  = "STUDENT_NOT_IN_COURSE"
	ReasonCodeNotVerified     = "CODE_NOT_VERIFIED"
	ReasonSenderUnverified    = "SENDER_UNVERIFIED"
	ReasonAlreadyReviewed     = "ALREADY_REVIEWED"
)

// Phương thức khớp (cột `match_method`).
const (
	MethodEmail  = "EMAIL"
	MethodCode   = "CODE"
	MethodManual = "MANUAL"
	MethodInApp  = "IN_APP"
)

// Signals là tập tín hiệu của một bài nộp từ nguồn ngoài.
type Signals struct {
	FromEmail      string
	Subject        string
	FileName       string
	SenderVerified bool       // chỉ ImapSource đặt false khi thư chưa được máy chủ thư xác thực (4.5.4); nguồn khác true
	AssignmentID   *uuid.UUID // nguồn gọi theo bài (upload / Teams): bỏ bước chọn bài
}

// RosterEntry là một sinh viên ACTIVE của lớp. Code = `student_code_snapshot`; chỉ dòng ROSTER / ADMIN là tín hiệu MSSV (dòng CODE là MSSV tự khai — TLR-1).
type RosterEntry struct {
	StudentID     uuid.UUID
	Email         string
	EmailVerified bool
	Code          string
	JoinedVia     string
}

// AssignmentRef là một bài đã công bố của lớp: mã nộp thư để tìm trong tiêu đề / tên tệp.
type AssignmentRef struct {
	ID   uuid.UUID
	Code string
}

// Verdict là kết quả khớp. Reason rỗng = MATCHED (đủ AssignmentID + StudentID + Method). Candidate: gợi ý cho giảng viên khi UNMATCHED.
type Verdict struct {
	AssignmentID *uuid.UUID
	StudentID    *uuid.UUID
	Method       string
	Reason       string
	Candidate    *uuid.UUID
}

// Matched báo đã khớp đủ bài và sinh viên.
func (v Verdict) Matched() bool { return v.Reason == "" && v.AssignmentID != nil && v.StudentID != nil }

var (
	tokenRe  = regexp.MustCompile(`[A-Za-z0-9]+`)
	codeLike = regexp.MustCompile(`^[A-Z0-9]{6,15}$`)
	digitsRe = regexp.MustCompile(`[0-9]`)
	tailZero = regexp.MustCompile(`\.0+$`)
)

// normCode: TrimSpace, chữ hoa, bỏ đuôi `.0` (Excel đọc MSSV thành số).
func normCode(s string) string {
	return strings.ToUpper(tailZero.ReplaceAllString(strings.TrimSpace(s), ""))
}

func tokens(s string) []string { return tokenRe.FindAllString(s, -1) }

// looksLikeMSSV: token `[A-Z0-9]{6,15}` có ≥ 4 chữ số (loại từ thường như "Nguyen", "Document").
func looksLikeMSSV(t string) bool {
	return codeLike.MatchString(t) && len(digitsRe.FindAllString(t, -1)) >= 4
}

// pickAssignment tìm bài theo mã trong Subject rồi FileName (không phân biệt hoa thường).
func pickAssignment(sig Signals, asgs []AssignmentRef) (*uuid.UUID, string) {
	if sig.AssignmentID != nil {
		return sig.AssignmentID, ""
	}
	for _, text := range []string{sig.Subject, sig.FileName} {
		found := map[uuid.UUID]bool{}
		for _, t := range tokens(text) {
			for _, a := range asgs {
				if strings.EqualFold(t, a.Code) {
					found[a.ID] = true
				}
			}
		}
		switch len(found) {
		case 0:
			continue
		case 1:
			for id := range found {
				return &id, ""
			}
		default:
			return nil, ReasonAmbiguousAssignment
		}
	}
	return nil, ReasonNoAssignmentCode
}

// Match là bộ khớp (SRS 4.4.2) — hàm thuần, theo thứ tự: (0) thư chưa xác thực; (1) bài; (2) email đã xác minh; (3) MSSV theo danh sách; (4) kết luận.
// KHÔNG bao giờ dùng MSSV tự khai (cột MSSV khai ở hồ sơ người dùng, snapshot của dòng `joined_via='CODE'`) làm tín hiệu khớp.
func Match(sig Signals, roster []RosterEntry, asgs []AssignmentRef) Verdict {
	if !sig.SenderVerified {
		return Verdict{Reason: ReasonSenderUnverified}
	}
	asg, asgReason := pickAssignment(sig, asgs)

	// (2) email
	var byEmail *uuid.UUID
	if from := strings.ToLower(strings.TrimSpace(sig.FromEmail)); from != "" {
		for _, r := range roster {
			if r.EmailVerified && r.Email == from {
				id := r.StudentID
				byEmail = &id
				break
			}
		}
	}
	// (3) MSSV
	var byCode *uuid.UUID
	var codeReason string
	selfDeclared, lookalike := false, false
	rosterHit := map[uuid.UUID]bool{}
	for _, t := range append(tokens(sig.Subject), tokens(sig.FileName)...) {
		t = normCode(t)
		if !codeLike.MatchString(t) {
			continue
		}
		matchedAny := false
		for _, r := range roster {
			if r.Code == "" || normCode(r.Code) != t {
				continue
			}
			matchedAny = true
			if r.JoinedVia == "ROSTER" || r.JoinedVia == "ADMIN" {
				rosterHit[r.StudentID] = true
			} else {
				selfDeclared = true
			}
		}
		if !matchedAny && looksLikeMSSV(t) {
			lookalike = true
		}
	}
	switch {
	case len(rosterHit) == 1:
		for id := range rosterHit {
			byCode = &id
		}
	case len(rosterHit) >= 2:
		codeReason = ReasonAmbiguousCode
	case selfDeclared:
		codeReason = ReasonCodeNotVerified
	case lookalike:
		codeReason = ReasonStudentNotInCourse
	}

	// (4) kết luận
	v := Verdict{AssignmentID: asg}
	switch {
	case byEmail != nil && byCode != nil && *byEmail != *byCode:
		v.Reason, v.Candidate = ReasonEmailCodeConflict, byEmail
	case byEmail != nil:
		v.StudentID, v.Method = byEmail, MethodEmail
	case byCode != nil:
		v.StudentID, v.Method = byCode, MethodCode
	case codeReason != "":
		v.Reason = codeReason
	default:
		v.Reason = ReasonNoStudentSignal
	}
	if asgReason != "" { // bài không rõ: không gán; vẫn giữ gợi ý sinh viên
		v.Candidate, v.Reason, v.AssignmentID = v.StudentID, asgReason, nil
		if v.Candidate == nil {
			v.Candidate = byEmail
		}
		v.StudentID, v.Method = nil, ""
	}
	return v
}
