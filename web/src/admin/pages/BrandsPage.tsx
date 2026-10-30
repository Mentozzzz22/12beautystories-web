import { useBrands, useLines } from '../api'
import { ImageField } from '../components/ImageUpload'
import { ResourceModal } from '../components/ResourceModal'
import { RichTextArea } from '../components/RichTextArea'
import { useEditing } from '../useEditing'
import { Badge, Empty, Loading, PageHeader, TextField, Toggle } from '../components/ui'
import type { Brand, Line } from '../types'

const newBrand = (): Brand => ({
  id: 0,
  slug: '',
  name: '',
  description: '',
  logoUrl: '',
  sort: 0,
  isActive: true,
  lines: [],
})

const newLine = (brandId: number): Line => ({
  id: 0,
  brandId,
  slug: '',
  name: '',
  description: '',
  imageUrl: '',
  sort: 0,
  isActive: true,
})

export function BrandsPage() {
  const brands = useBrands()
  const lines = useLines()
  const brandEditor = useEditing<Brand>(() => {
    brands.save.reset()
    brands.remove.reset()
  })
  const lineEditor = useEditing<Line>(() => {
    lines.save.reset()
    lines.remove.reset()
  })

  return (
    <div className="max-w-3xl">
      <PageHeader
        title="Бренды и линии"
        action={
          <button
            type="button"
            className="btn-primary"
            onClick={() => brandEditor.open(newBrand())}
          >
            + Бренд
          </button>
        }
      />

      {brands.list.isPending ? (
        <Loading />
      ) : !brands.list.data?.length ? (
        <Empty>Добавьте первый бренд</Empty>
      ) : (
        <div className="space-y-4">
          {brands.list.data.map((b) => (
            <section key={b.id} className="rounded-2xl bg-white p-5">
              <div className="flex items-center justify-between gap-3">
                <button type="button" className="text-left" onClick={() => brandEditor.open(b)}>
                  <span className="font-serif text-2xl hover:underline">{b.name}</span>{' '}
                  {!b.isActive && <Badge tone="gray">Скрыт</Badge>}
                </button>
                <button
                  type="button"
                  className="btn-outline py-1.5 text-xs"
                  onClick={() => lineEditor.open(newLine(b.id))}
                >
                  + Линия
                </button>
              </div>
              {b.lines.length > 0 ? (
                <ul className="mt-3 flex flex-wrap gap-2">
                  {b.lines.map((l) => (
                    <li key={l.id}>
                      <button
                        type="button"
                        onClick={() => lineEditor.open(l)}
                        className={`rounded-full border px-3 py-1.5 text-sm hover:border-ink ${
                          l.isActive ? 'border-line' : 'border-dashed border-line text-muted'
                        }`}
                      >
                        {l.name}
                      </button>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-2 text-sm text-muted">Линий пока нет</p>
              )}
            </section>
          ))}
        </div>
      )}

      {brandEditor.editing && (
        <ResourceModal
          title={brandEditor.editing.id ? 'Бренд' : 'Новый бренд'}
          initial={brandEditor.editing}
          save={brands.save}
          remove={brands.remove}
          onClose={brandEditor.close}
        >
          {(b, set, errors) => (
            <>
              <TextField
                label="Название"
                value={b.name}
                onChange={(v) => set('name', v)}
                error={errors.name}
              />
              <RichTextArea
                label="Описание"
                value={b.description}
                onChange={(v) => set('description', v)}
                error={errors.description}
                rows={3}
              />
              <ImageField
                label="Логотип"
                value={b.logoUrl}
                onChange={(v) => set('logoUrl', v)}
                aspect="aspect-[3/1]"
                error={errors.logoUrl}
              />
              <Toggle
                label="Показывать на сайте"
                checked={b.isActive}
                onChange={(v) => set('isActive', v)}
              />
              {errors.slug && <p className="text-xs text-sale">Адрес страницы: {errors.slug}</p>}
            </>
          )}
        </ResourceModal>
      )}

      {lineEditor.editing && (
        <ResourceModal
          title={lineEditor.editing.id ? 'Линия' : 'Новая линия'}
          initial={lineEditor.editing}
          save={lines.save}
          remove={lines.remove}
          onClose={lineEditor.close}
        >
          {(l, set, errors) => (
            <>
              <TextField
                label="Название"
                value={l.name}
                onChange={(v) => set('name', v)}
                error={errors.name}
              />
              <RichTextArea
                label="Описание"
                value={l.description}
                onChange={(v) => set('description', v)}
                hint="Показывается на странице линии"
                rows={3}
              />
              <ImageField
                label="Общее фото линии"
                value={l.imageUrl}
                onChange={(v) => set('imageUrl', v)}
                aspect="aspect-[16/6]"
                error={errors.imageUrl}
              />
              <Toggle
                label="Показывать на сайте"
                checked={l.isActive}
                onChange={(v) => set('isActive', v)}
              />
            </>
          )}
        </ResourceModal>
      )}
    </div>
  )
}
