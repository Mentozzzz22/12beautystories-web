import { Link } from 'react-router-dom'
import type { Brand } from '../api/types'
import { routes } from '../lib/site'

/** Home page brand grid: the logo when the brand has one, otherwise its name. */
export function BrandTiles({ brands }: { brands: Brand[] }) {
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
      {brands.map((b) => (
        <Link
          key={b.id}
          to={routes.brand(b.slug)}
          className="flex aspect-[3/2] items-center justify-center rounded-2xl border border-line bg-white p-4 transition-colors hover:border-ink"
        >
          {b.logoUrl ? (
            <img
              src={b.logoUrl}
              alt={b.name}
              loading="lazy"
              className="h-full w-full object-contain"
            />
          ) : (
            <span className="text-center font-serif text-2xl">{b.name}</span>
          )}
        </Link>
      ))}
    </div>
  )
}
