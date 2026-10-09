// Package similarity so độ giống mã nguồn C / C++ bằng winnowing (US-PE-07 AC7–AC8; research mục 4): bộ tách từ vựng tự viết → chuẩn hoá → dấu vân tay
// (k = 5, w = 4, FNV-1a 64 bit) → Jaccard. Mã KHÔNG rời hệ thống: gói thuần tính toán, không gọi mạng, không dùng số thực (điểm là số nguyên ‰).
package similarity

import "strings"

// Token là một đơn vị từ vựng sau chuẩn hoá cùng dòng nguồn (để tô phần khớp).
type Token struct {
	Text string
	Line int // 1-based
}

func isKeyword(w string) bool {
	switch w {
	case "alignas", "alignof", "and", "and_eq", "asm", "auto", "bitand", "bitor",
		"bool", "break", "case", "catch", "char", "char16_t", "char32_t", "class",
		"compl", "const", "const_cast", "constexpr", "continue", "decltype", "default", "delete",
		"do", "double", "dynamic_cast", "else", "enum", "explicit", "export", "extern",
		"false", "float", "for", "friend", "goto", "if", "inline", "int",
		"long", "mutable", "namespace", "new", "noexcept", "not", "not_eq", "nullptr",
		"operator", "or", "or_eq", "private", "protected", "public", "register", "reinterpret_cast",
		"restrict", "return", "short", "signed", "sizeof", "static", "static_assert", "static_cast",
		"struct", "switch", "template", "this", "thread_local", "throw", "true", "try",
		"typedef", "typeid", "typename", "union", "unsigned", "using", "virtual", "void",
		"volatile", "wchar_t", "while", "xor", "xor_eq":
		return true
	}
	return false
}

func isOp3(s string) bool {
	switch s {
	case "<<=", ">>=", "...", "->*":
		return true
	}
	return false
}

func isOp2(s string) bool {
	switch s {
	case "->", "++", "--", "&&", "||", "==", "!=", "<=", ">=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<", ">>", "::", ".*":
		return true
	}
	return false
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isIdent(c byte) bool { return isIdentStart(c) || isDigit(c) }

// Normalize tách và chuẩn hoá một đoạn mã: bỏ chú thích và khoảng trắng; dòng `#include` bị bỏ; định danh không phải từ khoá → `I`; số → `N`; chuỗi / ký tự → `S`;
// từ khoá và toán tử giữ nguyên. Tất định; không bao giờ lỗi (đầu vào hỏng vẫn cho ra dãy token).
func Normalize(src string) []Token {
	var out []Token
	line := 1
	i, n := 0, len(src)
	atLineStart := true
	emit := func(t string) { out = append(out, Token{Text: t, Line: line}) }
	for i < n {
		c := src[i]
		switch {
		case c == '\n':
			line++
			atLineStart = true
			i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v':
			i++
		case c == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				if src[i] == '\\' && i+1 < n && src[i+1] == '\n' { // dòng chú thích nối dòng
					line++
					i++
				}
				i++
			}
		case c == '/' && i+1 < n && src[i+1] == '*':
			i += 2
			for i < n && (src[i] != '*' || i+1 >= n || src[i+1] != '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			i = min(n, i+2)
		case c == '#' && atLineStart && hasInclude(src, i):
			for i < n && src[i] != '\n' { // bỏ cả dòng `#include`
				i++
			}
		case c == '"' || c == '\'':
			q := c
			i++
			for i < n && src[i] != q && src[i] != '\n' {
				if src[i] == '\\' && i+1 < n {
					i++
				}
				i++
			}
			i = min(n, i+1)
			atLineStart = false
			emit("S")
		case isDigit(c) || (c == '.' && i+1 < n && isDigit(src[i+1])):
			for i < n && (isIdent(src[i]) || src[i] == '.' || ((src[i] == '+' || src[i] == '-') && (src[i-1] == 'e' || src[i-1] == 'E' || src[i-1] == 'p' || src[i-1] == 'P'))) {
				i++
			}
			atLineStart = false
			emit("N")
		case isIdentStart(c):
			j := i
			for j < n && isIdent(src[j]) {
				j++
			}
			word := src[i:j]
			i = j
			atLineStart = false
			if isKeyword(word) {
				emit(word)
			} else {
				emit("I")
			}
		default:
			atLineStart = false
			if i+3 <= n && isOp3(src[i:i+3]) {
				emit(src[i : i+3])
				i += 3
			} else if i+2 <= n && isOp2(src[i:i+2]) {
				emit(src[i : i+2])
				i += 2
			} else {
				emit(string(c))
				i++
			}
		}
	}
	return out
}

func hasInclude(src string, i int) bool {
	j := i + 1
	for j < len(src) && (src[j] == ' ' || src[j] == '\t') {
		j++
	}
	return strings.HasPrefix(src[j:], "include")
}
