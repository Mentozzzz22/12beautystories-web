export const site = {
  name: '12 Beauty Stories',
  city: 'Астана',
  instagramUrl: 'https://www.instagram.com/12_beautystories/',
  // Shop's WhatsApp Business number, 8 705 518 18 29.
  whatsappPhone: '+77055181829',
  whatsappDisplay: '+7 705 518 18 29',
  // Personal data operator shown in the privacy policy.
  operatorName: 'ИП 12-BEAUTYSTORIES',
  operatorBin: '750822450176',
  operatorAddress: 'пр. Кабанбай батыра, 11',
} as const

/** wa.me link; text pre-fills the first message. */
export function whatsappLink(phone: string = site.whatsappPhone, text?: string): string {
  const digits = phone.replace(/\D/g, '')
  const q = text ? `?text=${encodeURIComponent(text)}` : ''
  return `https://wa.me/${digits}${q}`
}

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
  privacy: '/privacy',
} as const
