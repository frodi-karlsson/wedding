import type { Lang } from './i18n';
import { langFromPath } from './i18n';

interface UrlLike {
  pathname: string;
  search: string;
  hash: string;
}

interface SearchLike {
  search: string;
}

// `localeFromPath` is the canonical `langFromPath` from i18n.ts, re-exported
// here under the name nav consumers use.
export { langFromPath as localeFromPath };

export function buildLocaleHref(
  currentUrl: UrlLike,
  targetLang: Lang,
): string {
  const currentLang = langFromPath(currentUrl.pathname);
  const stripped =
    currentLang === 'en'
      ? currentUrl.pathname
      : currentUrl.pathname.slice(`/${currentLang}`.length) || '/';
  const prefix = targetLang === 'en' ? '' : `/${targetLang}`;
  const base = stripped === '/' ? (prefix || '/') : `${prefix}${stripped}`;
  return `${base}${currentUrl.search}${currentUrl.hash}`;
}

export function hasInviteId(url: SearchLike): boolean {
  return new URLSearchParams(url.search).get('id') !== null;
}

export function scrollToFragment(hash: string): void {
  if (!hash) return;
  const el = document.querySelector(hash);
  el?.scrollIntoView({ behavior: 'smooth' });
}
