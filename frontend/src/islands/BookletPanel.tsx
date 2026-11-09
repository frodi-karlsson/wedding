import { createMemo, createSignal, For, Match, onCleanup, onMount, Show, Switch, untrack, type JSX } from 'solid-js';
import { Dynamic, Portal } from 'solid-js/web';
import { LOCALES, localePrefix, translate, type Lang } from '../scripts/i18n';
import { api } from '../scripts/api';
import { bookletBlocks, impose, paddedPageCount, spreads, type Block } from '../scripts/booklet';
import { blockClass, blockTag, paginate } from '../scripts/booklet.layout';
import type { SpeechResponse } from '../scripts/types.gen';

type PrintLayout = 'home' | 'shop';

type Page = { kind: 'cover' } | { kind: 'content'; blocks: Block[] } | { kind: 'blank' };

// Every face the booklet uses. Measuring before these load would lay text out
// in a fallback font and break pages in the wrong places.
const BOOKLET_FONTS = ['500 11.5pt Cormorant', '600 19pt Cormorant', 'italic 500 12.5pt Cormorant'];

// An open spread is two A5 pages: 296mm at 96px per inch.
const SPREAD_WIDTH_PX = (296 / 25.4) * 96;

function isLang(value: string | null): value is Lang {
  return LOCALES.some((l) => l.code === value);
}

function BlockView(props: { block: Block }): JSX.Element {
  return (
    <Dynamic component={blockTag(props.block)} class={blockClass(props.block)}>
      {props.block.text}
    </Dynamic>
  );
}

