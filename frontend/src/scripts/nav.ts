import type { Lang } from './i18n';

interface UrlLike {
  pathname: string;
  search: string;
  hash: string;
}

interface SearchLike {
  search: string;
}

const NON_DEFAULT_LOCALES: Lang[] = ['is', 'de', 'sv'];

export function localeFromPath(pathname: string): Lang {
  const seg = pathname.split('/').filter(Boolean)[0];
  if (seg && (NON_DEFAULT_LOCALES as string[]).includes(seg)) {
    return seg as Lang;
  }
  return 'en';
}

export function buildLocaleHref(
  currentUrl: UrlLike,
  targetLang: Lang,
): string {
  const currentLang = localeFromPath(currentUrl.pathname);
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
