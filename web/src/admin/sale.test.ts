import { describe, expect, it } from 'vitest'
import { salePrice } from './sale'

describe('salePrice', () => {
  it.each([
    [12700, 30, 8890],
    [73100, 25, 54830],
    [49130, 15, 41760],
    [100, 1, null],
    [0, 10, null],
  ])('%i minus %i%% is %s', (price, percent, want) => {
    expect(salePrice(price, percent)).toBe(want)
  })
})
