package mailin

import (
	"strings"
)

// SenderVerified áp TLR-2: chỉ tin `From:` khi header `Authentication-Results` TRÊN CÙNG do máy chủ thư của trường thêm (`authserv-id` = trusted, không rỗng) cho `dmarc=pass`
// với `header.from` cùng domain với `From`. Header nằm dưới (người gửi tự thêm), thiếu header, authserv-id lạ, dmarc ≠ pass, nhiều kết quả dmarc không khớp, cấu hình rỗng → false.
func SenderVerified(authResults []string, from, trusted string) bool {
	trusted = strings.ToLower(strings.TrimSpace(trusted))
	if trusted == "" || len(authResults) == 0 {
		return false
	}
	at := strings.LastIndex(from, "@")
	if at < 0 || at == len(from)-1 {
		return false
	}
	domain := strings.ToLower(from[at+1:])
	top := stripComments(authResults[0])
	segs := strings.Split(top, ";")
	id := strings.Fields(strings.ToLower(segs[0]))
	if len(id) == 0 || id[0] != trusted {
		return false
	}
	pass := 0
	for _, seg := range segs[1:] {
		f := strings.Fields(strings.ToLower(seg))
		if len(f) == 0 || !strings.HasPrefix(f[0], "dmarc=") {
			continue
		}
		if f[0] != "dmarc=pass" {
			return false // một kết quả dmarc không đạt khiến cả header không xác định
		}
		ok := false
		for _, kv := range f[1:] {
			if v, found := strings.CutPrefix(kv, "header.from="); found && strings.Trim(v, "\"") == domain {
				ok = true
			}
		}
		if !ok {
			return false
		}
		pass++
	}
	return pass == 1
}

func stripComments(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '(':
			depth++
		case r == ')' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}
