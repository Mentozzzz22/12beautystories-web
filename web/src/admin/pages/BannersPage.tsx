import { useState } from 'react'
import { routes } from '../../lib/site'
import { useBanners, useBrands, useCategories } from '../api'
import { ImageField } from '../components/ImageUpload'
import { ResourceModal } from '../components/ResourceModal'
import { useEditing } from '../useEditing'
import {
  Badge,
  DateTimeField,
  Empty,
  Field,
  Loading,
  NumberField,
  PageHeader,
  TextField,
  Toggle,
} from '../components/ui'
import type { Banner } from '../types'

const newBanner = (): Banner => ({
  id: 0,
  title: '',
  subtitle: '',
  imageUrl: '',
  mobileImageUrl: '',
  linkUrl: '',
  sort: 0,
  isActive: true,
  startsAt: null,
  endsAt: null,
})

const CUSTOM = '__custom__'

export function BannersPage() {
  const { list, save, remove } = useBanners()
  const editor = useEditing<Banner>(() => {
    save.reset()
    remove.reset()
  })

  return (
    <div className="max-w-3xl">
      <PageHeader
        title="Баннеры"
        action={
          <button type="button" className="btn-primary" onClick={() => editor.open(newBanner())}>
            + Баннер
          </button>
        }
      />
      <p className="mb-4 text-sm text-muted">
        Слайды на главной: акции, новинки, линии. Лучше всего 3–4 штуки.
      </p>

      {list.isPending ? (
        <Loading />
      ) : !list.data?.length ? (
        <Empty>Баннеров пока нет</Empty>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {list.data.map((b) => (
            <button
              key={b.id}
              type="button"
              onClick={() => editor.open(b)}
              className="overflow-hidden rounded-2xl bg-white text-left hover:ring-2 hover:ring-ink"
            >
              <img src={b.imageUrl} alt="" className="aspect-[16/6] w-full object-cover" />
              <div className="flex items-start justify-between gap-2 p-4">
                <span>
                  <span className="block text-sm font-medium">{b.title || 'Без заголовка'}</span>
                  <span className="block text-xs text-muted">{b.linkUrl || 'Без ссылки'}</span>
                </span>
                <BannerStatus banner={b} />
              </div>
            </button>
          ))}
        </div>
      )}

      {editor.editing && (
        <ResourceModal
          title={editor.editing.id ? 'Баннер' : 'Новый баннер'}
          initial={editor.editing}
          save={save}
          remove={remove}
          onClose={editor.close}
        >
          {(b, set, errors) => (
            <>
              <ImageField
                label="Картинка (широкая, 1600×600)"
                value={b.imageUrl}
                onChange={(v) => set('imageUrl', v)}
                error={errors.imageUrl}
              />
              <ImageField
                label="Картинка для телефона (вертикальная, необязательно)"
                value={b.mobileImageUrl}
                onChange={(v) => set('mobileImageUrl', v)}
                aspect="aspect-[4/5] max-w-48"
              />
              <TextField
                label="Заголовок"
                value={b.title}
                onChange={(v) => set('title', v)}
                error={errors.title}
              />
              <TextField
                label="Подзаголовок"
                value={b.subtitle}
                onChange={(v) => set('subtitle', v)}
              />
              <LinkPicker
                value={b.linkUrl}
                onChange={(v) => set('linkUrl', v)}
                error={errors.linkUrl}
              />
              <div className="grid gap-4 sm:grid-cols-2">
                <DateTimeField
                  label="Показывать с"
                  value={b.startsAt}
                  onChange={(v) => set('startsAt', v)}
                />
                <DateTimeField
                  label="Показывать до"
                  value={b.endsAt}
                  onChange={(v) => set('endsAt', v)}
                  error={errors.endsAt && 'Должно быть позже начала'}
                />
              </div>
              <NumberField
                label="Порядок"
                value={b.sort}
                onChange={(v) => set('sort', v ?? 0)}
                hint="Меньше число — раньше в карусели"
              />
              <Toggle label="Включён" checked={b.isActive} onChange={(v) => set('isActive', v)} />
            </>
          )}
        </ResourceModal>
      )}
    </div>
  )
}

function BannerStatus({ banner }: { banner: Banner }) {
  const [now] = useState(Date.now)
  if (!banner.isActive) return <Badge tone="gray">Выключен</Badge>
  if (banner.startsAt && Date.parse(banner.startsAt) > now)
    return <Badge tone="amber">Запланирован</Badge>
  if (banner.endsAt && Date.parse(banner.endsAt) <= now) return <Badge tone="gray">Завершён</Badge>
  return <Badge tone="green">Показывается</Badge>
}

/** Choose a catalog page for the banner link, or type any address. */
function LinkPicker({
  value,
  onChange,
  error,
}: {
  value: string
  onChange: (v: string) => void
  error?: string
}) {
  const brands = useBrands().list.data ?? []
  const categories = useCategories().list.data ?? []

  const options = [
    { label: 'Весь каталог', to: routes.catalog },
    ...brands.flatMap((b) => [
      { label: b.name, to: routes.brand(b.slug) },
      ...b.lines.map((l) => ({ label: `${b.name} · ${l.name}`, to: routes.line(b.slug, l.slug) })),
    ]),
    ...categories.map((c) => ({ label: `Категория: ${c.name}`, to: routes.category(c.slug) })),
  ]
  const known = value === '' || options.some((o) => o.to === value)

  return (
    <div className="space-y-2">
      <Field label="Куда ведёт баннер" error={error}>
        {(id) => (
          <select
            id={id}
            className="input"
            value={known ? value : CUSTOM}
            onChange={(e) => onChange(e.target.value === CUSTOM ? '/' : e.target.value)}
          >
            <option value="">Никуда</option>
            {options.map((o) => (
              <option key={o.to} value={o.to}>
                {o.label}
              </option>
            ))}
            <option value={CUSTOM}>Свой адрес (например, товар)…</option>
          </select>
        )}
      </Field>
      {!known && (
        <input
          className="input"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="/products/davines-energizing-shampoo"
          aria-label="Свой адрес ссылки"
        />
      )}
    </div>
  )
}
