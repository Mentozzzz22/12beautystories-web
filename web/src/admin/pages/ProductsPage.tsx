import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { type ReactNode, useEffect, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { formatTenge, plural } from '../../lib/format'
import { errorMessage } from '../errors'
import { adminApi, useBrands, useSale } from '../api'
import { Badge, Empty, Loading, Modal, NumberField, PageHeader } from '../components/ui'
import { salePrice } from '../sale'
import type { ProductListItem } from '../types'

const productsWord = (n: number) => `${n} ${plural(n, ['товар', 'товара', 'товаров'])}`
const pricesWord = (n: number) => `${n} ${plural(n, ['цена', 'цены', 'цен'])}`
const hasSale = (p: ProductListItem) => p.salePriceFrom !== null

export function ProductsPage() {
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const brandId = Number(params.get('brandId')) || 0
  const lineId = Number(params.get('lineId')) || 0
  const onSale = params.get('sale') === '1'
  const page = Math.max(1, Number(params.get('page')) || 1)
  const [search, setSearch] = useState(q)
  const brands = useBrands().list.data ?? []
  const lines = brands
    .filter((b) => !brandId || b.id === brandId)
    .flatMap((b) =>
      b.lines.map((l) => ({ ...l, label: brandId ? l.name : `${b.name} · ${l.name}` })),
    )

  // Selection survives filters and pages, so a sale can cover several lines at once.
  const [selected, setSelected] = useState<ReadonlyMap<number, ProductListItem>>(new Map())
  const [action, setAction] = useState<'apply' | 'clear' | null>(null)
  const [message, setMessage] = useState('')

  const filter = { q, brandId, lineId, onSale, page }
  const products = useQuery({
    queryKey: ['admin', 'products', filter],
    queryFn: () => adminApi.products(filter),
    placeholderData: keepPreviousData,
  })

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(params)
    if (value && value !== '0') next.set(key, value)
    else next.delete(key)
    if (key === 'brandId') next.delete('lineId')
    next.delete('page')
    setParams(next)
  }
  const pages = products.data ? Math.ceil(products.data.total / products.data.pageSize) : 0
  const items = products.data?.items ?? []

  const toggle = (list: ProductListItem[], on: boolean) => {
    setMessage('')
    setSelected((prev) => {
      const next = new Map(prev)
      for (const p of list) {
        if (on) next.set(p.id, p)
        else next.delete(p.id)
      }
      return next
    })
  }
  const done = (text: string) => {
    setAction(null)
    setSelected(new Map())
    setMessage(text)
  }

  return (
    <div className={selected.size ? 'pb-28' : ''}>
      <PageHeader
        title="Товары"
        action={
          <Link to="/admin/products/new" className="btn-primary">
            + Добавить товар
          </Link>
        }
      />

      <div className="mb-3 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
        <form
          className="flex-1 sm:min-w-64"
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
          className="input sm:w-48"
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
        <select
          className="input sm:w-72"
          value={lineId}
          onChange={(e) => setParam('lineId', e.target.value)}
          aria-label="Линия"
        >
          <option value={0}>Все линии</option>
          {lines.map((l) => (
            <option key={l.id} value={l.id}>
              {l.label}
            </option>
          ))}
        </select>
      </div>
      <label className="mb-6 inline-flex cursor-pointer items-center gap-2 text-sm">
        <input
          type="checkbox"
          className="size-4 accent-ink"
          checked={onSale}
          onChange={(e) => setParam('sale', e.target.checked ? '1' : '')}
        />
        Только со скидкой
      </label>

      {message && (
        <p role="status" className="mb-4 rounded-2xl bg-sand px-4 py-3 text-sm">
          {message}
        </p>
      )}

      {products.isPending ? (
        <Loading />
      ) : products.isError ? (
        <p className="text-sm text-sale">Не удалось загрузить товары</p>
      ) : items.length === 0 ? (
        <Empty>Товаров не найдено</Empty>
      ) : (
        <div className="overflow-hidden rounded-2xl border border-line bg-white">
          <div className="border-b border-line px-4 py-3">
            <Check
              checked={items.every((p) => selected.has(p.id))}
              partly={items.some((p) => selected.has(p.id))}
              onChange={(on) => toggle(items, on)}
              label={<span className="text-sm text-muted">Выбрать все на странице</span>}
            />
          </div>
          <ul className="divide-y divide-line">
            {items.map((p) => (
              <li key={p.id} className="flex items-center gap-3 pl-4 hover:bg-sand">
                <Check
                  checked={selected.has(p.id)}
                  onChange={(on) => toggle([p], on)}
                  label={<span className="sr-only">Выбрать {p.name}</span>}
                />
                <ProductRow product={p} />
              </li>
            ))}
          </ul>
        </div>
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

      {selected.size > 0 && (
        <div className="fixed inset-x-0 bottom-0 z-20 border-t border-line bg-cream/95 backdrop-blur lg:left-[240px]">
          <div className="flex flex-wrap items-center gap-3 px-4 py-3 sm:px-8">
            <span className="text-sm">Выбрано {productsWord(selected.size)}</span>
            <button
              type="button"
              className="text-sm text-muted underline-offset-2 hover:underline"
              onClick={() => setSelected(new Map())}
            >
              Снять выделение
            </button>
            <ActionMenu canClear={[...selected.values()].some(hasSale)} onPick={setAction} />
          </div>
        </div>
      )}

      {action && (
        <SaleDialog
          action={action}
          products={[...selected.values()]}
          onClose={() => setAction(null)}
          onDone={done}
        />
      )}
    </div>
  )
}

function ProductRow({ product: p }: { product: ProductListItem }) {
  const from = p.variantCount > 1 ? 'от ' : ''
  return (
    <Link
      to={`/admin/products/${p.id}`}
      className="flex min-w-0 flex-1 items-center gap-4 py-3 pr-4"
    >
      <span className="h-14 w-14 shrink-0 overflow-hidden rounded-xl bg-white">
        {p.imageUrl && <img src={p.imageUrl} alt="" className="h-full w-full object-contain" />}
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium">{p.name}</span>
        <span className="block truncate text-xs text-muted">
          {[p.brandName, p.lineName, p.categoryNames].filter(Boolean).join(' · ')}
        </span>
      </span>
      <span className="hidden text-right text-sm sm:block">
        {p.priceFrom !== null && `${from}${formatTenge(p.priceFrom)}`}
        {p.salePriceFrom !== null && (
          <span className="block text-xs text-sale">
            Скидка: {from}
            {formatTenge(p.salePriceFrom)}
          </span>
        )}
        <span className={`block text-xs ${p.totalStock > 0 ? 'text-muted' : 'text-sale'}`}>
          {p.totalStock > 0 ? `В наличии: ${p.totalStock}` : 'Нет в наличии'}
        </span>
      </span>
      {!p.isActive && <Badge tone="gray">Скрыт</Badge>}
    </Link>
  )
}

function Check({
  checked,
  partly = false,
  onChange,
  label,
}: {
  checked: boolean
  partly?: boolean
  onChange: (on: boolean) => void
  label: ReactNode
}) {
  const ref = useRef<HTMLInputElement>(null)
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = partly && !checked
  }, [partly, checked])
  return (
    <label className="flex cursor-pointer items-center gap-3 py-2">
      <input
        ref={ref}
        type="checkbox"
        className="size-4 shrink-0 accent-ink"
        checked={checked}
        onChange={() => onChange(!checked)}
      />
      {label}
    </label>
  )
}

