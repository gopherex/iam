# OAuth 2.0 Form Post Response Mode

- OpenID Foundation · https://openid.net/specs/oauth-v2-form-post-response-mode-1_0.html · Conf: SHOULD

## Назначение
`response_mode=form_post`: ответ авторизации возвращается как HTML-документ с
самоотправляющейся формой, которая POST-ит параметры на `redirect_uri`. Код не
попадает ни в адресную строку, ни в историю браузера, ни в `Referer` страницы,
которую клиент отрисует следующей.

## Где в API
- `GET /oauth2/authorize?response_mode=form_post`
- `POST /v1/oauth/interaction/{id}/consent` и `.../reject` — поле `form_post`
  в ответе вместо `redirect_to`
- `response_modes_supported` в discovery-документе

## Conformance
- Поддерживаются `query` (по умолчанию), `fragment` и `form_post`; всё остальное —
  `unsupported_response_mode`.
- Набор параметров одинаков во всех режимах: `code`, `state`, `iss`
  ([RFC 9207](rfc9207.md)). Клиент, проверяющий `iss`, не ломается при смене
  режима.
- Ошибки, которые доходят до `redirect_uri`, доставляются тем же режимом, что и
  успешный ответ.
- Эффективный режим берётся уже после разбора pushed request
  ([RFC 9126](rfc9126.md)) и signed request object ([RFC 9101](rfc9101.md)):
  `response_mode` — один из параметров, которые они замещают.

## Gotchas
- Редирект на interaction UI — не ответ авторизации, он остаётся 302 в любом
  режиме. Формой отдаётся только то, что адресовано клиенту.
- Silent-путь (сессия и согласие уже есть) отдаёт документ прямо с
  `/oauth2/authorize`: 200 `text/html`, без `Location`.
- Интерактивный путь отдаёт форму хостовому UI в JSON — страница строит и
  отправляет её сама, потому что ответ формируется уже после согласия.
- Все значения экранируются шаблоном: `state` выбирает клиент, и разметка в
  чужом `state` не должна исполняться на нашей странице.
