import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../api/client'
import type { Order } from '../api/types'
import { cart } from '../lib/cart'
import { CheckoutPage } from './CheckoutPage'

const quote = { subtotal: 18000, discounts: [], discountTotal: 0, total: 18000 }

function renderCheckout() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <CheckoutPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

async function fillContacts(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('Имя'), 'Айгерим')
  await user.type(screen.getByLabelText('Телефон (WhatsApp)'), '87011234567')
}

describe('CheckoutPage consent', () => {
  beforeEach(() => {
    cart.clear()
    cart.add({
      variantId: 11,
      productSlug: 'davines-energizing-shampoo',
      productName: 'Energizing Shampoo',
      brandName: 'Davines',
      variantLabel: '250 ml',
      price: 18000,
      imageUrl: '',
    })
    vi.spyOn(api, 'quote').mockResolvedValue({ lines: [], quote })
  })
  afterEach(() => vi.restoreAllMocks())

  it('does not send the order without the privacy consent', async () => {
    const place = vi.spyOn(api, 'placeOrder')
    const user = userEvent.setup()
    renderCheckout()

    await fillContacts(user)
    await user.click(screen.getByRole('button', { name: 'Подтвердить заказ' }))

    expect(screen.getByText('Без согласия мы не сможем принять заказ')).toBeInTheDocument()
    expect(place).not.toHaveBeenCalled()
  })

  it('sends both consents, with promotions unchecked by default', async () => {
    const place = vi.spyOn(api, 'placeOrder').mockResolvedValue({ id: 1 } as Order)
    const user = userEvent.setup()
    renderCheckout()

    const marketing = screen.getByRole('checkbox', { name: /акциях и новинках/ })
    expect(marketing).not.toBeChecked()

    await fillContacts(user)
    await user.click(screen.getByRole('checkbox', { name: /обработку персональных данных/ }))
    await user.click(screen.getByRole('button', { name: 'Подтвердить заказ' }))

    expect(place).toHaveBeenCalledOnce()
    expect(place.mock.calls[0]?.[2]).toMatchObject({
      phone: '+7 (701) 123-45-67',
      privacyAccepted: true,
      marketingConsent: false,
    })
  })
})
