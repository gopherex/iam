# RateLimit Header Fields for HTTP (IETF draft)

- IETF draft-ietf-httpapi-ratelimit-headers · https://datatracker.ietf.org/doc/draft-ietf-httpapi-ratelimit-headers/ · Conf: SHOULD

## Назначение
Стандартизованные заголовки лимита: `RateLimit` / `RateLimit-Policy` (старый стиль: `RateLimit-Limit/Remaining/Reset`).

## Где в API
- §37 rate-limit events, все throttled ручки

## Gotchas
- Draft — синтаксис эволюционирует; зафиксировать выбранную версию заголовков в схемах. Дополняет 429 (6585).
