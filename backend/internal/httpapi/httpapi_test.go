package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/apperr"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/catalog"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/order"
)

type fakeCatalog struct {
	lastFilter catalog.Filter
}

func (f *fakeCatalog) Menu(context.Context) (catalog.Menu, error) {
	return catalog.Menu{Brands: []catalog.Brand{{Slug: "davines", Name: "Davines"}}}, nil
}

func (f *fakeCatalog) Banners(context.Context) ([]catalog.Banner, error) {
	return nil, errors.New("db down")
}

func (f *fakeCatalog) Products(_ context.Context, fl catalog.Filter) (catalog.Page, error) {
	f.lastFilter = fl
	return catalog.Page{Items: []catalog.ProductSummary{}, Page: 1, PageSize: 24}, nil
}

func (f *fakeCatalog) Product(_ context.Context, slug string) (catalog.Product, error) {
	if slug != "energizing-shampoo" {
		return catalog.Product{}, catalog.ErrNotFound
	}
	return catalog.Product{Slug: slug, Name: "Energizing Shampoo"}, nil
}

type fakeOrders struct {
	err error
}

func (f *fakeOrders) Quote(context.Context, []order.Item, string) (order.QuoteResult, error) {
	return order.QuoteResult{}, f.err
}

func (f *fakeOrders) Place(_ context.Context, _ []order.Item, _ string, c order.Customer) (order.Order, error) {
	if f.err != nil {
		return order.Order{}, f.err
	}
	return order.Order{ID: 1, Number: "BS-00001", Customer: c}, nil
}

func newTestAPI(cat *fakeCatalog, ord *fakeOrders) http.Handler {
	return New(Deps{Catalog: cat, Orders: ord, Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body errorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Code
}

func TestRoutes(t *testing.T) {
	h := newTestAPI(&fakeCatalog{}, &fakeOrders{})

	tests := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodGet, "/api/health", "", http.StatusOK},
		{http.MethodGet, "/api/menu", "", http.StatusOK},
		{http.MethodGet, "/api/banners", "", http.StatusInternalServerError},
		{http.MethodGet, "/api/products/energizing-shampoo", "", http.StatusOK},
		{http.MethodGet, "/api/products/missing", "", http.StatusNotFound},
		{http.MethodPost, "/api/menu", "", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/orders", `{"items":[{"variantId":1,"quantity":1}],"customer":{"name":"A"}}`, http.StatusCreated},
		{http.MethodPost, "/api/orders", `{"unknown":1}`, http.StatusBadRequest},
		{http.MethodPost, "/api/cart/quote", `not json`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			if rec := do(t, h, tt.method, tt.path, tt.body); rec.Code != tt.want {
				t.Errorf("status = %d, want %d; body: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}
}

func TestProductsFilterFromQuery(t *testing.T) {
	cat := &fakeCatalog{}
	h := newTestAPI(cat, &fakeOrders{})

	do(t, h, http.MethodGet, "/api/products?brand=davines&line=energizing&category=shampoo&q=hair&page=2&pageSize=12", "")

	want := catalog.Filter{Brand: "davines", Line: "energizing", Category: "shampoo", Query: "hair", Page: 2, PageSize: 12}
	if cat.lastFilter != want {
		t.Errorf("filter = %+v, want %+v", cat.lastFilter, want)
	}
}

func TestOrderErrorMapping(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{&apperr.ValidationError{Fields: map[string]string{"phone": "invalid"}}, http.StatusUnprocessableEntity, "validation"},
		{order.ErrEmptyCart, http.StatusUnprocessableEntity, "empty_cart"},
		{order.ErrVariantUnavailable, http.StatusConflict, "variant_unavailable"},
		{order.ErrInsufficientStock, http.StatusConflict, "insufficient_stock"},
		{order.ErrInvalidPromo, http.StatusUnprocessableEntity, "invalid_promo"},
		{errors.New("boom"), http.StatusInternalServerError, "internal"},
	}
	for _, tt := range tests {
		t.Run(tt.wantCode, func(t *testing.T) {
			h := newTestAPI(&fakeCatalog{}, &fakeOrders{err: tt.err})
			rec := do(t, h, http.MethodPost, "/api/cart/quote", `{"items":[]}`)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if code := errorCode(t, rec); code != tt.wantCode {
				t.Errorf("code = %q, want %q", code, tt.wantCode)
			}
		})
	}
}

func TestFrontendFallbackDoesNotSwallowAPI(t *testing.T) {
	web := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("app shell"))
	})
	h := New(Deps{Catalog: &fakeCatalog{}, Orders: &fakeOrders{}, Frontend: web,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil))})

	if rec := do(t, h, http.MethodGet, "/products/anything", ""); rec.Body.String() != "app shell" {
		t.Errorf("app route body = %q, want app shell", rec.Body)
	}
	rec := do(t, h, http.MethodGet, "/api/nope", "")
	if rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
		t.Errorf("unknown API path: status %d, want JSON 404", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/api/health", ""); rec.Code != http.StatusOK {
		t.Errorf("health status = %d", rec.Code)
	}
}
