/** Formats whole tenge as "12 500 ₸" (non-breaking spaces keep the amount on one line). */
export function formatTenge(amount: number): string {
  const sign = amount < 0 ? '−' : ''
  const digits = Math.abs(Math.trunc(amount))
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, ' ')
  return `${sign}${digits} ₸`
}

/** Russian plural form: plural(5, ['товар', 'товара', 'товаров']) → "товаров". */
export function plural(n: number, forms: readonly [string, string, string]): string {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return forms[0]
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return forms[1]
  return forms[2]
}

/** Live formatting for Kazakhstan phone input: "+7 (701) 123-45-67". */
export function formatPhoneInput(raw: string): string {
  let d = raw.replace(/\D/g, '')
  // A pasted 10-digit local number ("7011234567") has no country code yet.
  if (d.length === 10 || !/^[78]/.test(d)) d = '7' + d
  else if (d.startsWith('8')) d = '7' + d.slice(1)
  d = d.slice(0, 11)

  const rest = d.slice(1)
  let out = '+7'
  if (rest.length > 0) out += ` (${rest.slice(0, 3)}`
  if (rest.length >= 3) out += ')'
  if (rest.length > 3) out += ` ${rest.slice(3, 6)}`
  if (rest.length > 6) out += `-${rest.slice(6, 8)}`
  if (rest.length > 8) out += `-${rest.slice(8, 10)}`
  return out
}

export function isCompletePhone(formatted: string): boolean {
  return formatted.replace(/\D/g, '').length === 11
}
