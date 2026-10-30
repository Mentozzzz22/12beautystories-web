import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ApiError } from '../api/client'
import { useProduct } from '../api/queries'
import type { Media, Product, Variant } from '../api/types'
import { WhatsAppIcon } from '../components/icons'
import { Price } from '../components/Price'
import { RichText } from '../components/RichText'
import { cart } from '../lib/cart'
import { routes, whatsappLink } from '../lib/site'
import { NotFoundPage } from './NotFoundPage'

export function ProductPage() {
  const { slug = '' } = useParams()
  const { data: product, isPending, error } = useProduct(slug)

  if (isPending) {
    return (
      <div className="container-page grid animate-pulse gap-8 pt-8 md:grid-cols-2">
        <div className="aspect-square rounded-3xl bg-sand" />
        <div className="space-y-4">
          <div className="h-4 w-1/3 rounded bg-sand" />
          <div className="h-10 w-2/3 rounded bg-sand" />
        </div>
      </div>
    )
  }
  if (error instanceof ApiError && error.status === 404) return <NotFoundPage />
  if (error) {
    return <p className="container-page pt-8 text-sm text-sale">Не удалось загрузить товар</p>
  }
  return <ProductView key={product.id} product={product} />
}

export function ProductView({ product }: { product: Product }) {
  const firstAvailable = product.variants.find((v) => v.inStock) ?? product.variants[0]
  const [variantId, setVariantId] = useState(firstAvailable?.id)
  const [added, setAdded] = useState(false)
  const variant = product.variants.find((v) => v.id === variantId) ?? firstAvailable

  if (!variant) return null

  const addToCart = () => {
    cart.add({
      variantId: variant.id,
      productSlug: product.slug,
      productName: product.name,
      brandName: product.brand.name,
      variantLabel: variant.label,
      price: variant.price,
      imageUrl: product.media.find((m) => m.kind === 'image')?.url ?? '',
    })
    setAdded(true)
  }

  return (
    <div className="container-page pt-6">
      <nav aria-label="Навигация" className="text-xs text-muted">
        <Link to={routes.brand(product.brand.slug)} className="hover:text-ink">
          {product.brand.name}
        </Link>
        {product.line && (
          <>
            {' / '}
            <Link
              to={routes.line(product.brand.slug, product.line.slug)}
              className="hover:text-ink"
            >
              {product.line.name}
            </Link>
          </>
        )}
      </nav>

      <div className="mt-4 grid gap-8 md:grid-cols-2 md:gap-12">
        <Gallery media={product.media} alt={product.name} />

        <div>
          <p className="text-xs font-medium tracking-[0.14em] text-muted uppercase">
            {product.brand.name}
            {product.line && ` · ${product.line.name}`}
          </p>
          <h1 className="mt-2 text-4xl leading-tight sm:text-5xl">{product.name}</h1>

          <Price
            price={variant.price}
            compareAtPrice={variant.compareAtPrice}
            className="mt-4 text-2xl font-medium"
          />

          {product.optionKind !== 'none' && product.variants.length > 1 && (
            <VariantPicker
              kind={product.optionKind}
              variants={product.variants}
              selectedId={variant.id}
              onSelect={(id) => {
                setVariantId(id)
                setAdded(false)
              }}
            />
          )}

          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <button
              type="button"
              className="btn-primary flex-1"
              disabled={!variant.inStock}
              onClick={addToCart}
            >
              {variant.inStock ? 'Добавить в корзину' : 'Нет в наличии'}
            </button>
            {added && (
              <Link to={routes.cart} className="btn-outline flex-1">
                Перейти в корзину
              </Link>
            )}
          </div>
          {added && (
            <p role="status" className="mt-3 text-sm text-accent-dark">
              Товар добавлен в корзину
            </p>
          )}
          <a
            href={whatsappLink(
              undefined,
              `Здравствуйте! Вопрос по товару: ${product.name}${variant.label ? `, ${variant.label}` : ''}`,
            )}
            target="_blank"
            rel="noreferrer"
            className="mt-4 inline-flex items-center gap-2 text-sm text-muted hover:text-ink"
          >
            <WhatsAppIcon width={18} height={18} /> Задать вопрос в WhatsApp
          </a>

          {product.description && (
            <section className="mt-10">
              <h2 className="text-2xl">Описание</h2>
              <p className="mt-3 text-sm leading-relaxed whitespace-pre-line">
                <RichText text={product.description} />
              </p>
            </section>
          )}
          {product.ingredients && (
            <section className="mt-8">
              <h2 className="text-2xl">Активные ингредиенты</h2>
              <p className="mt-3 text-sm leading-relaxed whitespace-pre-line">
                <RichText text={product.ingredients} />
              </p>
            </section>
          )}
        </div>
      </div>
    </div>
  )
}

interface VariantPickerProps {
  kind: 'volume' | 'color'
  variants: Variant[]
  selectedId: number
  onSelect: (id: number) => void
}

export function VariantPicker({ kind, variants, selectedId, onSelect }: VariantPickerProps) {
  const label = kind === 'volume' ? 'Объём' : 'Цвет'
  return (
    <fieldset className="mt-6">
      <legend className="mb-3 text-sm font-medium">{label}</legend>
      <div className="flex flex-wrap gap-2">
        {variants.map((v) => (
          <label
            key={v.id}
            className={`cursor-pointer rounded-full border px-5 py-2.5 text-sm transition-colors has-focus-visible:ring-2 has-focus-visible:ring-accent ${
              v.id === selectedId
                ? 'border-ink bg-ink text-cream'
                : 'border-line bg-white hover:border-ink'
            } ${v.inStock ? '' : 'text-muted line-through'}`}
          >
            <input
              type="radio"
              name="variant"
              value={v.id}
              checked={v.id === selectedId}
              onChange={() => onSelect(v.id)}
              className="sr-only"
            />
            {v.label}
          </label>
        ))}
      </div>
    </fieldset>
  )
}

function Gallery({ media, alt }: { media: Media[]; alt: string }) {
  const [active, setActive] = useState(0)
  const current = media[active]

  return (
    <div>
      <div className="aspect-square overflow-hidden rounded-3xl bg-sand">
        {current?.kind === 'video' ? (
          <video src={current.url} controls playsInline className="h-full w-full object-cover" />
        ) : current ? (
          <img src={current.url} alt={alt} className="h-full w-full object-cover" />
        ) : null}
      </div>
      {media.length > 1 && (
        <div className="mt-3 flex gap-2 overflow-x-auto">
          {media.map((m, i) => (
            <button
              key={m.url}
              type="button"
              onClick={() => setActive(i)}
              aria-label={`Фото ${i + 1}`}
              aria-current={i === active}
              className={`h-20 w-20 shrink-0 overflow-hidden rounded-xl border-2 ${
                i === active ? 'border-ink' : 'border-transparent'
              }`}
            >
              {m.kind === 'video' ? (
                <span className="flex h-full w-full items-center justify-center bg-sand text-xs">
                  Видео
                </span>
              ) : (
                <img src={m.url} alt="" className="h-full w-full object-cover" />
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
