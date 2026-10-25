<?php
/**
 * @var array<string, string> $site
 * @var list<array{title: string, slug: string}> $nav
 * @var string $title
 * @var string $content
 * @var string $currentPath
 * @var string $homeSlug
 * @var string $baseUrl
 */
$siteTitle = $site['site_title'] ?? 'Website';
?>
<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title><?= e($title === $siteTitle ? $siteTitle : "{$title} – {$siteTitle}") ?></title>
    <meta name="description" content="<?= e($description ?? ($site['tagline'] ?? '')) ?>">
    <link rel="alternate" type="application/rss+xml" title="<?= e($siteTitle) ?>" href="/feed.xml">
    <link rel="stylesheet" href="/assets/theme.css">
</head>
<body>
<header class="site-header">
    <div class="wrap">
        <a class="brand" href="/">
            <strong><?= e($siteTitle) ?></strong>
            <?php if (!empty($site['tagline'])): ?><span><?= e($site['tagline']) ?></span><?php endif ?>
        </a>
        <nav aria-label="Main">
            <?php foreach ($nav as $item):
                $href = content_url($item + ['type' => 'page'], $homeSlug); ?>
                <a href="<?= e($href) ?>"<?= $href === $currentPath ? ' aria-current="page"' : '' ?>><?= e($item['title']) ?></a>
            <?php endforeach ?>
            <a href="/blog"<?= str_starts_with($currentPath, '/blog') ? ' aria-current="page"' : '' ?>>Blog</a>
        </nav>
    </div>
</header>

<main class="wrap">
    <?= $content ?>
</main>

<footer class="site-footer">
    <div class="wrap"><?= e($site['footer'] ?? '') ?></div>
</footer>
</body>
</html>
