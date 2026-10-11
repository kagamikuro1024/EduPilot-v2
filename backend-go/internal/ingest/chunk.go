// Package ingest: nạp tài liệu ở việc nền (SRS FEAT-docs-calendar 4.2): đọc bằng docling-serve → làm sạch → chia đoạn → nhúng qua internal/llm → ghi content_chunks.
package ingest

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Cỡ đoạn (bản chốt theo PoC, proposals #7): ≤ ChunkChars rune, không gối, mẩu < MinChunkChars gộp vào mẩu kề.
const (
	ChunkChars    = 800
	MinChunkChars = 200
	maxHeading    = 200
	// PageMarker là dấu ngắt trang docling chèn vào Markdown (md_page_break_placeholder).
	PageMarker = "<!-- page -->"
)

// Chunk là một đoạn đã chia. PageNo bắt đầu từ 1.
type Chunk struct {
	Ord     int
	PageNo  int
	Heading string
	Text    string
}

var (
	reHeading = regexp.MustCompile(`^\s{0,3}#{1,6}\s+(.+?)\s*#*\s*$`)
	reSpaces  = regexp.MustCompile(`[ \t\f\v\x{00a0}]+`)
	reBlanks  = regexp.MustCompile(`\n{3,}`)
	reImage   = regexp.MustCompile(`<!--\s*image\s*-->`)
	reSentEnd = regexp.MustCompile(`[.?!;…]\s+`)
)

// Clean bỏ ký tự điều khiển (trừ xuống dòng), dấu ảnh, gộp khoảng trắng. KHÔNG đổi Unicode về NFC ở đây (không có thư viện trong bảng):
// chữ được chuẩn hoá NFC bằng normalize(…, NFC) của Postgres lúc ghi.
func Clean(md string) string {
	md = strings.ToValidUTF8(md, "")
	md = strings.ReplaceAll(md, "\r\n", "\n")
	md = strings.ReplaceAll(md, "\r", "\n")
	md = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, md)
	md = reImage.ReplaceAllString(md, "")
	lines := strings.Split(md, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(reSpaces.ReplaceAllString(l, " "))
	}
	return strings.TrimSpace(reBlanks.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}

// PageCount là số trang suy từ dấu ngắt trang (n dấu → n+1 trang; trang trống không có dấu nên là cận dưới).
func PageCount(md string) int { return strings.Count(md, PageMarker) + 1 }

type block struct {
	page    int
	heading string
	text    string
}

// ChunkMarkdown chia Markdown (đã Clean) thành các đoạn: ranh giới theo thứ tự tiêu đề → đoạn → câu → khoảng trắng,
// mỗi đoạn ≤ max rune, mẩu < min gộp vào mẩu kề (nếu vẫn ≤ max), bỏ đoạn trùng văn bản.
func ChunkMarkdown(md string, maxRunes, minRunes int) []Chunk {
	var blocks []block
	page, heading := 1, ""
	for _, pg := range strings.Split(md, PageMarker) {
		for _, para := range strings.Split(pg, "\n\n") {
			var body []string
			flush := func() {
				if len(body) > 0 {
					blocks = append(blocks, block{page: page, heading: heading, text: strings.Join(body, "\n")})
					body = nil
				}
			}
			for _, line := range strings.Split(para, "\n") {
				if m := reHeading.FindStringSubmatch(line); m != nil {
					flush()
					heading = truncRunes(m[1], maxHeading)
					continue
				}
				if strings.TrimSpace(line) != "" {
					body = append(body, line)
				}
			}
			flush()
		}
		page++
	}

	// Tách khối quá dài; gom khối cùng tiêu đề tới khi đầy.
	var chunks []Chunk
	var cur *Chunk
	curRunes := 0
	emit := func() {
		if cur != nil {
			chunks = append(chunks, *cur)
			cur, curRunes = nil, 0
		}
	}
	for _, b := range blocks {
		for _, piece := range splitLong(b.text, maxRunes) {
			n := utf8.RuneCountInString(piece)
			if cur != nil && (cur.Heading != b.heading || curRunes+1+n > maxRunes) {
				emit()
			}
			if cur == nil {
				cur = &Chunk{PageNo: b.page, Heading: b.heading, Text: piece}
				curRunes = n
				continue
			}
			cur.Text += "\n" + piece
			curRunes += 1 + n
		}
	}
	emit()

	chunks = mergeShort(chunks, maxRunes, minRunes)
	seen := map[string]bool{}
	out := chunks[:0]
	for _, c := range chunks {
		if seen[c.Text] {
			continue
		}
		seen[c.Text] = true
		c.Ord = len(out)
		out = append(out, c)
	}
	return out
}

// mergeShort gộp mẩu < min vào mẩu trước (cùng hoặc khác tiêu đề đều được, miễn tổng ≤ max), không thì vào mẩu sau.
func mergeShort(cs []Chunk, maxRunes, minRunes int) []Chunk {
	for i := 0; i < len(cs); i++ {
		n := utf8.RuneCountInString(cs[i].Text)
		if n >= minRunes {
			continue
		}
		if i > 0 && utf8.RuneCountInString(cs[i-1].Text)+1+n <= maxRunes {
			cs[i-1].Text += "\n" + cs[i].Text
			cs = append(cs[:i], cs[i+1:]...)
			i--
			continue
		}
		if i+1 < len(cs) && n+1+utf8.RuneCountInString(cs[i+1].Text) <= maxRunes {
			cs[i+1].Text = cs[i].Text + "\n" + cs[i+1].Text
			cs[i+1].PageNo, cs[i+1].Heading = cs[i].PageNo, cs[i].Heading
			cs = append(cs[:i], cs[i+1:]...)
			i--
		}
	}
	return cs
}

// splitLong cắt một khối dài hơn max theo câu, rồi khoảng trắng, rồi (cùng lắm) giữa hai rune; không bao giờ cắt giữa một rune.
func splitLong(s string, maxRunes int) []string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return []string{s}
	}
	// các "đơn vị" theo câu
	var units []string
	last := 0
	for _, loc := range reSentEnd.FindAllStringIndex(s, -1) {
		units = append(units, s[last:loc[1]])
		last = loc[1]
	}
	units = append(units, s[last:])
	var out []string
	var cur strings.Builder
	curN := 0
	flush := func() {
		if t := strings.TrimSpace(cur.String()); t != "" {
			out = append(out, t)
		}
		cur.Reset()
		curN = 0
	}
	for _, u := range units {
		n := utf8.RuneCountInString(u)
		if n > maxRunes { // câu quá dài: cắt theo khoảng trắng / rune
			flush()
			out = append(out, hardSplit(u, maxRunes)...)
			continue
		}
		if curN+n > maxRunes {
			flush()
		}
		cur.WriteString(u)
		curN += n
	}
	flush()
	return out
}

func hardSplit(s string, maxRunes int) []string {
	var out []string
	r := []rune(s)
	for len(r) > maxRunes {
		cut := maxRunes
		for i := maxRunes; i > maxRunes/2; i-- { // lùi tới khoảng trắng gần nhất
			if unicode.IsSpace(r[i]) {
				cut = i
				break
			}
		}
		if t := strings.TrimSpace(string(r[:cut])); t != "" {
			out = append(out, t)
		}
		r = r[cut:]
	}
	if t := strings.TrimSpace(string(r)); t != "" {
		out = append(out, t)
	}
	return out
}

func truncRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// EmbedInput là chuỗi gửi đi nhúng: tiêu đề + xuống dòng + văn bản (cột text lưu nguyên văn).
func EmbedInput(c Chunk) string {
	if c.Heading == "" {
		return c.Text
	}
	return c.Heading + "\n" + c.Text
}
