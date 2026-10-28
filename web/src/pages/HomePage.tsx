import { Link } from 'react-router-dom'
import { useBanners, useMenu, useProducts } from '../api/queries'
import { BannerCarousel } from '../components/BannerCarousel'
import { ProductGrid, ProductGridSkeleton } from '../components/ProductCard'
import { routes } from '../lib/site'

export function HomePage() {
  const banners = useBanners()
  const menu = useMenu()
  const latest = useProducts({ pageSize: 8 })

  return (
    <div className="container-page space-y-14 pt-4 sm:pt-6">
      {banners.isPending ? (
        <div className="aspect-[4/5] animate-pulse rounded-3xl bg-sand sm:aspect-[16/6]" />
      ) : (
        <BannerCarousel banners={banners.data ?? []} />
      )}

      {menu.data && menu.data.brands.length > 0 && (
        <section>
          <SectionTitle title="Бренды" />
          <div className="flex flex-wrap gap-2">
            {menu.data.brands.map((b) => (
              <Link
                key={b.id}
                to={routes.brand(b.slug)}
                className="rounded-full border border-line bg-white px-5 py-2.5 text-sm hover:border-ink"
              >
                {b.name}
              </Link>
            ))}
          </div>
        </section>
      )}

      <section>
        <SectionTitle title="Новинки" link={{ to: routes.catalog, label: 'Весь каталог' }} />
        {latest.isPending ? (
          <ProductGridSkeleton />
        ) : latest.isError ? (
          <p className="text-sm text-sale">Не удалось загрузить товары</p>
        ) : (
          <ProductGrid products={latest.data.items} />
        )}
      </section>

      {menu.data && menu.data.categories.length > 0 && (
        <section>
          <SectionTitle title="Средства" />
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
            {menu.data.categories.map((c) => (
              <Link
                key={c.id}
                to={routes.category(c.slug)}
                className="rounded-2xl bg-sand px-4 py-6 text-center text-sm font-medium hover:bg-line"
              >
                {c.name}
              </Link>
            ))}
          </div>
        </section>
      )}
    </div>
  )
}

function SectionTitle({ title, link }: { title: string; link?: { to: string; label: string } }) {
  return (
    <div className="mb-5 flex items-end justify-between">
      <h2 className="text-3xl sm:text-4xl">{title}</h2>
      {link && (
        <Link
          to={link.to}
          className="text-sm text-muted underline-offset-4 hover:text-ink hover:underline"
        >
          {link.label}
        </Link>
      )}
    </div>
  )
}
