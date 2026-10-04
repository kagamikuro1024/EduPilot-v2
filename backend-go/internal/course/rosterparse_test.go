package course_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/edupilot/backend-go/internal/course"
)

const header = "Email,Họ và tên,MSSV\n"

func xlsxOf(t *testing.T, rows [][]any) []byte {
	t.Helper()
	f := excelize.NewFile()
	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		require.NoError(t, err)
		require.NoError(t, f.SetSheetRow("Sheet1", cell, &row))
	}
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)
	return buf.Bytes()
}

func codeOf(err error) string {
	var inv *course.InvalidError
	if err != nil && errors.As(err, &inv) {
		return inv.Code
	}
	return ""
}

func TestRosterCSVParse(t *testing.T) {
	t.Parallel()
	rows, err := course.ParseRoster([]byte("MSSV,EMAIL , Full_Name\n20229001,a@x.test, Nguyễn Văn A \n\n20229002,b@x.test,Lê B\n"))
	require.NoError(t, err)
	require.Equal(t, []course.RosterRow{
		{Line: 2, Email: "a@x.test", Name: "Nguyễn Văn A", Code: "20229001"},
		{Line: 4, Email: "b@x.test", Name: "Lê B", Code: "20229002"}, // dòng trống vẫn tính vào số dòng thật
	}, rows, "tiêu đề không phân biệt hoa thường / thứ tự cột; số dòng là dòng trong tệp")
	for _, h := range []string{"e-mail,Họ tên,Mã số sinh viên", "Mail,Name,student_code", "email,ho va ten,mssv"} {
		_, err := course.ParseRoster([]byte(h + "\na@x.test,A,20229001\n"))
		require.NoError(t, err, h)
	}
}

func TestRosterXLSXParse(t *testing.T) {
	t.Parallel()
	rows, err := course.ParseRoster(xlsxOf(t, [][]any{{"Email", "Họ và tên", "MSSV"}, {"a@x.test", "Nguyễn A", 20229001}, {}, {"b@x.test", "Lê B", "20229002"}}))
	require.NoError(t, err)
	require.Equal(t, []course.RosterRow{
		{Line: 2, Email: "a@x.test", Name: "Nguyễn A", Code: "20229001"},
		{Line: 4, Email: "b@x.test", Name: "Lê B", Code: "20229002"},
	}, rows, "ô số đọc như chữ; dòng trống giữ số dòng thật")
}

func TestRosterBOMAndSemicolon(t *testing.T) {
	t.Parallel()
	rows, err := course.ParseRoster([]byte("\xEF\xBB\xBFEmail;Họ và tên;MSSV\r\na@x.test;Trần, Văn A;20229001\r\n"))
	require.NoError(t, err)
	require.Equal(t, []course.RosterRow{{Line: 2, Email: "a@x.test", Name: "Trần, Văn A", Code: "20229001"}}, rows)
}

func TestRosterRowLimit500(t *testing.T) {
	t.Parallel()
	mk := func(n int) []byte {
		var b strings.Builder
		b.WriteString(header)
		for i := range n {
			fmt.Fprintf(&b, "sv%d@x.test,Sinh Viên %d,B20DC%05d\n", i, i, i)
		}
		return []byte(b.String())
	}
	rows, err := course.ParseRoster(mk(500))
	require.NoError(t, err)
	require.Len(t, rows, 500)
	_, err = course.ParseRoster(mk(501))
	require.Equal(t, "TOO_MANY_ROWS", codeOf(err))
}

func TestRosterMagicBytes(t *testing.T) {
	t.Parallel()
	// CSV đổi đuôi .xlsx / XLSX đổi đuôi .csv: loại tệp theo byte đầu, nên cả hai đọc được đúng như nội dung.
	_, err := course.ParseRoster([]byte(header + "a@x.test,A,20229001\n"))
	require.NoError(t, err)
	_, err = course.ParseRoster(xlsxOf(t, [][]any{{"Email", "Họ và tên", "MSSV"}, {"a@x.test", "A", "20229001"}}))
	require.NoError(t, err)
	for name, data := range map[string][]byte{
		"ELF":       append([]byte("\x7fELF\x02\x01\x01"), bytes.Repeat([]byte{0}, 64)...),
		"PNG":       []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"),
		"PDF":       []byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n"),
		"UTF-16":    []byte("\xff\xfeE\x00m\x00a\x00i\x00l\x00"),
		"rỗng":      {},
		"Latin-1":   []byte("Email,Họ tên,MSSV\na@x.test,Nguy\xeb,20229001\n"),
		"gzip":      []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00"),
		"PK giả":    []byte("PK\x03\x04 không phải zip"),
		"chỉ BOM":   []byte("\xEF\xBB\xBF"),
		"1 dòng":    []byte("Email,Họ và tên,MSSV\n"),
		"không cột": []byte("a\nb\n"),
	} {
		_, err := course.ParseRoster(data)
		switch name {
		case "1 dòng": // chỉ tiêu đề: hợp lệ, 0 dòng
			require.NoError(t, err, name)
		case "không cột":
			require.Equal(t, "MISSING_COLUMN", codeOf(err), name)
		default:
			require.Equal(t, "UNSUPPORTED_FILE", codeOf(err), name)
		}
	}
}

