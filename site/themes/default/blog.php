<?php
/**
 * @var list<array<string, mixed>> $posts
 * @var int $currentPage
 * @var int $totalPages
 */
?>
<h1>Blog</h1>

<?php if ($posts === []): ?>
    <p>No posts yet.</p>
<?php endif ?>

<ul class="post-list">
    <?php foreach ($posts as $post): ?>
        <li>
            <a href="/blog/<?= e($post['slug']) ?>"><h2><?= e($post['title']) ?></h2></a>
            <p class="meta"><?= e(format_date($post['published_at'])) ?></p>
            <?php if ($post['excerpt'] !== ''): ?><p><?= e($post['excerpt']) ?></p><?php endif ?>
        </li>
    <?php endforeach ?>
</ul>

<?php if ($totalPages > 1): ?>
    <nav class="pager" aria-label="Pagination">
        <?php if ($currentPage > 1): ?><a href="/blog?page=<?= $currentPage - 1 ?>">← Newer</a><?php endif ?>
        <span>Page <?= $currentPage ?> of <?= $totalPages ?></span>
        <?php if ($currentPage < $totalPages): ?><a href="/blog?page=<?= $currentPage + 1 ?>">Older →</a><?php endif ?>
    </nav>
<?php endif ?>
