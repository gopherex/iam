# W3C WebAuthn (Web Authentication) Level 2/3

- W3C Recommendation · https://www.w3.org/TR/webauthn-3/ · Conf: MUST

## Назначение
Passkeys / security keys: публичный ключ вместо пароля. `navigator.credentials.create/get`. Сервер выдаёт challenge, проверяет attestation/assertion.

## Где в API
- §9 passkeys (login/register options+verify, credentials CRUD)
- §10 WebAuthn как MFA-фактор

## Conformance
- RP-server: генерит challenge (§9 options), хранит credential_id+public_key+sign_count.
- Verify: проверка clientDataJSON (origin, type, challenge), authenticatorData (rpIdHash, flags UV/UP), подпись.
- Discoverable credentials (resident key) + `mediation:conditional` для passkey-autofill (§9 loginOptions).

## Gotchas
- rpId = домен (не путать с origin). Кросс-сабдомен — внимательно.
- sign_count regression → возможный клон, флагать в risk (§37).
- Headless ок: сервер отдаёт options, браузер продукта вызывает API, продукт шлёт credential на verify.