func TestRosterGarbageFile(t *testing.T) {
	t.Parallel()
	for name, data := range map[string]string{
		"ký tự điều khiển": header + "a@x.test,A\x00B,20229001\n",
		"ESC":              header + "a@x.test,A\x1bB,20229001\n",
	} {
		_, err := course.ParseRoster([]byte(data))
		require.Equal(t, "UNSUPPORTED_FILE", codeOf(err), name)
		var inv *course.InvalidError
		require.ErrorAs(t, err, &inv, name)
		require.Equal(t, "Không đọc được tệp. Hãy dùng CSV UTF-8 hoặc XLSX.", inv.Message, name)
	}
}

func TestRosterCorruptXLSX(t *testing.T) {
	t.Parallel()
	good := xlsxOf(t, [][]any{{"Email", "Họ và tên", "MSSV"}, {"a@x.test", "A", "20229001"}})
	for name, data := range map[string][]byte{
		"cắt cụt":        good[:len(good)/2],
		"hỏng giữa":      append(append([]byte{}, good[:200]...), bytes.Repeat([]byte{0xAB}, 300)...),
		"zip không XLSX": zipOf(t, map[string]string{"hello.txt": "hi"}),
	} {
		_, err := course.ParseRoster(data)
		require.Equal(t, "UNSUPPORTED_FILE", codeOf(err), name)
	}
}

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for n, c := range files {
		w, err := zw.Create(n)
		require.NoError(t, err)
		_, err = w.Write([]byte(c))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

// Bom nén: một XLSX hợp lệ nhưng sheet giải nén ra > 20 MiB trong < 2 MiB nén (lặp byte nén rất tốt). Phải bị từ chối, không cạn bộ nhớ.
func TestRosterZipBomb(t *testing.T) {
	t.Parallel()
	good := xlsxOf(t, [][]any{{"Email", "Họ và tên", "MSSV"}, {"a@x.test", "A", "20229001"}})
	zr, err := zip.NewReader(bytes.NewReader(good), int64(len(good)))
	require.NoError(t, err)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		w, err := zw.Create(f.Name)
		require.NoError(t, err)
		rc, err := f.Open()
		require.NoError(t, err)
		var b bytes.Buffer
		_, err = b.ReadFrom(rc)
		require.NoError(t, err)
		_ = rc.Close()
		data := b.Bytes()
		if strings.HasSuffix(f.Name, "sheet1.xml") {
			pad := bytes.Repeat([]byte(" "), 25<<20)
			data = append(append(append([]byte{}, data[:len(data)-len("</worksheet>")]...), pad...), []byte("</worksheet>")...)
		}
		_, err = w.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	require.Less(t, buf.Len(), course.RosterMaxFileBytes, "bom phải nhỏ hơn giới hạn tệp để thử đúng giới hạn giải nén")
	_, err = course.ParseRoster(buf.Bytes())
	require.Equal(t, "UNSUPPORTED_FILE", codeOf(err))
}

func TestRosterLongLine(t *testing.T) {
	t.Parallel()
	_, err := course.ParseRoster([]byte(header + "a@x.test," + strings.Repeat("A", 10001) + ",20229001\n"))
	require.Equal(t, "UNSUPPORTED_FILE", codeOf(err))
	_, err = course.ParseRoster([]byte(header + "a@x.test," + strings.Repeat("A", 9000) + ",20229001\n"))
	require.NoError(t, err, "dưới 10.000 ký tự vẫn đọc (tên sẽ bị cắt ở bước chuẩn hoá)")
}

func TestRosterMissingColumn(t *testing.T) {
	t.Parallel()
	for csvHeader, col := range map[string]string{"Họ và tên,MSSV": "Email", "Email,MSSV": "Họ và tên", "Email,Họ và tên": "MSSV"} {
		_, err := course.ParseRoster([]byte(csvHeader + "\nx,y\n"))
		require.Equal(t, "MISSING_COLUMN", codeOf(err), csvHeader)
		require.Contains(t, err.Error(), "MISSING_COLUMN")
		var inv *course.InvalidError
		require.ErrorAs(t, err, &inv)
		require.Contains(t, inv.Message, col, "thông báo nêu tên cột thiếu")
	}
}

func TestRosterFormulaStoredAsText(t *testing.T) {
	t.Parallel()
	rows, err := course.ParseRoster([]byte(header + "a@x.test,=HYPERLINK(\"http://evil\"),20229001\nb@x.test,+84 B,20229002\nc@x.test,@SUM(1),20229003\nd@x.test,-2+3,20229004\n"))
	require.NoError(t, err)
	require.Equal(t, `=HYPERLINK("http://evil")`, rows[0].Name, "giữ nguyên như chữ, không thực thi, không bỏ dấu")
	require.Equal(t, "+84 B", rows[1].Name)
	require.Equal(t, "@SUM(1)", rows[2].Name)
	require.Equal(t, "-2+3", rows[3].Name)
}
