# RFC 6265bis — HTTP Cookies (обновление 6265)

- IETF draft-ietf-httpbis-rfc6265bis · https://datatracker.ietf.org/doc/draft-ietf-httpbis-rfc6265bis/ · Conf: MUST

## Назначение
Cookie-семантика: `HttpOnly`, `Secure`, `SameSite=Lax/Strict/None`, `__Host-`/`__Secure-` префиксы.

## Где в API
- §0 cookie-mode (session/refresh/CSRF cookies)

## Conformance
- session/refresh cookie: `HttpOnly; Secure; SameSite=Lax` (или Strict). Префикс `__Host-`.
- CSRF cookie: читаемая (не HttpOnly), double-submit с `X-CSRF-Token` (§0).

## Gotchas
- SameSite=None (кросс-сайт SPA) → обязательно Secure + усиленный CSRF. OAuth-redirect совместимость с SameSite — проверять callback'и.
