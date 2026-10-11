package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Mã lỗi (jobs.error.code) và câu tiếng Việt lưu ở documents.error. Không lộ URL, đường dẫn, dấu vết.
const (
	CodeExtractUnavailable = "EXTRACT_UNAVAILABLE"
	CodeNoText             = "NO_TEXT"
	CodeTooManyPages       = "TOO_MANY_PAGES"
	CodeHashMismatch       = "HASH_MISMATCH"
	CodeEmbedFailed        = "EMBED_FAILED"
	CodeNotReady           = "DOCUMENT_NOT_READY"

	// TopicDocumentChanged: tạo / sẵn sàng / xoá / đổi cờ / chia sẻ / sửa đoạn → tăng ep:rag:ver:{course} (US-P8-01 AC14).
	TopicDocumentChanged = "document.changed"

	// Ngưỡng bản scan: lượt 1 ra < 100 ký tự / trang thì chạy OCR; sau đó < 50 ký tự → NO_TEXT.
	scanCharsPerPage = 100
	minTextChars     = 50
)

func message(code string, maxPages int) string {
	switch code {
	case CodeExtractUnavailable:
		return "Chưa đọc được tài liệu lúc này. Thử lại sau."
	case CodeNoText:
		return "Không đọc được chữ trong tệp (có thể là bản quét). Hãy tải bản có chữ."
	case CodeTooManyPages:
		return fmt.Sprintf("Tệp dài hơn %d trang. Hãy tách nhỏ.", maxPages)
	case CodeHashMismatch:
		return "Tệp bị lỗi khi tải lên. Tải lại."
	case CodeNotReady:
		return "Tài liệu chưa sẵn sàng."
	}
	return "Chưa lập chỉ mục được tài liệu. Thử lại sau."
}

// failure là lỗi đã phân loại; lỗi tạm thời đã được thử lại ≤ Settings.Retries lần trước khi tới đây.
type failure struct {
	code  string
	cause error
}

func (f *failure) Error() string { return f.code + ": " + fmt.Sprint(f.cause) }
func (f *failure) Unwrap() error { return f.cause }

