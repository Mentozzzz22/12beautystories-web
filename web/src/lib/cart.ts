import { useSyncExternalStore } from 'react'

/** Snapshot of what the customer saw; prices are re-quoted by the server before checkout. */
export interface CartItem {
  variantId: number
  quantity: number
  productSlug: string
  productName: string
  brandName: string
  variantLabel: string
  price: number
  imageUrl: string
}

export interface CartState {
  items: CartItem[]
  promoCode: string
}

export const MAX_QUANTITY = 99
const STORAGE_KEY = 'bs-cart-v1'
const EMPTY: CartState = { items: [], promoCode: '' }

type Listener = () => void

export class CartStore {
  private state: CartState
  private readonly listeners = new Set<Listener>()
  private readonly storage: Storage | undefined

  constructor(storage?: Storage) {
    this.storage = storage
    this.state = this.load()
  }

  getSnapshot = (): CartState => this.state

  subscribe = (listener: Listener): (() => void) => {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  add(item: Omit<CartItem, 'quantity'>, quantity = 1): void {
    const existing = this.state.items.find((i) => i.variantId === item.variantId)
    const items = existing
      ? this.state.items.map((i) =>
          i.variantId === item.variantId
            ? { ...i, ...item, quantity: clamp(i.quantity + quantity) }
            : i,
        )
      : [...this.state.items, { ...item, quantity: clamp(quantity) }]
    this.set({ ...this.state, items })
  }

  setQuantity(variantId: number, quantity: number): void {
    if (quantity <= 0) {
      this.remove(variantId)
      return
    }
    this.set({
      ...this.state,
      items: this.state.items.map((i) =>
        i.variantId === variantId ? { ...i, quantity: clamp(quantity) } : i,
      ),
    })
  }

  remove(variantId: number): void {
    this.set({ ...this.state, items: this.state.items.filter((i) => i.variantId !== variantId) })
  }

  setPromoCode(promoCode: string): void {
    this.set({ ...this.state, promoCode: promoCode.trim().toUpperCase() })
  }

  clear(): void {
    this.set(EMPTY)
  }

  private set(next: CartState): void {
    this.state = next
    try {
      this.storage?.setItem(STORAGE_KEY, JSON.stringify(next))
    } catch {
      // Storage may be full or blocked (private mode); the cart still works in memory.
    }
    this.listeners.forEach((l) => l())
  }

  private load(): CartState {
    try {
      const raw = this.storage?.getItem(STORAGE_KEY)
      if (!raw) return EMPTY
      const parsed = JSON.parse(raw) as Partial<CartState>
      return {
        items: Array.isArray(parsed.items) ? parsed.items.filter(isCartItem) : [],
        promoCode: typeof parsed.promoCode === 'string' ? parsed.promoCode : '',
      }
    } catch {
      return EMPTY
    }
  }
}

function clamp(quantity: number): number {
  return Math.min(Math.max(1, Math.trunc(quantity)), MAX_QUANTITY)
}

function isCartItem(v: unknown): v is CartItem {
  if (typeof v !== 'object' || v === null) return false
  const i = v as Record<string, unknown>
  return typeof i.variantId === 'number' && typeof i.quantity === 'number' && i.quantity > 0
}

function browserStorage(): Storage | undefined {
  try {
    return window.localStorage
  } catch {
    return undefined
  }
}

export const cart = new CartStore(browserStorage())

export function useCart(): CartState {
  return useSyncExternalStore(cart.subscribe, cart.getSnapshot)
}

export function cartCount(state: CartState): number {
  return state.items.reduce((sum, i) => sum + i.quantity, 0)
}
