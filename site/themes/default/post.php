<?php
/** @var array<string, mixed> $post */
?>
<article class="prose">
    <header class="post-header">
        <h1><?= e($post['title']) ?></h1>
        <p class="meta">
            <time datetime="<?= e($post['published_at']) ?>"><?= e(format_date($post['published_at'])) ?></time>
            <?php if ($post['author'] !== ''): ?> · <?= e($post['author']) ?><?php endif ?>
        </p>
    </header>
    <?= $post['body_html'] ?>
</article>
<p><a href="/blog">← All posts</a></p>
