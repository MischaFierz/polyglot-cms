export type Role = 'admin' | 'editor'
export type PageType = 'page' | 'post'
export type Status = 'draft' | 'published'

export interface User {
  id: number
  email: string
  name: string
  role: Role
}

export interface Page {
  id: number
  type: PageType
  title: string
  slug: string
  excerpt: string
  body_md?: string
  body_html?: string
  status: Status
  template: string
  show_in_nav: boolean
  nav_order: number
  author: string
  created_at: string
  updated_at: string
  published_at: string | null
}

export type PageInput = Pick<Page, 'type' | 'title' | 'slug' | 'excerpt' | 'status' | 'template' | 'show_in_nav' | 'nav_order'> & {
  body_md: string
}

export interface Revision {
  id: number
  title: string
  body_md: string
  author: string
  created_at: string
}

export interface Media {
  id: number
  url: string
  original_name: string
  mime: string
  size: number
  alt: string
  created_at: string
}

export type Settings = Record<string, string>

export class ApiError extends Error {
  constructor(public readonly status: number, message: string) {
    super(message)
  }
}

const TOKEN_KEY = 'cms_token'
const USER_KEY = 'cms_user'

export const session = {
  get token(): string | null {
    try { return localStorage.getItem(TOKEN_KEY) } catch { return null }
  },
  get user(): User | null {
    try { return JSON.parse(localStorage.getItem(USER_KEY) ?? 'null') as User | null } catch { return null }
  },
  save(token: string, user: User): void {
    try {
      localStorage.setItem(TOKEN_KEY, token)
      localStorage.setItem(USER_KEY, JSON.stringify(user))
    } catch { /* storage unavailable: session lasts until reload */ }
  },
  clear(): void {
    try {
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(USER_KEY)
    } catch { /* ignore */ }
  },
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  const token = session.token
  if (token) headers.Authorization = `Bearer ${token}`

  let payload: BodyInit | undefined
  if (body instanceof FormData) {
    payload = body
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }

  const res = await fetch(`/api${path}`, { method, headers, body: payload })
  const data: unknown = res.status === 204 ? null : await res.json().catch(() => null)

  if (res.status === 401 && token) {
    session.clear()
    location.hash = '#/login'
  }
  if (!res.ok) {
    const message = (data as { error?: string } | null)?.error ?? res.statusText
    throw new ApiError(res.status, message)
  }
  return data as T
}

export const api = {
  login: (email: string, password: string) =>
    request<{ token: string; user: User }>('POST', '/auth/login', { email, password }),
  logout: () => request<null>('POST', '/auth/logout'),

  listPages: (filter: { type?: PageType; status?: Status; q?: string } = {}) => {
    const params = new URLSearchParams(Object.entries(filter).filter(([, v]) => v) as [string, string][])
    return request<Page[]>('GET', `/pages?${params}`)
  },
  getPage: (id: number) => request<Page>('GET', `/pages/${id}`),
  createPage: (input: PageInput) => request<Page>('POST', '/pages', input),
  updatePage: (id: number, input: PageInput) => request<Page>('PUT', `/pages/${id}`, input),
  deletePage: (id: number) => request<null>('DELETE', `/pages/${id}`),
  revisions: (id: number) => request<Revision[]>('GET', `/pages/${id}/revisions`),
  preview: (body_md: string) => request<{ html: string }>('POST', '/preview', { body_md }),

  listMedia: () => request<Media[]>('GET', '/media'),
  uploadMedia: (file: File, alt = '') => {
    const form = new FormData()
    form.append('file', file)
    form.append('alt', alt)
    return request<Media>('POST', '/media', form)
  },
  deleteMedia: (id: number) => request<null>('DELETE', `/media/${id}`),

  getSettings: () => request<Settings>('GET', '/settings'),
  saveSettings: (settings: Settings) => request<Settings>('PUT', '/settings', settings),
}
