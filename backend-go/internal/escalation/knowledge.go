package escalation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/knowledge"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn của mục tri thức (câu hỏi đã che / trả lời đã che).
const (
	KnowledgeQuestionMax = 1000
	KnowledgeAnswerMax   = AnswerMaxChars
)

// KnowledgePreview là bản xem trước để giảng viên sửa: câu hỏi / trả lời đã qua `privacy.Redact`; Personal = câu hỏi gốc là câu hỏi riêng (không lưu được).
type KnowledgePreview struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Personal bool   `json:"personal"`
}

// KnowledgeState là trạng thái mục tri thức của ticket (giao diện giảng viên: "Chưa nạp vào tri thức" + Thử lại khi Loaded=false).
type KnowledgeState struct {
	Saved    bool   `json:"saved"`
	Loaded   bool   `json:"loaded"`
	Question string `json:"question,omitempty"` // bản đã lưu (để `Thử lại` / sửa tiếp không mất chữ đã sửa)
	Answer   string `json:"answer,omitempty"`
}

func (s *Service) redact(ctx context.Context, courseID uuid.UUID, text string) (string, error) {
	if s.PII == nil {
		return text, nil
	}
	fs, err := s.PII.Detect(ctx, courseID, text)
	if err != nil {
		return "", unavailable()
	}
	return privacy.Redact(text, fs), nil
}

// savable nạp ticket ANSWERED / CLOSED của lớp và cặp hỏi–đáp gốc (đọc qua truy vấn của ticket, không đọc tin nào khác).
func (s *Service) savable(ctx context.Context, courseID, ticketID uuid.UUID) (store.EscalationTicket, string, error) {
	t, err := store.New(s.Pool).TicketGet(ctx, store.TicketGetParams{CourseID: courseID, ID: ticketID})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.EscalationTicket{}, "", notFound()
	}
	if err != nil {
		return store.EscalationTicket{}, "", unavailable()
	}
	if t.Status != store.TicketStatusANSWERED && t.Status != store.TicketStatusCLOSED {
		return store.EscalationTicket{}, "", apierr.New(http.StatusConflict, apierr.TicketStateInvalid)
	}
	ex, err := s.ReadExchange(ctx, courseID, ticketID)
	if err != nil {
		return store.EscalationTicket{}, "", err
	}
	q := ""
	if ex.Question != nil {
		q = ex.Question.Content
	}
	return t, q, nil
}

// KnowledgePreview là `GET …/tickets/{id}/knowledge-preview`: câu hỏi + trả lời đã che để sửa. Câu hỏi riêng tư (điểm / chuyên cần của riêng người đó) → Personal.
func (s *Service) KnowledgePreview(ctx context.Context, courseID, ticketID uuid.UUID) (KnowledgePreview, error) {
	t, q, err := s.savable(ctx, courseID, ticketID)
	if err != nil {
		return KnowledgePreview{}, err
	}
	rq, err := s.redact(ctx, courseID, q)
	if err != nil {
		return KnowledgePreview{}, err
	}
	ans := ""
	if t.Answer != nil {
		ans = *t.Answer
	}
	ra, err := s.redact(ctx, courseID, ans)
	if err != nil {
		return KnowledgePreview{}, err
	}
	return KnowledgePreview{Question: rq, Answer: ra, Personal: privacy.IsPersonalPattern(q)}, nil
}

