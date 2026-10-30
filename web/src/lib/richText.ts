/**
 * Descriptions are plain text with three optional tags: <b>, <i> and <u>. Anything else stays literal
 * text, so the markup is parsed into segments and never injected as HTML.
 */
export interface RichSegment {
  text: string
  bold: boolean
  italic: boolean
  underline: boolean
}

const TAG = /<(\/?)([biu])>/g

export function parseRichText(markup: string): RichSegment[] {
  const segments: RichSegment[] = []
  const style = { bold: false, italic: false, underline: false }
  const push = (text: string) => {
    if (text) segments.push({ text, ...style })
  }

  let last = 0
  for (const match of markup.matchAll(TAG)) {
    push(markup.slice(last, match.index))
    last = match.index + match[0].length
    const on = match[1] === ''
    if (match[2] === 'b') style.bold = on
    else if (match[2] === 'i') style.italic = on
    else style.underline = on
  }
  push(markup.slice(last))
  return segments
}
