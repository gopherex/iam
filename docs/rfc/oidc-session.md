# OpenID Connect Session Management 1.0

- OpenID Foundation · https://openid.net/specs/openid-connect-session-1_0.html · Conf: OPT

## Назначение
`check_session_iframe`, `session_state` — RP мониторит состояние OP-сессии через postMessage.

## Где в API
- §13 sessions, §23 OIDC

## Gotchas
- Браузер-iframe механика, конфликтует с headless/cookie-strict. Для SPA с активным мониторингом; иначе полагаться на token TTL + introspection.
