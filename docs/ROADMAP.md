# Roadmap

Online store for **12 Beauty Stories** (professional hair & skin care, Astana).
Goal: replace the paid site builder with a cheap self-hosted shop. Sales today happen in Instagram DMs
with Kaspi invoices, so phase 1 keeps that flow and only collects orders in a structured way.

## Architecture

| Layer | Choice |
|---|---|
| Backend | Go monolith, std `net/http` routing, `pgx` (hand-written SQL), `goose` embedded migrations |
| Database | PostgreSQL 17 |
| Frontend | React + Vite + TypeScript SPA (storefront + `/admin`), PWA |
| Link previews | Go serves the SPA and injects Open Graph tags for product pages |
| Notifications | Telegram bot message to the owner on every new order |
| Hosting | Single VPS in Kazakhstan, Docker Compose + Caddy (auto HTTPS) |

Money is stored as whole tenge in `bigint`.

## Phase 0: Foundation
- [x] Repository layout, README, lint configs, `.env.example`
- [x] Docker Compose with Postgres for local development
- [x] CI: `go vet`, `golangci-lint`, `go test`, web lint/test/build

## Phase 1: MVP (replace the old site)
Backend
- [x] Schema: brands, product lines, categories, products, media, variants, banners, promo codes, discount rules, orders
- [x] Public catalog API: menu, banners, product list with filters, product details
- [x] Pricing engine: promo codes, toggleable brand threshold rules (e.g. "Davines from 50 000 ₸ → −10%")
- [x] Cart quote + order placement (stock check, server-side price recalculation)
- [x] Telegram notification on new order
- [x] Repository integration tests against real Postgres
- [x] Admin auth (single owner account, session cookie)
- [x] Admin API: CRUD for catalog, banners, promos, rules; order list and status
- [x] Image upload with resizing (JPEG, max 1600 px; Go has no WebP encoder in the standard library)

Storefront
- [x] Home: 3–4 banner carousel, new arrivals
- [x] Burger menu: brands → lines, product categories
- [x] Catalog page with brand / line / category filters
- [x] Product page: gallery, description, active ingredients, variant picker (volume or colour), price + sale price
- [x] Cart (localStorage), promo code, checkout form, success page with Kaspi instructions

Admin
- [x] Products with variants and photos, photo ordering
- [x] Banners, promo codes, discount rules with on/off switch
- [x] Orders: list, status, customer contacts
- [x] Customers: history and stats per phone, notes, promotions consent, CSV export
- [x] Privacy policy page and checkout consents

Release
- [x] Open Graph tags for products, sitemap.xml, robots.txt
- [x] Production image, Compose + Caddy, daily `pg_dump` backups (see DEPLOY.md)
- [ ] Rent the VPS and deploy
- [ ] Domain switch from the old builder

## Phase 2: Growth
- [ ] Online payment (Kaspi Pay or Freedom Pay)
- [ ] Customer accounts by phone (OTP via Telegram / WhatsApp)
- [ ] Bonus program: welcome bonus, cashback, first-order discount
- [ ] PWA install + Web Push campaigns

## Phase 3: Nice to have
- [ ] Product videos
- [ ] Stock sync / export to Kaspi Marketplace
- [ ] Analytics dashboard for the owner
