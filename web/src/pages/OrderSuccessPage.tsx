import { useEffect } from 'react'
import { Link, Navigate, useLocation } from 'react-router-dom'
import type { Order } from '../api/types'
import { OrderSummary } from '../components/OrderSummary'
import { cart } from '../lib/cart'
import { routes, site } from '../lib/site'

export function OrderSuccessPage() {
  const order = (useLocation().state as { order?: Order } | null)?.order

  useEffect(() => {
    if (order) cart.clear()
  }, [order])

  if (!order) return <Navigate to={routes.home} replace />

  return (
    <div className="container-page max-w-xl py-16 text-center">
      <p className="text-sm tracking-[0.14em] text-muted uppercase">Заказ {order.number}</p>
      <h1 className="mt-3 text-4xl sm:text-5xl">Спасибо за заказ!</h1>
      <p className="mt-4 text-sm leading-relaxed text-muted">
        Мы напишем вам в WhatsApp на номер {order.customer.phone}, подтвердим наличие и выставим
        счёт в Kaspi.
      </p>

      <div className="mt-10 rounded-3xl bg-sand p-6 text-left">
        <ul className="mb-5 space-y-2 text-sm">
          {order.lines.map((l) => (
            <li key={l.variantId} className="flex justify-between gap-4">
              <span>
                {l.productName}
                {l.variantLabel && `, ${l.variantLabel}`} × {l.quantity}
              </span>
            </li>
          ))}
        </ul>
        <OrderSummary quote={order.quote} fallbackSubtotal={order.quote.subtotal} />
      </div>

      <div className="mt-10 flex flex-col justify-center gap-3 sm:flex-row">
        <Link to={routes.catalog} className="btn-primary">
          Продолжить покупки
        </Link>
        <a href={site.instagramUrl} target="_blank" rel="noreferrer" className="btn-outline">
          Наш Instagram
        </a>
      </div>
    </div>
  )
}
