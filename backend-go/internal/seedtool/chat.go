package seedtool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ChatPlan là kế hoạch lịch sử chat riêng của seed (`scripts/seed.mjs` bước 12).
// Lý do không đi qua luồng chat thật: với `LLM_PROVIDER=fake` vectơ nhúng là băm của cả chuỗi (không có ngữ nghĩa), nên MỌI câu hỏi tự nhiên đều
// không có ngữ cảnh → độ tin cậy thấp → `ShouldEscalate` tự mở một ticket mỗi câu (126 ticket thay vì 5, BUG-S1). Lịch sử được ghi thẳng, KHÔNG tạo ticket.
type ChatPlan struct {
	Sessions []ChatSession `json:"sessions"`
}

// ChatSession là một phiên của một sinh viên của lớp; Key ổn định cho idempotency.
type ChatSession struct {
	Email string     `json:"email"`
	Key   string     `json:"key"`
	Turns []ChatTurn `json:"turns"`
}

// ChatTurn là một cặp hỏi / đáp. Outside: câu ngoài tài liệu (trợ lý trả "chưa tìm thấy", không có ngữ cảnh).
type ChatTurn struct {
	Key      string `json:"key"`
	Question string `json:"q"`
	Answer   string `json:"a"`
	Outside  bool   `json:"outside"`
}

// ParseChatPlan đọc JSON; khoá lạ bị từ chối.
func ParseChatPlan(raw []byte) (ChatPlan, error) {
	var p ChatPlan
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return p, fmt.Errorf("chat_plan: %w", err)
	}
	return p, nil
}

func detID(key string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("edupilot-seed-chat/"+key))
}

func hash32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

// SeedChat ghi phiên và tin chat của kế hoạch cho sinh viên ACTIVE của lớp; mốc thời gian rải theo băm của khoá trong `weeks` tuần tới `now`
// (một phiên nằm gọn trong một ngày; tin hỏi trước tin đáp 3 giây, các cặp cách nhau 4 phút). Chạy lại không thêm gì (id suy từ khoá). Trả số tin đã thêm.
func SeedChat(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, weeks int, p ChatPlan, now time.Time) (int64, error) {
	if weeks < 1 || weeks > 52 {
		return 0, errors.New("--weeks phải từ 1 đến 52")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var added int64
	for _, s := range p.Sessions {
		var uid uuid.UUID
		if err := tx.QueryRow(ctx, `select e.user_id from enrollments e join users u on u.id = e.user_id
			where e.course_id = $1 and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE' and u.email = $2`, courseID, s.Email).Scan(&uid); err != nil {
			return 0, fmt.Errorf("sinh viên %s không ACTIVE ở lớp: %w", s.Email, err)
		}
		if len(s.Turns) == 0 {
			continue
		}
		start := now.Add(-time.Duration(hash32(s.Key)%uint32(weeks*7))*24*time.Hour - time.Duration(1+hash32(s.Key+"h")%20)*time.Hour) //nolint:gosec // ≤ 52·7
		sid := detID(s.Key)
		title := s.Turns[0].Question
		if utf8.RuneCountInString(title) > 100 {
			title = string([]rune(title)[:100])
		}
		last := start
		for i, t := range s.Turns {
			at := start.Add(time.Duration(i) * 4 * time.Minute)
			last = at.Add(3 * time.Second)
			if i == 0 {
				if _, err := tx.Exec(ctx, `insert into chat_sessions (id, course_id, user_id, title, created_at, updated_at, last_message_at) values ($1, $2, $3, $4, $5, $5, $5) on conflict (id) do nothing`, sid, courseID, uid, title, at); err != nil {
					return 0, fmt.Errorf("ghi phiên: %w", err)
				}
			}
			qid, aid := detID(t.Key+"/q"), detID(t.Key+"/a")
			tag, err := tx.Exec(ctx, `insert into chat_messages (id, course_id, session_id, user_id, role, content, client_msg_id, completed_at, created_at, updated_at)
				values ($1, $2, $3, $4, 'USER', $5, $1, $6, $6, $6) on conflict (id) do nothing`, qid, courseID, sid, uid, t.Question, at)
			if err != nil {
				return 0, fmt.Errorf("ghi câu hỏi: %w", err)
			}
			if tag.RowsAffected() == 0 {
				continue // đã có từ lần chạy trước
			}
			conf, low := "0.900", false
			if t.Outside {
				conf, low = "0.150", true
			}
			if _, err := tx.Exec(ctx, `insert into chat_messages (id, course_id, session_id, user_id, role, content, reply_to, intent, confidence, low_confidence, no_context, completed_at, created_at, updated_at)
				values ($1, $2, $3, $4, 'ASSISTANT', $5, $6, 'COURSE_QA', $7::numeric, $8, $8, $9, $9, $9)`, aid, courseID, sid, uid, t.Answer, qid, conf, low, last); err != nil {
				return 0, fmt.Errorf("ghi câu trả lời: %w", err)
			}
			added += 2
		}
		if _, err := tx.Exec(ctx, `update chat_sessions set last_message_at = greatest(last_message_at, $2), updated_at = greatest(updated_at, $2) where id = $1`, sid, last); err != nil {
			return 0, fmt.Errorf("cập nhật phiên: %w", err)
		}
	}
	return added, tx.Commit(ctx)
}
