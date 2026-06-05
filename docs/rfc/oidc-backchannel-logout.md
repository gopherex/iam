# OpenID Connect Back-Channel Logout 1.0

- OpenID Foundation · https://openid.net/specs/openid-connect-backchannel-1_0.html · Conf: SHOULD

## Назначение
OP рассылает `logout_token` (JWT) напрямую на backchannel_logout_uri каждого RP — синхронный logout без браузера. Идеален для headless.

## Где в API
- §23b `/oauth2/backchannel-logout`

## Conformance
- logout_token claim: `iss, aud, iat, jti, events, sid/sub`. `nonce` ЗАПРЕЩЁН.
- RP регистрирует `backchannel_logout_uri`, `backchannel_logout_session_required`.

## Gotchas
- Предпочтительнее front-channel для headless (не нужны скрытые iframe).
- При sign-out-all (§3) — рассылать logout_token всем RP с активной сессией.
