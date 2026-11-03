import { type ReactNode, useEffect, useMemo, useRef, useState } from 'react'
import { formatTenge, plural } from '../../lib/format'
import { useSale } from '../api'
import { salePrice } from '../sale'
import { Empty, Loading, NumberField, PageHeader } from '../components/ui'
import type { SaleItem } from '../types'

interface LineGroup {
  key: string
  name: string
  items: SaleItem[]
}

interface BrandGroup {
  id: number
  name: string
  lines: LineGroup[]
}

/** Keeps the server order (brand, line, product) and nests products under brands and lines. */
function groupByBrand(items: SaleItem[]): BrandGroup[] {
  const brands: BrandGroup[] = []
  for (const it of items) {
    let brand = brands.at(-1)
    if (brand?.id !== it.brandId) {
      brand = { id: it.brandId, name: it.brandName, lines: [] }
      brands.push(brand)
    }
    const key = `${it.brandId}-${it.lineId}`
    let line = brand.lines.at(-1)
    if (line?.key !== key) {
      line = { key, name: it.lineName || 'Без линии', items: [] }
      brand.lines.push(line)
    }
    line.items.push(it)
  }
  return brands
}

const ids = (items: SaleItem[]) => items.map((it) => it.id)
const onSale = (it: SaleItem) => it.variants.some((v) => v.salePrice !== null)
const prices = (n: number) => `${n} ${plural(n, ['цена', 'цены', 'цен'])}`

export function SalePage() {
  const { items, apply, clear } = useSale()
  const [selected, setSelected] = useState<ReadonlySet<number>>(new Set())
  const [percent, setPercent] = useState<number | null>(null)
  const [message, setMessage] = useState('')
  const brands = useMemo(() => groupByBrand(items.data ?? []), [items.data])

  const toggle = (productIds: number[], on: boolean) => {
    setMessage('')
    setSelected((prev) => {
      const next = new Set(prev)
      for (const id of productIds) {
        if (on) next.add(id)
        else next.delete(id)
      }
      return next
    })
  }

  const validPercent = percent !== null && percent >= 1 && percent <= 99
  const preview = validPercent ? percent : null
  const busy = apply.isPending || clear.isPending
  const failed = apply.isError || clear.isError

  const done = (text: string) => {
    setSelected(new Set())
    setMessage(text)
  }
  const applySale = () => {
    if (!validPercent) return
    apply.mutate(
      { productIds: [...selected], percent },
      { onSuccess: (r) => done(`Скидка −${percent}% поставлена: обновлено ${prices(r.updated)}`) },
    )
  }
  const clearSale = () => {
    if (!window.confirm('Убрать скидку у выбранных товаров?')) return
    clear.mutate(
      { productIds: [...selected] },
      { onSuccess: (r) => done(`Скидка убрана: обновлено ${prices(r.updated)}`) },
    )
  }

  return (
    <div className="max-w-3xl">
      <PageHeader title="Скидки на товары" />
      <p className="mb-4 text-sm text-muted">
        Отметьте бренд, линию или отдельные товары, укажите скидку и нажмите «Поставить скидку».
        Цена со скидкой посчитается сама и округлится до 10 ₸. Товары со скидкой не участвуют в
        скидках на наборы.
      </p>

      {items.isPending ? (
        <Loading />
      ) : items.isError ? (
        <p className="text-sm text-sale">Не удалось загрузить товары</p>
      ) : !brands.length ? (
        <Empty>Товаров пока нет</Empty>
      ) : (
        <>
          <button
            type="button"
            className="mb-3 text-sm underline-offset-2 hover:underline"
            onClick={() => toggle(ids((items.data ?? []).filter(onSale)), true)}
          >
            Отметить все товары со скидкой
          </button>
          <div className="space-y-4 pb-52 lg:pb-40">
            {brands.map((b) => {
              const brandIds = b.lines.flatMap((l) => ids(l.items))
              return (
                <section key={b.id} className="rounded-2xl bg-white p-4">
                  <Check
                    ids={brandIds}
                    selected={selected}
                    onChange={toggle}
                    label={<span className="font-semibold">{b.name}</span>}
                  />
                  <div className="mt-3 space-y-4 pl-7">
                    {b.lines.map((l) => (
                      <div key={l.key}>
                        <Check
                          ids={ids(l.items)}
                          selected={selected}
                          onChange={toggle}
                          label={<span className="font-medium">{l.name}</span>}
                        />
                        <ul className="mt-2 space-y-2 pl-7">
                          {l.items.map((it) => (
                            <li key={it.id}>
                              <ProductRow
                                item={it}
                                selected={selected}
                                onChange={toggle}
                                preview={selected.has(it.id) ? preview : null}
                              />
                            </li>
                          ))}
                        </ul>
                      </div>
                    ))}
                  </div>
                </section>
              )
            })}
          </div>
        </>
      )}

      <div className="fixed inset-x-0 bottom-0 border-t border-line bg-cream/95 backdrop-blur lg:left-[240px]">
        <div className="mx-auto flex max-w-3xl flex-wrap items-end gap-3 p-4">
          <div className="w-32">
            <NumberField label="Скидка" suffix="%" value={percent} onChange={setPercent} />
          </div>
          <button
            type="button"
            className="btn-primary"
            disabled={!selected.size || !validPercent || busy}
            onClick={applySale}
          >
            Поставить скидку
          </button>
          <button
            type="button"
            className="btn-outline"
            disabled={!selected.size || busy}
            onClick={clearSale}
          >
            Убрать скидку
          </button>
          <p className="basis-full text-sm" role="status">
            {failed ? (
              <span className="text-sale">Не удалось сохранить, попробуйте ещё раз</span>
            ) : message ? (
              message
            ) : (
              <span className="text-muted">
                Выбрано {selected.size} {plural(selected.size, ['товар', 'товара', 'товаров'])}
                {percent !== null && !validPercent && ' · скидка от 1 до 99%'}
              </span>
            )}
          </p>
        </div>
      </div>
    </div>
  )
}

