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

const withVariantPhotos: Product = {
  ...product,
  media: [
    { url: '/a.jpg', kind: 'image' },
    { url: '/b.jpg', kind: 'image' },
    { url: '/c.jpg', kind: 'image' },
  ],
  variants: [
    { id: 10, label: '100 ml', sku: '', price: 9000, inStock: true, imageUrl: '/b.jpg' },
    { id: 11, label: '250 ml', sku: '', price: 18000, inStock: true, imageUrl: '/c.jpg' },
    { id: 12, label: '1000 ml', sku: '', price: 48000, inStock: true },
  ],
}

function renderProduct(p = product) {
  return render(
    <MemoryRouter>
      <ProductView product={p} />
    </MemoryRouter>,
  )
}

const mainPhoto = () => screen.getByRole('img', { name: product.name })

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

  it('opens the photo of the selected variant and keeps the current one without a photo', async () => {
    const user = userEvent.setup()
    renderProduct(withVariantPhotos)

    expect(mainPhoto()).toHaveAttribute('src', '/b.jpg')
    await user.click(screen.getByLabelText('250 ml'))
    expect(mainPhoto()).toHaveAttribute('src', '/c.jpg')
    await user.click(screen.getByLabelText('1000 ml'))
    expect(mainPhoto()).toHaveAttribute('src', '/c.jpg')
  })

  it('still lets the customer browse the other photos', async () => {
    const user = userEvent.setup()
    renderProduct(withVariantPhotos)

    await user.click(screen.getByRole('button', { name: 'Фото 1' }))
    expect(mainPhoto()).toHaveAttribute('src', '/a.jpg')
    expect(screen.getByLabelText('100 ml')).toBeChecked()
  })

  it('puts the photo of the selected variant into the cart', async () => {
    const user = userEvent.setup()
    renderProduct(withVariantPhotos)

    await user.click(screen.getByLabelText('250 ml'))
    await user.click(screen.getByRole('button', { name: 'Добавить в корзину' }))
    expect(cart.getSnapshot().items[0]).toMatchObject({ imageUrl: '/c.jpg' })

    await user.click(screen.getByLabelText('1000 ml'))
    await user.click(screen.getByRole('button', { name: 'Добавить в корзину' }))
    expect(cart.getSnapshot().items[1]).toMatchObject({ imageUrl: '/a.jpg' })
  })
})
