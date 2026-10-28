import { useState } from 'react'
import { Link } from 'react-router-dom'
import { OrderSummary } from '../components/OrderSummary'
import { cart, type CartItem, MAX_QUANTITY } from '../lib/cart'
import { formatTenge } from '../lib/format'
import { routes } from '../lib/site'
import { useCartQuote } from '../lib/useCartQuote'

export function CartPage() {
  const { state, quote, error, promoApplied, promoMessage, fallbackSubtotal } = useCartQuote()

  if (state.items.length === 0) {
    return (
      <div className="container-page py-20 text-center">
        <h1 className="text-4xl">Корзина пуста</h1>
        <p className="mt-3 text-sm text-muted">Загляните в каталог, там много интересного</p>
        <Link to={routes.catalog} className="btn-primary mt-8">
          В каталог
        </Link>
      </div>
    )
  }

  return (
    <div className="container-page pt-8">
      <h1 className="text-4xl sm:text-5xl">Корзина</h1>

      <div className="mt-8 grid gap-10 lg:grid-cols-[1fr_360px]">
        <ul className="divide-y divide-line border-y border-line">
          {state.items.map((item) => (
            <CartRow key={item.variantId} item={item} />
          ))}
        </ul>

        <aside className="h-fit space-y-6 rounded-3xl bg-sand p-6">
          <PromoForm initial={state.promoCode} applied={promoApplied} message={promoMessage} />
          <OrderSummary quote={quote} fallbackSubtotal={fallbackSubtotal} />
          {error && error.code !== 'invalid_promo' && (
            <p role="alert" className="text-sm text-sale">
              {error.code === 'variant_unavailable'
                ? 'Один из товаров больше недоступен, удалите его из корзины'
                : 'Не удалось рассчитать стоимость'}
            </p>
          )}
          <Link to={routes.checkout} className="btn-primary w-full">
            Оформить заказ
          </Link>
        </aside>
      </div>
    </div>
  )
}

function CartRow({ item }: { item: CartItem }) {
  return (
    <li className="flex gap-4 py-5">
      <Link
        to={routes.product(item.productSlug)}
        className="h-24 w-24 shrink-0 overflow-hidden rounded-2xl bg-sand"
      >
        {item.imageUrl && <img src={item.imageUrl} alt="" className="h-full w-full object-cover" />}
      </Link>
      <div className="flex flex-1 flex-col">
        <p className="text-[11px] font-medium tracking-[0.12em] text-muted uppercase">
          {item.brandName}
        </p>
        <Link to={routes.product(item.productSlug)} className="text-sm font-medium hover:underline">
          {item.productName}
        </Link>
        {item.variantLabel && <p className="text-xs text-muted">{item.variantLabel}</p>}

        <div className="mt-auto flex items-center justify-between pt-3">
          <div className="flex items-center rounded-full border border-line bg-white">
            <button
              type="button"
              aria-label="Уменьшить количество"
              className="h-9 w-9"
              onClick={() => cart.setQuantity(item.variantId, item.quantity - 1)}
            >
              −
            </button>
            <span className="w-8 text-center text-sm" aria-label="Количество">
              {item.quantity}
            </span>
            <button
              type="button"
              aria-label="Увеличить количество"
              className="h-9 w-9 disabled:opacity-40"
              disabled={item.quantity >= MAX_QUANTITY}
              onClick={() => cart.setQuantity(item.variantId, item.quantity + 1)}
            >
              +
            </button>
          </div>
          <div className="text-right">
            <p className="text-sm font-medium">{formatTenge(item.price * item.quantity)}</p>
            <button
              type="button"
              onClick={() => cart.remove(item.variantId)}
              className="text-xs text-muted underline-offset-2 hover:text-ink hover:underline"
            >
              Удалить
            </button>
          </div>
        </div>
      </div>
    </li>
  )
}

function PromoForm({
  initial,
  applied,
  message,
}: {
  initial: string
  applied: boolean
  message: string | null
}) {
  const [code, setCode] = useState(initial)

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        cart.setPromoCode(code)
      }}
    >
      <label htmlFor="promo" className="text-sm font-medium">
        Промокод
      </label>
      <div className="mt-2 flex gap-2">
        <input
          id="promo"
          value={code}
          onChange={(e) => setCode(e.target.value.toUpperCase())}
          placeholder="Введите код"
          className="input uppercase placeholder:normal-case"
          autoComplete="off"
        />
        {initial ? (
          <button
            type="button"
            className="btn-outline px-4"
            onClick={() => {
              setCode('')
              cart.setPromoCode('')
            }}
          >
            Сбросить
          </button>
        ) : (
          <button type="submit" className="btn-outline px-4" disabled={!code.trim()}>
            Применить
          </button>
        )}
      </div>
      {applied && <p className="mt-2 text-xs text-accent-dark">Промокод применён</p>}
      {message && <p className="mt-2 text-xs text-sale">{message}</p>}
    </form>
  )
}
