import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { RichTextArea } from './RichTextArea'
import { domToMarkup } from './richTextDom'

function fromHtml(html: string) {
  const root = document.createElement('div')
  root.innerHTML = html
  return root
}

describe('domToMarkup', () => {
  it.each([
    ['plain', 'plain'],
    ['a <b>b</b> <i>c</i> <u>d</u>', 'a <b>b</b> <i>c</i> <u>d</u>'],
    ['<strong>b</strong> <em>i</em>', '<b>b</b> <i>i</i>'],
    ['<b><i>both</i></b>', '<b><i>both</i></b>'],
    ['one<div>two</div><div><br></div><div>four</div>', 'one\ntwo\n\nfour'],
    ['<div>one<br></div><div>two</div><div><br></div>', 'one\ntwo'],
    ['<span style="color:red">a</span><script>x</script>', 'ax'],
    ['a&nbsp;b<b></b>', 'a b'],
  ])('%s → %s', (html, expected) => {
    expect(domToMarkup(fromHtml(html))).toBe(expected)
  })
})

describe('RichTextArea', () => {
  it('shows saved markup as formatted text without running it as HTML', () => {
    render(<RichTextArea label="Описание" value="<b>жирный</b> <img src=x>" onChange={vi.fn()} />)

    const editor = screen.getByRole('textbox', { name: 'Описание' })
    expect(editor.querySelector('b')).toHaveTextContent('жирный')
    expect(editor.querySelector('img')).toBeNull()
    expect(editor).toHaveTextContent('жирный <img src=x>')
  })
})
