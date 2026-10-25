<?php

declare(strict_types=1);

// With `php -S`, let the built-in server deliver static files (CSS, uploads) itself.
if (PHP_SAPI === 'cli-server') {
    $file = __DIR__ . parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
    if (is_file($file)) {
        return false;
    }
}

// Installed with Composer: use its autoloader. Otherwise (e.g. the Docker image) load the classes directly.
if (is_file(__DIR__ . '/../../vendor/autoload.php')) {
    require __DIR__ . '/../../vendor/autoload.php';
} else {
    require __DIR__ . '/../src/helpers.php';
    require __DIR__ . '/../src/ApiClient.php';
    require __DIR__ . '/../src/Site.php';
}

$root = dirname(__DIR__);
$theme = preg_replace('/[^a-z0-9_-]/', '', getenv('CMS_THEME') ?: 'default');

$site = new Cms\Site\Site(
    new Cms\Site\ApiClient(
        getenv('CMS_API_URL') ?: 'http://localhost:8080',
        $root . '/cache',
        (int) (getenv('CMS_CACHE_TTL') ?: 15),
    ),
    $root . '/themes/' . $theme,
    rtrim(getenv('CMS_SITE_URL') ?: 'http://localhost:8000', '/'),
);
$site->handle($_SERVER['REQUEST_URI'] ?? '/');
