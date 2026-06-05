# OpenID Connect Core 1.0

- OpenID Foundation · https://openid.net/specs/openid-connect-core-1_0.html · Conf: MUST

## Назначение
Identity-слой над OAuth2: `id_token` (JWT), `nonce`, `acr/amr/auth_time`, `prompt`, `max_age`, response_type, claims, UserInfo.

## Где в API
- §23 провайдер-режим (`/oauth2/authorize`, id_token в `/oauth2/token`, `/oauth2/userinfo`)
- §23a interaction (login/consent — наш headless аналог hosted UI)

## Conformance
- response_type `code` (Authorization Code Flow). Hybrid/implicit — не реализуем (9700).
- id_token claim: `iss, sub, aud, exp, iat, nonce, auth_time, acr, amr, azp`.
- `prompt=none` → если нет сессии, ошибка `login_required` (без рендера UI — соответствует headless).
- `max_age` → форсить reauth/step-up через §23a login.

## Gotchas
- nonce обязателен для code flow в OIDC, проверяется клиентом.
- `prompt=login|consent` маппится на наш interaction (§23a): сервер всегда редиректит на UI продукта.
- acr/amr согласовать с §10 MFA AAL.
