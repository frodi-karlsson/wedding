import {
  createSignal,
  createMemo,
  For,
  Show,
  Switch,
  Match,
  onMount,
  untrack,
  type JSX,
} from 'solid-js';
import type { Lang } from '../scripts/i18n';
import { translate } from '../scripts/i18n';
import { api } from '../scripts/api';
import {
  createRsvpState,
  addGuest,
  removeGuest,
  updateGuest,
  canSubmit,
  guestsToInput,
  type RsvpState,
} from '../scripts/rsvp.service';
import type { GuestInput } from '../scripts/types.gen';
import { GuestRow } from './components/GuestRow';

interface RsvpFormProps {
  lang: Lang;
}

export function RsvpForm(props: RsvpFormProps): JSX.Element {
  const lang = untrack(() => props.lang);

  const [state, setState] = createSignal<RsvpState>({
    invite: { id: '', name: '', min_plus: 0, max_plus: 0, submitted: false },
    guests: [],
    status: 'loading',
    lang,
  });

  let rootRef: HTMLDivElement | undefined;

  const inviteId = () => new URLSearchParams(globalThis.location.search).get('id');

  const submitDisabled = createMemo(
    () => !canSubmit(state()) || state().status === 'submitting',
  );

  const canAdd = createMemo(() => {
    const plusCount = state().guests.filter((g) => !g.is_primary).length;
    return plusCount < state().invite.max_plus;
  });

  const intro = createMemo(() => {
    const s = state();
    const minClause =
      s.invite.min_plus > 0
        ? translate('min_clause', s.lang).replace('{min}', String(s.invite.min_plus))
        : '';
    return translate('rsvp_intro', s.lang, s.invite.max_plus)
      .replace('{max}', String(s.invite.max_plus))
      .replace('{min_clause}', minClause);
  });

  onMount(() => {
    rootRef?.classList.add('fade-in');
    const id = inviteId();
    if (!id) return;

    api
      .getInvite(id)
      .run()
      .then((response) => {
        setState(createRsvpState(response.invite, response.guests, lang));
      })
      .catch(() => {
        setState((prev) => ({
          ...prev,
          status: 'error',
          errorMessage: translate('load_error', lang),
        }));
      });
  });

  function onAddGuest() {
    setState((prev) => ({ ...addGuest(prev), errorMessage: undefined }));
  }

  function onRemoveGuest(index: number) {
    setState((prev) => ({ ...removeGuest(prev, index), errorMessage: undefined }));
  }

  function onUpdateGuest(index: number, patch: Partial<GuestInput>) {
    setState((prev) => ({ ...updateGuest(prev, index, patch), errorMessage: undefined }));
  }

  function onSubmit(e: Event) {
    e.preventDefault();
    if (!canSubmit(state())) return;
    const id = inviteId();
    if (!id) return;

    const guests = guestsToInput(state());
    setState((prev) => ({ ...prev, status: 'submitting', errorMessage: undefined }));

    api
      .rsvp(id, guests)
      .run()
      .then(() => {
        setState((prev) => ({ ...prev, status: 'confirmed' }));
      })
      .catch(() => {
        setState((prev) => ({
          ...prev,
          status: 'ready',
          errorMessage: translate('submit_error', prev.lang),
        }));
      });
  }

  return (
    <div id="rsvp-root" ref={(el) => { rootRef = el; }}>
      <Switch>
        <Match when={state().status === 'loading'}>
          <p class="loading">{translate('loading', lang)}</p>
        </Match>
        <Match when={state().status === 'error'}>
          <p class="error">{state().errorMessage ?? translate('load_error', lang)}</p>
        </Match>
        <Match when={state().status === 'confirmed'}>
          <div class="confirmed card">
            <h2 class="heading heading--md">{translate('thank_you', lang)}</h2>
            <p>{translate('thank_you_body', lang)}</p>
          </div>
        </Match>
        <Match when={state().status === 'ready' || state().status === 'submitting'}>
          <form class="rsvp-form card" onSubmit={onSubmit}>
            <h2 class="heading heading--md">{state().invite.name}</h2>
            <p class="intro">{intro()}</p>
            <div class="guests">
              <For each={state().guests}>
                {(guest, index) => (
                  <GuestRow
                    guest={guest}
                    index={index()}
                    lang={lang}
                    onRemove={onRemoveGuest}
                    onUpdate={onUpdateGuest}
                  />
                )}
              </For>
            </div>
            <div class="actions">
              <Show when={canAdd()}>
                <button
                  type="button"
                  class="btn btn--secondary btn--md add"
                  onClick={onAddGuest}
                >
                  {translate('add_guest', lang)}
                </button>
              </Show>
              <button
                type="submit"
                class="btn btn--primary btn--md submit"
                disabled={submitDisabled()}
              >
                {state().status === 'submitting'
                  ? translate('submitting', lang)
                  : translate('submit', lang)}
              </button>
            </div>
            <Show when={state().status === 'ready' && state().errorMessage}>
              <p class="error">{state().errorMessage}</p>
            </Show>
          </form>
        </Match>
      </Switch>
    </div>
  );
}

export default RsvpForm;
