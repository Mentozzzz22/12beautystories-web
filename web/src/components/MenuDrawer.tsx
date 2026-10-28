import { type ReactNode, useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useMenu } from '../api/queries'
import type { Brand } from '../api/types'
import { routes } from '../lib/site'
import { ChevronIcon, ChevronLeftIcon, CloseIcon } from './icons'

type Tab = 'brands' | 'categories'

interface Props {
  open: boolean
  onClose: () => void
}

export function MenuDrawer({ open, onClose }: Props) {
  const { data: menu, isPending, isError } = useMenu()
  const [tab, setTab] = useState<Tab>('brands')
  const [brand, setBrand] = useState<Brand | null>(null)

  // Reset the brand drill-down so the menu reopens at the top level.
  const close = useCallback(() => {
    setBrand(null)
    onClose()
  }, [onClose])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && close()
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = ''
    }
  }, [open, close])

  return (
    <div className={`fixed inset-0 z-50 ${open ? '' : 'pointer-events-none'}`} aria-hidden={!open}>
      <div
        className={`absolute inset-0 bg-ink/40 transition-opacity ${open ? 'opacity-100' : 'opacity-0'}`}
        onClick={close}
      />
      <nav
        aria-label="Каталог"
        className={`absolute inset-y-0 left-0 flex w-[88%] max-w-sm flex-col bg-cream shadow-xl transition-transform duration-300 ${
          open ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex items-center justify-between border-b border-line px-5 py-4">
          <span className="font-serif text-2xl">Каталог</span>
          <button type="button" onClick={close} aria-label="Закрыть меню" className="p-1">
            <CloseIcon />
          </button>
        </div>

        <div className="flex border-b border-line text-sm" role="tablist">
          {(['brands', 'categories'] as const).map((t) => (
            <button
              key={t}
              type="button"
              role="tab"
              aria-selected={tab === t}
              onClick={() => {
                setTab(t)
                setBrand(null)
              }}
              className={`flex-1 py-3 font-medium transition-colors ${
                tab === t ? 'border-b-2 border-ink text-ink' : 'text-muted'
              }`}
            >
              {t === 'brands' ? 'Бренды' : 'Средства'}
            </button>
          ))}
        </div>

        <div className="flex-1 overflow-y-auto">
          {isPending && <p className="p-5 text-sm text-muted">Загрузка…</p>}
          {isError && <p className="p-5 text-sm text-sale">Не удалось загрузить меню</p>}

          {menu && tab === 'brands' && !brand && (
            <ul>
              <MenuLink to={routes.catalog} onClick={close}>
                Все товары
              </MenuLink>
              {menu.brands.map((b) =>
                b.lines.length > 0 ? (
                  <li key={b.id}>
                    <button
                      type="button"
                      onClick={() => setBrand(b)}
                      className="flex w-full items-center justify-between px-5 py-4 text-left hover:bg-sand"
                    >
                      {b.name}
                      <ChevronIcon width={18} height={18} className="text-muted" />
                    </button>
                  </li>
                ) : (
                  <MenuLink key={b.id} to={routes.brand(b.slug)} onClick={close}>
                    {b.name}
                  </MenuLink>
                ),
              )}
            </ul>
          )}

          {menu && tab === 'brands' && brand && (
            <div>
              <button
                type="button"
                onClick={() => setBrand(null)}
                className="flex w-full items-center gap-2 px-5 py-4 text-sm text-muted hover:text-ink"
              >
                <ChevronLeftIcon width={18} height={18} /> Все бренды
              </button>
              <ul>
                <MenuLink to={routes.brand(brand.slug)} onClick={close} strong>
                  Весь {brand.name}
                </MenuLink>
                {brand.lines.map((l) => (
                  <MenuLink key={l.id} to={routes.line(brand.slug, l.slug)} onClick={close}>
                    {l.name}
                  </MenuLink>
                ))}
              </ul>
            </div>
          )}

          {menu && tab === 'categories' && (
            <ul>
              {menu.categories.map((c) => (
                <MenuLink key={c.id} to={routes.category(c.slug)} onClick={close}>
                  {c.name}
                </MenuLink>
              ))}
            </ul>
          )}
        </div>
      </nav>
    </div>
  )
}

interface MenuLinkProps {
  to: string
  onClick: () => void
  strong?: boolean
  children: ReactNode
}

function MenuLink({ to, onClick, strong, children }: MenuLinkProps) {
  return (
    <li>
      <Link
        to={to}
        onClick={onClick}
        className={`block px-5 py-4 hover:bg-sand ${strong ? 'font-medium' : ''}`}
      >
        {children}
      </Link>
    </li>
  )
}
