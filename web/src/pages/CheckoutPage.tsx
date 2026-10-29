import { useMutation } from '@tanstack/react-query'
import { type FormEvent, type ReactNode, useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import type { Customer, DeliveryMethod } from '../api/types'
import { OrderSummary } from '../components/OrderSummary'
import { formatPhoneInput, isCompletePhone, plural } from '../lib/format'
import { routes } from '../lib/site'
import { useCartQuote } from '../lib/useCartQuote'

const DELIVERY: { value: DeliveryMethod; label: string; hint: string }[] = [
  { value: 'pickup', label: 'Самовывоз', hint: 'Астана, адрес пришлём в WhatsApp' },
  { value: 'courier', label: 'Курьер по Астане', hint: 'Стоимость уточним при подтверждении' },
  { value: 'post', label: 'Доставка по Казахстану', hint: 'Казпочта или СДЭК' },
]

const FIELD_ERRORS: Record<string, string> = {
  required: 'Обязательное поле',
  invalid: 'Проверьте формат',
  too_long: 'Слишком длинно',
}

const PRIVACY_REQUIRED = 'Без согласия мы не сможем принять заказ'

type FormErrors = Partial<Record<keyof Customer, string>>

export function CheckoutPage() {
  const navigate = useNavigate()
  const { state, items, quote, promoApplied, fallbackSubtotal } = useCartQuote()
  const [form, setForm] = useState<Customer>({
    name: '',
    phone: '',
    delivery: 'pickup',
    address: '',
    comment: '',
    privacyAccepted: false,
    // Must stay unchecked by default: a pre-ticked box is not valid consent.
    marketingConsent: false,
  })
  const [errors, setErrors] = useState<FormErrors>({})

  const placeOrder = useMutation({
    mutationFn: () => api.placeOrder(items, promoApplied ? state.promoCode : '', form),
    // The success page clears the cart; clearing here would re-render this page with an empty cart first.
    onSuccess: (order) => navigate(routes.orderSuccess, { replace: true, state: { order } }),
    onError: (err) => {
      if (err instanceof ApiError && err.code === 'validation') {
        setErrors(
          Object.fromEntries(
            Object.entries(err.fields).map(([k, v]) => [
              k,
              k === 'privacyAccepted'
                ? PRIVACY_REQUIRED
                : (FIELD_ERRORS[v] ?? FIELD_ERRORS.invalid),
            ]),
          ),
        )
      }
    },
  })

  if (state.items.length === 0) return <Navigate to={routes.cart} replace />

  const set = <K extends keyof Customer>(key: K, value: Customer[K]) => {
    setForm((f) => ({ ...f, [key]: value }))
    setErrors((e) => ({ ...e, [key]: undefined }))
  }

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const next: FormErrors = {}
    if (!form.name.trim()) next.name = FIELD_ERRORS.required
    if (!isCompletePhone(form.phone)) next.phone = 'Введите номер полностью'
    if (form.delivery !== 'pickup' && !form.address.trim()) next.address = FIELD_ERRORS.required
    if (!form.privacyAccepted) next.privacyAccepted = PRIVACY_REQUIRED
    setErrors(next)
    if (Object.keys(next).length === 0) placeOrder.mutate()
  }

  const serverError =
    placeOrder.error instanceof ApiError && placeOrder.error.code !== 'validation'
      ? describeError(placeOrder.error)
      : null

  return (
    <div className="container-page pt-8">
      <Link to={routes.cart} className="text-sm text-muted hover:text-ink">
        ← Вернуться в корзину
      </Link>
      <h1 className="mt-3 text-4xl sm:text-5xl">Оформление заказа</h1>

      <form onSubmit={submit} noValidate className="mt-8 grid gap-10 lg:grid-cols-[1fr_360px]">
        <div className="space-y-5">
          <Field label="Имя" error={errors.name} htmlFor="name">
            <input
              id="name"
              className="input"
              autoComplete="name"
              value={form.name}
              onChange={(e) => set('name', e.target.value)}
            />
          </Field>
          <Field label="Телефон (WhatsApp)" error={errors.phone} htmlFor="phone">
            <input
              id="phone"
              type="tel"
              inputMode="tel"
              className="input"
              autoComplete="tel"
              placeholder="+7 (7__) ___-__-__"
              value={form.phone}
              onChange={(e) => set('phone', e.target.value ? formatPhoneInput(e.target.value) : '')}
            />
          </Field>

          <fieldset>
            <legend className="mb-2 text-sm font-medium">Доставка</legend>
            <div className="space-y-2">
              {DELIVERY.map((d) => (
                <label
                  key={d.value}
                  className={`flex cursor-pointer items-start gap-3 rounded-2xl border bg-white p-4 ${
                    form.delivery === d.value ? 'border-ink' : 'border-line'
                  }`}
                >
                  <input
                    type="radio"
                    name="delivery"
                    value={d.value}
                    checked={form.delivery === d.value}
                    onChange={() => set('delivery', d.value)}
                    className="mt-1 accent-ink"
                  />
                  <span>
                    <span className="block text-sm font-medium">{d.label}</span>
                    <span className="block text-xs text-muted">{d.hint}</span>
                  </span>
                </label>
              ))}
            </div>
          </fieldset>

          {form.delivery !== 'pickup' && (
            <Field label="Адрес" error={errors.address} htmlFor="address">
              <input
                id="address"
                className="input"
                autoComplete="street-address"
                placeholder="Город, улица, дом, квартира"
                value={form.address}
                onChange={(e) => set('address', e.target.value)}
              />
            </Field>
          )}

          <Field label="Комментарий" htmlFor="comment">
            <textarea
              id="comment"
              rows={3}
              className="input resize-none"
              value={form.comment}
              onChange={(e) => set('comment', e.target.value)}
            />
          </Field>
        </div>

        <aside className="h-fit space-y-6 rounded-3xl bg-sand p-6">
          <p className="text-sm text-muted">
            {state.items.length} {plural(state.items.length, ['позиция', 'позиции', 'позиций'])} в
            заказе
          </p>
          <OrderSummary quote={quote} fallbackSubtotal={fallbackSubtotal} />
          <div className="space-y-3">
            <Checkbox
              checked={form.privacyAccepted}
              onChange={(v) => set('privacyAccepted', v)}
              error={errors.privacyAccepted}
            >
              Согласен(на) на обработку персональных данных и принимаю{' '}
              <Link to={routes.privacy} target="_blank" className="underline hover:text-ink">
                политику конфиденциальности
              </Link>
            </Checkbox>
            <Checkbox checked={form.marketingConsent} onChange={(v) => set('marketingConsent', v)}>
              Хочу получать новости об акциях и новинках в WhatsApp
            </Checkbox>
          </div>
          {serverError && (
            <p role="alert" className="text-sm text-sale">
              {serverError}
            </p>
          )}
          <button type="submit" className="btn-primary w-full" disabled={placeOrder.isPending}>
            {placeOrder.isPending ? 'Отправляем…' : 'Подтвердить заказ'}
          </button>
          <p className="text-xs leading-relaxed text-muted">
            Оплата через Kaspi: после подтверждения мы напишем вам в WhatsApp и выставим счёт.
          </p>
        </aside>
      </form>
    </div>
  )
}

