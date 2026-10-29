package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/admin"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/apperr"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/auth"
)

type fakeAuth struct{}

func (fakeAuth) Login(_ context.Context, email, password string) (auth.Session, error) {
	if email != "mama@example.com" || password != "secret-password" {
		return auth.Session{}, auth.ErrInvalidCredentials
	}
	return auth.Session{Token: "good-token", ExpiresAt: time.Now().Add(time.Hour), User: auth.User{ID: 1, Email: email}}, nil
}

func (fakeAuth) Authenticate(_ context.Context, token string) (auth.User, error) {
	if token != "good-token" {
		return auth.User{}, auth.ErrUnauthenticated
	}
	return auth.User{ID: 1, Email: "mama@example.com"}, nil
}

func (fakeAuth) Logout(context.Context, string) error { return nil }

// fakeAdmin embeds the interface so tests implement only the methods they call.
type fakeAdmin struct {
	AdminStore
	saved          admin.Brand
	customerFilter admin.CustomerFilter
}

func (f *fakeAdmin) Brands(context.Context) ([]admin.Brand, error) {
	return []admin.Brand{{ID: 1, Name: "Davines"}}, nil
}

func (f *fakeAdmin) SaveBrand(_ context.Context, b admin.Brand) (admin.Brand, error) {
	if b.Name == "" {
		return admin.Brand{}, apperr.Field("name", apperr.Required)
	}
	f.saved = b
	return b, nil
}

func (f *fakeAdmin) DeleteBrand(_ context.Context, id int64) error {
	if id == 1 {
		return apperr.ErrConflict
	}
	return apperr.ErrNotFound
}

func newAdminAPI(store *fakeAdmin) http.Handler {
	return New(Deps{
		Catalog: &fakeCatalog{},
		Orders:  &fakeOrders{},
		Admin:   store,
		Auth:    fakeAuth{},
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func adminRequest(method, path, body string, authed bool) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authed {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "good-token"})
	}
	return req
}

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdminRequiresSession(t *testing.T) {
	h := newAdminAPI(&fakeAdmin{})

	if rec := serve(h, adminRequest(http.MethodGet, "/api/admin/brands", "", false)); rec.Code != http.StatusUnauthorized {
		t.Errorf("no cookie: status = %d, want 401", rec.Code)
	}
	req := adminRequest(http.MethodGet, "/api/admin/brands", "", false)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "stolen"})
	if rec := serve(h, req); rec.Code != http.StatusUnauthorized {
		t.Errorf("bad cookie: status = %d, want 401", rec.Code)
	}
	if rec := serve(h, adminRequest(http.MethodGet, "/api/admin/brands", "", true)); rec.Code != http.StatusOK {
		t.Errorf("valid cookie: status = %d, want 200", rec.Code)
	}
}

func TestLoginSetsHardenedCookie(t *testing.T) {
	h := newAdminAPI(&fakeAdmin{})

	rec := serve(h, adminRequest(http.MethodPost, "/api/admin/login",
		`{"email":"mama@example.com","password":"secret-password"}`, false))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %v", cookies)
	}
	c := cookies[0]
	if c.Value != "good-token" || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/api/admin" {
		t.Errorf("cookie = %+v", c)
	}

	rec = serve(h, adminRequest(http.MethodPost, "/api/admin/login", `{"email":"mama@example.com","password":"nope"}`, false))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong password status = %d, want 401", rec.Code)
	}
}

func TestLoginRateLimited(t *testing.T) {
	h := newAdminAPI(&fakeAdmin{})
	var last int
	for range loginAttempts + 1 {
		last = serve(h, adminRequest(http.MethodPost, "/api/admin/login", `{"email":"x@y.z","password":"bad"}`, false)).Code
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("status after %d attempts = %d, want 429", loginAttempts+1, last)
	}
}

func TestAdminCRUDMapping(t *testing.T) {
	store := &fakeAdmin{}
	h := newAdminAPI(store)

	rec := serve(h, adminRequest(http.MethodPut, "/api/admin/brands/7", `{"id":99,"name":"Payot"}`, true))
	if rec.Code != http.StatusOK || store.saved.ID != 7 {
		t.Errorf("update: status %d, saved %+v; path ID must win over body", rec.Code, store.saved)
	}

	tests := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodPost, "/api/admin/brands", `{"name":""}`, http.StatusUnprocessableEntity},
		{http.MethodPost, "/api/admin/brands", `{"name":"X","bogus":1}`, http.StatusBadRequest},
		{http.MethodDelete, "/api/admin/brands/1", "", http.StatusConflict},
		{http.MethodDelete, "/api/admin/brands/2", "", http.StatusNotFound},
		{http.MethodDelete, "/api/admin/brands/abc", "", http.StatusNotFound},
	}
	for _, tt := range tests {
		if rec := serve(h, adminRequest(tt.method, tt.path, tt.body, true)); rec.Code != tt.want {
			t.Errorf("%s %s: status = %d, want %d (%s)", tt.method, tt.path, rec.Code, tt.want, rec.Body)
		}
	}
}

func TestAdminRejectsFormPosts(t *testing.T) {
	h := newAdminAPI(&fakeAdmin{})
	req := adminRequest(http.MethodPost, "/api/admin/brands", "", true)
	req.Body = io.NopCloser(strings.NewReader("name=Evil"))
	req.ContentLength = int64(len("name=Evil"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if rec := serve(h, req); rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("form post status = %d, want 415", rec.Code)
	}
}

func TestRateLimiterWindow(t *testing.T) {
	now := time.Now()
	l := newRateLimiter(2, time.Minute)
	l.now = func() time.Time { return now }
	got := []bool{l.allow("ip"), l.allow("ip"), l.allow("ip")}
	if !got[0] || !got[1] || got[2] {
		t.Fatalf("hits allowed = %v, want exactly the first 2", got)
	}
	now = now.Add(2 * time.Minute)
	if !l.allow("ip") {
		t.Error("limiter must reset after the window")
	}
}

func (f *fakeAdmin) ExportCustomers(_ context.Context, cf admin.CustomerFilter) ([]admin.CustomerListItem, error) {
	f.customerFilter = cf
	last := time.Date(2026, 10, 30, 22, 0, 0, 0, time.UTC) // already Oct 31 in Astana
	return []admin.CustomerListItem{
		{Name: "Айгерим", Phone: "+77011234567", OrderCount: 2, TotalSpent: 15000, LastOrderAt: &last, MarketingConsent: true},
		{Name: "=HYPERLINK(\"x\")", Phone: "+77779998877"},
	}, nil
}

func TestExportCustomersCSV(t *testing.T) {
	store := &fakeAdmin{}
	rec := serve(newAdminAPI(store), adminRequest(http.MethodGet, "/api/admin/customers/export.csv?marketing=1&q=a", "", true))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if !store.customerFilter.Marketing || store.customerFilter.Query != "a" {
		t.Errorf("filter = %+v, want marketing and query passed through", store.customerFilter)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("Content-Type = %q", ct)
	}

	want := utf8BOM +
		"Имя;Телефон;Заказов;Сумма, ₸;Последний заказ;Согласие на рассылку\r\n" +
		"Айгерим;8 701 123 45 67;2;15000;31.10.2026;да\r\n" +
		"\"'=HYPERLINK(\"\"x\"\")\";8 777 999 88 77;0;0;;нет\r\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("csv =\n%q\nwant\n%q", got, want)
	}
}
