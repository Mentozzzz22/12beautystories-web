import { type ReactNode, useEffect, useId, useState } from 'react'
import { errorMessage } from '../errors'

export function PageHeader({ title, action }: { title: string; action?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
      <h1 className="text-3xl sm:text-4xl">{title}</h1>
      {action}
    </div>
  )
}

interface FieldProps {
  label: string
  error?: string
  hint?: string
  children: (id: string) => ReactNode
}

export function Field({ label, error, hint, children }: FieldProps) {
  const id = useId()
  return (
    <div>
      <label htmlFor={id} className="mb-1.5 block text-sm font-medium">
        {label}
      </label>
      {children(id)}
      {hint && !error && <p className="mt-1 text-xs text-muted">{hint}</p>}
      {error && <p className="mt-1 text-xs text-sale">{error}</p>}
    </div>
  )
}

type InputProps = Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange' | 'value'>

export function TextField({
  label,
  value,
  onChange,
  error,
  hint,
  ...rest
}: {
  label: string
  value: string
  onChange: (v: string) => void
  error?: string
  hint?: string
} & InputProps) {
  return (
    <Field label={label} error={error} hint={hint}>
      {(id) => (
        <input
          id={id}
          className={`input ${error ? 'border-sale' : ''}`}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          {...rest}
        />
      )}
    </Field>
  )
}

export function TextArea({
  label,
  value,
  onChange,
  error,
  hint,
  rows = 4,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  error?: string
  hint?: string
  rows?: number
}) {
  return (
    <Field label={label} error={error} hint={hint}>
      {(id) => (
        <textarea
          id={id}
          rows={rows}
          className={`input resize-y ${error ? 'border-sale' : ''}`}
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
      )}
    </Field>
  )
}

/** Integer input that keeps an empty field as null instead of 0. */
export function NumberField({
  label,
  value,
  onChange,
  error,
  hint,
  suffix,
  placeholder,
}: {
  label: string
  value: number | null
  onChange: (v: number | null) => void
  error?: string
  hint?: string
  suffix?: string
  placeholder?: string
}) {
  return (
    <Field label={label} error={error} hint={hint}>
      {(id) => (
        <div className="relative">
          <input
            id={id}
            type="number"
            inputMode="numeric"
            min={0}
            step={1}
            placeholder={placeholder}
            className={`input ${suffix ? 'pr-10' : ''} ${error ? 'border-sale' : ''}`}
            value={value ?? ''}
            onChange={(e) => {
              const raw = e.target.value
              onChange(raw === '' ? null : Math.trunc(Number(raw)))
            }}
          />
          {suffix && (
            <span className="pointer-events-none absolute inset-y-0 right-4 flex items-center text-sm text-muted">
              {suffix}
            </span>
          )}
        </div>
      )}
    </Field>
  )
}

export function SelectField<T extends string | number>({
  label,
  value,
  onChange,
  options,
  empty,
  error,
}: {
  label: string
  value: T | null
  onChange: (v: T | null) => void
  options: { value: T; label: string }[]
  empty?: string
  error?: string
}) {
  return (
    <Field label={label} error={error}>
      {(id) => (
        <select
          id={id}
          className={`input ${error ? 'border-sale' : ''}`}
          value={value === null ? '' : String(value)}
          onChange={(e) => {
            const raw = e.target.value
            if (raw === '') return onChange(null)
            const match = options.find((o) => String(o.value) === raw)
            onChange(match ? match.value : null)
          }}
        >
          {empty !== undefined && <option value="">{empty}</option>}
          {options.map((o) => (
            <option key={String(o.value)} value={String(o.value)}>
              {o.label}
            </option>
          ))}
        </select>
      )}
    </Field>
  )
}

export function Toggle({
  label,
  checked,
  onChange,
  hint,
}: {
  label: string
  checked: boolean
  onChange: (v: boolean) => void
  hint?: string
}) {
  return (
    <label className="flex cursor-pointer items-start gap-3">
      <span className="relative mt-0.5 inline-flex shrink-0">
        <input
          type="checkbox"
          role="switch"
          className="peer sr-only"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
        />
        <span className="h-6 w-11 rounded-full bg-line transition-colors peer-checked:bg-accent peer-focus-visible:ring-2 peer-focus-visible:ring-accent/50" />
        <span className="absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform peer-checked:translate-x-5" />
      </span>
      <span>
        <span className="block text-sm font-medium">{label}</span>
        {hint && <span className="block text-xs text-muted">{hint}</span>}
      </span>
    </label>
  )
}

/** datetime-local works in local time; the API uses ISO strings. */
export function DateTimeField({
  label,
  value,
  onChange,
  error,
}: {
  label: string
  value: string | null
  onChange: (v: string | null) => void
  error?: string
}) {
  return (
    <Field label={label} error={error}>
      {(id) => (
        <input
          id={id}
          type="datetime-local"
          className={`input ${error ? 'border-sale' : ''}`}
          value={value ? toLocalInput(value) : ''}
          onChange={(e) => onChange(e.target.value ? new Date(e.target.value).toISOString() : null)}
        />
      )}
    </Field>
  )
}

function toLocalInput(iso: string): string {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function Badge({
  tone,
  children,
}: {
  tone: 'green' | 'gray' | 'amber' | 'red' | 'blue'
  children: ReactNode
}) {
  const tones = {
    green: 'bg-emerald-100 text-emerald-800',
    gray: 'bg-sand text-muted',
    amber: 'bg-amber-100 text-amber-800',
    red: 'bg-red-100 text-red-800',
    blue: 'bg-sky-100 text-sky-800',
  }
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${tones[tone]}`}
    >
      {children}
    </span>
  )
}

export function Modal({
  open,
  title,
  onClose,
  children,
}: {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
}) {
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = ''
    }
  }, [open, onClose])

  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center sm:p-6">
      <div className="absolute inset-0 bg-ink/40" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="relative max-h-[92dvh] w-full overflow-y-auto rounded-t-3xl bg-cream p-5 shadow-xl sm:max-w-lg sm:rounded-3xl sm:p-6"
      >
        <div className="mb-5 flex items-center justify-between">
          <h2 className="text-2xl">{title}</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Закрыть"
            className="p-1 text-2xl leading-none"
          >
            ×
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}

/** Form footer with save, optional delete (two-step confirmation) and an error line. */
export function FormActions({
  saving,
  error,
  onDelete,
  deleting,
}: {
  saving: boolean
  error?: unknown
  onDelete?: () => void
  deleting?: boolean
}) {
  const [confirm, setConfirm] = useState(false)
  const message = errorMessage(error)
  return (
    <div className="space-y-3 pt-2">
      {message && (
        <p role="alert" className="text-sm text-sale">
          {message}
        </p>
      )}
      <div className="flex flex-wrap gap-3">
        <button type="submit" className="btn-primary flex-1" disabled={saving}>
          {saving ? 'Сохраняем…' : 'Сохранить'}
        </button>
        {onDelete &&
          (confirm ? (
            <button
              type="button"
              className="btn border border-sale text-sale hover:bg-sale hover:text-white"
              disabled={deleting}
              onClick={onDelete}
            >
              {deleting ? 'Удаляем…' : 'Точно удалить?'}
            </button>
          ) : (
            <button type="button" className="btn-outline" onClick={() => setConfirm(true)}>
              Удалить
            </button>
          ))}
      </div>
    </div>
  )
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="rounded-2xl bg-sand p-8 text-center text-sm text-muted">{children}</div>
}

export function Loading() {
  return <p className="text-sm text-muted">Загрузка…</p>
}
