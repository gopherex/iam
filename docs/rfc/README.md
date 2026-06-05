# RFC / Standards index

Все стандарты, на которых стоит headless auth server (`docs/idea.md`). Один файл = один стандарт. Формат файла: назначение, где в API, conformance, gotchas. Текст самих RFC не копируем (copyright) — только ссылки + как применяем.

`Conf` колонка: **MUST** = ядро, без него фича не работает / небезопасна; **SHOULD** = сильно рекомендовано; **OPT** = опционально (enterprise / hardening).

## OAuth 2.0 / провайдер-режим

| Стандарт | Файл | Conf | Секции idea.md |
| --- | --- | --- | --- |
| RFC 6749 OAuth 2.0 Framework | [rfc6749.md](rfc6749.md) | MUST | 23, 8, 25 |
| RFC 6750 Bearer Token Usage | [rfc6750.md](rfc6750.md) | MUST | 0, все Bearer |
| RFC 7636 PKCE | [rfc7636.md](rfc7636.md) | MUST | 3, 8, 23 |
| RFC 8252 OAuth for Native Apps | [rfc8252.md](rfc8252.md) | SHOULD | 32, 8 |
| RFC 8628 Device Authorization Grant | [rfc8628.md](rfc8628.md) | MUST | 23, 24 |
| RFC 7009 Token Revocation | [rfc7009.md](rfc7009.md) | MUST | 23, 22 |
| RFC 7662 Token Introspection | [rfc7662.md](rfc7662.md) | MUST | 23, 22 |
| RFC 8414 AS Metadata | [rfc8414.md](rfc8414.md) | MUST | 2 |
| RFC 9126 PAR | [rfc9126.md](rfc9126.md) | OPT | 23 |
| RFC 9207 Issuer Identification | [rfc9207.md](rfc9207.md) | SHOULD | 23 |
| RFC 8693 Token Exchange | [rfc8693.md](rfc8693.md) | SHOULD | 22, 25 |
| RFC 7523 JWT Bearer (client auth/grant) | [rfc7523.md](rfc7523.md) | SHOULD | 25, 23 |
| RFC 7591 Dynamic Client Registration | [rfc7591.md](rfc7591.md) | OPT | 32 |
| RFC 7592 DCR Management | [rfc7592.md](rfc7592.md) | OPT | 32 |
| RFC 8705 mTLS / cert-bound tokens | [rfc8705.md](rfc8705.md) | OPT | 25, 32 |
| RFC 9449 DPoP | [rfc9449.md](rfc9449.md) | OPT | 3, 22 |
| RFC 9068 JWT Access Token Profile | [rfc9068.md](rfc9068.md) | MUST | 22, 33 |
| RFC 9101 JAR (request object) | [rfc9101.md](rfc9101.md) | OPT | 23 |
| RFC 9396 RAR (rich authz) | [rfc9396.md](rfc9396.md) | OPT | 20, 23 |
| RFC 9700 OAuth 2.0 Security BCP | [rfc9700.md](rfc9700.md) | MUST | 23, 8, 37 |
| RFC 6819 OAuth Threat Model | [rfc6819.md](rfc6819.md) | SHOULD | 37 |

## OpenID Connect (OpenID Foundation)

| Стандарт | Файл | Conf | Секции |
| --- | --- | --- | --- |
| OIDC Core 1.0 | [oidc-core.md](oidc-core.md) | MUST | 23, 23a |
| OIDC Discovery 1.0 | [oidc-discovery.md](oidc-discovery.md) | MUST | 2 |
| OIDC Dynamic Client Registration 1.0 | [oidc-dcr.md](oidc-dcr.md) | OPT | 32 |
| OIDC RP-Initiated Logout 1.0 | [oidc-rp-logout.md](oidc-rp-logout.md) | SHOULD | 23 |
| OIDC Back-Channel Logout 1.0 | [oidc-backchannel-logout.md](oidc-backchannel-logout.md) | SHOULD | 23b |
| OIDC Front-Channel Logout 1.0 | [oidc-frontchannel-logout.md](oidc-frontchannel-logout.md) | OPT | 23 |
| OIDC Session Management 1.0 | [oidc-session.md](oidc-session.md) | OPT | 13, 23 |

## JOSE / токены