function describeError(err: ApiError): string {
  switch (err.code) {
    case 'insufficient_stock':
      return 'Часть товаров закончилась. Уменьшите количество в корзине.'
    case 'variant_unavailable':
      return 'Один из товаров больше недоступен. Удалите его из корзины.'
    case 'invalid_promo':
      return 'Промокод больше не действует. Сбросьте его в корзине.'
    case 'network':
      return 'Нет соединения. Проверьте интернет и попробуйте снова.'
    default:
      return 'Не удалось оформить заказ. Попробуйте ещё раз или напишите нам в Instagram.'
  }
}

function Checkbox({
  checked,
  onChange,
  error,
  children,
}: {
  checked: boolean
  onChange: (checked: boolean) => void
  error?: string
  children: ReactNode
}) {
  return (
    <div>
      <label className="flex cursor-pointer items-start gap-3 text-xs leading-relaxed text-muted">
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          aria-invalid={error ? true : undefined}
          className="mt-0.5 h-4 w-4 shrink-0 accent-ink"
        />
        <span>{children}</span>
      </label>
      {error && <p className="mt-1.5 pl-7 text-xs text-sale">{error}</p>}
    </div>
  )
}

function Field({
  label,
  error,
  htmlFor,
  children,
}: {
  label: string
  error?: string
  htmlFor: string
  children: ReactNode
}) {
  return (
    <div>
      <label htmlFor={htmlFor} className="mb-2 block text-sm font-medium">
        {label}
      </label>
      {children}
      {error && <p className="mt-1.5 text-xs text-sale">{error}</p>}
    </div>
  )
}
