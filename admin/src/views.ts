import { api, session, type Page, type PageInput, type PageType, type Settings } from './api.js'
import { debounce, errorMessage, formatBytes, formatDate, h, insertAtCursor, toast } from './dom.js'

const siteUrl = document.querySelector<HTMLMetaElement>('meta[name="cms-site-url"]')?.content ?? 'http://localhost:8000'

function go(hash: string): void {
  location.hash = hash
}

function publicUrl(page: Page, homeSlug = 'home'): string {
  if (page.type === 'post') return `${siteUrl}/blog/${page.slug}`
  return page.slug === homeSlug ? `${siteUrl}/` : `${siteUrl}/${page.slug}`
}

// ─── Shell ────────────────────────────────────────────────────────────────

export function shell(active: string, content: HTMLElement): HTMLElement {
  const user = session.user
  const link = (hash: string, label: string, key: string) =>
    h('a', { href: hash, 'aria-current': active === key ? 'page' : false }, label)

  return h('div', { class: 'shell' },
    h('aside', { class: 'sidebar' },
      h('div', { class: 'logo' }, 'Polyglot CMS'),
      h('nav', {},
        link('#/pages', 'Pages', 'page'),
        link('#/posts', 'Posts', 'post'),
        link('#/media', 'Media', 'media'),
        user?.role === 'admin' && link('#/settings', 'Settings', 'settings'),
      ),
      h('div', { class: 'sidebar-foot' },
        h('a', { href: siteUrl, target: '_blank', rel: 'noopener' }, 'View site ↗'),
        h('span', { class: 'muted' }, user?.name ?? ''),
        h('button', {
          class: 'link',
          onclick: async () => {
            await api.logout().catch(() => undefined)
            session.clear()
            go('#/login')
          },
        }, 'Log out'),
      ),
    ),
    h('main', { class: 'content' }, content),
  )
}

// ─── Login ────────────────────────────────────────────────────────────────

export function loginView(): HTMLElement {
  const email = h('input', { type: 'email', name: 'email', required: true, autocomplete: 'username', autofocus: true })
  const password = h('input', { type: 'password', name: 'password', required: true, autocomplete: 'current-password' })
  const error = h('p', { class: 'error', role: 'alert' })
  const submit = h('button', { type: 'submit', class: 'primary' }, 'Log in')

  const form = h('form', {
    class: 'login card',
    onsubmit: async (e: Event) => {
      e.preventDefault()
      submit.disabled = true
      error.textContent = ''
      try {
        const { token, user } = await api.login(email.value, password.value)
        session.save(token, user)
        go('#/pages')
      } catch (err) {
        error.textContent = errorMessage(err)
      } finally {
        submit.disabled = false
      }
    },
  },
    h('h1', {}, 'Sign in'),
    h('label', {}, 'Email', email),
    h('label', {}, 'Password', password),
    error,
    submit,
  )
  return h('div', { class: 'login-wrap' }, form)
}

// ─── Content list ─────────────────────────────────────────────────────────

export async function listView(type: PageType): Promise<HTMLElement> {
  const label = type === 'page' ? 'Pages' : 'Posts'
  const tbody = h('tbody')
  const search = h('input', { type: 'search', placeholder: `Search ${label.toLowerCase()}…`, 'aria-label': 'Search' })
  const settings = await api.getSettings().catch((): Settings => ({}))

  const load = async () => {
    const pages = await api.listPages({ type, q: search.value.trim() || undefined })
    tbody.replaceChildren(
      ...pages.map((p) => h('tr', {},
        h('td', {},
          h('a', { href: `#/edit/${p.id}`, class: 'strong' }, p.title),
          h('div', { class: 'muted small' }, `/${p.slug}`),
        ),
        h('td', {}, h('span', { class: `badge badge-${p.status}` }, p.status)),
        h('td', { class: 'muted small' }, formatDate(p.updated_at)),
        h('td', { class: 'actions' },
          p.status === 'published' &&
            h('a', { href: publicUrl(p, settings.home_slug), target: '_blank', rel: 'noopener' }, 'View'),
        ),
      )),
    )
    if (pages.length === 0) {
      tbody.append(h('tr', {}, h('td', { colspan: 4, class: 'empty' }, `No ${label.toLowerCase()} found.`)))
    }
  }
  search.addEventListener('input', debounce(() => void load().catch((e) => toast(errorMessage(e), 'error')), 250))
  await load()

  return shell(type, h('section', {},
    h('header', { class: 'toolbar' },
      h('h1', {}, label),
      search,
      h('a', { href: `#/new/${type}`, class: 'button primary' }, `New ${type}`),
    ),
    h('table', { class: 'table' },
      h('thead', {}, h('tr', {}, h('th', {}, 'Title'), h('th', {}, 'Status'), h('th', {}, 'Updated'), h('th', {}))),
      tbody,
    ),
  ))
}

// ─── Editor ───────────────────────────────────────────────────────────────

