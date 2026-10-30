// Package admin implements the back office: catalog, marketing and order management.
package admin

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/apperr"
)

// Brand is an editable brand.
type Brand struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logoUrl"`
	Sort        int    `json:"sort"`
	IsActive    bool   `json:"isActive"`
	Lines       []Line `json:"lines"`
}

// Line is an editable product line.
type Line struct {
	ID          int64  `json:"id"`
	BrandID     int64  `json:"brandId"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	Sort        int    `json:"sort"`
	IsActive    bool   `json:"isActive"`
}

// Category is an editable product type.
type Category struct {
	ID       int64  `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Sort     int    `json:"sort"`
	IsActive bool   `json:"isActive"`
}

// ProductListItem is a row in the admin product table.
type ProductListItem struct {
	ID            int64  `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	BrandName     string `json:"brandName"`
	LineName      string `json:"lineName"`
	CategoryNames string `json:"categoryNames"`
	ImageURL      string `json:"imageUrl"`
	PriceFrom     *int64 `json:"priceFrom"`
	TotalStock    int    `json:"totalStock"`
	VariantCount  int    `json:"variantCount"`
	IsActive      bool   `json:"isActive"`
}

// ProductFilter narrows the admin product list.
type ProductFilter struct {
	Query   string
	BrandID int64
	Page    int
}

// ProductPage is a page of products.
type ProductPage struct {
	Items    []ProductListItem `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// Media is a product image or video.
type Media struct {
	URL  string `json:"url"`
	Kind string `json:"kind"`
}

// Variant is a purchasable option. ID 0 creates a new variant on save.
type Variant struct {
	ID             int64  `json:"id"`
	Label          string `json:"label"`
	SKU            string `json:"sku"`
	Price          int64  `json:"price"`
	CompareAtPrice *int64 `json:"compareAtPrice"`
	Stock          int    `json:"stock"`
	IsActive       bool   `json:"isActive"`
	ImageURL       string `json:"imageUrl"`
}

// Product is the full editable product. Saving replaces media and reconciles variants by ID.
type Product struct {
	ID          int64     `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	BrandID     int64     `json:"brandId"`
	LineID      *int64    `json:"lineId"`
	CategoryIDs []int64   `json:"categoryIds"`
	Description string    `json:"description"`
	Ingredients string    `json:"ingredients"`
	OptionKind  string    `json:"optionKind"`
	Sort        int       `json:"sort"`
	IsActive    bool      `json:"isActive"`
	Media       []Media   `json:"media"`
	Variants    []Variant `json:"variants"`
}

// Banner is an editable home page slide.
type Banner struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	Subtitle       string     `json:"subtitle"`
	ImageURL       string     `json:"imageUrl"`
	MobileImageURL string     `json:"mobileImageUrl"`
	LinkURL        string     `json:"linkUrl"`
	Sort           int        `json:"sort"`
	IsActive       bool       `json:"isActive"`
	StartsAt       *time.Time `json:"startsAt"`
	EndsAt         *time.Time `json:"endsAt"`
}

// Promo is an editable promo code.
type Promo struct {
	ID                  int64      `json:"id"`
	Code                string     `json:"code"`
	Kind                string     `json:"kind"`
	Value               int64      `json:"value"`
	MinSubtotal         int64      `json:"minSubtotal"`
	CombinableWithRules bool       `json:"combinableWithRules"`
	MaxUses             *int       `json:"maxUses"`
	UsedCount           int        `json:"usedCount"`
	IsActive            bool       `json:"isActive"`
	StartsAt            *time.Time `json:"startsAt"`
	EndsAt              *time.Time `json:"endsAt"`
}

// Rule is an editable automatic discount.
type Rule struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	BrandID     *int64     `json:"brandId"`
	BrandName   string     `json:"brandName"`
	MinSubtotal int64      `json:"minSubtotal"`
	Kind        string     `json:"kind"`
	Value       int64      `json:"value"`
	IsActive    bool       `json:"isActive"`
	StartsAt    *time.Time `json:"startsAt"`
	EndsAt      *time.Time `json:"endsAt"`
}

// Order statuses in their usual sequence.
const (
	StatusNew             = "new"
	StatusAwaitingPayment = "awaiting_payment"
	StatusPaid            = "paid"
	StatusShipped         = "shipped"
	StatusCompleted       = "completed"
	StatusCancelled       = "cancelled"
)

var orderStatuses = map[string]bool{
	StatusNew: true, StatusAwaitingPayment: true, StatusPaid: true,
	StatusShipped: true, StatusCompleted: true, StatusCancelled: true,
}

// OrderListItem is a row in the orders table.
type OrderListItem struct {
	ID           int64     `json:"id"`
	Number       string    `json:"number"`
	Status       string    `json:"status"`
	CustomerName string    `json:"customerName"`
	Phone        string    `json:"phone"`
	Delivery     string    `json:"delivery"`
	Total        int64     `json:"total"`
	ItemCount    int       `json:"itemCount"`
	CreatedAt    time.Time `json:"createdAt"`
}

// OrderItem is a line of a placed order.
type OrderItem struct {
	VariantID    *int64 `json:"variantId"`
	ProductName  string `json:"productName"`
	VariantLabel string `json:"variantLabel"`
	UnitPrice    int64  `json:"unitPrice"`
	Quantity     int    `json:"quantity"`
	LineTotal    int64  `json:"lineTotal"`
}

// Order is the full order card.
type Order struct {
	OrderListItem
	CustomerID    int64       `json:"customerId"`
	Address       string      `json:"address"`
	Comment       string      `json:"comment"`
	PromoCode     string      `json:"promoCode"`
	Subtotal      int64       `json:"subtotal"`
	DiscountTotal int64       `json:"discountTotal"`
	Items         []OrderItem `json:"items"`
}

// OrderFilter narrows the order list.
type OrderFilter struct {
	Status string
	Query  string
	Page   int
}

// OrderPage is a page of orders.
type OrderPage struct {
	Items    []OrderListItem `json:"items"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

// CustomerListItem is a row in the customers table. Statistics ignore cancelled orders.
type CustomerListItem struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	Phone            string     `json:"phone"`
	MarketingConsent bool       `json:"marketingConsent"`
	OrderCount       int        `json:"orderCount"`
	TotalSpent       int64      `json:"totalSpent"`
	LastOrderAt      *time.Time `json:"lastOrderAt"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// Customer is the customer card with the order history.
type Customer struct {
	CustomerListItem
	Note               string          `json:"note"`
	MarketingConsentAt *time.Time      `json:"marketingConsentAt"`
	Orders             []OrderListItem `json:"orders"`
}

// CustomerUpdate is what the owner can change on a customer card.
type CustomerUpdate struct {
	Name             string `json:"name"`
	Note             string `json:"note"`
	MarketingConsent bool   `json:"marketingConsent"`
}

// Customer list orderings.
const (
	SortRecent   = "recent"   // latest order first
	SortSpent    = "spent"    // biggest buyers first
	SortOrders   = "orders"   // most orders first
	SortInactive = "inactive" // longest without an order first
)

// CustomerFilter narrows the customer list.
type CustomerFilter struct {
	Query string
	// Marketing keeps only customers who agreed to receive promotions.
	Marketing bool
	Sort      string
	Page      int
}

// CustomerPage is a page of customers.
type CustomerPage struct {
	Items    []CustomerListItem `json:"items"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

// PageSize is the admin list page size.
const PageSize = 30

const (
	maxName        = 200
	maxText        = 5000
	maxURL         = 500
	maxMedia       = 20
	maxVariants    = 50
	maxPromoCode   = 30
	maxVariantText = 100
)

// validator accumulates field errors while normalizing input in place.
type validator struct {
	apperr.Fields
}

func newValidator() validator { return validator{apperr.Fields{}} }

func (v validator) text(field string, s *string, required bool, maxLen int) {
	*s = strings.TrimSpace(*s)
	switch {
	case required && *s == "":
		v.Add(field, apperr.Required)
	case len([]rune(*s)) > maxLen:
		v.Add(field, apperr.TooLong)
	}
}

// slug fills an empty slug from the name and reports whether it was generated.
func (v validator) slug(s *string, from string) bool {
	*s = strings.ToLower(strings.TrimSpace(*s))
	if *s == "" {
		*s = Slugify(from)
		if *s == "" {
			*s = "item"
		}
		return true
	}
	if !validSlug(*s) {
		v.Add("slug", apperr.Invalid)
	}
	return false
}

func (v validator) url(field string, s *string, required bool) {
	v.text(field, s, required, maxURL)
	if *s == "" {
		return
	}
	if strings.HasPrefix(*s, "/") && !strings.HasPrefix(*s, "//") {
		return
	}
	u, err := url.Parse(*s)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		v.Add(field, apperr.Invalid)
	}
}

func (v validator) discount(kind string, value int64) {
	switch kind {
	case "percent":
		if value < 1 || value > 100 {
			v.Add("value", apperr.Invalid)
		}
	case "fixed":
		if value < 1 {
			v.Add("value", apperr.Invalid)
		}
	default:
		v.Add("kind", apperr.Invalid)
	}
}

func (v validator) period(starts, ends *time.Time) {
	if starts != nil && ends != nil && !ends.After(*starts) {
		v.Add("endsAt", apperr.Invalid)
	}
}

func (b *Brand) validate() (autoSlug bool, err error) {
	v := newValidator()
	v.text("name", &b.Name, true, maxName)
	v.text("description", &b.Description, false, maxText)
	v.url("logoUrl", &b.LogoURL, false)
	autoSlug = v.slug(&b.Slug, b.Name)
	return autoSlug, v.Err()
}

func (l *Line) validate() (autoSlug bool, err error) {
	v := newValidator()
	if l.BrandID <= 0 {
		v.Add("brandId", apperr.Required)
	}
	v.text("name", &l.Name, true, maxName)
	v.text("description", &l.Description, false, maxText)
	v.url("imageUrl", &l.ImageURL, false)
	autoSlug = v.slug(&l.Slug, l.Name)
	return autoSlug, v.Err()
}

func (c *Category) validate() (autoSlug bool, err error) {
	v := newValidator()
	v.text("name", &c.Name, true, maxName)
	autoSlug = v.slug(&c.Slug, c.Name)
	return autoSlug, v.Err()
}

func (p *Product) validate() (autoSlug bool, err error) {
	v := newValidator()
	v.text("name", &p.Name, true, maxName)
	v.text("description", &p.Description, false, maxText)
	v.text("ingredients", &p.Ingredients, false, maxText)
	if p.BrandID <= 0 {
		v.Add("brandId", apperr.Required)
	}
	switch p.OptionKind {
	case "":
		p.OptionKind = "none"
	case "none", "volume", "color":
	default:
		v.Add("optionKind", apperr.Invalid)
	}

	if len(p.Media) > maxMedia {
		v.Add("media", apperr.TooLong)
	}
	for i := range p.Media {
		m := &p.Media[i]
		if m.Kind == "" {
			m.Kind = "image"
		}
		if m.Kind != "image" && m.Kind != "video" {
			v.Add(fmt.Sprintf("media.%d.kind", i), apperr.Invalid)
		}
		v.url(fmt.Sprintf("media.%d.url", i), &m.URL, true)
	}

	switch {
	case len(p.Variants) == 0:
		v.Add("variants", apperr.Required)
	case len(p.Variants) > maxVariants:
		v.Add("variants", apperr.TooLong)
	}
	needsLabel := len(p.Variants) > 1
	for i := range p.Variants {
		vr := &p.Variants[i]
		prefix := fmt.Sprintf("variants.%d.", i)
		v.text(prefix+"label", &vr.Label, needsLabel, maxVariantText)
		v.text(prefix+"sku", &vr.SKU, false, maxVariantText)
		if vr.Price < 0 {
			v.Add(prefix+"price", apperr.Invalid)
		}
		if vr.CompareAtPrice != nil && *vr.CompareAtPrice <= vr.Price {
			v.Add(prefix+"compareAtPrice", apperr.Invalid)
		}
		if vr.Stock < 0 {
			v.Add(prefix+"stock", apperr.Invalid)
		}
		if vr.ImageURL != "" && !p.hasImage(vr.ImageURL) {
			v.Add(prefix+"imageUrl", apperr.Invalid)
		}
	}

	autoSlug = v.slug(&p.Slug, p.Name)
	return autoSlug, v.Err()
}

func (p *Product) hasImage(url string) bool {
	return slices.ContainsFunc(p.Media, func(m Media) bool {
		return m.Kind == "image" && m.URL == url
	})
}

func (b *Banner) validate() error {
	v := newValidator()
	v.text("title", &b.Title, false, maxName)
	v.text("subtitle", &b.Subtitle, false, maxName)
	v.url("imageUrl", &b.ImageURL, true)
	v.url("mobileImageUrl", &b.MobileImageURL, false)
	v.url("linkUrl", &b.LinkURL, false)
	v.period(b.StartsAt, b.EndsAt)
	return v.Err()
}

var promoCodePattern = regexp.MustCompile(`^[A-Z0-9_-]+$`)

func (p *Promo) validate() error {
	v := newValidator()
	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
	switch {
	case p.Code == "":
		v.Add("code", apperr.Required)
	case len(p.Code) > maxPromoCode || !promoCodePattern.MatchString(p.Code):
		v.Add("code", apperr.Invalid)
	}
	v.discount(p.Kind, p.Value)
	if p.MinSubtotal < 0 {
		v.Add("minSubtotal", apperr.Invalid)
	}
	if p.MaxUses != nil && *p.MaxUses < 1 {
		v.Add("maxUses", apperr.Invalid)
	}
	v.period(p.StartsAt, p.EndsAt)
	return v.Err()
}

func (c *CustomerUpdate) validate() error {
	v := newValidator()
	v.text("name", &c.Name, true, maxName)
	v.text("note", &c.Note, false, maxText)
	return v.Err()
}

func (r *Rule) validate() error {
	v := newValidator()
	v.text("name", &r.Name, true, maxName)
	v.discount(r.Kind, r.Value)
	if r.MinSubtotal < 0 {
		v.Add("minSubtotal", apperr.Invalid)
	}
	v.period(r.StartsAt, r.EndsAt)
	return v.Err()
}
