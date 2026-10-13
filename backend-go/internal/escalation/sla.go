package escalation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/edupilot/backend-go/internal/workhours"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// SLALeaderKey là khoá leader: một bản worker chạy job (SRS 4.6.2).
const SLALeaderKey = "ep:lock:ticket.sla"

const slaPage = 200

// SLAJob là hai việc nền của ticket: `ticket.sla` (mỗi TICKET_SLA_TICK) và `ticket.autoclose` (mỗi giờ). Một bản nhờ khoá leader Redis.
type SLAJob struct {
	Svc   *Service
	Redis *appredis.Client
	Name  string
	Every time.Duration // TICKET_SLA_TICK
	Log   *slog.Logger
}

// Run chạy tới khi ctx huỷ.
func (j *SLAJob) Run(ctx context.Context) error {
	tick, closeTick := time.NewTicker(j.Every), time.NewTicker(time.Hour)
	defer tick.Stop()
	defer closeTick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			if !j.leader(ctx) {
				continue
			}
			if err := j.Svc.SLATick(ctx); err != nil && j.Log != nil {
				j.Log.ErrorContext(ctx, "ticket.sla lỗi", "error", err.Error())
			}
		case <-closeTick.C:
			if !j.leader(ctx) {
				continue
			}
			if n, err := j.Svc.AutoClose(ctx); err != nil && j.Log != nil {
				j.Log.ErrorContext(ctx, "ticket.autoclose lỗi", "error", err.Error())
			} else if n > 0 && j.Log != nil {
				j.Log.InfoContext(ctx, "ticket.autoclose", "closed", n)
			}
		}
	}
}

func (j *SLAJob) leader(ctx context.Context) bool {
	return j.Redis == nil || j.Redis.Leader(ctx, SLALeaderKey, j.Name, 2*j.Every)
}

// SLATick chạy MỘT vòng `ticket.sla` (SRS 4.6.2). Mỗi bước ghi `sla_stage` có điều kiện TRONG CÙNG giao dịch với sự kiện outbox nên không có "đã gửi mà chưa ghi";
// hai bản chạy song song vẫn mỗi ticket một lần. Mỗi lớp mỗi lượt chạy ≤ 1 sự kiện mỗi giai đoạn.
func (s *Service) SLATick(ctx context.Context) error {
	now := s.now()
	sla := s.Cfg.sla()
	if err := s.advance(ctx, now, 0, 1, sla.RemindAfter, sla.ReminderAt); err != nil {
		return err
	}
	if err := s.advance(ctx, now, 1, 2, sla.EscalateAfter, sla.EscalateAt); err != nil {
		return err
	}
	if err := s.remindClaimed(ctx, now, sla); err != nil {
		return err
	}
	return s.stalledCourses(ctx, now)
}

// advance đưa ticket OPEN từ `from` lên `to` khi đã tới mốc `due(waiting_since)`. Lọc rẻ theo giờ LỊCH (≥ giờ làm việc nên không sót) qua escalation_tickets_sla_idx.
func (s *Service) advance(ctx context.Context, now time.Time, from, to int16, after time.Duration, due func(time.Time) time.Time) error {
	cutoff := now.Add(-after)
	seen := map[uuid.UUID]bool{} // mỗi lớp ≤ 1 sự kiện mỗi lượt chạy
	var cur *store.EscalationTicket
	for {
		arg := store.TicketSLACandidatesParams{Stage: from, Before: cutoff, PageLimit: slaPage}
		if cur != nil {
			arg.CursorAt, arg.CursorID = &cur.WaitingSince, &cur.ID
		}
		rows, err := store.New(s.Pool).TicketSLACandidates(ctx, arg)
		if err != nil {
			return fmt.Errorf("escalation: ứng viên SLA: %w", err)
		}
		for i := range rows {
			t := rows[i]
			cur = &rows[i]
			if at := due(t.WaitingSince); at.IsZero() || at.After(now) {
				continue // chưa tới mốc giờ làm việc (giờ lịch đã đủ nhưng cuối tuần / ngoài khung chưa tính)
			}
			ok, err := s.advanceOne(ctx, t.ID, t.CourseID, from, to, now, !seen[t.CourseID])
			if err != nil {
				return err
			}
			if ok {
				seen[t.CourseID] = true
			}
		}
		if len(rows) < slaPage {
			return nil
		}
	}
}

