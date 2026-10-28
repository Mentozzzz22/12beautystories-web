import { formatTenge } from '../lib/format'

interface Props {
  price: number
  compareAtPrice?: number
  from?: boolean
  className?: string
}

export function Price({ price, compareAtPrice, from, className = '' }: Props) {
  const onSale = compareAtPrice !== undefined && compareAtPrice > price
  return (
    <span className={`inline-flex flex-wrap items-baseline gap-x-2 ${className}`}>
      <span className={onSale ? 'text-sale' : ''}>
        {from && 'от '}
        {formatTenge(price)}
      </span>
      {onSale && (
        <s
          className="text-[0.85em] text-muted"
          aria-label={`Старая цена ${formatTenge(compareAtPrice)}`}
        >
          {formatTenge(compareAtPrice)}
        </s>
      )}
    </span>
  )
}
