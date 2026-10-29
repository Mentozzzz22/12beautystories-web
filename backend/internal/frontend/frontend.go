// Package frontend serves the built React app with page-specific SEO tags, plus robots.txt and sitemap.xml.
//
// It is not server-side rendering: every page gets the same index.html, but the <head> block between
// the seo markers is replaced so that link previews in Instagram, WhatsApp and Telegram show the
// product photo, name and price (their crawlers do not run JavaScript).
package frontend

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/catalog"
)

const (
	siteName           = "12 Beauty Stories"
	defaultDescription = "12 Beauty Stories: профессиональная косметика для волос и кожи в Астане. Davines, Payot и другие бренды."
	maxDescription     = 160
	markerStart        = "<!-- seo:start"
	markerEnd          = "<!-- seo:end -->"
)

// Source is the catalog data used for titles, previews and the sitemap.
type Source interface {
	Menu(ctx context.Context) (catalog.Menu, error)
	Product(ctx context.Context, slug string) (catalog.Product, error)
	SitemapProducts(ctx context.Context) ([]catalog.SitemapProduct, error)
}

// Handler serves static files from the Vite build and falls back to index.html for app routes.
type Handler struct {
	dir       string
	publicURL string
	head      []byte // index.html before the seo block
	tail      []byte // index.html after the seo block
	files     http.Handler
	src       Source
	log       *slog.Logger
}

