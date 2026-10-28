# 12 Beauty Stories Shop

Self-hosted online store for **12 Beauty Stories**, a professional hair and skin care boutique in Astana, Kazakhstan.
It replaces a paid site builder with a small Go + PostgreSQL backend and a React storefront.

> Status: storefront MVP works end to end; admin panel is next. See [docs/ROADMAP.md](docs/ROADMAP.md).

## Features

- Catalog navigation by **brand → product line** and by **product category**
- Products with multiple **variants** (volume or colour), each with its own price, sale price and stock
- Home page **banners** with scheduling
- **Pricing engine**: promo codes plus toggleable automatic rules such as "Davines from 50 000 ₸ → −10%",
  with explicit stacking rules
- Checkout with server-side price recalculation and atomic stock reservation
- **Telegram notification** to the owner for every new order

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
  internal/
    catalog/           storefront read model (menu, banners, products)
    config/            environment configuration
    database/          pgx pool and embedded SQL migrations
    httpapi/           JSON API handlers and middleware
    notify/            Telegram notifications
    order/             cart quote and checkout service
    pricing/           pure discount calculation
  db/seed.sql          demo data for local development
web/
  src/api/             typed API client and queries
  src/components/      layout, burger menu, banners, product cards
  src/lib/             cart store (localStorage), formatting
  src/pages/           home, catalog, product, cart, checkout
docs/ROADMAP.md
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
