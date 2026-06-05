# SAML 2.0 Core (Assertions and Protocols)

- OASIS · https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf · Conf: MUST

## Назначение
Assertion (statements об аутентификации), protocols (AuthnRequest/Response), NameID, conditions, signing.

## Где в API
- §39 SAML runtime (ACS обрабатывает Response/Assertion)

## Conformance
- Проверять подпись Assertion/Response, `Conditions` (NotBefore/NotOnOrAfter, Audience), `SubjectConfirmation` (Recipient, InResponseTo).
- NameID + attribute statements → маппинг на user/groups.

## Gotchas
- XML Signature wrapping (XSW) атаки — использовать проверенную либу, валидировать canonical.
- Replay: кешировать assertion ID до NotOnOrAfter.
