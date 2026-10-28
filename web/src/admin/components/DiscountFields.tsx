import type { DiscountKind } from '../types'
import { NumberField } from './ui'

interface Props {
  kind: DiscountKind
  value: number
  minSubtotal: number
  onChange: (patch: { kind?: DiscountKind; value?: number; minSubtotal?: number }) => void
  errors: Record<string, string>
  minSubtotalLabel?: string
}

/** Shared "how much off, from what amount" inputs for promo codes and automatic rules. */
export function DiscountFields({
  kind,
  value,
  minSubtotal,
  onChange,
  errors,
  minSubtotalLabel,
}: Props) {
  return (
    <>
      <div className="grid grid-cols-[auto_1fr] items-end gap-3">
        <div
          className="flex rounded-xl border border-line bg-white p-1"
          role="radiogroup"
          aria-label="Тип скидки"
        >
          {(['percent', 'fixed'] as const).map((k) => (
            <button
              key={k}
              type="button"
              role="radio"
              aria-checked={kind === k}
              onClick={() => onChange({ kind: k })}
              className={`rounded-lg px-4 py-2 text-sm ${kind === k ? 'bg-ink text-cream' : ''}`}
            >
              {k === 'percent' ? '%' : '₸'}
            </button>
          ))}
        </div>
        <NumberField
          label="Размер скидки"
          value={value}
          suffix={kind === 'percent' ? '%' : '₸'}
          onChange={(v) => onChange({ value: v ?? 0 })}
          error={errors.value && (kind === 'percent' ? 'От 1 до 100' : 'Больше нуля')}
        />
      </div>
      <NumberField
        label={minSubtotalLabel ?? 'Минимальная сумма заказа'}
        value={minSubtotal || null}
        suffix="₸"
        placeholder="Без минимума"
        onChange={(v) => onChange({ minSubtotal: v ?? 0 })}
        error={errors.minSubtotal}
      />
    </>
  )
}
