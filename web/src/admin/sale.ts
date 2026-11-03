/** Same rounding as the server: percent off, rounded half up to whole tenge; null if not cheaper. */
export function salePrice(price: number, percent: number): number | null {
  const sale = Math.floor((price * (100 - percent) + 50) / 100)
  return sale > 0 && sale < price ? sale : null
}