export async function editorView(opts: { id?: number; type?: PageType }): Promise<HTMLElement> {
  const existing = opts.id ? await api.getPage(opts.id) : null
  const type: PageType = existing?.type ?? opts.type ?? 'page'
  let slugTouched = Boolean(existing)

  const title = h('input', { type: 'text', class: 'title-input', placeholder: 'Title', value: existing?.title ?? '', required: true })
  const slug = h('input', { type: 'text', value: existing?.slug ?? '', placeholder: 'auto-from-title' })
  const excerpt = h('textarea', { rows: 3, value: existing?.excerpt ?? '' })
  const status = h('select', {},
    h('option', { value: 'draft' }, 'Draft'),
    h('option', { value: 'published' }, 'Published'),
  )
  status.value = existing?.status ?? 'draft'
  const template = h('input', { type: 'text', value: existing?.template ?? 'default', pattern: '[a-z0-9_-]+' })
  const showInNav = h('input', { type: 'checkbox', checked: existing?.show_in_nav ?? false })
  const navOrder = h('input', { type: 'number', value: String(existing?.nav_order ?? 0), step: 1 })
  const body = h('textarea', { class: 'markdown', spellcheck: true, value: existing?.body_md ?? '', placeholder: 'Write in Markdown…' })
  const preview = h('div', { class: 'preview prose' })
  const revisionList = h('ul', { class: 'revisions' })
  const saveButton = h('button', { type: 'submit', class: 'primary' }, 'Save')

  title.addEventListener('input', () => {
    if (!slugTouched) slug.value = slugify(title.value)
  })
  slug.addEventListener('input', () => { slugTouched = slug.value !== '' })

  const renderPreview = debounce(async () => {
    try {
      const { html } = await api.preview(body.value)
      preview.innerHTML = html // server-rendered Markdown with raw HTML stripped
    } catch (err) {
      preview.textContent = errorMessage(err)
    }
  }, 300)
  body.addEventListener('input', renderPreview)
  renderPreview()

  const loadRevisions = async (id: number) => {
    const revisions = await api.revisions(id)
    revisionList.replaceChildren(
      ...revisions.map((r) => h('li', {},
        h('button', {
          type: 'button',
          class: 'link',
          title: 'Load this version into the editor (save to keep it)',
          onclick: () => {
            title.value = r.title
            body.value = r.body_md
            renderPreview()
            toast('Revision loaded — save to restore it')
          },
        }, formatDate(r.created_at)),
        r.author && h('span', { class: 'muted small' }, ` · ${r.author}`),
      )),
    )
    if (revisions.length === 0) revisionList.append(h('li', { class: 'muted small' }, 'No earlier versions yet.'))
  }
  if (existing) void loadRevisions(existing.id)

  const fileInput = h('input', {
    type: 'file',
    accept: 'image/*,application/pdf,video/mp4',
    hidden: true,
    onchange: async () => {
      const file = fileInput.files?.[0]
      if (!file) return
      try {
        const media = await api.uploadMedia(file)
        const snippet = media.mime.startsWith('image/') ? `![${media.alt}](${media.url})` : `[${media.original_name}](${media.url})`
        insertAtCursor(body, snippet)
        toast('Uploaded')
      } catch (err) {
        toast(errorMessage(err), 'error')
      } finally {
        fileInput.value = ''
      }
    },
  })

  const collect = (): PageInput => ({
    type,
    title: title.value,
    slug: slug.value,
    excerpt: excerpt.value,
    body_md: body.value,
    status: status.value as PageInput['status'],
    template: template.value,
    show_in_nav: showInNav.checked,
    nav_order: Number(navOrder.value) || 0,
  })

  let currentId = existing?.id
  const save = async () => {
    saveButton.disabled = true
    try {
      const saved = currentId ? await api.updatePage(currentId, collect()) : await api.createPage(collect())
      slug.value = saved.slug
      slugTouched = true
      toast('Saved')
      if (!currentId) {
        currentId = saved.id
        history.replaceState(null, '', `#/edit/${saved.id}`)
      }
      void loadRevisions(saved.id)
    } catch (err) {
      toast(errorMessage(err), 'error')
    } finally {
      saveButton.disabled = false
    }
  }

  const onKey = (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault()
      void save()
    }
  }
  document.addEventListener('keydown', onKey)
  window.addEventListener('hashchange', () => document.removeEventListener('keydown', onKey), { once: true })

  const deleteButton = existing && h('button', {
    type: 'button',
    class: 'danger',
    onclick: async () => {
      if (!window.confirm(`Delete "${existing.title}"? This cannot be undone.`)) return
      try {
        await api.deletePage(existing.id)
        toast('Deleted')
        go(type === 'page' ? '#/pages' : '#/posts')
      } catch (err) {
        toast(errorMessage(err), 'error')
      }
    },
  }, 'Delete')

  const form = h('form', { class: 'editor', onsubmit: (e: Event) => { e.preventDefault(); void save() } },
    h('div', { class: 'editor-main' },
      title,
      h('div', { class: 'editor-toolbar' },
        h('button', { type: 'button', onclick: () => fileInput.click() }, 'Insert media'),
        h('span', { class: 'muted small' }, 'Markdown · Ctrl+S to save'),
      ),
      fileInput,
      h('div', { class: 'split' }, body, preview),
    ),
    h('aside', { class: 'editor-side card' },
      h('div', { class: 'row' }, saveButton, deleteButton),
      h('label', {}, 'Status', status),
      h('label', {}, 'Slug', slug),
      h('label', {}, 'Excerpt', excerpt),
      h('label', {}, 'Template', template),
      type === 'page' && h('label', { class: 'check' }, showInNav, 'Show in navigation'),
      type === 'page' && h('label', {}, 'Navigation order', navOrder),
      existing && h('p', { class: 'muted small' }, `Last updated ${formatDate(existing.updated_at)}`),
      existing && h('details', {}, h('summary', {}, 'Revisions'), revisionList),
    ),
  )

  return shell(type, form)
}

