package submission

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"unicode/utf8"
)

// Reader là phần của platform/blob mà việc kiểm nội dung cần.
type Reader interface {
	GetRange(ctx context.Context, key string, offset, length int64) ([]byte, error)
}

// Kind là loại tệp nhận được, suy từ ĐUÔI (mime do máy khách khai bị bỏ — TLR-15).
type Kind struct {
	Mime  string
	sniff func(ctx context.Context, r Reader, key string, size int64) bool
}

const (
	mimePDF  = "application/pdf"
	mimeDocx = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	mimeText = "text/plain; charset=utf-8" // TXT và MD: không bao giờ text/html (không thể thành XSS lưu trên host lưu trữ)
	textHead = 64 << 10
)

// KindOf: đuôi thường → loại. ok=false: không nhận.
func KindOf(ext string) (Kind, bool) {
	switch ext {
	case ".pdf":
		return Kind{mimePDF, sniffPDF}, true
	case ".docx":
		return Kind{mimeDocx, sniffDocx}, true
	case ".txt", ".md":
		return Kind{mimeText, sniffText}, true
	}
	return Kind{}, false
}

// Sniff kiểm nội dung thật của object theo loại (PDF `%PDF-`; DOCX là zip có `[Content_Types].xml` và `word/document.xml`; TXT/MD UTF-8 hợp lệ, không byte NUL).
func (k Kind) Sniff(ctx context.Context, r Reader, key string, size int64) bool {
	return k.sniff(ctx, r, key, size)
}

func sniffPDF(ctx context.Context, r Reader, key string, size int64) bool {
	b, err := r.GetRange(ctx, key, 0, 5)
	return err == nil && bytes.Equal(b, []byte("%PDF-"))
}

// rangeAt đọc zip qua Range GET: zip.NewReader tự tìm thư mục trung tâm ở CUỐI tệp, không giả định mục nằm đầu.
type rangeAt struct {
	ctx  context.Context //nolint:containedctx // io.ReaderAt không có ctx; sống trong một lời gọi
	r    Reader
	key  string
	size int64
}

func (a rangeAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= a.size {
		return 0, io.EOF
	}
	b, err := a.r.GetRange(a.ctx, a.key, off, int64(len(p)))
	n := copy(p, b)
	if err != nil {
		return n, fmt.Errorf("đọc đoạn: %w", err)
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func sniffDocx(ctx context.Context, r Reader, key string, size int64) bool {
	zr, err := zip.NewReader(rangeAt{ctx, r, key, size}, size)
	if err != nil {
		return false
	}
	var types, doc bool
	for _, f := range zr.File {
		switch f.Name {
		case "[Content_Types].xml":
			types = true
		case "word/document.xml":
			doc = true
		}
	}
	return types && doc
}

func sniffText(ctx context.Context, r Reader, key string, size int64) bool {
	b, err := r.GetRange(ctx, key, 0, textHead)
	if err != nil || bytes.IndexByte(b, 0) >= 0 {
		return false
	}
	if utf8.Valid(b) {
		return true
	}
	// đầu đọc có thể cắt giữa một ký tự nhiều byte
	if size > int64(len(b)) {
		for k := 1; k < utf8.UTFMax && k < len(b); k++ {
			if utf8.Valid(b[:len(b)-k]) {
				return true
			}
		}
	}
	return false
}
