import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { WhatsAppIcon } from '../../components/icons'
import { formatTenge } from '../../lib/format'
import { whatsappLink } from '../../lib/site'
import { adminApi } from '../api'
import { Badge, Loading } from '../components/ui'
import { errorMessage } from '../errors'
import { DELIVERY_LABELS, formatDate, ORDER_STATUSES, statusInfo } from '../orderStatus'
import type { OrderStatus } from '../types'

export function OrderDetailPage() {
  const id = Number(useParams().id)
  const qc = useQueryClient()
  const order = useQuery({ queryKey: ['admin', 'order', id], queryFn: () => adminApi.order(id) })

  const setStatus = useMutation({
    mutationFn: (status: OrderStatus) => adminApi.setOrderStatus(id, status),
    onSuccess: (o) => {
      qc.setQueryData(['admin', 'order', id], o)
      void qc.invalidateQueries({ queryKey: ['admin', 'orders'] })
    },
  })

  if (order.isPending) return <Loading />
  if (order.isError) return <p className="text-sm text-sale">Заказ не найден</p>

  const o = order.data
  const s = statusInfo(o.status)
  const greeting = `Здравствуйте, ${o.customerName}! Это 12 Beauty Stories, по вашему заказу ${o.number} на сумму ${formatTenge(o.total)}.`

  let statusError = errorMessage(setStatus.error)
  if (setStatus.error instanceof ApiError && setStatus.error.code === 'conflict') {
    statusError = 'Нельзя вернуть заказ из отмены: часть товаров уже закончилась'
  }

  return (
    <div className="max-w-3xl">
      <Link to="/admin/orders" className="text-sm text-muted hover:text-ink">
        ← Все заказы
      </Link>
      <div className="mt-3 mb-6 flex flex-wrap items-center gap-3">
        <h1 className="text-3xl sm:text-4xl">Заказ {o.number}</h1>
        <Badge tone={s.tone}>{s.label}</Badge>
      </div>

      <div className="grid gap-6 md:grid-cols-[1fr_260px]">
        <div className="space-y-6">
          <section className="rounded-2xl bg-white p-5">
            <h2 className="mb-3 text-xl">Клиент</h2>
            <p className="font-medium">{o.customerName}</p>
            <a href={`tel:${o.phone}`} className="text-sm text-muted hover:text-ink">
              {o.phone}
            </a>
            <p className="mt-3 text-sm">
              {DELIVERY_LABELS[o.delivery]}
              {o.address && `: ${o.address}`}
            </p>
            {o.comment && <p className="mt-3 rounded-xl bg-sand p-3 text-sm">💬 {o.comment}</p>}
            <a
              href={whatsappLink(o.phone, greeting)}
              target="_blank"
              rel="noreferrer"
              className="btn-primary mt-4 bg-[#1f8f4e] hover:bg-[#17733e]"
            >
              <WhatsAppIcon width={18} height={18} /> Написать в WhatsApp
            </a>
          </section>

          <section className="rounded-2xl bg-white p-5">
            <h2 className="mb-3 text-xl">Состав</h2>
            <ul className="divide-y divide-line text-sm">
              {o.items.map((it, i) => (
                <li key={i} className="flex justify-between gap-4 py-2">
                  <span>
                    {it.productName}
                    {it.variantLabel && `, ${it.variantLabel}`}
                    <span className="text-muted">
                      {' '}
                      × {it.quantity} по {formatTenge(it.unitPrice)}
                    </span>
                  </span>
                  <span className="shrink-0">{formatTenge(it.lineTotal)}</span>
                </li>
              ))}
            </ul>
            <dl className="mt-3 space-y-1 border-t border-line pt-3 text-sm">
              <div className="flex justify-between">
                <dt className="text-muted">Сумма</dt>
                <dd>{formatTenge(o.subtotal)}</dd>
              </div>
              {o.discountTotal > 0 && (
                <div className="flex justify-between text-accent-dark">
                  <dt>Скидка{o.promoCode && ` (промокод ${o.promoCode})`}</dt>
                  <dd>{formatTenge(-o.discountTotal)}</dd>
                </div>
              )}
              <div className="flex justify-between text-base font-semibold">
                <dt>Итого</dt>
                <dd>{formatTenge(o.total)}</dd>
              </div>
            </dl>
          </section>
        </div>

        <aside className="h-fit space-y-3 rounded-2xl bg-white p-5">
          <h2 className="text-xl">Статус</h2>
          <p className="text-xs text-muted">{formatDate(o.createdAt)}</p>
          <div className="flex flex-col gap-2">
            {ORDER_STATUSES.map((st) => (
              <button
                key={st.value}
                type="button"
                disabled={setStatus.isPending || st.value === o.status}
                onClick={() => setStatus.mutate(st.value)}
                className={`rounded-xl border px-4 py-2 text-left text-sm transition-colors ${
                  st.value === o.status
                    ? 'border-ink bg-ink text-cream'
                    : 'border-line hover:border-ink disabled:opacity-50'
                }`}
              >
                {st.label}
              </button>
            ))}
          </div>
          <p className="text-xs text-muted">При отмене товары возвращаются на склад.</p>
          {statusError && <p className="text-sm text-sale">{statusError}</p>}
        </aside>
      </div>
    </div>
  )
}
