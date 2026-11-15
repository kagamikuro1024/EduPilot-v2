// QC2 TC-PE05-19: cài đặt độc lập SRS 4.3.2 (không dùng hàm của dev). stdin: JSON {attempt_id, shuffle_q, shuffle_o, items:[{id, opts:[{id,pinned}]}]} (đúng thứ tự position gốc); stdout: thứ tự dự đoán.
// k = chỉ số 0 của mục trong danh sách gốc theo position (SRS: "thứ tự đáp án của câu k").
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"

)

type Opt struct {
	ID     string `json:"id"`
	Pinned bool   `json:"pinned"`
}
type Item struct {
	ID   string `json:"id"`
	Opts []Opt  `json:"opts"`
}
type In struct {
	AttemptID string `json:"attempt_id"`
	ShuffleQ  bool   `json:"shuffle_q"`
	ShuffleO  bool   `json:"shuffle_o"`
	Items     []Item `json:"items"`
}

func main() {
	var in In
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		panic(err)
	}
	u, err := hex.DecodeString(strings.ReplaceAll(in.AttemptID, "-", ""))
	if err != nil || len(u) != 16 {
		panic("attempt_id không phải uuid")
	}
	seed := sha256.Sum256(u)
	h1, h2 := binary.BigEndian.Uint64(seed[0:8]), binary.BigEndian.Uint64(seed[8:16])
	n := len(in.Items)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	if in.ShuffleQ {
		r := rand.New(rand.NewPCG(h1, h2))
		for i := n - 1; i > 0; i-- {
			j := r.IntN(i + 1)
			order[i], order[j] = order[j], order[i]
		}
	}
	out := map[string]any{}
	var items []string
	opts := map[string][]string{}
	for _, k := range order {
		it := in.Items[k]
		items = append(items, it.ID)
		var free, pinned []Opt
		for _, o := range it.Opts {
			if o.Pinned {
				pinned = append(pinned, o)
			} else {
				free = append(free, o)
			}
		}
		if in.ShuffleO {
			r := rand.New(rand.NewPCG(h1^uint64(k+1)*0x9E3779B97F4A7C15, h2))
			for i := len(free) - 1; i > 0; i-- {
				j := r.IntN(i + 1)
				free[i], free[j] = free[j], free[i]
			}
		}
		for _, o := range append(free, pinned...) {
			opts[it.ID] = append(opts[it.ID], o.ID)
		}
	}
	out["items"], out["opts"] = items, opts
	_ = fmt.Sprint
	json.NewEncoder(os.Stdout).Encode(out)
}
