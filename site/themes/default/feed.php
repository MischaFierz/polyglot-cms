<?php
/**
 * @var list<array<string, mixed>> $posts
 * @var array<string, string> $site
 * @var string $baseUrl
 */
echo '<?xml version="1.0" encoding="UTF-8"?>', "\n";
?>
<rss version="2.0">
<channel>
    <title><?= e($site['site_title'] ?? '') ?></title>
    <link><?= e($baseUrl) ?>/</link>
    <description><?= e($site['tagline'] ?? '') ?></description>
    <?php foreach ($posts as $post): ?>
    <item>
        <title><?= e($post['title']) ?></title>
        <link><?= e("{$baseUrl}/blog/{$post['slug']}") ?></link>
        <guid><?= e("{$baseUrl}/blog/{$post['slug']}") ?></guid>
        <pubDate><?= e(format_date($post['published_at'], DATE_RSS)) ?></pubDate>
        <description><?= e($post['excerpt']) ?></description>
    </item>
    <?php endforeach ?>
</channel>
</rss>
