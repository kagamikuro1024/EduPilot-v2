package llm

// SetStaticForTest khởi tạo Registry rỗng và đặt một tuyến (test).
func (r *Registry) SetStaticForTest(t Task, rt Route) {
	if r.cur.Load() == nil {
		r.cur.Store(&regState{routes: map[Task]Route{}})
	}
	r.SetRoute(t, rt)
}
