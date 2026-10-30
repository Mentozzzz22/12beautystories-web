import type { ReactNode } from 'react'
import { parseRichText } from '../lib/richText'

/** Renders a description with its bold, italic and underlined parts. */
export function RichText({ text }: { text: string }) {
  return parseRichText(text).map((segment, i) => {
    let node: ReactNode = segment.text
    if (segment.underline) node = <u>{node}</u>
    if (segment.italic) node = <em>{node}</em>
    if (segment.bold) node = <strong className="font-semibold">{node}</strong>
    return <span key={i}>{node}</span>
  })
}