export function BookletPanel(props: { lang: Lang }): JSX.Element {
  const uiLang = untrack(() => props.lang);
  const t = (key: string) => translate(key, uiLang);
  const fromUrl = new URLSearchParams(globalThis.location.search).get('lang');

  const [bookLang, setBookLang] = createSignal<Lang>(isLang(fromUrl) ? fromUrl : uiLang);
  const [layout, setLayout] = createSignal<PrintLayout>('home');
  const [speeches, setSpeeches] = createSignal<SpeechResponse[] | undefined>(undefined);
  const [pages, setPages] = createSignal<Page[]>([]);
  const [failed, setFailed] = createSignal(false);
  const [previewScale, setPreviewScale] = createSignal(1);
  let measureBody: HTMLDivElement | undefined;
  let previewObserver: ResizeObserver | undefined;
  onCleanup(() => previewObserver?.disconnect());

  function watchPreviewWidth(el: HTMLDivElement) {
    previewObserver = new ResizeObserver(() => setPreviewScale(Math.min(1, el.clientWidth / SPREAD_WIDTH_PX)));
    previewObserver.observe(el);
  }

  const content = createMemo(() => bookletBlocks(speeches() ?? [], bookLang()));
  const pageCount = () => pages().length;

  function layOut() {
    if (!measureBody || speeches() === undefined) return;
    const contentPages = paginate(measureBody, content().blocks).map((blocks): Page => ({ kind: 'content', blocks }));
    const laidOut: Page[] = [{ kind: 'cover' }, ...contentPages];
    const total = paddedPageCount(laidOut.length);
    while (laidOut.length < total) laidOut.push({ kind: 'blank' });
    setPages(laidOut);
  }

  onMount(async () => {
    try {
      const list = await api.listSpeeches().run();
      setSpeeches(list.speeches);
    } catch {
      setFailed(true);
      return;
    }
    await Promise.all(BOOKLET_FONTS.map((font) => document.fonts.load(font)));
    layOut();
  });

  function chooseLang(code: Lang) {
    setBookLang(code);
    const url = new URL(globalThis.location.href);
    url.searchParams.set('lang', code);
    globalThis.history.replaceState(null, '', url);
    layOut();
  }

  function PageView(pageProps: { number: number | null }): JSX.Element {
    const page = () => (pageProps.number === null ? undefined : pages()[pageProps.number - 1]);
    return (
      <div
        class="booklet-page"
        classList={{ 'booklet-page--none': pageProps.number === null }}
        lang={bookLang()}
        data-page={pageProps.number ?? undefined}
      >
        <Switch>
          <Match when={page()?.kind === 'cover'}>
            <div class="booklet-cover">
              <p class="booklet-cover__names">{translate('hero_photo_alt', bookLang())}</p>
              <p class="booklet-cover__title">{translate('speeches_title', bookLang())}</p>
              <p class="booklet-cover__date">{translate('hero_date', bookLang())}</p>
            </div>
          </Match>
          <Match when={page()?.kind === 'content' ? (page() as { blocks: Block[] }) : undefined}>
            {(contentPage) => (
              <>
                <div class="booklet-page__body">
                  <For each={contentPage().blocks}>{(block) => <BlockView block={block} />}</For>
                </div>
                <span class="booklet-folio">{pageProps.number}</span>
              </>
            )}
          </Match>
        </Switch>
      </div>
    );
  }

  const pagesLabel = () =>
    t('booklet_pages')
      .replace('{pages}', String(pageCount()))
      .replace('{sheets}', String(pageCount() / 4));

  return (
    <div class="booklet">
      {/* Off-screen page used only to measure how much text fits. */}
      <div class="booklet-page booklet-measure" lang={bookLang()} aria-hidden="true">
        <div class="booklet-page__body" ref={(el) => { measureBody = el; }} />
      </div>

      <Show
        when={!failed()}
        fallback={
          <p class="error">
            {t('speeches_load_error')} <a href={`${localePrefix(uiLang)}/speeches`}>{t('speeches_title')}</a>
          </p>
        }
      >
        <div class="booklet-toolbar card">
          <a href={`${localePrefix(uiLang)}/speeches`}>← {t('speeches_title')}</a>
          <label>
            <span>{t('booklet_language')}</span>
            <select value={bookLang()} onChange={(e) => chooseLang(e.currentTarget.value as Lang)}>
              <For each={LOCALES}>{(l) => <option value={l.code}>{l.label}</option>}</For>
            </select>
          </label>
          <label>
            <span>{t('booklet_layout_label')}</span>
            <select value={layout()} onChange={(e) => setLayout(e.currentTarget.value as PrintLayout)}>
              <option value="home">{t('booklet_layout_home')}</option>
              <option value="shop">{t('booklet_layout_shop')}</option>
            </select>
          </label>
          <button type="button" class="btn btn--primary btn--md" onClick={() => globalThis.print()}>
            {t('booklet_print')}
          </button>
          <p class="booklet-hint">
            {layout() === 'home' ? t('booklet_home_instructions') : t('booklet_shop_instructions')}
          </p>
          <Show when={pageCount() > 0}>
            <p class="booklet-hint">{pagesLabel()}</p>
          </Show>
          <Show when={content().missing.length > 0}>
            <p class="error">{t('booklet_missing').replace('{names}', content().missing.join(', '))}</p>
          </Show>
        </div>

        <Show when={pageCount() > 0} fallback={<p class="loading">{t('loading')}</p>}>
          <div class="booklet-preview" ref={watchPreviewWidth}>
            <For each={spreads(pageCount())}>
              {([left, right]) => (
                <div class="booklet-spread" style={{ zoom: previewScale() }}>
                  <PageView number={left} />
                  <PageView number={right} />
                </div>
              )}
            </For>
          </div>

          <Portal mount={document.body} ref={(el) => el.classList.add('booklet-print')}>
            <style>{layout() === 'home' ? '@page { size: A4 landscape; margin: 0; }' : '@page { size: A5; margin: 0; }'}</style>
            <Show
              when={layout() === 'home'}
              fallback={
                <For each={pages()}>
                  {(_, i) => (
                    <div class="booklet-sheet booklet-sheet--a5">
                      <PageView number={i() + 1} />
                    </div>
                  )}
                </For>
              }
            >
              <For each={impose(pageCount())}>
                {(sheet) => (
                  <>
                    <div class="booklet-sheet booklet-sheet--a4">
                      <PageView number={sheet.front[0]} />
                      <PageView number={sheet.front[1]} />
                    </div>
                    <div class="booklet-sheet booklet-sheet--a4">
                      <PageView number={sheet.back[0]} />
                      <PageView number={sheet.back[1]} />
                    </div>
                  </>
                )}
              </For>
            </Show>
          </Portal>
        </Show>
      </Show>
    </div>
  );
}

export default BookletPanel;
