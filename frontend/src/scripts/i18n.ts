import en from '../locales/en.json';
import is from '../locales/is.json';
import de from '../locales/de.json';
import sv from '../locales/sv.json';

export type Lang = 'en' | 'is' | 'de' | 'sv';

const messages: Record<Lang, Record<string, string>> = { en, is, de, sv };
const LOCALES: Lang[] = ['is', 'de', 'sv']; // non-default (en has no prefix)

export function translate(key: string, lang: Lang): string {
  return messages[lang]?.[key] ?? messages.en[key] ?? key;
}

export function langFromPath(pathname: string): Lang {
  const seg = pathname.split('/').filter(Boolean)[0];
  if (seg && (LOCALES as string[]).includes(seg)) {
    return seg as Lang;
  }
  return 'en';
}
