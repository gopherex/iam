# OpenID Connect RP-Initiated Logout 1.0

- OpenID Foundation · https://openid.net/specs/openid-connect-rpinitiated-1_0.html · Conf: SHOULD

## Назначение
RP инициирует logout: `/oauth2/logout?id_token_hint&post_logout_redirect_uri&state`.

## Где в API
- §23 `/oauth2/logout`

## Conformance
- Валидировать `post_logout_redirect_uri` против зарегистрированных у клиента.
- Завершить OP-сессию, затем redirect.

## Gotchas
- Headless: подтверждение logout (если нужно) рендерит продукт; по умолчанию для first-party — без подтверждения.
- Для каскада на другие RP — back-channel logout.
