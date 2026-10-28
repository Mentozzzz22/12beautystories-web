import { formatTenge } from '../../lib/format'
import { usePromos } from '../api'
import { DiscountFields } from '../components/DiscountFields'
import { ResourceModal } from '../components/ResourceModal'
import { useEditing } from '../useEditing'
import {
  Badge,
  DateTimeField,
  Empty,
  Loading,
  NumberField,
  PageHeader,
  TextField,
  Toggle,
} from '../components/ui'
import type { Promo } from '../types'

const newPromo = (): Promo => ({
  id: 0,
  code: '',
  kind: 'percent',
  value: 10,
  minSubtotal: 0,
  combinableWithRules: false,
  maxUses: null,
  usedCount: 0,
  isActive: true,
  startsAt: null,
  endsAt: null,
})

export function PromosPage() {
  const { list, save, remove } = usePromos()
  const editor = useEditing<Promo>(() => {
    save.reset()
    remove.reset()
  })

  return (
    <div className="max-w-3xl">
      <PageHeader
        title="Промокоды"
        action={
          <button type="button" className="btn-primary" onClick={() => editor.open(newPromo())}>
            + Промокод
          </button>
        }
      />

      {list.isPending ? (
        <Loading />
      ) : !list.data?.length ? (
        <Empty>Промокодов пока нет</Empty>
      ) : (
        <ul className="divide-y divide-line overflow-hidden rounded-2xl border border-line bg-white">
          {list.data.map((p) => (
            <li key={p.id}>
              <button
                type="button"
                onClick={() => editor.open(p)}
                className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left hover:bg-sand"
              >
                <span>
                  <span className="block font-mono text-sm font-medium">{p.code}</span>
                  <span className="block text-xs text-muted">
                    {p.kind === 'percent' ? `−${p.value}%` : `−${formatTenge(p.value)}`}
                    {p.minSubtotal > 0 && ` от ${formatTenge(p.minSubtotal)}`} · использован{' '}
                    {p.usedCount}
                    {p.maxUses !== null && ` из ${p.maxUses}`}
                  </span>
                </span>
                <Badge tone={p.isActive ? 'green' : 'gray'}>
                  {p.isActive ? 'Активен' : 'Выключен'}
                </Badge>
              </button>
            </li>
          ))}
        </ul>
      )}

      {editor.editing && (
        <ResourceModal
          title={editor.editing.id ? 'Промокод' : 'Новый промокод'}
          initial={editor.editing}
          save={save}
          remove={remove}
          onClose={editor.close}
        >
          {(p, set, errors) => (
            <>
              <TextField
                label="Код"
                value={p.code}
                onChange={(v) => set('code', v.toUpperCase())}
                error={errors.code}
                hint="Латиница и цифры, например SPRING10"
                autoCapitalize="characters"
              />
              <DiscountFields
                kind={p.kind}
                value={p.value}
                minSubtotal={p.minSubtotal}
                onChange={(patch) => {
                  if (patch.kind !== undefined) set('kind', patch.kind)
                  if (patch.value !== undefined) set('value', patch.value)
                  if (patch.minSubtotal !== undefined) set('minSubtotal', patch.minSubtotal)
                }}
                errors={errors}
              />
              <NumberField
                label="Сколько раз можно использовать"
                value={p.maxUses}
                onChange={(v) => set('maxUses', v)}
                placeholder="Без ограничений"
                error={errors.maxUses}
              />
              <div className="grid gap-4 sm:grid-cols-2">
                <DateTimeField
                  label="Действует с"
                  value={p.startsAt}
                  onChange={(v) => set('startsAt', v)}
                />
                <DateTimeField
                  label="Действует до"
                  value={p.endsAt}
                  onChange={(v) => set('endsAt', v)}
                  error={errors.endsAt && 'Должно быть позже начала'}
                />
              </div>
              <Toggle
                label="Суммируется с автоматическими скидками"
                checked={p.combinableWithRules}
                onChange={(v) => set('combinableWithRules', v)}
                hint="Если выключено, клиент получит одну, более выгодную скидку"
              />
              <Toggle label="Активен" checked={p.isActive} onChange={(v) => set('isActive', v)} />
            </>
          )}
        </ResourceModal>
      )}
    </div>
  )
}
