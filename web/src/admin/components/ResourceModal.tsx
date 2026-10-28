import type { UseMutationResult } from '@tanstack/react-query'
import { type FormEvent, type ReactNode, useState } from 'react'
import { fieldErrors } from '../errors'
import { FormActions, Modal } from './ui'

interface Props<T extends { id: number }> {
  title: string
  initial: T
  save: UseMutationResult<T, Error, T>
  remove: UseMutationResult<void, Error, number>
  onClose: () => void
  children: (
    value: T,
    set: <K extends keyof T>(key: K, v: T[K]) => void,
    errors: Record<string, string>,
  ) => ReactNode
}

/** Modal form for a simple resource: holds the draft, saves, deletes and shows field errors. */
export function ResourceModal<T extends { id: number }>({
  title,
  initial,
  save,
  remove,
  onClose,
  children,
}: Props<T>) {
  const [value, setValue] = useState<T>(initial)
  const set = <K extends keyof T>(key: K, v: T[K]) => setValue((cur) => ({ ...cur, [key]: v }))

  const submit = (e: FormEvent) => {
    e.preventDefault()
    save.mutate(value, { onSuccess: onClose })
  }

  return (
    <Modal open title={title} onClose={onClose}>
      <form onSubmit={submit} noValidate className="space-y-4">
        {children(value, set, fieldErrors(save.error))}
        <FormActions
          saving={save.isPending}
          error={save.error ?? remove.error}
          onDelete={value.id ? () => remove.mutate(value.id, { onSuccess: onClose }) : undefined}
          deleting={remove.isPending}
        />
      </form>
    </Modal>
  )
}
