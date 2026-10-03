package llm

import "context"

// NewStaticRegistry dựng Registry chỉ-bộ-nhớ cho test (không DB, không env).
func NewStaticRegistry(routes map[Task]Route) *Registry {
	r := &Registry{}
	r.cur.Store(&regState{routes: routes})
	return r
}

// SetRoute đặt tuyến của một tác vụ (test).
func (r *Registry) SetRoute(t Task, rt Route) {
	st := r.cur.Load()
	next := &regState{routes: map[Task]Route{}, envOnly: st.envOnly}
	for k, v := range st.routes {
		next.routes[k] = v
	}
	next.routes[t] = rt
	r.cur.Store(next)
}

// FlushAudit đẩy hết dòng audit đang đệm (test).
func (g *Gateway) FlushAudit(ctx context.Context) {
	if g.o.Auditor != nil {
		g.o.Auditor.flush(ctx)
	}
}

// SetStaticForTest khởi tạo Registry rỗng và đặt một tuyến (test).
func (r *Registry) SetStaticForTest(t Task, rt Route) {
	if r.cur.Load() == nil {
		r.cur.Store(&regState{routes: map[Task]Route{}})
	}
	r.SetRoute(t, rt)
}
