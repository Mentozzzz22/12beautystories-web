import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, Navigate, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { CloseIcon, MenuIcon } from '../components/icons'
import { adminApi, isUnauthenticated, useMe } from './api'

const NAV = [
  { to: '/admin/orders', label: 'Заказы' },
  { to: '/admin/customers', label: 'Клиенты' },
  { to: '/admin/products', label: 'Товары' },
  { to: '/admin/brands', label: 'Бренды и линии' },
  { to: '/admin/categories', label: 'Категории' },
  { to: '/admin/banners', label: 'Баннеры' },
  { to: '/admin/promos', label: 'Промокоды' },
  { to: '/admin/rules', label: 'Скидки' },
] as const

export function AdminLayout() {
  const me = useMe()
  const { pathname } = useLocation()

  if (me.isPending) return <p className="p-6 text-sm text-muted">Загрузка…</p>
  if (isUnauthenticated(me.error)) {
    return <Navigate to="/admin/login" replace state={{ from: pathname }} />
  }
  if (me.isError) return <p className="p-6 text-sm text-sale">Сервер недоступен</p>

  return <Shell email={me.data.email} />
}

function Shell({ email }: { email: string }) {
  const [open, setOpen] = useState(false)
  const qc = useQueryClient()
  const navigate = useNavigate()
  // Light poll so a new order shows up without reloading.
  const newOrders = useQuery({
    queryKey: ['admin', 'orders', 'badge'],
    queryFn: () => adminApi.orders('new', '', 1),
    refetchInterval: 60_000,
    select: (p) => p.newCount,
  })

  const logout = async () => {
    await adminApi.logout().catch(() => undefined)
    qc.clear()
    navigate('/admin/login', { replace: true })
  }

  const nav = (
    <nav className="flex flex-col gap-1">
      {NAV.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          onClick={() => setOpen(false)}
          className={({ isActive }) =>
            `flex items-center justify-between rounded-xl px-4 py-2.5 text-sm transition-colors ${
              isActive ? 'bg-ink text-cream' : 'hover:bg-sand'
            }`
          }
        >
          {item.label}
          {item.to === '/admin/orders' && (newOrders.data ?? 0) > 0 && (
            <span className="rounded-full bg-accent px-2 text-xs text-white">{newOrders.data}</span>
          )}
        </NavLink>
      ))}
    </nav>
  )

  const footer = (
    <div className="mt-auto space-y-2 border-t border-line pt-4 text-xs text-muted">
      <p className="truncate">{email}</p>
      <div className="flex gap-4">
        <Link to="/" className="hover:text-ink">
          Открыть сайт
        </Link>
        <button type="button" onClick={logout} className="hover:text-ink">
          Выйти
        </button>
      </div>
    </div>
  )

  return (
    <div className="min-h-dvh lg:grid lg:grid-cols-[240px_1fr]">
      <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b border-line bg-cream/95 px-4 backdrop-blur lg:hidden">
        <button type="button" onClick={() => setOpen(true)} aria-label="Меню" className="p-1">
          <MenuIcon />
        </button>
        <span className="font-serif text-xl">Админка</span>
        <span className="w-8" />
      </header>

      {open && (
        <div className="fixed inset-0 z-50 lg:hidden">
          <div className="absolute inset-0 bg-ink/40" onClick={() => setOpen(false)} />
          <aside className="absolute inset-y-0 left-0 flex w-72 flex-col bg-cream p-4">
            <div className="mb-4 flex items-center justify-between">
              <span className="font-serif text-2xl">12 Beauty Stories</span>
              <button type="button" onClick={() => setOpen(false)} aria-label="Закрыть">
                <CloseIcon />
              </button>
            </div>
            {nav}
            {footer}
          </aside>
        </div>
      )}

      <aside className="sticky top-0 hidden h-dvh flex-col border-r border-line p-4 lg:flex">
        <Link to="/admin" className="mb-6 px-4 font-serif text-2xl">
          12 Beauty Stories
        </Link>
        {nav}
        {footer}
      </aside>

      <main className="min-w-0 px-4 py-6 sm:px-8 sm:py-8">
        <Outlet />
      </main>
    </div>
  )
}
