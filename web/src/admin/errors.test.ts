import { describe, expect, it } from 'vitest'
import { ApiError } from '../api/client'
import { errorMessage, fieldErrors } from './errors'

describe('fieldErrors', () => {
  it('maps API codes to Russian messages and keeps nested keys', () => {
    const err = new ApiError(422, 'validation', 'invalid', {
      name: 'required',
      'variants.1.price': 'invalid',
      slug: 'taken',
      other: 'something_new',
    })
    expect(fieldErrors(err)).toEqual({
      name: 'Обязательное поле',
      'variants.1.price': 'Неверное значение',
      slug: 'Уже занято, выберите другое',
      other: 'Неверное значение',
    })
  })

  it('ignores non-validation errors', () => {
    expect(fieldErrors(new ApiError(409, 'conflict', 'x'))).toEqual({})
    expect(fieldErrors(null)).toEqual({})
  })
})

describe('errorMessage', () => {
  it('explains conflicts in plain words', () => {
    expect(errorMessage(new ApiError(409, 'conflict', 'x'))).toMatch(/Нельзя удалить/)
    expect(errorMessage(null)).toBeNull()
  })
})