function slugify(value: string): string {
  return value
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

// ─── Media library ────────────────────────────────────────────────────────

export async function mediaView(): Promise<HTMLElement> {
  const grid = h('div', { class: 'media-grid' })

  const load = async () => {
    const items = await api.listMedia()
    grid.replaceChildren(...items.map((m) => {
      const markdown = m.mime.startsWith('image/') ? `![${m.alt}](${m.url})` : `[${m.original_name}](${m.url})`
      return h('figure', { class: 'card media-item' },
        m.mime.startsWith('image/')
          ? h('img', { src: m.url, alt: m.alt, loading: 'lazy' })
          : h('div', { class: 'file-icon' }, m.mime.split('/')[1]?.toUpperCase() ?? 'FILE'),
        h('figcaption', {},
          h('div', { class: 'truncate', title: m.original_name }, m.original_name),
          h('div', { class: 'muted small' }, formatBytes(m.size)),
          h('div', { class: 'row' },
            h('button', {
              type: 'button',
              onclick: async () => {
                await navigator.clipboard.writeText(markdown)
                toast('Markdown copied')
              },
            }, 'Copy'),
            h('button', {
              type: 'button',
              class: 'danger',
              onclick: async () => {
                if (!window.confirm(`Delete ${m.original_name}? Pages using it will show a broken link.`)) return
                await api.deleteMedia(m.id).then(load, (e) => toast(errorMessage(e), 'error'))
              },
            }, 'Delete'),
          ),
        ),
      )
    }))
    if (items.length === 0) grid.append(h('p', { class: 'empty' }, 'No files uploaded yet.'))
  }

  const upload = async (files: FileList | null) => {
    for (const file of Array.from(files ?? [])) {
      try {
        await api.uploadMedia(file)
      } catch (err) {
        toast(`${file.name}: ${errorMessage(err)}`, 'error')
      }
    }
    await load()
  }

  const input = h('input', { type: 'file', multiple: true, hidden: true, onchange: () => void upload(input.files) })
  const drop = h('div', {
    class: 'dropzone',
    ondragover: (e: DragEvent) => { e.preventDefault(); drop.classList.add('over') },
    ondragleave: () => drop.classList.remove('over'),
    ondrop: (e: DragEvent) => {
      e.preventDefault()
      drop.classList.remove('over')
      void upload(e.dataTransfer?.files ?? null)
    },
  }, 'Drop files here or ', h('button', { type: 'button', class: 'link', onclick: () => input.click() }, 'browse'), input)

  await load()
  return shell('media', h('section', {}, h('header', { class: 'toolbar' }, h('h1', {}, 'Media')), drop, grid))
}

// ─── Settings ─────────────────────────────────────────────────────────────

const settingFields: { key: string; label: string; type?: string }[] = [
  { key: 'site_title', label: 'Site title' },
  { key: 'tagline', label: 'Tagline' },
  { key: 'footer', label: 'Footer text' },
  { key: 'home_slug', label: 'Home page slug' },
  { key: 'posts_per_page', label: 'Posts per page', type: 'number' },
]

export async function settingsView(): Promise<HTMLElement> {
  const current = await api.getSettings()
  const inputs = new Map(settingFields.map((f) => [f.key, h('input', { type: f.type ?? 'text', value: current[f.key] ?? '' })]))

  const form = h('form', {
    class: 'card settings',
    onsubmit: async (e: Event) => {
      e.preventDefault()
      try {
        await api.saveSettings(Object.fromEntries([...inputs].map(([k, el]) => [k, el.value])))
        toast('Settings saved')
      } catch (err) {
        toast(errorMessage(err), 'error')
      }
    },
  },
    ...settingFields.map((f) => h('label', {}, f.label, inputs.get(f.key))),
    h('button', { type: 'submit', class: 'primary' }, 'Save settings'),
  )
  return shell('settings', h('section', {}, h('header', { class: 'toolbar' }, h('h1', {}, 'Settings')), form))
}
