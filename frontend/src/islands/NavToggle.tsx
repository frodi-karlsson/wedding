import { createSignal, createEffect, onMount, onCleanup, type JSX } from 'solid-js';

export function NavToggle(): JSX.Element {
  const [open, setOpen] = createSignal(false);
  let toggle: HTMLButtonElement | undefined;
  let panel: HTMLElement | undefined;

  onMount(() => {
    toggle = document.querySelector<HTMLButtonElement>('[data-nav-toggle]') ?? undefined;
    panel = document.querySelector<HTMLElement>('[data-nav-panel]') ?? undefined;

    function onToggleClick() {
      setOpen((prev) => !prev);
    }

    function onNavLinkClick() {
      setOpen(false);
    }

    function onKeydown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false);
    }

    toggle?.addEventListener('click', onToggleClick);
    panel?.querySelectorAll<HTMLAnchorElement>('[data-nav-link]').forEach((link) => {
      link.addEventListener('click', onNavLinkClick);
    });
    document.addEventListener('keydown', onKeydown);

    onCleanup(() => {
      toggle?.removeEventListener('click', onToggleClick);
      panel?.querySelectorAll<HTMLAnchorElement>('[data-nav-link]').forEach((link) => {
        link.removeEventListener('click', onNavLinkClick);
      });
      document.removeEventListener('keydown', onKeydown);
    });
  });

  // `NavToggle` renders nothing — the HTML is SSR'd by Navbar.astro.
  // This island only attaches behavior and syncs `open()` into the DOM.
  createEffect(() => {
    const isOpen = open();
    if (isOpen) {
      panel?.setAttribute('data-open', '');
      toggle?.setAttribute('aria-expanded', 'true');
      toggle?.setAttribute('aria-label', 'Close navigation menu');
    } else {
      panel?.removeAttribute('data-open');
      toggle?.setAttribute('aria-expanded', 'false');
      toggle?.setAttribute('aria-label', 'Open navigation menu');
    }
  });

  return <></>;
}
