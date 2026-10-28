import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { api } from './client'
import type { CartItemInput, ProductFilter } from './types'

const STATIC_STALE_TIME = 5 * 60_000

export function useMenu() {
  return useQuery({ queryKey: ['menu'], queryFn: api.menu, staleTime: STATIC_STALE_TIME })
}

export function useBanners() {
  return useQuery({ queryKey: ['banners'], queryFn: api.banners, staleTime: STATIC_STALE_TIME })
}

export function useProducts(filter: ProductFilter) {
  return useQuery({
    queryKey: ['products', filter],
    queryFn: () => api.products(filter),
    placeholderData: keepPreviousData,
  })
}

export function useProduct(slug: string) {
  return useQuery({ queryKey: ['product', slug], queryFn: () => api.product(slug) })
}

export function useQuote(items: CartItemInput[], promoCode: string) {
  return useQuery({
    queryKey: ['quote', items, promoCode],
    queryFn: () => api.quote(items, promoCode),
    enabled: items.length > 0,
    placeholderData: keepPreviousData,
    retry: false,
  })
}
