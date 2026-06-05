# OpenID Connect Front-Channel Logout 1.0

- OpenID Foundation · https://openid.net/specs/openid-connect-frontchannel-1_0.html · Conf: OPT

## Назначение
Logout через скрытые iframe на frontchannel_logout_uri RP.

## Где в API
- §23 logout (опциональная альтернатива back-channel)

## Gotchas
- Требует браузерных iframe → плохо ложится на headless/SSR. Предпочитать back-channel.
