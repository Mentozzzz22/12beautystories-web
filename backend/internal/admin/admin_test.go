package admin_test

import (
	"testing"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/admin"
)

func TestSalePrice(t *testing.T) {
	tests := []struct {
		price   int64
		percent int
		want    int64
		ok      bool
	}{
		{12700, 30, 8890, true},
		{73100, 25, 54830, true}, // 54 825 rounds half up
		{49130, 15, 41760, true}, // 41 760.5 rounds down to tens
		{100, 1, 100, false},     // rounds back to the regular price
		{0, 10, 0, false},
	}
	for _, tt := range tests {
		got, ok := admin.SalePrice(tt.price, tt.percent)
		if got != tt.want || ok != tt.ok {
			t.Errorf("SalePrice(%d, %d) = %d, %v; want %d, %v", tt.price, tt.percent, got, ok, tt.want, tt.ok)
		}
	}
}
