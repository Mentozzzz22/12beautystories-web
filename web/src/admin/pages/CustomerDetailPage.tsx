import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { WhatsAppIcon } from '../../components/icons'
import { formatTenge, plural } from '../../lib/format'
import { whatsappLink } from '../../lib/site'
import { adminApi } from '../api'
import { Badge, FormActions, Loading, TextArea, TextField, Toggle } from '../components/ui'
import { fieldErrors } from '../errors'
import { formatDate, formatDay, statusInfo } from '../orderStatus'
import type { Customer, CustomerUpdate } from '../types'

export function CustomerDetailPage() {
  const id = Number(useParams().id)
  const customer = useQuery({
    queryKey: ['admin', 'customer', id],
    queryFn: () => adminApi.customer(id),
  })

  if (customer.isPending) return <Loading />
  if (customer.isError) return <p className="text-sm text-sale">Клиент не найден</p>
  // Remount the form when another customer opens, so its state starts from fresh data.
  return <CustomerCard key={customer.data.id} customer={customer.data} />
}

function CustomerCard({ customer: c }: { customer: Customer }) {
  const qc = useQueryClient()
  const [form, setForm] = useState<CustomerUpdate>({
    name: c.name,
    note: c.note,
    marketingConsent: c.marketingConsent,
  })

  const save = useMutation({
    mutationFn: () => adminApi.updateCustomer(c.id, form),
    onSuccess: (updated) => {
      qc.setQueryData(['admin', 'customer', c.id], updated)
      void qc.invalidateQueries({ queryKey: ['admin', 'customers'] })
    },
  })
  const errors = fieldErrors(save.error)

  const set = <K extends keyof CustomerUpdate>(key: K, value: CustomerUpdate[K]) =>
    setForm((f) => ({ ...f, [key]: value }))

  const submit = (e: FormEvent) => {
    e.preventDefault()
    save.mutate()
  }

  const average = c.orderCount > 0 ? Math.round(c.totalSpent / c.orderCount) : 0

  return (
    <div className="max-w-3xl">
      <Link to="/admin/customers" className="text-sm text-muted hover:text-ink">
        ← Все клиенты
      </Link>
      <div className="mt-3 mb-6 flex flex-wrap items-center gap-3">
        <h1 className="text-3xl sm:text-4xl">{c.name}</h1>
        {c.marketingConsent && <Badge tone="green">согласие на рассылку</Badge>}
      </div>

      <dl className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Stat label="Заказов" value={String(c.orderCount)} />
        <Stat label="Сумма" value={formatTenge(c.totalSpent)} />
        <Stat label="Средний чек" value={average ? formatTenge(average) : '—'} />
        <Stat label="Последний заказ" value={c.lastOrderAt ? formatDay(c.lastOrderAt) : '—'} />
      </dl>

      <div className="grid gap-6 md:grid-cols-[1fr_280px]">
        <section className="rounded-2xl bg-white p-5">
          <h2 className="mb-3 text-xl">
            {c.orders.length} {plural(c.orders.length, ['заказ', 'заказа', 'заказов'])}
          </h2>
          <ul className="divide-y divide-line text-sm">
            {c.orders.map((o) => {
              const s = statusInfo(o.status)
              return (
                <li key={o.id}>
                  <Link
                    to={`/admin/orders/${o.id}`}
                    className="-mx-2 flex items-center justify-between gap-3 rounded-lg px-2 py-2.5 hover:bg-sand"
                  >
                    <span>
                      <span className="font-medium">{o.number}</span>
                      <span className="block text-xs text-muted">{formatDate(o.createdAt)}</span>
                    </span>
                    <span className="flex items-center gap-3">
                      <Badge tone={s.tone}>{s.label}</Badge>
                      <span className="w-24 text-right">{formatTenge(o.total)}</span>
                    </span>
                  </Link>
                </li>
              )
            })}
          </ul>
          <p className="mt-3 text-xs text-muted">
            Отменённые заказы не входят в сумму и количество.
          </p>
        </section>

        <aside className="h-fit space-y-4 rounded-2xl bg-white p-5">
          <a href={`tel:${c.phone}`} className="block text-sm text-muted hover:text-ink">
            {c.phone}
          </a>
          <a
            href={whatsappLink(c.phone, `Здравствуйте, ${c.name}! Это 12 Beauty Stories.`)}
            target="_blank"
            rel="noreferrer"
            className="btn-primary w-full bg-[#1f8f4e] hover:bg-[#17733e]"
          >
            <WhatsAppIcon width={18} height={18} /> Написать в WhatsApp
          </a>

          <form onSubmit={submit} className="space-y-4 border-t border-line pt-4">
            <TextField
              label="Имя"
              value={form.name}
              onChange={(v) => set('name', v)}
              error={errors.name}
            />
            <TextArea
              label="Заметка"
              hint="Видите только вы: тип волос и кожи, любимые бренды"
              value={form.note}
              onChange={(v) => set('note', v)}
              error={errors.note}
            />
            <Toggle
              label="Согласие на рассылку"
              hint={
                c.marketingConsentAt
                  ? `Изменено ${formatDate(c.marketingConsentAt)}`
                  : 'Выключите, если клиент попросил не присылать акции'
              }
              checked={form.marketingConsent}
              onChange={(v) => set('marketingConsent', v)}
            />
            <FormActions saving={save.isPending} error={save.error} />
            {save.isSuccess && !save.isPending && <p className="text-xs text-muted">Сохранено</p>}
          </form>
        </aside>
      </div>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-white p-4">
      <dt className="text-xs text-muted">{label}</dt>
      <dd className="mt-1 text-lg font-medium">{value}</dd>
    </div>
  )
}
