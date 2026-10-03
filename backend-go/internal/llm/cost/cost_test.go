package cost_test

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/llm/cost"
)

func TestCostOf(t *testing.T) {
	t.Parallel()
	d := decimal.RequireFromString
	cases := []struct {
		name         string
		in, out      int
		pin, pout    string
		want, wantUn string
		units        int64
	}{
		{"không token", 0, 0, "4000", "16000", "0", "0", 0},
		{"gpt-4o-mini cỡ 1 triệu", 1_000_000, 1_000_000, "4000", "16000", "20000", "20000", 200_000_000},
		{"nhỏ", 1200, 300, "4000", "16000", "9.6", "9.6", 96_000},
		{"làm tròn lên đơn vị", 1, 0, "1", "0", "0.000001", "0.0001", 1},
		{"giá 0", 5000, 5000, "0", "0", "0", "0", 0},
	}
	for _, tc := range cases {
		got := cost.Of(tc.in, tc.out, d(tc.pin), d(tc.pout))
		if !got.Equal(d(tc.want)) {
			t.Errorf("%s: Of = %s, muốn %s", tc.name, got, tc.want)
		}
		if u := cost.Units(got); u != tc.units {
			t.Errorf("%s: Units = %d, muốn %d", tc.name, u, tc.units)
		}
	}
	if !cost.FromUnits(96_000).Equal(d("9.6")) {
		t.Errorf("FromUnits sai")
	}
}
