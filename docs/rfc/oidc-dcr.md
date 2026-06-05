# OpenID Connect Dynamic Client Registration 1.0

- OpenID Foundation · https://openid.net/specs/openid-connect-registration-1_0.html · Conf: OPT

## Назначение
OIDC-надстройка над RFC 7591 с OIDC-метаданными клиента (redirect_uris, application_type, sector_identifier, id_token_signed_response_alg).

## Где в API
- §32 apps (если включён self-service DCR)

## Gotchas
- Парный к 7591. Для admin-managed клиентов не обязателен.
