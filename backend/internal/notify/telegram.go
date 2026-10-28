// Package notify sends order notifications to the shop owner.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/order"
)

// Telegram posts new orders to a chat through the Bot API.
type Telegram struct {
	token   string
	chatID  string
	baseURL string
	client  *http.Client
}

// NewTelegram creates a notifier. baseURL is empty in production and overridden in tests.
func NewTelegram(token, chatID string, client *http.Client) *Telegram {
	return &Telegram{token: token, chatID: chatID, baseURL: "https://api.telegram.org", client: client}
}

// NewOrder sends a formatted order summary.
func (t *Telegram) NewOrder(ctx context.Context, o order.Order) error {
	body, err := json.Marshal(map[string]any{
		"chat_id":    t.chatID,
		"text":       FormatOrder(o),
		"parse_mode": "HTML",
	})
	if err != nil {
		return fmt.Errorf("encode message: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", t.baseURL, t.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		// The URL contains the bot token; never surface it in logs.
		return fmt.Errorf("send message: %w", redact(err, t.token))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("telegram responded %d: %s", resp.StatusCode, msg)
	}
	return nil
}

var deliveryNames = map[order.DeliveryMethod]string{
	order.DeliveryPickup:  "Самовывоз",
	order.DeliveryCourier: "Курьер",
	order.DeliveryPost:    "Почта",
}

// FormatOrder renders an order as Telegram HTML in Russian (the owner's language).
func FormatOrder(o order.Order) string {
	var b strings.Builder
	e := html.EscapeString

	fmt.Fprintf(&b, "🛍 <b>Новый заказ %s</b>\n\n", e(o.Number))
	fmt.Fprintf(&b, "👤 %s\n📞 %s\n", e(o.Customer.Name), e(o.Customer.Phone))
	fmt.Fprintf(&b, "🚚 %s", deliveryNames[o.Customer.Delivery])
	if o.Customer.Address != "" {
		fmt.Fprintf(&b, ": %s", e(o.Customer.Address))
	}
	b.WriteString("\n\n")

	for _, l := range o.Lines {
		name := l.ProductName
		if l.VariantLabel != "" {
			name += ", " + l.VariantLabel
		}
		fmt.Fprintf(&b, "• %s × %d = %s\n", e(name), l.Quantity, Tenge(l.LineTotal))
	}

	b.WriteString("\n")
	if o.Quote.DiscountTotal > 0 {
		fmt.Fprintf(&b, "Сумма: %s\n", Tenge(o.Quote.Subtotal))
		for _, d := range o.Quote.Discounts {
			fmt.Fprintf(&b, "Скидка (%s): −%s\n", e(d.Label), Tenge(d.Amount))
		}
	}
	fmt.Fprintf(&b, "<b>Итого: %s</b>", Tenge(o.Quote.Total))

	if o.Customer.Comment != "" {
		fmt.Fprintf(&b, "\n\n💬 %s", e(o.Customer.Comment))
	}
	return b.String()
}

// Tenge formats an amount as "12 500 ₸".
func Tenge(amount int64) string {
	sign := ""
	if amount < 0 {
		sign, amount = "-", -amount
	}
	s := fmt.Sprint(amount)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return sign + b.String() + " ₸"
}

func redact(err error, secret string) error {
	if secret == "" {
		return err
	}
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), secret, "***"))
}
