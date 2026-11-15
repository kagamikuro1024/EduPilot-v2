package insight

import "math"

// Item: một câu hỏi đã che + vectơ (chuẩn hoá L2).
type Item struct {
	Vec []float32
}

// Cluster gom liên kết trung bình (UPGMA, thuật toán chuỗi láng giềng gần nhất): hai cụm hợp nhất khi cosine TRUNG BÌNH giữa mọi cặp câu của chúng ≥ tau.
// Không phụ thuộc thứ tự đầu vào (gom tham lam theo tâm chạy cho số cụm của cùng một chủ đề từ 1 đến 10 tuỳ thứ tự câu, đo trên 85 câu thật, P-11-49).
// Trả mỗi cụm là danh sách chỉ số tăng dần; các cụm xếp theo chỉ số nhỏ nhất. Vectơ 0 luôn là cụm riêng (cosine 0).
// ponytail: O(n²) thời gian, ma trận khoảng cách float32 n²/2 phần tử (3.000 câu ≈ 18 MB; 6.000 ≈ 72 MB); thay bằng HNSW nếu một lớp vượt hàng chục nghìn câu.
func Cluster(vecs [][]float32, tau float64) [][]int {
	n := len(vecs)
	if n == 0 {
		return nil
	}
	unit := make([][]float32, n)
	for i, v := range vecs {
		var nn float64
		for _, x := range v {
			nn += float64(x) * float64(x)
		}
		u := make([]float32, len(v))
		if nn > 0 {
			inv := float32(1 / math.Sqrt(nn))
			for k, x := range v {
				u[k] = x * inv
			}
		}
		unit[i] = u
	}
	at := func(i, j int) int { // ô (i, j), i < j, trong nửa trên của ma trận
		return i*n - i*(i+1)/2 + (j - i - 1)
	}
	dist := make([]float32, n*(n-1)/2)
	for i := range n {
		for j := i + 1; j < n; j++ {
			var dot float32
			for k, x := range unit[i] {
				dot += x * unit[j][k]
			}
			dist[at(i, j)] = 1 - dot
		}
	}
	d := func(i, j int) float32 {
		if i > j {
			i, j = j, i
		}
		return dist[at(i, j)]
	}
	size, alive := make([]int, n), make([]bool, n)
	parent := make([]int, n) // hợp nhất ≤ ngưỡng: gộp thành viên (union-find trên chỉ số gốc của ô)
	for i := range n {
		size[i], alive[i], parent[i] = 1, true, i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i], i = parent[parent[i]], parent[i]
		}
		return i
	}
	cut := float32(1 - tau)
	var chain []int
	for left := n; left > 1; {
		if len(chain) == 0 {
			for i := range n {
				if alive[i] {
					chain = append(chain, i)
					break
				}
			}
		}
		a := chain[len(chain)-1]
		prev := -1
		if len(chain) > 1 {
			prev = chain[len(chain)-2]
		}
		b, bd := -1, float32(math.MaxFloat32)
		for k := range n {
			if !alive[k] || k == a {
				continue
			}
			if dk := d(a, k); dk < bd || (dk == bd && k == prev) {
				b, bd = k, dk
			}
		}
		if b != prev {
			chain = append(chain, b)
			continue
		}
		chain = chain[:len(chain)-2] // a, b là láng giềng gần nhất của nhau: hợp nhất vào ô nhỏ hơn
		lo, hi := min(a, b), max(a, b)
		for k := range n {
			if alive[k] && k != lo && k != hi {
				dist[at(min(lo, k), max(lo, k))] = (float32(size[lo])*d(lo, k) + float32(size[hi])*d(hi, k)) / float32(size[lo]+size[hi])
			}
		}
		if bd <= cut { // liên kết trung bình đơn điệu: các lần hợp nhất ≤ ngưỡng chính là phép cắt cây ở ngưỡng
			parent[find(hi)] = find(lo)
		}
		size[lo] += size[hi]
		alive[hi] = false
		left--
	}
	idx := map[int]int{}
	var out [][]int
	for i := range n {
		r := find(i)
		g, ok := idx[r]
		if !ok {
			g = len(out)
			idx[r] = g
			out = append(out, nil)
		}
		out[g] = append(out[g], i)
	}
	return out
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
