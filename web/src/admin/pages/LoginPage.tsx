import { useMutation, useQueryClient } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { adminApi } from '../api'
import { TextField } from '../components/ui'

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const qc = useQueryClient()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const login = useMutation({
    mutationFn: () => adminApi.login(email, password),
    onSuccess: (user) => {
      qc.setQueryData(['admin', 'me'], user)
      const from = (location.state as { from?: string } | null)?.from
      navigate(from && from !== '/admin/login' ? from : '/admin/orders', { replace: true })
    },
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    login.mutate()
  }

  let error: string | null = null
  if (login.error instanceof ApiError) {
    error =
      login.error.code === 'invalid_credentials'
        ? 'Неверная почта или пароль'
        : login.error.code === 'too_many_attempts'
          ? 'Слишком много попыток. Подождите 15 минут'
          : 'Не удалось войти. Попробуйте ещё раз'
  }

  return (
    <div className="flex min-h-dvh items-center justify-center p-4">
      <form
        onSubmit={submit}
        className="w-full max-w-sm space-y-5 rounded-3xl bg-white p-8 shadow-sm"
      >
        <div className="text-center">
          <p className="font-serif text-3xl">12 Beauty Stories</p>
          <p className="mt-1 text-sm text-muted">Вход в админку</p>
        </div>
        <TextField
          label="Почта"
          type="email"
          autoComplete="username"
          value={email}
          onChange={setEmail}
          required
        />
        <TextField
          label="Пароль"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={setPassword}
          required
        />
        {error && (
          <p role="alert" className="text-sm text-sale">
            {error}
          </p>
        )}
        <button type="submit" className="btn-primary w-full" disabled={login.isPending}>
          {login.isPending ? 'Входим…' : 'Войти'}
        </button>
      </form>
    </div>
  )
}
