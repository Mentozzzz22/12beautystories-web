import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { formatTenge, plural } from '../../lib/format'
import { adminApi } from '../api'
import { Badge, Empty, Loading, PageHeader } from '../components/ui'
import { formatDay } from '../orderStatus'
import type { CustomerFilter, CustomerSort } from '../types'

const SORTS: { value: CustomerSort; label: string }[] = [
  { value: 'recent', label: 'Недавние заказы' },
  { value: 'spent', label: 'Больше всего купили' },
  { value: 'orders', label: 'Больше всего заказов' },
  { value: 'inactive', label: 'Давно не заказывали' },
]

function isSort(v: string | null): v is CustomerSort {
  return SORTS.some((s) => s.value === v)
}

export function CustomersPage() {
  const [params, setParams] = useSearchParams()
  const sortParam = params.get('sort')
  const filter: CustomerFilter = {
    q: params.get('q') ?? '',
    marketing: params.get('marketing') === '1',
    sort: isSort(sortParam) ? sortParam : 'recent',
    page: Math.max(1, Number(params.get('page')) || 1),
  }
  const [q, setQ] = useState(filter.q)

  const customers = useQuery({
    queryKey: ['admin', 'customers', filter],
    queryFn: () => adminApi.customers(filter),
    placeholderData: keepPreviousData,
  })

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    next.delete('page')
    setParams(next)
  }

  const pages = customers.data ? Math.ceil(customers.data.total / customers.data.pageSize) : 0

  return (
    <div>
      <PageHeader
        title="Клиенты"
        action={
          <a href={adminApi.customersExportUrl(filter)} className="btn-outline py-2" download>
            Скачать CSV
          </a>
        }
      />

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <form
          className="min-w-0 flex-1 basis-64"
          onSubmit={(e) => {
            e.preventDefault()
            setParam('q', q.trim())
          }}
        >
          <input
            type="search"
            className="input"
            placeholder="Имя или телефон"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </form>
        <select
          aria-label="Сортировка"
          className="input w-auto"
          value={filter.sort}
          onChange={(e) => setParam('sort', e.target.value === 'recent' ? '' : e.target.value)}
        >
          {SORTS.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
      </div>

      <div className="mb-6 flex gap-2">
        {[
          { value: false, label: 'Все' },
          { value: true, label: 'Согласны на рассылку' },
        ].map((f) => (
          <button
            key={f.label}
            type="button"
            onClick={() => setParam('marketing', f.value ? '1' : '')}
            className={`rounded-full border px-4 py-1.5 text-sm ${
              filter.marketing === f.value ? 'border-ink bg-ink text-cream' : 'border-line bg-white'
            }`}
          >
            {f.label}
          </button>
        ))}
        {customers.data && (
          <span className="self-center text-sm text-muted">
            {customers.data.total} {plural(customers.data.total, ['клиент', 'клиента', 'клиентов'])}
          </span>
        )}
      </div>

      {customers.isPending ? (
        <Loading />
      ) : customers.isError ? (
        <p className="text-sm text-sale">Не удалось загрузить клиентов</p>
      ) : customers.data.items.length === 0 ? (
        <Empty>Клиентов пока нет: они появятся после первого заказа</Empty>
      ) : (
        <ul className="divide-y divide-line overflow-hidden rounded-2xl border border-line bg-white">
          {customers.data.items.map((c) => (
            <li key={c.id}>
              <Link
                to={`/admin/customers/${c.id}`}
                className="grid grid-cols-[1fr_auto] items-center gap-x-4 gap-y-1 px-4 py-3 hover:bg-sand sm:grid-cols-[1fr_120px_130px_120px]"
              >
                <span className="text-sm">
                  <span className="font-medium">{c.name}</span>
                  {c.marketingConsent && (
                    <span className="ml-2 align-middle">
                      <Badge tone="green">рассылка</Badge>
                    </span>
                  )}
                  <span className="block text-xs text-muted">{c.phone}</span>
                </span>
                <span className="text-right text-sm font-medium sm:order-3">
                  {formatTenge(c.totalSpent)}
                </span>
                <span className="text-xs text-muted sm:order-2 sm:text-sm">
                  {c.orderCount} {plural(c.orderCount, ['заказ', 'заказа', 'заказов'])}
                </span>
                <span className="text-right text-xs text-muted sm:order-4 sm:text-sm">
                  {c.lastOrderAt ? formatDay(c.lastOrderAt) : '—'}
                </span>
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
              className={`h-9 w-9 rounded-full text-sm ${p === filter.page ? 'bg-ink text-cream' : 'hover:bg-sand'}`}
            >
              {p}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
