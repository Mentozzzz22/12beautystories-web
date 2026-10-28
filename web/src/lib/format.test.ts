import { describe, expect, it } from 'vitest'
import { formatPhoneInput, formatTenge, isCompletePhone, plural } from './format'

const nbsp = ' '

describe('formatTenge', () => {
  it.each([
    [0, `0${nbsp}₸`],
    [999, `999${nbsp}₸`],
    [12500, `12${nbsp}500${nbsp}₸`],
    [1234567, `1${nbsp}234${nbsp}567${nbsp}₸`],
    [-5000, `−5${nbsp}000${nbsp}₸`],
  ])('formats %d', (amount, expected) => {
    expect(formatTenge(amount)).toBe(expected)
  })
})

describe('plural', () => {
  const forms = ['товар', 'товара', 'товаров'] as const
  it.each([
    [1, 'товар'],
    [2, 'товара'],
    [5, 'товаров'],
    [11, 'товаров'],
    [21, 'товар'],
    [22, 'товара'],
    [112, 'товаров'],
  ])('%d → %s', (n, expected) => {
    expect(plural(n, forms)).toBe(expected)
  })
})

describe('formatPhoneInput', () => {
  it.each([
    ['7', '+7'],
    ['0', '+7 (0'],
    ['8701', '+7 (701)'],
    ['87011234567', '+7 (701) 123-45-67'],
    ['+7 701 123 45 67 99', '+7 (701) 123-45-67'],
    ['7011234567', '+7 (701) 123-45-67'],
  ])('%s → %s', (raw, expected) => {
    expect(formatPhoneInput(raw)).toBe(expected)
  })

  it('detects a complete number', () => {
    expect(isCompletePhone('+7 (701) 123-45-67')).toBe(true)
    expect(isCompletePhone('+7 (701) 123-45')).toBe(false)
  })
})
