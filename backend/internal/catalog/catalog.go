// Package catalog exposes the read side of the storefront: menu, banners and products.
package catalog

import "errors"

// ErrNotFound is returned when a requested entity does not exist or is hidden.
var ErrNotFound = errors.New("not found")

// Brand is a manufacturer with its product lines.
type Brand struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logoUrl"`
	Lines       []Line `json:"lines"`
}

// Line is a product line inside a brand (e.g. Davines Energizing).
type Line struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
}

// Category is a product type (shampoo, serum, ...).
type Category struct {
	ID   int64  `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Menu is the navigation tree of the storefront.
type Menu struct {
	Brands     []Brand    `json:"brands"`
	Categories []Category `json:"categories"`
}

// Banner is a home page slide.
type Banner struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	Subtitle       string `json:"subtitle"`
	ImageURL       string `json:"imageUrl"`
	MobileImageURL string `json:"mobileImageUrl"`
	LinkURL        string `json:"linkUrl"`
}

// Ref is a short reference to a related entity.
type Ref struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// ProductSummary is a product card in listings.
type ProductSummary struct {
	ID             int64  `json:"id"`
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Brand          Ref    `json:"brand"`
	Line           *Ref   `json:"line,omitempty"`
	ImageURL       string `json:"imageUrl"`
	PriceFrom      int64  `json:"priceFrom"`
	CompareAtPrice *int64 `json:"compareAtPrice,omitempty"`
	HasOptions     bool   `json:"hasOptions"`
	InStock        bool   `json:"inStock"`
}

// OptionKind says what distinguishes variants of a product.
type OptionKind string

// Supported option kinds.
const (
	OptionNone   OptionKind = "none"
	OptionVolume OptionKind = "volume"
	OptionColor  OptionKind = "color"
)

// Media is a product photo or video.
type Media struct {
	URL  string `json:"url"`
	Kind string `json:"kind"`
}

// Variant is a purchasable option of a product (a volume or a colour).
type Variant struct {
	ID             int64  `json:"id"`
	Label          string `json:"label"`
	SKU            string `json:"sku"`
	Price          int64  `json:"price"`
	CompareAtPrice *int64 `json:"compareAtPrice,omitempty"`
	InStock        bool   `json:"inStock"`
}

// Product is the full product page.
type Product struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Brand       Ref        `json:"brand"`
	Line        *Ref       `json:"line,omitempty"`
	Category    *Ref       `json:"category,omitempty"`
	Description string     `json:"description"`
	Ingredients string     `json:"ingredients"`
	OptionKind  OptionKind `json:"optionKind"`
	Media       []Media    `json:"media"`
	Variants    []Variant  `json:"variants"`
}

// Filter narrows the product list. Empty fields are ignored.
type Filter struct {
	Brand    string
	Line     string
	Category string
	Query    string
	Page     int
	PageSize int
}

// Paging limits for product lists.
const (
	DefaultPageSize = 24
	MaxPageSize     = 100
)

// Normalize clamps paging values to sane defaults.
func (f Filter) Normalize() Filter {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = DefaultPageSize
	}
	f.PageSize = min(f.PageSize, MaxPageSize)
	return f
}

// Page is one page of product summaries.
type Page struct {
	Items    []ProductSummary `json:"items"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}
