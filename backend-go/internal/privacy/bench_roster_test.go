package privacy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// BenchmarkDetectRoster1k — DEF-R106S-18 / TC-P11-07: roster 1.000 tên (2–4 âm tiết) × văn bản 5.000 ký tự không chứa tên nào: `Detect` ≤ +20 % so với trước P-106-11.
func BenchmarkDetectRoster1k(b *testing.B) {
	ho := []string{"Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Vũ", "Đặng", "Bùi", "Đỗ", "Ngô"}
	dem := []string{"Văn", "Thị", "Minh", "Quốc", "Ngọc", "Hữu", "Đức", "Thanh"}
	ten := []string{"An", "Bình", "Cường", "Dũng", "Giang", "Hà", "Khoa", "Lan", "Nam", "Phúc", "Quân", "Sơn", "Trang", "Vy", "Yến"}
	var members []Member
	for i := range 1000 {
		name := ho[i%len(ho)] + " " + dem[(i/len(ho))%len(dem)] + " " + ten[(i/3)%len(ten)]
		if i%4 == 0 {
			name = ho[i%len(ho)] + " " + ten[(i/3)%len(ten)]
		}
		members = append(members, Member{Name: name, Code: fmt.Sprintf("ZZ9988%04d", i)})
	}
	d, _ := newDetector(map[uuid.UUID][]Member{courseA: members})
	// văn bản dày âm tiết trùng tên (xấu nhất cho find): chỉ gồm âm tiết của roster, sắp theo cấp số nhân nên hiếm khi trùng một tên
	syl := append(append(append([]string{}, ho...), dem...), ten...)
	var sb strings.Builder
	for i := 0; sb.Len() < 5000; i++ {
		sb.WriteString(strings.ToLower(syl[(i*7)%len(syl)]))
		sb.WriteString(" ")
	}
	text := sb.String()[:5000]
	b.ResetTimer()
	for range b.N {
		if _, err := d.Detect(context.Background(), courseA, text); err != nil {
			b.Fatal(err)
		}
	}
}
