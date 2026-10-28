package httpapi

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/admin"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/apperr"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/auth"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/media"
)

// AdminStore is the back office repository.
type AdminStore interface {
	Brands(ctx context.Context) ([]admin.Brand, error)
	SaveBrand(ctx context.Context, b admin.Brand) (admin.Brand, error)
	DeleteBrand(ctx context.Context, id int64) error
	SaveLine(ctx context.Context, l admin.Line) (admin.Line, error)
	DeleteLine(ctx context.Context, id int64) error

	Categories(ctx context.Context) ([]admin.Category, error)
	SaveCategory(ctx context.Context, c admin.Category) (admin.Category, error)
	DeleteCategory(ctx context.Context, id int64) error

	Products(ctx context.Context, f admin.ProductFilter) (admin.ProductPage, error)
	Product(ctx context.Context, id int64) (admin.Product, error)
	SaveProduct(ctx context.Context, p admin.Product) (admin.Product, error)
	DeleteProduct(ctx context.Context, id int64) error

	Banners(ctx context.Context) ([]admin.Banner, error)
	SaveBanner(ctx context.Context, b admin.Banner) (admin.Banner, error)
	DeleteBanner(ctx context.Context, id int64) error

	Promos(ctx context.Context) ([]admin.Promo, error)
	SavePromo(ctx context.Context, p admin.Promo) (admin.Promo, error)
	DeletePromo(ctx context.Context, id int64) error

	Rules(ctx context.Context) ([]admin.Rule, error)
	SaveRule(ctx context.Context, r admin.Rule) (admin.Rule, error)
	DeleteRule(ctx context.Context, id int64) error

	Orders(ctx context.Context, f admin.OrderFilter) (admin.OrderPage, error)
	Order(ctx context.Context, id int64) (admin.Order, error)
	SetOrderStatus(ctx context.Context, id int64, status string) (admin.Order, error)
	NewOrdersCount(ctx context.Context) (int, error)
}

// Authenticator manages admin sessions.
type Authenticator interface {
	Login(ctx context.Context, email, password string) (auth.Session, error)
	Authenticate(ctx context.Context, token string) (auth.User, error)
	Logout(ctx context.Context, token string) error
}

// Uploader stores images.
type Uploader interface {
	SaveImage(r io.Reader) (string, error)
}

const (
	sessionCookie = "bs_admin"
	loginAttempts = 10
	loginWindow   = 15 * time.Minute
)

type userKey struct{}

func (a *API) registerAdmin(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/admin/login", a.login)

	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/admin/logout", a.logout)
	protected.HandleFunc("GET /api/admin/me", a.me)

	s := a.Admin
	protected.HandleFunc("GET /api/admin/brands", list(a, s.Brands))
	protected.HandleFunc("POST /api/admin/brands", create(a, s.SaveBrand))
	protected.HandleFunc("PUT /api/admin/brands/{id}", update(a, s.SaveBrand, func(b *admin.Brand, id int64) { b.ID = id }))
	protected.HandleFunc("DELETE /api/admin/brands/{id}", remove(a, s.DeleteBrand))
	protected.HandleFunc("POST /api/admin/lines", create(a, s.SaveLine))
	protected.HandleFunc("PUT /api/admin/lines/{id}", update(a, s.SaveLine, func(l *admin.Line, id int64) { l.ID = id }))
	protected.HandleFunc("DELETE /api/admin/lines/{id}", remove(a, s.DeleteLine))

	protected.HandleFunc("GET /api/admin/categories", list(a, s.Categories))
	protected.HandleFunc("POST /api/admin/categories", create(a, s.SaveCategory))
	protected.HandleFunc("PUT /api/admin/categories/{id}", update(a, s.SaveCategory, func(c *admin.Category, id int64) { c.ID = id }))
	protected.HandleFunc("DELETE /api/admin/categories/{id}", remove(a, s.DeleteCategory))

	protected.HandleFunc("GET /api/admin/products", a.adminProducts)
	protected.HandleFunc("GET /api/admin/products/{id}", get(a, s.Product))
	protected.HandleFunc("POST /api/admin/products", create(a, s.SaveProduct))
	protected.HandleFunc("PUT /api/admin/products/{id}", update(a, s.SaveProduct, func(p *admin.Product, id int64) { p.ID = id }))
	protected.HandleFunc("DELETE /api/admin/products/{id}", remove(a, s.DeleteProduct))

	protected.HandleFunc("GET /api/admin/banners", list(a, s.Banners))
	protected.HandleFunc("POST /api/admin/banners", create(a, s.SaveBanner))
	protected.HandleFunc("PUT /api/admin/banners/{id}", update(a, s.SaveBanner, func(b *admin.Banner, id int64) { b.ID = id }))
	protected.HandleFunc("DELETE /api/admin/banners/{id}", remove(a, s.DeleteBanner))

	protected.HandleFunc("GET /api/admin/promos", list(a, s.Promos))
	protected.HandleFunc("POST /api/admin/promos", create(a, s.SavePromo))
	protected.HandleFunc("PUT /api/admin/promos/{id}", update(a, s.SavePromo, func(p *admin.Promo, id int64) { p.ID = id }))
	protected.HandleFunc("DELETE /api/admin/promos/{id}", remove(a, s.DeletePromo))

	protected.HandleFunc("GET /api/admin/rules", list(a, s.Rules))
	protected.HandleFunc("POST /api/admin/rules", create(a, s.SaveRule))
	protected.HandleFunc("PUT /api/admin/rules/{id}", update(a, s.SaveRule, func(r *admin.Rule, id int64) { r.ID = id }))
	protected.HandleFunc("DELETE /api/admin/rules/{id}", remove(a, s.DeleteRule))

	protected.HandleFunc("GET /api/admin/orders", a.adminOrders)
	protected.HandleFunc("GET /api/admin/orders/{id}", get(a, s.Order))
	protected.HandleFunc("PUT /api/admin/orders/{id}/status", a.setOrderStatus)

	if a.Uploads != nil {
		protected.HandleFunc("POST /api/admin/uploads", a.upload)
	}

	mux.Handle("/api/admin/", a.requireAdmin(protected))
}

