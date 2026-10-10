package calendar_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
)

func tcOf(f *fx) agent.TrustedContext {
	return agent.TrustedContext{UserID: f.sv, CourseID: f.course, Role: "STUDENT"}
}

func run(t *testing.T, tool agent.Tool, tc agent.TrustedContext, args string) agent.Result {
	t.Helper()
	r, err := tool.Run(context.Background(), tc, json.RawMessage(args))
	require.NoError(t, err)
	return r
}

// TestExamScheduleTool — AC10: bài thi đã lên lịch + sự kiện EXAM từ nay; không có buổi học / sự kiện OTHER / bài DRAFT / việc đã qua.
func TestExamScheduleTool(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.exam(f.course, "Thi giữa kỳ", "SCHEDULED", 48*time.Hour)
	f.exam(f.course, "Nháp", "DRAFT", 0)
	f.event(f.course, "EXAM", "Thi thực hành", 72*time.Hour)
	f.event(f.course, "OTHER", "Nộp báo cáo", 24*time.Hour)
	f.session(f.course, 1, 12*time.Hour, "Buổi")
	f.event(f.course, "EXAM", "Đã qua", -48*time.Hour)
	r := run(t, agent.NewExamScheduleTool(f.svc), tcOf(f), `{}`)
	require.False(t, r.NoData)
	evs := r.Facts["events"].([]map[string]any)
	require.Len(t, evs, 2)
	require.Equal(t, "Thi giữa kỳ", evs[0]["title"])
	require.Equal(t, "Thi thực hành", evs[1]["title"])
}

// TestUpcomingEventsTool — AC10: mọi nguồn trong `days` ngày tới, theo thứ tự thời gian.
func TestUpcomingEventsTool(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.session(f.course, 1, 48*time.Hour, "Mật mã")
	f.event(f.course, "OTHER", "Nộp báo cáo", 24*time.Hour)
	f.event(f.course, "OTHER", "Xa", 10*24*time.Hour)
	r := run(t, agent.NewUpcomingEventsTool(f.svc), tcOf(f), `{"days":3}`)
	evs := r.Facts["events"].([]map[string]any)
	require.Len(t, evs, 2)
	require.Equal(t, "Nộp báo cáo", evs[0]["title"])
	require.Equal(t, "Buổi 1 · Mật mã", evs[1]["title"])
	require.Equal(t, 3, r.Facts["days"])
}

// TestUpcomingEventsDaysClamped — AC10: days 60 → 30, ≤ 0 → 7, không lỗi.
func TestUpcomingEventsDaysClamped(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.event(f.course, "OTHER", "Ngày 25", 25*24*time.Hour)
	f.event(f.course, "OTHER", "Ngày 40", 40*24*time.Hour)
	f.event(f.course, "OTHER", "Ngày 5", 5*24*time.Hour)
	tool := agent.NewUpcomingEventsTool(f.svc)
	r := run(t, tool, tcOf(f), `{"days":60}`)
	require.Equal(t, 30, r.Facts["days"])
	require.Len(t, r.Facts["events"], 2)
	for _, a := range []string{`{"days":0}`, `{"days":-5}`, `{}`} {
		r = run(t, tool, tcOf(f), a)
		require.Equal(t, 7, r.Facts["days"], a)
		require.Len(t, r.Facts["events"], 1, a)
	}
}

// TestCalendarToolsScopedToCourse — AC10: chỉ lớp của trusted_context; tool không có tham số danh tính.
func TestCalendarToolsScopedToCourse(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	other := f.newCourse("ACTIVE")
	f.exam(other, "Thi lớp khác", "SCHEDULED", 24*time.Hour)
	f.event(other, "OTHER", "Sự kiện lớp khác", time.Hour)
	f.event(f.course, "OTHER", "Của lớp này", time.Hour)
	r := run(t, agent.NewUpcomingEventsTool(f.svc), tcOf(f), `{"days":7}`)
	require.Contains(t, fmt.Sprint(r.Facts), "Của lớp này")
	require.NotContains(t, fmt.Sprint(r.Facts), "lớp khác")
	require.True(t, run(t, agent.NewExamScheduleTool(f.svc), tcOf(f), `{}`).NoData)
	for _, tool := range []agent.Tool{agent.NewExamScheduleTool(f.svc), agent.NewUpcomingEventsTool(f.svc)} {
		for i := range tool.ArgsType().NumField() {
			n := tool.ArgsType().Field(i).Name
			require.NotContains(t, []string{"UserID", "StudentCode", "CourseID"}, n)
		}
	}
}

// TestCalendarToolsNoData — AC10: lịch rỗng → NoData với đúng câu mẫu.
func TestCalendarToolsNoData(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	r := run(t, agent.NewExamScheduleTool(f.svc), tcOf(f), `{}`)
	require.True(t, r.NoData)
	require.Equal(t, "Chưa có lịch thi nào được công bố.", r.Message)
	r = run(t, agent.NewUpcomingEventsTool(f.svc), tcOf(f), `{"days":14}`)
	require.True(t, r.NoData)
	require.Equal(t, "Bạn không có sự kiện nào trong 14 ngày tới.", r.Message)
	r = run(t, agent.NewUpcomingEventsTool(f.svc), tcOf(f), `{"days":90}`)
	require.Equal(t, "Bạn không có sự kiện nào trong 30 ngày tới.", r.Message)
}
