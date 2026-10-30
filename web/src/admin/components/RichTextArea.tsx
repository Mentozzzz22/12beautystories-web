import { type ClipboardEvent, useEffect, useRef } from 'react'
import { domToMarkup, fillEditor } from './richTextDom'
import { Field } from './ui'

const BUTTONS = [
  { command: 'bold', label: 'Жирный', icon: 'Ж', className: 'font-bold' },
  { command: 'italic', label: 'Курсив', icon: 'К', className: 'italic' },
  { command: 'underline', label: 'Подчёркнутый', icon: 'Ч', className: 'underline' },
]

/** Description field with bold, italic and underline; execCommand is still the only native way to edit a selection. */
export function RichTextArea({
  label,
  value,
  onChange,
  error,
  hint,
  rows = 4,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  error?: string
  hint?: string
  rows?: number
}) {
  const editor = useRef<HTMLDivElement>(null)
  const current = useRef<string | null>(null)

  useEffect(() => {
    if (editor.current && value !== current.current) {
      fillEditor(editor.current, value)
      current.current = value
    }
  }, [value])

  const emit = () => {
    if (!editor.current) return
    current.current = domToMarkup(editor.current)
    onChange(current.current)
  }

  const onPaste = (e: ClipboardEvent) => {
    // Formatting copied from other sites and documents must not reach the description.
    e.preventDefault()
    document.execCommand('insertText', false, e.clipboardData.getData('text/plain'))
  }

  return (
    <Field label={label} error={error} hint={hint}>
      {(id) => (
        <>
          <div className="mb-1.5 flex gap-1">
            {BUTTONS.map((b) => (
              <button
                key={b.command}
                type="button"
                title={b.label}
                aria-label={b.label}
                className={`h-8 w-8 rounded-lg border border-line bg-white text-sm hover:border-ink ${b.className}`}
                // Keeps the text selection, which a click on a button would otherwise clear.
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => {
                  document.execCommand(b.command)
                  emit()
                }}
              >
                {b.icon}
              </button>
            ))}
          </div>
          <div
            id={id}
            ref={editor}
            role="textbox"
            aria-multiline="true"
            aria-label={label}
            contentEditable
            suppressContentEditableWarning
            style={{ minHeight: `${rows * 1.5 + 1.5}rem` }}
            className={`input resize-y overflow-auto whitespace-pre-wrap ${error ? 'border-sale' : ''}`}
            onInput={emit}
            onPaste={onPaste}
          />
        </>
      )}
    </Field>
  )
}
