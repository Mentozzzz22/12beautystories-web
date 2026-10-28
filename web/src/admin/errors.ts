import { ApiError } from '../api/client'

export const FIELD_MESSAGES: Record<string, string> = {
  required: 'Обязательное поле',
  invalid: 'Неверное значение',
  too_long: 'Слишком длинно',
  taken: 'Уже занято, выберите другое',
}

/** Maps a validation ApiError to Russian messages per field. */
export function fieldErrors(err: unknown): Record<string, string> {
  if (!(err instanceof ApiError) || err.code !== 'validation') return {}
  return Object.fromEntries(
    Object.entries(err.fields).map(([k, v]) => [k, FIELD_MESSAGES[v] ?? 'Неверное значение']),
  )
}

/** A human-readable message for non-field errors. */
export function errorMessage(err: unknown): string | null {
  if (!err) return null
  if (err instanceof ApiError) {
    switch (err.code) {
      case 'validation':
        return 'Проверьте выделенные поля'
      case 'conflict':
        return 'Нельзя удалить: к этому привязаны товары или данные изменились'
      case 'network':
        return 'Нет соединения с сервером'
      case 'unauthenticated':
        return 'Сессия истекла, войдите снова'
      default:
        return 'Что-то пошло не так. Попробуйте ещё раз'
    }
  }
  return 'Что-то пошло не так. Попробуйте ещё раз'
}
