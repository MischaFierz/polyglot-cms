<?php
/**
 * Example custom template: set a page's template to "landing" in the admin panel.
 *
 * @var array<string, mixed> $page
 */
?>
<section class="hero">
    <h1><?= e($page['title']) ?></h1>
    <?php if ($page['excerpt'] !== ''): ?><p class="lead"><?= e($page['excerpt']) ?></p><?php endif ?>
</section>
<article class="prose">
    <?= $page['body_html'] ?>
</article>