function ActionMenu({
  canClear,
  onPick,
}: {
  canClear: boolean
  onPick: (a: 'apply' | 'clear') => void
}) {
  const [open, setOpen] = useState(false)
  const pick = (a: 'apply' | 'clear') => {
    setOpen(false)
    onPick(a)
  }
  return (
    <div className="relative ml-auto">
      <button
        type="button"
        className="btn-primary"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
      >
        Действие ▾
      </button>
      {open && (
        <>
          <div className="fixed inset-0" onClick={() => setOpen(false)} />
          <div className="absolute right-0 bottom-full mb-2 w-56 rounded-xl border border-line bg-white p-1 shadow-lg">
            <MenuItem onClick={() => pick('apply')}>Поставить скидку %</MenuItem>
            <MenuItem disabled={!canClear} onClick={() => pick('clear')}>
              Убрать скидку
              {!canClear && (
                <span className="block text-xs text-muted">У выбранных товаров нет скидки</span>
              )}
            </MenuItem>
          </div>
        </>
      )}
    </div>
  )
}

function MenuItem({
  disabled = false,
  onClick,
  children,
}: {
  disabled?: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      className="block w-full rounded-lg px-3 py-2 text-left text-sm enabled:hover:bg-sand disabled:cursor-not-allowed disabled:text-muted"
      onClick={onClick}
    >
      {children}
    </button>
  )
}

