import en from '../locales/en.json';
import is from '../locales/is.json';
import de from '../locales/de.json';
import sv from '../locales/sv.json';

export type Lang = 'en' | 'is' | 'de' | 'sv';

const messages: Record<Lang, Record<string, string>> = { en, is, de, sv };
const LOCALES: Lang[] = ['is', 'de', 'sv']; // non-default (en has no prefix)

/**
 * Translate a key for a language.
 *
 * When `count` is provided, the key is resolved to a plural variant using the
 * language's CLDR plural category (via `Intl.PluralRules`): `${key}_${category}`
 * (e.g. `rsvp_intro_one`, `rsvp_intro_other`). This handles per-language rules —
 * Icelandic, for instance, treats 1/21/31… as "one". Resolution falls back to
 * the `_other` variant, then English, then the raw key.
 */
export function translate(key: string, lang: Lang, count?: number): string {
  if (count !== undefined) {
    const category = new Intl.PluralRules(lang).select(count);
    const variant = `${key}_${category}`;
    const other = `${key}_other`;
    return (
      messages[lang]?.[variant] ??
      messages[lang]?.[other] ??
      messages.en[variant] ??
      messages.en[other] ??
      key
    );
  }
  return messages[lang]?.[key] ?? messages.en[key] ?? key;
}

export function langFromPath(pathname: string): Lang {
  const seg = pathname.split('/').filter(Boolean)[0];
  if (seg && (LOCALES as string[]).includes(seg)) {
    return seg as Lang;
  }
  return 'en';
}
