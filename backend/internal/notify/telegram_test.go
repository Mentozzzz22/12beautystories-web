package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/order"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/pricing"
)

func sampleOrder() order.Order {
	return order.Order{
		Number: "BS-00042",
		Customer: order.Customer{
			Name: "Айгерим <script>", Phone: "+77011234567",
			Delivery: order.DeliveryCourier, Address: "Астана, Мангилик Ел 10",
		},
		Lines: []order.Line{{ProductName: "Energizing Shampoo", VariantLabel: "250 ml", Quantity: 2, LineTotal: 50_000}},
		Quote: pricing.Quote{
			Subtotal: 50_000, DiscountTotal: 5_000, Total: 45_000,
			Discounts: []pricing.Discount{{Source: pricing.SourceRule, Label: "Davines -10%", Amount: 5_000}},
		},
	}
}

func TestFormatOrder(t *testing.T) {
	got := FormatOrder(sampleOrder())

	for _, want := range []string{
		"Новый заказ BS-00042",
		"Айгерим &lt;script&gt;",
		"Курьер: Астана, Мангилик Ел 10",
		"Energizing Shampoo, 250 ml × 2 = 50 000 ₸",
		"Скидка (Davines -10%): −5 000 ₸",
		"<b>Итого: 45 000 ₸</b>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("message does not contain %q:\n%s", want, got)
		}
	}
}

func TestTenge(t *testing.T) {
	tests := map[int64]string{0: "0 ₸", 999: "999 ₸", 1000: "1 000 ₸", 1234567: "1 234 567 ₸", -5000: "-5 000 ₸"}
	for in, want := range tests {
		if got := Tenge(in); got != want {
			t.Errorf("Tenge(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestTelegramNewOrder(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botTOKEN/sendMessage" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := NewTelegram("TOKEN", "123", srv.Client())
	tg.baseURL = srv.URL

	if err := tg.NewOrder(context.Background(), sampleOrder()); err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if got["chat_id"] != "123" || got["parse_mode"] != "HTML" {
		t.Errorf("unexpected payload: %v", got)
	}
}

func TestTelegramNewOrderErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"ok":false}`, http.StatusBadRequest)
	}))
	defer srv.Close()

	tg := NewTelegram("TOKEN", "123", srv.Client())
	tg.baseURL = srv.URL

	err := tg.NewOrder(context.Background(), sampleOrder())
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("NewOrder() error = %v, want status 400", err)
	}
}
