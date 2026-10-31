package grade

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Ghi chú cố định ở hàng cuối tệp xuất (cùng chữ với dòng dưới bảng của `/gradebook`).
const (
	OfficialNote   = "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường."
	exportPreview  = "Bản xem trước — chưa chốt"
	exportChunk    = 500
	exportSheet    = "Sổ điểm"
	exportCellFmt2 = 2 // numFmt dựng sẵn "0.00"
)

// SnapBreakdown là phần của `final_grade_snapshots.breakdown` mà tệp xuất đọc khi sổ LOCKED (cùng khoá với lúc chốt, US-P6-08).
type SnapBreakdown struct {
	Items   map[string]string `json:"items"` // khoá item → điểm
	Bonus   string            `json:"bonus"`
	Penalty string            `json:"penalty"`
	QT      string            `json:"qt"`
	CK      string            `json:"ck"`
}

// ExportFile là tệp XLSX đã dựng cùng số dòng (cho audit và tiêu đề).
type ExportFile struct {
	Bytes     []byte
	Rows      int
	ClassCode string
}

type exportRow struct {
	code, name                                 string
	items                                      []string
	bonus, penalty, qt, ck, hp, letter, remark string
}

// Export dựng tệp XLSX của sổ điểm (SRS 4.6.5): LOCKED → từ bản chốt mới nhất; OPEN → từ `Compute` (bản xem trước). Mỗi lần xuất ghi `audit_log`.
func (s *Service) Export(ctx context.Context, actor, courseID uuid.UUID) (ExportFile, error) {
	q := store.New(s.Pool)
	conf, ok, err := LoadConfirmed(ctx, q, courseID)
	if err != nil {
		return ExportFile{}, err
	}
	if !ok {
		return ExportFile{}, apierr.New(http.StatusConflict, apierr.SchemeNotConfirmed)
	}
	info, err := q.GradeCourseInfo(ctx, courseID)
	if err != nil {
		return ExportFile{}, fmt.Errorf("grade: thông tin lớp: %w", err)
	}
	gb, err := q.GradebookState(ctx, courseID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ExportFile{}, fmt.Errorf("grade: trạng thái sổ điểm: %w", err)
	}
	locked := err == nil && gb.Status == store.GradebookStatusLOCKED
	students, err := ActiveStudents(ctx, q, courseID)
	if err != nil {
		return ExportFile{}, err
	}
	var items []Item
	for _, c := range conf.Scheme.Components {
		items = append(items, c.Items...)
	}

	snaps := map[uuid.UUID]store.GradeSnapshotsAtRow{}
	if locked {
		rows, err := q.GradeSnapshotsAt(ctx, store.GradeSnapshotsAtParams{CourseID: courseID, Version: gb.SnapshotVersion})
		if err != nil {
			return ExportFile{}, fmt.Errorf("grade: bản chốt: %w", err)
		}
		for _, r := range rows {
			snaps[r.StudentID] = r
		}
	}

	rows := make([]exportRow, 0, len(students))
	for lo := 0; lo < len(students); lo += exportChunk {
		part := students[lo:min(lo+exportChunk, len(students))]
		ids := make([]uuid.UUID, len(part))
		for i, st := range part {
			ids[i] = st.ID
		}
		var in map[uuid.UUID]Input
		if !locked {
			if in, err = LoadInputs(ctx, q, courseID, conf.ID, ids); err != nil {
				return ExportFile{}, err
			}
		}
		for _, st := range part {
			row := exportRow{code: st.Code, name: st.FullName, items: make([]string, len(items))}
			if locked {
				fillFromSnapshot(&row, items, snaps[st.ID])
			} else {
				fillFromCompute(&row, items, conf, Compute(conf.Scheme, in[st.ID]), in[st.ID])
			}
			rows = append(rows, row)
		}
	}

	version := exportPreview
	if locked {
		version = fmt.Sprintf("Bản chốt v%d", gb.SnapshotVersion)
	}
	data, err := buildXLSX(info.Name, info.ClassCode, s.now(), version, items, rows)
	if err != nil {
		return ExportFile{}, err
	}
	after, _ := json.Marshal(map[string]any{"rows": len(rows), "locked": locked, "version": gb.SnapshotVersion})
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &courseID, ActorID: &actor, Entity: "gradebook", EntityID: courseID.String(), Action: "export", After: after}); err != nil {
		return ExportFile{}, fmt.Errorf("grade: audit xuất: %w", err) // fail-closed: không có dấu vết thì không phát tệp
	}
	return ExportFile{Bytes: data, Rows: len(rows), ClassCode: info.ClassCode}, nil
}

