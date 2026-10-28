import { useCallback, useEffect, useState } from 'react'
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { cartCount, useCart } from '../lib/cart'
import { routes, site } from '../lib/site'
import { BagIcon, InstagramIcon, MenuIcon, SearchIcon, WhatsAppIcon } from './icons'
import { MenuDrawer } from './MenuDrawer'

export function Layout() {
  const [menuOpen, setMenuOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const count = cartCount(useCart())
  const { pathname } = useLocation()
  const closeMenu = useCallback(() => setMenuOpen(false), [])

  useEffect(() => {
    window.scrollTo(0, 0)
  }, [pathname])

  return (
    <div className="flex min-h-dvh flex-col">
      <header className="sticky top-0 z-40 border-b border-line bg-cream/95 backdrop-blur">
        <div className="container-page grid h-16 grid-cols-[1fr_auto_1fr] items-center">
          <button
            type="button"
            onClick={() => setMenuOpen(true)}
            aria-label="Открыть каталог"
            className="-ml-1 justify-self-start p-1"
          >
            <MenuIcon />
          </button>
          <Link to={routes.home} className="font-serif text-2xl tracking-wide sm:text-3xl">
            12 Beauty Stories
          </Link>
          <div className="flex items-center gap-3 justify-self-end">
            <button
              type="button"
              onClick={() => setSearchOpen((v) => !v)}
              aria-label="Поиск"
              aria-expanded={searchOpen}
              className="p-1"
            >
              <SearchIcon />
            </button>
            <Link
              to={routes.cart}
              aria-label={`Корзина, товаров: ${count}`}
              className="relative p-1"
            >
              <BagIcon />
              {count > 0 && (
                <span className="absolute -top-0.5 -right-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-accent px-1 text-[11px] font-semibold text-white">
                  {count}
                </span>
              )}
            </Link>
          </div>
        </div>
        {searchOpen && <SearchBar onDone={() => setSearchOpen(false)} />}
      </header>

      <MenuDrawer open={menuOpen} onClose={closeMenu} />

      <main className="flex-1">
        <Outlet />
      </main>

      <footer className="mt-16 border-t border-line bg-sand">
        <div className="container-page flex flex-col gap-6 py-10 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="font-serif text-2xl">{site.name}</p>
            <p className="mt-1 text-sm text-muted">
              Профессиональный уход за волосами и кожей · {site.city}
            </p>
          </div>
          <div className="flex gap-3">
            <a
              href={site.instagramUrl}
              target="_blank"
              rel="noreferrer"
              className="btn-outline py-2"
            >
              <InstagramIcon width={18} height={18} /> Instagram
            </a>
            <a
              href={site.whatsappUrl}
              target="_blank"
              rel="noreferrer"
              className="btn-outline py-2"
            >
              <WhatsAppIcon width={18} height={18} /> WhatsApp
            </a>
          </div>
        </div>
      </footer>
    </div>
  )
}

function SearchBar({ onDone }: { onDone: () => void }) {
  const navigate = useNavigate()
  const [q, setQ] = useState('')

  return (
    <form
      role="search"
      className="container-page pb-3"
      onSubmit={(e) => {
        e.preventDefault()
        const query = q.trim()
        if (!query) return
        navigate(`${routes.catalog}?q=${encodeURIComponent(query)}`)
        onDone()
      }}
    >
      <input
        autoFocus
        type="search"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="Поиск по названию, бренду или линии"
        className="input"
      />
    </form>
  )
}