// BlobReader đọc object (stream, không ghi đĩa).
type BlobReader interface {
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

// Processor chạy một việc ingest / lập chỉ mục lại. Idempotent: giao lại sau READY là no-op; ghi đoạn = xoá cũ + chèn mới trong MỘT giao dịch.
type Processor struct {
	Pool    *pgxpool.Pool
	Blob    BlobReader
	Docling *Docling
	LLM     llm.Client
	Jobs    *jobs.Runner
	Log     *slog.Logger
	Set     Settings
	Sleep   func(ctx context.Context, d time.Duration) // mặc định ngủ thật; test thay
}

// outcome của Ingest: done = job đã/ sẽ được đóng và tin có thể ACK; busy = chưa nhận được tài liệu → giữ tin để giao lại.
type outcome int

const (
	outDone outcome = iota
	outBusy
)

func (p *Processor) sleep(ctx context.Context, d time.Duration) {
	if p.Sleep != nil {
		p.Sleep(ctx, d)
		return
	}
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// Ingest xử lý tài liệu `docID` cho job `jobID`.
func (p *Processor) Ingest(ctx context.Context, jobID, docID uuid.UUID) (outcome, error) {
	q := store.New(p.Pool)
	doc, err := q.IngestClaim(ctx, store.IngestClaimParams{ID: docID, IdleSecs: p.Set.ClaimIdle.Seconds()})
	if errors.Is(err, pgx.ErrNoRows) {
		return p.unclaimed(ctx, q, jobID, docID)
	}
	if err != nil {
		return outBusy, fmt.Errorf("ingest: nhận việc: %w", err)
	}
	p.Jobs.Report(ctx, jobID, 5)

	lctx, stop := context.WithCancel(ctx)
	defer stop()
	go p.renewLease(lctx, docID)

	chunks, pages, ferr := p.extract(lctx, jobID, doc)
	if ferr == nil {
		var vecs [][]float32
		if doc.UseForRag {
			vecs, ferr = p.embed(lctx, jobID, doc.CourseID, doc.UploadedBy, chunks, 55, 95)
		}
		if ferr == nil {
			ok, err := p.write(ctx, doc, chunks, vecs, pages)
			if err != nil {
				return outBusy, err
			}
			if !ok { // đã mất thuê / tệp đổi: để lượt đang giữ tài liệu quyết
				return outDone, nil
			}
			return outDone, p.Jobs.Complete(ctx, jobID, map[string]string{"document_id": docID.String()})
		}
	}
	if errors.Is(ferr, errLostClaim) {
		return outBusy, nil
	}
	var f *failure
	if !errors.As(ferr, &f) {
		f = &failure{code: CodeExtractUnavailable, cause: ferr}
	}
	p.Log.WarnContext(ctx, "ingest thất bại", "document_id", docID.String(), "code", f.code, "error", f.cause.Error())
	msg := message(f.code, p.Set.MaxPages)
	if n, err := q.IngestMarkFailed(ctx, store.IngestMarkFailedParams{ID: docID, Error: &msg}); err != nil {
		return outBusy, fmt.Errorf("ingest: ghi lỗi tài liệu: %w", err)
	} else if n == 0 {
		return outDone, nil
	}
	return outDone, p.Jobs.Abort(ctx, jobID, f.code, msg)
}

// unclaimed: không nhận được dòng. READY / FAILED → job khớp theo (no-op với READY); QUEUED / PROCESSING (còn hạn, hoặc lớp đang bận) → KHÔNG đóng job.
func (p *Processor) unclaimed(ctx context.Context, q *store.Queries, jobID, docID uuid.UUID) (outcome, error) {
	st, err := q.IngestDocumentState(ctx, docID)
	if errors.Is(err, pgx.ErrNoRows) {
		return outDone, p.Jobs.Abort(ctx, jobID, CodeNotReady, message(CodeNotReady, 0)) // tài liệu đã bị xoá
	}
	if err != nil {
		return outBusy, fmt.Errorf("ingest: đọc trạng thái tài liệu: %w", err)
	}
	switch st.Status {
	case store.DocumentStatusREADY:
		return outDone, p.Jobs.Complete(ctx, jobID, map[string]string{"document_id": docID.String()})
	case store.DocumentStatusFAILED:
		msg := message(CodeExtractUnavailable, p.Set.MaxPages)
		if st.Error != nil {
			msg = *st.Error
		}
		return outDone, p.Jobs.Abort(ctx, jobID, CodeExtractUnavailable, msg)
	}
	return outBusy, nil
}

// renewLease gia hạn thuê mỗi LeaseRenew khi poll / nhúng: thuê tính từ nhịp cuối nên tệp lớn không bị nhận lại.
func (p *Processor) renewLease(ctx context.Context, docID uuid.UUID) {
	t := time.NewTicker(p.Set.LeaseRenew)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := store.New(p.Pool).IngestRenewLease(ctx, docID); err != nil {
				p.Log.WarnContext(ctx, "không gia hạn được thuê tài liệu", "document_id", docID.String(), "error", err.Error())
			}
		}
	}
}

// extract: đọc object → docling (lượt 1 không OCR; bản scan thì lượt 2 OCR Tesseract vie) → làm sạch → chia đoạn.
func (p *Processor) extract(ctx context.Context, jobID uuid.UUID, doc store.IngestClaimRow) ([]Chunk, int, error) {
	if doc.BlobKey == nil || doc.Filename == nil {
		return nil, 0, &failure{code: CodeExtractUnavailable, cause: errors.New("tài liệu không có tệp")}
	}
	isPDF := strings.EqualFold(strings.TrimPrefix(extOf(*doc.Filename), "."), "pdf")
	step := 10
	progress := func() {
		step = min(step+1, 44)
		p.Jobs.Report(ctx, jobID, step)
	}
	md, err := p.convert(ctx, doc, ConvertOptions{DocumentTimeout: p.Set.ExtractTimeout}, progress)
	if err != nil {
		return nil, 0, err
	}
	text := Clean(md)
	pages := PageCount(md)
	if isPDF && utf8.RuneCountInString(text) < scanCharsPerPage*pages {
		md2, err := p.convert(ctx, doc, ConvertOptions{OCR: true, DocumentTimeout: p.Set.ExtractTimeout}, progress)
		if err != nil {
			return nil, 0, err
		}
		if t2 := Clean(md2); utf8.RuneCountInString(t2) > utf8.RuneCountInString(text) {
			text, pages = t2, PageCount(md2)
		}
	}
	if utf8.RuneCountInString(text) < minTextChars {
		return nil, 0, &failure{code: CodeNoText, cause: errors.New("không có chữ")}
	}
	if pages > p.Set.MaxPages {
		return nil, 0, &failure{code: CodeTooManyPages, cause: fmt.Errorf("%d trang", pages)}
	}
	p.Jobs.Report(ctx, jobID, 50)
	chunks := ChunkMarkdown(text, p.Set.ChunkChars, MinChunkChars)
	if len(chunks) == 0 {
		return nil, 0, &failure{code: CodeNoText, cause: errors.New("không có đoạn")}
	}
	return chunks, pages, nil
}

