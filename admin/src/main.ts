import { session } from './api.js'
import { errorMessage, h } from './dom.js'
import { editorView, listView, loginView, mediaView, settingsView } from './views.js'

type Route = [RegExp, (match: RegExpMatchArray) => HTMLElement | Promise<HTMLElement>]

const routes: Route[] = [
  [/^#\/login$/, () => loginView()],
  [/^#\/pages$/, () => listView('page')],
  [/^#\/posts$/, () => listView('post')],
  [/^#\/new\/(page|post)$/, (m) => editorView({ type: m[1] as 'page' | 'post' })],
  [/^#\/edit\/(\d+)$/, (m) => editorView({ id: Number(m[1]) })],
  [/^#\/media$/, () => mediaView()],
  [/^#\/settings$/, () => settingsView()],
]

const app = document.getElementById('app')!
let renderId = 0

async function render(): Promise<void> {
  const hash = location.hash || '#/pages'
  if (!session.token && hash !== '#/login') {
    location.hash = '#/login'
    return
  }

  const route = routes.find(([pattern]) => pattern.test(hash))
  if (!route) {
    location.hash = '#/pages'
    return
  }

  const id = ++renderId
  try {
    const view = await route[1](hash.match(route[0])!)
    if (id === renderId) app.replaceChildren(view) // ignore stale navigations
  } catch (err) {
    if (id === renderId) app.replaceChildren(h('div', { class: 'fatal' }, h('h1', {}, 'Something went wrong'), h('p', {}, errorMessage(err))))
  }
}

window.addEventListener('hashchange', () => void render())
void render()
