import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import type { Product } from '../api/types'
import { cart } from '../lib/cart'
import { ProductView } from './ProductPage'

const product: Product = {
  id: 1,
  slug: 'davines-energizing-shampoo',
  name: 'Energizing Shampoo',
  brand: { slug: 'davines', name: 'Davines' },
  line: { slug: 'energizing', name: 'Energizing' },
  description: 'Энергетический шампунь.',
  ingredients: 'Кофеин',
  optionKind: 'volume',
  media: [{ url: '/a.jpg', kind: 'image' }],
  variants: [
    { id: 10, label: '100 ml', sku: '', price: 9000, inStock: true },
    { id: 11, label: '250 ml', sku: '', price: 18000, inStock: true },
    { id: 12, label: '1000 ml', sku: '', price: 48000, compareAtPrice: 54000, inStock: false },
  ],
}

function renderProduct() {
  return render(
    <MemoryRouter>
      <ProductView product={product} />
    </MemoryRouter>,
  )
}

describe('ProductView', () => {
  beforeEach(() => cart.clear())

  it('shows the price of the selected variant', async () => {
    const user = userEvent.setup()
    renderProduct()

    expect(screen.getByText(/9\s000/)).toBeInTheDocument()
    await user.click(screen.getByLabelText('250 ml'))
    expect(screen.getByText(/18\s000/)).toBeInTheDocument()
  })

  it('shows the sale price and disables out-of-stock variants', async () => {
    const user = userEvent.setup()
    renderProduct()

    await user.click(screen.getByLabelText('1000 ml'))
    expect(screen.getByLabelText(/Старая цена/)).toHaveTextContent(/54\s000/)
    expect(screen.getByRole('button', { name: 'Нет в наличии' })).toBeDisabled()
  })

  it('adds the selected variant to the cart', async () => {
    const user = userEvent.setup()
    renderProduct()

    await user.click(screen.getByLabelText('250 ml'))
    await user.click(screen.getByRole('button', { name: 'Добавить в корзину' }))

    expect(cart.getSnapshot().items).toEqual([
      expect.objectContaining({ variantId: 11, variantLabel: '250 ml', price: 18000, quantity: 1 }),
    ])
    expect(screen.getByRole('status')).toHaveTextContent('Товар добавлен в корзину')
  })
})