func extOf(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}

// retryDelays: chờ trước mỗi lần thử lại khi docling tạm không dùng được (SRS 3.3): 5 s, 30 s, 2 phút → tối đa 1 + 3 lần gọi.
func retryDelays() [3]time.Duration {
	return [3]time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute}
}

// errLostClaim: sau lúc chờ không nhận lại được tài liệu (lớp đang bận / tắt máy): để tin được giao lại, không ghi FAILED.
var errLostClaim = errors.New("ingest: không nhận lại được tài liệu")

// convert đọc object, gửi docling, so băm. Docling tắt / chết: tài liệu QUAY VỀ QUEUED trong lúc chờ rồi thử lại sau 5 s, 30 s, 2 phút; hết 3 lần chờ mới FAILED.
func (p *Processor) convert(ctx context.Context, doc store.IngestClaimRow, o ConvertOptions, progress func()) (string, error) {
	delays := retryDelays()
	var lastErr error
	for attempt := 0; ; attempt++ {
		md, err := p.convertOnce(ctx, doc, o, progress)
		if err == nil {
			return md, nil
		}
		lastErr = err
		var ef *ExtractFailure
		var f *failure
		switch {
		case errors.As(err, &f):
			return "", err // HASH_MISMATCH... không thử lại
		case errors.As(err, &ef):
			if ef.PageSize {
				return "", &failure{code: CodeTooManyPages, cause: err}
			}
			return "", &failure{code: CodeExtractUnavailable, cause: err}
		case ctx.Err() != nil:
			return "", errLostClaim // tắt máy giữa chừng: tin còn nguyên trong hàng
		case !errors.Is(err, ErrExtractUnavailable):
			return "", &failure{code: CodeExtractUnavailable, cause: err}
		}
		if attempt >= len(delays) {
			return "", &failure{code: CodeExtractUnavailable, cause: lastErr}
		}
		q := store.New(p.Pool)
		if _, err := q.IngestRelease(ctx, doc.ID); err != nil {
			return "", fmt.Errorf("ingest: trả tài liệu về hàng chờ: %w", err)
		}
		p.sleep(ctx, delays[attempt])
		if ctx.Err() != nil {
			return "", errLostClaim
		}
		if _, err := q.IngestClaim(ctx, store.IngestClaimParams{ID: doc.ID, IdleSecs: p.Set.ClaimIdle.Seconds()}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return "", errLostClaim
			}
			return "", fmt.Errorf("ingest: nhận lại tài liệu: %w", err)
		}
	}
}

func (p *Processor) convertOnce(ctx context.Context, doc store.IngestClaimRow, o ConvertOptions, progress func()) (string, error) {
	rc, err := p.Blob.Get(ctx, *doc.BlobKey)
	if err != nil {
		return "", &failure{code: CodeExtractUnavailable, cause: fmt.Errorf("đọc tệp: %w", err)}
	}
	defer func() { _ = rc.Close() }()
	h := sha256.New()
	tee := io.TeeReader(rc, h)
	cctx, cancel := context.WithTimeout(ctx, p.Set.ExtractTimeout+30*time.Second)
	defer cancel()
	res, err := p.Docling.Convert(cctx, *doc.Filename, tee, o, progress)
	if err != nil {
		return "", err
	}
	_, _ = io.Copy(io.Discard, tee) // băm phải phủ cả tệp kể cả khi docling không đọc hết
	if doc.Sha256 != nil && hex.EncodeToString(h.Sum(nil)) != *doc.Sha256 {
		return "", &failure{code: CodeHashMismatch, cause: errors.New("băm không khớp")}
	}
	return res.Markdown, nil
}