// New loads index.html from dir. publicURL is the absolute site origin, e.g. https://12-beautystories.com.
func New(dir, publicURL string, src Source, log *slog.Logger) (*Handler, error) {
	index, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		return nil, fmt.Errorf("read index.html: %w", err)
	}
	start := bytes.Index(index, []byte(markerStart))
	end := bytes.Index(index, []byte(markerEnd))
	if start < 0 || end < start {
		return nil, errors.New("index.html has no seo:start / seo:end markers")
	}
	return &Handler{
		dir:       dir,
		publicURL: strings.TrimRight(publicURL, "/"),
		head:      index[:start],
		tail:      index[end+len(markerEnd):],
		files:     http.FileServer(http.Dir(dir)),
		src:       src,
		log:       log,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := path.Clean("/" + r.URL.Path)
	switch p {
	case "/robots.txt":
		h.robots(w)
		return
	case "/sitemap.xml":
		h.sitemap(w, r)
		return
	case "/index.html":
		p = "/"
	}

	if p != "/" && h.isFile(p) {
		if strings.HasPrefix(p, "/assets/") {
			// Vite puts a content hash in asset names, so they never change.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		h.files.ServeHTTP(w, r)
		return
	}
	h.page(w, r, p)
}

func (h *Handler) isFile(p string) bool {
	info, err := os.Stat(filepath.Join(h.dir, filepath.FromSlash(p)))
	return err == nil && !info.IsDir()
}

// Meta describes a page for search engines and link previews.
type Meta struct {
	Title       string
	Description string
	Image       string
	Path        string
	Type        string
	Price       int64
	NoIndex     bool
}

func (h *Handler) page(w http.ResponseWriter, r *http.Request, p string) {
	meta, status := h.resolve(r.Context(), p)
	var buf bytes.Buffer
	buf.Write(h.head)
	buf.WriteString(h.render(meta))
	buf.Write(h.tail)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// resolve builds metadata for known routes; unknown products answer 404 so they drop out of search.
func (h *Handler) resolve(ctx context.Context, p string) (Meta, int) {
	meta := Meta{Title: siteName, Description: defaultDescription, Path: p, Type: "website"}
	parts := strings.Split(strings.Trim(p, "/"), "/")

	switch {
	case len(parts) == 2 && parts[0] == "products":
		pr, err := h.src.Product(ctx, parts[1])
		if errors.Is(err, catalog.ErrNotFound) {
			meta.Title = "Страница не найдена | " + siteName
			meta.NoIndex = true
			return meta, http.StatusNotFound
		}
		if err != nil {
			h.log.ErrorContext(ctx, "seo: load product", "slug", parts[1], "err", err)
			return meta, http.StatusOK
		}
		meta.Title = fmt.Sprintf("%s — %s | %s", pr.Name, pr.Brand.Name, siteName)
		if pr.Description != "" {
			meta.Description = truncate(pr.Description, maxDescription)
		}
		for _, m := range pr.Media {
			if m.Kind == "image" {
				meta.Image = m.URL
				break
			}
		}
		meta.Type = "product"
		meta.Price = minPrice(pr.Variants)

	case parts[0] == "brands" || parts[0] == "categories":
		menu, err := h.src.Menu(ctx)
		if err != nil {
			h.log.ErrorContext(ctx, "seo: load menu", "err", err)
			return meta, http.StatusOK
		}
		if title, desc, ok := menuTitle(menu, parts); ok {
			meta.Title = title + " | " + siteName
			if desc != "" {
				meta.Description = truncate(desc, maxDescription)
			}
		}

	case parts[0] == "catalog":
		meta.Title = "Каталог | " + siteName

	case parts[0] == "privacy":
		meta.Title = "Политика конфиденциальности | " + siteName

	case parts[0] == "admin" || parts[0] == "cart" || parts[0] == "checkout" || parts[0] == "order":
		meta.NoIndex = true
	}
	return meta, http.StatusOK
}

func menuTitle(menu catalog.Menu, parts []string) (title, description string, ok bool) {
	if parts[0] == "categories" && len(parts) == 2 {
		for _, c := range menu.Categories {
			if c.Slug == parts[1] {
				return c.Name, "", true
			}
		}
		return "", "", false
	}
	if len(parts) < 2 {
		return "", "", false
	}
	for _, b := range menu.Brands {
		if b.Slug != parts[1] {
			continue
		}
		if len(parts) == 2 {
			return b.Name, b.Description, true
		}
		for _, l := range b.Lines {
			if l.Slug == parts[2] {
				return b.Name + " " + l.Name, l.Description, true
			}
		}
	}
	return "", "", false
}

func (h *Handler) render(m Meta) string {
	e := html.EscapeString
	var b strings.Builder
	fmt.Fprintf(&b, "<title>%s</title>\n", e(m.Title))
	fmt.Fprintf(&b, "    <meta name=\"description\" content=\"%s\" />\n", e(m.Description))
	if m.NoIndex {
		b.WriteString("    <meta name=\"robots\" content=\"noindex\" />\n")
	} else {
		fmt.Fprintf(&b, "    <link rel=\"canonical\" href=\"%s\" />\n", e(h.absolute(m.Path)))
	}
	fmt.Fprintf(&b, "    <meta property=\"og:site_name\" content=\"%s\" />\n", siteName)
	fmt.Fprintf(&b, "    <meta property=\"og:type\" content=\"%s\" />\n", e(m.Type))
	fmt.Fprintf(&b, "    <meta property=\"og:title\" content=\"%s\" />\n", e(m.Title))
	fmt.Fprintf(&b, "    <meta property=\"og:description\" content=\"%s\" />\n", e(m.Description))
	fmt.Fprintf(&b, "    <meta property=\"og:url\" content=\"%s\" />\n", e(h.absolute(m.Path)))
	b.WriteString("    <meta property=\"og:locale\" content=\"ru_RU\" />\n")
	if m.Image != "" {
		fmt.Fprintf(&b, "    <meta property=\"og:image\" content=\"%s\" />\n", e(h.absolute(m.Image)))
		b.WriteString("    <meta name=\"twitter:card\" content=\"summary_large_image\" />\n")
	}
	if m.Type == "product" && m.Price > 0 {
		fmt.Fprintf(&b, "    <meta property=\"product:price:amount\" content=\"%d\" />\n", m.Price)
		b.WriteString("    <meta property=\"product:price:currency\" content=\"KZT\" />\n")
	}
	return b.String()
}

func (h *Handler) absolute(p string) string {
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	return h.publicURL + p
}

func (h *Handler) robots(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "User-agent: *\nDisallow: /admin\nDisallow: /api/\nDisallow: /cart\nDisallow: /checkout\nDisallow: /order/\n\nSitemap: %s/sitemap.xml\n", h.publicURL)
}

type urlset struct {
	XMLName xml.Name     `xml:"urlset"`
	NS      string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func (h *Handler) sitemap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	menu, err := h.src.Menu(ctx)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	products, err := h.src.SitemapProducts(ctx)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	set := urlset{NS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	add := func(p, lastMod string) { set.URLs = append(set.URLs, sitemapURL{Loc: h.absolute(p), LastMod: lastMod}) }
	add("/", "")
	add("/catalog", "")
	for _, b := range menu.Brands {
		add("/brands/"+b.Slug, "")
		for _, l := range b.Lines {
			add("/brands/"+b.Slug+"/"+l.Slug, "")
		}
	}
	for _, c := range menu.Categories {
		add("/categories/"+c.Slug, "")
	}
	for _, p := range products {
		add("/products/"+p.Slug, p.UpdatedAt.UTC().Format("2006-01-02"))
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(set)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	h.log.ErrorContext(r.Context(), "frontend request failed", "path", r.URL.Path, "err", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func truncate(s string, limit int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	r := []rune(s)[:limit-1]
	cut := string(r)
	if i := strings.LastIndex(cut, " "); i > limit/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:") + "…"
}

func minPrice(vs []catalog.Variant) int64 {
	var lowest int64
	for i, v := range vs {
		if i == 0 || v.Price < lowest {
			lowest = v.Price
		}
	}
	return lowest
}
