# OpenID Connect Discovery 1.0

- OpenID Foundation · https://openid.net/specs/openid-connect-discovery-1_0.html · Conf: MUST

## Назначение
`/.well-known/openid-configuration` — метадата OP (надмножество RFC 8414): + `userinfo_endpoint, id_token_signing_alg_values_supported, claims_supported, subject_types_supported`.

## Где в API
- §2 `/.well-known/openid-configuration`

## Conformance
- `issuer` точно = base URL, совпадает с `iss`.
- Список scopes/claims/alg реально поддерживаемых.

## Gotchas
- Кешируется клиентами — менять осторожно, держать стабильным.
