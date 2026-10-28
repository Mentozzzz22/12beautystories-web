import { beforeEach, describe, expect, it } from 'vitest'
import { type CartItem, cartCount, CartStore, MAX_QUANTITY } from './cart'

const shampoo: Omit<CartItem, 'quantity'> = {
  variantId: 1,
  productSlug: 'davines-energizing-shampoo',
  productName: 'Energizing Shampoo',
  brandName: 'Davines',
  variantLabel: '250 ml',
  price: 18000,
  imageUrl: '',
}

describe('CartStore', () => {
  let storage: Storage
  let store: CartStore

  beforeEach(() => {
    localStorage.clear()
    storage = localStorage
    store = new CartStore(storage)
  })

  it('merges the same variant and clamps quantity', () => {
    store.add(shampoo)
    store.add(shampoo, 2)
    expect(store.getSnapshot().items).toHaveLength(1)
    expect(store.getSnapshot().items[0]?.quantity).toBe(3)

    store.add(shampoo, 500)
    expect(store.getSnapshot().items[0]?.quantity).toBe(MAX_QUANTITY)
  })

  it('removes an item when quantity drops to zero', () => {
    store.add(shampoo)
    store.setQuantity(shampoo.variantId, 0)
    expect(store.getSnapshot().items).toHaveLength(0)
  })

  it('normalizes promo codes', () => {
    store.setPromoCode('  welcome5 ')
    expect(store.getSnapshot().promoCode).toBe('WELCOME5')
  })

  it('persists and restores state', () => {
    store.add(shampoo, 2)
    store.add({ ...shampoo, variantId: 2, variantLabel: '1000 ml' })
    const restored = new CartStore(storage)
    expect(cartCount(restored.getSnapshot())).toBe(3)
  })

  it('ignores corrupted storage', () => {
    storage.setItem('bs-cart-v1', '{not json')
    expect(new CartStore(storage).getSnapshot().items).toEqual([])

    storage.setItem(
      'bs-cart-v1',
      JSON.stringify({ items: [{ variantId: 'x' }, { ...shampoo, quantity: 1 }] }),
    )
    expect(new CartStore(storage).getSnapshot().items).toHaveLength(1)
  })

  it('notifies subscribers and stops after unsubscribe', () => {
    let calls = 0
    const unsubscribe = store.subscribe(() => calls++)
    store.add(shampoo)
    unsubscribe()
    store.clear()
    expect(calls).toBe(1)
  })
})
