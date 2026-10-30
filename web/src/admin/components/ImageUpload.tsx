import { useRef, useState } from 'react'
import { adminApi } from '../api'
import type { Media } from '../types'

const ACCEPT = 'image/jpeg,image/png,image/webp'

function useUpload() {
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const upload = async (files: File[]): Promise<string[]> => {
    setUploading(true)
    setError(null)
    const urls: string[] = []
    try {
      for (const file of files) {
        urls.push((await adminApi.upload(file)).url)
      }
    } catch {
      setError('Не удалось загрузить фото. Подходят JPG, PNG и WebP до 15 МБ')
    } finally {
      setUploading(false)
    }
    return urls
  }
  return { upload, uploading, error }
}

/** Single image (banner, brand logo). */
export function ImageField({
  label,
  value,
  onChange,
  error,
  aspect = 'aspect-[16/6]',
  fit = 'object-cover',
}: {
  label: string
  value: string
  onChange: (url: string) => void
  error?: string
  aspect?: string
  /** How the preview fits: match how the site shows this image. */
  fit?: 'object-cover' | 'object-contain'
}) {
  const input = useRef<HTMLInputElement>(null)
  const { upload, uploading, error: uploadError } = useUpload()

  return (
    <div>
      <p className="mb-1.5 text-sm font-medium">{label}</p>
      <button
        type="button"
        onClick={() => input.current?.click()}
        className={`relative flex w-full items-center justify-center overflow-hidden rounded-2xl border-2 border-dashed bg-white ${aspect} ${
          error ? 'border-sale' : 'border-line hover:border-ink'
        }`}
      >
        {value ? (
          <img src={value} alt="" className={`absolute inset-0 h-full w-full ${fit}`} />
        ) : (
          <span className="text-sm text-muted">
            {uploading ? 'Загружаем…' : 'Нажмите, чтобы выбрать фото'}
          </span>
        )}
        {value && uploading && (
          <span className="absolute inset-0 flex items-center justify-center bg-white/70 text-sm">
            Загружаем…
          </span>
        )}
      </button>
      <input
        ref={input}
        type="file"
        accept={ACCEPT}
        hidden
        onChange={async (e) => {
          const file = e.target.files?.[0]
          e.target.value = ''
          if (!file) return
          const [url] = await upload([file])
          if (url) onChange(url)
        }}
      />
      {value && (
        <button
          type="button"
          onClick={() => onChange('')}
          className="mt-1 text-xs text-muted hover:text-sale"
        >
          Убрать фото
        </button>
      )}
      {(uploadError || error) && <p className="mt-1 text-xs text-sale">{uploadError || error}</p>}
    </div>
  )
}

/** Ordered product gallery: first photo is the cover in listings. */
export function GalleryField({
  value,
  onChange,
  error,
}: {
  value: Media[]
  onChange: (media: Media[]) => void
  error?: string
}) {
  const input = useRef<HTMLInputElement>(null)
  const { upload, uploading, error: uploadError } = useUpload()

  const move = (from: number, to: number) => {
    if (to < 0 || to >= value.length) return
    const next = [...value]
    const [item] = next.splice(from, 1)
    if (item) next.splice(to, 0, item)
    onChange(next)
  }

  return (
    <div>
      <p className="mb-1.5 text-sm font-medium">Фотографии</p>
      <p className="mb-3 text-xs text-muted">
        Первое фото показывается в каталоге. Стрелками меняйте порядок.
      </p>
      <div className="grid grid-cols-3 gap-3 sm:grid-cols-4">
        {value.map((m, i) => (
          <div
            key={m.url}
            className="group relative aspect-square overflow-hidden rounded-xl bg-sand"
          >
            <img src={m.url} alt="" className="h-full w-full object-cover" />
            {i === 0 && (
              <span className="absolute top-1.5 left-1.5 rounded-full bg-ink px-2 py-0.5 text-[10px] text-cream">
                Обложка
              </span>
            )}
            <div className="absolute inset-x-0 bottom-0 flex justify-between bg-ink/60 p-1 text-white">
              <button
                type="button"
                aria-label="Сдвинуть влево"
                className="px-2 disabled:opacity-30"
                disabled={i === 0}
                onClick={() => move(i, i - 1)}
              >
                ←
              </button>
              <button
                type="button"
                aria-label="Удалить фото"
                className="px-2"
                onClick={() => onChange(value.filter((_, j) => j !== i))}
              >
                ✕
              </button>
              <button
                type="button"
                aria-label="Сдвинуть вправо"
                className="px-2 disabled:opacity-30"
                disabled={i === value.length - 1}
                onClick={() => move(i, i + 1)}
              >
                →
              </button>
            </div>
          </div>
        ))}
        <button
          type="button"
          onClick={() => input.current?.click()}
          disabled={uploading}
          className="flex aspect-square flex-col items-center justify-center rounded-xl border-2 border-dashed border-line bg-white text-sm text-muted hover:border-ink"
        >
          <span className="text-2xl leading-none">+</span>
          <span className="mt-1 text-xs">{uploading ? 'Загружаем…' : 'Добавить'}</span>
        </button>
      </div>
      <input
        ref={input}
        type="file"
        accept={ACCEPT}
        multiple
        hidden
        onChange={async (e) => {
          const files = Array.from(e.target.files ?? [])
          e.target.value = ''
          if (files.length === 0) return
          const urls = await upload(files)
          onChange([...value, ...urls.map((url) => ({ url, kind: 'image' as const }))])
        }}
      />
      {(uploadError || error) && <p className="mt-2 text-xs text-sale">{uploadError || error}</p>}
    </div>
  )
}
