package catalog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/catalog"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/database/dbtest"
)

func newRepo(t *testing.T) *catalog.Postgres {
	t.Helper()
	pool := dbtest.New(t)
	dbtest.Seed(t, pool)
	return catalog.NewPostgres(pool)
}

func TestMenu(t *testing.T) {
	repo := newRepo(t)

	menu, err := repo.Menu(context.Background())
	if err != nil {
		t.Fatalf("Menu() error = %v", err)
	}
	if len(menu.Brands) != 2 || menu.Brands[0].Slug != "davines" {
		t.Fatalf("brands = %+v, want davines first of 2", menu.Brands)
	}
	if got := len(menu.Brands[0].Lines); got != 2 {
		t.Errorf("davines lines = %d, want 2", got)
	}
	if len(menu.Categories) != 5 {
		t.Errorf("categories = %d, want 5", len(menu.Categories))
	}
}

func TestMenuHidesInactive(t *testing.T) {
	pool := dbtest.New(t)
	dbtest.Seed(t, pool)
	dbtest.Exec(t, pool, `UPDATE brands SET is_active = false WHERE slug = 'payot'`)
	dbtest.Exec(t, pool, `UPDATE product_lines SET is_active = false WHERE slug = 'minu'`)

	menu, err := catalog.NewPostgres(pool).Menu(context.Background())
	if err != nil {
		t.Fatalf("Menu() error = %v", err)
	}
	if len(menu.Brands) != 1 || len(menu.Brands[0].Lines) != 1 {
		t.Errorf("menu = %+v, want only davines with one line", menu.Brands)
	}
}

func TestBannersRespectSchedule(t *testing.T) {
	pool := dbtest.New(t)
	dbtest.Seed(t, pool)
	dbtest.Exec(t, pool, `UPDATE banners SET ends_at = now() - interval '1 day' WHERE sort = 1`)
	dbtest.Exec(t, pool, `UPDATE banners SET starts_at = now() + interval '1 day' WHERE sort = 2`)

	banners, err := catalog.NewPostgres(pool).Banners(context.Background())
	if err != nil {
		t.Fatalf("Banners() error = %v", err)
	}
	if len(banners) != 1 || banners[0].Title != "Payot Hydra 24+" {
		t.Errorf("banners = %+v, want only the Payot banner", banners)
	}
}

func TestProducts(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		filter catalog.Filter
		want   []string
	}{
		{"all", catalog.Filter{}, []string{"davines-paddle-brush", "payot-hydra-24-cream", "davines-minu-shampoo", "davines-energizing-shampoo"}},
		{"by brand", catalog.Filter{Brand: "payot"}, []string{"payot-hydra-24-cream"}},
		{"by line", catalog.Filter{Brand: "davines", Line: "energizing"}, []string{"davines-energizing-shampoo"}},
		{"by category", catalog.Filter{Category: "shampoo"}, []string{"davines-minu-shampoo", "davines-energizing-shampoo"}},
		{"search", catalog.Filter{Query: "hydra"}, []string{"payot-hydra-24-cream"}},
		{"search escapes wildcards", catalog.Filter{Query: "%"}, nil},
		{"paging", catalog.Filter{Page: 2, PageSize: 3}, []string{"davines-energizing-shampoo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := repo.Products(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Products() error = %v", err)
			}
			var got []string
			for _, p := range page.Items {
				got = append(got, p.Slug)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("slugs = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("slugs = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestProductsSummaryFields(t *testing.T) {
	repo := newRepo(t)

	page, err := repo.Products(context.Background(), catalog.Filter{Line: "energizing"})
	if err != nil {
		t.Fatalf("Products() error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("total = %d, want 1", page.Total)
	}
	p := page.Items[0]
	if p.PriceFrom != 9000 || !p.HasOptions || !p.InStock || p.Line == nil || p.ImageURL == "" {
		t.Errorf("summary = %+v", p)
	}
}

func TestProductsHidesProductsWithoutActiveVariants(t *testing.T) {
	pool := dbtest.New(t)
	dbtest.Seed(t, pool)
	dbtest.Exec(t, pool, `UPDATE product_variants SET is_active = false
		WHERE product_id = (SELECT id FROM products WHERE slug = 'payot-hydra-24-cream')`)
	repo := catalog.NewPostgres(pool)

	page, err := repo.Products(context.Background(), catalog.Filter{Brand: "payot"})
	if err != nil {
		t.Fatalf("Products() error = %v", err)
	}
	if page.Total != 0 {
		t.Errorf("total = %d, want 0", page.Total)
	}
	if _, err := repo.Product(context.Background(), "payot-hydra-24-cream"); !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("Product() error = %v, want ErrNotFound", err)
	}
}

func TestProduct(t *testing.T) {
	repo := newRepo(t)

	p, err := repo.Product(context.Background(), "davines-energizing-shampoo")
	if err != nil {
		t.Fatalf("Product() error = %v", err)
	}
	if p.OptionKind != catalog.OptionVolume || p.Category == nil || p.Category.Slug != "shampoo" {
		t.Errorf("product = %+v", p)
	}
	if len(p.Variants) != 3 || p.Variants[0].Label != "100 ml" {
		t.Fatalf("variants = %+v, want 3 sorted by sort", p.Variants)
	}
	big := p.Variants[2]
	if big.CompareAtPrice == nil || *big.CompareAtPrice != 54000 {
		t.Errorf("1000 ml compare price = %v, want 54000", big.CompareAtPrice)
	}
	if len(p.Media) != 1 {
		t.Errorf("media = %d, want 1", len(p.Media))
	}

	if _, err := repo.Product(context.Background(), "missing"); !errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("Product(missing) error = %v, want ErrNotFound", err)
	}
}

func TestProductVariantImage(t *testing.T) {
	pool := dbtest.New(t)
	dbtest.Seed(t, pool)
	dbtest.Exec(t, pool, `UPDATE product_variants SET image_url = '/uploads/250.jpg' WHERE label = '250 ml'`)

	p, err := catalog.NewPostgres(pool).Product(context.Background(), "davines-energizing-shampoo")
	if err != nil {
		t.Fatalf("Product() error = %v", err)
	}
	if p.Variants[0].ImageURL != "" || p.Variants[1].ImageURL != "/uploads/250.jpg" {
		t.Errorf("variants = %+v, want a photo only on 250 ml", p.Variants)
	}
}
