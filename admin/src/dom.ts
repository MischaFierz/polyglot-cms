type Child = Node | string | number | null | undefined | false
type Attrs = Record<string, unknown>

/**
 * Tiny element builder. Text children are always inserted as text nodes, so
 * user content can never be interpreted as HTML.
 */
export function h<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  attrs: Attrs = {},
  ...children: (Child | Child[])[]
): HTMLElementTagNameMap[K] {
  const el = document.createElement(tag)
  for (const [key, value] of Object.entries(attrs)) {
    if (value == null || value === false) continue
    if (key.startsWith('on') && typeof value === 'function') {
      el.addEventListener(key.slice(2).toLowerCase(), value as EventListener)
    } else if (key === 'class') {
      el.className = String(value)
    } else if (key === 'value' || key === 'checked') {
      ;(el as unknown as Record<string, unknown>)[key] = value
    } else {
      el.setAttribute(key, value === true ? '' : String(value))
    }
  }
  for (const child of children.flat()) {
    if (child == null || child === false) continue
    el.append(child instanceof Node ? child : String(child))
  }
  return el
}

export function toast(message: string, kind: 'ok' | 'error' = 'ok'): void {
  const host = document.getElementById('toasts')
  if (!host) return
  const el = h('div', { class: `toast toast-${kind}`, role: kind === 'error' ? 'alert' : 'status' }, message)
  host.append(el)
  setTimeout(() => el.remove(), 4000)
}

export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

export function debounce<A extends unknown[]>(fn: (...args: A) => void, ms: number): (...args: A) => void {
  let timer: ReturnType<typeof setTimeout> | undefined
  return (...args: A) => {
    clearTimeout(timer)
    timer = setTimeout(() => fn(...args), ms)
  }
}

export function formatDate(iso: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / 1024 ** 2).toFixed(1)} MB`
}

/** Insert text at the textarea cursor and keep focus. */
export function insertAtCursor(area: HTMLTextAreaElement, text: string): void {
  const { selectionStart: start, selectionEnd: end, value } = area
  area.value = value.slice(0, start) + text + value.slice(end)
  area.selectionStart = area.selectionEnd = start + text.length
  area.focus()
  area.dispatchEvent(new Event('input'))
}
