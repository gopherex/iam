/**
 * Localization for the hosted OIDC provider pages.
 *
 * The pages are rendered by IAM on behalf of a project, so the language they
 * speak belongs to that project, not to whoever happens to be running the admin
 * console. The interaction context returns the project's `default_locale` and
 * `supported_locales` (from its auth config); resolveLocale picks between them
 * and the browser's own preference, and never invents a language the project
 * has not declared.
 *
 * Strings are looked up as `t(key, fallback)`: the fallback is the English
 * source text, so a key with no translation degrades to readable English rather
 * than to a missing-key marker. That keeps a new string usable the moment it is
 * written and makes the catalogue additive.
 *
 * Deliberately not an i18n library: the provider pages are a handful of screens
 * and a dependency would buy plurals and ICU formatting that none of them need.
 */

export type Locale = string;

/** Translator: look up `key`, falling back to the English source text. */
export type T = (key: string, fallback: string) => string;

/**
 * Catalogues keyed by base language. English is the fallback and therefore has
 * no catalogue of its own — `t` returns the source text.
 */
const catalogues: Record<string, Record<string, string>> = {
  ru: {
    'provider.signInTo': 'Вход в {app}',
    'provider.continueAs': 'Продолжить как {email}',
    'provider.continueDescription': 'Вы уже вошли в этот аккаунт.',
    'provider.useAnotherAccount': 'Войти в другой аккаунт',
    'provider.continue': 'Продолжить',
    'provider.consentTitle': 'Разрешить доступ',
    'provider.consentDescription': '{app} запрашивает доступ к вашему аккаунту.',
    'provider.allow': 'Разрешить',
    'provider.deny': 'Отказать',
    'provider.remember': 'Запомнить решение',
    'provider.redirecting': 'Перенаправляем…',

    'provider.error.expired': 'Срок действия запроса истёк. Начните вход заново из приложения.',
    'provider.error.notFound': 'Запрос не найден или уже завершён.',
    'provider.error.denied': 'Доступ не предоставлен.',
    'provider.error.generic': 'Что-то пошло не так.',

    'scope.openid': 'Подтвердить, кто вы',
    'scope.profile': 'Ваше имя и данные профиля',
    'scope.email': 'Ваш адрес электронной почты',
    'scope.offline_access': 'Доступ, когда вас нет в сети',
    'scope.groups': 'Ваши роли',

    'device.title': 'Подключение устройства',
    'device.description': 'Введите код, показанный на устройстве.',
    'device.codeLabel': 'Код',
    'device.submit': 'Продолжить',
    'device.confirmTitle': 'Подтвердить устройство',
    'device.confirmDescription': '{app} просит доступ к вашему аккаунту с устройства.',
    'device.approve': 'Подтвердить',
    'device.deny': 'Отклонить',
    'device.approved': 'Устройство подключено. Можно вернуться к нему.',
    'device.denied': 'Запрос отклонён.',
    'device.expired': 'Срок действия кода истёк. Запросите новый код на устройстве.',

    'flow.title.collect_credentials': 'Добро пожаловать',
    'flow.desc.collect_credentials': 'Войдите или создайте аккаунт.',
    'flow.title.verify_email': 'Проверьте почту',
    'flow.desc.verify_email': 'Введите код, который мы отправили.',
    'flow.title.verify_phone': 'Подтвердите телефон',
    'flow.desc.verify_phone': 'Введите код, который мы отправили.',
    'flow.title.set_password': 'Новый пароль',
    'flow.desc.set_password': 'Выберите надёжный пароль.',
    'flow.title.mfa_required': 'Двухфакторная аутентификация',
    'flow.desc.mfa_required': 'Введите код подтверждения.',
    'flow.title.accept_consents': 'Ознакомьтесь и примите',
    'flow.desc.accept_consents': 'Прочитайте и примите обязательные документы.',
    'flow.title.request_access': 'Запрос доступа',
    'flow.desc.request_access': 'Отправьте заявку на присоединение.',
    'flow.title.awaiting_approval': 'Ожидает одобрения',
    'flow.desc.awaiting_approval': 'Ваша заявка на рассмотрении.',
    'flow.title.completed': 'Готово!',
    'flow.desc.completed': 'Перенаправляем…',
    'flow.title.blocked': 'Доступ запрещён',

    'flow.expired': 'Сессия истекла. Начните заново.',
    'flow.aborted': 'Вход отменён. Начните заново.',
    'flow.startOver': 'Начать заново',
    'flow.cancel': 'Отменить и начать заново',
  },
};

/**
 * resolveLocale picks the language for a page.
 *
 * The project's supported list is authoritative: a locale the project has not
 * declared is never chosen, however loudly the browser asks for it. Preference
 * order is the browser's languages (so a Russian-speaking user of a project that
 * supports Russian gets Russian), then the project default, then English.
 */
export function resolveLocale(
  defaultLocale: string | undefined,
  supported: string[] | undefined,
  browserLanguages: readonly string[] = typeof navigator === 'undefined' ? [] : navigator.languages ?? [],
): Locale {
  const allowed = (supported ?? []).filter(Boolean);
  const fallback = defaultLocale || allowed[0] || 'en';

  if (allowed.length === 0) return fallback;

  for (const want of browserLanguages) {
    const hit = allowed.find((l) => sameLanguage(l, want));
    if (hit) return hit;
  }

  return allowed.find((l) => sameLanguage(l, fallback)) ?? fallback;
}

/** sameLanguage compares two tags by their base language (en-GB ~ en). */
function sameLanguage(a: string, b: string): boolean {
  return baseLanguage(a) === baseLanguage(b);
}

/** baseLanguage strips the region subtag and lowercases (ru-RU -> ru). */
export function baseLanguage(tag: string): string {
  return tag.toLowerCase().split(/[-_]/)[0] ?? '';
}

/**
 * translator returns the lookup function for a locale. Interpolation is
 * `{name}` placeholders substituted from `vars`.
 */
export function translator(locale: Locale): T {
  const table = catalogues[baseLanguage(locale)] ?? {};
  return (key, fallback) => table[key] ?? fallback;
}

/** interpolate substitutes {name} placeholders. */
export function interpolate(template: string, vars: Record<string, string>): string {
  return template.replace(/\{(\w+)\}/g, (whole, name: string) => vars[name] ?? whole);
}

/**
 * scopeLabel renders an OAuth scope in words a person can act on. An unknown
 * scope shows its raw name — better a technical string than a silent omission
 * of something the user is about to grant.
 */
export function scopeLabel(t: T, scope: string): string {
  const known: Record<string, string> = {
    openid: 'Confirm who you are',
    profile: 'Your name and profile details',
    email: 'Your email address',
    offline_access: 'Access while you are away',
    groups: 'Your roles',
  };
  const fallback = known[scope];
  return fallback ? t(`scope.${scope}`, fallback) : scope;
}
