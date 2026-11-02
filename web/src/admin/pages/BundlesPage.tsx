import { useBundleOptions, useBundles } from '../api'
import { ResourceModal } from '../components/ResourceModal'
import {
  DateTimeField,
  Empty,
  Loading,
  NumberField,
  PageHeader,
  SelectField,
  TextField,
  Toggle,
} from '../components/ui'
import { useEditing } from '../useEditing'
import type { BundleGroup, BundleOffer, BundleOption, BundleTier } from '../types'

const ANY_VOLUME = '*'

const poster = (percent: number, volume: string, withLeaveIn: boolean): BundleTier => ({
  percent,
  slots: [
    { group: 0, label: volume },
    { group: 1, label: volume },
    ...(withLeaveIn ? [{ group: 2, label: '' }] : []),
  ],
})

/** Prefilled with the usual offer: the owner only ticks the products of each group. */
const newOffer = (): BundleOffer => ({
  id: 0,
  name: '',
  isActive: true,
  startsAt: null,
  endsAt: null,
  groups: [
    { name: 'шампунь', productIds: [] },
    { name: 'тритмент', productIds: [] },
    { name: 'несмываемый уход', productIds: [] },
  ],
  tiers: [
    poster(15, '200', false),
    poster(20, '200', true),
    poster(20, '400', false),
    poster(25, '400', true),
  ],
})

export function BundlesPage() {
  const { list, save, remove } = useBundles()
  const options = useBundleOptions().data ?? []
  const editor = useEditing<BundleOffer>(() => {
    save.reset()
    remove.reset()
  })

  return (
    <div className="max-w-3xl">
      <PageHeader
        title="Скидки на наборы"
        action={
          <button type="button" className="btn-primary" onClick={() => editor.open(newOffer())}>
            + Акция
          </button>
        }
      />
      <p className="mb-4 text-sm text-muted">
        Набор собирается из товаров одной линии, скидка действует только на его товары. С другими
        автоматическими скидками не суммируется: покупатель получает более выгодную.
      </p>

      {list.isPending ? (
        <Loading />
      ) : !list.data?.length ? (
        <Empty>Акций пока нет</Empty>
      ) : (
        <ul className="space-y-3">
          {list.data.map((o) => (
            <li
              key={o.id}
              className="flex items-center justify-between gap-4 rounded-2xl bg-white p-4"
            >
              <button type="button" className="min-w-0 text-left" onClick={() => editor.open(o)}>
                <span className="block text-sm font-medium hover:underline">{o.name}</span>
                <span className="block text-xs text-muted">
                  Скидки: {o.tiers.map((t) => `${t.percent}%`).join(', ')}
                </span>
              </button>
              <Toggle
                label={o.isActive ? 'Вкл' : 'Выкл'}
                checked={o.isActive}
                onChange={(isActive) => save.mutate({ ...o, isActive })}
              />
            </li>
          ))}
        </ul>
      )}

      {editor.editing && (
        <ResourceModal
          title={editor.editing.id ? 'Акция' : 'Новая акция'}
          initial={editor.editing}
          save={save}
          remove={remove}
          onClose={editor.close}
        >
          {(o, set, errors) => (
            <>
              <TextField
                label="Название"
                value={o.name}
                onChange={(v) => set('name', v)}
                error={errors.name}
                placeholder="Праздничное предложение"
              />

              <section className="space-y-4">
                <h3 className="text-sm font-semibold">Какие товары входят в группы</h3>
                {o.groups.map((g, i) => (
                  <GroupEditor
                    key={i}
                    group={g}
                    options={options}
                    onChange={(next) =>
                      set(
                        'groups',
                        o.groups.map((cur, j) => (j === i ? next : cur)),
                      )
                    }
                  />
                ))}
                {errors.groups && (
                  <p className="text-xs text-sale">Выберите товары в каждой группе</p>
                )}
              </section>

              <section className="space-y-3">
                <h3 className="text-sm font-semibold">Уровни скидки</h3>
                {o.tiers.map((t, i) => (
                  <TierEditor
                    key={i}
                    number={i + 1}
                    tier={t}
                    groups={o.groups}
                    options={options}
                    onChange={(next) =>
                      set(
                        'tiers',
                        o.tiers.map((cur, j) => (j === i ? next : cur)),
                      )
                    }
                    onRemove={() =>
                      set(
                        'tiers',
                        o.tiers.filter((_, j) => j !== i),
                      )
                    }
                  />
                ))}
                {errors.tiers && (
                  <p className="text-xs text-sale">
                    У каждого уровня нужна скидка от 1 до 100% и хотя бы одна группа
                  </p>
                )}
                <button
                  type="button"
                  className="btn-outline px-4"
                  onClick={() => set('tiers', [...o.tiers, { percent: 10, slots: [] }])}
                >
                  + Уровень
                </button>
              </section>

              <div className="grid gap-4 sm:grid-cols-2">
                <DateTimeField
                  label="Действует с"
                  value={o.startsAt}
                  onChange={(v) => set('startsAt', v)}
                />
                <DateTimeField
                  label="Действует до"
                  value={o.endsAt}
                  onChange={(v) => set('endsAt', v)}
                  error={errors.endsAt && 'Должно быть позже начала'}
                />
              </div>
              <Toggle label="Включена" checked={o.isActive} onChange={(v) => set('isActive', v)} />
            </>
          )}
        </ResourceModal>
      )}
    </div>
  )
}