| Стандарт | Файл | Conf | Секции |
| --- | --- | --- | --- |
| RFC 7519 JWT | [rfc7519.md](rfc7519.md) | MUST | 22, 33 |
| RFC 7515 JWS | [rfc7515.md](rfc7515.md) | MUST | 22, 33 |
| RFC 7516 JWE | [rfc7516.md](rfc7516.md) | OPT | 22 |
| RFC 7517 JWK / JWKS | [rfc7517.md](rfc7517.md) | MUST | 2, 33 |
| RFC 7518 JWA | [rfc7518.md](rfc7518.md) | MUST | 33 |
| RFC 8037 EdDSA for JOSE (Ed25519) | [rfc8037.md](rfc8037.md) | SHOULD | 33 |

## MFA / OTP

| Стандарт | Файл | Conf | Секции |
| --- | --- | --- | --- |
| RFC 6238 TOTP | [rfc6238.md](rfc6238.md) | MUST | 10 |
| RFC 4226 HOTP | [rfc4226.md](rfc4226.md) | MUST | 10 |
| W3C WebAuthn L2/L3 | [webauthn.md](webauthn.md) | MUST | 9, 10 |
| FIDO CTAP2 | [fido-ctap2.md](fido-ctap2.md) | OPT | 9 |

## SCIM (provisioning)

| Стандарт | Файл | Conf | Секции |
| --- | --- | --- | --- |
| RFC 7642 SCIM Requirements | [rfc7642.md](rfc7642.md) | MUST | 41 |
| RFC 7643 SCIM Core Schema | [rfc7643.md](rfc7643.md) | MUST | 41 |
| RFC 7644 SCIM Protocol | [rfc7644.md](rfc7644.md) | MUST | 41, 42 |

## SAML 2.0 (OASIS, enterprise SSO)

| Стандарт | Файл | Conf | Секции |
| --- | --- | --- | --- |
| SAML 2.0 Core | [saml2-core.md](saml2-core.md) | MUST | 39 |
| SAML 2.0 Bindings | [saml2-bindings.md](saml2-bindings.md) | MUST | 39 |
| SAML 2.0 Web Browser SSO Profile | [saml2-profiles.md](saml2-profiles.md) | MUST | 38, 39 |
| SAML 2.0 Metadata | [saml2-metadata.md](saml2-metadata.md) | MUST | 38, 39 |

## HTTP / транспорт / форматы

| Стандарт | Файл | Conf | Секции |
| --- | --- | --- | --- |
| RFC 9457 Problem Details | [rfc9457.md](rfc9457.md) | MUST | 0 (error envelope) |
| RFC 6585 Additional Status Codes (429) | [rfc6585.md](rfc6585.md) | MUST | 37 |
| draft-ietf-httpapi-ratelimit-headers | [ratelimit-headers.md](ratelimit-headers.md) | SHOULD | 37 |
| RFC 6265bis Cookies | [rfc6265bis.md](rfc6265bis.md) | MUST | 0 (cookie mode) |
| RFC 9110 HTTP Semantics | [rfc9110.md](rfc9110.md) | MUST | все |
| RFC 7617 Basic Auth (client_secret_basic) | [rfc7617.md](rfc7617.md) | MUST | 23, 25 |
| RFC 6797 HSTS | [rfc6797.md](rfc6797.md) | SHOULD | транспорт |
| RFC 8594 Sunset header | [rfc8594.md](rfc8594.md) | SHOULD | API versioning |
| RFC 8446 TLS 1.3 | [rfc8446.md](rfc8446.md) | MUST | транспорт |
| RFC 3339 Timestamps | [rfc3339.md](rfc3339.md) | MUST | все time-поля |
| RFC 9562 UUID | [rfc9562.md](rfc9562.md) | SHOULD | id-поля |
| RFC 5322 Email format | [rfc5322.md](rfc5322.md) | MUST | email-валидация |
| E.164 Phone numbers (ITU-T) | [e164.md](e164.md) | MUST | phone-валидация |

## Security guidance / password

| Стандарт | Файл | Conf | Секции |
| --- | --- | --- | --- |
| NIST SP 800-63B Digital Identity | [nist-800-63b.md](nist-800-63b.md) | SHOULD | 6, 34 |
| Standard Webhooks (HMAC signing) | [standard-webhooks.md](standard-webhooks.md) | SHOULD | 27, 36 |
