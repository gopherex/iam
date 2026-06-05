# FIDO2 CTAP2 (Client to Authenticator Protocol)

- FIDO Alliance · https://fidoalliance.org/specs/ · Conf: OPT (reference)

## Назначение
Протокол браузер↔аутентификатор (роуминговые ключи, platform authenticators). Парный к WebAuthn со стороны устройства.

## Где в API
- §9 passkeys (косвенно — реализуется браузером/ОС, не нашим сервером)

## Gotchas
- Сервер CTAP2 напрямую не трогает — только WebAuthn-уровень. Файл для справки по attestation-форматам (packed, tpm, apple).
