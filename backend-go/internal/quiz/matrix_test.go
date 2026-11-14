package quiz_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/quiz"
)

// pool80 là ngân hàng mẫu: 8 chủ đề × 10 câu, độ khó EASY / MEDIUM / HARD xoay vòng.
func pool80() []quiz.Candidate {
	var p []quiz.Candidate
	for tp := 0; tp < 8; tp++ {
		for i := 0; i < 10; i++ {
			p = append(p, quiz.Candidate{ID: fmt.Sprintf("q-%d-%02d", tp, i), Topic: fmt.Sprintf("Chủ đề %d", tp), Difficulty: []string{"EASY", "MEDIUM", "HARD"}[i%3]})
		}
	}
	return p
}

func distinct(ids []string) bool {
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

// TestBuildMatrix — US-P9-05 AC1–AC5: đủ câu, không lặp, tất định, ưu tiên sai / chưa làm, bù ô thiếu, ngắn hơn kèm notes, xáo ổn định.
func TestBuildMatrix(t *testing.T) {
	t.Parallel()
	pool := pool80()
	m := []quiz.Cell{{Topic: "Chủ đề 0", Difficulty: "EASY", Count: 2}, {Topic: "Chủ đề 1", Count: 5}, {Topic: "Chủ đề 2", Difficulty: "HARD", Count: 3}}

	t.Run("đúng số câu, không lặp, tất định", func(t *testing.T) {
		a, notes := quiz.BuildMatrix(pool, m, quiz.History{}, 7)
		b, _ := quiz.BuildMatrix(pool, m, quiz.History{}, 7)
		require.Len(t, a, 10)
		require.Empty(t, notes)
		require.True(t, distinct(a))
		require.Equal(t, a, b, "cùng seed → cùng đề")
		c, _ := quiz.BuildMatrix(pool, m, quiz.History{}, 8)
		require.NotEqual(t, a, c, "seed khác → đề khác")
	})
	t.Run("ô khớp chủ đề và độ khó", func(t *testing.T) {
		ids, _ := quiz.BuildMatrix(pool, []quiz.Cell{{Topic: "Chủ đề 3", Difficulty: "MEDIUM", Count: 3}}, quiz.History{}, 1)
		byID := map[string]quiz.Candidate{}
		for _, c := range pool {
			byID[c.ID] = c
		}
		for _, id := range ids {
			require.Equal(t, "Chủ đề 3", byID[id].Topic)
			require.Equal(t, "MEDIUM", byID[id].Difficulty)
		}
	})
	t.Run("ưu tiên câu sai, rồi chưa làm, cuối cùng câu vừa đúng", func(t *testing.T) {
		cell := []quiz.Cell{{Topic: "Chủ đề 4", Count: 4}}
		h := quiz.History{Wrong: map[string]bool{"q-4-00": true, "q-4-01": true}, Right: map[string]bool{}}
		for i := 2; i < 10; i++ { // 8 câu còn lại: 6 vừa đúng, 2 chưa làm
			if i >= 4 {
				h.Right[fmt.Sprintf("q-4-%02d", i)] = true
			}
		}
		ids, _ := quiz.BuildMatrix(pool, cell, h, 3)
		require.ElementsMatch(t, []string{"q-4-00", "q-4-01", "q-4-02", "q-4-03"}, ids)
	})
	t.Run("bù độ khó gần nhất cùng chủ đề rồi chủ đề khác", func(t *testing.T) {
		p := []quiz.Candidate{{ID: "a", Topic: "X", Difficulty: "EASY"}, {ID: "b", Topic: "X", Difficulty: "MEDIUM"}, {ID: "c", Topic: "X", Difficulty: "HARD"}, {ID: "d", Topic: "Y", Difficulty: "HARD"}}
		ids, notes := quiz.BuildMatrix(p, []quiz.Cell{{Topic: "X", Difficulty: "HARD", Count: 3}}, quiz.History{}, 1)
		require.Equal(t, []string{"c", "b", "a"}, ids, "HARD trước, rồi MEDIUM (gần), rồi EASY")
		require.Empty(t, notes)
		ids, _ = quiz.BuildMatrix(p, []quiz.Cell{{Topic: "X", Difficulty: "HARD", Count: 4}}, quiz.History{}, 1)
		require.Len(t, ids, 4)
		require.True(t, distinct(ids))
	})
	t.Run("thiếu câu: đề ngắn hơn kèm notes, không lỗi", func(t *testing.T) {
		p := []quiz.Candidate{{ID: "a", Topic: "X", Difficulty: "EASY"}, {ID: "b", Topic: "X", Difficulty: "EASY"}}
		ids, notes := quiz.BuildMatrix(p, []quiz.Cell{{Topic: "X", Count: 10}}, quiz.History{}, 1)
		require.Len(t, ids, 2)
		require.Equal(t, []string{"thiếu câu chủ đề X"}, notes)
		ids, notes = quiz.BuildMatrix(nil, []quiz.Cell{{Topic: "X", Count: 3}}, quiz.History{}, 1)
		require.Empty(t, ids)
		require.Len(t, notes, 1)
	})
	t.Run("hai ô cùng chủ đề không lặp câu", func(t *testing.T) {
		ids, _ := quiz.BuildMatrix(pool, []quiz.Cell{{Topic: "Chủ đề 5", Difficulty: "EASY", Count: 3}, {Topic: "Chủ đề 5", Count: 7}}, quiz.History{}, 9)
		require.Len(t, ids, 10)
		require.True(t, distinct(ids))
	})
	t.Run("dựng 20 câu từ 80 câu nhanh", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 100; i++ {
			ids, _ := quiz.BuildMatrix(pool, []quiz.Cell{{Count: 20}}, quiz.History{}, uint64(i))
			require.Len(t, ids, 20)
		}
		require.Less(t, time.Since(start), 3*time.Second, "100 lần dựng")
	})
}

func TestShuffleStable(t *testing.T) {
	t.Parallel()
	mk := func() []int { return []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} }
	a, b, c := mk(), mk(), mk()
	quiz.Shuffle(a, quiz.SeedOf("attempt-1"))
	quiz.Shuffle(b, quiz.SeedOf("attempt-1"))
	quiz.Shuffle(c, quiz.SeedOf("attempt-2"))
	require.Equal(t, a, b, "ổn định trong lượt")
	require.NotEqual(t, a, c, "xáo riêng từng lượt")
	require.ElementsMatch(t, mk(), a)
}
