# Polyglot CMS

A CMS for websites, where each part uses the language that suits it best.

| Part | Language | Why |
|------|----------|-----|
| `api/` – content API, auth, media, Markdown | **Go** | Fast, safe concurrency, single static binary, excellent stdlib HTTP server |
| `api/migrations/` – schema | **SQL** (SQLite) | Zero-ops database, one file to back up, plenty for most websites |
| `site/` – public website & themes | **PHP** | The best templating story for websites: themes are plain `.php` files any web developer can edit, and it runs on every host |
| `admin/` – editor UI | **TypeScript** | Type-safe UI code against the API's types, no framework or bundler needed |

```
 Browser ──► site/ (PHP, :8000) ──► api/ (Go, :8080) ──► SQLite
 Editor  ──► admin/ (TypeScript, served by the API at :8080/admin/)
```

## Features

- Pages and blog posts written in Markdown (GitHub-flavoured), with live preview
- Draft / published workflow, custom slugs, excerpts
- Revision history (last 50 versions per page), restore with one click
- Navigation menu built from pages, with ordering
- Media library with drag-and-drop uploads (file type is checked from the content, not the name)
- Per-page theme templates (`template: landing` → `site/themes/default/landing.php`)
- RSS feed (`/feed.xml`) and sitemap (`/sitemap.xml`)
- Admin and editor roles; bearer-token sessions; passwords hashed with bcrypt
- The website keeps serving cached content if the API goes down

## Install with Composer

```sh
composer create-project mischafierz/polyglot-cms my-site
cd my-site
composer build   # compile the admin panel, fetch the API's Go modules
composer api     # terminal 1: API + admin on :8080
composer serve   # terminal 2: website on :8000
```

## Run with Docker (easiest)

```sh
docker compose up --build
```

- Website: http://localhost:8000
- Admin: http://localhost:8080/admin/ (`admin@example.com` / `admin`)

Set `CMS_ADMIN_PASSWORD` before the first start to pick a different password.

## Run without Docker

You need Go ≥ 1.22, PHP ≥ 8.1 and Node ≥ 18 (for the TypeScript compiler). Install on Windows with:

```powershell
winget install GoLang.Go PHP.PHP.8.3 OpenJS.NodeJS.LTS
```

Then, in three terminals:

```sh
# 1. Build the admin panel
cd admin && npm install && npm run build      # or: npm run watch

# 2. Start the API (creates cms.db and the admin user on first run)
cd api && go mod tidy && go run .

# 3. Start the website
php -S localhost:8000 -t site/public site/public/index.php
```

## Configuration

| Variable | Default | Used by |
|----------|---------|---------|
| `CMS_ADDR` | `:8080` | API |
| `CMS_DB` | `cms.db` | API |
| `CMS_UPLOAD_DIR` | `../site/public/uploads` | API |
| `CMS_ADMIN_DIR` | `../admin` | API |
| `CMS_ADMIN_EMAIL` / `CMS_ADMIN_PASSWORD` | `admin@example.com` / `admin` | API, first start only |
| `CMS_API_URL` | `http://localhost:8080` | site |
| `CMS_SITE_URL` | `http://localhost:8000` | site (feed & sitemap links) |
| `CMS_CACHE_TTL` | `15` (seconds, `0` disables) | site |
| `CMS_THEME` | `default` | site |

## Making a theme

Copy `site/themes/default` to `site/themes/mytheme` and set `CMS_THEME=mytheme`.
Each template gets `$site` (settings), `$nav`, and `$page` / `$post` / `$posts`.
Escape everything with `e()`. `body_html` is already safe: the API strips raw HTML from Markdown.

## API

Public (no auth): `GET /api/public/site`, `/api/public/pages/{slug}`, `/api/public/posts?page=`, `/api/public/sitemap`

Authenticated (`Authorization: Bearer <token>` from `POST /api/auth/login`):
`/api/pages` (GET, POST), `/api/pages/{id}` (GET, PUT, DELETE), `/api/pages/{id}/revisions`,
`/api/preview`, `/api/media` (GET, POST multipart, DELETE `/{id}`), `/api/settings` (GET, PUT – admin only).

## Before going live

- Change the admin password, and put both services behind HTTPS.
- Login has no rate limit yet, so add one at your reverse proxy.
- Back up `cms.db` and the uploads directory.
