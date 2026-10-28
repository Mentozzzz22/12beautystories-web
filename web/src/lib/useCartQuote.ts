import { useMemo } from 'react'
import { ApiError } from '../api/client'
import { useQuote } from '../api/queries'
import type { CartItemInput } from '../api/types'
import { useCart } from './cart'

const PROMO_MESSAGES = {
  min_subtotal_not_met: 'Сумма заказа меньше минимальной для этого промокода',
  not_better: 'Действующая скидка выгоднее, промокод не применён',
  invalid: 'Промокод не найден или истёк',
} as const

/** Prices the current cart on the server. An invalid promo is reported but does not block the cart. */
export function useCartQuote() {
  const state = useCart()
  const items = useMemo<CartItemInput[]>(
    () => state.items.map(({ variantId, quantity }) => ({ variantId, quantity })),
    [state.items],
  )

  const withPromo = useQuote(items, state.promoCode)
  const promoInvalid =
    withPromo.error instanceof ApiError && withPromo.error.code === 'invalid_promo'
  // Fall back to a quote without the code so totals and rule discounts still show.
  const withoutPromo = useQuote(promoInvalid ? items : [], '')
  const active = promoInvalid ? withoutPromo : withPromo

  const status = withPromo.data?.quote.promoStatus
  let promoMessage: string | null = null
  if (state.promoCode) {
    if (promoInvalid) promoMessage = PROMO_MESSAGES.invalid
    else if (status === 'min_subtotal_not_met' || status === 'not_better')
      promoMessage = PROMO_MESSAGES[status]
  }

  const fallbackSubtotal = state.items.reduce((sum, i) => sum + i.price * i.quantity, 0)

  return {
    state,
    items,
    quote: active.data?.quote,
    isLoading: active.isFetching,
    error: active.error instanceof ApiError ? active.error : null,
    promoApplied: status === 'applied' && !promoInvalid,
    promoMessage,
    fallbackSubtotal,
  }
}
