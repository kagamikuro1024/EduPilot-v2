package auth

import "strings"

// deviceLabel suy ra nhãn thiết bị ngắn từ User-Agent, ví dụ "Chrome trên macOS". Không phân tích sâu.
// ponytail: so khớp chuỗi con theo thứ tự ưu tiên; thêm thư viện phân tích UA khi cần độ chính xác cao hơn.
func deviceLabel(ua string) string {
	l := strings.ToLower(ua)
	if l == "" {
		return "Trình duyệt khác"
	}
	browser := "Trình duyệt khác"
	switch {
	case strings.Contains(l, "edg/") || strings.Contains(l, "edge/"):
		browser = "Edge"
	case strings.Contains(l, "firefox/"):
		browser = "Firefox"
	case strings.Contains(l, "chrome/") || strings.Contains(l, "crios/"):
		browser = "Chrome"
	case strings.Contains(l, "safari/"):
		browser = "Safari"
	}
	osName := ""
	switch {
	case strings.Contains(l, "android"):
		osName = "Android"
	case strings.Contains(l, "iphone") || strings.Contains(l, "ipad") || strings.Contains(l, "ios"):
		osName = "iOS"
	case strings.Contains(l, "windows"):
		osName = "Windows"
	case strings.Contains(l, "mac os x") || strings.Contains(l, "macintosh"):
		osName = "macOS"
	case strings.Contains(l, "linux"):
		osName = "Linux"
	}
	if osName == "" {
		return browser
	}
	return browser + " trên " + osName
}
