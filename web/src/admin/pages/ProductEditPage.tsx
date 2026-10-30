import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { adminApi, useBrands, useCategories } from '../api'
import { GalleryField } from '../components/ImageUpload'
import { RichTextArea } from '../components/RichTextArea'
import { fieldErrors } from '../errors'
import { FormActions, Loading, NumberField, SelectField, TextField, Toggle } from '../components/ui'
import type { Media, OptionKind, Product, Variant } from '../types'

const emptyVariant = (): Variant => ({
  id: 0,
  label: '',
  sku: '',
  price: 0,
  compareAtPrice: null,
  stock: 0,
  isActive: true,
  imageUrl: '',
})

const emptyProduct = (): Product => ({
  id: 0,
  slug: '',
  name: '',
  brandId: 0,
  lineId: null,
  categoryId: null,
  description: '',
  ingredients: '',
  optionKind: 'none',
  sort: 0,
  isActive: true,
  media: [],
  variants: [emptyVariant()],
})

export function ProductEditPage() {
  const param = useParams().id
  const id = param === 'new' ? 0 : Number(param)
  const product = useQuery({
    queryKey: ['admin', 'product', id],
    queryFn: () => adminApi.product(id),
    enabled: id > 0,
  })

  if (id > 0 && product.isPending) return <Loading />
  if (id > 0 && product.isError) return <p className="text-sm text-sale">Товар не найден</p>
  return <ProductForm key={id} initial={product.data ?? emptyProduct()} />
}

/** Picks which gallery photo opens with a variant; a second click on the photo clears the choice. */
function VariantPhotoPicker({
  label,
  media,
  value,
  onChange,
  error,
}: {
  label: string
  media: Media[]
  value: string
  onChange: (url: string) => void
  error?: string
}) {
  const images = media.filter((m) => m.kind === 'image')
  if (images.length === 0) return null

  return (
    <div className="mt-3">
      <p className="mb-1.5 text-sm font-medium">{label}</p>
      <div className="flex flex-wrap gap-2">
        {images.map((m, i) => (
          <button
            key={m.url}
            type="button"
            aria-label={`${label}: фото ${i + 1}`}
            aria-pressed={m.url === value}
            onClick={() => onChange(m.url === value ? '' : m.url)}
            className={`h-14 w-14 overflow-hidden rounded-lg border-2 ${
              m.url === value ? 'border-ink' : 'border-line hover:border-muted'
            }`}
          >
            <img src={m.url} alt="" className="h-full w-full object-cover" />
          </button>
        ))}
      </div>
      {error && <p className="mt-1 text-xs text-sale">{error}</p>}
    </div>
  )
}

const OPTION_KINDS: { value: OptionKind; label: string; hint: string }[] = [
  { value: 'none', label: 'Один вариант', hint: 'Например, крем в одном объёме' },
  { value: 'volume', label: 'Разные объёмы', hint: '90 мл, 250 мл, 1 л' },
  { value: 'color', label: 'Разные цвета', hint: 'Расчёски, аксессуары' },
]

