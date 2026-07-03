import { Show, type JSX } from 'solid-js';
import type { Lang } from '../../scripts/i18n';
import { translate } from '../../scripts/i18n';
import type { GuestInput } from '../../scripts/types.gen';

interface GuestRowProps {
  guest: GuestInput;
  index: number;
  lang: Lang;
  onRemove: (index: number) => void;
  onUpdate: (index: number, patch: Partial<GuestInput>) => void;
  // Whether this row can be removed (false hides the Remove button, e.g. the
  // last remaining co-primary of a group invite).
  canRemove?: boolean;
}

export function GuestRow(props: GuestRowProps): JSX.Element {
  // Co-primaries are a unit — they carry no "You"/"Guest N" legend; the editable
  // name identifies them.
  const showLegend = () => !props.guest.co_primary;
  const legend = () =>
    props.guest.is_primary
      ? translate('you_label', props.lang)
      : `${translate('guest_label', props.lang)} ${props.index}`;

  const nameLabel = () => translate('name_label', props.lang);

  function onNameInput(e: InputEvent & { currentTarget: HTMLInputElement }) {
    props.onUpdate(props.index, { name: e.currentTarget.value });
  }

  function onDietaryInput(e: InputEvent & { currentTarget: HTMLInputElement }) {
    props.onUpdate(props.index, { dietary_preference: e.currentTarget.value });
  }

  function onAlcoholFreeChange(e: Event & { currentTarget: HTMLInputElement }) {
    props.onUpdate(props.index, { alcohol_free: e.currentTarget.checked });
  }

  function onRemoveClick() {
    props.onRemove(props.index);
  }

  return (
    <fieldset class="guest-row" data-index={props.index}>
      <Show when={showLegend()}>
        <legend>{legend()}</legend>
      </Show>
      <label>
        <span>{nameLabel()}</span>
        <input type="text" value={props.guest.name} required onInput={onNameInput} />
      </label>
      <label>
        <span>{translate('dietary_label', props.lang)}</span>
        <input
          type="text"
          value={props.guest.dietary_preference}
          onInput={onDietaryInput}
        />
      </label>
      <label class="checkbox">
        <input
          type="checkbox"
          checked={props.guest.alcohol_free}
          onChange={onAlcoholFreeChange}
        />
        <span>{translate('alcohol_free_label', props.lang)}</span>
      </label>
      {!props.guest.is_primary && props.canRemove !== false && (
        <button
          type="button"
          class="btn btn--ghost btn--sm remove"
          onClick={onRemoveClick}
        >
          {translate('remove_guest', props.lang)}
        </button>
      )}
    </fieldset>
  );
}
