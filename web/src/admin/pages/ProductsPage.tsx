import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { formatTenge } from '../../lib/format'
import { adminApi, useBrands } from '../api'
import { Badge, Empty, Loading, PageHeader } from '../components/ui'

export function ProductsPage() {
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const brandId = Number(params.get('brandId')) || 0
  const page = Math.max(1, Number(params.get('page')) || 1)
  const [search, setSearch] = useState(q)
  const brands = useBrands().list.data ?? []

  const products = useQuery({
    queryKey: ['admin', 'products', q, brandId, page],
    queryFn: () => adminApi.products(q, brandId, page),
    placeholderData: keepPreviousData,
  })

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(params)
    if (value && value !== '0') next.set(key, value)
    else next.delete(key)
    next.delete('page')
    setParams(next)
  }
  const pages = products.data ? Math.ceil(products.data.total / products.data.pageSize) : 0

  return (
    <div>
      <PageHeader
        title="Товары"
        action={
          <Link to="/admin/products/new" className="btn-primary">
            + Добавить товар
          </Link>
        }
      />

      <div className="mb-6 flex flex-col gap-3 sm:flex-row">
        <form
          className="flex-1"
          onSubmit={(e) => {
            e.preventDefault()
            setParam('q', search.trim())
          }}
        >
          <input
            type="search"
            className="input"
            placeholder="Поиск по названию, бренду, линии"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </form>
        <select
          className="input sm:w-56"
          value={brandId}
          onChange={(e) => setParam('brandId', e.target.value)}
          aria-label="Бренд"
        >
          <option value={0}>Все бренды</option>
          {brands.map((b) => (
            <option key={b.id} value={b.id}>
              {b.name}
            </option>
          ))}
        </select>
      </div>

      {products.isPending ? (
        <Loading />
      ) : products.isError ? (
        <p className="text-sm text-sale">Не удалось загрузить товары</p>
      ) : products.data.items.length === 0 ? (
        <Empty>Товаров не найдено</Empty>
      ) : (
        <ul className="divide-y divide-line overflow-hidden rounded-2xl border border-line bg-white">
          {products.data.items.map((p) => (
            <li key={p.id}>
              <Link
                to={`/admin/products/${p.id}`}
                className="flex items-center gap-4 px-4 py-3 hover:bg-sand"
              >
                <span className="h-14 w-14 shrink-0 overflow-hidden rounded-xl bg-sand">
                  {p.imageUrl && (
                    <img src={p.imageUrl} alt="" className="h-full w-full object-cover" />
                  )}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium">{p.name}</span>
                  <span className="block truncate text-xs text-muted">
                    {[p.brandName, p.lineName, p.categoryNames].filter(Boolean).join(' · ')}
                  </span>
                </span>
                <span className="hidden text-right text-sm sm:block">
                  {p.priceFrom !== null && (
                    <>
                      {p.variantCount > 1 && 'от '}
                      {formatTenge(p.priceFrom)}
                    </>
                  )}
                  <span
                    className={`block text-xs ${p.totalStock > 0 ? 'text-muted' : 'text-sale'}`}
                  >
                    {p.totalStock > 0 ? `В наличии: ${p.totalStock}` : 'Нет в наличии'}
                  </span>
                </span>
                {!p.isActive && <Badge tone="gray">Скрыт</Badge>}
              </Link>
            </li>
          ))}
        </ul>
      )}

      {pages > 1 && (
        <div className="mt-6 flex justify-center gap-2">
          {Array.from({ length: pages }, (_, i) => i + 1).map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => {
                const next = new URLSearchParams(params)
                next.set('page', String(p))
                setParams(next)
              }}
              className={`h-9 w-9 rounded-full text-sm ${p === page ? 'bg-ink text-cream' : 'hover:bg-sand'}`}
            >
              {p}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
