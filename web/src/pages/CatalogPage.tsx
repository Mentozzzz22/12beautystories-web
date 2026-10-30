import { Link, useParams, useSearchParams } from 'react-router-dom'
import { useMenu, useProducts } from '../api/queries'
import type { Menu } from '../api/types'
import { ProductGrid, ProductGridSkeleton } from '../components/ProductCard'
import { RichText } from '../components/RichText'
import { plural } from '../lib/format'
import { routes } from '../lib/site'

const PAGE_SIZE = 24

interface Chip {
  key: string
  label: string
  to: string
  active: boolean
}

/** One page for /catalog, /brands/:brand, /brands/:brand/:line and /categories/:category. */
export function CatalogPage() {
  const params = useParams<{ brand?: string; line?: string; category?: string }>()
  const [search] = useSearchParams()
  const menu = useMenu()

  const brand = params.brand ?? search.get('brand') ?? ''
  const line = params.line ?? ''
  const category = params.category ?? search.get('category') ?? ''
  const q = search.get('q') ?? ''
  const page = Math.max(1, Number(search.get('page')) || 1)

  const products = useProducts({ brand, line, category, q, page, pageSize: PAGE_SIZE })

  const heading = describe(menu.data, { brand, line, category, q })
  const chips = buildChips(menu.data, {
    brand,
    line,
    category,
    fromBrandRoute: Boolean(params.brand),
  })
  const total = products.data?.total ?? 0
  const pages = Math.ceil(total / PAGE_SIZE)

  return (
    <div className="container-page pt-8">
      <nav aria-label="Навигация" className="text-xs text-muted">
        <Link to={routes.home} className="hover:text-ink">
          Главная
        </Link>
        {heading.crumbs.map((c) => (
          <span key={c.to}>
            {' / '}
            <Link to={c.to} className="hover:text-ink">
              {c.label}
            </Link>
          </span>
        ))}
      </nav>

      <div className="mt-3 flex items-center gap-5">
        {heading.image && (
          <img
            src={heading.image}
            alt=""
            className="h-20 w-20 shrink-0 rounded-2xl border border-line bg-white object-contain p-2 sm:h-28 sm:w-28"
          />
        )}
        <div>
          <h1 className="text-4xl sm:text-5xl">{heading.title}</h1>
          {heading.description && (
            <p className="mt-3 max-w-2xl text-sm whitespace-pre-line text-muted">
              <RichText text={heading.description} />
            </p>
          )}
        </div>
      </div>

      {heading.cover && (
        <img
          src={heading.cover}
          alt={heading.title}
          className="mt-6 aspect-[16/9] w-full rounded-3xl object-cover sm:aspect-[16/6]"
        />
      )}

      {chips.length > 0 && (
        <div className="-mx-4 mt-6 flex gap-2 overflow-x-auto px-4 pb-1 sm:mx-0 sm:flex-wrap sm:px-0">
          {chips.map((c) => (
            <Link
              key={c.key}
              to={c.to}
              className={`shrink-0 rounded-full border px-4 py-2 text-sm transition-colors ${
                c.active ? 'border-ink bg-ink text-cream' : 'border-line bg-white hover:border-ink'
              }`}
            >
              {c.label}
            </Link>
          ))}
        </div>
      )}

      <p className="mt-6 mb-4 text-sm text-muted">
        {products.data && `${total} ${plural(total, ['товар', 'товара', 'товаров'])}`}
      </p>

      {products.isPending ? (
        <ProductGridSkeleton />
      ) : products.isError ? (
        <p className="text-sm text-sale">Не удалось загрузить товары</p>
      ) : products.data.items.length === 0 ? (
        <div className="rounded-2xl bg-sand p-10 text-center">
          <p className="font-serif text-2xl">Ничего не нашлось</p>
          <Link to={routes.catalog} className="btn-outline mt-5">
            Смотреть весь каталог
          </Link>
        </div>
      ) : (
        <ProductGrid products={products.data.items} />
      )}

      {pages > 1 && <Pagination page={page} pages={pages} />}
    </div>
  )
}

function Pagination({ page, pages }: { page: number; pages: number }) {
  const [search] = useSearchParams()
  const href = (p: number) => {
    const next = new URLSearchParams(search)
    next.set('page', String(p))
    return `?${next.toString()}`
  }
  return (
    <nav aria-label="Страницы" className="mt-10 flex justify-center gap-2">
      {Array.from({ length: pages }, (_, i) => i + 1).map((p) => (
        <Link
          key={p}
          to={href(p)}
          aria-current={p === page ? 'page' : undefined}
          className={`flex h-10 w-10 items-center justify-center rounded-full text-sm ${
            p === page ? 'bg-ink text-cream' : 'hover:bg-sand'
          }`}
        >
          {p}
        </Link>
      ))}
    </nav>
  )
}

interface Selection {
  brand: string
  line: string
  category: string
  q?: string
}

function describe(menu: Menu | undefined, s: Selection) {
  const brand = menu?.brands.find((b) => b.slug === s.brand)
  const line = brand?.lines.find((l) => l.slug === s.line)
  const category = menu?.categories.find((c) => c.slug === s.category)
  const crumbs: { label: string; to: string }[] = []

  if (brand) crumbs.push({ label: brand.name, to: routes.brand(brand.slug) })
  if (brand && line) crumbs.push({ label: line.name, to: routes.line(brand.slug, line.slug) })
  if (!brand && category) crumbs.push({ label: category.name, to: routes.category(category.slug) })

  let title = 'Каталог'
  if (s.q) title = `Поиск: «${s.q}»`
  else if (line && brand) title = `${brand.name} ${line.name}`
  else if (brand) title = brand.name
  else if (category) title = category.name

  const description = line?.description || (!line ? brand?.description : '') || ''
  const image = (s.q ? '' : brand?.logoUrl) || ''
  const cover = (s.q ? '' : line?.imageUrl) || ''
  return { title, crumbs, description, image, cover }
}

function buildChips(menu: Menu | undefined, s: Selection & { fromBrandRoute: boolean }): Chip[] {
  if (!menu) return []
  const brand = menu.brands.find((b) => b.slug === s.brand)

  // Brand pages: switch between product lines.
  if (s.fromBrandRoute && brand && brand.lines.length > 0) {
    return [
      { key: 'all', label: 'Все линии', to: routes.brand(brand.slug), active: !s.line },
      ...brand.lines.map((l) => ({
        key: l.slug,
        label: l.name,
        to: routes.line(brand.slug, l.slug),
        active: l.slug === s.line,
      })),
    ]
  }

  // Catalog and category pages: switch between product types.
  if (!s.fromBrandRoute) {
    return [
      { key: 'all', label: 'Все', to: routes.catalog, active: !s.category },
      ...menu.categories.map((c) => ({
        key: c.slug,
        label: c.name,
        to: routes.category(c.slug),
        active: c.slug === s.category,
      })),
    ]
  }
  return []
}
