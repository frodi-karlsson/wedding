import en from '../locales/en.json';
import is from '../locales/is.json';
import de from '../locales/de.json';
import sv from '../locales/sv.json';

export type Lang = 'en' | 'is' | 'de' | 'sv';

const messages: Record<Lang, Record<string, string>> = { en, is, de, sv };

/** The locale served without a URL prefix, at `/`. */
export const DEFAULT_LOCALE: Lang = 'sv';

export interface LocaleMeta {
  code: Lang;
  label: string;
  flag: string;
}

/**
 * Every locale, ordered with the default first. Single source of truth for
 * which locales exist and how the language picker labels them.
 */
export const LOCALES: LocaleMeta[] = [
  { code: 'sv', label: 'Svenska', flag: 'se' },
  { code: 'en', label: 'English', flag: 'gb' },
  { code: 'is', label: 'Íslenska', flag: 'is' },
  { code: 'de', label: 'Deutsch', flag: 'de' },
];

export const LOCALE_CODES: Lang[] = LOCALES.map((l) => l.code);

/** Locales that carry a URL prefix. Everything except the default. */
export const NON_DEFAULT_LOCALES: Lang[] = LOCALE_CODES.filter((c) => c !== DEFAULT_LOCALE);

/** URL path prefix for a locale. Empty for the default, `/xx` otherwise. */
export function localePrefix(lang: Lang): string {
  return lang === DEFAULT_LOCALE ? '' : `/${lang}`;
}

/**
 * Translate a key for a language.
 *
 * When `count` is provided, the key is resolved to a plural variant using the
 * language's CLDR plural category (via `Intl.PluralRules`): `${key}_${category}`
 * (e.g. `rsvp_intro_one`, `rsvp_intro_other`). This handles per-language rules.
 * Icelandic, for instance, treats 1/21/31… as "one". Resolution falls back to
 * the `_other` variant, then the default locale, then the raw key.
 */
export function translate(key: string, lang: Lang, count?: number): string {
  if (count !== undefined) {
    const category = new Intl.PluralRules(lang).select(count);
    const variant = `${key}_${category}`;
    const other = `${key}_other`;
    return (
      messages[lang]?.[variant] ??
      messages[lang]?.[other] ??
      messages[DEFAULT_LOCALE][variant] ??
      messages[DEFAULT_LOCALE][other] ??
      key
    );
  }
  return messages[lang]?.[key] ?? messages[DEFAULT_LOCALE][key] ?? key;
}

/** Resolve the active language from a pathname's leading segment. */
export function langFromPath(pathname: string): Lang {
  const seg = pathname.split('/').filter(Boolean)[0];
  if (seg && (NON_DEFAULT_LOCALES as string[]).includes(seg)) {
    return seg as Lang;
  }
  return DEFAULT_LOCALE;
}