func fillFromCompute(row *exportRow, items []Item, conf Loaded, r Result, in Input) {
	for i, it := range items {
		if v, ok := in.Scores[it.ID]; ok {
			row.items[i] = v.Round(2).String()
		}
	}
	row.bonus, row.penalty = r.BonusApplied.Round(2).String(), r.Penalty.Round(2).String()
	if r.HasQT {
		row.qt = r.QT.Round(2).String()
	}
	if r.HasCK {
		row.ck = r.CK.Round(2).String()
	}
	if r.Status == StatusComplete {
		row.hp, row.letter = r.HP.Round(2).String(), r.Letter
	}
	row.remark = remarkOf(r, conf)
}

func fillFromSnapshot(row *exportRow, items []Item, sn store.GradeSnapshotsAtRow) {
	var b SnapBreakdown
	_ = json.Unmarshal(sn.Breakdown, &b)
	for i, it := range items {
		row.items[i] = b.Items[it.Key]
	}
	row.bonus, row.penalty, row.qt, row.ck = b.Bonus, b.Penalty, b.QT, b.CK
	row.hp, row.letter = sn.NumericGrade.Round(2).String(), sn.Letter
	if sn.FailedReason != nil {
		row.remark = lietRemark(*sn.FailedReason)
	}
}

// lietRemark: "Liệt: …" từ mã lý do (`LIET:final<3.0`, `LIET:absent>=7`).
func lietRemark(verdict string) string {
	switch {
	case strings.Contains(verdict, "final<"):
		return "Liệt: thi cuối kỳ dưới ngưỡng"
	case strings.Contains(verdict, "absent>="):
		return "Liệt: vắng quá số buổi cho phép"
	}
	return "Liệt"
}

// remarkOf: "Liệt: …" hoặc "Thiếu điểm: …" (tên thành phần còn thiếu).
func remarkOf(r Result, conf Loaded) string {
	switch {
	case strings.HasPrefix(r.Verdict, "LIET"):
		return lietRemark(r.Verdict)
	case r.Status != StatusComplete:
		names := make([]string, 0, len(r.Missing))
		for _, k := range r.Missing {
			name := k
			for _, c := range conf.Scheme.Components {
				if c.Key == k {
					name = c.Name
				}
			}
			names = append(names, name)
		}
		if len(names) == 0 {
			return "Thiếu điểm"
		}
		return "Thiếu điểm: " + strings.Join(names, ", ")
	}
	return ""
}

func buildXLSX(className, classCode string, at time.Time, version string, items []Item, rows []exportRow) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetName("Sheet1", exportSheet); err != nil {
		return nil, fmt.Errorf("grade: tên trang tính: %w", err)
	}
	num, err := f.NewStyle(&excelize.Style{NumFmt: exportCellFmt2})
	if err != nil {
		return nil, fmt.Errorf("grade: kiểu số: %w", err)
	}
	txt := func(cell, v string) { _ = f.SetCellStr(exportSheet, cell, v) } // luôn là chuỗi: ô bắt đầu `=`, `+`, `-`, `@` không thành công thức
	cell := func(col, row int) string { c, _ := excelize.CoordinatesToCellName(col, row); return c }
	ict := time.FixedZone("ICT", 7*3600)
	txt("A1", className)
	txt("B1", classCode)
	txt("C1", at.In(ict).Format("02/01/2006 15:04"))
	txt("D1", version)

	head := []string{"MSSV", "Họ tên"}
	for _, it := range items {
		head = append(head, it.Name)
	}
	head = append(head, "Điểm cộng", "Điểm trừ", "Điểm quá trình", "Điểm thi cuối kỳ", "Điểm học phần", "Điểm chữ", "Ghi chú")
	for i, h := range head {
		txt(cell(i+1, 2), h)
	}
	numeric := func(c int, r int, v string) {
		if v == "" {
			return
		}
		pos := cell(c, r)
		_ = f.SetCellDefault(exportSheet, pos, v) // chuỗi số → ô số, không đi qua float
		_ = f.SetCellStyle(exportSheet, pos, pos, num)
	}
	for i, r := range rows {
		line := i + 3
		txt(cell(1, line), r.code)
		txt(cell(2, line), r.name)
		c := 3
		for _, v := range r.items {
			numeric(c, line, v)
			c++
		}
		numeric(c, line, r.bonus)
		numeric(c+1, line, r.penalty)
		numeric(c+2, line, r.qt)
		numeric(c+3, line, r.ck)
		numeric(c+4, line, r.hp)
		if r.letter != "" {
			txt(cell(c+5, line), r.letter)
		}
		if r.remark != "" {
			txt(cell(c+6, line), r.remark)
		}
	}
	txt(cell(1, len(rows)+4), OfficialNote)
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("grade: ghi tệp xuất: %w", err)
	}
	return buf.Bytes(), nil
}
