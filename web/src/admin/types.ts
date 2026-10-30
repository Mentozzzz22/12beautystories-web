// Mirrors backend/internal/admin JSON contracts.

export interface AdminUser {
  id: number
  email: string
}

export interface Line {
  id: number
  brandId: number
  slug: string
  name: string
  description: string
  imageUrl: string
  sort: number
  isActive: boolean
}

export interface Brand {
  id: number
  slug: string
  name: string
  description: string
  logoUrl: string
  sort: number
  isActive: boolean
  lines: Line[]
}

export interface Category {
  id: number
  slug: string
  name: string
  sort: number
  isActive: boolean
}

export interface ProductListItem {
  id: number
  slug: string
  name: string
  brandName: string
  lineName: string
  categoryName: string
  imageUrl: string
  priceFrom: number | null
  totalStock: number
  variantCount: number
  isActive: boolean
}

export interface Page<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

export interface Media {
  url: string
  kind: 'image' | 'video'
}

export interface Variant {
  id: number
  label: string
  sku: string
  price: number
  compareAtPrice: number | null
  stock: number
  isActive: boolean
  imageUrl: string
}

export type OptionKind = 'none' | 'volume' | 'color'

export interface Product {
  id: number
  slug: string
  name: string
  brandId: number
  lineId: number | null
  categoryId: number | null
  description: string
  ingredients: string
  optionKind: OptionKind
  sort: number
  isActive: boolean
  media: Media[]
  variants: Variant[]
}

export interface Banner {
  id: number
  title: string
  subtitle: string
  imageUrl: string
  mobileImageUrl: string
  linkUrl: string
  sort: number
  isActive: boolean
  startsAt: string | null
  endsAt: string | null
}

export type DiscountKind = 'percent' | 'fixed'

export interface Promo {
  id: number
  code: string
  kind: DiscountKind
  value: number
  minSubtotal: number
  combinableWithRules: boolean
  maxUses: number | null
  usedCount: number
  isActive: boolean
  startsAt: string | null
  endsAt: string | null
}

export interface Rule {
  id: number
  name: string
  brandId: number | null
  brandName: string
  minSubtotal: number
  kind: DiscountKind
  value: number
  isActive: boolean
  startsAt: string | null
  endsAt: string | null
}

export type OrderStatus =
  'new' | 'awaiting_payment' | 'paid' | 'shipped' | 'completed' | 'cancelled'

export interface OrderListItem {
  id: number
  number: string
  status: OrderStatus
  customerName: string
  phone: string
  delivery: 'pickup' | 'courier' | 'post'
  total: number
  itemCount: number
  createdAt: string
}

export interface OrderItem {
  variantId: number | null
  productName: string
  variantLabel: string
  unitPrice: number
  quantity: number
  lineTotal: number
}

export interface Order extends OrderListItem {
  customerId: number
  address: string
  comment: string
  promoCode: string
  subtotal: number
  discountTotal: number
  items: OrderItem[]
}

export interface OrderPage extends Page<OrderListItem> {
  newCount: number
}

export interface CustomerListItem {
  id: number
  name: string
  phone: string
  marketingConsent: boolean
  /** Statistics ignore cancelled orders. */
  orderCount: number
  totalSpent: number
  lastOrderAt: string | null
  createdAt: string
}

export interface Customer extends CustomerListItem {
  note: string
  marketingConsentAt: string | null
  orders: OrderListItem[]
}

export interface CustomerUpdate {
  name: string
  note: string
  marketingConsent: boolean
}

export type CustomerSort = 'recent' | 'spent' | 'orders' | 'inactive'

export interface CustomerFilter {
  q: string
  marketing: boolean
  sort: CustomerSort
  page: number
}
