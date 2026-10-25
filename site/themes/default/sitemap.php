<?php
/**
 * @var list<array<string, mixed>> $entries
 * @var string $baseUrl
 * @var string $homeSlug
 */
echo '<?xml version="1.0" encoding="UTF-8"?>', "\n";
?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <url><loc><?= e($baseUrl) ?>/blog</loc></url>
    <?php foreach ($entries as $entry): ?>
    <url>
        <loc><?= e($baseUrl . content_url($entry, $homeSlug)) ?></loc>
        <lastmod><?= e(format_date($entry['updated_at'], 'Y-m-d')) ?></lastmod>
    </url>
    <?php endforeach ?>
</urlset>
