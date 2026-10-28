import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { formatTenge } from '../../lib/format'
import { adminApi } from '../api'
import { Badge, Empty, Loading, PageHeader } from '../components/ui'
import { DELIVERY_LABELS, formatDate, ORDER_STATUSES, statusInfo } from '../orderStatus'

export function OrdersPage() {
  const [params, setParams] = useSearchParams()
  const status = params.get('status') ?? ''
  const page = Math.max(1, Number(params.get('page')) || 1)
  const [q, setQ] = useState(params.get('q') ?? '')

  const orders = useQuery({
    queryKey: ['admin', 'orders', status, params.get('q') ?? '', page],
    queryFn: () => adminApi.orders(status, params.get('q') ?? '', page),
    placeholderData: keepPreviousData,
    refetchInterval: 60_000,
  })

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    next.delete('page')
    setParams(next)
  }

  const pages = orders.data ? Math.ceil(orders.data.total / orders.data.pageSize) : 0

  return (
    <div>
      <PageHeader title="Заказы" />

      <div className="mb-4 flex gap-2 overflow-x-auto pb-1">
        {[{ value: '', label: 'Все' }, ...ORDER_STATUSES].map((s) => (
          <button
            key={s.value}
            type="button"
            onClick={() => setParam('status', s.value)}
            className={`shrink-0 rounded-full border px-4 py-1.5 text-sm ${
              status === s.value ? 'border-ink bg-ink text-cream' : 'border-line bg-white'
            }`}
          >
            {s.label}
            {s.value === 'new' &&
              orders.data &&
              orders.data.newCount > 0 &&
              ` · ${orders.data.newCount}`}
          </button>
        ))}
      </div>

      <form
        className="mb-6"
        onSubmit={(e) => {
          e.preventDefault()
          setParam('q', q.trim())
        }}
      >
        <input
          type="search"
          className="input max-w-md"
          placeholder="Имя, телефон или номер заказа"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </form>

      {orders.isPending ? (
        <Loading />
      ) : orders.isError ? (
        <p className="text-sm text-sale">Не удалось загрузить заказы</p>
      ) : orders.data.items.length === 0 ? (
        <Empty>Заказов пока нет</Empty>
      ) : (
        <ul className="divide-y divide-line overflow-hidden rounded-2xl border border-line bg-white">
          {orders.data.items.map((o) => {
            const s = statusInfo(o.status)
            return (
              <li key={o.id}>
                <Link
                  to={`/admin/orders/${o.id}`}
                  className="grid grid-cols-[1fr_auto] items-center gap-x-4 gap-y-1 px-4 py-3 hover:bg-sand sm:grid-cols-[110px_1fr_140px_120px_110px]"
                >
                  <span className="text-sm font-medium">{o.number}</span>
                  <span className="justify-self-end sm:order-last">
                    <Badge tone={s.tone}>{s.label}</Badge>
                  </span>
                  <span className="text-sm">
                    {o.customerName}
                    <span className="block text-xs text-muted">{o.phone}</span>
                  </span>
                  <span className="text-xs text-muted sm:text-sm">
                    {DELIVERY_LABELS[o.delivery]} · {o.itemCount} шт.
                  </span>
                  <span className="text-sm font-medium sm:text-right">
                    {formatTenge(o.total)}
                    <span className="block text-xs font-normal text-muted">
                      {formatDate(o.createdAt)}
                    </span>
                  </span>
                </Link>
              </li>
            )
          })}
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
