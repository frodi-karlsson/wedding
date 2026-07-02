import { createSignal, createMemo, For, onMount, onCleanup, type JSX } from 'solid-js';
import type { Lang } from '../scripts/i18n';
import { buildLocaleHref } from '../scripts/nav';

interface LocaleOption {
  code: Lang;
  label: string;
  icon: string;
}

interface LangPickerProps {
  lang: Lang;
}

const LOCALES: LocaleOption[] = [
  { code: 'en', label: 'English', icon: 'gb' },
  { code: 'is', label: 'Íslenska', icon: 'is' },
  { code: 'de', label: 'Deutsch', icon: 'de' },
  { code: 'sv', label: 'Svenska', icon: 'se' },
];

export function LangPicker(props: LangPickerProps): JSX.Element {
  const [open, setOpen] = createSignal(false);
  const current = createMemo(() => LOCALES.find((l) => l.code === props.lang) ?? LOCALES[0]);

  function hrefFor(code: Lang): string {
    return buildLocaleHref(globalThis.location, code);
  }

  // Close on outside click. Solid delegates events to document, so
  // stopPropagation on the trigger can't prevent this listener from firing
  // (both are on document). Instead, check whether the click was inside
  // the dropdown — the standard "click outside" pattern.
  onMount(() => {
    function onDocClick(e: MouseEvent) {
      if (!(e.target as HTMLElement).closest('[data-lang-dropdown]')) {
        setOpen(false);
      }
    }
    document.addEventListener('click', onDocClick);
    onCleanup(() => document.removeEventListener('click', onDocClick));
  });

  return (
    <div class="lang-dropdown" data-lang-dropdown>
      <button
        type="button"
        class="lang-dropdown__trigger"
        aria-haspopup="listbox"
        aria-expanded={open()}
        data-lang-toggle
        onClick={() => setOpen(!open())}
      >
        <img
          class="lang-dropdown__flag"
          src={`/flags/${current().icon}.svg`}
          width="20"
          height="15"
          alt=""
          aria-hidden="true"
          decoding="sync"
        />
        <span class="lang-dropdown__label">{current().label}</span>
        <svg
          class="lang-dropdown__chevron"
          width="12"
          height="12"
          viewBox="0 0 12 12"
          fill="none"
          aria-hidden="true"
        >
          <path
            d="M3 4.5L6 7.5L9 4.5"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
      <ul class="lang-dropdown__menu" role="listbox" hidden={!open()} data-lang-menu>
        <For each={LOCALES}>
          {(l) => (
            <li role="option" aria-selected={l.code === props.lang ? 'true' : 'false'}>
              <a
                href={hrefFor(l.code)}
                class="lang-dropdown__item"
                aria-current={l.code === props.lang ? 'true' : undefined}
              >
                <img
                  class="lang-dropdown__flag"
                  src={`/flags/${l.icon}.svg`}
                  width="20"
                  height="15"
                  alt=""
                  aria-hidden="true"
                  decoding="sync"
                />
                <span>{l.label}</span>
              </a>
            </li>
          )}
        </For>
      </ul>
    </div>
  );
}

export default LangPicker;
