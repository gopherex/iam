# SAML 2.0 Metadata

- OASIS · https://docs.oasis-open.org/security/saml/v2.0/saml-metadata-2.0-os.pdf · Conf: MUST

## Назначение
Обмен конфигурацией SP↔IdP: entityID, endpoints (ACS/SLO), сертификаты подписи/шифрования.

## Где в API
- §39 `/sso/saml/{id}/metadata` (наш SP-metadata), §38 импорт IdP-metadata

## Conformance
- Отдавать SP EntityDescriptor с ACS URL, SP cert, NameIDFormat.
- Импорт IdP metadata → автозаполнение connection (§38).

## Gotchas
- Ротация SP-cert (§38 rotate-certificate) — публиковать оба cert в metadata на период перехода.