// ---- Generic CRUD handlers ----

func list[T any](a *API, fn func(context.Context) ([]T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := fn(r.Context())
		if err != nil {
			a.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	}
}

func get[T any](a *API, fn func(context.Context, int64) (T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		item, err := fn(r.Context(), id)
		if err != nil {
			a.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	}
}

func create[T any](a *API, save func(context.Context, T) (T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in T
		if !a.decode(w, r, &in) {
			return
		}
		out, err := save(r.Context(), in)
		if err != nil {
			a.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// update decodes the body, forces the ID from the path and saves.
func update[T any](a *API, save func(context.Context, T) (T, error), setID func(*T, int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		var in T
		if !a.decode(w, r, &in) {
			return
		}
		setID(&in, id)
		out, err := save(r.Context(), in)
		if err != nil {
			a.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func remove(a *API, del func(context.Context, int64) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		if err := del(r.Context(), id); err != nil {
			a.fail(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return 0, false
	}
	return id, true
}

// ---- Specific handlers ----

func (a *API) adminProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	brandID, _ := strconv.ParseInt(q.Get("brandId"), 10, 64)
	res, err := a.Admin.Products(r.Context(), admin.ProductFilter{Query: q.Get("q"), BrandID: brandID, Page: page})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type adminOrdersResponse struct {
	admin.OrderPage
	NewCount int `json:"newCount"`
}

func (a *API) adminOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	res, err := a.Admin.Orders(r.Context(), admin.OrderFilter{Status: q.Get("status"), Query: q.Get("q"), Page: page})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	newCount, err := a.Admin.NewOrdersCount(r.Context())
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, adminOrdersResponse{OrderPage: res, NewCount: newCount})
}

func (a *API) setOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !a.decode(w, r, &in) {
		return
	}
	o, err := a.Admin.SetOrderStatus(r.Context(), id, in.Status)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (a *API) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, media.MaxUploadBytes+1<<20)
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "multipart field \"file\" is required")
		return
	}
	defer func() { _ = file.Close() }()

	url, err := a.Uploads.SaveImage(file)
	if errors.Is(err, media.ErrUnsupported) {
		writeJSON(w, http.StatusUnprocessableEntity, errorBody{
			Code: "validation", Message: err.Error(), Fields: map[string]string{"file": apperr.Invalid},
		})
		return
	}
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"url": url})
}

// ---- Auth ----

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	ip := a.clientIP(r)
	if !a.loginLimiter.allow(ip) {
		writeError(w, http.StatusTooManyRequests, "too_many_attempts", "too many login attempts, try again later")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !a.decode(w, r, &in) {
		return
	}
	sess, err := a.Auth.Login(r.Context(), in.Email, in.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	if err != nil {
		a.fail(w, r, err)
		return
	}
	a.loginLimiter.reset(ip)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    sess.Token,
		Path:     "/api/admin",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   a.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, sess.User)
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := a.Auth.Logout(r.Context(), c.Value); err != nil {
			a.fail(w, r, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/api/admin", MaxAge: -1,
		HttpOnly: true, Secure: a.CookieSecure, SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, r.Context().Value(userKey{}))
}

// requireAdmin rejects requests without a valid session. Unsafe methods must send JSON or multipart
// bodies, which together with the SameSite=Strict cookie blocks cross-site form submissions.
func (a *API) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
			return
		}
		u, err := a.Auth.Authenticate(r.Context(), c.Value)
		if errors.Is(err, auth.ErrUnauthenticated) {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
			return
		}
		if err != nil {
			a.fail(w, r, err)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.ContentLength != 0 {
			ct := r.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "application/json") && !strings.HasPrefix(ct, "multipart/form-data") {
				writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "use application/json")
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	})
}

func (a *API) clientIP(r *http.Request) string {
	if a.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// noListing hides directory indexes of the upload folder.
func noListing(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		h.ServeHTTP(w, r)
	})
}

// rateLimiter counts failed-or-not attempts per key in a fixed window.
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*window
	now    func() time.Time
}

type window struct {
	start time.Time
	count int
}

func newRateLimiter(limit int, d time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: d, hits: make(map[string]*window), now: time.Now}
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.hits) > 10_000 {
		for k, w := range l.hits {
			if now.Sub(w.start) > l.window {
				delete(l.hits, k)
			}
		}
	}
	w, ok := l.hits[key]
	if !ok || now.Sub(w.start) > l.window {
		l.hits[key] = &window{start: now, count: 1}
		return true
	}
	w.count++
	return w.count <= l.limit
}

func (l *rateLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}
