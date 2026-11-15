package insight

import "math"

// Item: một câu hỏi đã che + vectơ (chuẩn hoá L2).
type Item struct {
	Vec []float32
}

// Cluster gom tham lam theo cosine: duyệt theo thứ tự, câu vào cụm có tâm gần nhất nếu cosine ≥ tau, không thì mở cụm mới; tâm cập nhật theo trung bình chạy.
// Tất định theo thứ tự đầu vào. Trả mỗi cụm là danh sách chỉ số câu. ponytail: O(n·k), đủ cho ≤ vài nghìn câu / lớp; thay bằng HNSW / k-means nếu cần.
func Cluster(vecs [][]float32, tau float64) [][]int {
	var cents [][]float64
	var members [][]int
	for i, v := range vecs {
		best, bs := -1, -2.0
		for c, ct := range cents {
			if s := cosine(v, ct); s > bs {
				best, bs = c, s
			}
		}
		if best < 0 || bs < tau {
			c := make([]float64, len(v))
			for k, x := range v {
				c[k] = float64(x)
			}
			cents, members = append(cents, c), append(members, []int{i})
			continue
		}
		n := float64(len(members[best]))
		for k, x := range v {
			cents[best][k] = (cents[best][k]*n + float64(x)) / (n + 1)
		}
		members[best] = append(members[best], i)
	}
	return members
}

func cosine[T float32 | float64](a []float32, b []T) float64 {
	var dot, na, nb float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// Centroid trả tâm (trung bình) của các vectơ được chọn.
func Centroid(vecs [][]float32, idx []int) []float32 {
	if len(idx) == 0 {
		return nil
	}
	c := make([]float32, len(vecs[idx[0]]))
	for _, i := range idx {
		for k, x := range vecs[i] {
			c[k] += x
		}
	}
	for k := range c {
		c[k] /= float32(len(idx))
	}
	return c
}
