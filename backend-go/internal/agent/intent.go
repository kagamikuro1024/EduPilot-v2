package agent

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/edupilot/backend-go/internal/auth"
)

// Luật theo thứ tự ưu tiên của SRS 4.5; khớp trên văn bản đã bỏ dấu + chữ thường nên phủ cả có dấu lẫn không dấu.
var (
	reCrisis = regexp.MustCompile(`\b(tu tu|tu sat|tu hai|tu lam dau|khong muon song|muon chet|ket thuc cuoc song|ket lieu|chan doi muon chet|khong muon ton tai)\b`)
	// Giả định (neu, gia su, gia du, gia dinh, gia nhu, thu) + thành phần điểm + được/đạt/có + số 0–10 (QUESTIONS Q-QC-P3-04-7); "thì" tuỳ chọn.
	reWhatIf   = regexp.MustCompile(`\b(neu|gia su|gia du|gia dinh|gia nhu|thu)\b.{0,40}\b(duoc|dat|co)\b.{0,20}\b(10|[0-9])([.,][0-9]+)?\b`)
	reComp     = regexp.MustCompile(`\b(diem|giua ky|cuoi ky|qua trinh|bai tap|thuc hanh|chuyen can|tong ket|ket qua)\b`)
	reFormula  = regexp.MustCompile(`\b(cach tinh diem|trong so|cong thuc|ty le diem|ti le diem|diem (tong ket |cuoi ky |qua trinh )?duoc tinh|tinh diem nhu the nao)\b`)
	reSelf     = regexp.MustCompile(`\b(em|minh|toi|tao)\b`)
	reAtt      = regexp.MustCompile(`\b(vang|nghi|diem danh|chuyen can)\b`)
	rePart     = regexp.MustCompile(`\b(diem cong|phat bieu|diem phat bieu)\b`)
	reGrade    = regexp.MustCompile(`\b(diem|ket qua|bang diem|xep loai|qua mon|truot mon|tong ket)\b`)
	reExam     = regexp.MustCompile(`\b(lich thi|lich kiem tra|(cho hoi |xem )(ngay|gio|phong) thi|(ngay|gio|phong) thi ((cuoi ky|giua ky|mon nay|lan 2|lan 1) )*((la|o|vao) )*(khi nao|bao gio|luc nao|ngay nao|o dau|phong nao|may gio|nao)|(khi nao|bao gio|luc nao|ngay nao) ((em|minh|toi|lop|se|co|duoc|phai|nhom) )*(thi|kiem tra)|(thi|kiem tra) ((cuoi ky|giua ky|mon nay|hoc ky|lan 2|lan 1|bai 1|bai 2|lai) )*(khi nao|bao gio|luc nao|ngay nao))\b`)
	reUpcoming = regexp.MustCompile(`\b(sap toi|tuan nay|tuan sau|han nop|lich hoc|ngay mai|hom nay co|co gi trong tuan|su kien)\b`)
	reLibrary  = regexp.MustCompile(`\b(tim tai lieu|tai lieu ve|tim slide|slide|giao trinh|tai lieu nao|co tai lieu|tim file|bai giang ve)\b`)
	reGreet    = regexp.MustCompile(`\b(chao|xin chao|hello|hi|cam on|cam on|thanks|thank you|ok|oke|vang|da|tam biet|bye)\b`)
	reQuestion = regexp.MustCompile(`\b(gi|sao|nao|dau|ai|bao nhieu|the nao|nhu the nao|tai sao|co phai|la)\b`)
	// Từ đệm: tin chỉ gồm từ xã giao + từ đệm mới là SMALLTALK (Q-QC-P3-04-8); thêm nội dung thì đi COURSE_QA.
	reFiller = regexp.MustCompile(`^(chao|xin|hello|hi|cam|on|thanks|thank|you|ok|oke|vang|da|tam|biet|bye|ban|thay|co|em|minh|nhe|nha|a|nhieu|qua|la|vay|roi|day|ha|nhi|rat|that|oi|cac|anh|chi)$`)
)

