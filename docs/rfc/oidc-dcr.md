# OpenID Connect Dynamic Client Registration 1.0

- OpenID Foundation · https://openid.net/specs/openid-connect-registration-1_0.html · Conf: OPT

## Назначение
OIDC-надстройка над RFC 7591 с OIDC-метаданными клиента (redirect_uris, application_type, sector_identifier, id_token_signed_response_alg).

## Где в API
- `POST /oauth2/register` (тот же endpoint, что у [RFC 7591](rfc7591.md))
- `registration_endpoint` в discovery-документе

## Conformance
- Принимаются OIDC-метаданные `application_type`, `jwks`, `jwks_uri`,
  `post_logout_redirect_uris`, `backchannel_logout_uri` — то есть ровно те, что
  влияют на поведение провайдера.
- `id_token_signed_response_alg` не настраивается: id_token подписывается RS256,
  как объявлено в discovery.
- `sector_identifier_uri` не поддерживается — subject тип только `public`.

## Gotchas
- Парный к 7591. Для admin-managed клиентов не обязателен.
