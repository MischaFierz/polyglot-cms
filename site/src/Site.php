<?php

declare(strict_types=1);

namespace Cms\Site;

/**
 * Front controller: maps URLs to content from the API and renders theme templates.
 */
final class Site
{
    /** @var array{settings: array<string, string>, nav: list<array{title: string, slug: string}>}|null */
    private ?array $site = null;

    public function __construct(
        private readonly ApiClient $api,
        private readonly string $themeDir,
        private readonly string $baseUrl,
    ) {
    }

    public function handle(string $uri): void
    {
        $path = '/' . trim((string) parse_url($uri, PHP_URL_PATH), '/');

        try {
            match (true) {
                $path === '/' => $this->showPage($this->setting('home_slug', 'home')),
                $path === '/blog' => $this->showBlog(max(1, (int) ($_GET['page'] ?? 1))),
                $path === '/feed.xml' => $this->showFeed(),
                $path === '/sitemap.xml' => $this->showSitemap(),
                (bool) preg_match('#^/blog/([a-z0-9-]+)$#', $path, $m) => $this->showPost($m[1]),
                (bool) preg_match('#^/([a-z0-9-]+)$#', $path, $m) => $this->showPage($m[1]),
                default => $this->notFound(),
            };
        } catch (\Throwable $e) {
            error_log((string) $e);
            http_response_code(503);
            $this->render('error', ['title' => 'Temporarily unavailable']);
        }
    }

    private function showPage(string $slug): void
    {
        $page = $this->api->get('/api/public/pages/' . rawurlencode($slug));
        if ($page === null || $page['type'] !== 'page') {
            $this->notFound();
            return;
        }
        // Visiting /home directly would duplicate the front page.
        if ($slug === $this->setting('home_slug', 'home') && $this->requestPath() !== '/') {
            $this->redirect('/');
            return;
        }
        $this->render($this->templateFor($page, 'page'), ['title' => $page['title'], 'page' => $page]);
    }

    private function showPost(string $slug): void
    {
        $post = $this->api->get('/api/public/pages/' . rawurlencode($slug));
        if ($post === null || $post['type'] !== 'post') {
            $this->notFound();
            return;
        }
        $this->render($this->templateFor($post, 'post'), ['title' => $post['title'], 'post' => $post]);
    }

    private function showBlog(int $page): void
    {
        $result = $this->api->get('/api/public/posts?page=' . $page);
        $this->render('blog', [
            'title' => 'Blog',
            'posts' => $result['posts'] ?? [],
            'currentPage' => $result['page'] ?? 1,
            'totalPages' => $result['total_pages'] ?? 1,
        ]);
    }

    private function showFeed(): void
    {
        $result = $this->api->get('/api/public/posts?limit=20');
        header('Content-Type: application/rss+xml; charset=utf-8');
        $this->renderBare('feed', ['posts' => $result['posts'] ?? []]);
    }

    private function showSitemap(): void
    {
        $entries = $this->api->get('/api/public/sitemap') ?? [];
        header('Content-Type: application/xml; charset=utf-8');
        $this->renderBare('sitemap', ['entries' => $entries]);
    }

    private function notFound(): void
    {
        http_response_code(404);
        $this->render('404', ['title' => 'Page not found']);
    }

    /**
     * A page may ask for a custom template (e.g. "landing"); fall back if the theme lacks it.
     *
     * @param array<string, mixed> $content
     */
    private function templateFor(array $content, string $fallback): string
    {
        $template = (string) ($content['template'] ?? 'default');
        if ($template !== 'default' && preg_match('/^[a-z0-9_-]+$/', $template)
            && is_file("{$this->themeDir}/{$template}.php")) {
            return $template;
        }
        return $fallback;
    }

    /**
     * Renders a template inside the theme layout.
     *
     * @param array<string, mixed> $vars
     */
    private function render(string $template, array $vars): void
    {
        $content = $this->capture($template, $vars);
        echo $this->capture('layout', $vars + ['content' => $content]);
    }

    /**
     * @param array<string, mixed> $vars
     */
    private function renderBare(string $template, array $vars): void
    {
        echo $this->capture($template, $vars);
    }

    /**
     * @param array<string, mixed> $vars
     */
    private function capture(string $template, array $vars): string
    {
        $site = $this->site();
        $vars += [
            'site' => $site['settings'],
            'nav' => $site['nav'],
            'baseUrl' => $this->baseUrl,
            'currentPath' => $this->requestPath(),
            'homeSlug' => $this->setting('home_slug', 'home'),
        ];
        $file = "{$this->themeDir}/{$template}.php";

        ob_start();
        (static function (string $__file, array $__vars): void {
            extract($__vars, EXTR_SKIP);
            require $__file;
        })($file, $vars);
        return (string) ob_get_clean();
    }

    /**
     * @return array{settings: array<string, string>, nav: list<array{title: string, slug: string}>}
     */
    private function site(): array
    {
        if ($this->site === null) {
            try {
                $data = $this->api->get('/api/public/site') ?? [];
            } catch (\Throwable) {
                $data = []; // still lets the error page render
            }
            $this->site = [
                'settings' => $data['settings'] ?? ['site_title' => 'Website'],
                'nav' => $data['nav'] ?? [],
            ];
        }
        return $this->site;
    }

    private function setting(string $key, string $default): string
    {
        return (string) ($this->site()['settings'][$key] ?? $default);
    }

    private function requestPath(): string
    {
        return '/' . trim((string) parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH), '/');
    }

    private function redirect(string $to): void
    {
        header('Location: ' . $to, true, 301);
    }
}
