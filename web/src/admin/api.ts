import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, request } from '../api/client'
import type {
  AdminUser,
  Banner,
  Brand,
  Category,
  Line,
  Order,
  OrderPage,
  OrderStatus,
  Page,
  Product,
  ProductListItem,
  Promo,
  Rule,
} from './types'

const BASE = '/api/admin'

const json = (method: string, body: unknown): RequestInit => ({
  method,
  body: JSON.stringify(body),
})

function qs(params: Record<string, string | number | undefined>): string {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params))
    if (v !== undefined && v !== '' && v !== 0) p.set(k, String(v))
  const s = p.toString()
  return s ? `?${s}` : ''
}

export const adminApi = {
  me: () => request<AdminUser>(`${BASE}/me`),
  login: (email: string, password: string) =>
    request<AdminUser>(`${BASE}/login`, json('POST', { email, password })),
  logout: () => request<void>(`${BASE}/logout`, { method: 'POST' }),

  products: (q: string, brandId: number, page: number) =>
    request<Page<ProductListItem>>(`${BASE}/products${qs({ q, brandId, page })}`),
  product: (id: number) => request<Product>(`${BASE}/products/${id}`),
  saveProduct: (p: Product) =>
    p.id
      ? request<Product>(`${BASE}/products/${p.id}`, json('PUT', p))
      : request<Product>(`${BASE}/products`, json('POST', p)),
  deleteProduct: (id: number) => request<void>(`${BASE}/products/${id}`, { method: 'DELETE' }),

  orders: (status: string, q: string, page: number) =>
    request<OrderPage>(`${BASE}/orders${qs({ status, q, page })}`),
  order: (id: number) => request<Order>(`${BASE}/orders/${id}`),
  setOrderStatus: (id: number, status: OrderStatus) =>
    request<Order>(`${BASE}/orders/${id}/status`, json('PUT', { status })),

  upload: (file: File) => {
    const body = new FormData()
    body.append('file', file)
    return request<{ url: string }>(`${BASE}/uploads`, { method: 'POST', body })
  },
}

export function useMe() {
  return useQuery({
    queryKey: ['admin', 'me'],
    queryFn: adminApi.me,
    retry: false,
    staleTime: 5 * 60_000,
  })
}

export function isUnauthenticated(err: unknown): boolean {
  return err instanceof ApiError && err.status === 401
}

/** Generic list + save + delete for simple admin resources sharing one REST shape. */
export function useResource<T extends { id: number }>(resource: string, listable = true) {
  const qc = useQueryClient()
  const list = useQuery({
    queryKey: ['admin', resource],
    queryFn: () => request<T[]>(`${BASE}/${resource}`),
    enabled: listable,
  })
  // Storefront caches (menu, banners, products) depend on admin data too.
  const invalidate = () => qc.invalidateQueries()

  const save = useMutation({
    mutationFn: (item: T) =>
      item.id
        ? request<T>(`${BASE}/${resource}/${item.id}`, json('PUT', item))
        : request<T>(`${BASE}/${resource}`, json('POST', item)),
    onSuccess: invalidate,
  })
  const remove = useMutation({
    mutationFn: (id: number) => request<void>(`${BASE}/${resource}/${id}`, { method: 'DELETE' }),
    onSuccess: invalidate,
  })
  return { list, save, remove }
}

export const useBrands = () => useResource<Brand>('brands')
// Lines are listed inside brands; this hook only saves and deletes.
export const useLines = () => useResource<Line>('lines', false)
export const useCategories = () => useResource<Category>('categories')
export const useBanners = () => useResource<Banner>('banners')
export const usePromos = () => useResource<Promo>('promos')
export const useRules = () => useResource<Rule>('rules')