function GroupEditor({
  group,
  options,
  onChange,
}: {
  group: BundleGroup
  options: BundleOption[]
  onChange: (g: BundleGroup) => void
}) {
  return (
    <div className="space-y-2 rounded-2xl border border-line p-3">
      <TextField
        label="Группа"
        value={group.name}
        onChange={(name) => onChange({ ...group, name })}
        hint="Покажем в подсказке корзины: «Добавьте …»"
      />
      <div className="flex flex-wrap gap-2">
        {options.map((p) => {
          const checked = group.productIds.includes(p.id)
          return (
            <label
              key={p.id}
              className={`cursor-pointer rounded-full border px-3 py-1.5 text-sm has-focus-visible:ring-2 has-focus-visible:ring-accent ${
                checked ? 'border-ink bg-ink text-cream' : 'border-line hover:border-ink'
              }`}
            >
              <input
                type="checkbox"
                className="sr-only"
                checked={checked}
                onChange={() =>
                  onChange({
                    ...group,
                    productIds: checked
                      ? group.productIds.filter((id) => id !== p.id)
                      : [...group.productIds, p.id],
                  })
                }
              />
              {p.name}
            </label>
          )
        })}
      </div>
    </div>
  )
}

function TierEditor({
  number,
  tier,
  groups,
  options,
  onChange,
  onRemove,
}: {
  number: number
  tier: BundleTier
  groups: BundleGroup[]
  options: BundleOption[]
  onChange: (t: BundleTier) => void
  onRemove: () => void
}) {
  const setSlot = (group: number, value: string | null) => {
    const others = tier.slots.filter((s) => s.group !== group)
    const slots =
      value === null
        ? others
        : [...others, { group, label: value === ANY_VOLUME ? '' : value }].sort(
            (a, b) => a.group - b.group,
          )
    onChange({ ...tier, slots })
  }

  return (
    <div className="grid gap-3 rounded-2xl border border-line p-3 sm:grid-cols-2">
      <div className="flex items-center justify-between sm:col-span-2">
        <span className="text-sm font-semibold">Уровень {number}</span>
        <button
          type="button"
          className="text-sm text-sale underline-offset-2 hover:underline"
          onClick={onRemove}
        >
          Удалить
        </button>
      </div>
      <NumberField
        label="Скидка"
        suffix="%"
        value={tier.percent}
        onChange={(v) => onChange({ ...tier, percent: v ?? 0 })}
      />
      {groups.map((g, i) => {
        const slot = tier.slots.find((s) => s.group === i)
        const labels = new Set(
          options.filter((p) => g.productIds.includes(p.id)).flatMap((p) => p.labels),
        )
        if (slot?.label) labels.add(slot.label) // Keep the saved value even if no product has it now.
        return (
          <SelectField<string>
            key={i}
            label={g.name || `Группа ${i + 1}`}
            value={slot ? slot.label || ANY_VOLUME : null}
            empty="Не нужен"
            options={[
              { value: ANY_VOLUME, label: 'Любой объём' },
              ...[...labels].sort().map((l) => ({ value: l, label: l })),
            ]}
            onChange={(v) => setSlot(i, v)}
          />
        )
      })}
    </div>
  )
}