// embed nhúng theo lô ≤ llm.EmbedBatch (làn BATCH, 1536 chiều). Lỗi tạm thời thử lại; sai chiều là lỗi cứng.
func (p *Processor) embed(ctx context.Context, jobID, courseID uuid.UUID, owner *uuid.UUID, chunks []Chunk, from, to int) ([][]float32, error) {
	ctx = llm.WithIdentity(ctx, llm.Identity{UserID: owner, CourseID: &courseID})
	inputs := make([]string, len(chunks))
	for i, c := range chunks {
		inputs[i] = EmbedInput(c)
	}
	return p.embedStrings(ctx, jobID, inputs, from, to)
}

func (p *Processor) embedStrings(ctx context.Context, jobID uuid.UUID, inputs []string, from, to int) ([][]float32, error) {
	out := make([][]float32, 0, len(inputs))
	nb := (len(inputs) + llm.EmbedBatch - 1) / llm.EmbedBatch
	for b := range nb {
		lo, hi := b*llm.EmbedBatch, min((b+1)*llm.EmbedBatch, len(inputs))
		var vecs [][]float32
		var err error
		for attempt := 1; attempt <= p.Set.Retries; attempt++ {
			vecs, err = p.LLM.Embed(ctx, llm.EmbedRequest{Inputs: inputs[lo:hi]})
			var dm *llm.ErrDimsMismatch
			if err == nil || errors.As(err, &dm) || ctx.Err() != nil {
				break
			}
			if attempt < p.Set.Retries {
				p.sleep(ctx, time.Duration(attempt)*5*time.Second)
			}
		}
		if err == nil && len(vecs) != hi-lo {
			err = fmt.Errorf("nhận %d vectơ cho %d đoạn", len(vecs), hi-lo)
		}
		if err != nil {
			return nil, &failure{code: CodeEmbedFailed, cause: err}
		}
		for _, v := range vecs {
			if len(v) != llm.EmbedDims {
				return nil, &failure{code: CodeEmbedFailed, cause: &llm.ErrDimsMismatch{Expected: llm.EmbedDims, Actual: len(v)}}
			}
		}
		out = append(out, vecs...)
		p.Jobs.Report(ctx, jobID, from+(to-from)*(b+1)/nb)
	}
	return out, nil
}

// audienceOf: GRADING cho ANSWER_KEY; STAFF nếu ẩn với sinh viên; còn lại ALL (SRS 4.1).
func audienceOf(t store.DocumentType, visible bool) store.ChunkAudience {
	switch {
	case t == store.DocumentTypeANSWERKEY:
		return store.ChunkAudienceGRADING
	case !visible:
		return store.ChunkAudienceSTAFF
	}
	return store.ChunkAudienceALL
}

