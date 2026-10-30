import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Brand, Category, Product } from '../types'
import { ProductForm } from './ProductEditPage'

const brands: Brand[] = [
  {
    id: 1,
    slug: 'davines',
    name: 'Davines',
    description: '',
    logoUrl: '',
    sort: 0,
    isActive: true,
    lines: [
      {
        id: 7,
        brandId: 1,
        slug: 'energizing',
        name: 'Energizing',
        description: '',
        imageUrl: '',
        sort: 0,
        isActive: true,
      },
    ],
  },
  {
    id: 2,
    slug: 'payot',
    name: 'Payot',
    description: '',
    logoUrl: '',
    sort: 0,
    isActive: true,
    lines: [],
  },
]

const categories: Category[] = [
  { id: 3, slug: 'leave-in', name: 'Несмываемые', sort: 0, isActive: true },
  { id: 4, slug: 'oils', name: 'Масла', sort: 1, isActive: true },
]

const product: Product = {
  id: 0,
  slug: '',
  name: '',
  brandId: 0,
  lineId: null,
  categoryIds: [],
  description: '',
  ingredients: '',
  optionKind: 'none',
  sort: 0,
  isActive: true,
  media: [],
  variants: [
    {
      id: 0,
      label: '',
      sku: '',
      price: 0,
      compareAtPrice: null,
      stock: 0,
      isActive: true,
      imageUrl: '',
    },
  ],
}

const withPhotos: Product = {
  ...product,
  name: 'OI Oil',
  brandId: 1,
  optionKind: 'volume',
  media: [
    { url: '/a.jpg', kind: 'image' },
    { url: '/b.jpg', kind: 'image' },
  ],
}

let saved: unknown

beforeEach(() => {
  saved = undefined
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string, init?: RequestInit) => {
      if (url === '/api/admin/brands') return Response.json(brands)
      if (url === '/api/admin/categories') return Response.json(categories)
      if (url === '/api/admin/products' && init?.method === 'POST') {
        saved = JSON.parse(String(init.body))
        return Response.json({ ...(saved as Product), id: 42 }, { status: 201 })
      }
      return new Response(null, { status: 404 })
    }),
  )
})

afterEach(() => vi.unstubAllGlobals())

function renderForm(initial = product) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <ProductForm initial={initial} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('ProductForm', () => {
  it('offers only lines of the selected brand and resets the line on brand change', async () => {
    const user = userEvent.setup()
    renderForm()

    const brand = await screen.findByLabelText('Бренд')
    await screen.findByRole('option', { name: 'Davines' })
    await user.selectOptions(brand, '1')
    const line = screen.getByLabelText('Линия')
    expect(within(line).getByRole('option', { name: 'Energizing' })).toBeInTheDocument()
    await user.selectOptions(line, '7')

    await user.selectOptions(brand, '2')
    expect(within(line).queryByRole('option', { name: 'Energizing' })).not.toBeInTheDocument()
    expect(line).toHaveValue('')
  })

  it('manages volume variants and sends them on save', async () => {
    const user = userEvent.setup()
    renderForm()
    await screen.findByRole('option', { name: 'Davines' })

    await user.type(screen.getByLabelText('Название'), 'OI Oil')
    await user.selectOptions(screen.getByLabelText('Бренд'), '1')
    await user.click(screen.getByText('Разные объёмы'))
    await user.click(screen.getByRole('button', { name: '+ Добавить объём' }))

    const volumes = screen.getAllByLabelText('Объём')
    const prices = screen.getAllByLabelText('Цена')
    expect(volumes).toHaveLength(2)
    await user.type(volumes[0]!, '50 мл')
    await user.type(prices[0]!, '16000')
    await user.type(volumes[1]!, '135 мл')
    await user.type(prices[1]!, '32000')

    await user.click(screen.getByRole('button', { name: 'Сохранить' }))
    await screen.findByText('Сохранено')

    expect(saved).toMatchObject({
      name: 'OI Oil',
      brandId: 1,
      optionKind: 'volume',
      variants: [
        { label: '50 мл', price: 16000 },
        { label: '135 мл', price: 32000 },
      ],
    })
  })

  it('keeps a single variant when switching back to one option', async () => {
    const user = userEvent.setup()
    renderForm()
    await screen.findByRole('option', { name: 'Davines' })

    await user.click(screen.getByText('Разные цвета'))
    await user.click(screen.getByRole('button', { name: '+ Добавить цвет' }))
    expect(screen.getAllByLabelText('Цвет')).toHaveLength(2)

    await user.click(screen.getByText('Один вариант'))
    expect(screen.queryByLabelText('Цвет')).not.toBeInTheDocument()
    expect(screen.getAllByLabelText('Цена')).toHaveLength(1)
  })

  const photo = (n: number) => screen.getByRole('button', { name: `Фото этого объёма: фото ${n}` })

  it('saves the photo chosen for a variant and clears it on a second click', async () => {
    const user = userEvent.setup()
    renderForm(withPhotos)
    await screen.findByRole('option', { name: 'Davines' })

    await user.click(photo(1))
    await user.click(photo(1))
    expect(photo(1)).toHaveAttribute('aria-pressed', 'false')
    await user.click(photo(2))
    expect(photo(2)).toHaveAttribute('aria-pressed', 'true')

    await user.click(screen.getByRole('button', { name: 'Сохранить' }))
    await screen.findByText('Сохранено')
    expect(saved).toMatchObject({ variants: [{ imageUrl: '/b.jpg' }] })
  })

  it('clears the variant photo when it is removed from the gallery', async () => {
    const user = userEvent.setup()
    renderForm(withPhotos)
    await screen.findByRole('option', { name: 'Davines' })

    await user.click(photo(2))
    await user.click(screen.getAllByRole('button', { name: 'Удалить фото' })[1]!)
    await user.click(screen.getByRole('button', { name: 'Сохранить' }))
    await screen.findByText('Сохранено')

    expect(saved).toMatchObject({ variants: [{ imageUrl: '' }] })
  })

  it('saves several categories and removes one on a second click', async () => {
    const user = userEvent.setup()
    renderForm(withPhotos)

    await user.click(await screen.findByRole('checkbox', { name: 'Несмываемые' }))
    await user.click(screen.getByRole('checkbox', { name: 'Масла' }))
    await user.click(screen.getByRole('button', { name: 'Сохранить' }))
    await screen.findByText('Сохранено')
    expect(saved).toMatchObject({ categoryIds: [3, 4] })

    await user.click(screen.getByRole('checkbox', { name: 'Несмываемые' }))
    expect(screen.getByRole('checkbox', { name: 'Несмываемые' })).not.toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Масла' })).toBeChecked()
  })

  it('offers no variant photos for a single-option product', async () => {
    renderForm({ ...withPhotos, optionKind: 'none' })
    await screen.findByRole('option', { name: 'Davines' })

    expect(screen.queryByText('Фото этого объёма')).not.toBeInTheDocument()
  })
})
