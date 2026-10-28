export const site = {
  name: '12 Beauty Stories',
  city: 'Астана',
  instagramUrl: 'https://www.instagram.com/12_beautystories/',
  // TODO: replace with the shop's real WhatsApp number before launch.
  whatsappUrl: 'https://wa.me/',
} as const

export const routes = {
  home: '/',
  catalog: '/catalog',
  brand: (brand: string) => `/brands/${brand}`,
  line: (brand: string, line: string) => `/brands/${brand}/${line}`,
  category: (category: string) => `/categories/${category}`,
  product: (slug: string) => `/products/${slug}`,
  cart: '/cart',
  checkout: '/checkout',
  orderSuccess: '/order/success',
} as const
