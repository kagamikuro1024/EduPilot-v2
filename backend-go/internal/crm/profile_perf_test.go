package crm_test

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestProfileUnder3s — US-P5-04 AC3 vế hồ sơ: trên dữ liệu SRS 9.4 (lớp 1.000 và lớp 100 sinh viên) mở hồ sơ một sinh viên (kèm một dòng audit) ≪ 3 s,
// và ở lớp 100 các truy vấn đọc của hồ sơ không có Seq Scan trên năm bảng lớn.
func TestProfileUnder3s(t *testing.T) {
	r := newRig(t)
	big, small := r.bigData(t)
	actor := r.addUser(uniq("gv"), "TEACHER", "ACTIVE")
	svc := r.svc()
	for name, course := range map[string]uuid.UUID{"lớp 1.000": big, "lớp 100": small} {
		var student uuid.UUID
		require.NoError(t, r.pool.QueryRow(t.Context(), `select student_id from v_student_360 where course_id = $1 order by student_id limit 1 offset 50`, course).Scan(&student), name)
		t0 := time.Now()
		_, err := svc.Profile(t.Context(), actor.ID, course, student)
		require.NoError(t, err, name)
		require.Less(t, time.Since(t0), 3*time.Second, name)
		require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'student_profile' and entity_id = $1 and course_id = $2`, student.String(), course), name+": đúng một dòng audit")
		if course != small {
			continue
		}
		touched := 0
		for _, q := range []string{"CrmProfileGet", "CrmProfileStudyWeeks"} {
			for _, sn := range r.explain(q) {
				if slices.Contains(bigTables, sn[1]) {
					touched++
					require.NotEqual(t, "Seq Scan", sn[0], "%s: Seq Scan trên %s", q, sn[1])
				}
			}
		}
		require.Positive(t, touched, "kế hoạch phải chạm các bảng lớn (kiểm EXPLAIN đúng câu)")
	}
}
