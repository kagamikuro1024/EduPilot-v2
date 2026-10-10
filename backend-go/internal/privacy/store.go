package privacy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/store"
)

// StoreRoster nạp roster từ Postgres: sinh viên ACTIVE của lớp.
type StoreRoster struct{ Pool *pgxpool.Pool }

// Members cài RosterSource.
func (s StoreRoster) Members(ctx context.Context, courseID uuid.UUID) ([]Member, error) {
	rows, err := store.New(s.Pool).PrivacyRoster(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("truy vấn roster: %w", err)
	}
	ms := make([]Member, len(rows))
	for i, r := range rows {
		ms[i] = Member{Name: r.FullName}
		if r.StudentCodeSnapshot != nil {
			ms[i].Code = *r.StudentCodeSnapshot
		}
	}
	return ms, nil
}
