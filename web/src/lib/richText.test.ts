import { describe, expect, it } from 'vitest'
import { parseRichText } from './richText'

const plain = { bold: false, italic: false, underline: false }

describe('parseRichText', () => {
  it('keeps plain text as one segment', () => {
    expect(parseRichText('Шампунь\nдля волос')).toEqual([{ text: 'Шампунь\nдля волос', ...plain }])
  })

  it('applies nested styles', () => {
    expect(parseRichText('a <b>b <i>c</i></b> <u>d</u>')).toEqual([
      { text: 'a ', ...plain },
      { text: 'b ', ...plain, bold: true },
      { text: 'c', ...plain, bold: true, italic: true },
      { text: ' ', ...plain },
      { text: 'd', ...plain, underline: true },
    ])
  })

  it('leaves unknown tags and comparisons as literal text', () => {
    const text = 'pH<5 <script>alert(1)</script>'
    expect(parseRichText(text)).toEqual([{ text, ...plain }])
  })

  it('returns nothing for an empty string', () => {
    expect(parseRichText('')).toEqual([])
  })
})