func (s *Service) advanceOne(ctx context.Context, id, course uuid.UUID, from, to int16, now time.Time, emit bool) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("escalation: mở giao dịch SLA: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	if _, err := q.TicketSLAAdvance(ctx, store.TicketSLAAdvanceParams{ID: id, FromStage: from, ToStage: to, Now: now}); err != nil {
		if isNoRows(err) {
			return false, nil // bản khác đã làm / ticket đã đổi trạng thái
		}
		return false, fmt.Errorf("escalation: cập nhật giai đoạn: %w", err)
	}
	if emit {
		var err error
		if to == 1 {
			_, err = outbox.Write(ctx, tx, TopicSLAReminder, map[string]any{"course_id": course})
		} else {
			_, err = outbox.Write(ctx, tx, TopicSLAEscalated, map[string]any{"course_id": course})
		}
		if err != nil {
			return false, fmt.Errorf("escalation: outbox SLA: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("escalation: commit SLA: %w", err)
	}
	return true, nil
}

// remindClaimed nhắc NGƯỜI NHẬN khi ticket CLAIMED quá ReminderAt(claimed_at) mà chưa trả lời (TICKET_STALE cá nhân, không thư).
// dedupe_key = ticket_stale:<ticket_id>:<unix(claimed_at)>: bỏ nhận rồi nhận lại được nhắc lại.
func (s *Service) remindClaimed(ctx context.Context, now time.Time, sla workhours.SLA) error {
	if s.Notify == nil {
		return nil
	}
	rows, err := store.New(s.Pool).TicketClaimedCandidates(ctx, store.TicketClaimedCandidatesParams{Before: now.Add(-sla.RemindAfter), PageLimit: slaPage})
	if err != nil {
		return fmt.Errorf("escalation: ứng viên CLAIMED: %w", err)
	}
	for _, t := range rows {
		if t.ClaimedBy == nil || t.ClaimedAt == nil {
			continue
		}
		if at := sla.ReminderAt(*t.ClaimedAt); at.IsZero() || at.After(now) {
			continue
		}
		c, err := store.New(s.Pool).GetCourseBasic(ctx, t.CourseID)
		if err != nil {
			return fmt.Errorf("escalation: đọc lớp: %w", err)
		}
		body, link := c.ClassCode, "/inbox?course="+t.CourseID.String()+"&ticket="+t.ID.String()
		if _, err := s.Notify.Send(ctx, notify.Spec{UserID: *t.ClaimedBy, CourseID: &t.CourseID, Type: notify.TypeTicketStale, Title: "Có câu hỏi bạn đã nhận chưa trả lời", Body: &body, Link: &link,
			DedupeKey: fmt.Sprintf("ticket_stale:%s:%d", t.ID, t.ClaimedAt.Unix())}); err != nil {
			return fmt.Errorf("escalation: nhắc người nhận: %w", err)
		}
	}
	return nil
}

// stalledCourses: MỖI tick, lớp có ticket OPEN ở giai đoạn 2 mà không TA / TEACHER nào hoạt động trong TICKET_INACTIVE_AFTER → phát `escalation.course_stalled`
// (handler báo ADMIN, một thông báo mỗi ngày). Đã báo trong ngày thì không phát lại.
func (s *Service) stalledCourses(ctx context.Context, now time.Time) error {
	q := store.New(s.Pool)
	courses, err := q.TicketStalledCourses(ctx)
	if err != nil {
		return fmt.Errorf("escalation: lớp quá hạn: %w", err)
	}
	dayStart := time.Date(now.In(workhours.ICT).Year(), now.In(workhours.ICT).Month(), now.In(workhours.ICT).Day(), 0, 0, 0, 0, workhours.ICT)
	for _, c := range courses {
		active, err := q.TicketCourseHasActivity(ctx, store.TicketCourseHasActivityParams{CourseID: c.CourseID, Since: now.Add(-s.Cfg.inactiveAfter())})
		if err != nil {
			return fmt.Errorf("escalation: kiểm hoạt động: %w", err)
		}
		if active {
			continue
		}
		if done, err := q.NotifyStalledToday(ctx, store.NotifyStalledTodayParams{CourseID: &c.CourseID, Since: dayStart}); err != nil {
			return fmt.Errorf("escalation: kiểm đã báo: %w", err)
		} else if done {
			continue
		}
		admins, err := q.TodayAdminIDs(ctx)
		if err != nil {
			return fmt.Errorf("escalation: admin: %w", err)
		}
		ids := make([]string, len(admins))
		for i, a := range admins {
			ids[i] = a.String()
		}
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("escalation: mở giao dịch: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicCourseStall, map[string]any{"course_id": c.CourseID, "n": c.N, "user_ids": ids}); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("escalation: outbox: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("escalation: commit: %w", err)
		}
	}
	return nil
}

// ---- handler outbox của job (worker) ----

// HandleSLAReminder (`escalation.sla_reminder`): thông báo gộp `TICKET_STALE` theo lớp ("N câu hỏi chờ quá 24 giờ làm việc") + thư `ticket_reminder` tới TA / TEACHER ACTIVE của lớp.
func (n *Notifier) HandleSLAReminder(ctx context.Context, m outbox.Message) error {
	var p courseEvent
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil {
		return fmt.Errorf("payload %s không hợp lệ", m.Topic)
	}
	if err := n.recountStale(ctx, p.CourseID); err != nil {
		return err
	}
	tx, err := n.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("escalation: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := EnqueueReminder(ctx, tx, p.CourseID, n.now()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// HandleSLAEscalated (`escalation.sla_escalated`): đếm lại `TICKET_STALE` (ticket vừa lên giai đoạn 2 vẫn "quá hạn"); mục `Hôm nay` Overdue do provider tự tính.
func (n *Notifier) HandleSLAEscalated(ctx context.Context, m outbox.Message) error {
	var p courseEvent
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil {
		return fmt.Errorf("payload %s không hợp lệ", m.Topic)
	}
	return n.recountStale(ctx, p.CourseID)
}

func (n *Notifier) recountStale(ctx context.Context, courseID uuid.UUID) error {
	q := store.New(n.Pool)
	c, err := q.GetCourseBasic(ctx, courseID)
	if isNoRows(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("escalation: đọc lớp: %w", err)
	}
	staff, err := q.TicketStaffIDs(ctx, courseID)
	if err != nil {
		return fmt.Errorf("escalation: người nhận: %w", err)
	}
	body := c.ClassCode
	return n.Notify.Aggregate(ctx, notify.Agg{
		CourseID: courseID, Type: notify.TypeTicketStale, Recipients: staff, Link: "/inbox?course=" + courseID.String(), Body: &body,
		Title: func(k int) string { return fmt.Sprintf("%d câu hỏi chờ quá 24 giờ làm việc", k) },
		Count: func(ctx context.Context, q *store.Queries) (int, error) {
			k, err := q.TicketOpenAtStage(ctx, store.TicketOpenAtStageParams{CourseID: courseID, Stage: 1})
			return int(k), err
		},
	})
}

// HandleCourseStalled (`escalation.course_stalled`): ADMIN nhận MỘT thông báo `COURSE_TICKETS_STALLED` mỗi ngày (dedupe `stalled:<course_id>:<yyyymmdd>`), chỉ mã lớp + số lượng.
func (n *Notifier) HandleCourseStalled(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID uuid.UUID `json:"course_id"`
		N        int       `json:"n"`
		UserIDs  []string  `json:"user_ids"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil || p.CourseID == uuid.Nil {
		return fmt.Errorf("payload %s không hợp lệ", m.Topic)
	}
	c, err := store.New(n.Pool).GetCourseBasic(ctx, p.CourseID)
	if isNoRows(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("escalation: đọc lớp: %w", err)
	}
	day := n.now().In(workhours.ICT).Format("20060102")
	title := fmt.Sprintf("Lớp %s có %d câu hỏi chờ quá 72 giờ làm việc", c.ClassCode, p.N)
	for _, raw := range p.UserIDs {
		uid, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		if _, err := n.Notify.Send(ctx, notify.Spec{UserID: uid, CourseID: &p.CourseID, Type: notify.TypeCourseTicketsStalled, Title: title, DedupeKey: "stalled:" + p.CourseID.String() + ":" + day}); err != nil {
			return fmt.Errorf("escalation: báo admin: %w", err)
		}
	}
	return nil
}
