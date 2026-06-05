# SAML 2.0 Bindings

- OASIS · https://docs.oasis-open.org/security/saml/v2.0/saml-bindings-2.0-os.pdf · Conf: MUST

## Назначение
Транспорт сообщений: HTTP-Redirect (AuthnRequest), HTTP-POST (Response на ACS), HTTP-Artifact, SOAP.

## Где в API
- §39 `/sso/saml/{id}/login` (Redirect), `/acs` (POST), `/slo`

## Conformance
- ACS принимает HTTP-POST с `SAMLResponse` (base64) + `RelayState`.
- AuthnRequest — HTTP-Redirect с deflate+base64, опц. подпись.

## Gotchas
- RelayState несёт наш `redirect_to`/state; пусто при IdP-initiated → `default_redirect_uri` (§39).
