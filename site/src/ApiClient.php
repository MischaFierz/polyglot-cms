<?php

declare(strict_types=1);

namespace Cms\Site;

/**
 * Talks to the Go content API and caches responses on disk.
 * If the API is down, a stale cached copy is served instead of an error page.
 */
final class ApiClient
{
    public function __construct(
        private readonly string $baseUrl,
        private readonly string $cacheDir,
        private readonly int $ttl,
    ) {
        if (!is_dir($this->cacheDir)) {
            @mkdir($this->cacheDir, 0775, true);
        }
    }

    /**
     * @return array<string, mixed>|null null when the API answers 404
     */
    public function get(string $path): ?array
    {
        $cacheFile = $this->cacheDir . '/' . sha1($path) . '.json';
        if ($this->ttl > 0 && is_file($cacheFile) && filemtime($cacheFile) > time() - $this->ttl) {
            return $this->decode((string) file_get_contents($cacheFile));
        }

        [$status, $body] = $this->fetch($path);

        if ($status === 404) {
            return null;
        }
        if ($status !== 200) {
            if (is_file($cacheFile)) {
                return $this->decode((string) file_get_contents($cacheFile));
            }
            throw new \RuntimeException("Content API unavailable (HTTP {$status}) for {$path}");
        }

        if ($this->ttl > 0) {
            file_put_contents($cacheFile . '.tmp', $body, LOCK_EX);
            rename($cacheFile . '.tmp', $cacheFile);
        }
        return $this->decode($body);
    }

    /**
     * @return array{0: int, 1: string}
     */
    private function fetch(string $path): array
    {
        $context = stream_context_create([
            'http' => [
                'method' => 'GET',
                'timeout' => 5,
                'ignore_errors' => true,
                'header' => "Accept: application/json\r\n",
            ],
        ]);
        $body = @file_get_contents(rtrim($this->baseUrl, '/') . $path, false, $context);
        if ($body === false) {
            return [0, ''];
        }

        $status = 0;
        foreach ($http_response_header ?? [] as $line) {
            if (preg_match('#^HTTP/\S+\s+(\d{3})#', $line, $m)) {
                $status = (int) $m[1];
            }
        }
        return [$status, $body];
    }

    /**
     * @return array<string, mixed>
     */
    private function decode(string $json): array
    {
        $data = json_decode($json, true, 512, JSON_THROW_ON_ERROR);
        return is_array($data) ? $data : [];
    }
}
