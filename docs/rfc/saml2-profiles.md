# SAML 2.0 Profiles (Web Browser SSO)

- OASIS · https://docs.oasis-open.org/security/saml/v2.0/saml-profiles-2.0-os.pdf · Conf: MUST

## Назначение
Web Browser SSO Profile (SP-initiated и IdP-initiated), Single Logout Profile.

## Где в API
- §38 SSO connections, §39 login/acs/slo

## Conformance
- SP-initiated: login → AuthnRequest → IdP → ACS.
- Поддержать IdP-initiated (unsolicited Response, без InResponseTo) — с осторожностью (нет CSRF-привязки).

## Gotchas
- IdP-initiated отключаемо per-connection (security). При включении — строгая валидация Audience.