// write: MỘT giao dịch — xoá đoạn cũ, chèn mới, READY (chỉ khi còn PROCESSING và băm không đổi), outbox document.changed.
func (p *Processor) write(ctx context.Context, doc store.IngestClaimRow, chunks []Chunk, vecs [][]float32, pages int) (bool, error) {
	ctx = context.WithoutCancel(ctx)
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("ingest: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if err := q.IngestDeleteChunks(ctx, store.IngestDeleteChunksParams{DocumentID: doc.ID, CourseID: doc.CourseID}); err != nil {
		return false, fmt.Errorf("ingest: xoá đoạn cũ: %w", err)
	}
	aud := audienceOf(doc.Type, doc.VisibleToStudents)
	for i, c := range chunks {
		page := int32(c.PageNo) //nolint:gosec // ≤ vài trăm
		var heading *string
		if c.Heading != "" {
			h := c.Heading
			heading = &h
		}
		var emb *pgvector.Vector
		if vecs != nil {
			v := pgvector.NewVector(vecs[i])
			emb = &v
		}
		if err := q.IngestInsertChunk(ctx, store.IngestInsertChunkParams{
			DocumentID: doc.ID, CourseID: doc.CourseID, Audience: aud, Ord: int32(i), //nolint:gosec // ≤ vài nghìn
			PageNo: &page, Heading: heading, Text: c.Text, Embedding: emb,
		}); err != nil {
			return false, fmt.Errorf("ingest: chèn đoạn: %w", err)
		}
	}
	n, err := q.IngestMarkReady(ctx, store.IngestMarkReadyParams{ID: doc.ID, PageCount: ptr(int32(pages)), Sha256: doc.Sha256}) //nolint:gosec // ≤ DOC_MAX_PAGES
	if err != nil {
		return false, fmt.Errorf("ingest: đánh dấu READY: %w", err)
	}
	if n == 0 {
		return false, nil // rollback: không để đoạn của lượt này đè lên tài liệu đã đổi
	}
	if _, err := outbox.Write(ctx, tx, TopicDocumentChanged, map[string]string{"course_id": doc.CourseID.String(), "document_id": doc.ID.String()}); err != nil {
		return false, fmt.Errorf("ingest: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("ingest: commit: %w", err)
	}
	return true, nil
}

func ptr[T any](v T) *T { return &v }

// Reindex nhúng lại một tài liệu từ CHỮ ĐÃ LƯU (không gọi docling), ghi từng lô trong một giao dịch. Idempotent theo (document_id, ord).
func (p *Processor) Reindex(ctx context.Context, jobID, docID uuid.UUID) error {
	st, err := store.New(p.Pool).IngestDocumentState(ctx, docID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && st.Status != store.DocumentStatusREADY) {
		return p.Jobs.Abort(ctx, jobID, CodeNotReady, message(CodeNotReady, 0))
	}
	if err != nil {
		return fmt.Errorf("ingest: đọc tài liệu: %w", err)
	}
	if err := p.reindexDoc(ctx, jobID, st.CourseID, docID, st.UploadedBy, 5, 95); err != nil {
		var f *failure
		if !errors.As(err, &f) {
			return err
		}
		return p.Jobs.Abort(ctx, jobID, f.code, message(f.code, 0))
	}
	return p.Jobs.Complete(ctx, jobID, map[string]string{"document_id": docID.String()})
}

// ReindexAll nhúng lại mọi tài liệu READY + use_for_rag của lớp (sau khi Admin đổi mô hình nhúng).
func (p *Processor) ReindexAll(ctx context.Context, jobID, courseID uuid.UUID, owner uuid.UUID) error {
	ids, err := store.New(p.Pool).ListCourseDocumentsForReindex(ctx, courseID)
	if err != nil {
		return fmt.Errorf("ingest: danh sách tài liệu: %w", err)
	}
	for i, id := range ids {
		lo, hi := 5+90*i/len(ids), 5+90*(i+1)/len(ids)
		if err := p.reindexDoc(ctx, jobID, courseID, id, &owner, lo, hi); err != nil {
			var f *failure
			if !errors.As(err, &f) {
				return err
			}
			return p.Jobs.Abort(ctx, jobID, f.code, message(f.code, 0))
		}
	}
	return p.Jobs.Complete(ctx, jobID, map[string]int{"documents": len(ids)})
}

func (p *Processor) reindexDoc(ctx context.Context, jobID, courseID, docID uuid.UUID, owner *uuid.UUID, from, to int) error {
	ctx = llm.WithIdentity(ctx, llm.Identity{UserID: owner, CourseID: &courseID})
	rows, err := store.New(p.Pool).ListChunkTexts(ctx, store.ListChunkTextsParams{DocumentID: docID, CourseID: courseID})
	if err != nil {
		return fmt.Errorf("ingest: đọc chữ đã lưu: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}
	nb := (len(rows) + llm.EmbedBatch - 1) / llm.EmbedBatch
	for b := range nb {
		lo, hi := b*llm.EmbedBatch, min((b+1)*llm.EmbedBatch, len(rows))
		inputs := make([]string, hi-lo)
		for i, r := range rows[lo:hi] {
			c := Chunk{Text: r.Text}
			if r.Heading != nil {
				c.Heading = *r.Heading
			}
			inputs[i] = EmbedInput(c)
		}
		vecs, err := p.embedStrings(ctx, jobID, inputs, from+(to-from)*b/nb, from+(to-from)*(b+1)/nb)
		if err != nil {
			return err
		}
		if err := p.writeEmbeddings(ctx, courseID, docID, rows[lo:hi], vecs); err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) writeEmbeddings(ctx context.Context, courseID, docID uuid.UUID, rows []store.ListChunkTextsRow, vecs [][]float32) error {
	ctx = context.WithoutCancel(ctx)
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ingest: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	for i, r := range rows {
		if _, err := q.SetChunkEmbedding(ctx, store.SetChunkEmbeddingParams{ID: r.ID, DocumentID: docID, CourseID: courseID, Embedding: ptr(pgvector.NewVector(vecs[i]))}); err != nil {
			return fmt.Errorf("ingest: ghi vectơ: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ingest: commit: %w", err)
	}
	return nil
}
