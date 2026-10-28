import { Navigate, type RouteObject, useRoutes } from 'react-router-dom'
import { AdminLayout } from './AdminLayout'
import { BannersPage } from './pages/BannersPage'
import { BrandsPage } from './pages/BrandsPage'
import { CategoriesPage } from './pages/CategoriesPage'
import { LoginPage } from './pages/LoginPage'
import { OrderDetailPage } from './pages/OrderDetailPage'
import { OrdersPage } from './pages/OrdersPage'
import { ProductEditPage } from './pages/ProductEditPage'
import { ProductsPage } from './pages/ProductsPage'
import { PromosPage } from './pages/PromosPage'
import { RulesPage } from './pages/RulesPage'

const adminRoutes: RouteObject[] = [
  { path: 'login', element: <LoginPage /> },
  {
    element: <AdminLayout />,
    children: [
      { index: true, element: <Navigate to="orders" replace /> },
      { path: 'orders', element: <OrdersPage /> },
      { path: 'orders/:id', element: <OrderDetailPage /> },
      { path: 'products', element: <ProductsPage /> },
      { path: 'products/:id', element: <ProductEditPage /> },
      { path: 'brands', element: <BrandsPage /> },
      { path: 'categories', element: <CategoriesPage /> },
      { path: 'banners', element: <BannersPage /> },
      { path: 'promos', element: <PromosPage /> },
      { path: 'rules', element: <RulesPage /> },
      { path: '*', element: <Navigate to="orders" replace /> },
    ],
  },
]

/** Admin entry, mounted at /admin/* and loaded lazily so customers never download the back office. */
export function AdminRoot() {
  return useRoutes(adminRoutes)
}
