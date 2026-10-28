import { formatTenge } from '../../lib/format'
import { useBrands, useRules } from '../api'
import { DiscountFields } from '../components/DiscountFields'
import { ResourceModal } from '../components/ResourceModal'
import { useEditing } from '../useEditing'
import {
  DateTimeField,
  Empty,
  Loading,
  PageHeader,
  SelectField,
  TextField,
  Toggle,
} from '../components/ui'
import type { Rule } from '../types'

const newRule = (): Rule => ({
  id: 0,
  name: '',
  brandId: null,
  brandName: '',
  minSubtotal: 50000,
  kind: 'percent',
  value: 10,
  isActive: true,
  startsAt: null,
  endsAt: null,
})

function describeRule(r: Pick<Rule, 'brandName' | 'minSubtotal' | 'kind' | 'value'>): string {
  const off = r.kind === 'percent' ? `−${r.value}%` : `−${formatTenge(r.value)}`
  const scope = r.brandName ? `товары ${r.brandName}` : 'вся корзина'
  const from = r.minSubtotal > 0 ? ` от ${formatTenge(r.minSubtotal)}` : ''
  return `${scope}${from}: ${off}`
}

export function RulesPage() {
  const { list, save, remove } = useRules()
  const brands = useBrands().list.data ?? []
  const editor = useEditing<Rule>(() => {
    save.reset()
    remove.reset()
  })

  return (
    <div className="max-w-3xl">
      <PageHeader
        title="Автоматические скидки"
        action={
          <button type="button" className="btn-primary" onClick={() => editor.open(newRule())}>
            + Скидка
          </button>
        }
      />
      <p className="mb-4 text-sm text-muted">
        Применяются в корзине сами, без промокода. Например: «Davines от 50 000 ₸ — минус 10%».
      </p>

      {list.isPending ? (
        <Loading />
      ) : !list.data?.length ? (
        <Empty>Скидок пока нет</Empty>
      ) : (
        <ul className="space-y-3">
          {list.data.map((r) => (
            <li
              key={r.id}
              className="flex items-center justify-between gap-4 rounded-2xl bg-white p-4"
            >
              <button type="button" className="min-w-0 text-left" onClick={() => editor.open(r)}>
                <span className="block text-sm font-medium hover:underline">{r.name}</span>
                <span className="block text-xs text-muted">{describeRule(r)}</span>
              </button>
              {/* Quick on/off without opening the form. */}
              <Toggle
                label={r.isActive ? 'Вкл' : 'Выкл'}
                checked={r.isActive}
                onChange={(isActive) => save.mutate({ ...r, isActive })}
              />
            </li>
          ))}
        </ul>
      )}

      {editor.editing && (
        <ResourceModal
          title={editor.editing.id ? 'Скидка' : 'Новая скидка'}
          initial={editor.editing}
          save={save}
          remove={remove}
          onClose={editor.close}
        >
          {(r, set, errors) => (
            <>
              <TextField
                label="Название"
                value={r.name}
                onChange={(v) => set('name', v)}
                error={errors.name}
                placeholder="Davines от 50 000 ₸: −10%"
              />
              <SelectField
                label="На какие товары"
                value={r.brandId}
                onChange={(v) => set('brandId', v)}
                options={brands.map((b) => ({ value: b.id, label: `Только ${b.name}` }))}
                empty="Вся корзина"
                error={errors.brandId}
              />
              <DiscountFields
                kind={r.kind}
                value={r.value}
                minSubtotal={r.minSubtotal}
                minSubtotalLabel={r.brandId ? 'Сумма товаров бренда от' : 'Сумма заказа от'}
                onChange={(patch) => {
                  if (patch.kind !== undefined) set('kind', patch.kind)
                  if (patch.value !== undefined) set('value', patch.value)
                  if (patch.minSubtotal !== undefined) set('minSubtotal', patch.minSubtotal)
                }}
                errors={errors}
              />
              <div className="grid gap-4 sm:grid-cols-2">
                <DateTimeField
                  label="Действует с"
                  value={r.startsAt}
                  onChange={(v) => set('startsAt', v)}
                />
                <DateTimeField
                  label="Действует до"
                  value={r.endsAt}
                  onChange={(v) => set('endsAt', v)}
                  error={errors.endsAt && 'Должно быть позже начала'}
                />
              </div>
              <Toggle label="Включена" checked={r.isActive} onChange={(v) => set('isActive', v)} />
            </>
          )}
        </ResourceModal>
      )}
    </div>
  )
}