// KnowledgeSave là `POST …/tickets/{id}/knowledge {question, answer}`: nhận BẢN ĐÃ SỬA, chạy lại Detect (còn PII → 422 PII_DETECTED), câu hỏi gốc riêng tư → 422 KNOWLEDGE_PERSONAL;
// thành công, trong MỘT giao dịch: content_chunks (embedding NULL) + knowledge_items + outbox knowledge.sync + save_as_knowledge + audit. Lưu hai lần = cập nhật.
func (s *Service) KnowledgeSave(ctx context.Context, a Actor, courseID, ticketID uuid.UUID, question, answer string) (KnowledgeState, error) {
	question, answer = strings.TrimSpace(question), strings.TrimSpace(answer)
	if n := utf8.RuneCountInString(question); n == 0 || n > KnowledgeQuestionMax {
		return KnowledgeState{}, apierr.Validation(apierr.FieldError{Field: "question", Code: "length", Message: fmt.Sprintf("Câu hỏi dài từ 1 đến %d ký tự.", KnowledgeQuestionMax)})
	}
	if n := utf8.RuneCountInString(answer); n == 0 || n > KnowledgeAnswerMax {
		return KnowledgeState{}, apierr.Validation(apierr.FieldError{Field: "answer", Code: "length", Message: fmt.Sprintf("Trả lời dài từ 1 đến %d ký tự.", KnowledgeAnswerMax)})
	}
	t, orig, err := s.savable(ctx, courseID, ticketID)
	if err != nil {
		return KnowledgeState{}, err
	}
	if privacy.IsPersonalPattern(orig) {
		return KnowledgeState{}, apierr.New(http.StatusUnprocessableEntity, apierr.KnowledgePersonal)
	}
	if s.PII != nil {
		var bad []PIIError
		for _, txt := range []string{question, answer} {
			fs, err := s.PII.Detect(ctx, courseID, txt)
			if err != nil {
				return KnowledgeState{}, unavailable()
			}
			for _, f := range fs {
				bad = append(bad, PIIError{Start: f.Start, End: f.End, Kind: string(f.Kind)})
			}
			if len(fs) > 0 {
				break
			}
		}
		if len(bad) > 0 {
			return KnowledgeState{}, apierr.New(http.StatusUnprocessableEntity, apierr.PIIDetected).WithDetails(bad)
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return KnowledgeState{}, unavailable()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := knowledge.Upsert(ctx, tx, knowledge.Source{CourseID: courseID, Type: knowledge.SourceTicket, ID: ticketID, By: &a.UserID}, question, answer); err != nil {
		return KnowledgeState{}, err
	}
	q := store.New(tx)
	if _, err := q.TicketSetKnowledge(ctx, store.TicketSetKnowledgeParams{CourseID: courseID, ID: ticketID, Saved: true}); err != nil {
		return KnowledgeState{}, unavailable()
	}
	if err := s.audit(ctx, q, courseID, a.UserID, t.ID, "escalation_ticket.knowledge_saved", t.Status, t.Status); err != nil {
		return KnowledgeState{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return KnowledgeState{}, unavailable()
	}
	return KnowledgeState{Saved: true}, nil
}

// KnowledgeRemove là `DELETE …/tickets/{id}/knowledge`: gỡ đoạn (≤ 5 giây ra khỏi kết quả tìm), `save_as_knowledge=false`. Gỡ khi chưa lưu = 200 không đổi.
func (s *Service) KnowledgeRemove(ctx context.Context, a Actor, courseID, ticketID uuid.UUID) (KnowledgeState, error) {
	t, err := store.New(s.Pool).TicketGet(ctx, store.TicketGetParams{CourseID: courseID, ID: ticketID})
	if errors.Is(err, pgx.ErrNoRows) {
		return KnowledgeState{}, notFound()
	}
	if err != nil {
		return KnowledgeState{}, unavailable()
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return KnowledgeState{}, unavailable()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	removed, err := knowledge.Remove(ctx, tx, courseID, knowledge.SourceTicket, ticketID)
	if err != nil {
		return KnowledgeState{}, err
	}
	q := store.New(tx)
	if removed || t.SaveAsKnowledge {
		if _, err := q.TicketClearKnowledge(ctx, store.TicketClearKnowledgeParams{CourseID: courseID, ID: ticketID}); err != nil {
			return KnowledgeState{}, unavailable()
		}
		if err := s.audit(ctx, q, courseID, a.UserID, t.ID, "escalation_ticket.knowledge_removed", t.Status, t.Status); err != nil {
			return KnowledgeState{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return KnowledgeState{}, unavailable()
	}
	return KnowledgeState{}, nil
}

// KnowledgeOf trả trạng thái mục tri thức của ticket (Detail).
func (s *Service) KnowledgeOf(ctx context.Context, courseID, ticketID uuid.UUID) (KnowledgeState, error) {
	row, err := store.New(s.Pool).KnowledgeForTicket(ctx, store.KnowledgeForTicketParams{CourseID: courseID, TicketID: ticketID})
	if errors.Is(err, pgx.ErrNoRows) {
		return KnowledgeState{}, nil
	}
	if err != nil {
		return KnowledgeState{}, unavailable()
	}
	st := KnowledgeState{Saved: true, Loaded: row.Loaded}
	if q, a, ok := strings.Cut(strings.TrimPrefix(row.Text, "Câu hỏi: "), "\nTrả lời: "); ok {
		st.Question, st.Answer = q, a
	}
	return st, nil
}
