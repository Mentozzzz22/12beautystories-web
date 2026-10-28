import { Link } from 'react-router-dom'
import { routes } from '../lib/site'

export function NotFoundPage() {
  return (
    <div className="container-page py-20 text-center">
      <p className="font-serif text-6xl">404</p>
      <p className="mt-4 text-muted">Такой страницы нет</p>
      <Link to={routes.catalog} className="btn-primary mt-8">
        Перейти в каталог
      </Link>
    </div>
  )
}
