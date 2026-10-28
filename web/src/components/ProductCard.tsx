import { Link } from 'react-router-dom'
import type { ProductSummary } from '../api/types'
import { routes } from '../lib/site'
import { Price } from './Price'

export function ProductCard({ product }: { product: ProductSummary }) {
  const onSale = product.compareAtPrice !== undefined
  return (
    <Link to={routes.product(product.slug)} className="group flex flex-col">
      <div className="relative aspect-square overflow-hidden rounded-2xl bg-sand">
        {product.imageUrl && (
          <img
            src={product.imageUrl}
            alt={product.name}
            loading="lazy"
            className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
          />
        )}
        {onSale && (
          <span className="absolute top-3 left-3 rounded-full bg-sale px-2.5 py-1 text-[11px] font-semibold text-white">
            SALE
          </span>
        )}
        {!product.inStock && (
          <span className="absolute inset-x-3 bottom-3 rounded-full bg-cream/90 py-1.5 text-center text-xs font-medium">
            Нет в наличии
          </span>
        )}
      </div>
      <p className="mt-3 text-[11px] font-medium tracking-[0.12em] text-muted uppercase">
        {product.brand.name}
        {product.line && ` · ${product.line.name}`}
      </p>
      <h3 className="mt-1 font-sans text-sm leading-snug font-medium">{product.name}</h3>
      <Price
        price={product.priceFrom}
        compareAtPrice={product.compareAtPrice}
        from={product.hasOptions}
        className="mt-1.5 text-sm"
      />
    </Link>
  )
}

export function ProductGrid({ products }: { products: ProductSummary[] }) {
  return (
    <div className="grid grid-cols-2 gap-x-4 gap-y-8 sm:grid-cols-3 lg:grid-cols-4">
      {products.map((p) => (
        <ProductCard key={p.id} product={p} />
      ))}
    </div>
  )
}

export function ProductGridSkeleton({ count = 8 }: { count?: number }) {
  return (
    <div
      className="grid grid-cols-2 gap-x-4 gap-y-8 sm:grid-cols-3 lg:grid-cols-4"
      aria-hidden="true"
    >
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="animate-pulse">
          <div className="aspect-square rounded-2xl bg-sand" />
          <div className="mt-3 h-3 w-1/2 rounded bg-sand" />
          <div className="mt-2 h-4 w-3/4 rounded bg-sand" />
        </div>
      ))}
    </div>
  )
}
