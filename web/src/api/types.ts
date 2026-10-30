// Mirrors the JSON contracts of the Go API (backend/internal/catalog, order, pricing).

export interface Line {
  id: number
  slug: string
  name: string
  description: string
  imageUrl: string
}

export interface Brand {
  id: number
  slug: string
  name: string
  description: string
  logoUrl: string
  lines: Line[]
}

export interface Category {
  id: number
  slug: string
  name: string
}

export interface Menu {
  brands: Brand[]
  categories: Category[]
}

export interface Banner {
  id: number
  title: string
  subtitle: string
  imageUrl: string
  mobileImageUrl: string
  linkUrl: string
}

export interface Ref {
  slug: string
  name: string
}

export interface ProductSummary {
  id: number
  slug: string
  name: string
  brand: Ref
  line?: Ref
  imageUrl: string
  priceFrom: number
  compareAtPrice?: number
  hasOptions: boolean
  inStock: boolean
}

export interface ProductPage {
  items: ProductSummary[]
  total: number
  page: number
  pageSize: number
}

export type OptionKind = 'none' | 'volume' | 'color'

export interface Media {
  url: string
  kind: 'image' | 'video'
}

export interface Variant {
  id: number
  label: string
  sku: string
  price: number
  compareAtPrice?: number
  inStock: boolean
  imageUrl?: string
}

export interface Product {
  id: number
  slug: string
  name: string
  brand: Ref
  line?: Ref
  category?: Ref
  description: string
  ingredients: string
  optionKind: OptionKind
  media: Media[]
  variants: Variant[]
}

export interface ProductFilter {
  brand?: string
  line?: string
  category?: string
  q?: string
  page?: number
  pageSize?: number
}

export interface CartItemInput {
  variantId: number
  quantity: number
}

export interface Discount {
  source: 'rule' | 'promo'
  label: string
  amount: number
}

export type PromoStatus = 'applied' | 'min_subtotal_not_met' | 'not_better'

export interface Quote {
  subtotal: number
  discounts: Discount[]
  discountTotal: number
  total: number
  promoStatus?: PromoStatus
}

export interface OrderLine {
  variantId: number
  productName: string
  variantLabel: string
  unitPrice: number
  quantity: number
  lineTotal: number
}

export interface QuoteResult {
  lines: OrderLine[]
  quote: Quote
}

export type DeliveryMethod = 'pickup' | 'courier' | 'post'

export interface Customer {
  name: string
  phone: string
  delivery: DeliveryMethod
  address: string
  comment: string
  privacyAccepted: boolean
  marketingConsent: boolean
}

export interface Order {
  id: number
  number: string
  customer: Customer
  lines: OrderLine[]
  promoCode?: string
  quote: Quote
  createdAt: string
}
