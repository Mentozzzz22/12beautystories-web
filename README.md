# 12 Beauty Stories Shop

Self-hosted online store for **12 Beauty Stories**, a professional hair and skin care boutique in Astana, Kazakhstan.
It replaces a paid site builder with a small Go + PostgreSQL backend and a React storefront.

> Status: storefront and admin panel MVP work end to end; production setup is ready, see [docs/DEPLOY.md](docs/DEPLOY.md). See [docs/ROADMAP.md](docs/ROADMAP.md).

## Features

- Catalog navigation by **brand → product line** and by **product category**
- Products with multiple **variants** (volume or colour), each with its own price, sale price and stock
- Home page **banners** with scheduling
- **Pricing engine**: promo codes plus toggleable automatic rules such as "Davines from 50 000 ₸ → −10%",
  with explicit stacking rules
- Checkout with server-side price recalculation and atomic stock reservation
- **Telegram notification** to the owner for every new order
- **Admin panel** (`/admin`, mobile friendly): orders with status flow and WhatsApp link, products with
  variants and photo gallery, brands and lines, categories, banners with scheduling, promo codes,
  discount rules with an on/off switch
- Cancelling an order returns items to stock; reopening reserves them again
- Admin sessions: bcrypt passwords, SHA-256 hashed session tokens, `HttpOnly` + `SameSite=Strict` cookie,
  login rate limiting
- Image uploads are decoded, downscaled and re-encoded, which strips EXIF data and rejects SVG/HTML
- Link previews: the Go server serves the built web app and fills page-specific title and Open Graph
  tags (product photo, name, price), plus `sitemap.xml` and `robots.txt`
- Production: 40 MB distroless non-root image, Caddy with automatic HTTPS, daily database and photo backups

## Tech stack

| Layer | Tools |
|---|---|
| Backend | Go, `net/http`, `pgx`, `goose` (embedded migrations), `slog` |
| Database | PostgreSQL 17 |
| Frontend | React 19, Vite, TypeScript (strict), React Router, TanStack Query, Tailwind CSS v4 |
| Infra | Docker Compose, Caddy, GitHub Actions |

## Project layout

```
backend/
  cmd/server/          entry point
  cmd/admin/           CLI to create admin accounts
  internal/
    admin/             back office repository and validation
    apperr/            shared validation / not found / conflict errors
    auth/              admin accounts and sessions
    catalog/           storefront read model (menu, banners, products)
    frontend/          serves the web build with SEO tags, sitemap, robots
    config/            environment configuration
    database/          pgx pool and embedded SQL migrations
    httpapi/           JSON API handlers and middleware (storefront + admin)
    media/             image upload processing
    notify/            Telegram notifications
    order/             cart quote and checkout service
    pricing/           pure discount calculation
  db/seed.sql          demo data for local development
web/
  src/api/             typed API client and queries
  src/components/      layout, burger menu, banners, product cards
  src/lib/             cart store (localStorage), formatting
  src/pages/           home, catalog, product, cart, checkout
  src/admin/           admin panel (lazy-loaded chunk)
deploy/                Caddyfile, backup script
Dockerfile             multi-stage build (web + Go) into a distroless image
docker-compose.prod.yml
docs/ROADMAP.md, docs/DEPLOY.md
```

## Getting started

Requirements: Go 1.27+, Node.js 22+, Docker. Postgres is exposed on port 5433 to avoid clashing with a local installation.

```bash
cp .env.example .env
docker compose up -d db
cd backend
DATABASE_URL=postgres://shop:shop@localhost:5433/shop?sslmode=disable go run ./cmd/server
```

The server applies migrations on start. Load demo data:

```bash
docker compose exec -T db psql -U shop -d shop < backend/db/seed.sql
```

Create an admin account (the password is read from the environment, not from flags):

```bash
cd backend
ADMIN_PASSWORD='choose-a-strong-one' go run ./cmd/admin create -email owner@example.com
```

Then sign in at http://localhost:5173/admin.

Run the storefront (proxies `/api` to the Go server on :8080):

```bash
cd web
npm install
npm run dev
```

## API

| Method | Path | Description |
|---|---|---|
| GET | `/api/health` | Liveness check |
| GET | `/api/menu` | Brands with lines and categories |
| GET | `/api/banners` | Active home page banners |
| GET | `/api/products?brand=&line=&category=&q=&page=&pageSize=` | Product list |
| GET | `/api/products/{slug}` | Product details with media and variants |
| POST | `/api/cart/quote` | Price a cart: `{"items":[{"variantId":1,"quantity":2}],"promoCode":"WELCOME5"}` |
| POST | `/api/orders` | Place an order: quote body plus `customer` (`name`, `phone`, `delivery`, `address`, `comment`) |

Admin endpoints live under `/api/admin/*` (login, CRUD for brands, lines, categories, products, banners,
promos and rules, orders with status updates, image uploads) and require the session cookie.

Errors use one shape: `{"code":"insufficient_stock","message":"...","fields":{...}}`.

## Development

```bash
cd backend
go vet ./...
go test ./...                       # unit tests
TEST_DATABASE_URL=postgres://shop:shop@localhost:5433/shop?sslmode=disable go test ./...  # + integration tests
golangci-lint run

cd ../web
npm run lint && npm test && npm run build
```

## License

Private project. All rights reserved.
