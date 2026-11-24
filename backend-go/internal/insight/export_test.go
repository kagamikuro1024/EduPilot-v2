package insight

import (
	"context"

	"github.com/google/uuid"
)

// Generate đồng bộ (không qua hàng đợi) cho test.
func (s *Service) RunNow(ctx context.Context, owner, course uuid.UUID, days int) (any, error) {
	return s.generate(ctx, owner, payload{CourseID: course, PeriodDays: days}, func(int, string) {})
}
