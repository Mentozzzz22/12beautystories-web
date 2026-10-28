import type {
  Banner,
  CartItemInput,
  Customer,
  Menu,
  Order,
  Product,
  ProductFilter,
  ProductPage,
  QuoteResult,
} from './types'

export type ApiErrorCode =
  | 'bad_request'
  | 'not_found'
  | 'validation'
  | 'empty_cart'
  | 'variant_unavailable'
  | 'insufficient_stock'
  | 'invalid_promo'
  | 'internal'
  | 'network'

export class ApiError extends Error {
  readonly status: number
  readonly code: ApiErrorCode
  readonly fields: Record<string, string>

  constructor(
    status: number,
    code: ApiErrorCode,
    message: string,
    fields: Record<string, string> = {},
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

interface ErrorBody {
  code?: ApiErrorCode
  message?: string
  fields?: Record<string, string>
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response
  try {
    res = await fetch(path, {
      ...init,
      headers: { 'Content-Type': 'application/json', ...init?.headers },
    })
  } catch {
    throw new ApiError(0, 'network', 'Нет соединения с сервером')
  }

  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as ErrorBody
    throw new ApiError(
      res.status,
      body.code ?? 'internal',
      body.message ?? res.statusText,
      body.fields,
    )
  }
  return (await res.json()) as T
}

function toQuery(filter: ProductFilter): string {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(filter)) {
    if (value !== undefined && value !== '' && value !== null) params.set(key, String(value))
  }
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

export const api = {
  menu: () => request<Menu>('/api/menu'),
  banners: () => request<Banner[]>('/api/banners'),
  products: (filter: ProductFilter) => request<ProductPage>(`/api/products${toQuery(filter)}`),
  product: (slug: string) => request<Product>(`/api/products/${encodeURIComponent(slug)}`),
  quote: (items: CartItemInput[], promoCode: string) =>
    request<QuoteResult>('/api/cart/quote', {
      method: 'POST',
      body: JSON.stringify({ items, promoCode }),
    }),
  placeOrder: (items: CartItemInput[], promoCode: string, customer: Customer) =>
    request<Order>('/api/orders', {
      method: 'POST',
      body: JSON.stringify({ items, promoCode, customer }),
    }),
}