function Check({
  ids,
  selected,
  onChange,
  label,
}: {
  ids: number[]
  selected: ReadonlySet<number>
  onChange: (ids: number[], on: boolean) => void
  label: ReactNode
}) {
  const ref = useRef<HTMLInputElement>(null)
  const count = ids.filter((id) => selected.has(id)).length
  const all = count > 0 && count === ids.length
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = count > 0 && !all
  }, [count, all])

  return (
    <label className="flex cursor-pointer items-center gap-3 text-sm">
      <input
        ref={ref}
        type="checkbox"
        className="size-4 shrink-0 accent-ink"
        checked={all}
        onChange={() => onChange(ids, !all)}
      />
      {label}
    </label>
  )
}

function ProductRow({
  item,
  selected,
  onChange,
  preview,
}: {
  item: SaleItem
  selected: ReadonlySet<number>
  onChange: (ids: number[], on: boolean) => void
  preview: number | null
}) {
  return (
    <Check
      ids={[item.id]}
      selected={selected}
      onChange={onChange}
      label={
        <span className="flex min-w-0 items-center gap-3">
          <span className="size-10 shrink-0 overflow-hidden rounded-lg border border-line bg-white">
            {item.imageUrl && (
              <img src={item.imageUrl} alt="" className="h-full w-full object-contain" />
            )}
          </span>
          <span className="min-w-0">
            <span className="block">{item.name}</span>
            <span className="block text-xs text-muted">
              {item.variants.map((v, i) => {
                const next = preview === null ? null : salePrice(v.price, preview)
                return (
                  <span key={i} className="mr-3 inline-block">
                    {v.label && `${v.label}: `}
                    {v.salePrice !== null ? (
                      <>
                        <s>{formatTenge(v.price)}</s>{' '}
                        <span className="text-sale">{formatTenge(v.salePrice)}</span>
                      </>
                    ) : (
                      formatTenge(v.price)
                    )}
                    {next !== null && (
                      <span className="font-medium text-accent"> → {formatTenge(next)}</span>
                    )}
                  </span>
                )
              })}
            </span>
          </span>
        </span>
      }
    />
  )
}
