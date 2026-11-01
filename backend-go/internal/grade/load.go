package grade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/store"
)

// Loaded là một công thức đã đọc từ DB: dòng gốc, dạng cho engine (`Scheme`) và dạng cho bộ kiểm (`Draft`).
type Loaded struct {
	ID          uuid.UUID
	No          int
	Version     int
	Status      string
	SourceTitle string
	SourceDocID *uuid.UUID
	ConfirmedAt *time.Time
	Scheme      Scheme
	Draft       Draft
	// RuleErr: luật có tham số hỏng (engine không dùng được Scheme.Rules đầy đủ); nil với công thức đã CONFIRMED.
	RuleErr error
}

// LoadScheme đọc MỘT công thức của lớp (thành phần, item, luật). Không thuộc lớp → pgx.ErrNoRows.
func LoadScheme(ctx context.Context, q *store.Queries, courseID, schemeID uuid.UUID) (Loaded, error) {
	sc, err := q.GradeSchemeGet(ctx, store.GradeSchemeGetParams{CourseID: courseID, ID: schemeID})
	if err != nil {
		return Loaded{}, err
	}
	ids := []string{schemeID.String()}
	comps, err := q.GradeComponentsOf(ctx, ids)
	if err != nil {
		return Loaded{}, fmt.Errorf("grade: đọc thành phần: %w", err)
	}
	rules, err := q.GradeRulesOf(ctx, ids)
	if err != nil {
		return Loaded{}, fmt.Errorf("grade: đọc luật: %w", err)
	}
	items, err := LoadItems(ctx, q, courseID, schemeID)
	if err != nil {
		return Loaded{}, err
	}
	l := Loaded{ID: sc.ID, No: int(sc.SchemeNo), Version: int(sc.Version), Status: string(sc.Status), SourceDocID: sc.SourceDocumentID, ConfirmedAt: sc.ConfirmedAt, Draft: draftOf(comps, rules)}
	if sc.SourceTitle != nil {
		l.SourceTitle = *sc.SourceTitle
	}
	l.Scheme = Scheme{ID: sc.ID, No: l.No, Version: l.Version}
	for _, c := range comps {
		var cs []storedCite
		_ = json.Unmarshal(c.Citations, &cs)
		comp := Component{ID: c.ID, Key: c.Key, Name: c.Name, Group: Group(c.Grp), Weight: c.Weight, FilledByUser: c.FilledByUser}
		for _, x := range cs {
			comp.Citations = append(comp.Citations, Citation(x))
		}
		for _, it := range items {
			if it.ComponentID == c.ID {
				comp.Items = append(comp.Items, it)
			}
		}
		l.Scheme.Components = append(l.Scheme.Components, comp)
	}
	rr := make([]RuleRow, len(rules))
	for i, r := range rules {
		rr[i] = RuleRow{Kind: RuleKind(r.Kind), Applicable: r.Applicable, Params: r.Params, FilledByUser: r.FilledByUser}
		if r.SourceQuote != nil {
			rr[i].SourceQuote = *r.SourceQuote
		}
		if r.SourcePage != nil {
			rr[i].SourcePage = int(*r.SourcePage)
		}
	}
	l.Scheme.Rules, l.RuleErr = BuildRules(rr)
	return l, nil
}

// LoadConfirmed đọc công thức CONFIRMED của lớp; ok=false khi lớp chưa có.
func LoadConfirmed(ctx context.Context, q *store.Queries, courseID uuid.UUID) (Loaded, bool, error) {
	id, err := q.GradeSchemeIDByStatus(ctx, store.GradeSchemeIDByStatusParams{CourseID: courseID, Status: store.GradeSchemeStatusCONFIRMED})
	if errors.Is(err, pgx.ErrNoRows) {
		return Loaded{}, false, nil
	}
	if err != nil {
		return Loaded{}, false, fmt.Errorf("grade: công thức đã xác nhận: %w", err)
	}
	l, err := LoadScheme(ctx, q, courseID, id)
	return l, err == nil, err
}

// PageText là văn bản của một trang tài liệu (các đoạn của trang nối bằng xuống dòng, đúng thứ tự `ord`).
type PageText struct {
	No   int
	Text string
}

const docChunkPage = 200

// DocPages đọc văn bản tài liệu của lớp theo trang (qua `DocChunks`, đã lọc lớp; đoạn không có số trang tính là trang 1). Dừng khi tổng ký tự (rune) vượt
// `maxChars` (>0) và trả tooLong=true; chars là tổng đã đọc.
func DocPages(ctx context.Context, q *store.Queries, courseID, docID uuid.UUID, maxChars int) (pages []PageText, chars int, tooLong bool, err error) {
	after := int32(-1)
	for {
		rows, err := q.DocChunks(ctx, store.DocChunksParams{DocID: docID, CourseID: courseID, AfterOrd: after, PageLimit: docChunkPage})
		if err != nil {
			return nil, 0, false, fmt.Errorf("grade: đọc đoạn tài liệu: %w", err)
		}
		for _, r := range rows {
			no := 1
			if r.PageNo != nil {
				no = int(*r.PageNo)
			}
			chars += len([]rune(r.Text))
			if n := len(pages); n > 0 && pages[n-1].No == no {
				pages[n-1].Text += "\n" + r.Text
			} else {
				pages = append(pages, PageText{No: no, Text: r.Text})
			}
			after = r.Ord
		}
		if maxChars > 0 && chars > maxChars {
			return pages, chars, true, nil
		}
		if len(rows) < docChunkPage {
			return pages, chars, false, nil
		}
	}
}
