import { useCategories } from '../api'
import { ResourceModal } from '../components/ResourceModal'
import { useEditing } from '../useEditing'
import { Badge, Empty, Loading, NumberField, PageHeader, TextField, Toggle } from '../components/ui'
import type { Category } from '../types'

const newCategory = (): Category => ({ id: 0, slug: '', name: '', sort: 0, isActive: true })

export function CategoriesPage() {
  const { list, save, remove } = useCategories()
  const editor = useEditing<Category>(() => {
    save.reset()
    remove.reset()
  })

  return (
    <div className="max-w-2xl">
      <PageHeader
        title="Категории"
        action={
          <button type="button" className="btn-primary" onClick={() => editor.open(newCategory())}>
            + Категория
          </button>
        }
      />
      <p className="mb-4 text-sm text-muted">
        Тип средства: шампуни, маски, сыворотки. Порядок задаётся числом.
      </p>

      {list.isPending ? (
        <Loading />
      ) : !list.data?.length ? (
        <Empty>Категорий пока нет</Empty>
      ) : (
        <ul className="divide-y divide-line overflow-hidden rounded-2xl border border-line bg-white">
          {list.data.map((c) => (
            <li key={c.id}>
              <button
                type="button"
                onClick={() => editor.open(c)}
                className="flex w-full items-center justify-between px-4 py-3 text-left hover:bg-sand"
              >
                <span className="text-sm">
                  <span className="mr-3 inline-block w-6 text-muted">{c.sort}</span>
                  {c.name}
                </span>
                {!c.isActive && <Badge tone="gray">Скрыта</Badge>}
              </button>
            </li>
          ))}
        </ul>
      )}

      {editor.editing && (
        <ResourceModal
          title={editor.editing.id ? 'Категория' : 'Новая категория'}
          initial={editor.editing}
          save={save}
          remove={remove}
          onClose={editor.close}
        >
          {(c, set, errors) => (
            <>
              <TextField
                label="Название"
                value={c.name}
                onChange={(v) => set('name', v)}
                error={errors.name}
              />
              <NumberField
                label="Порядок"
                value={c.sort}
                onChange={(v) => set('sort', v ?? 0)}
                hint="Меньше число — выше в меню"
              />
              <Toggle
                label="Показывать на сайте"
                checked={c.isActive}
                onChange={(v) => set('isActive', v)}
              />
            </>
          )}
        </ResourceModal>
      )}
    </div>
  )
}
