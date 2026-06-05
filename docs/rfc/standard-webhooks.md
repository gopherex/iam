# Standard Webhooks (HMAC signing spec)

- standardwebhooks.com · https://www.standardwebhooks.com/ · Conf: SHOULD (или собственный HMAC)

## Назначение
Конвенция подписи вебхуков: `webhook-id`, `webhook-timestamp`, `webhook-signature` (HMAC-SHA256), защита от replay.

## Где в API
- §27 webhooks (signing_secret, deliveries), §36 hooks (blocking)

## Conformance
- Подпись `HMAC(secret, "{id}.{timestamp}.{body}")`. Получатель сверяет + проверяет timestamp-окно (±5 мин).
- Версионирование секрета при rotate (§27 rotate-secret) — поддержать оба на период.

## Gotchas
- Blocking hooks (§36) — те же подписи + строгий timeout; при таймауте политика fail-open/closed документируется per-hook.