export function ProductForm({ initial }: { initial: Product }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [p, setP] = useState<Product>(initial)
  const brands = useBrands().list.data ?? []
  const categories = useCategories().list.data ?? []
  const lines = brands.find((b) => b.id === p.brandId)?.lines ?? []

  const save = useMutation({
    mutationFn: () => adminApi.saveProduct(p),
    onSuccess: (saved) => {
      void qc.invalidateQueries()
      qc.setQueryData(['admin', 'product', saved.id], saved)
      if (!p.id) navigate(`/admin/products/${saved.id}`, { replace: true })
      setP(saved)
    },
  })
  const remove = useMutation({
    mutationFn: () => adminApi.deleteProduct(p.id),
    onSuccess: () => {
      void qc.invalidateQueries()
      navigate('/admin/products', { replace: true })
    },
  })
  const errors = fieldErrors(save.error)

  const set = <K extends keyof Product>(key: K, value: Product[K]) =>
    setP((cur) => ({ ...cur, [key]: value }))
  const setVariant = (i: number, patch: Partial<Variant>) =>
    setP((cur) => ({
      ...cur,
      variants: cur.variants.map((v, j) => (j === i ? { ...v, ...patch } : v)),
    }))

  const setOptionKind = (kind: OptionKind) =>
    setP((cur) => ({
      ...cur,
      optionKind: kind,
      // A single-option product keeps only its first variant, with nothing to switch photos.
      variants:
        kind === 'none' ? cur.variants.slice(0, 1).map((v) => ({ ...v, imageUrl: '' })) : cur.variants,
    }))

  const setMedia = (media: Media[]) =>
    setP((cur) => ({
      ...cur,
      media,
      // A variant must not keep pointing at a photo removed from the gallery.
      variants: cur.variants.map((v) =>
        media.some((m) => m.url === v.imageUrl) ? v : { ...v, imageUrl: '' },
      ),
    }))

  const submit = (e: FormEvent) => {
    e.preventDefault()
    save.mutate()
  }

  const multi = p.optionKind !== 'none'
  const variantLabel = p.optionKind === 'color' ? 'Цвет' : 'Объём'

  return (
    <form onSubmit={submit} noValidate className="max-w-3xl space-y-8 pb-10">
      <div>
        <Link to="/admin/products" className="text-sm text-muted hover:text-ink">
          ← Все товары
        </Link>
        <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
          <h1 className="text-3xl sm:text-4xl">{p.id ? p.name || 'Товар' : 'Новый товар'}</h1>
          {p.id > 0 && p.isActive && (
            <a
              href={`/products/${p.slug}`}
              target="_blank"
              rel="noreferrer"
              className="text-sm text-muted underline-offset-4 hover:text-ink hover:underline"
            >
              Открыть на сайте ↗
            </a>
          )}
        </div>
      </div>

      <section className="space-y-4 rounded-2xl bg-white p-5">
        <TextField
          label="Название"
          value={p.name}
          onChange={(v) => set('name', v)}
          error={errors.name}
        />
        <div className="grid gap-4 sm:grid-cols-3">
          <SelectField
            label="Бренд"
            value={p.brandId || null}
            onChange={(v) => setP((cur) => ({ ...cur, brandId: v ?? 0, lineId: null }))}
            options={brands.map((b) => ({ value: b.id, label: b.name }))}
            empty="Выберите бренд"
            error={errors.brandId}
          />
          <SelectField
            label="Линия"
            value={p.lineId}
            onChange={(v) => set('lineId', v)}
            options={lines.map((l) => ({ value: l.id, label: l.name }))}
            empty={lines.length ? 'Без линии' : 'У бренда нет линий'}
            error={errors.lineId}
          />
          <SelectField
            label="Категория"
            value={p.categoryId}
            onChange={(v) => set('categoryId', v)}
            options={categories.map((c) => ({ value: c.id, label: c.name }))}
            empty="Без категории"
            error={errors.categoryId}
          />
        </div>
        <RichTextArea
          label="Описание"
          value={p.description}
          onChange={(v) => set('description', v)}
          error={errors.description}
          hint="3–4 предложения: для чего средство, текстура, результат"
        />
        <RichTextArea
          label="Активные ингредиенты"
          value={p.ingredients}
          onChange={(v) => set('ingredients', v)}
          error={errors.ingredients}
          rows={3}
        />
      </section>

      <section className="rounded-2xl bg-white p-5">
        <GalleryField value={p.media} onChange={setMedia} error={errors.media} />
      </section>

      <section className="space-y-4 rounded-2xl bg-white p-5">
        <h2 className="text-2xl">Цена и наличие</h2>
        <fieldset className="grid gap-2 sm:grid-cols-3">
          <legend className="sr-only">Варианты товара</legend>
          {OPTION_KINDS.map((k) => (
            <label
              key={k.value}
              className={`cursor-pointer rounded-xl border p-3 ${p.optionKind === k.value ? 'border-ink' : 'border-line'}`}
            >
              <input
                type="radio"
                name="optionKind"
                className="sr-only"
                checked={p.optionKind === k.value}
                onChange={() => setOptionKind(k.value)}
              />
              <span className="block text-sm font-medium">{k.label}</span>
              <span className="block text-xs text-muted">{k.hint}</span>
            </label>
          ))}
        </fieldset>
        {errors.variants && <p className="text-xs text-sale">{errors.variants}</p>}

        <div className="space-y-3">
          {p.variants.map((v, i) => {
            const err = (f: string) => errors[`variants.${i}.${f}`]
            return (
              <div key={v.id || `new-${i}`} className="rounded-xl border border-line p-4">
                <div className="grid gap-3 sm:grid-cols-4">
                  {multi && (
                    <TextField
                      label={variantLabel}
                      value={v.label}
                      onChange={(label) => setVariant(i, { label })}
                      placeholder={p.optionKind === 'color' ? 'Чёрный' : '250 мл'}
                      error={err('label')}
                    />
                  )}
                  <NumberField
                    label="Цена"
                    suffix="₸"
                    value={v.price}
                    onChange={(price) => setVariant(i, { price: price ?? 0 })}
                    error={err('price')}
                  />
                  <NumberField
                    label="Старая цена"
                    suffix="₸"
                    value={v.compareAtPrice}
                    onChange={(compareAtPrice) => setVariant(i, { compareAtPrice })}
                    hint="Если есть скидка"
                    error={err('compareAtPrice') && 'Должна быть больше цены'}
                  />
                  <NumberField
                    label="Остаток"
                    suffix="шт"
                    value={v.stock}
                    onChange={(stock) => setVariant(i, { stock: stock ?? 0 })}
                    error={err('stock')}
                  />
                </div>
                {multi && (
                  <VariantPhotoPicker
                    label={p.optionKind === 'color' ? 'Фото этого цвета' : 'Фото этого объёма'}
                    media={p.media}
                    value={v.imageUrl}
                    onChange={(imageUrl) => setVariant(i, { imageUrl })}
                    error={err('imageUrl')}
                  />
                )}
                {multi && (
                  <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
                    <Toggle
                      label="Продаётся"
                      checked={v.isActive}
                      onChange={(isActive) => setVariant(i, { isActive })}
                    />
                    {p.variants.length > 1 && (
                      <button
                        type="button"
                        className="text-xs text-muted hover:text-sale"
                        onClick={() =>
                          set(
                            'variants',
                            p.variants.filter((_, j) => j !== i),
                          )
                        }
                      >
                        Удалить вариант
                      </button>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
        {multi && (
          <button
            type="button"
            className="btn-outline py-2"
            onClick={() => set('variants', [...p.variants, emptyVariant()])}
          >
            + Добавить {p.optionKind === 'color' ? 'цвет' : 'объём'}
          </button>
        )}
      </section>

      <section className="space-y-4 rounded-2xl bg-white p-5">
        <Toggle
          label="Показывать на сайте"
          checked={p.isActive}
          onChange={(v) => set('isActive', v)}
          hint="Выключите, чтобы временно скрыть товар"
        />
        <details>
          <summary className="cursor-pointer text-sm text-muted">Дополнительно</summary>
          <div className="mt-4">
            <TextField
              label="Адрес страницы"
              value={p.slug}
              onChange={(v) => set('slug', v)}
              error={errors.slug}
              hint="Латиница и дефисы. Оставьте пустым, заполнится из названия"
            />
          </div>
        </details>
      </section>

      <div className="sticky bottom-0 -mx-4 border-t border-line bg-cream/95 px-4 py-3 backdrop-blur sm:mx-0 sm:rounded-2xl sm:border">
        <FormActions
          saving={save.isPending}
          error={save.error ?? remove.error}
          onDelete={p.id ? () => remove.mutate() : undefined}
          deleting={remove.isPending}
        />
        {save.isSuccess && !save.isPending && (
          <p className="mt-2 text-sm text-accent-dark">Сохранено</p>
        )}
      </div>
    </form>
  )
}