// PersonalIntent là ý định cá nhân (đọc dữ liệu của chính người hỏi) suy từ luật; "" nếu không có. Dùng cả để chọn tool lẫn để quyết "hỏi hộ".
func PersonalIntent(text string) Intent {
	f := auth.Fold(text)
	self := reSelf.MatchString(f)
	switch {
	case reWhatIf.MatchString(f) && reComp.MatchString(f):
		return IntentWhatIf
	case rePart.MatchString(f) && (self || reGrade.MatchString(f)):
		return IntentParticipation
	case reAtt.MatchString(f) && self:
		return IntentAttendance
	case reGrade.MatchString(f) && self:
		return IntentGrade
	}
	return ""
}

// DetectIntent phân loại ý định bằng luật. Thứ tự: CRISIS > WHAT_IF > GRADE_FORMULA > PERSONAL_* > EXAM_SCHEDULE > UPCOMING_EVENTS > LIBRARY_SEARCH > COURSE_QA > SMALLTALK.
// OTHER_PERSON không ở đây: cần danh tính (roster) nên do Analyzer quyết rồi đè lên.
func DetectIntent(text string) Intent {
	f := auth.Fold(text)
	switch {
	case reCrisis.MatchString(f):
		return IntentCrisis
	case reWhatIf.MatchString(f) && reComp.MatchString(f):
		return IntentWhatIf
	case reFormula.MatchString(f):
		return IntentGradeFormula
	}
	if pi := PersonalIntent(text); pi != "" {
		return pi
	}
	switch {
	case reExam.MatchString(f):
		return IntentExamSchedule
	case reUpcoming.MatchString(f):
		return IntentUpcoming
	case reLibrary.MatchString(f):
		return IntentLibrary
	case smalltalk(text, f):
		return IntentSmalltalk
	}
	return IntentCourseQA
}

// hasPersonalKeyword: câu nhắc tới thứ cá nhân (điểm, vắng, lịch thi, điểm cộng…) — điều kiện để "hỏi hộ" thành vấn đề.
func hasPersonalKeyword(text string) bool {
	f := auth.Fold(text)
	return reGrade.MatchString(f) || reComp.MatchString(f) || reAtt.MatchString(f) || rePart.MatchString(f) || reExam.MatchString(f)
}

var reDeclare = regexp.MustCompile(`(mssv|ma so sinh vien|ma sv|msv)( cua)? (em|minh|toi|tao)( la|:| )`)

// isSelfDeclaration: mã nằm ngay sau "MSSV của em là" — câu TỰ KHAI, không phải đối tượng của câu hỏi (US-P3-04 AC9).
func isSelfDeclaration(runeStart int, runes []rune) bool {
	// cắt đoạn ngay trước vị trí (≤ 40 rune) rồi tìm cụm tự khai ở cuối
	lo := max(0, runeStart-40)
	before := auth.Fold(string(runes[lo:runeStart]))
	loc := reDeclare.FindAllStringIndex(before, -1)
	if len(loc) == 0 {
		return false
	}
	tail := strings.TrimSpace(before[loc[len(loc)-1][1]:])
	return tail == "" || tail == ":" || tail == "la"
}

// smalltalk: (A) có từ xã giao và MỌI từ còn lại là từ đệm — không giới hạn độ dài; hoặc (B) tin ≤ 12 ký tự không khớp từ khoá intent nào (Q-QC-P3-04-8).
func smalltalk(text, folded string) bool {
	// (B) rào chắn của dev (ngoài chữ BA, ghi ở handoff): câu hỏi ngắn ("TCP là gì ạ") vẫn đi truy xuất, không rơi vào SMALLTALK
	if utf8.RuneCountInString(strings.TrimSpace(text)) <= 12 && !reQuestion.MatchString(folded) && !strings.Contains(text, "?") {
		return true // tới đây đã không khớp intent nào ở trên
	}
	if !reGreet.MatchString(folded) {
		return false
	}
	for _, w := range strings.FieldsFunc(folded, func(r rune) bool { return (r < 'a' || r > 'z') && (r < '0' || r > '9') }) {
		if !reFiller.MatchString(w) {
			return false
		}
	}
	return true
}
