// Package httpapi exposes the storefront and admin JSON APIs.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/apperr"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/catalog"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/order"
)

// Catalog is the read side used by the storefront.
type Catalog interface {
	Menu(ctx context.Context) (catalog.Menu, error)
	Banners(ctx context.Context) ([]catalog.Banner, error)
	Products(ctx context.Context, f catalog.Filter) (catalog.Page, error)
	Product(ctx context.Context, slug string) (catalog.Product, error)
}

// Orders quotes carts and places orders.
type Orders interface {
	Quote(ctx context.Context, items []order.Item, promoCode string) (order.QuoteResult, error)
	Place(ctx context.Context, items []order.Item, promoCode string, c order.Customer) (order.Order, error)
}

const maxBodyBytes = 64 << 10

// Deps are the handler dependencies. Admin, Auth and Uploads may be nil to serve only the storefront.
type Deps struct {
	Catalog Catalog
	Orders  Orders
	Admin   AdminStore
	Auth    Authenticator
	Uploads Uploader
	// UploadDir is served under /uploads/.
	UploadDir    string
	CookieSecure bool
	TrustProxy   bool
	Log          *slog.Logger
}

// API holds handler dependencies.
type API struct {
	Deps
	loginLimiter *rateLimiter
}

// New builds the HTTP handler with all routes and middleware.
func New(d Deps) http.Handler {
	a := &API{Deps: d, loginLimiter: newRateLimiter(loginAttempts, loginWindow)}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", a.health)
	mux.HandleFunc("GET /api/menu", a.menu)
	mux.HandleFunc("GET /api/banners", a.banners)
	mux.HandleFunc("GET /api/products", a.products)
	mux.HandleFunc("GET /api/products/{slug}", a.product)
	mux.HandleFunc("POST /api/cart/quote", a.quote)
	mux.HandleFunc("POST /api/orders", a.placeOrder)
	if d.UploadDir != "" {
		mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", noListing(http.FileServer(http.Dir(d.UploadDir)))))
	}
	if d.Admin != nil && d.Auth != nil {
		a.registerAdmin(mux)
	}

	return a.recoverer(a.logRequests(mux))
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) menu(w http.ResponseWriter, r *http.Request) {
	menu, err := a.Catalog.Menu(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, menu)
}

func (a *API) banners(w http.ResponseWriter, r *http.Request) {
	banners, err := a.Catalog.Banners(r.Context())
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, banners)
}

func (a *API) products(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))

	res, err := a.Catalog.Products(r.Context(), catalog.Filter{
		Brand:    q.Get("brand"),
		Line:     q.Get("line"),
		Category: q.Get("category"),
		Query:    q.Get("q"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (a *API) product(w http.ResponseWriter, r *http.Request) {
	p, err := a.Catalog.Product(r.Context(), r.PathValue("slug"))
	if errors.Is(err, catalog.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "product not found")
		return
	}
	if err != nil {
		a.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type quoteRequest struct {
	Items     []order.Item `json:"items"`
	PromoCode string       `json:"promoCode"`
}

func (a *API) quote(w http.ResponseWriter, r *http.Request) {
	var req quoteRequest
	if !a.decode(w, r, &req) {
		return
	}
	res, err := a.Orders.Quote(r.Context(), req.Items, req.PromoCode)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type placeOrderRequest struct {
	Items     []order.Item   `json:"items"`
	PromoCode string         `json:"promoCode"`
	Customer  order.Customer `json:"customer"`
}

func (a *API) placeOrder(w http.ResponseWriter, r *http.Request) {
	var req placeOrderRequest
	if !a.decode(w, r, &req) {
		return
	}
	o, err := a.Orders.Place(r.Context(), req.Items, req.PromoCode, req.Customer)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

// fail maps service errors to HTTP responses.
func (a *API) fail(w http.ResponseWriter, r *http.Request, err error) {
	var verr *apperr.ValidationError
	switch {
	case errors.As(err, &verr):
		writeJSON(w, http.StatusUnprocessableEntity, errorBody{Code: "validation", Message: "invalid fields", Fields: verr.Fields})
	case errors.Is(err, apperr.ErrNotFound), errors.Is(err, catalog.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, apperr.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, order.ErrEmptyCart):
		writeError(w, http.StatusUnprocessableEntity, "empty_cart", err.Error())
	case errors.Is(err, order.ErrVariantUnavailable):
		writeError(w, http.StatusConflict, "variant_unavailable", err.Error())
	case errors.Is(err, order.ErrInsufficientStock):
		writeError(w, http.StatusConflict, "insufficient_stock", err.Error())
	case errors.Is(err, order.ErrInvalidPromo):
		writeError(w, http.StatusUnprocessableEntity, "invalid_promo", err.Error())
	default:
		a.internalError(w, r, err)
	}
}

func (a *API) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return false
	}
	return true
}

func (a *API) internalError(w http.ResponseWriter, r *http.Request, err error) {
	a.Log.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "internal server error")
}

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errorBody{Code: code, Message: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (a *API) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		a.Log.InfoContext(r.Context(), "http",
			"method", r.Method, "path", r.URL.Path, "status", rec.status, "duration", time.Since(start))
	})
}

func (a *API) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler { //nolint:errorlint // sentinel compared by identity, as net/http does
					panic(v)
				}
				a.Log.ErrorContext(r.Context(), "panic", "value", v, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
