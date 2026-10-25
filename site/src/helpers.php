<?php

declare(strict_types=1);

/** Escape for HTML output. Use for every value that is not trusted HTML. */
function e(mixed $value): string
{
    return htmlspecialchars((string) $value, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
}

/** Format an RFC 3339 timestamp from the API, e.g. "24 October 2026". */
function format_date(?string $iso, string $format = 'j F Y'): string
{
    if ($iso === null || $iso === '') {
        return '';
    }
    try {
        return (new DateTimeImmutable($iso))->format($format);
    } catch (Exception) {
        return '';
    }
}

/** URL for a page or post, honouring the configured home page. */
function content_url(array $item, string $homeSlug): string
{
    if (($item['type'] ?? 'page') === 'post') {
        return '/blog/' . $item['slug'];
    }
    return $item['slug'] === $homeSlug ? '/' : '/' . $item['slug'];
}