function SaleDialog({
  action,
  products,
  onClose,
  onDone,
}: {
  action: 'apply' | 'clear'
  products: ProductListItem[]
  onClose: () => void
  onDone: (message: string) => void
}) {
  const { apply, clear } = useSale()
  const [percent, setPercent] = useState<number | null>(null)
  const valid = percent !== null && percent >= 1 && percent <= 99
  const mutation = action === 'apply' ? apply : clear
  // Clearing only touches products that have a sale; a new sale replaces an old one, never adds up.
  const shown = action === 'clear' ? products.filter(hasSale) : products
  const productIds = shown.map((p) => p.id)
  const replaced = action === 'apply' ? products.filter(hasSale).length : 0

  const submit = () => {
    if (action === 'clear') {
      clear.mutate(
        { productIds },
        { onSuccess: (r) => onDone(`Скидка убрана: обновлено ${pricesWord(r.updated)}`) },
      )
    } else if (valid) {
      apply.mutate(
        { productIds, percent },
        {
          onSuccess: (r) =>
            onDone(`Скидка −${percent}% поставлена: обновлено ${pricesWord(r.updated)}`),
        },
      )
    }
  }

  return (
    <Modal open title={action === 'apply' ? 'Поставить скидку' : 'Убрать скидку'} onClose={onClose}>
      <form
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault()
          submit()
        }}
      >
        {action === 'apply' ? (
          <NumberField
            label="Скидка"
            suffix="%"
            value={percent}
            onChange={setPercent}
            hint="Цена со скидкой посчитается для всех объёмов. Товары со скидкой не участвуют в наборах."
            error={percent !== null && !valid ? 'От 1 до 99%' : undefined}
          />
        ) : (
          <p className="text-sm">
            Скидка будет убрана: {productsWord(shown.length)}, останется обычная цена.
            {shown.length < products.length &&
              ` Без скидки и не изменятся: ${productsWord(products.length - shown.length)}.`}
          </p>
        )}
        {replaced > 0 && (
          <p className="rounded-xl bg-sand px-3 py-2 text-sm">
            Скидка уже есть: {productsWord(replaced)}. Она заменится новой, считается от обычной
            цены, скидки не складываются.
          </p>
        )}
        <ul className="max-h-64 divide-y divide-line overflow-y-auto rounded-xl border border-line bg-white text-sm">
          {shown.map((p) => {
            const next = valid && p.priceFrom !== null ? salePrice(p.priceFrom, percent) : null
            return (
              <li key={p.id} className="flex justify-between gap-3 px-3 py-2">
                <span className="min-w-0 truncate">{p.name}</span>
                {p.priceFrom !== null && (
                  <span className="shrink-0 text-muted">
                    {p.variantCount > 1 && 'от '}
                    {formatTenge(p.priceFrom)}
                    {next !== null ? (
                      <span className="font-medium text-sale"> → {formatTenge(next)}</span>
                    ) : (
                      p.salePriceFrom !== null && (
                        <span className="text-sale"> · сейчас {formatTenge(p.salePriceFrom)}</span>
                      )
                    )}
                  </span>
                )}
              </li>
            )
          })}
        </ul>
        {mutation.error && (
          <p role="alert" className="text-sm text-sale">
            {errorMessage(mutation.error)}
          </p>
        )}
        <button
          type="submit"
          className="btn-primary w-full"
          disabled={mutation.isPending || (action === 'apply' && !valid)}
        >
          {action === 'apply' ? 'Поставить скидку' : 'Убрать скидку'}
        </button>
      </form>
    </Modal>
  )
}
