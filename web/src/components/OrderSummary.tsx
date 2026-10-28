import type { Quote } from '../api/types'
import { formatTenge } from '../lib/format'

export function OrderSummary({
  quote,
  fallbackSubtotal,
}: {
  quote?: Quote
  fallbackSubtotal: number
}) {
  const subtotal = quote?.subtotal ?? fallbackSubtotal
  return (
    <dl className="space-y-2 text-sm">
      <div className="flex justify-between">
        <dt className="text-muted">Сумма</dt>
        <dd>{formatTenge(subtotal)}</dd>
      </div>
      {quote?.discounts.map((d) => (
        <div key={`${d.source}-${d.label}`} className="flex justify-between gap-4 text-accent-dark">
          <dt>{d.source === 'promo' ? `Промокод ${d.label}` : d.label}</dt>
          <dd className="shrink-0">{formatTenge(-d.amount)}</dd>
        </div>
      ))}
      <div className="flex justify-between border-t border-line pt-3 text-base font-semibold">
        <dt>Итого</dt>
        <dd>{formatTenge(quote?.total ?? subtotal)}</dd>
      </div>
    </dl>
  )
}
