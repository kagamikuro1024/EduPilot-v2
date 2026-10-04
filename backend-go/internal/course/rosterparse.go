package course

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"

	"github.com/edupilot/backend-go/internal/auth"
)

// Giới hạn tệp roster (SRS 4.5). Tệp chỉ sống trong bộ nhớ yêu cầu: không ghi đĩa, không ghi blob.
const (
	RosterMaxFileBytes = 2 << 20  // 2 MiB; vượt ⇒ 413
	RosterMaxRows      = 500      // dòng dữ liệu (không tính tiêu đề, bỏ dòng trống)
	rosterMaxUnzipped  = 20 << 20 // tổng XLSX sau giải nén
	rosterMaxZipFiles  = 200
	rosterMaxLineLen   = 10000 // ký tự / dòng CSV
	rosterMaxScanLines = 5000  // dòng XLSX quét tối đa kể cả dòng trống (chặn sheet 1 triệu dòng trống)
)

// ErrFileTooLarge: tệp > 2 MiB (handler ánh xạ 413).
var ErrFileTooLarge = errors.New("course: tệp roster quá lớn")

// errUnsupportedFile: không đọc được tệp (sai loại, hỏng, ký tự điều khiển, dòng quá dài, bom nén). Thông báo không lộ chi tiết nội bộ.
func errUnsupportedFile() error {
	return &InvalidError{Field: "file", Code: "UNSUPPORTED_FILE", Message: "Không đọc được tệp. Hãy dùng CSV UTF-8 hoặc XLSX."}
}

func errTooManyRows() error {
	return &InvalidError{Field: "file", Code: "TOO_MANY_ROWS", Message: "Tệp có quá 500 dòng. Hãy tách thành nhiều tệp."}
}

// RosterRow là một dòng dữ liệu đã đọc, giá trị mới cắt khoảng trắng — CHƯA kiểm. Line là số dòng trong tệp (tiêu đề = 1).
type RosterRow struct {
	Line              int
	Email, Name, Code string
}

const (
	colEmail = iota
	colName
	colCode
	colCount
)

// columnOf: tiêu đề (đã bỏ dấu, chữ thường) ⇒ cột; -1 nếu không nhận ra. SRS 4.5.
func columnOf(h string) int {
	switch h {
	case "email", "e-mail", "mail":
		return colEmail
	case "full_name", "ho va ten", "ho ten", "name":
		return colName
	case "student_code", "mssv", "ma so sinh vien":
		return colCode
	}
	return -1
}

func columnLabel(c int) string { return [colCount]string{"Email", "Họ và tên", "MSSV"}[c] }

// ParseRoster đọc CSV (UTF-8, BOM tuỳ chọn, dấu , hoặc ;) hoặc XLSX (sheet đầu); loại tệp theo magic bytes, không theo đuôi.
func ParseRoster(data []byte) ([]RosterRow, error) {
	if len(data) > RosterMaxFileBytes {
		return nil, ErrFileTooLarge
	}
	var grid [][]string
	var lines []int
	var err error
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		grid, lines, err = readXLSX(data)
	} else {
		grid, lines, err = readCSV(data)
	}
	if err != nil {
		return nil, err
	}
	return rowsOf(grid, lines)
}

// rowsOf tìm tiêu đề (dòng đầu), dựng bản đồ cột rồi đổi các dòng sau thành RosterRow; bỏ dòng trống.
func rowsOf(grid [][]string, lines []int) ([]RosterRow, error) {
	if len(grid) == 0 {
		return nil, errUnsupportedFile()
	}
	pos := [colCount]int{-1, -1, -1}
	for i, h := range grid[0] {
		if c := columnOf(strings.TrimSpace(auth.Fold(h))); c >= 0 && pos[c] < 0 {
			pos[c] = i
		}
	}
	for c, p := range pos {
		if p < 0 {
			return nil, &InvalidError{Field: "file", Code: "MISSING_COLUMN", Message: fmt.Sprintf("Thiếu cột %s.", columnLabel(c))}
		}
	}
	var out []RosterRow
	for i := 1; i < len(grid); i++ {
		cell := func(c int) string {
			if pos[c] < len(grid[i]) {
				return strings.TrimSpace(grid[i][pos[c]])
			}
			return ""
		}
		row := RosterRow{Line: lines[i], Email: cell(colEmail), Name: cell(colName), Code: cell(colCode)}
		if row.Email == "" && row.Name == "" && row.Code == "" {
			continue
		}
		if len(out) == RosterMaxRows {
			return nil, errTooManyRows()
		}
		out = append(out, row)
	}
	return out, nil
}

// readCSV: UTF-8 hợp lệ, không ký tự điều khiển (trừ \t \r \n), dòng ≤ 10.000 ký tự. Trả lưới ô và số dòng thật của từng bản ghi.
func readCSV(data []byte) ([][]string, []int, error) {
	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF"))
	if !utf8.Valid(data) || bytes.ContainsFunc(data, func(r rune) bool { return unicode.IsControl(r) && r != '\t' && r != '\r' && r != '\n' }) {
		return nil, nil, errUnsupportedFile()
	}
	for _, ln := range bytes.Split(data, []byte("\n")) {
		if utf8.RuneCount(ln) > rosterMaxLineLen {
			return nil, nil, errUnsupportedFile()
		}
	}
	first, _, _ := bytes.Cut(data, []byte("\n"))
	comma := ','
	if bytes.Count(first, []byte(";")) > bytes.Count(first, []byte(",")) {
		comma = ';'
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = comma
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	var grid [][]string
	var lines []int
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, errUnsupportedFile()
		}
		line, _ := r.FieldPos(0)
		grid = append(grid, rec)
		lines = append(lines, line)
		if len(grid) > RosterMaxRows+1+rosterMaxScanLines {
			return nil, nil, errUnsupportedFile()
		}
	}
	return grid, lines, nil
}

// readXLSX: kiểm kích thước giải nén theo mục lục zip TRƯỚC khi mở (chống bom nén), rồi đọc sheet đầu theo dòng.
func readXLSX(data []byte) ([][]string, []int, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) > rosterMaxZipFiles {
		return nil, nil, errUnsupportedFile()
	}
	var total uint64
	for _, f := range zr.File {
		total += f.UncompressedSize64
		if total > rosterMaxUnzipped {
			return nil, nil, errUnsupportedFile()
		}
	}
	// UnzipSizeLimit chặn tiếp trường hợp mục lục khai nhỏ hơn thật.
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: rosterMaxUnzipped, UnzipXMLSizeLimit: rosterMaxUnzipped})
	if err != nil {
		return nil, nil, errUnsupportedFile()
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, errUnsupportedFile()
	}
	rows, err := f.Rows(sheets[0])
	if err != nil {
		return nil, nil, errUnsupportedFile()
	}
	defer func() { _ = rows.Close() }()
	var grid [][]string
	var lines []int
	for line := 1; rows.Next(); line++ {
		if line > RosterMaxRows+1+rosterMaxScanLines {
			return nil, nil, errUnsupportedFile()
		}
		cols, err := rows.Columns()
		if err != nil {
			return nil, nil, errUnsupportedFile()
		}
		for _, c := range cols {
			if utf8.RuneCountInString(c) > rosterMaxLineLen {
				return nil, nil, errUnsupportedFile()
			}
		}
		grid = append(grid, cols)
		lines = append(lines, line)
	}
	return grid, lines, nil
}
