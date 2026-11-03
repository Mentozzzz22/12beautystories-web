/** Same rounding as the server: percent off, to the nearest 10 tenge; null if not cheaper. */
export function salePrice(price: number, percent: number): number | null {
  const sale = Math.floor((price * (100 - percent) + 500) / 1000) * 10
  return sale > 0 && sale < price ? sale : null
}
