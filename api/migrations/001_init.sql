-- Core schema. Timestamps are RFC 3339 UTC strings.

CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    email         TEXT NOT NULL UNIQUE COLLATE NOCASE,
    name          TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'editor' CHECK (role IN ('admin', 'editor')),
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL
);

-- Pages and blog posts share one table; `type` tells them apart.
CREATE TABLE pages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    type         TEXT NOT NULL DEFAULT 'page' CHECK (type IN ('page', 'post')),
    title        TEXT NOT NULL,
    slug         TEXT NOT NULL UNIQUE,
    excerpt      TEXT NOT NULL DEFAULT '',
    body_md      TEXT NOT NULL DEFAULT '',
    body_html    TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    template     TEXT NOT NULL DEFAULT 'default',
    show_in_nav  INTEGER NOT NULL DEFAULT 0,
    nav_order    INTEGER NOT NULL DEFAULT 0,
    author_id    INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    published_at TEXT
);

CREATE INDEX idx_pages_listing ON pages (type, status, published_at DESC);
CREATE INDEX idx_pages_nav ON pages (show_in_nav, nav_order);

CREATE TABLE page_revisions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id    INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    body_md    TEXT NOT NULL,
    author_id  INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_revisions_page ON page_revisions (page_id, id DESC);

CREATE TABLE media (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    filename      TEXT NOT NULL UNIQUE,
    original_name TEXT NOT NULL,
    mime          TEXT NOT NULL,
    size          INTEGER NOT NULL,
    alt           TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT INTO settings (key, value) VALUES
    ('site_title', 'My Website'),
    ('tagline', 'Powered by Polyglot CMS'),
    ('footer', 'Made with Go, PHP, TypeScript and SQL.'),
    ('home_slug', 'home'),
    ('posts_per_page', '10');

-- body_html is left empty; the API renders it from Markdown on startup.
INSERT INTO pages (type, title, slug, body_md, status, show_in_nav, nav_order, published_at) VALUES
    ('page', 'Home', 'home',
     '# Welcome

This is your new website. Log into the [admin panel](http://localhost:8080/admin/) to edit this page.

- Write pages and blog posts in **Markdown**
- Upload images in the media library
- Every save keeps a revision',
     'published', 1, 0, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('page', 'About', 'about', 'Tell your visitors who you are.', 'published', 1, 10,
     strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ('post', 'Hello, world', 'hello-world',
     'Your first blog post. Edit or delete it in the admin panel.', 'published', 0, 0,
     strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));
