import { parseRichText } from '../../lib/richText'

const TAGS: Record<string, string> = { B: 'b', STRONG: 'b', I: 'i', EM: 'i', U: 'u' }
const BLOCKS = new Set(['DIV', 'P'])

/** Serializes the editor DOM to description markup, dropping every other tag the browser produced. */
export function domToMarkup(root: Node): string {
  let out = ''
  const walk = (node: Node) => {
    if (node.nodeType === Node.TEXT_NODE) {
      out += (node.textContent ?? '').replaceAll(' ', ' ')
      return
    }
    if (!(node instanceof HTMLElement)) return
    if (node.tagName === 'BR') {
      out += '\n'
      return
    }
    if (BLOCKS.has(node.tagName) && out !== '' && !out.endsWith('\n')) out += '\n'
    const tag = TAGS[node.tagName]
    if (tag) out += `<${tag}>`
    node.childNodes.forEach(walk)
    if (tag) out += `</${tag}>`
  }
  root.childNodes.forEach(walk)
  return out.replaceAll(/<([biu])><\/\1>/g, '').trimEnd()
}

export function fillEditor(root: HTMLElement, markup: string) {
  root.replaceChildren(
    ...parseRichText(markup).map((segment) => {
      let node: Node = document.createTextNode(segment.text)
      const wrap = (tag: string) => {
        const element = document.createElement(tag)
        element.append(node)
        node = element
      }
      if (segment.underline) wrap('u')
      if (segment.italic) wrap('i')
      if (segment.bold) wrap('b')
      return node
    }),
  )
}
