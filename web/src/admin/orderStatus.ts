import type { OrderListItem, OrderStatus } from './types'

export const ORDER_STATUSES: {
  value: OrderStatus
  label: string
  tone: 'blue' | 'amber' | 'green' | 'gray' | 'red'
}[] = [
  { value: 'new', label: 'Новый', tone: 'blue' },
  { value: 'awaiting_payment', label: 'Ждёт оплаты', tone: 'amber' },
  { value: 'paid', label: 'Оплачен', tone: 'green' },
  { value: 'shipped', label: 'Отправлен', tone: 'green' },
  { value: 'completed', label: 'Выполнен', tone: 'gray' },
  { value: 'cancelled', label: 'Отменён', tone: 'red' },
]

export function statusInfo(status: OrderStatus) {
  return ORDER_STATUSES.find((s) => s.value === status) ?? ORDER_STATUSES[0]!
}

export const DELIVERY_LABELS: Record<OrderListItem['delivery'], string> = {
  pickup: 'Самовывоз',
  courier: 'Курьер',
  post: 'Почта',
}

export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso))
}
