import { useState } from 'react'

/** Tracks which item is open in a ResourceModal; resets mutation errors between openings. */
export function useEditing<T>(reset: () => void) {
  const [editing, setEditing] = useState<T | null>(null)
  return {
    editing,
    open: (item: T) => {
      reset()
      setEditing(item)
    },
    close: () => setEditing(null),
  }
}
