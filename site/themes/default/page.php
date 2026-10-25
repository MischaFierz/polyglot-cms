<?php
/** @var array<string, mixed> $page */
?>
<article class="prose">
    <?php if (!str_starts_with(ltrim($page['body_html']), '<h1')): ?>
        <h1><?= e($page['title']) ?></h1>
    <?php endif ?>
    <?php // body_html is rendered by the API from Markdown with raw HTML stripped. ?>
    <?= $page['body_html'] ?>
</article>
