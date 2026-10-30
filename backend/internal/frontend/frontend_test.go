package frontend

import (
	"context"
	"encoding/xml"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/catalog"
)

const indexHTML = `<!doctype html><html><head>
    <!-- seo:start (replaced) -->
    <title>default</title>
    <!-- seo:end -->
<script src="/assets/app.js"></script></head><body><div id="root"></div></body></html>`

type fakeSource struct{}

func (fakeSource) Menu(context.Context) (catalog.Menu, error) {
	return catalog.Menu{
		Brands: []catalog.Brand{{Slug: "davines", Name: "Davines", Description: "Итальянский бренд",
			Lines: []catalog.Line{{Slug: "energizing", Name: "Energizing", Description: "Против выпадения"}}}},
		Categories: []catalog.Category{{Slug: "shampoo", Name: "Шампуни"}},
	}, nil
}

func (fakeSource) Product(_ context.Context, slug string) (catalog.Product, error) {
	if slug != "oi-oil" {
		return catalog.Product{}, catalog.ErrNotFound
	}
	return catalog.Product{
		Slug: "oi-oil", Name: `OI "Oil" <b>`, Brand: catalog.Ref{Name: "Davines"},
		Description: strings.Repeat("Масло для блеска волос. ", 20),
		Media:       []catalog.Media{{URL: "/uploads/oi.jpg", Kind: "image"}},
		Variants:    []catalog.Variant{{Price: 32000}, {Price: 16000}},
	}, nil
}

func (fakeSource) SitemapProducts(context.Context) ([]catalog.SitemapProduct, error) {
	return []catalog.SitemapProduct{{Slug: "oi-oil", UpdatedAt: time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)}}, nil
}

func newHandler(t *testing.T) *Handler {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(indexHTML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := New(dir, "https://shop.test/", fakeSource{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
	return rec
}

func TestProductPreview(t *testing.T) {
	rec := get(t, newHandler(t), "/products/oi-oil")
	body := rec.Body.String()

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	for _, want := range []string{
		`<title>OI &#34;Oil&#34; &lt;b&gt; — Davines | 12 Beauty Stories</title>`,
		`<meta property="og:image" content="https://shop.test/uploads/oi.jpg" />`,
		`<meta property="og:type" content="product" />`,
		`<meta property="product:price:amount" content="16000" />`,
		`<link rel="canonical" href="https://shop.test/products/oi-oil" />`,
		`<div id="root"></div>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %s", want)
		}
	}
	if strings.Contains(body, "<title>default</title>") || strings.Count(body, "<title>") != 1 {
		t.Error("default seo block must be replaced exactly once")
	}
	if !strings.Contains(body, "…") {
		t.Error("long description must be truncated")
	}
}

func TestUnknownProductIs404(t *testing.T) {
	rec := get(t, newHandler(t), "/products/missing")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `content="noindex"`) {
		t.Errorf("status = %d, want 404 with noindex", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `<div id="root"></div>`) {
		t.Error("the app shell must still render so the SPA shows its 404 page")
	}
}

func TestBrandLineAndPrivatePages(t *testing.T) {
	h := newHandler(t)
	if body := get(t, h, "/brands/davines/energizing").Body.String(); !strings.Contains(body, "<title>Davines Energizing | 12 Beauty Stories</title>") ||
		!strings.Contains(body, `content="Против выпадения"`) {
		t.Errorf("line page meta missing:\n%s", body)
	}
	if body := get(t, h, "/categories/shampoo").Body.String(); !strings.Contains(body, "<title>Шампуни | 12 Beauty Stories</title>") {
		t.Error("category title missing")
	}
	if body := get(t, h, "/admin/orders").Body.String(); !strings.Contains(body, `content="noindex"`) {
		t.Error("admin pages must be noindex")
	}
}

func TestStaticAssetsAreCachedForever(t *testing.T) {
	rec := get(t, newHandler(t), "/assets/app.js")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Fatalf("asset: %d %q", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("Cache-Control = %q", rec.Header().Get("Cache-Control"))
	}
	if cc := get(t, newHandler(t), "/").Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("index Cache-Control = %q, want no-cache", cc)
	}
}

func TestPathTraversalStaysInDir(t *testing.T) {
	rec := get(t, newHandler(t), "/../../etc/passwd")
	if !strings.Contains(rec.Body.String(), `<div id="root"></div>`) {
		t.Errorf("traversal must fall back to the app shell, got %d", rec.Code)
	}
}

func TestSitemapAndRobots(t *testing.T) {
	h := newHandler(t)
	var set urlset
	if err := xml.Unmarshal(get(t, h, "/sitemap.xml").Body.Bytes(), &set); err != nil {
		t.Fatalf("sitemap is not valid XML: %v", err)
	}
	locs := map[string]string{}
	for _, u := range set.URLs {
		locs[u.Loc] = u.LastMod
	}
	for _, want := range []string{"https://shop.test/", "https://shop.test/brands/davines/energizing",
		"https://shop.test/categories/shampoo", "https://shop.test/products/oi-oil"} {
		if _, ok := locs[want]; !ok {
			t.Errorf("sitemap misses %s", want)
		}
	}
	if locs["https://shop.test/products/oi-oil"] != "2026-10-28" {
		t.Error("product lastmod missing")
	}

	robots := get(t, h, "/robots.txt").Body.String()
	if !strings.Contains(robots, "Disallow: /admin") || !strings.Contains(robots, "Sitemap: https://shop.test/sitemap.xml") {
		t.Errorf("robots.txt = %q", robots)
	}
}

func TestTruncateRemovesFormatting(t *testing.T) {
	if got := truncate("<b>Масло</b> для <i>блеска</i>\n<u>волос</u>", 160); got != "Масло для блеска волос" {
		t.Errorf("truncate() = %q", got)
	}
}

func TestNoIndexHidesSite(t *testing.T) {
	h := newHandler(t)
	h.NoIndex = true

	if robots := get(t, h, "/robots.txt").Body.String(); robots != "User-agent: *\nDisallow: /\n" {
		t.Errorf("robots.txt = %q", robots)
	}
	page := get(t, h, "/products/oi-oil").Body.String()
	if !strings.Contains(page, `content="noindex"`) || strings.Contains(page, `rel="canonical"`) {
		t.Errorf("product page must be noindex without canonical:\n%s", page)
	}
}

func TestNewRequiresMarkers(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o644)
	if _, err := New(dir, "https://shop.test", fakeSource{}, slog.Default()); err == nil {
		t.Error("New() must fail without seo markers")
	}
}
